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
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path"
	"testing"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	"github.com/mendersoftware/artifacts/parser"
	atutils "github.com/mendersoftware/artifacts/test_utils"
	"github.com/mendersoftware/artifacts/writer"
	"github.com/mendersoftware/deployments/resources/images"
	. "github.com/mendersoftware/deployments/resources/images/controller"
	"github.com/mendersoftware/deployments/resources/images/controller/mocks"
	"github.com/mendersoftware/deployments/resources/images/view"
	"github.com/mendersoftware/deployments/utils/pointers"
	h "github.com/mendersoftware/deployments/utils/testing"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/mendersoftware/go-lib-micro/requestlog"
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
	FieldName   string
	FieldValue  string
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

func (fim *fakeImageModeler) CreateImage(
	imageFile *os.File,
	metaConstructor *images.SoftwareImageMetaConstructor,
	metaYoctoConstructor *images.SoftwareImageMetaYoctoConstructor) (string, error) {
	return "", nil
}

func (fim *fakeImageModeler) EditImage(id string, metaConstructor *images.SoftwareImageMetaConstructor) (bool, error) {
	return fim.editImage, fim.editError
}

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
	imageMeta := images.NewSoftwareImageMetaConstructor()
	imageMetaYocto := images.NewSoftwareImageMetaYoctoConstructor()
	constructorImage := images.NewSoftwareImage(imageMeta, imageMetaYocto)
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
	imageMeta := images.NewSoftwareImageMetaConstructor()
	imageMetaYocto := images.NewSoftwareImageMetaYoctoConstructor()
	constructorImage := images.NewSoftwareImage(imageMeta, imageMetaYocto)
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

	imageMeta := images.NewSoftwareImageMetaConstructor()
	imageMetaYocto := images.NewSoftwareImageMetaYoctoConstructor()
	constructorImage := images.NewSoftwareImage(imageMeta, imageMetaYocto)

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

	req := test.MakeSimpleRequest("PUT", "http://localhost/api/0.0.1/images/"+id,
		map[string]string{"yocto_id": "1234-1234", "name": "myImage", "device_type": "myDevice"})
	req.Header.Add(requestid.RequestIdHeader, "test")
	recorded = test.RunRequest(t, api.MakeHandler(), req)
	recorded.CodeIs(http.StatusNoContent)
	recorded.BodyIs("")
}

func TestSoftwareImagesControllerNewImage(t *testing.T) {
	t.Parallel()

	// create temp dir
	td, _ := ioutil.TempDir("", "mender-install-update-")
	defer os.RemoveAll(td)
	// make a fake update artifact
	upath, err := makeFakeUpdate(t, path.Join(td, "update-root"), true)
	// open archive file
	imageBody, err := ioutil.ReadFile(upath)
	assert.NoError(t, err)
	assert.NotNil(t, imageBody)

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
			InputBodyObject:  []Part{},
			InputContentType: "multipart/form-data",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Request does not contain firmware part: EOF")),
			},
		},
		{
			InputBodyObject: []Part{
				Part{
					FieldName:   "firmware",
					ContentType: "application/octet-stream",
				},
			},
			InputContentType: "multipart/form-data",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Validating metadata: Name: non zero value required;")),
			},
		},
		{
			InputBodyObject: []Part{
				Part{
					FieldName:   "firmware",
					ContentType: "application/octet-stream",
					ImageData:   []byte{0},
				},
			},
			InputContentType: "multipart/form-data",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Validating metadata: Name: non zero value required;")),
			},
		},
		{
			InputBodyObject: []Part{
				Part{
					FieldName:  "name",
					FieldValue: "n",
				},
				Part{
					FieldName:  "device_type",
					FieldValue: "dt",
				},
				Part{
					FieldName:  "yocto_id",
					FieldValue: "yi",
				},
			},
			InputContentType: "multipart/form-data",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Request does not contain firmware part: EOF")),
			},
		},
		{
			InputBodyObject: []Part{
				Part{
					FieldName:  "name",
					FieldValue: "n",
				},
				Part{
					FieldName:  "device_type",
					FieldValue: "dt",
				},
				Part{
					FieldName:  "yocto_id",
					FieldValue: "yi",
				},
				Part{
					FieldName:  "firmware",
					FieldValue: "ff",
				},
			},
			InputContentType: "multipart/form-data",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Last part should be an image")),
			},
		},
		{
			InputBodyObject: []Part{
				Part{
					FieldName:  "name",
					FieldValue: "n",
				},
				Part{
					FieldName:  "device_type",
					FieldValue: "dt",
				},
				Part{
					FieldName:  "yocto_id",
					FieldValue: "yi",
				},
				Part{
					FieldName:   "firmware",
					ContentType: "application/octet-stream",
					ImageData:   []byte{0},
				},
			},
			InputContentType: "multipart/form-data",
			InputModelID:     "1234",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("info error: reader: error reading archive")),
			},
		},
		{
			InputBodyObject: []Part{
				Part{
					FieldName:  "name",
					FieldValue: "n",
				},
				Part{
					FieldName:   "firmware",
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
			InputBodyObject: []Part{
				Part{
					FieldName:  "name",
					FieldValue: "n",
				},
				Part{
					FieldName:   "firmware",
					ContentType: "application/octet-stream",
					ImageData:   imageBody,
				},
			},
			InputContentType: "multipart/form-data",
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

		model.On(
			"CreateImage",
			mock.AnythingOfType("*os.File"),
			mock.AnythingOfType("*images.SoftwareImageMetaConstructor"),
			mock.AnythingOfType("*images.SoftwareImageMetaYoctoConstructor")).
			Return(testCase.InputModelID, testCase.InputModelError)

		api := setUpRestTest("/r", rest.Post, NewSoftwareImagesController(model, new(view.RESTView)).NewImage)

		req := MakeMultipartRequest("POST", "http://localhost/r", testCase.InputContentType, testCase.InputBodyObject)
		req.Header.Add(requestid.RequestIdHeader, "test")
		recorded := test.RunRequest(t, api.MakeHandler(), req)

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
		if part.ContentType == "" && part.ImageData == nil {
			mh.Set("Content-Disposition", "form-data; name=\""+part.FieldName+"\"")
		} else {
			mh.Set("Content-Disposition", "form-data; name=\""+part.FieldName+"\"; filename=\"firmware-213.tar.gz\"")
		}
		part_writer, err := body_writer.CreatePart(mh)
		if nil != err {
			panic(err.Error())
		}
		if part.ContentType == "" && part.ImageData == nil {
			b := []byte(part.FieldValue)
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

func makeFakeUpdate(t *testing.T, root string, valid bool) (string, error) {

	var dirStructOK = []atutils.TestDirEntry{
		{Path: "0000", IsDir: true},
		{Path: "0000/data", IsDir: true},
		{Path: "0000/data/update.ext4", Content: []byte("my first update")},
		{Path: "0000/type-info",
			Content: []byte(`{"type": "rootfs-image"}`),
		},
		{Path: "0000/meta-data",
			Content: []byte(`{"DeviceType": "vexpress-qemu", "ImageID": "core-image-minimal-201608110900"}`),
		},
		{Path: "0000/signatures", IsDir: true},
		{Path: "0000/signatures/update.sig"},
		{Path: "0000/scripts", IsDir: true},
		{Path: "0000/scripts/pre", IsDir: true},
		{Path: "0000/scripts/pre/my_script", Content: []byte("my first script")},
		{Path: "0000/scripts/post", IsDir: true},
		{Path: "0000/scripts/check", IsDir: true},
	}

	err := atutils.MakeFakeUpdateDir(root, dirStructOK)
	assert.NoError(t, err)

	aw := awriter.NewWriter("mender", 1)
	defer aw.Close()

	rp := &parser.RootfsParser{}
	aw.Register(rp)

	upath := path.Join(root, "update.tar")
	err = aw.Write(root, upath)
	assert.NoError(t, err)

	return upath, nil
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
				OutputBodyObject: h.ErrorToErrStruct(errors.New(`internal error`)),
			},
		},
		{
			InputID:         "83241c4b-6281-40dd-b6fa-932633e21bab",
			InputModelError: errors.New("file service down"),
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusInternalServerError,
				OutputBodyObject: h.ErrorToErrStruct(errors.New(`internal error`)),
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

		api := setUpRestTest("/:id", rest.Post, NewSoftwareImagesController(model, new(view.RESTView)).DownloadLink)

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
