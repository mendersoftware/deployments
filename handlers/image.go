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
package handlers

import (
	"net/http"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/artifacts/controllers"
	"github.com/mendersoftware/artifacts/models/images"
	"github.com/mendersoftware/artifacts/models/users"
)

const (
	QueryExpireName = "expire"
	// 7 days in minutes
	QueryExpireMaxLimit = 60 * 7 * 24
	// 1 minute
	QueryExpireMinLimit = 1
)

// Takes care of input processing, responce building, calling appropriate controller
type ImageMeta struct {
	controler controllers.ImagesControllerI
}

func NewImageMeta(controler controllers.ImagesControllerI) *ImageMeta {

	return &ImageMeta{
		controler: controler,
	}
}

func (m *ImageMeta) Lookup(w rest.ResponseWriter, r *rest.Request) {

	u := users.NewDummyUser()

	images, err := m.controler.Lookup(u)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteJson(images)
}

func (m *ImageMeta) Get(w rest.ResponseWriter, r *rest.Request) {

	u := users.NewDummyUser()
	id := r.PathParam("id")

	image, err := m.controler.Get(u, id)
	if err != nil {
		rest.NotFound(w, r)
		return
	}

	w.Header().Set(HttpHeaderLastModified, image.LastUpdated.UTC().Format(http.TimeFormat))
	w.WriteJson(image)
}

// Location for GET object is hardcoded here.
func (m *ImageMeta) Create(w rest.ResponseWriter, r *rest.Request) {

	u := users.NewDummyUser()

	// Validate incomming request

	imagePub := &images.ImageMetaPublic{}

	if err := r.DecodeJsonPayload(&imagePub); err != nil {
		rest.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := imagePub.Valid(); err != nil {
		rest.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Pass to controller
	imgNew, err := m.controler.Create(u, imagePub)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add(HttpHeaderLocation, "/api/0.0.1/images/"+imgNew.Id)
	w.WriteHeader(http.StatusCreated)
}

func (m *ImageMeta) Edit(w rest.ResponseWriter, r *rest.Request) {

	u := users.NewDummyUser()
	id := r.PathParam("id")

	// Validate incomming request

	imagePub := &images.ImageMetaPublic{}

	if err := r.DecodeJsonPayload(&imagePub); err != nil {
		rest.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := imagePub.Valid(); err != nil {
		rest.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Pass to controller
	if err := m.controler.Edit(u, id, imagePub); err != nil {
		if err.Error() == controllers.ErrNotFound.Error() {
			rest.NotFound(w, r)
			return
		}

		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add(HttpHeaderLocation, r.URL.RequestURI())
	w.WriteHeader(http.StatusNoContent)
}

func (m *ImageMeta) Delete(w rest.ResponseWriter, r *rest.Request) {

	u := users.NewDummyUser()
	id := r.PathParam("id")

	if err := m.controler.Delete(u, id); err != nil {
		if err.Error() == controllers.ErrNotFound.Error() {
			rest.NotFound(w, r)
			return
		}

		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (m *ImageMeta) UploadLink(w rest.ResponseWriter, r *rest.Request) {

	u := users.NewDummyUser()
	id := r.PathParam("id")

	minutes, err := ParseAndValidateUIntQuery(QueryExpireName,
		r.URL.Query().Get(QueryExpireName),
		QueryExpireMinLimit, QueryExpireMaxLimit)

	if err != nil {
		rest.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	link, err := m.controler.UploadLink(u, id, time.Duration(minutes)*time.Minute)

	if err != nil {
		if err.Error() == controllers.ErrNotFound.Error() {
			rest.NotFound(w, r)
			return
		}

		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(HttpHeaderExpires, link.Expire.UTC().Format(http.TimeFormat))
	w.WriteJson(link)
}

func (m *ImageMeta) DownloadLink(w rest.ResponseWriter, r *rest.Request) {

	u := users.NewDummyUser()
	id := r.PathParam("id")

	minutes, err := ParseAndValidateUIntQuery(QueryExpireName,
		r.URL.Query().Get(QueryExpireName),
		QueryExpireMinLimit, QueryExpireMaxLimit)

	if err != nil {
		rest.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	link, err := m.controler.DownloadLink(u, id, time.Duration(minutes)*time.Minute)

	if err != nil {
		if err.Error() == controllers.ErrNotFound.Error() {
			rest.NotFound(w, r)
			return
		}

		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(HttpHeaderExpires, link.Expire.UTC().Format(http.TimeFormat))
	w.WriteJson(link)

}
