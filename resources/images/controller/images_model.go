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

package controller

import (
	"errors"
	"os"
	"time"

	"github.com/mendersoftware/deployments/resources/images"
)

// Errors expected from interface
var (
	ErrImageMetaNotFound = errors.New("Image metadata is not found")
)

type ImagesModel interface {
	ListImages(filters map[string]string) ([]*images.SoftwareImage, error)
	DownloadLink(imageID string, expire time.Duration) (*images.Link, error)
	GetImage(id string) (*images.SoftwareImage, error)
	DeleteImage(imageID string) error
	CreateImage(
		imageFile *os.File,
		metaConstructor *images.SoftwareImageMetaConstructor,
		metaYoctoConstructor *images.SoftwareImageMetaYoctoConstructor) (string, error)
	EditImage(id string, constructorData *images.SoftwareImageMetaConstructor) (bool, error)
}
