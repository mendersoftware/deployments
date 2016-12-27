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
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/asaskevich/govalidator"
	"github.com/mendersoftware/artifacts/metadata"
	"github.com/mendersoftware/artifacts/parser"
	"github.com/mendersoftware/artifacts/reader"
	"github.com/mendersoftware/deployments/resources/images"
	"github.com/mendersoftware/go-lib-micro/requestlog"
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
	l := requestlog.GetRequestLogger(r.Env)

	id := r.PathParam("id")

	if !govalidator.IsUUIDv4(id) {
		s.view.RenderError(w, r, ErrIDNotUUIDv4, http.StatusBadRequest, l)
		return
	}

	image, err := s.model.GetImage(id)
	if err != nil {
		s.view.RenderInternalError(w, r, err, l)
		return
	}

	if image == nil {
		s.view.RenderErrorNotFound(w, r, l)
		return
	}

	s.view.RenderSuccessGet(w, image)
}

func (s *SoftwareImagesController) ListImages(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r.Env)

	list, err := s.model.ListImages(r.PathParams)
	if err != nil {
		s.view.RenderInternalError(w, r, err, l)
		return
	}

	s.view.RenderSuccessGet(w, list)
}

func (s *SoftwareImagesController) DownloadLink(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r.Env)

	id := r.PathParam("id")

	if !govalidator.IsUUIDv4(id) {
		s.view.RenderError(w, r, ErrIDNotUUIDv4, http.StatusBadRequest, l)
		return
	}

	expire, err := s.getLinkExpireParam(r, DefaultDownloadLinkExpire)
	if err != nil {
		s.view.RenderError(w, r, err, http.StatusBadRequest, l)
		return
	}

	link, err := s.model.DownloadLink(id, expire)
	if err != nil {
		s.view.RenderInternalError(w, r, err, l)
		return
	}

	if link == nil {
		s.view.RenderErrorNotFound(w, r, l)
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
	l := requestlog.GetRequestLogger(r.Env)

	id := r.PathParam("id")

	if !govalidator.IsUUIDv4(id) {
		s.view.RenderError(w, r, ErrIDNotUUIDv4, http.StatusBadRequest, l)
		return
	}

	if err := s.model.DeleteImage(id); err != nil {
		if err == ErrImageMetaNotFound {
			s.view.RenderErrorNotFound(w, r, l)
			return
		}
		s.view.RenderInternalError(w, r, err, l)
		return
	}

	s.view.RenderSuccessDelete(w)
}

func (s *SoftwareImagesController) EditImage(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r.Env)

	id := r.PathParam("id")

	if !govalidator.IsUUIDv4(id) {
		s.view.RenderError(w, r, ErrIDNotUUIDv4, http.StatusBadRequest, l)
		return
	}

	constructor, err := s.getSoftwareImageMetaConstructorFromBody(r)
	if err != nil {
		s.view.RenderError(w, r, errors.Wrap(err, "Validating request body"), http.StatusBadRequest, l)
		return
	}

	found, err := s.model.EditImage(id, constructor)
	if err != nil {
		s.view.RenderInternalError(w, r, err, l)
		return
	}

	if !found {
		s.view.RenderErrorNotFound(w, r, l)
		return
	}

	s.view.RenderSuccessPut(w)
}

// Multipart Image/Meta upload handler.
// Request should be of type "multipart/form-data".
// First part should contain Metadata file. This file should be of type "application/json".
// Second part should contain Image file. This part should be of type "application/octet-strem".
func (s *SoftwareImagesController) NewImage(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r.Env)

	// limits just for safety;
	const (
		// Max image size
		DefaultMaxImageSize = 1024 * 1024 * 1024 * 10
		// Max meta size
		DefaultMaxMetaSize = 1024 * 1024 * 10
	)

	// parse content type and params according to RFC 1521
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		s.view.RenderError(w, r, err, http.StatusBadRequest, l)
		return
	}

	mr := multipart.NewReader(r.Body, params["boundary"])

	metaConstructor, imagePart, err := s.handleMeta(mr, DefaultMaxMetaSize)
	if err != nil || imagePart == nil {
		s.view.RenderError(w, r, err, http.StatusBadRequest, l)
		return
	}

	reader, metaArtifactConstructor, status, err := s.handleImage(imagePart, DefaultMaxImageSize)
	if err != nil {
		if status == http.StatusInternalServerError {
			s.view.RenderInternalError(w, r, err, l)
		} else {
			s.view.RenderError(w, r, err, status, l)
		}
		return
	}
	imgId, err := s.model.CreateImage(reader, metaConstructor, metaArtifactConstructor)
	switch err {
	default:
		s.view.RenderInternalError(w, r, err, l)
	case nil:
		s.view.RenderSuccessPost(w, r, imgId)
	case ErrModelArtifactNotUnique:
		s.view.RenderError(w, r, err, http.StatusUnprocessableEntity, l)
	case ErrModelMissingInputMetadata, ErrModelInvalidMetadata:
		s.view.RenderError(w, r, err, http.StatusBadRequest, l)
	}

	return
}

// Meta part of multipart meta/image request handler.
// Parses meta body, returns image meta constructor, reader to image part of the multipart message and nil on success.
func (s *SoftwareImagesController) handleMeta(mr *multipart.Reader, maxMetaSize int64) (*images.SoftwareImageMetaConstructor, *multipart.Part, error) {
	constructor := &images.SoftwareImageMetaConstructor{}
	for {
		p, err := mr.NextPart()
		if err != nil {
			return nil, nil, errors.Wrap(err, "Request does not contain artifact")
		}
		switch p.FormName() {
		case "name":
			name, err := s.getFormFieldValue(p, maxMetaSize)
			if err != nil {
				return nil, nil, err
			}
			constructor.Name = *name
		case "description":
			desc, err := s.getFormFieldValue(p, maxMetaSize)
			if err != nil {
				return nil, nil, err
			}
			constructor.Description = *desc
		case "artifact":
			if err := constructor.Validate(); err != nil {
				return nil, nil, errors.Wrap(err, "Validating metadata")
			}
			return constructor, p, nil
		}
	}
}

// Image part of multipart meta/image request handler.
// Saves uploaded image in temporary file.
// Returns temporary file, image metadata, success code and nil on success.
func (s *SoftwareImagesController) handleImage(
	p *multipart.Part, maxImageSize int64) (io.Reader, *images.SoftwareImageMetaArtifactConstructor, int, error) {
	// HTML form can't set specific content-type, it's automatic, if not empty - it's a file
	if p.Header.Get("Content-Type") == "" {
		return nil, nil, http.StatusBadRequest, errors.New("Last part should be an image")
	}
	var metaData bytes.Buffer
	metaWriter := bufio.NewWriter(&metaData)

	lr := io.LimitReader(p, maxImageSize)
	tee := io.TeeReader(lr, metaWriter)

	meta, err := s.getMetaFromArchive(&tee, maxImageSize)
	if err != nil {
		return nil, nil, http.StatusBadRequest, err
	}
	metaWriter.Flush()
	metaReader := bufio.NewReader(&metaData)
	multiReader := io.MultiReader(metaReader, lr)

	return multiReader, meta, http.StatusOK, nil
}

func getArtifactInfo(info metadata.Info) *images.ArtifactInfo {
	return &images.ArtifactInfo{
		Format:  info.Format,
		Version: uint(info.Version),
	}
}

func getUpdateFiles(maxImageSize int64, uFiles map[string]parser.UpdateFile) ([]images.UpdateFile, error) {
	var files []images.UpdateFile
	for _, u := range uFiles {
		if u.Size > maxImageSize {
			return nil, errors.New("Image too large")
		}
		files = append(files, images.UpdateFile{
			Name:      u.Name,
			Size:      u.Size,
			Signature: string(u.Signature),
			Date:      &u.Date,
			Checksum:  string(u.Checksum),
		})
	}
	return files, nil
}

func (s *SoftwareImagesController) getMetaFromArchive(
	r *io.Reader, maxImageSize int64) (*images.SoftwareImageMetaArtifactConstructor, error) {
	metaArtifact := images.NewSoftwareImageMetaArtifactConstructor()

	aReader := areader.NewReader(*r)
	defer aReader.Close()

	data, err := aReader.Read()
	if err != nil {
		return nil, errors.Wrap(err, "reading artifact error")
	}
	metaArtifact.Info = getArtifactInfo(aReader.GetInfo())
	metaArtifact.DeviceTypesCompatible = aReader.GetCompatibleDevices()
	metaArtifact.ArtifactName = aReader.GetArtifactName()

	for _, p := range data {
		uFiles, err := getUpdateFiles(maxImageSize, p.GetUpdateFiles())
		if err != nil {
			return nil, errors.Wrap(err, "Cannot get update files:")
		}

		metaArtifact.Updates = append(
			metaArtifact.Updates,
			images.Update{
				TypeInfo: images.ArtifactUpdateTypeInfo{
					Type: p.GetUpdateType().Type,
				},
				MetaData: p.GetMetadata(),
				Files:    uFiles,
			})
	}

	return metaArtifact, nil
}

func (s *SoftwareImagesController) getFormFieldValue(p *multipart.Part, maxMetaSize int64) (*string, error) {
	metaReader := io.LimitReader(p, maxMetaSize)
	bytes, err := ioutil.ReadAll(metaReader)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "Failed to obtain value for "+p.FormName())
	}

	strValue := string(bytes)
	return &strValue, nil
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
