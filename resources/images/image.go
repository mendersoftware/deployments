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

package images

import (
	"time"

	"github.com/asaskevich/govalidator"
)

// Informations provided by the user
type SoftwareImageMetaConstructor struct {
	// Image description
	Description string `json:"description,omitempty" valid:"length(1|4096),optional"`
}

// Creates new, empty SoftwareImageMetaConstructor
func NewSoftwareImageMetaConstructor() *SoftwareImageMetaConstructor {
	return &SoftwareImageMetaConstructor{}
}

// Validate checkes structure according to valid tags.
func (s *SoftwareImageMetaConstructor) Validate() error {
	_, err := govalidator.ValidateStruct(s)
	return err
}

// Structure with artifact version informations
type ArtifactInfo struct {
	// Mender artifact format - the only possible value is "mender"
	//Format string `json:"format" valid:"string,equal("mender"),required"`
	Format string `json:"format" valid:"required"`

	// Mender artifact format version
	//Version uint `json:"version" valid:"uint,equal(1),required"`
	Version uint `json:"version" valid:"required"`
}

// Information provided with YOCTO image
type SoftwareImageMetaArtifactConstructor struct {
	// artifact_name from artifact file
	Name string `json:"name" bson:"name" valid:"length(1|4096),required"`

	// Compatible device types for the application
	DeviceTypesCompatible []string `json:"device_types_compatible" bson:"device_types_compatible" valid:"length(1|4096),required"`

	// Artifact version info
	Info *ArtifactInfo `json:"info"`

	// Flag that indicates if artifact is signed or not
	Signed bool `json:"signed" bson:"signed"`

	// List of updates
	Updates []Update `json:"updates" valid:"-"`
}

func NewSoftwareImageMetaArtifactConstructor() *SoftwareImageMetaArtifactConstructor {
	return &SoftwareImageMetaArtifactConstructor{}
}

// Validate checkes structure according to valid tags.
func (s *SoftwareImageMetaArtifactConstructor) Validate() error {
	_, err := govalidator.ValidateStruct(s)
	return err
}

// SoftwareImage YOCTO image with user application
type SoftwareImage struct {
	// User provided field set
	SoftwareImageMetaConstructor `bson:"meta"`

	// Field set provided with yocto image
	SoftwareImageMetaArtifactConstructor `bson:"meta_artifact"`

	// Image ID
	Id string `json:"id" bson:"_id" valid:"uuidv4,required"`

	// Artifact total size
	Size int64 `json:"size" bson:"size" valid:"-"`

	// Last modification time, including image upload time
	Modified *time.Time `json:"modified" valid:"-"`
}

// NewSoftwareImage creates new software image object.
func NewSoftwareImage(
	id string,
	metaConstructor *SoftwareImageMetaConstructor,
	metaArtifactConstructor *SoftwareImageMetaArtifactConstructor,
	artifactSize int64) *SoftwareImage {

	now := time.Now()

	return &SoftwareImage{
		SoftwareImageMetaConstructor:         *metaConstructor,
		SoftwareImageMetaArtifactConstructor: *metaArtifactConstructor,
		Modified:                             &now,
		Id:                                   id,
		Size:                                 artifactSize,
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
