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

import "testing"

func TestValidateEmptyImageMeta(t *testing.T) {
	image := NewSoftwareImageMetaConstructor()

	if err := image.Validate(); err == nil {
		t.FailNow()
	}
}

func TestValidateEmptyImageMetaYocto(t *testing.T) {
	image := NewSoftwareImageMetaYoctoConstructor()

	if err := image.Validate(); err == nil {
		t.FailNow()
	}
}

func TestValidateCorrectImageMeta(t *testing.T) {
	image := NewSoftwareImageMetaConstructor()
	required := "required"

	image.Name = required

	if err := image.Validate(); err != nil {
		t.FailNow()
	}
}

func TestValidateCorrectImageMetaYocot(t *testing.T) {
	image := NewSoftwareImageMetaYoctoConstructor()
	required := "required"

	image.YoctoId = required
	image.DeviceType = required
	image.Checksum = required

	if err := image.Validate(); err != nil {
		t.FailNow()
	}
}

func TestValidateCorrectImage(t *testing.T) {
	required := "required"
	imageMeta := NewSoftwareImageMetaConstructor()
	imageMetaYocto := NewSoftwareImageMetaYoctoConstructor()

	imageMetaYocto.YoctoId = required
	imageMetaYocto.DeviceType = required
	imageMetaYocto.Checksum = required
	imageMeta.Name = required

	image := NewSoftwareImage(imageMeta, imageMetaYocto)

	if err := image.Validate(); err != nil {
		t.FailNow()
	}
}
