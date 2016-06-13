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

	"github.com/pkg/errors"
)

var (
	ErrModelMissingInputMetadata     = errors.New("Missing input metadata")
	ErrModelImageInActiveDeployment  = errors.New("Image is used in active deployment and cannot be removed")
	ErrModelImageUsedInAnyDeployment = errors.New("Image have been already used in deployment")
)

type SoftwareImagesStorager interface {
	Exists(id string) (bool, error)
	Update(image *SoftwareImage) (bool, error)
	Insert(image *SoftwareImage) error
	FindByID(id string) (*SoftwareImage, error)
	Delete(id string) error
	FindAll() ([]*SoftwareImage, error)
}

type FileStorager interface {
	Delete(objectId string) error
	Exists(objectId string) (bool, error)
	LastModified(objectId string) (time.Time, error)
	PutRequest(objectId string, duration time.Duration) (*Link, error)
	GetRequest(objectId string, duration time.Duration) (*Link, error)
}

type ImageInUseChecker interface {
	ImageUsedInActiveDeployment(imageId string) (bool, error)
	ImageUsedInDeployment(imageId string) (bool, error)
}

type ImagesModel struct {
	fileStorage   FileStorager
	deployments   ImageInUseChecker
	imagesStorage SoftwareImagesStorager
}

func NewImagesModel(
	fileStorage FileStorager,
	checker ImageInUseChecker,
	imagesStorage SoftwareImagesStorager,
) *ImagesModel {
	return &ImagesModel{
		fileStorage:   fileStorage,
		deployments:   checker,
		imagesStorage: imagesStorage,
	}
}

func (i *ImagesModel) CreateImage(constructor *SoftwareImageConstructor) (string, error) {

	if constructor == nil {
		return "", ErrModelMissingInputMetadata
	}

	if err := constructor.Validate(); err != nil {
		return "", errors.Wrap(err, "Validating image metadata")
	}

	image := NewSoftwareImageFromConstructor(constructor)

	if err := i.imagesStorage.Insert(image); err != nil {
		return "", errors.Wrap(err, "Storing image metadata")
	}

	return *image.Id, nil
}

// GetImage allows to fetch image obeject with specified id
// On each lookup it syncs last file upload time with metadata in case file was uploaded or reuploaded
// Nil if not found
func (i *ImagesModel) GetImage(id string) (*SoftwareImage, error) {

	image, err := i.imagesStorage.FindByID(id)
	if err != nil {
		return nil, errors.Wrap(err, "Searching for image with specified ID")
	}

	if image == nil {
		return nil, nil
	}

	if err := i.syncLastModifiedTimeWithFileUpload(image); err != nil {
		return nil, errors.Wrap(err, "Synchronizing image upload time")
	}

	return image, nil
}

// DeleteImage removes metadata and image file
// Noop for not exisitng images
// Allowed to remove image only if image is not scheduled or in progress for an updates - then image file is needed
// In case of already finished updates only image file is not needed, metadata is attached directly to device deployment
// therefore we still have some information about image that have been used (but not the file)
func (i *ImagesModel) DeleteImage(imageID string) error {

	inUse, err := i.deployments.ImageUsedInActiveDeployment(imageID)
	if err != nil {
		return errors.Wrap(err, "Checking if image is used in active deployment")
	}

	// Image is in use, not allowed to delete
	if inUse {
		return ErrModelImageInActiveDeployment
	}

	// Delete image file (call to external service)
	// Noop for not existing file
	if err := i.fileStorage.Delete(imageID); err != nil {
		return errors.Wrap(err, "Deleting image file")
	}

	// Delete metadata
	if err := i.imagesStorage.Delete(imageID); err != nil {
		return errors.Wrap(err, "Deleting image metadata")
	}

	return nil
}

// ListImages according to specified filers.
// On each lookup it syncs last file upload time with metadata in case file was uploaded or reuploaded
func (i *ImagesModel) ListImages(filters map[string]string) ([]*SoftwareImage, error) {

	images, err := i.imagesStorage.FindAll()
	if err != nil {
		return nil, errors.Wrap(err, "Searching for image metadata")
	}

	if images == nil {
		return make([]*SoftwareImage, 0), nil
	}

	for _, image := range images {
		if err := i.syncLastModifiedTimeWithFileUpload(image); err != nil {
			return nil, errors.Wrap(err, "Synchronizing image upload time")
		}
	}

	return images, nil
}

// Sync file upload time with last modified time of image metadata.
// Need to check when image was uploaded and if it was overwritten
// Ugly but required by frontend design, in future can be split.
// Expensive! Will go away with switching to one-step file upload.
func (i *ImagesModel) syncLastModifiedTimeWithFileUpload(image *SoftwareImage) error {

	uploaded, err := i.fileStorage.LastModified(*image.Id)
	if err != nil {
		if errors.Cause(err).Error() == ErrFileStorageFileNotFound.Error() {
			return nil
		}

		return errors.Wrap(err, "Cheking last modified time for image file")
	}

	if image.Modified.Before(uploaded) {
		image.Modified = &uploaded
		if _, err := i.imagesStorage.Update(image); err != nil {
			return errors.Wrap(err, "Updating image metadata")
		}
	}

	return nil
}

// EditObject allows editing only if image have not been used yet in any deployment.
func (i *ImagesModel) EditImage(imageID string, constructor *SoftwareImageConstructor) (bool, error) {

	if err := constructor.Validate(); err != nil {
		return false, errors.Wrap(err, "Validating image metadata")
	}

	found, err := i.deployments.ImageUsedInDeployment(imageID)
	if err != nil {
		return false, errors.Wrap(err, "Searching for usage of the image among deployments")
	}

	if found {
		return false, ErrModelImageUsedInAnyDeployment
	}

	foundImage, err := i.imagesStorage.FindByID(imageID)
	if err != nil {
		return false, errors.Wrap(err, "Searching for image with specified ID")
	}

	if foundImage == nil {
		return false, nil
	}

	foundImage.SoftwareImageConstructor = constructor
	foundImage.SetModified(time.Now())

	_, err = i.imagesStorage.Update(foundImage)
	if err != nil {
		return false, errors.Wrap(err, "Updating image matadata")
	}

	return true, nil
}

// UploadLink generated presigned PUT link to upload image file.
// Image meta has to be created first.
func (i *ImagesModel) UploadLink(imageID string, expire time.Duration) (*Link, error) {

	found, err := i.imagesStorage.Exists(imageID)
	if err != nil {
		return nil, errors.Wrap(err, "Searching for image with specified ID")
	}

	if !found {
		return nil, nil
	}

	link, err := i.fileStorage.PutRequest(imageID, expire)
	if err != nil {
		return nil, errors.Wrap(err, "Generating upload link")
	}

	return link, nil
}

// DownloadLink presigned GET link to download image file.
// Returns error if image have not been uploaded.
func (i *ImagesModel) DownloadLink(imageID string, expire time.Duration) (*Link, error) {

	found, err := i.imagesStorage.Exists(imageID)
	if err != nil {
		return nil, errors.Wrap(err, "Searching for image with specified ID")
	}

	if !found {
		return nil, nil
	}

	found, err = i.fileStorage.Exists(imageID)
	if err != nil {
		return nil, errors.Wrap(err, "Searching for image file")
	}

	if !found {
		return nil, nil
	}

	link, err := i.fileStorage.GetRequest(imageID, expire)
	if err != nil {
		return nil, errors.Wrap(err, "Generating download link")
	}

	return link, nil
}
