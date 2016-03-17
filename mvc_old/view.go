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
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
)

const (
	HttpHeaderLocation string = "Location"
)

var (
	ErrNotFound = errors.New("Resource not found")
)

type Viewer interface {
	RenderSuccess(w rest.ResponseWriter, object interface{})
	RenderError(w rest.ResponseWriter, err error, status int)
}

// TODO: device model specific view, perhaps move out of package
type ViewRestGetDeviceUpdateObject struct {
	ViewRestError
}

func NewViewRestGetDeviceUpdateObject() *ViewRestGetDeviceUpdateObject {
	return &ViewRestGetDeviceUpdateObject{}
}

// RenderSuccess returns "204 No Content" on nil object or renders object with "200 Success"
func (v *ViewRestGetDeviceUpdateObject) RenderSuccess(w rest.ResponseWriter, object interface{}) {
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
	if err == nil {
		w.WriteHeader(status)
		return
	}

	rest.Error(w, err.Error(), status)
}

type ViewRestGet struct {
	ViewRestError
}

// NewViewRestGet render view for single object response, 404 on nil object
func NewViewRestGet() *ViewRestGet {
	return &ViewRestGet{}
}

func (v *ViewRestGet) RenderSuccess(w rest.ResponseWriter, object interface{}) {
	if object == nil {
		v.RenderError(w, ErrNotFound, http.StatusNotFound)
		return
	}

	w.WriteJson(object)
}

type ViewRestDelete struct {
	ViewRestError
}

// NewViewRestDelete render view for object deletion
func NewViewRestDelete() *ViewRestDelete {
	return &ViewRestDelete{}
}

func (v *ViewRestDelete) RenderSuccess(w rest.ResponseWriter, object interface{}) {
	w.WriteHeader(http.StatusNoContent)
}

type ViewRestList struct {
	ViewRestError
}

// NewViewRestList render view for object lookup
func NewViewRestList() *ViewRestList {
	return &ViewRestList{}
}

func (v *ViewRestList) RenderSuccess(w rest.ResponseWriter, object interface{}) {
	if object == nil {
		v.RenderError(w, ErrNotFound, http.StatusNotFound)
		return
	}

	w.WriteJson(object)
}

type ViewRestPut struct {
	ViewRestError
}

// NewViewRestPut render view for object edition
func NewViewRestPut() *ViewRestPut {
	return &ViewRestPut{}
}

func (v *ViewRestPut) RenderSuccess(w rest.ResponseWriter, object interface{}) {
	w.WriteHeader(http.StatusNoContent)
}
