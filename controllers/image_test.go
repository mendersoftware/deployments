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
	"reflect"
	"testing"
	"time"

	"github.com/mendersoftware/artifacts/models/fileservice"
	"github.com/mendersoftware/artifacts/models/images"
	"github.com/mendersoftware/artifacts/models/users"
)

type MockUser struct{}

func (user *MockUser) GetUserID() string     { return "user" }
func (user *MockUser) GetCustomerID() string { return "customer" }

type MockImagesModel struct {
	mockFind    func(user users.UserI) ([]*images.ImageMeta, error)
	mockFindOne func(user users.UserI, id string) (*images.ImageMeta, error)
	mockExists  func(user users.UserI, id string) (bool, error)
	mockInsert  func(user users.UserI, image *images.ImageMeta) (string, error)
	mockUpdate  func(user users.UserI, image *images.ImageMeta) error
	mockDelete  func(user users.UserI, id string) error
}

func (model *MockImagesModel) Find(user users.UserI) ([]*images.ImageMeta, error) {
	return model.mockFind(user)
}
func (model *MockImagesModel) FindOne(user users.UserI, id string) (*images.ImageMeta, error) {
	return model.mockFindOne(user, id)
}
func (model *MockImagesModel) Exists(user users.UserI, id string) (bool, error) {
	return model.mockExists(user, id)
}
func (model *MockImagesModel) Insert(user users.UserI, image *images.ImageMeta) (string, error) {
	return model.mockInsert(user, image)
}
func (model *MockImagesModel) Update(user users.UserI, image *images.ImageMeta) error {
	return model.mockUpdate(user, image)
}
func (model *MockImagesModel) Delete(user users.UserI, id string) error {
	return model.mockDelete(user, id)
}

type MockFileService struct {
	mockDelete       func(customerId, objectId string) error
	mockExists       func(customerId, objectId string) (bool, error)
	mockLastModified func(customerId, objectId string) (time.Time, error)
	mockPutRequest   func(customerId, objectId string, duration time.Duration) (*fileservice.Link, error)
	mockGetRequest   func(customerId, objectId string, duration time.Duration) (*fileservice.Link, error)
}

func (service *MockFileService) Delete(customerId, objectId string) error {
	return service.mockDelete(customerId, objectId)
}
func (service *MockFileService) Exists(customerId, objectId string) (bool, error) {
	return service.mockExists(customerId, objectId)
}
func (service *MockFileService) LastModified(customerId, objectId string) (time.Time, error) {
	return service.mockLastModified(customerId, objectId)
}
func (service *MockFileService) PutRequest(customerId, objectId string, duration time.Duration) (*fileservice.Link, error) {
	return service.mockPutRequest(customerId, objectId, duration)
}
func (service *MockFileService) GetRequest(customerId, objectId string, duration time.Duration) (*fileservice.Link, error) {
	return service.mockGetRequest(customerId, objectId, duration)
}

func TestImagesControlerGet(t *testing.T) {

	testList := []struct {
		expectedImage *images.ImageMeta
		expectedError error

		mockModelFindOneImage *images.ImageMeta
		mockModelFindOneError error
		mockModelUpdateError  error

		mockFileServiceLastModificationTime  time.Time
		mockFileServiceLastModificationError error
	}{
		{
			expectedImage: nil,
			expectedError: errors.New("Internal Issue"),

			mockModelFindOneError: errors.New("Internal Issue"),
		},
		{
			expectedImage: nil,
			expectedError: errors.New("Internal Issue"),

			mockModelFindOneImage:                &images.ImageMeta{ImageMetaPrivate: &images.ImageMetaPrivate{}},
			mockModelFindOneError:                nil,
			mockFileServiceLastModificationError: errors.New("Internal Issue"),
		},
		{
			expectedImage: &images.ImageMeta{ImageMetaPrivate: &images.ImageMetaPrivate{}},
			expectedError: nil,

			mockModelFindOneImage:                &images.ImageMeta{ImageMetaPrivate: &images.ImageMetaPrivate{}},
			mockModelFindOneError:                nil,
			mockFileServiceLastModificationError: fileservice.ErrNotFound,
		},
		{
			expectedImage: nil,
			expectedError: errors.New("Internal Issue"),

			mockModelFindOneImage:                &images.ImageMeta{ImageMetaPrivate: &images.ImageMetaPrivate{}},
			mockModelFindOneError:                nil,
			mockFileServiceLastModificationError: nil,
			mockFileServiceLastModificationTime:  time.Now(),
			mockModelUpdateError:                 errors.New("Internal Issue"),
		},
		{
			expectedImage: &images.ImageMeta{
				ImageMetaPrivate: &images.ImageMetaPrivate{
					LastUpdated: time.Unix(200, 0),
				},
			},
			expectedError: nil,

			mockModelFindOneImage: &images.ImageMeta{
				ImageMetaPrivate: &images.ImageMetaPrivate{
					LastUpdated: time.Unix(200, 0),
				},
			},
			mockModelFindOneError:                nil,
			mockFileServiceLastModificationError: nil,
			mockFileServiceLastModificationTime:  time.Unix(100, 0),
			mockModelUpdateError:                 nil,
		},
		{
			expectedImage: &images.ImageMeta{
				ImageMetaPrivate: &images.ImageMetaPrivate{
					LastUpdated: time.Unix(100, 0),
				},
			},
			expectedError: nil,

			mockModelFindOneImage: &images.ImageMeta{
				ImageMetaPrivate: &images.ImageMetaPrivate{
					LastUpdated: time.Unix(50, 0),
				},
			},
			mockModelFindOneError:                nil,
			mockFileServiceLastModificationError: nil,
			mockFileServiceLastModificationTime:  time.Unix(100, 0),
			mockModelUpdateError:                 nil,
		},
	}

	for _, test := range testList {

		model := &MockImagesModel{
			mockFindOne: func(user users.UserI, id string) (*images.ImageMeta, error) {
				return test.mockModelFindOneImage, test.mockModelFindOneError
			},
			mockUpdate: func(user users.UserI, image *images.ImageMeta) error {
				return test.mockModelUpdateError
			},
		}

		fileService := &MockFileService{
			mockLastModified: func(customerId, objectId string) (time.Time, error) {
				return test.mockFileServiceLastModificationTime, test.mockFileServiceLastModificationError
			},
		}

		controller := NewImagesController(model, fileService)
		image, err := controller.Get(&MockUser{}, "id_123")

		if test.expectedError == nil || err == nil {
			if err != test.expectedError {
				t.FailNow()
			}
		} else if test.expectedError.Error() != err.Error() {
			t.FailNow()
		}

		if !reflect.DeepEqual(image, test.expectedImage) {
			t.FailNow()
		}
	}
}

func TestImagesControlerLookup(t *testing.T) {

	testList := []struct {
		expectedImages []*images.ImageMeta
		expectedError  error

		mockModelFindImages  []*images.ImageMeta
		mockModelFindError   error
		mockModelUpdateError error

		mockFileServiceLastModificationTime  time.Time
		mockFileServiceLastModificationError error
	}{
		{
			expectedImages: nil,
			expectedError:  errors.New("Internal Issue"),

			mockModelFindError: errors.New("Internal Issue"),
		},
		{
			expectedImages: nil,
			expectedError:  errors.New("Internal Issue"),

			mockModelFindImages: []*images.ImageMeta{
				&images.ImageMeta{
					ImageMetaPrivate: &images.ImageMetaPrivate{},
				},
			},
			mockModelFindError:                   nil,
			mockFileServiceLastModificationError: errors.New("Internal Issue"),
		},
		{
			expectedImages: []*images.ImageMeta{
				&images.ImageMeta{
					ImageMetaPrivate: &images.ImageMetaPrivate{},
				},
			},
			expectedError: nil,

			mockModelFindImages: []*images.ImageMeta{
				&images.ImageMeta{
					ImageMetaPrivate: &images.ImageMetaPrivate{},
				},
			},
			mockModelFindError:                   nil,
			mockFileServiceLastModificationError: fileservice.ErrNotFound,
		},
		{
			expectedImages: nil,
			expectedError:  errors.New("Internal Issue"),

			mockModelFindImages: []*images.ImageMeta{
				&images.ImageMeta{
					ImageMetaPrivate: &images.ImageMetaPrivate{},
				},
			},
			mockModelFindError:                   nil,
			mockFileServiceLastModificationError: nil,
			mockFileServiceLastModificationTime:  time.Now(),
			mockModelUpdateError:                 errors.New("Internal Issue"),
		},
		{
			expectedImages: []*images.ImageMeta{
				&images.ImageMeta{
					ImageMetaPrivate: &images.ImageMetaPrivate{
						LastUpdated: time.Unix(200, 0),
					},
				},
			},
			expectedError: nil,

			mockModelFindImages: []*images.ImageMeta{
				&images.ImageMeta{
					ImageMetaPrivate: &images.ImageMetaPrivate{
						LastUpdated: time.Unix(200, 0),
					},
				},
			},
			mockModelFindError:                   nil,
			mockFileServiceLastModificationError: nil,
			mockFileServiceLastModificationTime:  time.Unix(100, 0),
			mockModelUpdateError:                 nil,
		},
		{
			expectedImages: []*images.ImageMeta{
				&images.ImageMeta{
					ImageMetaPrivate: &images.ImageMetaPrivate{
						LastUpdated: time.Unix(200, 0),
					},
				},
			},
			expectedError: nil,

			mockModelFindImages: []*images.ImageMeta{
				&images.ImageMeta{
					ImageMetaPrivate: &images.ImageMetaPrivate{
						LastUpdated: time.Unix(100, 0),
					},
				},
			},
			mockModelFindError:                   nil,
			mockFileServiceLastModificationError: nil,
			mockFileServiceLastModificationTime:  time.Unix(200, 0),
			mockModelUpdateError:                 nil,
		},
	}

	for _, test := range testList {

		model := &MockImagesModel{
			mockFind: func(user users.UserI) ([]*images.ImageMeta, error) {
				return test.mockModelFindImages, test.mockModelFindError
			},
			mockUpdate: func(user users.UserI, image *images.ImageMeta) error {
				return test.mockModelUpdateError
			},
		}

		fileService := &MockFileService{
			mockLastModified: func(customerId, objectId string) (time.Time, error) {
				return test.mockFileServiceLastModificationTime, test.mockFileServiceLastModificationError
			},
		}

		controller := NewImagesController(model, fileService)
		images, err := controller.Lookup(&MockUser{})

		if test.expectedError == nil || err == nil {
			if err != test.expectedError {
				t.FailNow()
			}
		} else if test.expectedError.Error() != err.Error() {
			t.FailNow()
		}

		if len(images) != len(test.expectedImages) {
			t.FailNow()
		}

		if !reflect.DeepEqual(images, test.expectedImages) {
			t.FailNow()
		}
	}
}

func TestImagesControlerCreate(t *testing.T) {

	const ID string = "1234-12-1234"

	testList := []struct {
		expectedImage *images.ImageMeta
		expectedError error

		inImageMeta          *images.ImageMetaPublic
		mockModelInsertError error
	}{
		{
			expectedImage: &images.ImageMeta{
				ImageMetaPrivate: &images.ImageMetaPrivate{
					Id:          ID,
					LastUpdated: time.Unix(123, 0),
				},
				ImageMetaPublic: &images.ImageMetaPublic{
					Name: "MyName",
				},
			},
			expectedError: nil,
			inImageMeta: &images.ImageMetaPublic{
				Name: "MyName",
			},
			mockModelInsertError: nil,
		},
		{
			expectedImage:        nil,
			expectedError:        errors.New("Internal issue"),
			inImageMeta:          nil,
			mockModelInsertError: errors.New("Internal issue"),
		},
	}

	for _, test := range testList {

		model := &MockImagesModel{
			mockInsert: func(user users.UserI, image *images.ImageMeta) (string, error) {
				return ID, test.mockModelInsertError
			},
		}

		controller := NewImagesController(model, nil)
		image, err := controller.Create(&MockUser{}, test.inImageMeta)

		if test.expectedError == nil || err == nil {
			if err != test.expectedError {
				t.FailNow()
			}
		} else if test.expectedError.Error() != err.Error() {
			t.FailNow()
		}

		if test.expectedImage == nil || image == nil {
			if image != test.expectedImage {
				t.FailNow()
			}

			return
		}

		image.LastUpdated = test.expectedImage.LastUpdated

		if !reflect.DeepEqual(test.expectedImage, image) {
			t.FailNow()
		}
	}
}
