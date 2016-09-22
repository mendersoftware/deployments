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

package integration

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithHTTPClient(t *testing.T) {

	t.Parallel()

	cases := map[string]struct {
		Client *http.Client
		Uri    string

		OutErr error
	}{
		"Nil client, set default": {
			Uri:    "http://localhost",
			Client: nil,
			OutErr: nil,
		},
		"Set client to custom one": {
			Uri:    "http://localhost",
			Client: &http.Client{},
			OutErr: nil,
		},
		"broken uri": {
			Uri:    "ht/localhost",
			OutErr: errors.New("invalid server uri"),
		},
	}

	for caseName, test := range cases {

		t.Logf("Case: %s \n", caseName)

		api, err := NewMenderAPI(test.Uri, WithHTTPClient(test.Client))

		if err == nil {
			assert.NotNil(t, api)

			if test.Client != nil {
				// Accesing internal field
				assert.Equal(t, test.Client, api.client)
			}
		} else {
			assert.Equal(t, test.OutErr, err)
			assert.Nil(t, api)
		}
	}

}
