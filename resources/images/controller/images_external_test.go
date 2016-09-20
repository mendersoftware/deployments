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

package controller_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"testing"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	"github.com/mendersoftware/deployments/resources/images"
	. "github.com/mendersoftware/deployments/resources/images/controller"
	"github.com/mendersoftware/deployments/resources/images/controller/mocks"
	"github.com/mendersoftware/deployments/resources/images/view"
	"github.com/mendersoftware/deployments/utils/pointers"
	h "github.com/mendersoftware/deployments/utils/testing"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Notice: 	Controller tests are not pure unit tests,
// 			they are more of integration test beween controller and view
//			testing actuall HTTP endpoint input/reponse

type fakeImageModeler struct {
	getImage          *images.SoftwareImage
	getImageError     error
	imagesList        []*images.SoftwareImage
	listImagesError   error
	downloadLink      *images.Link
	downloadLinkError error
	editImage         bool
	editError         error
	deleteError       error
}

type Part struct {
	ContentType string
	ImageData   []byte
	MetaStruct  interface{}
}

func (fim *fakeImageModeler) ListImages(filters map[string]string) ([]*images.SoftwareImage, error) {
	return fim.imagesList, fim.listImagesError
}

func (fim *fakeImageModeler) DownloadLink(imageID string, expire time.Duration) (*images.Link, error) {
	return fim.downloadLink, fim.downloadLinkError
}

func (fim *fakeImageModeler) GetImage(id string) (*images.SoftwareImage, error) {
	return fim.getImage, fim.getImageError
}

func (fim *fakeImageModeler) DeleteImage(imageID string) error {
	return fim.deleteError
}

func (fim *fakeImageModeler) CreateImage(imageFile *os.File, constructorData *images.SoftwareImageConstructor) (string, error) {
	return "", nil
}

func (fim *fakeImageModeler) EditImage(id string, constructorData *images.SoftwareImageConstructor) (bool, error) {
	return fim.editImage, fim.editError
}

type routerTypeHandler func(pathExp string, handlerFunc rest.HandlerFunc) *rest.Route

func setUpRestTest(route string, routeType routerTypeHandler, handler func(w rest.ResponseWriter, r *rest.Request)) *rest.Api {
	router, _ := rest.MakeRouter(routeType(route, handler))
	api := rest.NewApi()
	api.SetApp(router)

	return api
}

func TestControllerGetImage(t *testing.T) {
	imagesModel := new(fakeImageModeler)
	controller := NewSoftwareImagesController(imagesModel, new(view.RESTView))

	api := setUpRestTest("/api/0.0.1/images/:id", rest.Get, controller.GetImage)

	//no uuid provided
	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images/123", nil))
	recorded.CodeIs(http.StatusBadRequest)

	//have correct id, but no image
	id := uuid.NewV4().String()
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images/"+id, nil))
	recorded.CodeIs(http.StatusNotFound)

	//have correct id, but error getting image
	imagesModel.getImageError = errors.New("error")
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images/"+id, nil))
	recorded.CodeIs(http.StatusInternalServerError)

	// have image, get OK
	image := images.NewSoftwareImageConstructor()
	constructorImage := images.NewSoftwareImageFromConstructor(image)
	imagesModel.getImageError = nil
	imagesModel.getImage = constructorImage
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
	imagesModel := new(fakeImageModeler)
	controller := NewSoftwareImagesController(imagesModel, new(view.RESTView))

	api := setUpRestTest("/api/0.0.1/images", rest.Get, controller.ListImages)

	//getting list error
	imagesModel.listImagesError = errors.New("error")
	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images", nil))
	recorded.CodeIs(http.StatusInternalServerError)

	//getting list OK
	imagesModel.listImagesError = nil
	image := images.NewSoftwareImageConstructor()
	constructorImage := images.NewSoftwareImageFromConstructor(image)
	imagesModel.imagesList = append(imagesModel.imagesList, constructorImage)
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images", nil))
	recorded.CodeIs(http.StatusOK)
	recorded.ContentTypeIsJson()
}

func TestControllerDeleteImage(t *testing.T) {
	imagesModel := new(fakeImageModeler)
	controller := NewSoftwareImagesController(imagesModel, new(view.RESTView))

	api := setUpRestTest("/api/0.0.1/images/:id", rest.Delete, controller.DeleteImage)

	image := images.NewSoftwareImageConstructor()
	constructorImage := images.NewSoftwareImageFromConstructor(image)

	// wrong id
	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("DELETE", "http://localhost/api/0.0.1/images/wrong_id", nil))
	recorded.CodeIs(http.StatusBadRequest)

	// valid id; doesn't exist
	id := uuid.NewV4().String()
	imagesModel.deleteError = ErrImageMetaNotFound
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("DELETE", "http://localhost/api/0.0.1/images/"+id, nil))
	recorded.CodeIs(http.StatusNotFound)

	// valid id; image exists
	imagesModel.deleteError = nil
	imagesModel.getImage = constructorImage

	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("DELETE", "http://localhost/api/0.0.1/images/"+id, nil))
	recorded.CodeIs(http.StatusNoContent)
	recorded.BodyIs("")
}

func TestControllerEditImage(t *testing.T) {
	imagesModel := new(fakeImageModeler)
	controller := NewSoftwareImagesController(imagesModel, new(view.RESTView))

	api := setUpRestTest("/api/0.0.1/images/:id", rest.Put, controller.EditImage)

	// wrong id
	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("PUT", "http://localhost/api/0.0.1/images/wrong_id", nil))
	recorded.CodeIs(http.StatusBadRequest)

	// correct id; no payload
	id := uuid.NewV4().String()
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("PUT", "http://localhost/api/0.0.1/images/"+id, nil))
	recorded.CodeIs(http.StatusBadRequest)

	// correct id; invalid payload
	//image := NewSoftwareImageConstructor()
	//constructorImage := NewSoftwareImageFromConstructor(image)
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("PUT", "http://localhost/api/0.0.1/images/"+id, map[string]string{"image": "bad_image"}))
	recorded.CodeIs(http.StatusBadRequest)

	// correct id; correct payload; edit error
	imagesModel.editError = errors.New("error")
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("PUT", "http://localhost/api/0.0.1/images/"+id,
			map[string]string{"yocto_id": "1234-1234", "name": "myImage", "device_type": "myDevice"}))
	recorded.CodeIs(http.StatusInternalServerError)

	// correct id; correct payload; edit no image
	imagesModel.editError = nil
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("PUT", "http://localhost/api/0.0.1/images/"+id,
			map[string]string{"yocto_id": "1234-1234", "name": "myImage", "device_type": "myDevice"}))
	recorded.CodeIs(http.StatusNotFound)

	// correct id; correct payload; have image
	imagesModel.editImage = true
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("PUT", "http://localhost/api/0.0.1/images/"+id,
			map[string]string{"yocto_id": "1234-1234", "name": "myImage", "device_type": "myDevice"}))
	recorded.CodeIs(http.StatusNoContent)
	recorded.BodyIs("")
}

// TODO test mulitpart upload
func TestSoftwareImagesControllerNewImage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		h.JSONResponseParams

		InputBodyObject []Part

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
			InputBodyObject: []Part{
				Part{
					ContentType: "application/json",
				},
			},
			InputContentType: "multipart/form-data",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusUnsupportedMediaType,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Content-Type should be multipart/mixed")),
			},
		},
		{
			InputBodyObject:  []Part{},
			InputContentType: "multipart/mixed",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Request does not contain metadata part: EOF")),
			},
		},
		{
			InputBodyObject: []Part{
				Part{
					ContentType: "application/octet-stream",
				},
			},
			InputContentType: "multipart/mixed",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("First part should be a metadata (application/json)")),
			},
		},
		{
			InputBodyObject: []Part{
				Part{
					ContentType: "application/json",
				},
			},
			InputContentType: "multipart/mixed",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Parsing matadata: unexpected end of JSON input")),
			},
		},
		{
			InputBodyObject: []Part{
				Part{
					ContentType: "application/json",
					MetaStruct:  images.NewSoftwareImageConstructor(),
				},
			},
			InputContentType: "multipart/mixed",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Validating metadata: YoctoId: non zero value required;Name: non zero value required;DeviceType: non zero value required;")),
			},
		},
		{
			InputBodyObject: []Part{
				Part{
					ContentType: "application/json",
					MetaStruct: &images.SoftwareImageConstructor{
						YoctoId:    pointers.StringToPointer("yocto-id"),
						Name:       pointers.StringToPointer("name"),
						DeviceType: pointers.StringToPointer("dev-type"),
					},
				},
			},
			InputContentType: "multipart/mixed",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Request does not contain image part: EOF")),
			},
		},
		{
			InputBodyObject: []Part{
				Part{
					ContentType: "application/json",
					MetaStruct: &images.SoftwareImageConstructor{
						YoctoId:    pointers.StringToPointer("yocto-id"),
						Name:       pointers.StringToPointer("name"),
						DeviceType: pointers.StringToPointer("dev-type"),
					},
				},
				Part{
					ContentType: "application/json",
					MetaStruct: &images.SoftwareImageConstructor{
						YoctoId:    pointers.StringToPointer("yocto-id"),
						Name:       pointers.StringToPointer("name"),
						DeviceType: pointers.StringToPointer("dev-type"),
					},
				},
			},
			InputContentType: "multipart/mixed",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Second part should be an image (octet-stream)")),
			},
		},
		{
			InputBodyObject: []Part{
				Part{
					ContentType: "application/json",
					MetaStruct: &images.SoftwareImageConstructor{
						YoctoId:    pointers.StringToPointer("yocto-id"),
						Name:       pointers.StringToPointer("name"),
						DeviceType: pointers.StringToPointer("dev-type"),
					},
				},
				Part{
					ContentType: "application/octet-stream",
					ImageData:   []byte{0},
				},
			},
			InputContentType: "multipart/mixed",
			InputModelID:     "1234",
			InputModelError:  errors.New("create image error"),
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusInternalServerError,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("create image error")),
			},
		},
		{
			InputBodyObject: []Part{
				Part{
					ContentType: "application/json",
					MetaStruct: &images.SoftwareImageConstructor{
						YoctoId:    pointers.StringToPointer("yocto-id"),
						Name:       pointers.StringToPointer("name"),
						DeviceType: pointers.StringToPointer("dev-type"),
					},
				},
				Part{
					ContentType: "application/octet-stream",
					ImageData:   []byte{0},
				},
			},
			InputContentType: "multipart/mixed",
			InputModelID:     "1234",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusCreated,
				OutputBodyObject: nil,
				OutputHeaders:    map[string]string{"Location": "http://localhost/r/1234"},
			},
		},
	}

	for _, testCase := range testCases {

		model := new(mocks.ImagesModel)

		model.On("CreateImage", mock.AnythingOfType("*os.File"), mock.AnythingOfType("*images.SoftwareImageConstructor")).
			Return(testCase.InputModelID, testCase.InputModelError)

		router, err := rest.MakeRouter(
			rest.Post("/r",
				NewSoftwareImagesController(model, new(view.RESTView)).NewImage))
		assert.NoError(t, err)

		api := rest.NewApi()
		api.SetApp(router)

		recorded := test.RunRequest(t, api.MakeHandler(),
			MakeMultipartRequest("POST", "http://localhost/r", testCase.InputContentType, testCase.InputBodyObject))

		h.CheckRecordedResponse(t, recorded, testCase.JSONResponseParams)
	}
}

// MakeMultipartRequest returns a http.Request.
func MakeMultipartRequest(method string, urlStr string, contentType string, payload []Part) *http.Request {
	body_buf := new(bytes.Buffer)
	body_writer := multipart.NewWriter(body_buf)
	for _, part := range payload {
		mh := make(textproto.MIMEHeader)
		mh.Set("Content-Type", part.ContentType)
		part_writer, err := body_writer.CreatePart(mh)
		if nil != err {
			panic(err.Error())
		}
		if part.ContentType == "application/json" && part.MetaStruct != nil {
			b, err := json.Marshal(part.MetaStruct)
			if err != nil {
				panic(err)
			}
			io.Copy(part_writer, bytes.NewReader(b))
		} else {
			io.Copy(part_writer, bytes.NewReader(part.ImageData))
		}
	}
	body_writer.Close()

	r, err := http.NewRequest(method, urlStr, bytes.NewReader(body_buf.Bytes()))
	if err != nil {
		panic(err)
	}
	r.Header.Set("Accept-Encoding", "gzip")
	if payload != nil {
		r.Header.Set("Content-Type", contentType+";boundary="+body_writer.Boundary())
	}

	return r
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
		{
			InputID:          "83241c4b-6281-40dd-b6fa-932633e21bab",
			InputParamExpire: pointers.StringToPointer("ala ma kota"),
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(ErrInvalidExpireParam),
			},
		},
		{
			InputID:          "83241c4b-6281-40dd-b6fa-932633e21bab",
			InputParamExpire: pointers.StringToPointer("1.1"),
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(ErrInvalidExpireParam),
			},
		},
		{
			InputID:          "83241c4b-6281-40dd-b6fa-932633e21bab",
			InputParamExpire: pointers.StringToPointer("9999999"),
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(ErrInvalidExpireParam),
			},
		},
		{
			InputID:          "83241c4b-6281-40dd-b6fa-932633e21bab",
			InputParamExpire: pointers.StringToPointer("123"),
			InputModelError:  errors.New("file service down"),
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusInternalServerError,
				OutputBodyObject: h.ErrorToErrStruct(errors.New(`file service down`)),
			},
		},
		{
			InputID:         "83241c4b-6281-40dd-b6fa-932633e21bab",
			InputModelError: errors.New("file service down"),
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusInternalServerError,
				OutputBodyObject: h.ErrorToErrStruct(errors.New(`file service down`)),
			},
		},
		// no file found
		{
			InputID: "83241c4b-6281-40dd-b6fa-932633e21bab",
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

		model := new(mocks.ImagesModel)

		model.On("DownloadLink", testCase.InputID, mock.AnythingOfType("time.Duration")).
			Return(testCase.InputModelLink, testCase.InputModelError)

		router, err := rest.MakeRouter(
			rest.Post("/:id",
				NewSoftwareImagesController(model, new(view.RESTView)).DownloadLink))
		assert.NoError(t, err)

		api := rest.NewApi()
		api.SetApp(router)

		var expire string
		if testCase.InputParamExpire != nil {
			expire = "?expire=" + *testCase.InputParamExpire
		}

		recorded := test.RunRequest(t, api.MakeHandler(),
			test.MakeSimpleRequest("POST",
				fmt.Sprintf("http://localhost/%s%s", testCase.InputID, expire),
				nil))

		h.CheckRecordedResponse(t, recorded, testCase.JSONResponseParams)
	}
}
