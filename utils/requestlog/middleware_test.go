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
package requestlog

import (
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRequestLogMiddleware(t *testing.T) {
	api := rest.NewApi()

	api.Use(&RequestLogMiddleware{})

	api.SetApp(rest.AppSimple(func(w rest.ResponseWriter, r *rest.Request) {
		log := r.Env[ReqLog]
		assert.NotNil(t, log)
		w.WriteJson(map[string]string{"foo": "bar"})
	}))

	handler := api.MakeHandler()

	req := test.MakeSimpleRequest("GET", "http://localhost/", nil)

	_ = test.RunRequest(t, handler, req)
}
