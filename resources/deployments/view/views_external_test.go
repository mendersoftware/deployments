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

package view_test

import (
	"net/http"
	"testing"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	. "github.com/mendersoftware/deployments/resources/deployments/view"
	"github.com/stretchr/testify/assert"
)

func TestRenderNoUpdateForDevice(t *testing.T) {

	t.Parallel()

	router, err := rest.MakeRouter(rest.Get("/test", func(w rest.ResponseWriter, r *rest.Request) {
		view := &DeploymentsViews{}
		view.RenderNoUpdateForDevice(w)
	}))

	if err != nil {
		assert.NoError(t, err)
	}

	api := rest.NewApi()
	api.SetApp(router)

	recorded := test.RunRequest(t, api.MakeHandler(),
		test.MakeSimpleRequest("GET", "http://localhost/test", nil))

	recorded.CodeIs(http.StatusNoContent)
}
