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

func TestImageMetaPublicValid(t *testing.T) {

	testList := []struct {
		out   error
		image *ImageMetaPublic
	}{
		{ErrMissingImageAttrName, &ImageMetaPublic{}},
		{nil, NewImageMetaPublic("SOMETHING", "SOMETHING", "SOMETHING")},
		{ErrMissingImageAttrModel, NewImageMetaPublic("SOMETHING", "", "SOMETHING")},
		{ErrMissingImageAttrYoctoId, NewImageMetaPublic("SOMETHING", "SOMETHING", "")},
	}

	for id, test := range testList {
		if err := test.out; err != test.image.Valid() {
			t.Errorf("TestCase: %d Error: %s", id, err)
			continue
		}
	}
}
