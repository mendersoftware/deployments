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
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestCreateImageEmptyConstructor(t *testing.T) {
	iModel := NewImagesModel(nil, nil, nil)
	if _, err := iModel.CreateImage(nil); err != ErrModelMissingInputMetadata {
		t.FailNow()
	}
}

func TestCreateImageMissingFields(t *testing.T) {
	iModel := NewImagesModel(nil, nil, nil)

	image := NewSoftwareImageConstructor()
	if _, err := iModel.CreateImage(image); err == nil {
		t.FailNow()
	}
}

type FakeImageStorage struct {
	insertError      error
	findByIdError    error
	findByIdImage    *SoftwareImage
	deleteError      error
	findAllImages    []*SoftwareImage
	findAllError     error
	imageExists      bool
	imageEsistsError error
	update           bool
	updateError      error
}

func (fis *FakeImageStorage) Exists(id string) (bool, error) {
	return fis.imageExists, fis.imageEsistsError
}

func (fis *FakeImageStorage) Update(image *SoftwareImage) (bool, error) {
	return fis.update, fis.updateError
}

func (fis *FakeImageStorage) Insert(image *SoftwareImage) error {
	return fis.insertError
}

func (fis *FakeImageStorage) FindByID(id string) (*SoftwareImage, error) {
	return fis.findByIdImage, fis.findByIdError
}

func (fis *FakeImageStorage) Delete(id string) error {
	return fis.deleteError
}

func (fis *FakeImageStorage) FindAll() ([]*SoftwareImage, error) {
	return fis.findAllImages, fis.findAllError
}

func createValidImage() *SoftwareImageConstructor {
	image := NewSoftwareImageConstructor()
	required := "required"

	image.YoctoId = &required
	image.Name = &required
	image.DeviceType = &required

	return image
}

func TestCreateImageInsertError(t *testing.T) {
	fakeIS := new(FakeImageStorage)
	fakeIS.insertError = errors.New("insert error")

	iModel := NewImagesModel(nil, nil, fakeIS)
	image := createValidImage()

	if _, err := iModel.CreateImage(image); err == nil {
		t.FailNow()
	}
}

func TestCreateImageCreateOK(t *testing.T) {
	fakeIS := new(FakeImageStorage)
	fakeIS.insertError = nil

	iModel := NewImagesModel(nil, nil, fakeIS)

	image := createValidImage()

	if _, err := iModel.CreateImage(image); err != nil {
		t.FailNow()
	}
}

func TestGetImageFindByIDError(t *testing.T) {
	fakeIS := new(FakeImageStorage)
	fakeIS.findByIdError = errors.New("find by id error")

	iModel := NewImagesModel(nil, nil, fakeIS)
	if _, err := iModel.GetImage(""); err == nil {
		t.FailNow()
	}
}

func TestGetImageFindByIDEmptyImage(t *testing.T) {
	fakeIS := new(FakeImageStorage)
	fakeIS.findByIdImage = nil

	iModel := NewImagesModel(nil, nil, fakeIS)
	if image, err := iModel.GetImage(""); err != nil || image != nil {
		t.FailNow()
	}
}

type FakeFileStorage struct {
	lastModifiedTime  time.Time
	lastModifiedError error
	deleteError       error
	imageExists       bool
	imageEsistsError  error
	putReq            *Link
	putError          error
	getReq            *Link
	getError          error
}

func (ffs *FakeFileStorage) Delete(objectId string) error {
	return ffs.deleteError
}

func (ffs *FakeFileStorage) Exists(objectId string) (bool, error) {
	return ffs.imageExists, ffs.imageEsistsError
}

func (ffs *FakeFileStorage) LastModified(objectId string) (time.Time, error) {
	return ffs.lastModifiedTime, ffs.lastModifiedError
}

func (ffs *FakeFileStorage) PutRequest(objectId string, duration time.Duration) (*Link, error) {
	return ffs.putReq, ffs.putError
}

func (ffs *FakeFileStorage) GetRequest(objectId string, duration time.Duration) (*Link, error) {
	return ffs.getReq, ffs.getError
}

func TestSyncLastModifiedTimeWithFileUpload(t *testing.T) {
	fakeIS := new(FakeImageStorage)
	fakeIS.findByIdImage = nil

	fakeFS := new(FakeFileStorage)
	fakeFS.lastModifiedTime = time.Now()
	fakeFS.lastModifiedError = ErrFileStorageFileNotFound

	iModel := NewImagesModel(fakeFS, nil, fakeIS)

	image := createValidImage()
	constructorImage := NewSoftwareImageFromConstructor(image)
	now := time.Now()
	constructorImage.Modified = &now

	if err := iModel.syncLastModifiedTimeWithFileUpload(constructorImage); err != nil {
		t.FailNow()
	}

	fakeFS.lastModifiedError = errors.New("error")
	if err := iModel.syncLastModifiedTimeWithFileUpload(constructorImage); err == nil {
		t.FailNow()
	}

	fakeFS.lastModifiedError = nil
	if err := iModel.syncLastModifiedTimeWithFileUpload(constructorImage); err != nil {
		t.FailNow()
	}

	fakeFS.lastModifiedTime = time.Now()
	if err := iModel.syncLastModifiedTimeWithFileUpload(constructorImage); err != nil {
		t.FailNow()
	}
}

func TestGetImageOK(t *testing.T) {
	image := createValidImage()
	constructorImage := NewSoftwareImageFromConstructor(image)
	now := time.Now()
	constructorImage.Modified = &now

	fakeIS := new(FakeImageStorage)
	fakeIS.findByIdImage = constructorImage
	fakeFS := new(FakeFileStorage)
	fakeFS.lastModifiedTime = time.Now()

	iModel := NewImagesModel(fakeFS, nil, fakeIS)
	if image, err := iModel.GetImage(""); err != nil || image == nil {
		t.FailNow()
	}
}

type FakeUseChecker struct {
	usedInActiveDeploymentsErr error
	isUsedInActiveDeployment   bool
	usedInDeploymentsErr       error
	isUsedInDeployment         bool
}

func (fus *FakeUseChecker) ImageUsedInActiveDeployment(imageId string) (bool, error) {
	return fus.isUsedInActiveDeployment, fus.usedInActiveDeploymentsErr
}

func (fus *FakeUseChecker) ImageUsedInDeployment(imageId string) (bool, error) {
	return fus.isUsedInDeployment, fus.usedInDeploymentsErr
}

func TestDeleteImage(t *testing.T) {
	fakeChecker := new(FakeUseChecker)
	fakeFS := new(FakeFileStorage)
	fakeIS := new(FakeImageStorage)

	fakeChecker.usedInActiveDeploymentsErr = errors.New("error")

	iModel := NewImagesModel(fakeFS, fakeChecker, fakeIS)

	if err := iModel.DeleteImage(""); err == nil {
		t.FailNow()
	}

	fakeChecker.usedInActiveDeploymentsErr = nil
	fakeChecker.isUsedInActiveDeployment = true
	if err := iModel.DeleteImage(""); err != ErrModelImageInActiveDeployment {
		t.FailNow()
	}

	// we should delete image successfully
	fakeChecker.isUsedInActiveDeployment = false
	if err := iModel.DeleteImage(""); err != nil {
		t.FailNow()
	}

	fakeFS.deleteError = errors.New("error")
	if err := iModel.DeleteImage(""); err == nil {
		t.FailNow()
	}

	fakeFS.deleteError = nil
	fakeIS.deleteError = errors.New("error")
	if err := iModel.DeleteImage(""); err == nil {
		t.FailNow()
	}

}

func TestListImages(t *testing.T) {
	fakeChecker := new(FakeUseChecker)
	fakeFS := new(FakeFileStorage)
	fakeIS := new(FakeImageStorage)
	iModel := NewImagesModel(fakeFS, fakeChecker, fakeIS)

	fakeIS.findAllError = errors.New("error")
	if _, err := iModel.ListImages(nil); err == nil {
		t.FailNow()
	}

	//no error; empty images list
	fakeIS.findAllError = nil
	if _, err := iModel.ListImages(nil); err != nil {
		t.FailNow()
	}

	//have some valid image that will pass syncLastModifiedTimeWithFileUpload check
	image := createValidImage()
	constructorImage := NewSoftwareImageFromConstructor(image)
	now := time.Now()
	constructorImage.Modified = &now

	listedImages := []*SoftwareImage{constructorImage}
	fakeIS.findAllImages = listedImages
	if _, err := iModel.ListImages(nil); err != nil {
		t.FailNow()
	}

	//have some valid image that won't pass syncLastModifiedTimeWithFileUpload check
	fakeFS.lastModifiedError = errors.New("error")
	if _, err := iModel.ListImages(nil); err == nil {
		t.FailNow()
	}
}

func TestEditImage(t *testing.T) {
	image := createValidImage()

	fakeChecker := new(FakeUseChecker)
	fakeIS := new(FakeImageStorage)
	iModel := NewImagesModel(nil, fakeChecker, fakeIS)

	// error checking if image is used in deployments
	fakeChecker.usedInDeploymentsErr = errors.New("error")
	if _, err := iModel.EditImage("", image); err == nil {
		t.FailNow()
	}

	// image used in deployments
	fakeChecker.usedInDeploymentsErr = nil
	fakeChecker.isUsedInDeployment = true
	if _, err := iModel.EditImage("", image); err != ErrModelImageUsedInAnyDeployment {
		t.FailNow()
	}

	// not used in deployments; finding error
	fakeChecker.isUsedInDeployment = false
	fakeIS.findByIdError = errors.New("error")
	if _, err := iModel.EditImage("", image); err == nil {
		t.FailNow()
	}

	// not used in deployments; cannot find image
	fakeIS.findByIdError = nil
	fakeIS.findByIdImage = nil
	if image, err := iModel.EditImage("", image); err != nil || image == true {
		t.FailNow()
	}

	// image does not exists
	constructorImage := NewSoftwareImageFromConstructor(image)
	fakeIS.findByIdImage = constructorImage
	fakeIS.updateError = errors.New("error")
	if _, err := iModel.EditImage("", image); err == nil {
		t.FailNow()
	}

	// update OK
	fakeIS.updateError = nil
	if image, err := iModel.EditImage("", image); err != nil || !image {
		t.FailNow()
	}
}

func TestUploadLink(t *testing.T) {
	fakeChecker := new(FakeUseChecker)
	fakeIS := new(FakeImageStorage)
	fakeFS := new(FakeFileStorage)
	iModel := NewImagesModel(fakeFS, fakeChecker, fakeIS)

	// image exists error
	fakeIS.imageEsistsError = errors.New("error")
	if _, err := iModel.UploadLink("iamge", time.Hour); err == nil {
		t.FailNow()
	}

	// iamge does not esists
	fakeIS.imageEsistsError = nil
	fakeIS.imageExists = false
	if link, err := iModel.UploadLink("iamge", time.Hour); err != nil || link != nil {
		t.FailNow()
	}

	// can not generate link
	fakeIS.imageExists = true
	fakeFS.putError = errors.New("error")
	if _, err := iModel.UploadLink("iamge", time.Hour); err == nil {
		t.FailNow()
	}

	// upload link generation success
	fakeFS.putError = nil
	link := NewLink("uri", time.Now())
	fakeFS.putReq = link

	receivedLink, err := iModel.UploadLink("image", time.Hour)
	if err != nil || !reflect.DeepEqual(link, receivedLink) {
		t.FailNow()
	}

}

func TestDownloadLink(t *testing.T) {
	fakeChecker := new(FakeUseChecker)
	fakeIS := new(FakeImageStorage)
	fakeFS := new(FakeFileStorage)
	iModel := NewImagesModel(fakeFS, fakeChecker, fakeIS)

	// image exists error
	fakeIS.imageEsistsError = errors.New("error")
	if _, err := iModel.DownloadLink("iamge", time.Hour); err == nil {
		t.FailNow()
	}

	// iamge does not esists
	fakeIS.imageEsistsError = nil
	fakeIS.imageExists = false
	if link, err := iModel.DownloadLink("iamge", time.Hour); err != nil || link != nil {
		t.FailNow()
	}

	// can not generate link
	fakeIS.imageExists = true
	fakeFS.imageExists = true
	fakeFS.getError = errors.New("error")
	if _, err := iModel.DownloadLink("iamge", time.Hour); err == nil {
		t.FailNow()
	}

	// upload link generation success
	fakeFS.getError = nil
	link := NewLink("uri", time.Now())
	fakeFS.getReq = link

	receivedLink, err := iModel.DownloadLink("image", time.Hour)
	if err != nil || !reflect.DeepEqual(link, receivedLink) {
		t.FailNow()
	}
}
