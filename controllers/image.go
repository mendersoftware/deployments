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
package controllers

import (
	"errors"
	"fmt"
	"time"

	"github.com/mendersoftware/artifacts/models/fileservice"
	"github.com/mendersoftware/artifacts/models/images"
	"github.com/mendersoftware/artifacts/models/users"
)

var (
	// Entry not found
	ErrNotFound = errors.New("Resource not found")
)

var (
	// DefaultUploadLinkExpireTime expire time for generated links
	DefaultUploadLinkExpireTime = time.Hour * 24
)

// ImagesControllerI interface for images controller
type ImagesControllerI interface {
	Lookup(user users.UserI) ([]*images.ImageMeta, error)
	Get(user users.UserI, id string) (*images.ImageMeta, error)
	Create(user users.UserI, public *images.ImageMetaPublic) (*images.ImageMeta, error)
	Edit(user users.UserI, id string, public *images.ImageMetaPublic) error
	Delete(user users.UserI, id string) error
	UploadLink(user users.UserI, id string, expire time.Duration) (*fileservice.Link, error)
	DownloadLink(user users.UserI, id string, expire time.Duration) (*fileservice.Link, error)
}

// ImagesControler business logic for images controller.
type ImagesControler struct {
	images      images.ImagesModelI
	fileStorage fileservice.FileServiceModelI
}

// NewImagesController new ImagesControler
func NewImagesController(images images.ImagesModelI,
	fileStorage fileservice.FileServiceModelI) *ImagesControler {
	return &ImagesControler{
		images:      images,
		fileStorage: fileStorage,
	}
}

func (i *ImagesControler) syncLastModifiedTimeWithFileUpload(user users.UserI, image *images.ImageMeta) error {
	lastModified, err := i.fileStorage.LastModified(user.GetCustomerID(), image.Id)
	switch {
	case err == fileservice.ErrNotFound:
		return nil
	case err != nil:
		return err
	}

	if image.LastUpdated.Before(lastModified) {
		image.LastUpdated = lastModified
		if err := i.images.Update(user, image); err != nil {
			return err
		}
	}

	return nil
}

// Lookup images
func (i *ImagesControler) Lookup(user users.UserI) ([]*images.ImageMeta, error) {
	images, err := i.images.Find(user)
	if err != nil {
		return nil, err
	}

	for _, image := range images {
		if err := i.syncLastModifiedTimeWithFileUpload(user, image); err != nil {
			return nil, err
		}
	}

	return images, nil
}

// Get image by id
func (i *ImagesControler) Get(user users.UserI, id string) (*images.ImageMeta, error) {
	image, err := i.images.FindOne(user, id)
	if err != nil {
		return nil, err
	}

	if err := i.syncLastModifiedTimeWithFileUpload(user, image); err != nil {
		return nil, err
	}

	return image, nil
}

// Create new image metadata entry
func (i *ImagesControler) Create(user users.UserI, public *images.ImageMetaPublic) (*images.ImageMeta, error) {

	image := images.NewImageMetaFromPublic(public)
	id, err := i.images.Insert(user, image)
	if err != nil {
		return nil, err
	}

	// ID is assigned on save
	image.Id = id

	return image, nil
}

// Edit public part of image metadata
func (i *ImagesControler) Edit(user users.UserI, id string, public *images.ImageMetaPublic) error {

	img, err := i.images.FindOne(user, id)
	if err != nil {
		return err
	}
	if img == nil {
		return ErrNotFound
	}

	updatedImg := images.NewImageMetaMerge(public, img.ImageMetaPrivate)
	if err := i.images.Update(user, updatedImg); err != nil {
		return err
	}

	return nil
}

// Delete image. Removes also image binary file.
func (i *ImagesControler) Delete(user users.UserI, id string) error {

	found, err := i.images.Exists(user, id)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}

	if found, err := i.fileStorage.Exists(user.GetCustomerID(), id); err != nil {
		return err
	} else if found {
		if err := i.fileStorage.Delete(user.GetCustomerID(), id); err != nil {
			return err
		}
	}

	if err := i.images.Delete(user, id); err != nil {
		return err
	}

	return nil
}

// UploadLink presigned PUT link to upload image file.
func (i *ImagesControler) UploadLink(user users.UserI, id string, expire time.Duration) (*fileservice.Link, error) {

	found, err := i.images.Exists(user, id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}

	link, err := i.fileStorage.PutRequest(user.GetCustomerID(), id, expire)
	if err != nil {
		return nil, err
	}

	return link, nil
}

// DownloadLink presigned GET link to download image file.
// Returns error if image have not been uploaded.
func (i *ImagesControler) DownloadLink(user users.UserI, id string, expire time.Duration) (*fileservice.Link, error) {

	var found bool
	var err error

	found, err = i.images.Exists(user, id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}

	found, err = i.fileStorage.Exists(user.GetCustomerID(), id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}

	link, err := i.fileStorage.PutRequest(user.GetCustomerID(), id, expire)
	if err != nil {
		return nil, err
	}

	return link, nil
}

// FindImageByApplicationAndModel searches matching image by application vesion and device model.
// Images without files uploaded will be excluded from the result.
func (i *ImagesControler) FindImageByApplicationAndModel(user users.UserI, version, model string) (*images.ImageMeta, error) {
	images, err := i.images.Find(user)
	if err != nil {
		return nil, err
	}

	fmt.Printf("images: %v\n", images)
	fmt.Printf("search: model: %v version: %v user: %v\n", model, version, user)

	for _, image := range images {
		// Check if image have been uploaded
		if image.Name == version && image.Model == model {
			found, err := i.fileStorage.Exists(user.GetCustomerID(), image.Id)
			fmt.Printf("image: %v %v %v\n", image, found, err)
			if err != nil {
				return nil, err
			}
			if !found {
				return nil, nil
			}

			return image, nil
		}
	}

	return nil, nil
}
