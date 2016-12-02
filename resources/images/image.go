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

// Informations provided by the user
type SoftwareImageMetaConstructor struct {
	// Application Name & Version
	Name string `json:"name" bson:"name" valid:"length(1|4096),required"`

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

// Type info structure
type ArtifactUpdateTypeInfo struct {
	Type string `json:"type" valid:"required"`
}

// Update file structure
type UpdateFile struct {
	// Image name
	Name string `json:"name" valid:"required"`

	// Image file checksum
	Checksum string `json:"checksum" valid:"optional"`

	// Image file signature
	Signature string `json:"signature" valid:"optional"`

	// Image size
	Size int64 `json:"size" valid:"optional"`

	// Date build
	Date *time.Time `json:"date" valid:"optional"`
}

// Update structure
type Update struct {
	TypeInfo ArtifactUpdateTypeInfo `json:"type_info" valid:"required"`
	Files    []UpdateFile           `json:"files"`
	MetaData interface{}            `json:"meta_data" valid:"optional"` //TODO check this
}

// Information provided with YOCTO image
type SoftwareImageMetaArtifactConstructor struct {
	// artifact_name from artifact file
	ArtifactName string `json:"artifact_name" bson:"artifact_name" valid:"length(1|4096),required"`

	// Compatible device types for the application
	DeviceTypesCompatible []string `json:"device_types_compatible" bson:"device_types_compatible" valid:"length(1|4096),required"`

	// Artifact version info
	Info *ArtifactInfo `json:"info" valid:"required"`

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

	// Last modification time, including image upload time
	Modified *time.Time `json:"modified" valid:"_"`
}

// NewSoftwareImage create new software image object.
func NewSoftwareImage(
	metaConstructor *SoftwareImageMetaConstructor,
	metaArtifactConstructor *SoftwareImageMetaArtifactConstructor) *SoftwareImage {

	now := time.Now()
	id := uuid.NewV4().String()

	return &SoftwareImage{
		SoftwareImageMetaConstructor:         *metaConstructor,
		SoftwareImageMetaArtifactConstructor: *metaArtifactConstructor,
		Modified: &now,
		Id:       id,
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
