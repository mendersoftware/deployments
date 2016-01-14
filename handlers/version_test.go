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
