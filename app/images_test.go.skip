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

package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/mendersoftware/mender-artifact/artifact"
	"github.com/mendersoftware/mender-artifact/awriter"
	"github.com/mendersoftware/mender-artifact/handlers"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/store/mocks"
)

const (
	validUUIDv4  = "d50eda0d-2cea-4de1-8d42-9cd3e7e8670d"
	artifactSize = 10000
)

func TestCreateImageEmptyMessage(t *testing.T) {
	iModel := NewImagesModel(nil, nil, nil)
	if _, err := iModel.CreateImage(context.Background(),
		nil); err != ErrModelMultipartUploadMsgMalformed {
		t.FailNow()
	}
}
func TestCreateImageEmptyMetaConstructor(t *testing.T) {
	iModel := NewImagesModel(nil, nil, nil)
	multipartUploadMessage := &model.MultipartUploadMsg{}
	if _, err := iModel.CreateImage(context.Background(),
		multipartUploadMessage); err != ErrModelMissingInputMetadata {
		t.FailNow()
	}
}

func TestCreateImageMissingFields(t *testing.T) {
	iModel := NewImagesModel(nil, nil, nil)
	multipartUploadMessage := &model.MultipartUploadMsg{
		MetaConstructor: model.NewSoftwareImageMetaConstructor(),
	}

	if _, err := iModel.CreateImage(context.Background(),
		multipartUploadMessage); err == nil {
		t.FailNow()
	}
}

func createValidImageMeta() *model.SoftwareImageMetaConstructor {
	return model.NewSoftwareImageMetaConstructor()
}

func createValidImageMetaArtifact() *model.SoftwareImageMetaArtifactConstructor {
	imageMetaArtifact := model.NewSoftwareImageMetaArtifactConstructor()
	required := "required"

	imageMetaArtifact.DeviceTypesCompatible = []string{"required"}
	imageMetaArtifact.Name = required
	imageMetaArtifact.Info = &model.ArtifactInfo{
		Format:  required,
		Version: 1,
	}
	return imageMetaArtifact
}

func createValidImageMetaDataArtifact() *model.SoftwareImageMetaArtifactConstructor {
	imageMetaArtifact := createValidImageMetaArtifact()
	metaData := map[string]interface{}{
		"foo":   "bar",
		"image": "alpine:sha123",
	}
	imageMetaArtifact.Updates = append(
		imageMetaArtifact.Updates,
		model.Update{
			MetaData: metaData,
		})
	return imageMetaArtifact
}

func TestCreateImageInsertError(t *testing.T) {
	fakeIS := mocks.DataStore{}

	iModel := NewImagesModel(nil, nil, &fakeIS)
	multipartUploadMessage := &model.MultipartUploadMsg{
		MetaConstructor: createValidImageMeta(),
	}
	fakeIS.On("InsertImage",
		mock.MatchedBy(
			func(_ context.Context) bool {
				return true
			}), mock.AnythingOfType("*model.SoftwareImage")).Return(errors.New("insert error"))

	if _, err := iModel.CreateImage(context.Background(),
		multipartUploadMessage); err == nil {

		t.FailNow()
	}
}

func TestCreateImageArtifactUploadError(t *testing.T) {
	fakeIS := mocks.DataStore{}
	fakeIS.On("InsertImage",
		mock.MatchedBy(
			func(_ context.Context) bool {
				return true
			}), mock.AnythingOfType("*model.SoftwareImage")).Return(nil)

	fakeFS := new(FakeFileStorage)
	fakeFS.uploadArtifactError = errors.New("Cannot upload artifact")

	iModel := NewImagesModel(fakeFS, nil, &fakeIS)

	td, _ := ioutil.TempDir("", "mender-install-update-")
	defer os.RemoveAll(td)
	upd, err := MakeRootfsImageArtifact(1, false)
	assert.NoError(t, err)

	multipartUploadMessage := &model.MultipartUploadMsg{
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
	fakeIS := mocks.DataStore{}
	fakeIS.On("InsertImage",
		mock.MatchedBy(
			func(_ context.Context) bool {
				return true
			}), mock.AnythingOfType("*model.SoftwareImage")).Return(nil)
	fakeIS.On("IsArtifactUnique",
		mock.MatchedBy(
			func(_ context.Context) bool {
				return true
			}), mock.AnythingOfType("string"),
		mock.AnythingOfType("[]string")).Return(true, nil)

	fakeFS := new(FakeFileStorage)

	iModel := NewImagesModel(fakeFS, nil, &fakeIS)

	td, _ := ioutil.TempDir("", "mender-install-update-")
	defer os.RemoveAll(td)
	upd, err := MakeRootfsImageArtifact(1, false)
	assert.NoError(t, err)

	multipartUploadMessage := &model.MultipartUploadMsg{
		MetaConstructor: createValidImageMeta(),
		ArtifactSize:    int64(upd.Len()),
		ArtifactReader:  upd,
	}

	if _, err := iModel.CreateImage(context.Background(),
		multipartUploadMessage); err != nil {

		t.FailNow()
	}
}

func TestCreateImageArtifactNotUnique(t *testing.T) {
	fakeIS := mocks.DataStore{}
	fakeIS.On("InsertImage",
		mock.MatchedBy(
			func(_ context.Context) bool {
				return true
			}), mock.AnythingOfType("*model.SoftwareImage")).Return(nil)
	fakeIS.On("IsArtifactUnique",
		mock.MatchedBy(
			func(_ context.Context) bool {
				return true
			}), mock.AnythingOfType("string"),
		mock.AnythingOfType("[]string")).Return(false, nil)

	fakeFS := new(FakeFileStorage)

	iModel := NewImagesModel(fakeFS, nil, &fakeIS)

	td, _ := ioutil.TempDir("", "mender-install-update-")
	defer os.RemoveAll(td)
	upd, err := MakeRootfsImageArtifact(1, false)
	assert.NoError(t, err)

	multipartUploadMessage := &model.MultipartUploadMsg{
		MetaConstructor: createValidImageMeta(),
		ArtifactSize:    int64(upd.Len()),
		ArtifactReader:  upd,
	}

	if _, err := iModel.CreateImage(context.Background(),
		multipartUploadMessage); err != ErrModelArtifactNotUnique {

		t.FailNow()
	}
}

func TestCreateImageArtifactNotUniqueCleanupError(t *testing.T) {
	fakeIS := mocks.DataStore{}
	fakeIS.On("InsertImage",
		mock.MatchedBy(
			func(_ context.Context) bool {
				return true
			}), mock.AnythingOfType("*model.SoftwareImage")).Return(nil)
	fakeIS.On("IsArtifactUnique",
		mock.MatchedBy(
			func(_ context.Context) bool {
				return true
			}), mock.AnythingOfType("string"),
		mock.AnythingOfType("[]string")).Return(false, nil)

	fakeFS := new(FakeFileStorage)
	deleteErr := errors.New("expected error")
	fakeFS.deleteError = deleteErr

	iModel := NewImagesModel(fakeFS, nil, &fakeIS)

	td, _ := ioutil.TempDir("", "mender-install-update-")
	defer os.RemoveAll(td)
	upd, err := MakeRootfsImageArtifact(1, false)
	assert.NoError(t, err)

	multipartUploadMessage := &model.MultipartUploadMsg{
		MetaConstructor: createValidImageMeta(),
		ArtifactSize:    int64(upd.Len()),
		ArtifactReader:  upd,
	}

	_, err = iModel.CreateImage(context.Background(), multipartUploadMessage)
	cause := errors.Cause(err)
	expectedErr := errors.Wrap(ErrModelArtifactNotUnique, deleteErr.Error())
	if cause != ErrModelArtifactNotUnique || err.Error() != expectedErr.Error() {
		t.FailNow()
	}
}

func TestCreateSignedImageCreateOK(t *testing.T) {
	fakeIS := mocks.DataStore{}
	fakeIS.On("InsertImage",
		mock.MatchedBy(
			func(_ context.Context) bool {
				return true
			}), mock.AnythingOfType("*model.SoftwareImage")).Return(nil)
	fakeIS.On("IsArtifactUnique",
		mock.MatchedBy(
			func(_ context.Context) bool {
				return true
			}), mock.AnythingOfType("string"),
		mock.AnythingOfType("[]string")).Return(true, nil)

	fakeFS := new(FakeFileStorage)

	iModel := NewImagesModel(fakeFS, nil, &fakeIS)

	td, _ := ioutil.TempDir("", "mender-install-update-")
	defer os.RemoveAll(td)
	upd, err := MakeRootfsImageArtifact(2, true)
	assert.NoError(t, err)

	multipartUploadMessage := &model.MultipartUploadMsg{
		MetaConstructor: createValidImageMeta(),
		ArtifactSize:    int64(upd.Len()),
		ArtifactReader:  upd,
	}

	if _, err := iModel.CreateImage(context.Background(),
		multipartUploadMessage); err != nil {

		t.FailNow()
	}
}

func TestCreateImageMetaDataOK(t *testing.T) {
	imageMeta := createValidImageMeta()
	imageMetaArtifact := createValidImageMetaDataArtifact()
	constructorImage := model.NewSoftwareImage(validUUIDv4, imageMeta, imageMetaArtifact, artifactSize)
	now := time.Now()
	constructorImage.Modified = &now

	fakeIS := mocks.DataStore{}
	fakeIS.On("FindImageByID",
		mock.MatchedBy(
			func(_ context.Context) bool {
				return true
			}), mock.AnythingOfType("string")).Return(constructorImage, nil)

	fakeFS := new(FakeFileStorage)
	fakeFS.lastModifiedTime = time.Now()

	iModel := NewImagesModel(fakeFS, nil, &fakeIS)
	image, err := iModel.GetImage(context.Background(), "")
	if err != nil || image == nil {
		t.FailNow()
	}
	if image.Updates == nil || image.Updates[0].MetaData == nil {
		t.FailNow()
	}
	assert.Equal(t, image.Updates[0].MetaData.(map[string]interface{})["foo"], "bar")
}

func TestGetImageFindByIDError(t *testing.T) {
	fakeIS := mocks.DataStore{}
	fakeIS.On("FindImageByID",
		mock.MatchedBy(
			func(_ context.Context) bool {
				return true
			}), mock.AnythingOfType("string")).Return(nil, errors.New("find by id error"))

	iModel := NewImagesModel(nil, nil, &fakeIS)
	if _, err := iModel.GetImage(context.Background(), ""); err == nil {
		t.FailNow()
	}
}

func TestGetImageFindByIDEmptyImage(t *testing.T) {
	fakeIS := mocks.DataStore{}
	fakeIS.On("FindImageByID",
		mock.MatchedBy(
			func(_ context.Context) bool {
				return true
			}), mock.AnythingOfType("string")).Return(nil, nil)

	iModel := NewImagesModel(nil, nil, &fakeIS)
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
	putReq              *model.Link
	putError            error
	getReq              *model.Link
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
	duration time.Duration) (*model.Link, error) {
	return ffs.putReq, ffs.putError
}

func (ffs *FakeFileStorage) GetRequest(ctx context.Context, objectId string,
	duration time.Duration, responseContentType string) (*model.Link, error) {
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
	constructorImage := model.NewSoftwareImage(
		validUUIDv4, imageMeta, imageMetaArtifact, artifactSize)
	now := time.Now()
	constructorImage.Modified = &now

	fakeIS := mocks.DataStore{}
	fakeIS.On("FindImageByID",
		mock.MatchedBy(
			func(_ context.Context) bool {
				return true
			}), mock.AnythingOfType("string")).Return(constructorImage, nil)

	fakeFS := new(FakeFileStorage)
	fakeFS.lastModifiedTime = time.Now()

	iModel := NewImagesModel(fakeFS, nil, &fakeIS)
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
	testCases := []struct {
		name string

		deleteImageError           error
		findImageError             error
		usedInActiveDeploymentsErr error
		constructorImage           *model.SoftwareImage
		isUsedInActiveDeployment   bool

		expectedErr error
	}{
		{
			name:             "ok",
			constructorImage: model.NewSoftwareImage(validUUIDv4, createValidImageMeta(), createValidImageMetaArtifact(), artifactSize),
		},
		{
			name:        "not found",
			expectedErr: errors.New("Image metadata is not found"),
		},
		{
			name:             "delete error",
			deleteImageError: errors.New("delete error"),
			constructorImage: model.NewSoftwareImage(validUUIDv4, createValidImageMeta(), createValidImageMetaArtifact(), artifactSize),
			expectedErr:      errors.New("Deleting image metadata: delete error"),
		},
		{
			name:           "find image error",
			findImageError: errors.New("find image error"),
			expectedErr:    errors.New("Getting image metadata: Searching for image with specified ID: find image error"),
		},
		{
			name: "deployment in use",
			isUsedInActiveDeployment: true,
			constructorImage:         model.NewSoftwareImage(validUUIDv4, createValidImageMeta(), createValidImageMetaArtifact(), artifactSize),
			expectedErr:              ErrModelImageInActiveDeployment,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			fakeFS := new(FakeFileStorage)
			fakeChecker := new(FakeUseChecker)
			fakeChecker.usedInActiveDeploymentsErr = tc.usedInActiveDeploymentsErr
			fakeChecker.isUsedInActiveDeployment = tc.isUsedInActiveDeployment

			fakeIS := mocks.DataStore{}
			fakeIS.On("FindImageByID",
				mock.MatchedBy(
					func(_ context.Context) bool {
						return true
					}), mock.AnythingOfType("string")).Return(tc.constructorImage, tc.findImageError)
			fakeIS.On("DeleteImage",
				mock.MatchedBy(
					func(_ context.Context) bool {
						return true
					}), mock.AnythingOfType("string")).Return(tc.deleteImageError)

			iModel := NewImagesModel(fakeFS, fakeChecker, &fakeIS)
			err := iModel.DeleteImage(context.Background(), "")
			if tc.expectedErr != nil {
				assert.EqualError(t, err, tc.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestListImages(t *testing.T) {
	t.Skip("test refactoring needed - MEN-2607")
	var findAllError error
	var images []*model.SoftwareImage
	fakeChecker := new(FakeUseChecker)
	fakeFS := new(FakeFileStorage)

	fakeIS := mocks.DataStore{}
	fakeIS.On("FindAll",
		mock.MatchedBy(
			func(_ context.Context) bool {
				return true
			})).Return(images, findAllError)

	iModel := NewImagesModel(fakeFS, fakeChecker, &fakeIS)

	findAllError = errors.New("error")
	if _, err := iModel.ListImages(context.Background(), nil); err == nil {
		t.FailNow()
	}

	//no error; empty images list
	findAllError = nil
	if _, err := iModel.ListImages(context.Background(), nil); err != nil {
		t.FailNow()
	}

	//have some valid image
	imageMeta := createValidImageMeta()
	imageMetaArtifact := createValidImageMetaArtifact()
	constructorImage := model.NewSoftwareImage(
		validUUIDv4, imageMeta, imageMetaArtifact, artifactSize)
	now := time.Now()
	constructorImage.Modified = &now

	listedImages := []*model.SoftwareImage{constructorImage}
	images = listedImages
	if _, err := iModel.ListImages(context.Background(), nil); err != nil {
		t.FailNow()
	}
}

func TestEditImage(t *testing.T) {
	t.Skip("test refactoring needed - MEN-2607")
	var constructorImage *model.SoftwareImage
	var findImageByIdError error
	var updateError error

	imageMeta := createValidImageMeta()
	imageMetaArtifact := createValidImageMetaArtifact()

	fakeChecker := new(FakeUseChecker)
	fakeIS := mocks.DataStore{}
	fakeIS.On("FindImageByID",
		mock.MatchedBy(
			func(_ context.Context) bool {
				return true
			}), mock.AnythingOfType("string")).Return(constructorImage, findImageByIdError)
	fakeIS.On("Update",
		mock.MatchedBy(
			func(_ context.Context) bool {
				return true
			}), mock.AnythingOfType("*model.SoftwareImage")).Return(true, updateError)

	iModel := NewImagesModel(nil, fakeChecker, &fakeIS)

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
		"", imageMeta); err != ErrModelImageUsedInAnyDeployment {
		t.FailNow()
	}

	// not used in deployments; finding error
	fakeChecker.isUsedInDeployment = false
	findImageByIdError = errors.New("error")
	if _, err := iModel.EditImage(context.Background(),
		"", imageMeta); err == nil {
		t.FailNow()
	}

	// not used in deployments; cannot find image
	findImageByIdError = nil
	constructorImage = nil
	if imageMeta, err := iModel.EditImage(context.Background(),
		"", imageMeta); err != nil || imageMeta == true {
		t.FailNow()
	}

	// image does not exists
	constructorImage = model.NewSoftwareImage(
		validUUIDv4, imageMeta, imageMetaArtifact, artifactSize)
	updateError = errors.New("error")
	if _, err := iModel.EditImage(context.Background(),
		"", imageMeta); err == nil {
		t.FailNow()
	}

	// update OK
	updateError = nil
	if imageMeta, err := iModel.EditImage(context.Background(),
		"", imageMeta); err != nil || !imageMeta {
		t.FailNow()
	}
}

func TestDownloadLink(t *testing.T) {
	t.Skip("test refactoring needed - MEN-2607")
	var imageExists bool
	var imageExistsError error
	fakeIS := mocks.DataStore{}
	fakeIS.On("Exists",
		mock.MatchedBy(
			func(_ context.Context) bool {
				return true
			}), mock.AnythingOfType("string")).Return(imageExists, imageExistsError)

	fakeChecker := new(FakeUseChecker)
	fakeFS := new(FakeFileStorage)
	iModel := NewImagesModel(fakeFS, fakeChecker, &fakeIS)

	// image exists error
	imageExistsError = errors.New("error")
	if _, err := iModel.DownloadLink(context.Background(),
		"iamge", time.Hour); err == nil {
		t.FailNow()
	}

	// searching for image failed
	imageExistsError = errors.New("Serarching for image failed")
	imageExists = false
	if link, err := iModel.DownloadLink(context.Background(),
		"iamge", time.Hour); err == nil || link != nil {
		t.FailNow()
	}

	// iamge does not esists
	imageExistsError = nil
	imageExists = false
	if link, err := iModel.DownloadLink(context.Background(),
		"iamge", time.Hour); err != nil || link != nil {
		t.FailNow()
	}

	// can not generate link
	imageExists = true
	fakeFS.imageExists = true
	fakeFS.getError = errors.New("error")
	if _, err := iModel.DownloadLink(context.Background(),
		"iamge", time.Hour); err == nil {
		t.FailNow()
	}

	// upload link generation success
	fakeFS.getError = nil
	link := model.NewLink("uri", time.Now())
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
	comp := artifact.NewCompressorGzip()
	var aw *awriter.Writer
	if !signed {
		aw = awriter.NewWriter(art, comp)
	} else {
		s := artifact.NewSigner([]byte(PrivateKey))
		aw = awriter.NewWriterSigned(art, comp, s)
	}
	var u handlers.Composer
	switch version {
	case 1:
		u = handlers.NewRootfsV1(upd)
	case 2:
		u = handlers.NewRootfsV2(upd)
	case 3:
		u = handlers.NewRootfsV3(upd)
	}

	updates := &awriter.Updates{Updates: []handlers.Composer{u}}
	artifactArgs := &awriter.WriteArtifactArgs{Format: "mender", Version: version,
		Devices: []string{"vexpress-qemu"}, Name: "mender-1.1", Updates: updates}
	err = aw.WriteArtifact(artifactArgs)
	if err != nil {
		return nil, err
	}
	return art, nil
}
