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

package view_test

import (
	"net/http"
	"testing"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	. "github.com/mendersoftware/deployments/resources/images/view"
	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/stretchr/testify/assert"
)

func TestRenderPost(t *testing.T) {

	router, err := rest.MakeRouter(rest.Post("/test", func(w rest.ResponseWriter, r *rest.Request) {
		new(RESTView).RenderSuccessPost(w, r, "test_id")
	}))

	if err != nil {
		assert.NoError(t, err)
	}

	api := rest.NewApi()
	api.SetApp(router)

	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("POST", "http://localhost/test", "blah"))

	recorded.CodeIs(http.StatusCreated)
	recorded.ContentTypeIsJson()
	recorded.HeaderIs(HttpHeaderLocation, "http://localhost/test/test_id")
}

func TestRenderSuccessGet(t *testing.T) {

	router, err := rest.MakeRouter(rest.Get("/test", func(w rest.ResponseWriter, r *rest.Request) {
		new(RESTView).RenderSuccessGet(w, "test")
	}))

	if err != nil {
		assert.NoError(t, err)
	}

	api := rest.NewApi()
	api.SetApp(router)

	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/test", nil))

	recorded.CodeIs(http.StatusOK)
	recorded.ContentTypeIsJson()
	recorded.BodyIs(`"test"`)
}

func TestRenderSuccessDelete(t *testing.T) {

	router, err := rest.MakeRouter(rest.Delete("/test", func(w rest.ResponseWriter, r *rest.Request) {
		new(RESTView).RenderSuccessDelete(w)
	}))

	if err != nil {
		assert.NoError(t, err)
	}

	api := rest.NewApi()
	api.SetApp(router)

	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("DELETE", "http://localhost/test", nil))

	recorded.CodeIs(http.StatusNoContent)
}

func TestRenderSuccessPut(t *testing.T) {

	router, err := rest.MakeRouter(rest.Put("/test", func(w rest.ResponseWriter, r *rest.Request) {
		new(RESTView).RenderSuccessPut(w)
	}))

	if err != nil {
		assert.NoError(t, err)
	}

	api := rest.NewApi()
	api.SetApp(router)

	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("PUT", "http://localhost/test", nil))

	recorded.CodeIs(http.StatusNoContent)
}

func TestRenderErrorNotFound(t *testing.T) {

	router, err := rest.MakeRouter(rest.Get("/test", func(w rest.ResponseWriter, r *rest.Request) {

		l := log.New(log.Ctx{})
		new(RESTView).RenderErrorNotFound(w, r, l)
	}))

	if err != nil {
		assert.NoError(t, err)
	}

	api := rest.NewApi()
	api.SetApp(router)

	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/test", nil))

	recorded.CodeIs(http.StatusNotFound)
	recorded.BodyIs(`{"error":"Resource not found","request_id":""}`)
}
