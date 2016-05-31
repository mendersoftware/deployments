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
import "time"

func TestValidateEmptyImage(t *testing.T) {
	image := NewSoftwareImageConstructor()

	if err := image.Validate(); err == nil {
		t.FailNow()
	}
}

func TestValidateCorrectImage(t *testing.T) {
	image := NewSoftwareImageConstructor()
	required := "required"

	image.YoctoId = &required
	image.Name = &required
	image.Model = &required

	if err := image.Validate(); err != nil {
		t.FailNow()
	}
}

func TestValidateEmptyImageFromConstructor(t *testing.T) {
	image := NewSoftwareImageConstructor()

	constructorImage := NewSoftwareImageFromConstructor(image)
	if err := constructorImage.Validate(); err != nil {
		t.FailNow()
	}
}

func TestModifyImageSetTime(t *testing.T) {
	image := NewSoftwareImageConstructor()

	constructorImage := NewSoftwareImageFromConstructor(image)
	constructorImage.Validate()

	modifiedTime := time.Now().Add(time.Hour)
	constructorImage.SetModified(modifiedTime)

	if !modifiedTime.Equal(*constructorImage.Modified) {
		t.FailNow()
	}

}
