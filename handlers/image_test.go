package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"runtime"
	"testing"
	"time"

	"github.com/mendersoftware/services/Godeps/_workspace/src/github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/services/Godeps/_workspace/src/github.com/ant0ine/go-json-rest/rest/test"
	"github.com/mendersoftware/services/controllers"
	"github.com/mendersoftware/services/models/images"
	"github.com/mendersoftware/services/models/users"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

type SetUserMiddleware struct{}

func (mw *SetUserMiddleware) MiddlewareFunc(handler rest.HandlerFunc) rest.HandlerFunc {
	return func(writer rest.ResponseWriter, request *rest.Request) {
		request.Env["REMOTE_USER"] = "admin"
		handler(writer, request)
	}
}

func TestImageMetaLookup(t *testing.T) {

	mock := &ImageControllerMock{
		mockLookup: func(user users.UserI) ([]*images.ImageMeta, error) {
			return make([]*images.ImageMeta, 0), nil
		},
	}

	router, err := rest.MakeRouter(rest.Get("/r", NewImageMeta(mock).Lookup))
	if err != nil {
		t.FailNow()
	}

	api := rest.NewApi()
	api.Use(&SetUserMiddleware{})
	api.SetApp(router)

	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://1.2.3.4/r", nil))

	recorded.CodeIs(http.StatusOK)
	recorded.ContentTypeIsJson()
	recorded.BodyIs("[]")
}

func TestImageMetaLookupError(t *testing.T) {

	apiErr := errors.New("My dummy error")

	mock := &ImageControllerMock{
		mockLookup: func(user users.UserI) ([]*images.ImageMeta, error) {
			return nil, apiErr
		},
	}

	router, err := rest.MakeRouter(rest.Get("/r", NewImageMeta(mock).Lookup))
	if err != nil {
		t.FailNow()
	}

	api := rest.NewApi()
	api.Use(&SetUserMiddleware{})
	api.SetApp(router)

	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://1.2.3.4/r", nil))

	recorded.CodeIs(http.StatusInternalServerError)
	recorded.ContentTypeIsJson()
	recorded.BodyIs(RestErrorMsg(apiErr))
}

func TestImageMetaGetNotFound(t *testing.T) {

	mock := &ImageControllerMock{
		mockGet: func(user users.UserI, id string) (*images.ImageMeta, error) {
			return nil, controllers.ErrNotFound
		},
	}

	router, err := rest.MakeRouter(rest.Get("/r/:id", NewImageMeta(mock).Get))
	if err != nil {
		t.FailNow()
	}

	api := rest.NewApi()
	api.Use(&SetUserMiddleware{})
	api.SetApp(router)

	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://1.2.3.4/r/test", nil))

	recorded.CodeIs(http.StatusNotFound)
	recorded.ContentTypeIsJson()
	recorded.BodyIs(RestErrorMsg(controllers.ErrNotFound))
}

func TestImageMetaGet(t *testing.T) {

	img := images.NewImageMetaFromPublic(nil)
	now := time.Now()
	img.LastUpdated = now
	j, _ := json.Marshal(img)

	mock := &ImageControllerMock{
		mockGet: func(user users.UserI, id string) (*images.ImageMeta, error) {
			return img, nil
		},
	}

	router, err := rest.MakeRouter(rest.Get("/r/:id", NewImageMeta(mock).Get))
	if err != nil {
		t.FailNow()
	}

	api := rest.NewApi()
	api.Use(&SetUserMiddleware{})
	api.SetApp(router)

	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://1.2.3.4/r/test", nil))

	recorded.CodeIs(http.StatusOK)
	recorded.ContentTypeIsJson()
	recorded.BodyIs(string(j))
	recorded.HeaderIs("Last-Modified", now.UTC().Format(http.TimeFormat))
}

func TestImageMetaCreate(t *testing.T) {

	id := "123-dummy-id-123"
	now := time.Now()

	img := images.NewImageMetaFromPublic(images.NewImageMetaPublic("My name"))
	img.LastUpdated = now
	img.Id = id
	j, _ := json.Marshal(img)

	mock := &ImageControllerMock{
		mockCreate: func(user users.UserI, public *images.ImageMetaPublic) (*images.ImageMeta, error) {
			i := images.NewImageMetaFromPublic(public)
			i.LastUpdated = now
			i.Id = id

			return i, nil
		},
	}

	router, err := rest.MakeRouter(rest.Post("/r/", NewImageMeta(mock).Create))
	if err != nil {
		t.FailNow()
	}

	api := rest.NewApi()
	api.Use(&SetUserMiddleware{})
	api.SetApp(router)

	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("POST", "http://1.2.3.4/r/", img))

	recorded.CodeIs(http.StatusCreated)
	recorded.ContentTypeIsJson()
	recorded.HeaderIs("Location", "/api/0.0.1/images/"+id)
	recorded.BodyIs(string(j))
}

func TestImageMetaCreateInvalidAttr(t *testing.T) {

	img := images.NewImageMetaFromPublic(images.NewImageMetaPublic(""))

	mock := &ImageControllerMock{
		mockCreate: func(user users.UserI, public *images.ImageMetaPublic) (*images.ImageMeta, error) {
			return images.NewImageMetaFromPublic(public), nil
		},
	}

	router, err := rest.MakeRouter(rest.Post("/r/", NewImageMeta(mock).Create))
	if err != nil {
		t.FailNow()
	}

	api := rest.NewApi()
	api.Use(&SetUserMiddleware{})
	api.SetApp(router)

	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("POST", "http://1.2.3.4/r/", img))

	recorded.CodeIs(http.StatusBadRequest)
	recorded.ContentTypeIsJson()
	recorded.BodyIs(RestErrorMsg(images.ErrMissingImageAttrName))
}

func TestImageMetaCreateInvalidPayload(t *testing.T) {

	mock := &ImageControllerMock{
		mockCreate: func(user users.UserI, public *images.ImageMetaPublic) (*images.ImageMeta, error) {
			return images.NewImageMetaFromPublic(public), nil
		},
	}

	router, err := rest.MakeRouter(rest.Post("/r/", NewImageMeta(mock).Create))
	if err != nil {
		t.FailNow()
	}

	api := rest.NewApi()
	api.Use(&SetUserMiddleware{})
	api.SetApp(router)

	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("POST", "http://1.2.3.4/r/", "ala ma kota"))

	recorded.CodeIs(http.StatusBadRequest)
	recorded.ContentTypeIsJson()
	recorded.BodyIs(RestErrorMsg(
		errors.New("json: cannot unmarshal string into Go value of type images.ImageMetaPublic")))
}

func TestImageMetaCreateControlerError(t *testing.T) {

	apiError := errors.New("Dummy error")
	img := images.NewImageMetaFromPublic(images.NewImageMetaPublic("Name"))

	mock := &ImageControllerMock{
		mockCreate: func(user users.UserI, public *images.ImageMetaPublic) (*images.ImageMeta, error) {
			return nil, apiError
		},
	}

	router, err := rest.MakeRouter(rest.Post("/r/", NewImageMeta(mock).Create))
	if err != nil {
		t.FailNow()
	}

	api := rest.NewApi()
	api.Use(&SetUserMiddleware{})
	api.SetApp(router)

	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("POST", "http://1.2.3.4/r/", img))

	recorded.CodeIs(http.StatusInternalServerError)
	recorded.ContentTypeIsJson()
	recorded.BodyIs(RestErrorMsg(apiError))
}
