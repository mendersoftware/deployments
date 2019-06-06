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
	"context"
	"errors"

	"github.com/mendersoftware/deployments/model"
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
	Exists(ctx context.Context, id string) (bool, error)
	Update(ctx context.Context, image *model.SoftwareImage) (bool, error)
	Insert(ctx context.Context, image *model.SoftwareImage) error
	FindByID(ctx context.Context, id string) (*model.SoftwareImage, error)
	IsArtifactUnique(ctx context.Context, artifactName string,
		deviceTypesCompatible []string) (bool, error)
	Delete(ctx context.Context, id string) error
	FindAll(ctx context.Context) ([]*model.SoftwareImage, error)
}
