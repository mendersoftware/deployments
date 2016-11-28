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

package model

import (
	"os"
	"time"

	"github.com/mendersoftware/deployments/resources/images"
	"github.com/mendersoftware/deployments/resources/images/controller"
	"github.com/pkg/errors"
)

var (
	ErrModelMissingInputMetadata     = errors.New("Missing input metadata")
	ErrModelInvalidMetadata          = errors.New("Metadata invalid")
	ErrModelImageInActiveDeployment  = errors.New("Image is used in active deployment and cannot be removed")
	ErrModelImageUsedInAnyDeployment = errors.New("Image have been already used in deployment")
)

const (
	ImageContentType = "application/vnd.mender-artifact"
)

type ImagesModel struct {
	fileStorage   FileStorage
	deployments   ImageUsedIn
	imagesStorage SoftwareImagesStorage
}

func NewImagesModel(
	fileStorage FileStorage,
	checker ImageUsedIn,
	imagesStorage SoftwareImagesStorage,
) *ImagesModel {
	return &ImagesModel{
		fileStorage:   fileStorage,
		deployments:   checker,
		imagesStorage: imagesStorage,
	}
}

func (i *ImagesModel) CreateImage(
	imageFile *os.File,
	metaConstructor *images.SoftwareImageMetaConstructor,
	metaArtifactConstructor *images.SoftwareImageMetaArtifactConstructor) (string, error) {

	if metaConstructor == nil || metaArtifactConstructor == nil {
		return "", ErrModelMissingInputMetadata
	}

	if err := metaConstructor.Validate(); err != nil {
		return "", ErrModelInvalidMetadata
	}
	if err := metaArtifactConstructor.Validate(); err != nil {
		return "", ErrModelInvalidMetadata
	}

	image := images.NewSoftwareImage(metaConstructor, metaArtifactConstructor)

	if err := i.imagesStorage.Insert(image); err != nil {
		return "", errors.Wrap(err, "Fail to store the metadata")
	}

	if err := i.fileStorage.PutFile(image.Id, imageFile, ImageContentType); err != nil {
		i.imagesStorage.Delete(image.Id)
		return "", errors.Wrap(err, "Fail to store the image")
	}

	return image.Id, nil
}

// GetImage allows to fetch image obeject with specified id
// Nil if not found
func (i *ImagesModel) GetImage(id string) (*images.SoftwareImage, error) {

	image, err := i.imagesStorage.FindByID(id)
	if err != nil {
		return nil, errors.Wrap(err, "Searching for image with specified ID")
	}

	if image == nil {
		return nil, nil
	}

	return image, nil
}

// DeleteImage removes metadata and image file
// Noop for not exisitng images
// Allowed to remove image only if image is not scheduled or in progress for an updates - then image file is needed
// In case of already finished updates only image file is not needed, metadata is attached directly to device deployment
// therefore we still have some information about image that have been used (but not the file)
func (i *ImagesModel) DeleteImage(imageID string) error {
	found, err := i.GetImage(imageID)

	if err != nil {
		return errors.Wrap(err, "Getting image metadata")
	}

	if found == nil {
		return controller.ErrImageMetaNotFound
	}

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
func (i *ImagesModel) ListImages(filters map[string]string) ([]*images.SoftwareImage, error) {

	imageList, err := i.imagesStorage.FindAll()
	if err != nil {
		return nil, errors.Wrap(err, "Searching for image metadata")
	}

	if imageList == nil {
		return make([]*images.SoftwareImage, 0), nil
	}

	return imageList, nil
}

// EditObject allows editing only if image have not been used yet in any deployment.
func (i *ImagesModel) EditImage(imageID string, constructor *images.SoftwareImageMetaConstructor) (bool, error) {

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

	foundImage.SetModified(time.Now())

	_, err = i.imagesStorage.Update(foundImage)
	if err != nil {
		return false, errors.Wrap(err, "Updating image matadata")
	}

	return true, nil
}

// DownloadLink presigned GET link to download image file.
// Returns error if image have not been uploaded.
func (i *ImagesModel) DownloadLink(imageID string, expire time.Duration) (*images.Link, error) {

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

	link, err := i.fileStorage.GetRequest(imageID, expire, ImageContentType)
	if err != nil {
		return nil, errors.Wrap(err, "Generating download link")
	}

	return link, nil
}
