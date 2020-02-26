// Copyright 2020 Northern.tech AS
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
	"io"
	"time"

	"github.com/asaskevich/govalidator"
	"go.mongodb.org/mongo-driver/bson"
)

// Information provided by the user
type ImageMeta struct {
	// Image description
	Description string `json:"description,omitempty" valid:"length(1|4096),optional"`
}

// Creates new, empty ImageMeta
func NewImageMeta() *ImageMeta {
	return &ImageMeta{}
}

// Validate checks structure according to valid tags.
func (s *ImageMeta) Validate() error {
	_, err := govalidator.ValidateStruct(s)
	return err
}

// Structure with artifact version information
type ArtifactInfo struct {
	// Mender artifact format - the only possible value is "mender"
	//Format string `json:"format" valid:"string,equal("mender"),required"`
	Format string `json:"format" valid:"required"`

	// Mender artifact format version
	//Version uint `json:"version" valid:"uint,equal(1),required"`
	Version uint `json:"version" valid:"required"`
}

// Information provided by the Mender Artifact header
type ArtifactMeta struct {
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

	// Provides is a map[string]interface{} (JSON) of artifact_provides used
	// for checking artifact (version 3) dependencies.
	Provides bson.M `json:"artifact_provides,omitempty" bson:"provides" valid:"-"`

	// Depends is a map[string]interface{} (JSON) of artifact_depends used
	// for checking/validate against artifact (version 3) provides.
	Depends bson.M `json:"artifact_depends,omitempty" bson:"depends" valid:"-"`

	// DependsIdx is a list of 'exploded' depends (cross products across all array items)
	// used only internally as an index for detecting 'uniqueness', i.e. array overlap
	DependsIdx []bson.D `json:"-" bson:"depends_idx" valid:"-"`
}

func NewArtifactMeta() *ArtifactMeta {
	return &ArtifactMeta{}
}

// Validate checks structure according to valid tags.
func (s *ArtifactMeta) Validate() error {
	_, err := govalidator.ValidateStruct(s)
	return err
}

// Image YOCTO image with user application
type Image struct {
	// User provided field set
	ImageMeta `bson:"meta"`

	// Field set provided with yocto image
	ArtifactMeta `bson:"meta_artifact"`

	// Image ID
	Id string `json:"id" bson:"_id" valid:"uuidv4,required"`

	// Artifact total size
	Size int64 `json:"size" bson:"size" valid:"-"`

	// Last modification time, including image upload time
	Modified *time.Time `json:"modified" valid:"-"`
}

// NewImage creates new software image object.
func NewImage(
	id string,
	metaConstructor *ImageMeta,
	metaArtifactConstructor *ArtifactMeta,
	artifactSize int64) *Image {

	now := time.Now()

	return &Image{
		ImageMeta:    *metaConstructor,
		ArtifactMeta: *metaArtifactConstructor,
		Modified:     &now,
		Id:           id,
		Size:         artifactSize,
	}
}

// SetModified set last modification time for the image.
func (s *Image) SetModified(time time.Time) {
	s.Modified = &time
}

// Validate checks structure according to valid tags.
func (s *Image) Validate() error {
	_, err := govalidator.ValidateStruct(s)
	return err
}

// MultipartUploadMsg is a structure with fields extracted from the multipart/form-data form
// send in the artifact upload request
type MultipartUploadMsg struct {
	// user metadata constructor
	MetaConstructor *ImageMeta
	// ArtifactID contains the artifact ID
	ArtifactID string
	// size of the artifact file
	ArtifactSize int64
	// reader pointing to the beginning of the artifact data
	ArtifactReader io.Reader
}

// MultipartGenerateImageMsg is a structure with fields extracted from the multipart/form-data
// form sent in the artifact generation request
type MultipartGenerateImageMsg struct {
	Name                  string    `json:"name"`
	Description           string    `json:"description"`
	Size                  int64     `json:"size"`
	DeviceTypesCompatible []string  `json:"device_types_compatible"`
	Type                  string    `json:"type"`
	Args                  string    `json:"args"`
	ArtifactID            string    `json:"artifact_id"`
	GetArtifactURI        string    `json:"get_artifact_uri"`
	DeleteArtifactURI     string    `json:"delete_artifact_uri"`
	TenantID              string    `json:"tenant_id"`
	Token                 string    `json:"token"`
	FileReader            io.Reader `json:"-"`
}
