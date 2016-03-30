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
package mvc

import (
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
)

const (
	HttpHeaderLocation string = "Location"
)

type Viewer interface {
	RenderSuccess(w rest.ResponseWriter, object interface{})
	RenderError(w rest.ResponseWriter, err error, status int)
}

type ViewRestGetOne struct {
	ViewRestError
}

func NewViewRestGetOne() *ViewRestGetOne {
	return &ViewRestGetOne{}
}

// RenderSuccess returns "204 No Content" on nil object or renders object with "200 Success"
func (v *ViewRestGetOne) RenderSuccess(w rest.ResponseWriter, object interface{}) {
	if object == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.WriteJson(object)
}

type ViewRestPost struct {
	baseUri string
	ViewRestError
}

func NewViewRestPost(baseUri string) *ViewRestPost {
	return &ViewRestPost{
		baseUri: baseUri,
	}
}

func (v *ViewRestPost) RenderSuccess(w rest.ResponseWriter, object interface{}) {
	id := object.(string)
	w.Header().Add(HttpHeaderLocation, v.baseUri+"/"+id)
	w.WriteHeader(http.StatusCreated)
}

type ViewRestError struct{}

func (v *ViewRestError) RenderError(w rest.ResponseWriter, err error, status int) {
	rest.Error(w, err.Error(), status)
}
