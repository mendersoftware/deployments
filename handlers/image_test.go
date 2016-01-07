package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"runtime"
	"testing"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	"github.com/mendersoftware/artifacts/models/images"
	"github.com/mendersoftware/artifacts/models/users"
	"github.com/mendersoftware/services/controllers"
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

	testList := []struct {
		responseCode int
		body         string
		results      []*images.ImageMeta
		err          error
	}{
		{http.StatusOK, "[]", make([]*images.ImageMeta, 0), nil},
		{http.StatusInternalServerError, RestErrorMsg(errors.New("My dummy error")), nil, errors.New("My dummy error")},
	}

	for _, testCase := range testList {
		mock := &ImageControllerMock{
			mockLookup: func(user users.UserI) ([]*images.ImageMeta, error) {
				return testCase.results, testCase.err
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

		recorded.CodeIs(testCase.responseCode)
		recorded.ContentTypeIsJson()
		recorded.BodyIs(testCase.body)
	}
}

func TestImageMetaGet(t *testing.T) {

	makePublicImage := func(setTime time.Time, public *images.ImageMetaPublic) *images.ImageMeta {
		img := images.NewImageMetaFromPublic(public)
		img.LastUpdated = setTime
		return img
	}

	imageToJson := func(img *images.ImageMeta) string {
		j, _ := json.Marshal(img)
		return string(j)
	}

	testList := []struct {
		responseCode int
		body         string
		image        *images.ImageMeta
		err          error
		time         time.Time
	}{
		{
			http.StatusNotFound,
			RestErrorMsg(controllers.ErrNotFound),
			nil,
			controllers.ErrNotFound,
			time.Time{},
		},
		{
			http.StatusOK,
			imageToJson(makePublicImage(time.Unix(12345, 0), nil)),
			makePublicImage(time.Unix(12345, 0), nil),
			nil,
			time.Unix(12345, 0),
		},
	}

	for _, testCase := range testList {
		mock := &ImageControllerMock{
			mockGet: func(user users.UserI, id string) (*images.ImageMeta, error) {
				return testCase.image, testCase.err
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

		recorded.CodeIs(testCase.responseCode)
		recorded.ContentTypeIsJson()
		recorded.BodyIs(testCase.body)

		if !testCase.time.Equal(time.Time{}) {
			recorded.HeaderIs("Last-Modified", testCase.time.UTC().Format(http.TimeFormat))
		}
	}
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
