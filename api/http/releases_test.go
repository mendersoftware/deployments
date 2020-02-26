// Copyright 2019 Northern.tech AS
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
package http

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	"github.com/mendersoftware/go-lib-micro/requestid"

	"github.com/mendersoftware/deployments/app"
	"github.com/mendersoftware/deployments/model"
	dmodel "github.com/mendersoftware/deployments/model"
	fs_mocks "github.com/mendersoftware/deployments/s3/mocks"
	store_mocks "github.com/mendersoftware/deployments/store/mocks"
	"github.com/mendersoftware/deployments/utils/restutil/view"
	deployments_testing "github.com/mendersoftware/deployments/utils/testing"
	mt "github.com/mendersoftware/go-lib-micro/testing"
)

func TestGetReleases(t *testing.T) {

	testCases := map[string]struct {
		filter        *dmodel.ReleaseFilter
		storeReleases []dmodel.Release
		storeErr      error
		checker       mt.ResponseChecker
	}{
		"ok": {
			storeReleases: []dmodel.Release{
				dmodel.Release{
					Artifacts: []model.Image{
						model.Image{
							Id: "1",
							ImageMeta: model.ImageMeta{
								Description: "description",
							},

							ArtifactMeta: model.ArtifactMeta{
								Name: "App1 v1.0",
								DeviceTypesCompatible: []string{"bar", "baz"},
								Updates:               []model.Update{},
							},
						},
					},
				},
			},
			checker: mt.NewJSONResponse(
				http.StatusOK,
				nil,
				[]dmodel.Release{
					dmodel.Release{
						Artifacts: []model.Image{
							model.Image{
								Id: "1",
								ImageMeta: model.ImageMeta{
									Description: "description",
								},

								ArtifactMeta: model.ArtifactMeta{
									Name: "App1 v1.0",
									DeviceTypesCompatible: []string{"bar", "baz"},
									Updates:               []model.Update{},
								},
							},
						},
					},
				}),
		},
		"ok, empty": {
			storeReleases: []dmodel.Release{},
			checker: mt.NewJSONResponse(
				http.StatusOK,
				nil,
				[]dmodel.Release{}),
		},
		"ok, filter": {
			filter:        &dmodel.ReleaseFilter{Name: "foo"},
			storeReleases: []dmodel.Release{},
			checker: mt.NewJSONResponse(
				http.StatusOK,
				nil,
				[]dmodel.Release{}),
		},
		"error: generic": {
			storeReleases: nil,
			storeErr:      errors.New("database error"),
			checker: mt.NewJSONResponse(
				http.StatusInternalServerError,
				nil,
				deployments_testing.RestError("internal error")),
		},
	}

	for name := range testCases {
		tc := testCases[name]

		t.Run(fmt.Sprintf("%s", name), func(t *testing.T) {
			store := &store_mocks.DataStore{}

			store.On("GetReleases", deployments_testing.ContextMatcher(), tc.filter).
				Return(tc.storeReleases, tc.storeErr)

			fileStorage := &fs_mocks.FileStorage{}

			restView := new(view.RESTView)
			app := app.NewDeployments(store, fileStorage, app.ArtifactContentType)

			c := NewDeploymentsApiHandlers(store, restView, app)

			api := deployments_testing.SetUpTestApi("/api/management/v1/deployments/releases", rest.Get, c.GetReleases)

			reqUrl := "http://1.2.3.4/api/management/v1/deployments/releases"

			if tc.filter != nil {
				reqUrl += "?name=" + tc.filter.Name
			}

			req := test.MakeSimpleRequest("GET",
				reqUrl,
				nil)

			req.Header.Add(requestid.RequestIdHeader, "test")

			recorded := test.RunRequest(t, api, req)

			mt.CheckResponse(t, tc.checker, recorded)
		})
	}
}
