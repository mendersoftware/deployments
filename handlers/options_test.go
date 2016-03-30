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
	"net/http"
	"testing"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
)

func TestOptionsHandle(t *testing.T) {
	router, err := rest.MakeRouter(rest.Options("/r", NewOptionsHandler(http.MethodGet, http.MethodGet)))
	if err != nil {
		t.FailNow()
	}

	api := rest.NewApi()
	api.SetApp(router)

	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest(http.MethodOptions, "http://1.2.3.4/r", nil))

	recorded.CodeIs(http.StatusOK)

	if len(recorded.Recorder.Header()[HttpHeaderAllow]) != 2 {
		t.FailNow()
	}

	for _, method := range recorded.Recorder.Header()[HttpHeaderAllow] {
		switch method {
		case http.MethodGet:
			continue
		case http.MethodOptions:
			continue
		default:
			t.FailNow()
		}
	}
}
