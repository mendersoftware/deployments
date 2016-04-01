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
	"time"

	"github.com/mendersoftware/artifacts/models/users"
)

var (
	// Missing required attibute
	ErrMissingImageAttrName    = errors.New("Required field missing: 'name'")
	ErrMissingImageAttrModel   = errors.New("Required field missing: 'model'")
	ErrMissingImageAttrYoctoId = errors.New("Required field missing: 'yocto_id'")
)

type ImagesModelI interface {
	Find(user users.UserI) ([]*ImageMeta, error)
	FindOne(user users.UserI, id string) (*ImageMeta, error)
	Exists(user users.UserI, id string) (bool, error)

	// ImageMeta.Name attribute is required to be unique
	Insert(user users.UserI, image *ImageMeta) (string, error)
	Update(user users.UserI, image *ImageMeta) error
	Delete(user users.UserI, id string) error
}

// Public - READ ONLY
type ImageMetaPrivate struct {

	// Unique field
	Id string `json:"id"`

	Verified    bool      `json:"verified"`
	LastUpdated time.Time `json:"modified"`
}

// Public - WRITTABLE (CREATE / EDIT)
type ImageMetaPublic struct {

	//Unique & required field
	Name string `json:"name"`

	// Optional
	Description string `json:"description"`
	Checksum    string `json:"checksum"`

	// Required
	Model   string `json:"model"`
	YoctoId string `json:"yocto_id"`
}

// NewImageMetaPublic create new struct
func NewImageMetaPublic(name, model, yoctoId string) *ImageMetaPublic {
	return &ImageMetaPublic{
		Name:    name,
		Model:   model,
		YoctoId: yoctoId,
	}
}

// Check if required fields are set.
// Can be improved with some reflection and tag magic ("required" tag)
func (i *ImageMetaPublic) Valid() error {

	if i.Name == "" {
		return ErrMissingImageAttrName
	}

	if i.Model == "" {
		return ErrMissingImageAttrModel
	}

	if i.YoctoId == "" {
		return ErrMissingImageAttrYoctoId
	}

	return nil
}

type ImageMeta struct {
	*ImageMetaPublic
	*ImageMetaPrivate
}

func NewImageMetaMerge(public *ImageMetaPublic, private *ImageMetaPrivate) *ImageMeta {
	img := &ImageMeta{
		ImageMetaPublic:  public,
		ImageMetaPrivate: private,
	}

	img.LastUpdated = time.Now()

	return img
}

func NewImageMetaFromPublic(public *ImageMetaPublic) *ImageMeta {

	return &ImageMeta{
		ImageMetaPublic: public,
		ImageMetaPrivate: &ImageMetaPrivate{
			LastUpdated: time.Now(),
		},
	}
}
