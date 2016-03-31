// Copyright 2016 Mender Software AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
package memmap

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/mendersoftware/artifacts/models/images"
	"github.com/mendersoftware/artifacts/models/users"
	"github.com/mendersoftware/artifacts/utils/safemap"
	"github.com/satori/go.uuid"
)

var (
	ErrNotFound = errors.New("Entry not found.")
)

// Mem mapped file based simple model implementation
// To be replaced with DATABASE backed storage solution.
type ImagesInMem struct {
	storage safemap.Map
}

func NewImagesInMem(m safemap.Map) *ImagesInMem {
	return &ImagesInMem{
		storage: m,
	}
}

func makeInternamKey(customerId, imageId string) string {
	return customerId + "." + imageId
}

func isKeyBelongToCustomer(customerId, key string) bool {
	return strings.HasPrefix(key, customerId+".")
}

func (i *ImagesInMem) Find(user users.UserI) ([]*images.ImageMeta, error) {

	keys := i.storage.Keys()
	sort.Strings(keys)

	list := make([]*images.ImageMeta, 0)
	for _, key := range keys {

		if !isKeyBelongToCustomer(user.GetCustomerID(), key) {
			continue
		}

		if img, found := i.storage.Get(key); found {
			list = append(list, img.(*images.ImageMeta))
		}
	}

	return list, nil
}

func (i *ImagesInMem) FindOne(user users.UserI, id string) (*images.ImageMeta, error) {
	img, found := i.storage.Get(makeInternamKey(user.GetCustomerID(), id))
	if !found {
		return nil, ErrNotFound
	}

	return img.(*images.ImageMeta), nil
}

func (i *ImagesInMem) Exists(user users.UserI, id string) (bool, error) {

	return i.storage.Has(makeInternamKey(user.GetCustomerID(), id)), nil
}

func (i *ImagesInMem) Insert(user users.UserI, image *images.ImageMeta) (string, error) {

	if err := image.Valid(); err != nil {
		return "", err
	}

	id, err := i.makeID()
	if err != nil {
		return "", err
	}

	image.Id = id
	i.storage.Set(makeInternamKey(user.GetCustomerID(), image.Id), image)

	return id, nil
}

func (i *ImagesInMem) Update(user users.UserI, image *images.ImageMeta) error {
	if err := image.Valid(); err != nil {
		return err
	}

	local, found := i.storage.Get(makeInternamKey(user.GetCustomerID(), image.Id))
	if !found {
		return ErrNotFound
	}

	local.(*images.ImageMeta).LastUpdated = time.Now()
	i.storage.Set(makeInternamKey(user.GetCustomerID(), image.Id), image)

	return nil
}

func (i *ImagesInMem) Delete(user users.UserI, id string) error {
	i.storage.Remove(makeInternamKey(user.GetCustomerID(), id))
	return nil
}

// TODO: Not used anymore, possibly can be removed
func (i *ImagesInMem) FindByName(user users.UserI, name string) ([]*images.ImageMeta, error) {
	keys := i.storage.Keys()
	sort.Strings(keys)

	list := make([]*images.ImageMeta, 0)
	for _, key := range keys {

		if !isKeyBelongToCustomer(user.GetCustomerID(), key) {
			continue
		}

		img, found := i.storage.Get(key)
		if !found {
			continue
		}

		if img.(*images.ImageMeta).Name != name && name != "" {
			continue
		}

		list = append(list, img.(*images.ImageMeta))
	}

	return list, nil
}

func (i *ImagesInMem) makeID() (string, error) {
	var id string
	for {
		id = uuid.NewV4().String()
		if found := i.storage.Has(id); !found {
			break
		}
	}

	return id, nil
}
