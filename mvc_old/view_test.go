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
package mvc_old

import (
	"errors"
	"reflect"
	"testing"

	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
)

func TestViewRestGetDeviceUpdateObjectSuccess(t *testing.T) {
	testList := []struct {
		inObject  interface{}
		outCode   int
		outObject interface{}
	}{
		{
			inObject: nil,
			outCode:  http.StatusNoContent,
		},
		{
			inObject:  "Message",
			outCode:   http.StatusOK,
			outObject: "Message",
		},
	}

	for testID, testCase := range testList {
		router, err := rest.MakeRouter(rest.Get("/api", func(w rest.ResponseWriter, r *rest.Request) {
			view := NewViewRestGetDeviceUpdateObject()
			view.RenderSuccess(w, testCase.inObject)
		}))

		if err != nil {
			t.Errorf("Test: %d Error: %s", testID, err)
			t.FailNow()
		}

		api := rest.NewApi()
		api.SetApp(router)

		recorded := test.RunRequest(t, api.MakeHandler(),
			test.MakeSimpleRequest("GET", "http://hostname/api", nil))

		recorded.CodeIs(testCase.outCode)
		recorded.ContentTypeIsJson()

		if testCase.outObject != nil {
			var body interface{}
			if err := recorded.DecodeJsonPayload(&body); err != nil {
				t.Errorf("Test: %d Error: %s", testID, err)
				t.FailNow()
			}

			if !reflect.DeepEqual(body, testCase.outObject) {
				t.Errorf("Test: %d Error: %s %v != %v", testID, "Body has undexpected output.", body, testCase.outObject)
				t.FailNow()
			}
		}
	}
}

func TestViewRestGetDeviceUpdateObjectError(t *testing.T) {
	testList := []struct {
		inError   error
		inCode    int
		outCode   int
		outObject interface{}
	}{
		{
			inError: nil,
			inCode:  http.StatusInternalServerError,
			outCode: http.StatusInternalServerError,
		},
		{
			inError:   errors.New("Message"),
			inCode:    http.StatusInternalServerError,
			outCode:   http.StatusInternalServerError,
			outObject: map[string]string{"Error": "Message"},
		},
	}

	for testID, testCase := range testList {
		router, err := rest.MakeRouter(rest.Get("/api", func(w rest.ResponseWriter, r *rest.Request) {
			view := NewViewRestGetDeviceUpdateObject()
			view.RenderError(w, testCase.inError, testCase.inCode)
		}))

		if err != nil {
			t.Errorf("Test: %d Error: %s", testID, err)
			t.FailNow()
		}

		api := rest.NewApi()
		api.SetApp(router)

		recorded := test.RunRequest(t, api.MakeHandler(),
			test.MakeSimpleRequest("GET", "http://hostname/api", nil))

		recorded.CodeIs(testCase.outCode)
		recorded.ContentTypeIsJson()

		if testCase.outObject != nil {
			var body map[string]string
			if err := recorded.DecodeJsonPayload(&body); err != nil {
				t.Errorf("Test: %d Error: %s", testID, err)
				t.FailNow()
			}

			if !reflect.DeepEqual(body, testCase.outObject) {
				t.Errorf("Test: %d Error: %s %v != %v", testID, "Body has undexpected output.", body, testCase.outObject)
				t.FailNow()
			}
		}
	}
}

func TestViewRestPostSuccess(t *testing.T) {
	testList := []struct {
		inObject interface{}
		location string
	}{
		{
			inObject: "id",
			location: "/api/id",
		},
	}

	for testID, testCase := range testList {
		router, err := rest.MakeRouter(rest.Get("/api", func(w rest.ResponseWriter, r *rest.Request) {
			view := NewViewRestPost("/api")
			view.RenderSuccess(w, testCase.inObject)
		}))

		if err != nil {
			t.Errorf("Test: %d Error: %s", testID, err)
			t.FailNow()
		}

		api := rest.NewApi()
		api.SetApp(router)

		recorded := test.RunRequest(t, api.MakeHandler(),
			test.MakeSimpleRequest("GET", "http://hostname/api", nil))

		recorded.CodeIs(http.StatusCreated)
		recorded.HeaderIs(HttpHeaderLocation, testCase.location)

	}
}
