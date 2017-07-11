// Copyright 2017 Northern.tech AS
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
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetDeviceInventory(t *testing.T) {

	t.Parallel()

	tm := time.Unix(10, 10).UTC()
	testCases := map[string]struct {
		// Input
		Code int
		Body interface{}

		//Output
		Device *Device
		Err    error
	}{
		"random server error": {
			Code: http.StatusUnavailableForLegalReasons,
			Err:  errors.New("error server response: parsing server error response: EOF"),
		},
		"internal server error with payload": {
			Code: http.StatusInternalServerError,
			Body: struct {
				Error string `json:"error"`
			}{Error: "dead db"},

			Err: errors.New("error server response: dead db"),
		},
		"not found": {
			Code: http.StatusNotFound,
		},
		"success - broken payload": {
			Code: http.StatusOK,
			Body: &Device{},

			Err: errors.New("validating server response: ID: non zero value required;Updated: non zero value required;"),
		},
		"success": {
			Code: http.StatusOK,
			Body: &Device{
				ID:      "lalala",
				Updated: tm,
				Attributes: []*Attribute{
					{
						Name:        "sialalala",
						Description: "lala",
						Value:       "something",
					},
				},
			},

			Device: &Device{
				ID:      "lalala",
				Updated: tm,
				Attributes: []*Attribute{
					{
						Name:        "sialalala",
						Description: "lala",
						Value:       "something",
					},
				},
			},
		},
	}

	for caseName, test := range testCases {

		t.Logf("Case: %s\n", caseName)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(test.Code)
			if test.Body != nil {
				payload, err := json.Marshal(test.Body)
				assert.NoError(t, err, "invalid test")

				_, err = w.Write(payload)
				assert.NoError(t, err, "invalid test")
			}
		}))
		defer ts.Close()

		api, err := NewMenderAPI(ts.URL)
		assert.NoError(t, err, "api client init")

		device, err := api.GetDeviceInventory(context.TODO(), DeviceID("whatever"))

		if test.Err != nil {
			assert.EqualError(t, err, test.Err.Error())
		} else {
			assert.NoError(t, err)
		}

		assert.EqualValues(t, test.Device, device)
	}

}
