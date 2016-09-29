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

package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/asaskevich/govalidator"
	"github.com/mendersoftware/artifacts/parser"
	"github.com/mendersoftware/artifacts/reader"
	"github.com/mendersoftware/deployments/resources/images"
	"github.com/pkg/errors"
)

// API input validation constants
const (
	DefaultDownloadLinkExpire = 60

	// AWS limitation is 1 week
	MaxLinkExpire = 60 * 7 * 24
)

var (
	ErrIDNotUUIDv4        = errors.New("ID is not UUIDv4")
	ErrInvalidExpireParam = errors.New("Invalid expire parameter")
)

type SoftwareImagesController struct {
	view  RESTView
	model ImagesModel
}

func NewSoftwareImagesController(model ImagesModel, view RESTView) *SoftwareImagesController {
	return &SoftwareImagesController{
		model: model,
		view:  view,
	}
}

func (s *SoftwareImagesController) GetImage(w rest.ResponseWriter, r *rest.Request) {

	id := r.PathParam("id")

	if !govalidator.IsUUIDv4(id) {
		s.view.RenderError(w, ErrIDNotUUIDv4, http.StatusBadRequest)
		return
	}

	image, err := s.model.GetImage(id)
	if err != nil {
		s.view.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	if image == nil {
		s.view.RenderErrorNotFound(w)
		return
	}

	s.view.RenderSuccessGet(w, image)
}

func (s *SoftwareImagesController) ListImages(w rest.ResponseWriter, r *rest.Request) {

	list, err := s.model.ListImages(r.PathParams)
	if err != nil {
		s.view.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	s.view.RenderSuccessGet(w, list)
}

func (s *SoftwareImagesController) DownloadLink(w rest.ResponseWriter, r *rest.Request) {

	id := r.PathParam("id")

	if !govalidator.IsUUIDv4(id) {
		s.view.RenderError(w, ErrIDNotUUIDv4, http.StatusBadRequest)
		return
	}

	expire, err := s.getLinkExpireParam(r, DefaultDownloadLinkExpire)
	if err != nil {
		s.view.RenderError(w, err, http.StatusBadRequest)
		return
	}

	link, err := s.model.DownloadLink(id, expire)
	if err != nil {
		s.view.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	if link == nil {
		s.view.RenderErrorNotFound(w)
		return
	}

	s.view.RenderSuccessGet(w, link)
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
		s.view.RenderError(w, ErrIDNotUUIDv4, http.StatusBadRequest)
		return
	}

	if err := s.model.DeleteImage(id); err != nil {
		if err == ErrImageMetaNotFound {
			s.view.RenderErrorNotFound(w)
			return
		}
		s.view.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	s.view.RenderSuccessDelete(w)
}

func (s *SoftwareImagesController) EditImage(w rest.ResponseWriter, r *rest.Request) {

	id := r.PathParam("id")

	if !govalidator.IsUUIDv4(id) {
		s.view.RenderError(w, ErrIDNotUUIDv4, http.StatusBadRequest)
		return
	}

	constructor, err := s.getSoftwareImageMetaConstructorFromBody(r)
	if err != nil {
		s.view.RenderError(w, errors.Wrap(err, "Validating request body"), http.StatusBadRequest)
		return
	}

	found, err := s.model.EditImage(id, constructor)
	if err != nil {
		s.view.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	if !found {
		s.view.RenderErrorNotFound(w)
		return
	}

	s.view.RenderSuccessPut(w)
}

// Multipart Image/Meta upload handler.
// Request should be of type "multipart/mixed".
// First part should contain Metadata file. This file should be of type "application/json".
// Second part should contain Image file. This part should be of type "application/octet-strem".
func (s *SoftwareImagesController) NewImage(w rest.ResponseWriter, r *rest.Request) {

	// limits just for safety;
	const (
		// Max image size
		DefaultMaxImageSize = 1024 * 1024 * 1024 * 10
		// Max meta size
		DefaultMaxMetaSize = 1024 * 1024 * 10
	)

	// parse content type and params according to RFC 1521
	contentType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		s.view.RenderError(w, err, http.StatusBadRequest)
		return
	}
	if contentType != "multipart/mixed" {
		s.view.RenderError(
			w, errors.New("Content-Type should be multipart/mixed"),
			http.StatusUnsupportedMediaType)
		return
	}

	mr := multipart.NewReader(r.Body, params["boundary"])

	// fist part is the metadata part
	p, err := mr.NextPart()
	if err != nil {
		s.view.RenderError(
			w, errors.Wrap(err, "Request does not contain metadata part"),
			http.StatusBadRequest)
		return
	}
	metaConstructor, status, err := s.handleMeta(p, DefaultMaxMetaSize)
	if err != nil {
		s.view.RenderError(w, err, status)
		return
	}

	// Second part is the image part
	p, err = mr.NextPart()
	if err != nil {
		s.view.RenderError(
			w, errors.Wrap(err, "Request does not contain image part"),
			http.StatusBadRequest)
		return
	}
	imageFile, metaYoctoConstructor, status, err := s.handleImage(p, DefaultMaxImageSize)
	if err != nil {
		s.view.RenderError(w, err, status)
		return
	}
	defer os.Remove(imageFile.Name())
	defer imageFile.Close()

	imgId, err := s.model.CreateImage(imageFile, metaConstructor, metaYoctoConstructor)
	if err != nil {
		// TODO: check if this is bad request or internal error
		s.view.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	s.view.RenderSuccessPost(w, r, imgId)
	return
}

// Meta part of multipart meta/image request handler.
// Parses meta body, returns image constructor, success code and nil on success.
func (s *SoftwareImagesController) handleMeta(p *multipart.Part, maxMetaSize int64) (*images.SoftwareImageMetaConstructor, int, error) {
	if p.Header.Get("Content-Type") != "application/json" {
		return nil, http.StatusBadRequest, errors.New("First part should be a metadata (application/json)")
	}
	metaReader := io.LimitReader(p, maxMetaSize)
	metaPart, err := ioutil.ReadAll(metaReader)
	if err != nil && err != io.EOF {
		return nil, http.StatusBadRequest, errors.Wrap(err, "Failed to obtain metadata")
	}
	//parse meta
	var constructor *images.SoftwareImageMetaConstructor
	if err := json.Unmarshal(metaPart, &constructor); err != nil {
		return nil, http.StatusBadRequest, errors.Wrap(err, "Parsing matadata")
	}
	if err := constructor.Validate(); err != nil {
		return nil, http.StatusBadRequest, errors.Wrap(err, "Validating metadata")
	}
	return constructor, http.StatusOK, nil
}

// Image part of multipart meta/image request handler.
// Saves uploaded image in temporary file.
// Returns temporary file name, success code and nil on success.
func (s *SoftwareImagesController) handleImage(
	p *multipart.Part, maxImageSize int64) (*os.File, *images.SoftwareImageMetaYoctoConstructor, int, error) {
	if p.Header.Get("Content-Type") != "application/octet-stream" {
		return nil, nil, http.StatusBadRequest, errors.New("Second part should be an image (octet-stream)")
	}

	tmpfile, err := ioutil.TempFile("", "firmware-")
	if err != nil {
		return nil, nil, http.StatusInternalServerError, err
	}

	//w := io.MultiWriter(aReader, tmpfile)
	n, err := io.CopyN(tmpfile, p, maxImageSize+1)
	if err != nil && err != io.EOF {
		return nil, nil, http.StatusBadRequest, errors.Wrap(err, "Request body invalid")
	}
	if n == maxImageSize+1 {
		return nil, nil, http.StatusBadRequest, errors.New("Image file too large")
	}

	// return to the beginning of the file
	_, err = tmpfile.Seek(0, 0)
	if err != nil {
		return nil, nil, http.StatusInternalServerError, err
	}

	meta, err := s.getMetaFromArchive(tmpfile)
	if err != nil {
		return nil, nil, http.StatusBadRequest, err
	}

	return tmpfile, meta, http.StatusOK, nil
}

func (s *SoftwareImagesController) getMetaFromArchive(f *os.File) (*images.SoftwareImageMetaYoctoConstructor, error) {
	metaYocto := images.NewSoftwareImageMetaYoctoConstructor()
	aReader := areader.NewReader(f)
	defer aReader.Close()
	rp := &parser.RootfsParser{}
	aReader.Register(rp)

	_, err := aReader.ReadInfo()
	if err != nil {
		return nil, errors.Wrap(err, "info error")
	}
	hInfo, err := aReader.ReadHeaderInfo()
	if err != nil {
		return nil, errors.Wrap(err, "header info error")
	}
	//check if there is only one update
	if len(hInfo.Updates) != 1 {
		return nil, errors.New("Too many updats")
	}
	uCnt := 0
	for cnt, update := range hInfo.Updates {
		if update.Type == "rootfs-image" {
			rp := &parser.RootfsParser{}
			aReader.PushWorker(rp, fmt.Sprintf("%04d", cnt))
			uCnt += 1
		}
	}
	if uCnt != 1 {
		return nil, errors.New("Only rootfs-image updates supported")
	}

	_, err = aReader.ReadHeader()
	if err != nil {
		return nil, errors.Wrap(err, "header error")
	}
	w, err := aReader.ReadData()
	if err != nil {
		return nil, errors.Wrap(err, "read data error")
	}
	for _, p := range w {
		deviceType := p.GetDeviceType()
		metaYocto.DeviceType = &deviceType
		if rp, ok := p.(*parser.RootfsParser); ok {
			yoctoId := rp.GetImageID()
			metaYocto.YoctoId = &yoctoId
		}
		updateFiles := p.GetUpdateFiles()
		if len(updateFiles) != 1 {
			return nil, errors.New("Too many update files")
		}
		for _, u := range updateFiles {
			checksum := string(u.Checksum)
			metaYocto.Checksum = &checksum
			metaYocto.ImageSize = u.Size / (1024 * 1024)
			metaYocto.DateBuilt = u.Date
		}
	}
	return metaYocto, nil
}

func (s SoftwareImagesController) getSoftwareImageMetaConstructorFromBody(r *rest.Request) (*images.SoftwareImageMetaConstructor, error) {

	var constructor *images.SoftwareImageMetaConstructor

	if err := r.DecodeJsonPayload(&constructor); err != nil {
		return nil, err
	}

	if err := constructor.Validate(); err != nil {
		return nil, err
	}

	return constructor, nil
}
