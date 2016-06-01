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
	"github.com/mendersoftware/artifacts/mvc"
	"github.com/satori/go.uuid"
)

type fakeImageModeler struct {
	getImage        *SoftwareImage
	getImageError   error
	imagesList      []*SoftwareImage
	listImagesError error
}

func (fim *fakeImageModeler) ListImages(filters map[string]string) ([]*SoftwareImage, error) {
	return fim.imagesList, fim.listImagesError
}

func (fim *fakeImageModeler) UploadLink(imageID string, expire time.Duration) (*Link, error) {
	return nil, nil
}

func (fim *fakeImageModeler) DownloadLink(imageID string, expire time.Duration) (*Link, error) {
	return nil, nil
}

func (fim *fakeImageModeler) GetImage(id string) (*SoftwareImage, error) {
	return fim.getImage, fim.getImageError
}

func (fim *fakeImageModeler) DeleteImage(imageID string) error {
	return nil
}

func (fim *fakeImageModeler) CreateImage(constructorData *SoftwareImageConstructor) (string, error) {
	return "", nil
}

func (fim *fakeImageModeler) EditImage(id string, constructorData *SoftwareImageConstructor) (bool, error) {
	return true, nil
}

func setUpRestTest(route string, handler func(w rest.ResponseWriter, r *rest.Request)) *rest.Api {
	router, _ := rest.MakeRouter(rest.Get(route, handler))
	api := rest.NewApi()
	api.SetApp(router)

	return api
}

func TestControllerGetImage(t *testing.T) {
	imagesModel := new(fakeImageModeler)
	controller := NewSoftwareImagesController(imagesModel, mvc.RESTViewDefaults{})

	api := setUpRestTest("/api/0.0.1/images/:id", controller.GetImage)

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
}

func TestControllerListImages(t *testing.T) {
	imagesModel := new(fakeImageModeler)
	controller := NewSoftwareImagesController(imagesModel, mvc.RESTViewDefaults{})

	api := setUpRestTest("/api/0.0.1/images", controller.ListImages)

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

	api := setUpRestTest("/api/0.0.1/images/:id/upload", controller.UploadLink)

	// wrong id
	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images/wrong_id/upload", nil))
	recorded.CodeIs(http.StatusBadRequest)

	// correct id
	id := uuid.NewV4().String()
	recorded = test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/images/"+id+"/upload", nil))
	recorded.CodeIs(http.StatusNotFound)

}
