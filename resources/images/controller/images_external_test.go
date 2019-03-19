// Copyright 2019 Northern.tech AS
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

package controller_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/mendersoftware/go-lib-micro/requestlog"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/mendersoftware/deployments/resources/images"
	. "github.com/mendersoftware/deployments/resources/images/controller"
	"github.com/mendersoftware/deployments/resources/images/controller/mocks"
	"github.com/mendersoftware/deployments/utils/pointers"
	"github.com/mendersoftware/deployments/utils/restutil/view"
	h "github.com/mendersoftware/deployments/utils/testing"
)

// Notice: 	Controller tests are not pure unit tests,
// 			they are more of integration test beween controller and view
//			testing actuall HTTP endpoint input/reponse

const (
	validUUIDv4  = "d50eda0d-2cea-4de1-8d42-9cd3e7e8670d"
	artifactSize = 10000
)

type routerTypeHandler func(pathExp string, handlerFunc rest.HandlerFunc) *rest.Route

func setUpRestTest(route string, routeType routerTypeHandler, handler func(w rest.ResponseWriter, r *rest.Request)) *rest.Api {
	router, _ := rest.MakeRouter(routeType(route, handler))
	api := rest.NewApi()
	api.Use(
		&requestlog.RequestLogMiddleware{
			BaseLogger: &logrus.Logger{Out: ioutil.Discard},
		},
		&requestid.RequestIdMiddleware{},
	)
	api.SetApp(router)

	return api
}

func TestControllerGetImage(t *testing.T) {
	imagesModel := &mocks.ImagesModel{}
	controller := NewSoftwareImagesController(imagesModel, new(view.RESTView))

	api := setUpRestTest("/api/0.0.1/images/:id", rest.Get, controller.GetImage)

	//no uuid provided
	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images/123", nil))
	recorded.CodeIs(http.StatusBadRequest)

	//have correct id, but no image
	uid, err := uuid.NewV4()
	assert.NoError(t, err)

	id := uid.String()

	imagesModel.On("GetImage", h.ContextMatcher(), id).
		Return(nil, nil)
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images/"+id, nil))
	recorded.CodeIs(http.StatusNotFound)

	//have correct id, but error getting image
	uid, err = uuid.NewV4()
	assert.NoError(t, err)

	id = uid.String()

	imagesModel.On("GetImage", h.ContextMatcher(), id).
		Return(nil, errors.New("error"))
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images/"+id, nil))
	recorded.CodeIs(http.StatusInternalServerError)

	// have image, get OK
	uid, err = uuid.NewV4()
	assert.NoError(t, err)

	id = uid.String()

	imageMeta := images.NewSoftwareImageMetaConstructor()
	imageMetaArtifact := images.NewSoftwareImageMetaArtifactConstructor()
	constructorImage := images.NewSoftwareImage(
		validUUIDv4, imageMeta, imageMetaArtifact, artifactSize)
	imagesModel.On("GetImage", h.ContextMatcher(), id).
		Return(constructorImage, nil)
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images/"+id, nil))
	recorded.CodeIs(http.StatusOK)
	recorded.ContentTypeIsJson()

	var receivedImage images.SoftwareImage
	if err := recorded.DecodeJsonPayload(&receivedImage); err != nil {
		t.FailNow()
	}
}

func TestControllerListImages(t *testing.T) {
	imagesModel := &mocks.ImagesModel{}
	controller := NewSoftwareImagesController(imagesModel, new(view.RESTView))

	api := setUpRestTest("/api/0.0.1/images", rest.Get, controller.ListImages)

	//getting list error
	imagesModel.On("ListImages", h.ContextMatcher(), mock.Anything).
		Return(nil, errors.New("error"))
	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images", nil))
	recorded.CodeIs(http.StatusInternalServerError)

	//getting list OK
	imagesModel = &mocks.ImagesModel{}
	controller = NewSoftwareImagesController(imagesModel, new(view.RESTView))
	api = setUpRestTest("/api/0.0.1/images", rest.Get, controller.ListImages)
	imageMeta := images.NewSoftwareImageMetaConstructor()
	imageMetaArtifact := images.NewSoftwareImageMetaArtifactConstructor()
	constructorImage := images.NewSoftwareImage(
		validUUIDv4, imageMeta, imageMetaArtifact, artifactSize)
	imagesModel.On("ListImages", h.ContextMatcher(), mock.Anything).
		Return([]*images.SoftwareImage{constructorImage}, nil)
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images", nil))
	recorded.CodeIs(http.StatusOK)
	recorded.ContentTypeIsJson()
}

func TestControllerDeleteImage(t *testing.T) {
	imagesModel := &mocks.ImagesModel{}
	controller := NewSoftwareImagesController(imagesModel, new(view.RESTView))

	api := setUpRestTest("/api/0.0.1/images/:id", rest.Delete, controller.DeleteImage)

	imageMeta := images.NewSoftwareImageMetaConstructor()
	imageMetaArtifact := images.NewSoftwareImageMetaArtifactConstructor()
	constructorImage := images.NewSoftwareImage(
		validUUIDv4, imageMeta, imageMetaArtifact, artifactSize)

	// wrong id
	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("DELETE", "http://localhost/api/0.0.1/images/wrong_id", nil))
	recorded.CodeIs(http.StatusBadRequest)

	// valid id; doesn't exist
	uid, err := uuid.NewV4()
	assert.NoError(t, err)

	id := uid.String()

	imagesModel.On("DeleteImage", h.ContextMatcher(), id).
		Return(ErrImageMetaNotFound)
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("DELETE", "http://localhost/api/0.0.1/images/"+id, nil))
	recorded.CodeIs(http.StatusNotFound)

	// valid id; image exists
	uid, err = uuid.NewV4()
	assert.NoError(t, err)

	id = uid.String()

	imagesModel.On("DeleteImage", h.ContextMatcher(), id).Return(nil)
	imagesModel.On("GetImage", h.ContextMatcher(), id).
		Return(constructorImage, nil)

	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("DELETE", "http://localhost/api/0.0.1/images/"+id, nil))
	recorded.CodeIs(http.StatusNoContent)
	recorded.BodyIs("")
}

func TestControllerEditImage(t *testing.T) {
	imagesModel := &mocks.ImagesModel{}
	controller := NewSoftwareImagesController(imagesModel, new(view.RESTView))

	api := setUpRestTest("/api/0.0.1/images/:id", rest.Put, controller.EditImage)

	// wrong id
	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("PUT", "http://localhost/api/0.0.1/images/wrong_id", nil))
	recorded.CodeIs(http.StatusBadRequest)

	// correct id; no payload
	uid, err := uuid.NewV4()
	assert.NoError(t, err)

	id := uid.String()

	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("PUT", "http://localhost/api/0.0.1/images/"+id, nil))
	recorded.CodeIs(http.StatusBadRequest)

	// correct id; correct payload; edit error
	uid, err = uuid.NewV4()
	assert.NoError(t, err)

	id = uid.String()

	imagesModel.On("EditImage", h.ContextMatcher(), id, mock.Anything).
		Return(false, errors.New("error"))
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("PUT", "http://localhost/api/0.0.1/images/"+id,
			map[string]string{"name": "myImage"}))
	recorded.CodeIs(http.StatusInternalServerError)

	// correct id; correct payload; image in use
	uid, err = uuid.NewV4()
	assert.NoError(t, err)

	id = uid.String()

	imagesModel.On("EditImage", h.ContextMatcher(), id, mock.Anything).
		Return(false, ErrModelImageUsedInAnyDeployment)
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("PUT", "http://localhost/api/0.0.1/images/"+id,
			map[string]string{"name": "myImage"}))
	recorded.CodeIs(http.StatusUnprocessableEntity)

	// correct id; correct payload; edit no image
	uid, err = uuid.NewV4()
	assert.NoError(t, err)

	id = uid.String()

	imagesModel.On("EditImage", h.ContextMatcher(), id, mock.Anything).
		Return(false, nil)
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("PUT", "http://localhost/api/0.0.1/images/"+id,
			map[string]string{"name": "myImage"}))
	recorded.CodeIs(http.StatusNotFound)

	// correct id; correct payload; have image
	uid, err = uuid.NewV4()
	assert.NoError(t, err)

	id = uid.String()

	imagesModel.On("EditImage", h.ContextMatcher(), id, mock.Anything).
		Return(true, nil)

	req := test.MakeSimpleRequest("PUT", "http://localhost/api/0.0.1/images/"+id,
		map[string]string{"name": "myImage"})
	req.Header.Add(requestid.RequestIdHeader, "test")
	recorded = test.RunRequest(t, api.MakeHandler(), req)
	recorded.CodeIs(http.StatusNoContent)
	recorded.BodyIs("")
}

func TestSoftwareImagesControllerNewImage(t *testing.T) {
	t.Parallel()

	file := h.CreateValidImageFile()
	imageBody, err := ioutil.ReadAll(file)
	assert.NoError(t, err)
	assert.NotNil(t, imageBody)
	defer os.Remove(file.Name())
	defer file.Close()

	testCases := []struct {
		h.JSONResponseParams

		InputBodyObject []h.Part

		InputContentType string
		InputModelID     string
		InputModelError  error
	}{
		{
			InputBodyObject: nil,
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("mime: no media type")),
			},
		},
		{
			InputBodyObject:  []h.Part{},
			InputContentType: "multipart/form-data",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Request does not contain artifact: EOF")),
			},
		},
		{
			InputBodyObject: []h.Part{
				{
					FieldName:  "size",
					FieldValue: "1",
				},
				{
					FieldName:   "artifact",
					ContentType: "application/octet-stream",
					ImageData:   []byte{0},
				},
			},
			InputContentType: "multipart/form-data",
			InputModelID:     "1234",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusCreated,
				OutputBodyObject: nil,
				OutputHeaders:    map[string]string{"Location": "./r/1234"},
			},
		},
		{
			InputBodyObject: []h.Part{
				{
					FieldName:  "description",
					FieldValue: "dt",
				},
			},
			InputContentType: "multipart/form-data",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Request does not contain artifact: EOF")),
			},
		},
		{
			InputBodyObject: []h.Part{
				{
					FieldName:  "size",
					FieldValue: "123",
				},
				{
					FieldName:  "artifact",
					FieldValue: "ff",
				},
			},
			InputContentType: "multipart/form-data",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("The last part of the multipart/form-data message should be an artifact.")),
			},
		},
		{
			InputBodyObject: []h.Part{
				{
					FieldName:  "size",
					FieldValue: strconv.Itoa(len(imageBody)),
				},
				{
					FieldName:   "artifact",
					ContentType: "application/octet-stream",
					ImageData:   imageBody,
				},
			},
			InputContentType: "multipart/mixed",
			InputModelID:     "1234",
			InputModelError:  errors.New("create image error"),
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusInternalServerError,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("internal error")),
			},
		},
		{
			InputBodyObject: []h.Part{
				{
					FieldName:  "size",
					FieldValue: strconv.Itoa(len(imageBody)),
				},
				{
					FieldName:   "artifact",
					ContentType: "application/octet-stream",
					ImageData:   imageBody,
				},
			},
			InputContentType: "multipart/form-data",
			InputModelID:     "1234",
			InputModelError:  ErrModelArtifactNotUnique,
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusUnprocessableEntity,
				OutputBodyObject: h.ErrorToErrStruct(ErrModelArtifactNotUnique),
			},
		},
		{
			InputBodyObject: []h.Part{
				{
					FieldName:  "size",
					FieldValue: strconv.Itoa(len(imageBody)),
				},
				{
					FieldName:   "artifact",
					ContentType: "application/octet-stream",
					ImageData:   imageBody,
				},
			},
			InputContentType: "multipart/form-data",
			InputModelID:     "1234",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusCreated,
				OutputBodyObject: nil,
				OutputHeaders:    map[string]string{"Location": "./r/1234"},
			},
		},
	}

	for testCaseNumber, testCase := range testCases {

		// Run each test case as individual subtest
		t.Run(fmt.Sprintf("Test case number: %v", testCaseNumber+1), func(t *testing.T) {
			model := &mocks.ImagesModel{}

			model.On("CreateImage", h.ContextMatcher(),
				mock.AnythingOfType("*controller.MultipartUploadMsg")).
				Return(testCase.InputModelID, testCase.InputModelError)

			api := setUpRestTest("/r", rest.Post,
				NewSoftwareImagesController(model, new(view.RESTView)).NewImage)

			req := h.MakeMultipartRequest("POST", "http://localhost/r",
				testCase.InputContentType, testCase.InputBodyObject)
			req.Header.Add(requestid.RequestIdHeader, "test")
			recorded := test.RunRequest(t, api.MakeHandler(), req)

			h.CheckRecordedResponse(t, recorded, testCase.JSONResponseParams)
		})
	}
}

func TestSoftwareImagesControllerDownloadLink(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		h.JSONResponseParams

		InputID          string
		InputParamExpire *string

		InputModelLink  *images.Link
		InputModelError error
	}{
		{
			InputID: "89r89r4y",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(ErrIDNotUUIDv4),
			},
		},
		// expire is ignored
		{
			InputID:          "83241c4b-6281-40dd-b6fa-932633e21baa",
			InputParamExpire: pointers.StringToPointer("1234"),
			InputModelLink:   images.NewLink("http://come.and.get.me", time.Time{}),
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusOK,
				OutputBodyObject: images.NewLink("http://come.and.get.me", time.Time{}),
			},
		},
		{
			InputID:         "83241c4b-6281-40dd-b6fa-932633e21bae",
			InputModelError: errors.New("file service down"),
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusInternalServerError,
				OutputBodyObject: h.ErrorToErrStruct(errors.New(`internal error`)),
			},
		},
		// no file found
		{
			InputID: "83241c4b-6281-40dd-b6fa-932633e21baf",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusNotFound,
				OutputBodyObject: h.ErrorToErrStruct(errors.New(`Resource not found`)),
			},
		},
		{
			InputID:        "83241c4b-6281-40dd-b6fa-932633e21bab",
			InputModelLink: images.NewLink("http://come.and.get.me", time.Time{}),
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusOK,
				OutputBodyObject: images.NewLink("http://come.and.get.me", time.Time{}),
			},
		},
	}

	for _, testCase := range testCases {

		model := &mocks.ImagesModel{}

		model.On("DownloadLink", h.ContextMatcher(),
			testCase.InputID, DefaultDownloadLinkExpire).
			Return(testCase.InputModelLink, testCase.InputModelError)

		api := setUpRestTest("/:id", rest.Post,
			NewSoftwareImagesController(model, new(view.RESTView)).DownloadLink)

		var expire string
		if testCase.InputParamExpire != nil {
			expire = "?expire=" + *testCase.InputParamExpire
		}

		req := test.MakeSimpleRequest("POST",
			fmt.Sprintf("http://localhost/%s%s", testCase.InputID, expire),
			nil)
		req.Header.Add(requestid.RequestIdHeader, "test")
		recorded := test.RunRequest(t, api.MakeHandler(), req)

		h.CheckRecordedResponse(t, recorded, testCase.JSONResponseParams)
	}
}
