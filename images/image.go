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
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/satori/go.uuid"
)

type SoftwareImageConstructor struct {
	// Yocto ID build in the image
	YoctoId *string `json:"yocto_id" valid:"length(1|4096),required"`

	// Application Name & Version
	Name *string `json:"name" valid:"length(1|4096),required"`

	// Compatible device model for the application
	DeviceType *string `json:"device_type" valid:"length(1|4096),required"`

	// Image description
	Description *string `json:"description,omitempty" valid:"length(1|4096),optional"`

	// Image file checksum
	Checksum *string `json:"checksum,omitempty" valid:"optional"`
}

func NewSoftwareImageConstructor() *SoftwareImageConstructor {
	return &SoftwareImageConstructor{}
}

// Validate checkes structure according to valid tags.
func (s *SoftwareImageConstructor) Validate() error {
	_, err := govalidator.ValidateStruct(s)
	return err
}

// SoftwareImage YOCTO image with user application
type SoftwareImage struct {
	// User provided field set
	*SoftwareImageConstructor

	// Image ID
	Id *string `json:"id" bson:"_id" valid:"uuidv4,required"`

	// Status if image was verified after upload
	Verified bool `json:"verified" valid:"-"`

	// Last modification time, including image upload time
	Modified *time.Time `json:"modified" valid:"_"`
}

// NewSoftwareImageFromConstructor create new software image object.
func NewSoftwareImageFromConstructor(constructor *SoftwareImageConstructor) *SoftwareImage {

	now := time.Now()
	id := uuid.NewV4().String()

	return &SoftwareImage{
		SoftwareImageConstructor: constructor,
		Modified:                 &now,
		Verified:                 false,
		Id:                       &id,
	}
}

// SetModified set last modification time for the image.
func (s *SoftwareImage) SetModified(time time.Time) {
	s.Modified = &time
}

// Validate checkes structure according to valid tags.
func (s *SoftwareImage) Validate() error {
	_, err := govalidator.ValidateStruct(s)
	return err
}
