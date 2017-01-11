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
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"
	"time"

	"github.com/mendersoftware/deployments/resources/images"
	"github.com/mendersoftware/deployments/resources/images/controller"
	"github.com/mendersoftware/mender-artifact/parser"
	atutils "github.com/mendersoftware/mender-artifact/test_utils"
	"github.com/mendersoftware/mender-artifact/writer"
	"github.com/stretchr/testify/assert"
)

const validUUIDv4 = "d50eda0d-2cea-4de1-8d42-9cd3e7e8670d"

func TestCreateImageEmptyConstructor(t *testing.T) {
	iModel := NewImagesModel(nil, nil, nil)
	if _, err := iModel.CreateImage(nil, nil); err != controller.ErrModelMissingInputMetadata {
		t.FailNow()
	}
}

func TestCreateImageMissingFields(t *testing.T) {
	iModel := NewImagesModel(nil, nil, nil)

	imageMeta := images.NewSoftwareImageMetaConstructor()
	if _, err := iModel.CreateImage(imageMeta, nil); err == nil {
		t.FailNow()
	}
}

type FakeImageStorage struct {
	insertError           error
	findByIdError         error
	findByIdImage         *images.SoftwareImage
	deleteError           error
	findAllImages         []*images.SoftwareImage
	findAllError          error
	imageExists           bool
	imageEsistsError      error
	update                bool
	updateError           error
	uploadArtifactError   error
	isArtifactUnique      bool
	isArtifactUniqueError error
}

func (fis *FakeImageStorage) Exists(id string) (bool, error) {
	return fis.imageExists, fis.imageEsistsError
}

func (fis *FakeImageStorage) Update(image *images.SoftwareImage) (bool, error) {
	return fis.update, fis.updateError
}

func (fis *FakeImageStorage) Insert(image *images.SoftwareImage) error {
	return fis.insertError
}

func (fis *FakeImageStorage) FindByID(id string) (*images.SoftwareImage, error) {
	return fis.findByIdImage, fis.findByIdError
}

func (fis *FakeImageStorage) Delete(id string) error {
	return fis.deleteError
}

func (fis *FakeImageStorage) FindAll() ([]*images.SoftwareImage, error) {
	return fis.findAllImages, fis.findAllError
}

func (fis *FakeImageStorage) IsArtifactUnique(artifactName string, deviceTypesCompatible []string) (bool, error) {
	return fis.isArtifactUnique, fis.isArtifactUniqueError
}

func createValidImageMeta() *images.SoftwareImageMetaConstructor {
	imageMeta := images.NewSoftwareImageMetaConstructor()
	required := "required"

	imageMeta.Name = required

	return imageMeta
}

func createValidImageMetaArtifact() *images.SoftwareImageMetaArtifactConstructor {
	imageMetaArtifact := images.NewSoftwareImageMetaArtifactConstructor()
	required := "required"

	imageMetaArtifact.DeviceTypesCompatible = []string{"required"}
	imageMetaArtifact.ArtifactName = required
	imageMetaArtifact.Info = &images.ArtifactInfo{
		Format:  required,
		Version: 1,
	}
	return imageMetaArtifact
}

func TestCreateImageInsertError(t *testing.T) {
	fakeIS := new(FakeImageStorage)
	fakeIS.insertError = errors.New("insert error")

	iModel := NewImagesModel(nil, nil, fakeIS)
	imageMeta := createValidImageMeta()

	if _, err := iModel.CreateImage(imageMeta, nil); err == nil {
		t.FailNow()
	}
}

func TestCreateImageArtifactUploadError(t *testing.T) {
	fakeIS := new(FakeImageStorage)
	fakeIS.insertError = nil
	fakeFS := new(FakeFileStorage)
	fakeFS.uploadArtifactError = errors.New("Cannot upload artifact")

	iModel := NewImagesModel(fakeFS, nil, fakeIS)

	imageMeta := createValidImageMeta()
	td, _ := ioutil.TempDir("", "mender-install-update-")
	defer os.RemoveAll(td)
	upath, err := makeFakeUpdate(t, path.Join(td, "update-root"), true)
	if err != nil {
		t.FailNow()
	}
	f, err := os.Open(upath)
	if err != nil {
		t.FailNow()
	}
	defer f.Close()
	if _, err := iModel.CreateImage(imageMeta, f); err == nil {
		t.FailNow()
	}
}

func TestCreateImageCreateOK(t *testing.T) {
	fakeIS := new(FakeImageStorage)
	fakeIS.insertError = nil
	fakeIS.isArtifactUnique = true
	fakeFS := new(FakeFileStorage)

	iModel := NewImagesModel(fakeFS, nil, fakeIS)

	imageMeta := createValidImageMeta()
	td, _ := ioutil.TempDir("", "mender-install-update-")
	defer os.RemoveAll(td)
	upath, err := makeFakeUpdate(t, path.Join(td, "update-root"), true)
	if err != nil {
		t.FailNow()
	}
	f, err := os.Open(upath)
	defer f.Close()
	if err != nil {
		t.FailNow()
	}

	if _, err := iModel.CreateImage(imageMeta, f); err != nil {
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
	lastModifiedTime    time.Time
	lastModifiedError   error
	deleteError         error
	imageExists         bool
	imageEsistsError    error
	putReq              *images.Link
	putError            error
	getReq              *images.Link
	getError            error
	uploadArtifactError error
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

func (ffs *FakeFileStorage) PutRequest(objectId string, duration time.Duration) (*images.Link, error) {
	return ffs.putReq, ffs.putError
}

func (ffs *FakeFileStorage) GetRequest(objectId string, duration time.Duration, responseContentType string) (*images.Link, error) {
	return ffs.getReq, ffs.getError
}

func (fis *FakeFileStorage) UploadArtifact(id string, img io.Reader, contentType string) error {
	if _, err := io.Copy(ioutil.Discard, img); err != nil {
		return err
	}
	return fis.uploadArtifactError
}

func TestGetImageOK(t *testing.T) {
	imageMeta := createValidImageMeta()
	imageMetaArtifact := createValidImageMetaArtifact()
	constructorImage := images.NewSoftwareImage(validUUIDv4, imageMeta, imageMetaArtifact)
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
	imageMeta := createValidImageMeta()
	imageMetaArtifact := createValidImageMetaArtifact()
	constructorImage := images.NewSoftwareImage(validUUIDv4, imageMeta, imageMetaArtifact)

	fakeFS := new(FakeFileStorage)
	fakeChecker := new(FakeUseChecker)
	fakeIS := new(FakeImageStorage)

	fakeIS.findByIdImage = constructorImage

	fakeChecker.usedInActiveDeploymentsErr = errors.New("error")

	iModel := NewImagesModel(fakeFS, fakeChecker, fakeIS)

	if err := iModel.DeleteImage(""); err == nil {
		t.FailNow()
	}

	fakeChecker.usedInActiveDeploymentsErr = nil
	fakeChecker.isUsedInActiveDeployment = true
	if err := iModel.DeleteImage(""); err != controller.ErrModelImageInActiveDeployment {
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

	fakeIS.deleteError = errors.New("error")
	fakeIS.findByIdImage = nil

	if err := iModel.DeleteImage(""); err == nil {
		t.FailNow()
	}

	fakeFS.getError = errors.New("error")
	fakeChecker.isUsedInActiveDeployment = false
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

	//have some valid image
	imageMeta := createValidImageMeta()
	imageMetaArtifact := createValidImageMetaArtifact()
	constructorImage := images.NewSoftwareImage(validUUIDv4, imageMeta, imageMetaArtifact)
	now := time.Now()
	constructorImage.Modified = &now

	listedImages := []*images.SoftwareImage{constructorImage}
	fakeIS.findAllImages = listedImages
	if _, err := iModel.ListImages(nil); err != nil {
		t.FailNow()
	}
}

func TestEditImage(t *testing.T) {
	imageMeta := createValidImageMeta()
	imageMetaArtifact := createValidImageMetaArtifact()

	fakeChecker := new(FakeUseChecker)
	fakeIS := new(FakeImageStorage)
	iModel := NewImagesModel(nil, fakeChecker, fakeIS)

	// error checking if image is used in deployments
	fakeChecker.usedInDeploymentsErr = errors.New("error")
	if _, err := iModel.EditImage("", imageMeta); err == nil {
		t.FailNow()
	}

	// image used in deployments
	fakeChecker.usedInDeploymentsErr = nil
	fakeChecker.isUsedInDeployment = true
	if _, err := iModel.EditImage("", imageMeta); err != controller.ErrModelImageUsedInAnyDeployment {
		t.FailNow()
	}

	// not used in deployments; finding error
	fakeChecker.isUsedInDeployment = false
	fakeIS.findByIdError = errors.New("error")
	if _, err := iModel.EditImage("", imageMeta); err == nil {
		t.FailNow()
	}

	// not used in deployments; cannot find image
	fakeIS.findByIdError = nil
	fakeIS.findByIdImage = nil
	if imageMeta, err := iModel.EditImage("", imageMeta); err != nil || imageMeta == true {
		t.FailNow()
	}

	// image does not exists
	constructorImage := images.NewSoftwareImage(validUUIDv4, imageMeta, imageMetaArtifact)
	fakeIS.findByIdImage = constructorImage
	fakeIS.updateError = errors.New("error")
	if _, err := iModel.EditImage("", imageMeta); err == nil {
		t.FailNow()
	}

	// update OK
	fakeIS.updateError = nil
	if imageMeta, err := iModel.EditImage("", imageMeta); err != nil || !imageMeta {
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

	// searching for image failed
	fakeIS.imageEsistsError = errors.New("Serarching for image failed")
	fakeIS.imageExists = false
	if link, err := iModel.DownloadLink("iamge", time.Hour); err == nil || link != nil {
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
	link := images.NewLink("uri", time.Now())
	fakeFS.getReq = link

	receivedLink, err := iModel.DownloadLink("image", time.Hour)
	if err != nil || !reflect.DeepEqual(link, receivedLink) {
		t.FailNow()
	}
}

func makeFakeUpdate(t *testing.T, root string, valid bool) (string, error) {

	var dirStructOK = []atutils.TestDirEntry{
		{Path: "0000", IsDir: true},
		{Path: "0000/data", IsDir: true},
		{Path: "0000/data/update.ext4", Content: []byte("first update"), IsDir: false},
		{Path: "0000/type-info", Content: []byte(`{"type": "rootfs-image"}`), IsDir: false},
		{Path: "0000/meta-data", Content: []byte(`{"DeviceType": "vexpress-qemu", "ImageID": "core-image-minimal-201608110900"}`), IsDir: false},
		{Path: "0000/signatures", IsDir: true},
		{Path: "0000/signatures/update.sig", IsDir: false},
		{Path: "0000/scripts", IsDir: true},
		{Path: "0000/scripts/pre", IsDir: true},
		{Path: "0000/scripts/pre/0000_install.sh", Content: []byte("run me!"), IsDir: false},
		{Path: "0000/scripts/post", IsDir: true},
		{Path: "0000/scripts/check", IsDir: true},
	}

	err := atutils.MakeFakeUpdateDir(root, dirStructOK)
	assert.NoError(t, err)

	aw := awriter.NewWriter("mender", 1, []string{"vexpress"}, "mender-1.0")

	rp := &parser.RootfsParser{}
	aw.Register(rp)

	upath := path.Join(root, "update.tar")
	err = aw.Write(root, upath)
	assert.NoError(t, err)

	return upath, nil
}
