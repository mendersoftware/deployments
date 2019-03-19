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

package model

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/mendersoftware/mender-artifact/artifact"
	"github.com/mendersoftware/mender-artifact/awriter"
	"github.com/mendersoftware/mender-artifact/handlers"

	"github.com/stretchr/testify/assert"

	"github.com/mendersoftware/deployments/resources/images"
	"github.com/mendersoftware/deployments/resources/images/controller"
)

const (
	validUUIDv4  = "d50eda0d-2cea-4de1-8d42-9cd3e7e8670d"
	artifactSize = 10000
)

func TestCreateImageEmptyMessage(t *testing.T) {
	iModel := NewImagesModel(nil, nil, nil)
	if _, err := iModel.CreateImage(context.Background(),
		nil); err != controller.ErrModelMultipartUploadMsgMalformed {
		t.FailNow()
	}
}
func TestCreateImageEmptyMetaConstructor(t *testing.T) {
	iModel := NewImagesModel(nil, nil, nil)
	multipartUploadMessage := &controller.MultipartUploadMsg{}
	if _, err := iModel.CreateImage(context.Background(),
		multipartUploadMessage); err != controller.ErrModelMissingInputMetadata {
		t.FailNow()
	}
}

func TestCreateImageMissingFields(t *testing.T) {
	iModel := NewImagesModel(nil, nil, nil)
	multipartUploadMessage := &controller.MultipartUploadMsg{
		MetaConstructor: images.NewSoftwareImageMetaConstructor(),
	}

	if _, err := iModel.CreateImage(context.Background(),
		multipartUploadMessage); err == nil {
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

func (fis *FakeImageStorage) Exists(ctx context.Context, id string) (bool, error) {
	return fis.imageExists, fis.imageEsistsError
}

func (fis *FakeImageStorage) Update(ctx context.Context,
	image *images.SoftwareImage) (bool, error) {
	return fis.update, fis.updateError
}

func (fis *FakeImageStorage) Insert(ctx context.Context,
	image *images.SoftwareImage) error {
	return fis.insertError
}

func (fis *FakeImageStorage) FindByID(ctx context.Context,
	id string) (*images.SoftwareImage, error) {
	return fis.findByIdImage, fis.findByIdError
}

func (fis *FakeImageStorage) Delete(ctx context.Context, id string) error {
	return fis.deleteError
}

func (fis *FakeImageStorage) FindAll(ctx context.Context) ([]*images.SoftwareImage, error) {
	return fis.findAllImages, fis.findAllError
}

func (fis *FakeImageStorage) IsArtifactUnique(ctx context.Context,
	artifactName string, deviceTypesCompatible []string) (bool, error) {
	return fis.isArtifactUnique, fis.isArtifactUniqueError
}

func createValidImageMeta() *images.SoftwareImageMetaConstructor {
	return images.NewSoftwareImageMetaConstructor()
}

func createValidImageMetaArtifact() *images.SoftwareImageMetaArtifactConstructor {
	imageMetaArtifact := images.NewSoftwareImageMetaArtifactConstructor()
	required := "required"

	imageMetaArtifact.DeviceTypesCompatible = []string{"required"}
	imageMetaArtifact.Name = required
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
	multipartUploadMessage := &controller.MultipartUploadMsg{
		MetaConstructor: createValidImageMeta(),
	}

	if _, err := iModel.CreateImage(context.Background(),
		multipartUploadMessage); err == nil {

		t.FailNow()
	}
}

func TestCreateImageArtifactUploadError(t *testing.T) {
	fakeIS := new(FakeImageStorage)
	fakeIS.insertError = nil
	fakeFS := new(FakeFileStorage)
	fakeFS.uploadArtifactError = errors.New("Cannot upload artifact")

	iModel := NewImagesModel(fakeFS, nil, fakeIS)

	td, _ := ioutil.TempDir("", "mender-install-update-")
	defer os.RemoveAll(td)
	upd, err := MakeRootfsImageArtifact(1, false)
	assert.NoError(t, err)

	multipartUploadMessage := &controller.MultipartUploadMsg{
		MetaConstructor: createValidImageMeta(),
		ArtifactSize:    int64(upd.Len()),
		ArtifactReader:  upd,
	}
	if _, err := iModel.CreateImage(context.Background(),
		multipartUploadMessage); err == nil {
		t.FailNow()
	}
}

func TestCreateImageCreateOK(t *testing.T) {
	fakeIS := new(FakeImageStorage)
	fakeIS.insertError = nil
	fakeIS.isArtifactUnique = true
	fakeFS := new(FakeFileStorage)

	iModel := NewImagesModel(fakeFS, nil, fakeIS)

	td, _ := ioutil.TempDir("", "mender-install-update-")
	defer os.RemoveAll(td)
	upd, err := MakeRootfsImageArtifact(1, false)
	assert.NoError(t, err)

	multipartUploadMessage := &controller.MultipartUploadMsg{
		MetaConstructor: createValidImageMeta(),
		ArtifactSize:    int64(upd.Len()),
		ArtifactReader:  upd,
	}

	if _, err := iModel.CreateImage(context.Background(),
		multipartUploadMessage); err != nil {

		t.FailNow()
	}
}

func TestCreateSignedImageCreateOK(t *testing.T) {
	fakeIS := new(FakeImageStorage)
	fakeIS.insertError = nil
	fakeIS.isArtifactUnique = true
	fakeFS := new(FakeFileStorage)

	iModel := NewImagesModel(fakeFS, nil, fakeIS)

	td, _ := ioutil.TempDir("", "mender-install-update-")
	defer os.RemoveAll(td)
	upd, err := MakeRootfsImageArtifact(2, true)
	assert.NoError(t, err)

	multipartUploadMessage := &controller.MultipartUploadMsg{
		MetaConstructor: createValidImageMeta(),
		ArtifactSize:    int64(upd.Len()),
		ArtifactReader:  upd,
	}

	if _, err := iModel.CreateImage(context.Background(),
		multipartUploadMessage); err != nil {

		t.FailNow()
	}
}

func TestGetImageFindByIDError(t *testing.T) {
	fakeIS := new(FakeImageStorage)
	fakeIS.findByIdError = errors.New("find by id error")

	iModel := NewImagesModel(nil, nil, fakeIS)
	if _, err := iModel.GetImage(context.Background(), ""); err == nil {
		t.FailNow()
	}
}

func TestGetImageFindByIDEmptyImage(t *testing.T) {
	fakeIS := new(FakeImageStorage)
	fakeIS.findByIdImage = nil

	iModel := NewImagesModel(nil, nil, fakeIS)
	if image, err := iModel.GetImage(context.Background(),
		""); err != nil || image != nil {

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

func (ffs *FakeFileStorage) Delete(ctx context.Context, objectId string) error {
	return ffs.deleteError
}

func (ffs *FakeFileStorage) Exists(ctx context.Context, objectId string) (bool, error) {
	return ffs.imageExists, ffs.imageEsistsError
}

func (ffs *FakeFileStorage) LastModified(ctx context.Context,
	objectId string) (time.Time, error) {
	return ffs.lastModifiedTime, ffs.lastModifiedError
}

func (ffs *FakeFileStorage) PutRequest(ctx context.Context, objectId string,
	duration time.Duration) (*images.Link, error) {
	return ffs.putReq, ffs.putError
}

func (ffs *FakeFileStorage) GetRequest(ctx context.Context, objectId string,
	duration time.Duration, responseContentType string) (*images.Link, error) {
	return ffs.getReq, ffs.getError
}

func (fis *FakeFileStorage) UploadArtifact(ctx context.Context, id string,
	size int64, img io.Reader, contentType string) error {
	if _, err := io.Copy(ioutil.Discard, img); err != nil {
		return err
	}
	return fis.uploadArtifactError
}

func TestGetImageOK(t *testing.T) {
	imageMeta := createValidImageMeta()
	imageMetaArtifact := createValidImageMetaArtifact()
	constructorImage := images.NewSoftwareImage(
		validUUIDv4, imageMeta, imageMetaArtifact, artifactSize)
	now := time.Now()
	constructorImage.Modified = &now

	fakeIS := new(FakeImageStorage)
	fakeIS.findByIdImage = constructorImage
	fakeFS := new(FakeFileStorage)
	fakeFS.lastModifiedTime = time.Now()

	iModel := NewImagesModel(fakeFS, nil, fakeIS)
	if image, err := iModel.GetImage(context.Background(),
		""); err != nil || image == nil {

		t.FailNow()
	}
}

type FakeUseChecker struct {
	usedInActiveDeploymentsErr error
	isUsedInActiveDeployment   bool
	usedInDeploymentsErr       error
	isUsedInDeployment         bool
}

func (fus *FakeUseChecker) ImageUsedInActiveDeployment(ctx context.Context,
	imageId string) (bool, error) {

	return fus.isUsedInActiveDeployment, fus.usedInActiveDeploymentsErr
}

func (fus *FakeUseChecker) ImageUsedInDeployment(ctx context.Context, imageId string) (bool, error) {
	return fus.isUsedInDeployment, fus.usedInDeploymentsErr
}

func TestDeleteImage(t *testing.T) {
	imageMeta := createValidImageMeta()
	imageMetaArtifact := createValidImageMetaArtifact()
	constructorImage := images.NewSoftwareImage(
		validUUIDv4, imageMeta, imageMetaArtifact, artifactSize)

	fakeFS := new(FakeFileStorage)
	fakeChecker := new(FakeUseChecker)
	fakeIS := new(FakeImageStorage)

	fakeIS.findByIdImage = constructorImage

	fakeChecker.usedInActiveDeploymentsErr = errors.New("error")

	iModel := NewImagesModel(fakeFS, fakeChecker, fakeIS)

	if err := iModel.DeleteImage(context.Background(), ""); err == nil {
		t.FailNow()
	}

	fakeChecker.usedInActiveDeploymentsErr = nil
	fakeChecker.isUsedInActiveDeployment = true
	if err := iModel.DeleteImage(context.Background(),
		""); err != controller.ErrModelImageInActiveDeployment {
		t.FailNow()
	}

	// we should delete image successfully
	fakeChecker.isUsedInActiveDeployment = false
	if err := iModel.DeleteImage(context.Background(), ""); err != nil {
		t.FailNow()
	}

	fakeFS.deleteError = errors.New("error")
	if err := iModel.DeleteImage(context.Background(), ""); err == nil {
		t.FailNow()
	}

	fakeFS.deleteError = nil
	fakeIS.deleteError = errors.New("error")
	if err := iModel.DeleteImage(context.Background(), ""); err == nil {
		t.FailNow()
	}

	fakeIS.deleteError = errors.New("error")
	fakeIS.findByIdImage = nil

	if err := iModel.DeleteImage(context.Background(), ""); err == nil {
		t.FailNow()
	}

	fakeFS.getError = errors.New("error")
	fakeChecker.isUsedInActiveDeployment = false
	if err := iModel.DeleteImage(context.Background(), ""); err == nil {
		t.FailNow()
	}
}

func TestListImages(t *testing.T) {
	fakeChecker := new(FakeUseChecker)
	fakeFS := new(FakeFileStorage)
	fakeIS := new(FakeImageStorage)
	iModel := NewImagesModel(fakeFS, fakeChecker, fakeIS)

	fakeIS.findAllError = errors.New("error")
	if _, err := iModel.ListImages(context.Background(), nil); err == nil {
		t.FailNow()
	}

	//no error; empty images list
	fakeIS.findAllError = nil
	if _, err := iModel.ListImages(context.Background(), nil); err != nil {
		t.FailNow()
	}

	//have some valid image
	imageMeta := createValidImageMeta()
	imageMetaArtifact := createValidImageMetaArtifact()
	constructorImage := images.NewSoftwareImage(
		validUUIDv4, imageMeta, imageMetaArtifact, artifactSize)
	now := time.Now()
	constructorImage.Modified = &now

	listedImages := []*images.SoftwareImage{constructorImage}
	fakeIS.findAllImages = listedImages
	if _, err := iModel.ListImages(context.Background(), nil); err != nil {
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
	if _, err := iModel.EditImage(context.Background(),
		"", imageMeta); err == nil {
		t.FailNow()
	}

	// image used in deployments
	fakeChecker.usedInDeploymentsErr = nil
	fakeChecker.isUsedInDeployment = true
	if _, err := iModel.EditImage(context.Background(),
		"", imageMeta); err != controller.ErrModelImageUsedInAnyDeployment {
		t.FailNow()
	}

	// not used in deployments; finding error
	fakeChecker.isUsedInDeployment = false
	fakeIS.findByIdError = errors.New("error")
	if _, err := iModel.EditImage(context.Background(),
		"", imageMeta); err == nil {
		t.FailNow()
	}

	// not used in deployments; cannot find image
	fakeIS.findByIdError = nil
	fakeIS.findByIdImage = nil
	if imageMeta, err := iModel.EditImage(context.Background(),
		"", imageMeta); err != nil || imageMeta == true {
		t.FailNow()
	}

	// image does not exists
	constructorImage := images.NewSoftwareImage(
		validUUIDv4, imageMeta, imageMetaArtifact, artifactSize)
	fakeIS.findByIdImage = constructorImage
	fakeIS.updateError = errors.New("error")
	if _, err := iModel.EditImage(context.Background(),
		"", imageMeta); err == nil {
		t.FailNow()
	}

	// update OK
	fakeIS.updateError = nil
	if imageMeta, err := iModel.EditImage(context.Background(),
		"", imageMeta); err != nil || !imageMeta {
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
	if _, err := iModel.DownloadLink(context.Background(),
		"iamge", time.Hour); err == nil {
		t.FailNow()
	}

	// searching for image failed
	fakeIS.imageEsistsError = errors.New("Serarching for image failed")
	fakeIS.imageExists = false
	if link, err := iModel.DownloadLink(context.Background(),
		"iamge", time.Hour); err == nil || link != nil {
		t.FailNow()
	}

	// iamge does not esists
	fakeIS.imageEsistsError = nil
	fakeIS.imageExists = false
	if link, err := iModel.DownloadLink(context.Background(),
		"iamge", time.Hour); err != nil || link != nil {
		t.FailNow()
	}

	// can not generate link
	fakeIS.imageExists = true
	fakeFS.imageExists = true
	fakeFS.getError = errors.New("error")
	if _, err := iModel.DownloadLink(context.Background(),
		"iamge", time.Hour); err == nil {
		t.FailNow()
	}

	// upload link generation success
	fakeFS.getError = nil
	link := images.NewLink("uri", time.Now())
	fakeFS.getReq = link

	receivedLink, err := iModel.DownloadLink(context.Background(),
		"image", time.Hour)
	if err != nil || !reflect.DeepEqual(link, receivedLink) {
		t.FailNow()
	}
}

func MakeFakeUpdate(data string) (string, error) {
	f, err := ioutil.TempFile("", "test_update")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if len(data) > 0 {
		if _, err := f.WriteString(data); err != nil {
			return "", err
		}
	}
	return f.Name(), nil
}

const PrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQDSTLzZ9hQq3yBB+dMDVbKem6iav1J6opg6DICKkQ4M/yhlw32B
CGm2ArM3VwQRgq6Q1sNSq953n5c1EO3Xcy/qTAKcXwaUNml5EhW79AdibBXZiZt8
fMhCjUd/4ce3rLNjnbIn1o9L6pzV4CcVJ8+iNhne5vbA+63vRCnrc8QuYwIDAQAB
AoGAQKIRELQOsrZsxZowfj/ia9jPUvAmO0apnn2lK/E07k2lbtFMS1H4m1XtGr8F
oxQU7rLyyP/FmeJUqJyRXLwsJzma13OpxkQtZmRpL9jEwevnunHYJfceVapQOJ7/
6Oz0pPWEq39GCn+tTMtgSmkEaSH8Ki9t32g9KuQIKBB2hbECQQDsg7D5fHQB1BXG
HJm9JmYYX0Yk6Z2SWBr4mLO0C4hHBnV5qPCLyevInmaCV2cOjDZ5Sz6iF5RK5mw7
qzvFa8ePAkEA46Anom3cNXO5pjfDmn2CoqUvMeyrJUFL5aU6W1S6iFprZ/YwdHcC
kS5yTngwVOmcnT65Vnycygn+tZan2A0h7QJBAJNlowZovDdjgEpeCqXp51irD6Dz
gsLwa6agK+Y6Ba0V5mJyma7UoT//D62NYOmdElnXPepwvXdMUQmCtpZbjBsCQD5H
VHDJlCV/yzyiJz9+tZ5giaAkO9NOoUBsy6GvdfXWn2prXmiPI0GrrpSvp7Gj1Tjk
r3rtT0ysHWd7l+Kx/SUCQGlitd5RDfdHl+gKrCwhNnRG7FzRLv5YOQV81+kh7SkU
73TXPIqLESVrqWKDfLwfsfEpV248MSRou+y0O1mtFpo=
-----END RSA PRIVATE KEY-----
`

func MakeRootfsImageArtifact(version int, signed bool) (*bytes.Buffer, error) {
	upd, err := MakeFakeUpdate("test update")
	if err != nil {
		return nil, err
	}
	defer os.Remove(upd)

	art := bytes.NewBuffer(nil)
	var aw *awriter.Writer
	if !signed {
		aw = awriter.NewWriter(art, artifact.NewCompressorGzip())
	} else {
		s := artifact.NewSigner([]byte(PrivateKey))
		aw = awriter.NewWriterSigned(art, artifact.NewCompressorGzip(), s)
	}
	var u handlers.Composer
	switch version {
	case 1:
		u = handlers.NewRootfsV1(upd, artifact.NewCompressorGzip())
	case 2:
		u = handlers.NewRootfsV2(upd, artifact.NewCompressorGzip())
	}

	updates := &awriter.Updates{U: []handlers.Composer{u}}
	err = aw.WriteArtifact("mender", version, []string{"vexpress-qemu"},
		"mender-1.1", updates, nil)
	if err != nil {
		return nil, err
	}
	return art, nil
}
