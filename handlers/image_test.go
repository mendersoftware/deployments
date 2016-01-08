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

func ToJson(data interface{}) string {
	j, _ := json.Marshal(data)
	return string(j)
}

func MakeImageMeta(setTime time.Time, id string, public *images.ImageMetaPublic) *images.ImageMeta {
	img := images.NewImageMetaFromPublic(public)
	img.LastUpdated = setTime
	img.Id = id
	return img
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
		{http.StatusOK, ToJson([]string{}), make([]*images.ImageMeta, 0), nil},
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
			ToJson(MakeImageMeta(time.Unix(12345, 0), "id1", nil)),
			MakeImageMeta(time.Unix(12345, 0), "id1", nil),
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

	const Id string = "123-123-123"

	testList := []struct {
		outResponseCode int
		outBody         string
		inPayload       interface{}
		inCreateError   error
	}{
		{
			http.StatusBadRequest,
			RestErrorMsg(errors.New("json: cannot unmarshal string into Go value of type images.ImageMetaPublic")),
			"{ invalid payload",
			nil,
		},
		{
			http.StatusBadRequest,
			RestErrorMsg(images.ErrMissingImageAttrName),
			images.NewImageMetaPublic(""),
			nil,
		},
		{
			http.StatusInternalServerError,
			RestErrorMsg(errors.New("TestError")),
			images.NewImageMetaPublic("MyImage"),
			errors.New("TestError"),
		},
		{
			http.StatusCreated,
			ToJson(MakeImageMeta(time.Unix(123, 0), Id, images.NewImageMetaPublic("MyImage"))),
			images.NewImageMetaPublic("MyImage"),
			nil,
		},
	}

	for _, testCase := range testList {

		mock := &ImageControllerMock{
			mockCreate: func(user users.UserI, public *images.ImageMetaPublic) (*images.ImageMeta, error) {
				img := images.NewImageMetaFromPublic(public)
				img.LastUpdated = time.Unix(123, 0)
				img.Id = Id
				return img, testCase.inCreateError
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
			test.MakeSimpleRequest("POST", "http://1.2.3.4/r/", testCase.inPayload))

		recorded.CodeIs(testCase.outResponseCode)
		recorded.ContentTypeIsJson()
		recorded.BodyIs(testCase.outBody)

		// Header should be set only for successul opration
		if testCase.outResponseCode == http.StatusCreated {
			recorded.HeaderIs("Location", "/api/0.0.1/images/"+Id)
		}
	}
}

func TestImageMetaEdit(t *testing.T) {

	const Id string = "123-123-123"

	testList := []struct {
		outResponseCode int
		outBody         string
		inPayload       interface{}
		inError         error
	}{
		{
			http.StatusBadRequest,
			RestErrorMsg(errors.New("json: cannot unmarshal string into Go value of type images.ImageMetaPublic")),
			"{ invalid payload",
			nil,
		},
		{
			http.StatusBadRequest,
			RestErrorMsg(images.ErrMissingImageAttrName),
			images.NewImageMetaPublic(""),
			nil,
		},
		{
			http.StatusInternalServerError,
			RestErrorMsg(errors.New("TestError")),
			images.NewImageMetaPublic("MyImage"),
			errors.New("TestError"),
		},
		{
			http.StatusNotFound,
			RestErrorMsg(errors.New("Resource not found")),
			images.NewImageMetaPublic("MyImage"),
			controllers.ErrNotFound,
		},
		{
			http.StatusNoContent,
			"",
			images.NewImageMetaPublic("MyImage"),
			nil,
		},
	}

	for _, testCase := range testList {

		mock := &ImageControllerMock{
			mockEdit: func(user users.UserI, id string, public *images.ImageMetaPublic) error {
				return testCase.inError
			},
		}

		router, err := rest.MakeRouter(rest.Put("/r/:id", NewImageMeta(mock).Edit))
		if err != nil {
			t.FailNow()
		}

		api := rest.NewApi()
		api.Use(&SetUserMiddleware{})
		api.SetApp(router)

		recorded := test.RunRequest(t, api.MakeHandler(),
			test.MakeSimpleRequest("PUT", "http://1.2.3.4/r/"+Id, testCase.inPayload))

		recorded.CodeIs(testCase.outResponseCode)
		recorded.ContentTypeIsJson()
		recorded.BodyIs(testCase.outBody)

		// Header should be set only for successul opration
		if testCase.outResponseCode == http.StatusNoContent {
			recorded.HeaderIs("Location", "/r/"+Id)
		}
	}
}

func TestImageMetaDelete(t *testing.T) {

	const Id string = "123-123-123"

	testList := []struct {
		outResponseCode int
		outBody         string
		inError         error
	}{
		{
			http.StatusInternalServerError,
			RestErrorMsg(errors.New("TestError")),
			errors.New("TestError"),
		},
		{
			http.StatusNotFound,
			RestErrorMsg(errors.New("Resource not found")),
			controllers.ErrNotFound,
		},
		{
			http.StatusNoContent,
			"",
			nil,
		},
	}

	for _, testCase := range testList {

		mock := &ImageControllerMock{
			mockDelete: func(user users.UserI, id string) error {
				return testCase.inError
			},
		}

		router, err := rest.MakeRouter(rest.Delete("/r/:id", NewImageMeta(mock).Delete))
		if err != nil {
			t.FailNow()
		}

		api := rest.NewApi()
		api.Use(&SetUserMiddleware{})
		api.SetApp(router)

		recorded := test.RunRequest(t, api.MakeHandler(),
			test.MakeSimpleRequest("DELETE", "http://1.2.3.4/r/"+Id, nil))

		recorded.CodeIs(testCase.outResponseCode)
		recorded.ContentTypeIsJson()
		recorded.BodyIs(testCase.outBody)
	}
}
