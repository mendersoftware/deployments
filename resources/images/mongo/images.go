// Copyright 2019 Northern.tech AS
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
	"context"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/mendersoftware/go-lib-micro/store"

	"github.com/mendersoftware/deployments/model"
	dmodel "github.com/mendersoftware/deployments/resources/images/model"
)

// Database KEYS
const (
	// Keys are corelated to field names in SoftwareImageMeta
	// and SoftwareImageMetaArtifact structures
	// Need to be kept in sync with that structure filed names
	StorageKeySoftwareImageDeviceTypes = "meta_artifact.device_types_compatible"
	StorageKeySoftwareImageName        = "meta_artifact.name"
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

// Ensure required indexes exists; create if not.
func (i *SoftwareImagesStorage) ensureIndexing(ctx context.Context, session *mgo.Session) error {

	uniqueNameVersionIndex := mgo.Index{
		Key:    []string{StorageKeySoftwareImageName, StorageKeySoftwareImageDeviceTypes},
		Unique: true,
		Name:   IndexUniqeNameAndDeviceTypeStr,
		// Build index upfront - make sure this index is allways on.
		Background: false,
	}

	return session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).EnsureIndex(uniqueNameVersionIndex)
}

// Exists checks if object with ID exists
func (i *SoftwareImagesStorage) Exists(ctx context.Context, id string) (bool, error) {

	if govalidator.IsNull(id) {
		return false, dmodel.ErrSoftwareImagesStorageInvalidID
	}

	session := i.session.Copy()
	defer session.Close()

	var image *model.SoftwareImage
	if err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).FindId(id).One(&image); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Update proviced SoftwareImage
// Return false if not found
func (i *SoftwareImagesStorage) Update(ctx context.Context,
	image *model.SoftwareImage) (bool, error) {

	if err := image.Validate(); err != nil {
		return false, err
	}

	session := i.session.Copy()
	defer session.Close()

	image.SetModified(time.Now())
	if err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).UpdateId(image.Id, image); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// ImageByNameAndDeviceType finds image with speficied application name and targed device type
func (i *SoftwareImagesStorage) ImageByNameAndDeviceType(ctx context.Context,
	name, deviceType string) (*model.SoftwareImage, error) {

	if govalidator.IsNull(name) {
		return nil, dmodel.ErrSoftwareImagesStorageInvalidName

	}

	if govalidator.IsNull(deviceType) {
		return nil, dmodel.ErrSoftwareImagesStorageInvalidDeviceType
	}

	// equal to device type & software version (application name + version)
	query := bson.M{
		StorageKeySoftwareImageDeviceTypes: deviceType,
		StorageKeySoftwareImageName:        name,
	}

	session := i.session.Copy()
	defer session.Close()

	// Both we lookup uniqe object, should be one or none.
	var image model.SoftwareImage
	if err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).Find(query).One(&image); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return &image, nil
}

// ImageByIdsAndDeviceType finds image with id from ids and targed device type
func (i *SoftwareImagesStorage) ImageByIdsAndDeviceType(ctx context.Context,
	ids []string, deviceType string) (*model.SoftwareImage, error) {

	if govalidator.IsNull(deviceType) {
		return nil, dmodel.ErrSoftwareImagesStorageInvalidDeviceType
	}

	if len(ids) == 0 {
		return nil, dmodel.ErrSoftwareImagesStorageInvalidID
	}

	query := bson.M{
		StorageKeySoftwareImageDeviceTypes: deviceType,
		StorageKeySoftwareImageId:          bson.M{"$in": ids},
	}

	session := i.session.Copy()
	defer session.Close()

	// Both we lookup uniqe object, should be one or none.
	var image model.SoftwareImage
	if err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).Find(query).One(&image); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return &image, nil
}

// ImagesByName finds images with speficied artifact name
func (i *SoftwareImagesStorage) ImagesByName(
	ctx context.Context, name string) ([]*model.SoftwareImage, error) {

	if govalidator.IsNull(name) {
		return nil, dmodel.ErrSoftwareImagesStorageInvalidName

	}

	// equal to artifact name
	query := bson.M{
		StorageKeySoftwareImageName: name,
	}

	session := i.session.Copy()
	defer session.Close()

	// Both we lookup uniqe object, should be one or none.
	var images []*model.SoftwareImage
	if err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).Find(query).All(&images); err != nil {
		return nil, err
	}

	return images, nil
}

// Insert persists object
func (i *SoftwareImagesStorage) Insert(ctx context.Context, image *model.SoftwareImage) error {

	if image == nil {
		return dmodel.ErrSoftwareImagesStorageInvalidImage
	}

	if err := image.Validate(); err != nil {
		return err
	}

	session := i.session.Copy()
	defer session.Close()

	if err := i.ensureIndexing(ctx, session); err != nil {
		return err
	}

	return session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).Insert(image)
}

// FindByID search storage for image with ID, returns nil if not found
func (i *SoftwareImagesStorage) FindByID(ctx context.Context,
	id string) (*model.SoftwareImage, error) {

	if govalidator.IsNull(id) {
		return nil, dmodel.ErrSoftwareImagesStorageInvalidID
	}

	session := i.session.Copy()
	defer session.Close()

	var image *model.SoftwareImage
	if err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).FindId(id).One(&image); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return image, nil
}

// IsArtifactUnique checks if there is no artifact with the same artifactName
// supporting one of the device types from deviceTypesCompatible list.
// Returns true, nil if artifact is unique;
// false, nil if artifact is not unique;
// false, error in case of error.
func (i *SoftwareImagesStorage) IsArtifactUnique(ctx context.Context,
	artifactName string, deviceTypesCompatible []string) (bool, error) {

	if govalidator.IsNull(artifactName) {
		return false, dmodel.ErrSoftwareImagesStorageInvalidArtifactName
	}

	session := i.session.Copy()
	defer session.Close()

	query := bson.M{
		"$and": []bson.M{
			{
				StorageKeySoftwareImageName: artifactName,
			},
			{
				StorageKeySoftwareImageDeviceTypes: bson.M{"$in": deviceTypesCompatible},
			},
		},
	}

	var image *model.SoftwareImage
	if err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).Find(query).One(&image); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return true, nil
		}
		return false, err
	}

	return false, nil
}

// Delete image specified by ID
// Noop on if not found.
func (i *SoftwareImagesStorage) Delete(ctx context.Context, id string) error {

	if govalidator.IsNull(id) {
		return dmodel.ErrSoftwareImagesStorageInvalidID
	}

	session := i.session.Copy()
	defer session.Close()

	if err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).RemoveId(id); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil
		}
		return err
	}

	return nil
}

// FindAll lists all images
func (i *SoftwareImagesStorage) FindAll(ctx context.Context) ([]*model.SoftwareImage, error) {

	session := i.session.Copy()
	defer session.Close()

	var images []*model.SoftwareImage
	if err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).Find(nil).All(&images); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return images, nil
		}
		return nil, err
	}

	return images, nil
}
