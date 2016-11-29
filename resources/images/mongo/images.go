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

package mongo

import (
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/mendersoftware/deployments/resources/images"
	"github.com/mendersoftware/deployments/resources/images/model"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Database KEYS
const (
	// Keys are corelated to field names in SoftwareImageMeta
	// and SoftwareImageMetaArtifact structures
	// Need to be kept in sync with that structure filed names
	StorageKeySoftwareImageDeviceTypes = "meta_yocto.device_types_compatible"
	StorageKeySoftwareImageName        = "meta.name"
	StorageKeySoftwareImageId          = "_id"
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

// SoftwareImagesStorage is a data layer for SoftwareImages based on MongoDB
// Implements model.SoftwareImagesStorage
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
		Key:    []string{StorageKeySoftwareImageName, StorageKeySoftwareImageDeviceTypes},
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
		return false, model.ErrSoftwareImagesStorageInvalidID
	}

	session := i.session.Copy()
	defer session.Close()

	var image *images.SoftwareImage
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
func (i *SoftwareImagesStorage) Update(image *images.SoftwareImage) (bool, error) {

	if err := image.Validate(); err != nil {
		return false, err
	}

	session := i.session.Copy()
	defer session.Close()

	image.SetModified(time.Now())
	if err := session.DB(DatabaseName).C(CollectionImages).UpdateId(image.Id, image); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// ImageByNameAndDeviceType find image with speficied application name and targed device type
func (i *SoftwareImagesStorage) ImageByNameAndDeviceType(name, deviceType string) (*images.SoftwareImage, error) {

	if govalidator.IsNull(name) {
		return nil, model.ErrSoftwareImagesStorageInvalidName

	}

	if govalidator.IsNull(deviceType) {
		return nil, model.ErrSoftwareImagesStorageInvalidDeviceType
	}

	// equal to device type & software version (application name + version)
	query := bson.M{
		StorageKeySoftwareImageDeviceTypes: deviceType,
		StorageKeySoftwareImageName:        name,
	}

	session := i.session.Copy()
	defer session.Close()

	// Both we lookup uniqe object, should be one or none.
	var image images.SoftwareImage
	if err := session.DB(DatabaseName).C(CollectionImages).Find(query).One(&image); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return &image, nil
}

// Insert persists object
func (i *SoftwareImagesStorage) Insert(image *images.SoftwareImage) error {

	if image == nil {
		return model.ErrSoftwareImagesStorageInvalidImage
	}

	if err := image.Validate(); err != nil {
		return err
	}

	session := i.session.Copy()
	defer session.Close()

	return session.DB(DatabaseName).C(CollectionImages).Insert(image)
}

// FindByID search storage for image with ID, returns nil if not found
func (i *SoftwareImagesStorage) FindByID(id string) (*images.SoftwareImage, error) {

	if govalidator.IsNull(id) {
		return nil, model.ErrSoftwareImagesStorageInvalidID
	}

	session := i.session.Copy()
	defer session.Close()

	var image *images.SoftwareImage
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
		return model.ErrSoftwareImagesStorageInvalidID
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
func (i *SoftwareImagesStorage) FindAll() ([]*images.SoftwareImage, error) {

	session := i.session.Copy()
	defer session.Close()

	var images []*images.SoftwareImage
	if err := session.DB(DatabaseName).C(CollectionImages).Find(nil).All(&images); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return images, nil
		}
		return nil, err
	}

	return images, nil
}
