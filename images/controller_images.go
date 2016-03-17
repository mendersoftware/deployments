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

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/asaskevich/govalidator"
	"github.com/mendersoftware/artifacts/mvc"
	"github.com/pkg/errors"
)

// API input validation constants
const (
	DefaultDownloadLinkExpire = 60
	DefaultUploadLinkExpire   = 60

	// AWS limitation is 1 week
	MaxLinkExpire = 60 * 7 * 24
)

var (
	ErrIDNotUUIDv4        = errors.New("ID is not UUIDv4")
	ErrInvalidExpireParam = errors.New("Invalid expire parameter")
)

type ImagesModeler interface {
	ListImages(filters map[string]string) ([]*SoftwareImage, error)
	UploadLink(imageID string, expire time.Duration) (*Link, error)
	DownloadLink(imageID string, expire time.Duration) (*Link, error)
	GetImage(id string) (*SoftwareImage, error)
	DeleteImage(imageID string) error
	CreateImage(constructorData *SoftwareImageConstructor) (string, error)
	EditImage(id string, constructorData *SoftwareImageConstructor) (bool, error)
}

type SoftwareImagesController struct {
	views mvc.RESTViewDefaults
	model ImagesModeler
}

func NewSoftwareImagesController(model ImagesModeler, views mvc.RESTViewDefaults) *SoftwareImagesController {
	return &SoftwareImagesController{
		model: model,
		views: views,
	}
}

func (s *SoftwareImagesController) GetImage(w rest.ResponseWriter, r *rest.Request) {

	id := r.PathParam("id")

	if !govalidator.IsUUIDv4(id) {
		s.views.RenderError(w, ErrIDNotUUIDv4, http.StatusBadRequest)
		return
	}

	image, err := s.model.GetImage(id)
	if err != nil {
		s.views.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	if image == nil {
		s.views.RenderErrorNotFound(w)
	}

	s.views.RenderSuccessGet(w, image)
}

func (s *SoftwareImagesController) ListImages(w rest.ResponseWriter, r *rest.Request) {

	list, err := s.model.ListImages(r.PathParams)
	if err != nil {
		s.views.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	s.views.RenderSuccessGet(w, list)
}

func (s *SoftwareImagesController) UploadLink(w rest.ResponseWriter, r *rest.Request) {

	id := r.PathParam("id")

	if !govalidator.IsUUIDv4(id) {
		s.views.RenderError(w, ErrIDNotUUIDv4, http.StatusBadRequest)
		return
	}

	expire := DefaultDownloadLinkExpire
	expireStr := r.URL.Query().Get("expire")

	// Validate input
	if !govalidator.IsNull(expireStr) {
		if !s.validExpire(expireStr) {
			s.views.RenderError(w, ErrInvalidExpireParam, http.StatusBadRequest)
			return
		}

		var err error
		expire, err = strconv.Atoi(expireStr)
		if err != nil {
			s.views.RenderError(w, err, http.StatusInternalServerError)
			return
		}
	}

	link, err := s.model.UploadLink(id, time.Duration(expire)*time.Minute)
	if err != nil {
		s.views.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	if link == nil {
		s.views.RenderErrorNotFound(w)
		return
	}

	s.views.RenderSuccessGet(w, link)
}

func (s *SoftwareImagesController) DownloadLink(w rest.ResponseWriter, r *rest.Request) {

	id := r.PathParam("id")

	if !govalidator.IsUUIDv4(id) {
		s.views.RenderError(w, ErrIDNotUUIDv4, http.StatusBadRequest)
		return
	}

	expire := DefaultUploadLinkExpire
	expireStr := r.URL.Query().Get("expire")

	// Validate input
	if !govalidator.IsNull(expireStr) {
		if !s.validExpire(expireStr) {
			s.views.RenderError(w, ErrInvalidExpireParam, http.StatusBadRequest)
			return
		}

		var err error
		expire, err = strconv.Atoi(expireStr)
		if err != nil {
			s.views.RenderError(w, err, http.StatusInternalServerError)
			return
		}
	}

	link, err := s.model.DownloadLink(id, time.Duration(expire)*time.Minute)
	if err != nil {
		s.views.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	if link == nil {
		s.views.RenderErrorNotFound(w)
		return
	}

	s.views.RenderSuccessGet(w, link)
}

func (s *SoftwareImagesController) validExpire(expire string) bool {

	if govalidator.IsNull(expire) {
		return false
	}

	if !govalidator.IsInt(expire) {
		return false
	}

	number, err := strconv.ParseFloat(expire, 64)
	if err != nil {
		return false
	}

	if !govalidator.InRange(number, 0, MaxLinkExpire) {
		return false
	}

	return true
}

func (s *SoftwareImagesController) DeleteImage(w rest.ResponseWriter, r *rest.Request) {

	id := r.PathParam("id")

	if !govalidator.IsUUIDv4(id) {
		s.views.RenderError(w, ErrIDNotUUIDv4, http.StatusBadRequest)
		return
	}

	if err := s.model.DeleteImage(id); err != nil {
		s.views.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	s.views.RenderSuccessDelete(w)
}

func (s *SoftwareImagesController) EditImage(w rest.ResponseWriter, r *rest.Request) {

	id := r.PathParam("id")

	if !govalidator.IsUUIDv4(id) {
		s.views.RenderError(w, ErrIDNotUUIDv4, http.StatusBadRequest)
		return
	}

	var constructor *SoftwareImageConstructor

	if err := r.DecodeJsonPayload(&constructor); err != nil {
		s.views.RenderError(w, err, http.StatusBadRequest)
		return
	}

	if err := constructor.Validate(); err != nil {
		s.views.RenderError(w, err, http.StatusBadRequest)
		return
	}

	found, err := s.model.EditImage(id, constructor)
	if err != nil {
		s.views.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	if !found {
		s.views.RenderErrorNotFound(w)
		return
	}

	s.views.RenderSuccessPut(w)
}

func (s *SoftwareImagesController) NewImage(w rest.ResponseWriter, r *rest.Request) {

	var constructor *SoftwareImageConstructor

	if err := r.DecodeJsonPayload(&constructor); err != nil {
		s.views.RenderError(w, err, http.StatusBadRequest)
		return
	}

	if err := constructor.Validate(); err != nil {
		s.views.RenderError(w, err, http.StatusBadRequest)
		return
	}

	id, err := s.model.CreateImage(constructor)
	if err != nil {
		s.views.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	s.views.RenderSuccessPost(w, r, id)
}
