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

package images

import (
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Database KEYS
const (
	// Keys are corelated to field names in SoftwareImage structure
	// Need to be kept in sync with that structure filed names
	StorageKeySoftwareImageDeviceType = "softwareimageconstructor.devicetype"
	StorageKeySoftwareImageName       = "softwareimageconstructor.name"
	StorageKeySoftwareImageId         = "_id"
)

// Indexes
const (
	IndexUniqeNameAndDeviceTypeStr = "uniqueNameAndDeviceTypeIndex"
)

// Database
const (
	DatabaseName     = "deployment_service"
	CollectionImages = "images"
)

// Errors
var (
	ErrStorageInvalidID         = errors.New("Invalid id")
	ErrStorageInvalidName       = errors.New("Invalid name")
	ErrStorageInvalidDeviceType = errors.New("Invalid device type")
	ErrStorageInvalidImage      = errors.New("Invalid image")
)

// SoftwareImagesStorage is a data layer for SoftwareImages based on MongoDB
type SoftwareImagesStorage struct {
	session *mgo.Session
}

// NewSoftwareImagesStorage new data layer object
func NewSoftwareImagesStorage(session *mgo.Session) *SoftwareImagesStorage {

	return &SoftwareImagesStorage{
		session: session,
	}
}

// IndexStorage set required indexes.
// * Set unique index on name-model image keys.
func (i *SoftwareImagesStorage) IndexStorage() error {

	session := i.session.Copy()
	defer session.Close()

	uniqueNameVersionIndex := mgo.Index{
		Key:    []string{StorageKeySoftwareImageName, StorageKeySoftwareImageDeviceType},
		Unique: true,
		Name:   IndexUniqeNameAndDeviceTypeStr,
		// Build index upfront - make sure this index is allways on.
		Background: false,
	}

	return session.DB(DatabaseName).C(CollectionImages).EnsureIndex(uniqueNameVersionIndex)
}

// Exists checks if object with ID exists
func (i *SoftwareImagesStorage) Exists(id string) (bool, error) {

	if govalidator.IsNull(id) {
		return false, ErrStorageInvalidID
	}

	session := i.session.Copy()
	defer session.Close()

	var image *SoftwareImage
	if err := session.DB(DatabaseName).C(CollectionImages).FindId(id).One(&image); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Update proviced SoftwareImage
// Return false if not found
func (i *SoftwareImagesStorage) Update(image *SoftwareImage) (bool, error) {

	if err := image.Validate(); err != nil {
		return false, err
	}

	session := i.session.Copy()
	defer session.Close()

	image.SetModified(time.Now())
	if err := session.DB(DatabaseName).C(CollectionImages).UpdateId(*image.Id, image); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// ImageByNameAndDeviceType find image with speficied application name and targed device type
func (i *SoftwareImagesStorage) ImageByNameAndDeviceType(name, deviceType string) (*SoftwareImage, error) {

	if govalidator.IsNull(name) {
		return nil, ErrStorageInvalidName
	}

	if govalidator.IsNull(deviceType) {
		return nil, ErrStorageInvalidDeviceType
	}

	// equal to device type & software version (application name + version)
	query := bson.M{
		StorageKeySoftwareImageDeviceType: deviceType,
		StorageKeySoftwareImageName:       name,
	}

	session := i.session.Copy()
	defer session.Close()

	// Both we lookup uniqe object, should be one or none.
	var image SoftwareImage
	if err := session.DB(DatabaseName).C(CollectionImages).Find(query).One(&image); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return &image, nil
}

// Insert persists object
func (i *SoftwareImagesStorage) Insert(image *SoftwareImage) error {

	if image == nil {
		return ErrStorageInvalidImage
	}

	if err := image.Validate(); err != nil {
		return err
	}

	session := i.session.Copy()
	defer session.Close()

	return session.DB(DatabaseName).C(CollectionImages).Insert(image)
}

// FindByID search storage for image with ID, returns nil if not found
func (i *SoftwareImagesStorage) FindByID(id string) (*SoftwareImage, error) {

	if govalidator.IsNull(id) {
		return nil, ErrStorageInvalidID
	}

	session := i.session.Copy()
	defer session.Close()

	var image *SoftwareImage
	if err := session.DB(DatabaseName).C(CollectionImages).FindId(id).One(&image); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return image, nil
}

// Delete image specified by ID
// Noop on if not found.
func (i *SoftwareImagesStorage) Delete(id string) error {

	if govalidator.IsNull(id) {
		return ErrStorageInvalidID
	}

	session := i.session.Copy()
	defer session.Close()

	if err := session.DB(DatabaseName).C(CollectionImages).RemoveId(id); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil
		}
		return err
	}

	return nil
}

// FindAll lists all images
func (i *SoftwareImagesStorage) FindAll() ([]*SoftwareImage, error) {

	session := i.session.Copy()
	defer session.Close()

	var images []*SoftwareImage
	if err := session.DB(DatabaseName).C(CollectionImages).Find(nil).All(&images); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return images, nil
		}
		return nil, err
	}

	return images, nil
}
