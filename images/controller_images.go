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
		return
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

	expire, err := s.getLinkExpireParam(r, DefaultDownloadLinkExpire)
	if err != nil {
		s.views.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	link, err := s.model.UploadLink(id, expire)
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

	expire, err := s.getLinkExpireParam(r, DefaultUploadLinkExpire)
	if err != nil {
		s.views.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	link, err := s.model.DownloadLink(id, expire)
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

func (s *SoftwareImagesController) getLinkExpireParam(r *rest.Request, defaultValue uint64) (time.Duration, error) {

	expire := defaultValue
	expireStr := r.URL.Query().Get("expire")

	// Validate input
	if !govalidator.IsNull(expireStr) {
		if !s.validExpire(expireStr) {
			return 0, ErrInvalidExpireParam
		}

		var err error
		expire, err = strconv.ParseUint(expireStr, 10, 64)
		if err != nil {
			return 0, err
		}
	}

	return time.Duration(int64(expire)) * time.Minute, nil
}

func (s *SoftwareImagesController) validExpire(expire string) bool {

	if govalidator.IsNull(expire) {
		return false
	}

	number, err := strconv.ParseUint(expire, 10, 64)
	if err != nil {
		return false
	}

	if number > MaxLinkExpire {
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

	constructor, err := s.getSoftwareImageConstructorFromBody(r)
	if err != nil {
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

	constructor, err := s.getSoftwareImageConstructorFromBody(r)
	if err != nil {
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

func (s SoftwareImagesController) getSoftwareImageConstructorFromBody(r *rest.Request) (*SoftwareImageConstructor, error) {

	var constructor *SoftwareImageConstructor

	if err := r.DecodeJsonPayload(&constructor); err != nil {
		return nil, err
	}

	if err := constructor.Validate(); err != nil {
		return nil, err
	}

	return constructor, nil
}
