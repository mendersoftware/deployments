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

	"github.com/mendersoftware/deployments/resources/images"
)

// Common errors for interface SoftwareImagesStorage
var (
	ErrSoftwareImagesStorageInvalidID           = errors.New("Invalid id")
	ErrSoftwareImagesStorageInvalidArtifactName = errors.New("Invalid artifact name")
	ErrSoftwareImagesStorageInvalidName         = errors.New("Invalid name")
	ErrSoftwareImagesStorageInvalidDeviceType   = errors.New("Invalid device type")
	ErrSoftwareImagesStorageInvalidImage        = errors.New("Invalid image")
)

// SoftwareImagesStorage allow to store and manage image.SoftwareImages
type SoftwareImagesStorage interface {
	Exists(id string) (bool, error)
	Update(image *images.SoftwareImage) (bool, error)
	Insert(image *images.SoftwareImage) error
	FindByID(id string) (*images.SoftwareImage, error)
	IsArtifactUnique(artifactName string, deviceTypesCompatible []string) (bool, error)
	Delete(id string) error
	FindAll() ([]*images.SoftwareImage, error)
}
