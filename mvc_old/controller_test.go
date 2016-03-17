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
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
)

// Helper implementation of Viewer interface
type testView struct {
}

func (t *testView) RenderSuccess(w rest.ResponseWriter, object interface{}) {
	w.WriteHeader(http.StatusOK)
	w.WriteJson(object)
}

func (t *testView) RenderError(w rest.ResponseWriter, err error, status int) {
	rest.Error(w, err.Error(), status)
}

// Mock implementation of GetObjectModeler interface
type testGetObjectModel struct {
	getObject func(id string) (interface{}, error)
}

func (t *testGetObjectModel) GetObject(id string) (interface{}, error) {
	return t.getObject(id)
}

func TestNewGetObjectController(t *testing.T) {

	testCases := []struct {
		inGetObject   func(id string) (interface{}, error)
		inID          string
		outStatusCode int
		outBody       interface{}
	}{
		{
			inGetObject:   func(id string) (interface{}, error) { return nil, errors.New("Test Error") },
			inID:          "myID",
			outStatusCode: http.StatusInternalServerError,
			outBody:       struct{ Error string }{Error: "Test Error"},
		},
		{
			inGetObject:   func(id string) (interface{}, error) { return nil, nil },
			inID:          "myID",
			outStatusCode: http.StatusNotFound,
			outBody:       struct{ Error string }{Error: "Resource not found"},
		},
		{
			inGetObject:   func(id string) (interface{}, error) { return id, nil },
			inID:          "myID",
			outStatusCode: http.StatusOK,
			outBody:       "myID",
		},
	}

	for testID, testCase := range testCases {

		model := &testGetObjectModel{getObject: testCase.inGetObject}
		view := &testView{}

		handler := NewGetObjectController(model, view)
		router, err := rest.MakeRouter(rest.Get("/:id", handler))
		if err != nil {
			t.Errorf("Test: %d Error: %s", testID, err)
			t.FailNow()
		}

		api := rest.NewApi()
		api.SetApp(router)

		recorded := test.RunRequest(t, api.MakeHandler(),
			test.MakeSimpleRequest("GET", "http://1.2.3.4/"+testCase.inID, nil))

		bodyExpectedJSON, _ := json.Marshal(testCase.outBody)

		recorded.CodeIs(testCase.outStatusCode)
		recorded.ContentTypeIsJson()
		recorded.BodyIs(string(bodyExpectedJSON))
	}
}

// Mock implementation of CreateModeler interface
type testCreateModel struct {
	newObject func() interface{}
	validate  func(interface{}) error
	create    func(interface{}) (string, error)
}

func (t *testCreateModel) NewObject() interface{} {
	return t.newObject()
}

func (t *testCreateModel) Validate(object interface{}) error {
	return t.validate(object)
}

func (t *testCreateModel) Create(object interface{}) (string, error) {
	return t.create(object)
}

func TestNewCreateController(t *testing.T) {

	testCases := []struct {
		newObject     func() interface{}
		validate      func(interface{}) error
		create        func(interface{}) (string, error)
		inBody        interface{}
		outStatusCode int
		outBody       interface{}
	}{
		{
			newObject:     func() interface{} { return nil },
			validate:      func(interface{}) error { return nil },
			create:        func(interface{}) (string, error) { return "", nil },
			inBody:        nil,
			outStatusCode: http.StatusBadRequest,
			outBody:       struct{ Error string }{Error: rest.ErrJsonPayloadEmpty.Error()},
		},
		{
			newObject:     func() interface{} { return new(string) },
			validate:      func(interface{}) error { return errors.New("Test Error") },
			create:        func(interface{}) (string, error) { return "", nil },
			inBody:        "",
			outStatusCode: http.StatusBadRequest,
			outBody:       struct{ Error string }{Error: "Test Error"},
		},
		{
			newObject:     func() interface{} { return new(string) },
			validate:      func(interface{}) error { return errors.New("Test Error") },
			create:        func(interface{}) (string, error) { return "", nil },
			inBody:        "Ala porzucila kota",
			outStatusCode: http.StatusBadRequest,
			outBody:       struct{ Error string }{Error: "Test Error"},
		},
		{
			newObject:     func() interface{} { return new(string) },
			validate:      func(interface{}) error { return nil },
			create:        func(interface{}) (string, error) { return "", errors.New("Test Error") },
			inBody:        "Ala porzucila kota",
			outStatusCode: http.StatusInternalServerError,
			outBody:       struct{ Error string }{Error: "Test Error"},
		},
		{
			newObject:     func() interface{} { return new(string) },
			validate:      func(interface{}) error { return nil },
			create:        func(interface{}) (string, error) { return "", nil },
			inBody:        "Ala porzucila kota",
			outStatusCode: http.StatusOK,
			outBody:       "",
		},
		{
			newObject:     func() interface{} { return new(string) },
			validate:      func(interface{}) error { return nil },
			create:        func(interface{}) (string, error) { return "kajdhkasdhkashd", nil },
			inBody:        "Ala porzucila kota",
			outStatusCode: http.StatusOK,
			outBody:       "kajdhkasdhkashd",
		},
	}

	for testID, testCase := range testCases {

		model := &testCreateModel{
			newObject: testCase.newObject,
			validate:  testCase.validate,
			create:    testCase.create,
		}

		view := &testView{}

		handler := NewCreateController(model, view)
		router, err := rest.MakeRouter(rest.Get("/", handler))
		if err != nil {
			t.Errorf("Test: %d Error: %s", testID, err)
			t.FailNow()
		}

		api := rest.NewApi()
		api.SetApp(router)

		recorded := test.RunRequest(t, api.MakeHandler(),
			test.MakeSimpleRequest("GET", "http://1.2.3.4/", testCase.inBody))

		bodyExpectedJSON, _ := json.Marshal(testCase.outBody)

		recorded.CodeIs(testCase.outStatusCode)
		recorded.ContentTypeIsJson()
		recorded.BodyIs(string(bodyExpectedJSON))
	}
}
