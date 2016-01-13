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
package handlers

import (
	"time"

	"github.com/mendersoftware/artifacts/models/fileservice"
	"github.com/mendersoftware/artifacts/models/images"
	"github.com/mendersoftware/artifacts/models/users"
)

// Mock of ImagesControllerI interafce
// Uses dependency injection to mock method functionality.
type ImageControllerMock struct {
	mockLookup       func(user users.UserI) ([]*images.ImageMeta, error)
	mockGet          func(user users.UserI, id string) (*images.ImageMeta, error)
	mockCreate       func(user users.UserI, public *images.ImageMetaPublic) (*images.ImageMeta, error)
	mockEdit         func(user users.UserI, id string, public *images.ImageMetaPublic) error
	mockDelete       func(user users.UserI, id string) error
	mockUploadLink   func(user users.UserI, id string, expire time.Duration) (*fileservice.Link, error)
	mockDownloadLink func(user users.UserI, id string, expire time.Duration) (*fileservice.Link, error)
}

func (i *ImageControllerMock) Lookup(user users.UserI) ([]*images.ImageMeta, error) {
	return i.mockLookup(user)
}

func (i *ImageControllerMock) Get(user users.UserI, id string) (*images.ImageMeta, error) {
	return i.mockGet(user, id)
}

func (i *ImageControllerMock) Create(user users.UserI, public *images.ImageMetaPublic) (*images.ImageMeta, error) {
	return i.mockCreate(user, public)
}

func (i *ImageControllerMock) Edit(user users.UserI, id string, public *images.ImageMetaPublic) error {
	return i.mockEdit(user, id, public)
}

func (i *ImageControllerMock) Delete(user users.UserI, id string) error {
	return i.mockDelete(user, id)
}

func (i *ImageControllerMock) UploadLink(user users.UserI, id string, expire time.Duration) (*fileservice.Link, error) {
	return i.mockUploadLink(user, id, expire)
}

func (i *ImageControllerMock) DownloadLink(user users.UserI, id string, expire time.Duration) (*fileservice.Link, error) {
	return i.mockDownloadLink(user, id, expire)
}
