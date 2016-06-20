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
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	"github.com/mendersoftware/deployments/mvc"
	"github.com/satori/go.uuid"
)

type fakeImageModeler struct {
	getImage          *SoftwareImage
	getImageError     error
	imagesList        []*SoftwareImage
	listImagesError   error
	uploadLink        *Link
	uploadLinkError   error
	downloadLink      *Link
	downloadLinkError error
	editImage         bool
	editError         error
	deleteError       error
}

func (fim *fakeImageModeler) ListImages(filters map[string]string) ([]*SoftwareImage, error) {
	return fim.imagesList, fim.listImagesError
}

func (fim *fakeImageModeler) UploadLink(imageID string, expire time.Duration) (*Link, error) {
	return fim.uploadLink, fim.uploadLinkError
}

func (fim *fakeImageModeler) DownloadLink(imageID string, expire time.Duration) (*Link, error) {
	return fim.downloadLink, fim.downloadLinkError
}

func (fim *fakeImageModeler) GetImage(id string) (*SoftwareImage, error) {
	return fim.getImage, fim.getImageError
}

func (fim *fakeImageModeler) DeleteImage(imageID string) error {
	return fim.deleteError
}

func (fim *fakeImageModeler) CreateImage(constructorData *SoftwareImageConstructor) (string, error) {
	return "", nil
}

func (fim *fakeImageModeler) EditImage(id string, constructorData *SoftwareImageConstructor) (bool, error) {
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
	controller := NewSoftwareImagesController(imagesModel, mvc.RESTViewDefaults{})

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
	image := NewSoftwareImageConstructor()
	constructorImage := NewSoftwareImageFromConstructor(image)
	imagesModel.getImageError = nil
	imagesModel.getImage = constructorImage
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images/"+id, nil))
	recorded.CodeIs(http.StatusOK)
	recorded.ContentTypeIsJson()

	var receivedImage SoftwareImage
	if err := recorded.DecodeJsonPayload(&receivedImage); err != nil {
		t.FailNow()
	}
}

func TestControllerListImages(t *testing.T) {
	imagesModel := new(fakeImageModeler)
	controller := NewSoftwareImagesController(imagesModel, mvc.RESTViewDefaults{})

	api := setUpRestTest("/api/0.0.1/images", rest.Get, controller.ListImages)

	//getting list error
	imagesModel.listImagesError = errors.New("error")
	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images", nil))
	recorded.CodeIs(http.StatusInternalServerError)

	//getting list OK
	imagesModel.listImagesError = nil
	image := NewSoftwareImageConstructor()
	constructorImage := NewSoftwareImageFromConstructor(image)
	imagesModel.imagesList = append(imagesModel.imagesList, constructorImage)
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images", nil))
	recorded.CodeIs(http.StatusOK)
	recorded.ContentTypeIsJson()
}

func TestControllerUploadLink(t *testing.T) {
	imagesModel := new(fakeImageModeler)
	controller := NewSoftwareImagesController(imagesModel, mvc.RESTViewDefaults{})

	api := setUpRestTest("/api/0.0.1/images/:id/upload", rest.Get, controller.UploadLink)

	// wrong id
	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images/wrong_id/upload", nil))
	recorded.CodeIs(http.StatusBadRequest)

	// correct id; no upload link
	id := uuid.NewV4().String()
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images/"+id+"/upload", nil))
	recorded.CodeIs(http.StatusNotFound)

	// correct id; error generating link
	imagesModel.uploadLinkError = errors.New("error")
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images/"+id+"/upload", nil))
	recorded.CodeIs(http.StatusInternalServerError)

	// upload link OK
	imagesModel.uploadLinkError = nil
	link := NewLink("uri", time.Now())
	imagesModel.uploadLink = link
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images/"+id+"/upload", nil))
	recorded.CodeIs(http.StatusOK)
	recorded.ContentTypeIsJson()

	var receivedLink Link
	// check if returned body has Link structure
	if err := recorded.DecodeJsonPayload(&receivedLink); err != nil {
		t.FailNow()
	}
	// check if 'uri' is set correctly
	if receivedLink.Uri != "uri" {
		t.FailNow()
	}

}

func TestControllerDownloadLink(t *testing.T) {
	imagesModel := new(fakeImageModeler)
	controller := NewSoftwareImagesController(imagesModel, mvc.RESTViewDefaults{})

	api := setUpRestTest("/api/0.0.1/images/:id/download", rest.Get, controller.DownloadLink)

	// wrong id
	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images/wrong_id/download", nil))
	recorded.CodeIs(http.StatusBadRequest)

	// correct id; no upload link
	id := uuid.NewV4().String()
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images/"+id+"/download", nil))
	recorded.CodeIs(http.StatusNotFound)

	// correct id; error generating link
	imagesModel.downloadLinkError = errors.New("error")
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images/"+id+"/download", nil))
	recorded.CodeIs(http.StatusInternalServerError)

	// download link OK
	imagesModel.downloadLinkError = nil
	time := time.Now()
	link := NewLink("uri", time)
	imagesModel.downloadLink = link
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images/"+id+"/download", nil))
	recorded.CodeIs(http.StatusOK)
	recorded.ContentTypeIsJson()

	var receivedLink Link
	// check if returned body has Link structure
	if err := recorded.DecodeJsonPayload(&receivedLink); err != nil {
		t.FailNow()
	}
	// check if 'uri' is set correctly
	if receivedLink.Uri != "uri" {
		t.FailNow()
	}
}

func TestControllerDeleteImage(t *testing.T) {
	imagesModel := new(fakeImageModeler)
	controller := NewSoftwareImagesController(imagesModel, mvc.RESTViewDefaults{})

	api := setUpRestTest("/api/0.0.1/images/:id", rest.Delete, controller.DeleteImage)

	image := NewSoftwareImageConstructor()
	constructorImage := NewSoftwareImageFromConstructor(image)

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
	controller := NewSoftwareImagesController(imagesModel, mvc.RESTViewDefaults{})

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
