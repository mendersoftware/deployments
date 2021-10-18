// Copyright 2021 Northern.tech AS
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
	"net/http"
	"testing"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/stretchr/testify/assert"

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
		filter        *dmodel.ReleaseOrImageFilter
		storeReleases []dmodel.Release
		storeErr      error
		checker       mt.ResponseChecker
	}{
		"ok": {
			filter: &dmodel.ReleaseOrImageFilter{},
			storeReleases: []dmodel.Release{
				{
					Artifacts: []model.Image{
						{
							Id: "1",
							ImageMeta: &model.ImageMeta{
								Description: "description",
							},

							ArtifactMeta: &model.ArtifactMeta{
								Name:                  "App1 v1.0",
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
					{
						Artifacts: []model.Image{
							{
								Id: "1",
								ImageMeta: &model.ImageMeta{
									Description: "description",
								},

								ArtifactMeta: &model.ArtifactMeta{
									Name:                  "App1 v1.0",
									DeviceTypesCompatible: []string{"bar", "baz"},
									Updates:               []model.Update{},
								},
							},
						},
					},
				}),
		},
		"ok, empty": {
			filter:        &dmodel.ReleaseOrImageFilter{},
			storeReleases: []dmodel.Release{},
			checker: mt.NewJSONResponse(
				http.StatusOK,
				nil,
				[]dmodel.Release{}),
		},
		"ok, filter": {
			filter:        &dmodel.ReleaseOrImageFilter{Name: "foo"},
			storeReleases: []dmodel.Release{},
			checker: mt.NewJSONResponse(
				http.StatusOK,
				nil,
				[]dmodel.Release{}),
		},
		"error: generic": {
			filter:        &dmodel.ReleaseOrImageFilter{},
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

		t.Run(name, func(t *testing.T) {
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

func TestListReleases(t *testing.T) {
	testCases := map[string]struct {
		filter        *dmodel.ReleaseOrImageFilter
		storeReleases []dmodel.Release
		storeErr      error
		checker       mt.ResponseChecker
	}{
		"ok": {
			filter: &dmodel.ReleaseOrImageFilter{Page: 1, PerPage: 20},
			storeReleases: []dmodel.Release{
				{
					Artifacts: []model.Image{
						{
							Id: "1",
							ImageMeta: &model.ImageMeta{
								Description: "description",
							},

							ArtifactMeta: &model.ArtifactMeta{
								Name:                  "App1 v1.0",
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
					{
						Artifacts: []model.Image{
							{
								Id: "1",
								ImageMeta: &model.ImageMeta{
									Description: "description",
								},

								ArtifactMeta: &model.ArtifactMeta{
									Name:                  "App1 v1.0",
									DeviceTypesCompatible: []string{"bar", "baz"},
									Updates:               []model.Update{},
								},
							},
						},
					},
				}),
		},
		"ok, empty": {
			filter:        &dmodel.ReleaseOrImageFilter{Page: 1, PerPage: 20},
			storeReleases: []dmodel.Release{},
			checker: mt.NewJSONResponse(
				http.StatusOK,
				nil,
				[]dmodel.Release{}),
		},
		"ok, filter": {
			filter:        &dmodel.ReleaseOrImageFilter{Name: "foo", Page: 1, PerPage: 20},
			storeReleases: []dmodel.Release{},
			checker: mt.NewJSONResponse(
				http.StatusOK,
				nil,
				[]dmodel.Release{}),
		},
		"error: generic": {
			filter:        &dmodel.ReleaseOrImageFilter{Page: 1, PerPage: 20},
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

		t.Run(name, func(t *testing.T) {
			store := &store_mocks.DataStore{}

			store.On("GetReleases", deployments_testing.ContextMatcher(), tc.filter).
				Return(tc.storeReleases, tc.storeErr)

			fileStorage := &fs_mocks.FileStorage{}

			restView := new(view.RESTView)
			app := app.NewDeployments(store, fileStorage, app.ArtifactContentType)

			c := NewDeploymentsApiHandlers(store, restView, app)

			api := deployments_testing.SetUpTestApi("/api/management/v1/deployments/releases/list", rest.Get, c.ListReleases)

			reqUrl := "http://1.2.3.4/api/management/v1/deployments/releases/list"

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

func TestGetReleasesFilter(t *testing.T) {
	testCases := map[string]struct {
		queryString string
		paginated   bool
		filter      *dmodel.ReleaseOrImageFilter
	}{
		"ok, empty": {
			filter: &dmodel.ReleaseOrImageFilter{},
		},
		"ok, name": {
			queryString: "name=foo",
			filter:      &dmodel.ReleaseOrImageFilter{Name: "foo"},
		},
		"ok, paginated, empty": {
			paginated: true,
			filter: &dmodel.ReleaseOrImageFilter{
				Page:    1,
				PerPage: DefaultPerPage,
			},
		},
		"ok, paginated, name": {
			queryString: "name=foo",
			paginated:   true,
			filter: &dmodel.ReleaseOrImageFilter{
				Name:    "foo",
				Page:    1,
				PerPage: DefaultPerPage,
			},
		},
		"ok, paginated, full options": {
			queryString: "name=foo&page=2&per_page=200&sort=name:asc",
			paginated:   true,
			filter: &dmodel.ReleaseOrImageFilter{
				Name:    "foo",
				Page:    2,
				PerPage: 200,
				Sort:    "name:asc",
			},
		},
		"ok, paginated, per page too high": {
			queryString: "per_page=10000000",
			paginated:   true,
			filter: &dmodel.ReleaseOrImageFilter{
				Page:    1,
				PerPage: DefaultPerPage,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			reqUrl := "http://1.2.3.4/api/management/v1/deployments/releases"
			req := &rest.Request{
				Request: test.MakeSimpleRequest("GET", reqUrl+"?"+tc.queryString, nil),
			}
			req.Header.Add(requestid.RequestIdHeader, "test")
			out := getReleaseOrImageFilter(req, tc.paginated)
			assert.Equal(t, out, tc.filter)
		})
	}
}
