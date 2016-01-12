package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
)

func TestVersionGet(t *testing.T) {

	toJson := func(data interface{}) string {
		j, _ := json.Marshal(data)
		return string(j)
	}

	testList := []struct {
		outResponseCode int
		outBody         string
		inVersion       string
		inBuild         string
	}{
		{
			http.StatusOK,
			toJson(Version{Version: "0.0.1", Build: "123"}),
			"0.0.1",
			"123",
		},
	}

	for _, testCase := range testList {

		router, err := rest.MakeRouter(rest.Get("/r/", NewVersion(testCase.inVersion, testCase.inBuild).Get))
		if err != nil {
			t.FailNow()
		}

		api := rest.NewApi()
		api.Use(&SetUserMiddleware{})
		api.SetApp(router)

		recorded := test.RunRequest(t, api.MakeHandler(),
			test.MakeSimpleRequest("GET", "http://1.2.3.4/r/", nil))

		recorded.CodeIs(testCase.outResponseCode)
		recorded.ContentTypeIsJson()
		recorded.BodyIs(testCase.outBody)
	}
}
