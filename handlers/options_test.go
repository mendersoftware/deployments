package handlers

import (
	"net/http"
	"testing"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
)

func TestOptionsHandle(t *testing.T) {
	router, err := rest.MakeRouter(rest.Options("/r", NewOptionsHandler(HttpMethodGet, HttpMethodGet).Handle))
	if err != nil {
		t.FailNow()
	}

	api := rest.NewApi()
	api.SetApp(router)

	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest(HttpMethodOptions, "http://1.2.3.4/r", nil))

	recorded.CodeIs(http.StatusOK)

	if len(recorded.Recorder.Header()[HttpHeaderAllow]) != 2 {
		t.FailNow()
	}

	for _, method := range recorded.Recorder.Header()[HttpHeaderAllow] {
		switch method {
		case HttpMethodGet:
			continue
		case HttpMethodOptions:
			continue
		default:
			t.FailNow()
		}
	}
}
