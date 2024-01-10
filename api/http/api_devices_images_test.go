// Copyright 2024 Northern.tech AS
//
//	Licensed under the Apache License, Version 2.0 (the "License");
//	you may not use this file except in compliance with the License.
//	You may obtain a copy of the License at
//
//	    http://www.apache.org/licenses/LICENSE-2.0
//
//	Unless required by applicable law or agreed to in writing, software
//	distributed under the License is distributed on an "AS IS" BASIS,
//	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	See the License for the specific language governing permissions and
//	limitations under the License.

package http

import (
	"net/http"
	"testing"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	"github.com/google/uuid"
	"github.com/mendersoftware/go-lib-micro/requestid"
	mt "github.com/mendersoftware/go-lib-micro/testing"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"

	app_mocks "github.com/mendersoftware/deployments/app/mocks"
	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/utils/restutil/view"
	deployments_testing "github.com/mendersoftware/deployments/utils/testing"
)

func TestGetImagesForDevice(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		filter   *model.ReleaseOrImageFilter
		images   []*model.Image
		appError error
		checker  mt.ResponseChecker
	}{
		"ok": {
			filter: &model.ReleaseOrImageFilter{
				Page:    1,
				PerPage: MaxImagesForDevice,
			},
			images: []*model.Image{
				{
					Id:   "1",
					Size: 1000,
				},
			},
			checker: mt.NewJSONResponse(
				http.StatusOK,
				nil,
				[]*model.Image{
					{
						Id:   "1",
						Size: 1000,
					},
				},
			),
		},
		"ok, empty": {
			filter: &model.ReleaseOrImageFilter{
				Page:    1,
				PerPage: MaxImagesForDevice,
			},
			images: []*model.Image{},
			checker: mt.NewJSONResponse(
				http.StatusOK,
				nil,
				[]*model.Image{},
			),
		},
		"ok, filter release": {
			filter: &model.ReleaseOrImageFilter{
				Name:    "foo",
				Page:    1,
				PerPage: MaxImagesForDevice,
			},
			images: []*model.Image{},
			checker: mt.NewJSONResponse(
				http.StatusOK,
				nil,
				[]*model.Image{},
			),
		},
		"ok, filter device type": {
			filter: &model.ReleaseOrImageFilter{
				DeviceType: "foo",
				Page:       1,
				PerPage:    MaxImagesForDevice,
			},
			images: []*model.Image{},
			checker: mt.NewJSONResponse(
				http.StatusOK,
				nil,
				[]*model.Image{},
			),
		},
		"error: generic": {
			filter: &model.ReleaseOrImageFilter{
				Page:    1,
				PerPage: MaxImagesForDevice,
			},
			images:   []*model.Image{},
			appError: errors.New("database error"),
			checker: mt.NewJSONResponse(
				http.StatusInternalServerError,
				nil,
				deployments_testing.RestError("internal error"),
			),
		},
	}

	for name := range testCases {
		tc := testCases[name]

		t.Run(name, func(t *testing.T) {
			restView := new(view.RESTView)
			app := &app_mocks.App{}
			defer app.AssertExpectations(t)

			app.On("ListImages",
				deployments_testing.ContextMatcher(),
				tc.filter,
			).Return(tc.images, len(tc.images), tc.appError)

			c := NewDeploymentsApiHandlers(nil, restView, app)

			reqUrl := "http://1.2.3.4/api/devices/v1/artifacts"

			if tc.filter != nil {
				reqUrl += "?name=" + tc.filter.Name + "&device_type=" + tc.filter.DeviceType
			}

			req := test.MakeSimpleRequest("GET", reqUrl, nil)
			req.Header.Add(requestid.RequestIdHeader, "test")

			api := deployments_testing.SetUpTestApi("/api/devices/v1/artifacts", rest.Get, c.GetImagesForDevice)
			recorded := test.RunRequest(t, api, req)

			mt.CheckResponse(t, tc.checker, recorded)
		})
	}
}

func TestGetImageForDevice(t *testing.T) {
	t.Parallel()

	id := uuid.New().String()
	testCases := map[string]struct {
		id       string
		image    *model.Image
		appError error
		checker  mt.ResponseChecker
	}{
		"ok": {
			id: id,
			image: &model.Image{
				Id:   id,
				Size: 1000,
			},
			checker: mt.NewJSONResponse(
				http.StatusOK,
				nil,
				&model.Image{
					Id:   id,
					Size: 1000,
				},
			),
		},
		"error: not found": {
			id: id,
			checker: mt.NewJSONResponse(
				http.StatusNotFound,
				nil,
				deployments_testing.RestError("Resource not found"),
			),
		},
		"error: invalid uuid": {
			id: "id",
			checker: mt.NewJSONResponse(
				http.StatusBadRequest,
				nil,
				deployments_testing.RestError("ID is not a valid UUID"),
			),
		},
		"error: generic": {
			id:       id,
			appError: errors.New("database error"),
			checker: mt.NewJSONResponse(
				http.StatusInternalServerError,
				nil,
				deployments_testing.RestError("internal error"),
			),
		},
	}

	for name := range testCases {
		tc := testCases[name]

		t.Run(name, func(t *testing.T) {
			restView := new(view.RESTView)
			app := &app_mocks.App{}
			defer app.AssertExpectations(t)

			if tc.id != "id" {
				app.On("GetImage",
					deployments_testing.ContextMatcher(),
					tc.id,
				).Return(tc.image, tc.appError)
			}

			c := NewDeploymentsApiHandlers(nil, restView, app)

			reqUrl := "http://1.2.3.4/api/devices/v1/artifacts/" + tc.id

			req := test.MakeSimpleRequest("GET", reqUrl, nil)
			req.Header.Add(requestid.RequestIdHeader, "test")

			api := deployments_testing.SetUpTestApi("/api/devices/v1/artifacts/#id", rest.Get, c.GetImageForDevice)
			recorded := test.RunRequest(t, api, req)

			mt.CheckResponse(t, tc.checker, recorded)
		})
	}
}

func TestDownloadImageForDevice(t *testing.T) {
	t.Parallel()

	id := uuid.New().String()
	testCases := map[string]struct {
		id       string
		link     *model.Link
		appError error
		checker  mt.ResponseChecker
	}{
		"ok": {
			id: id,
			link: &model.Link{
				Uri: "http://localhost/artifact.mender",
			},
			checker: mt.NewJSONResponse(
				http.StatusOK,
				nil,
				&model.Link{
					Uri: "http://localhost/artifact.mender",
				},
			),
		},
		"error: not found": {
			id: id,
			checker: mt.NewJSONResponse(
				http.StatusNotFound,
				nil,
				deployments_testing.RestError("Resource not found"),
			),
		},
		"error: invalid uuid": {
			id: "id",
			checker: mt.NewJSONResponse(
				http.StatusBadRequest,
				nil,
				deployments_testing.RestError("ID is not a valid UUID"),
			),
		},
		"error: generic": {
			id:       id,
			appError: errors.New("database error"),
			checker: mt.NewJSONResponse(
				http.StatusInternalServerError,
				nil,
				deployments_testing.RestError("internal error"),
			),
		},
	}

	for name := range testCases {
		tc := testCases[name]

		t.Run(name, func(t *testing.T) {
			restView := new(view.RESTView)
			app := &app_mocks.App{}
			defer app.AssertExpectations(t)

			if tc.id != "id" {
				app.On("DownloadLink",
					deployments_testing.ContextMatcher(),
					tc.id,
					mock.AnythingOfType("time.Duration"),
				).Return(tc.link, tc.appError)
			}

			c := NewDeploymentsApiHandlers(nil, restView, app)

			reqUrl := "http://1.2.3.4/api/devices/v1/artifacts/" + tc.id + "/download"

			req := test.MakeSimpleRequest("GET", reqUrl, nil)
			req.Header.Add(requestid.RequestIdHeader, "test")

			api := deployments_testing.SetUpTestApi("/api/devices/v1/artifacts/#id/download", rest.Get, c.DownloadImageForDevice)
			recorded := test.RunRequest(t, api, req)

			mt.CheckResponse(t, tc.checker, recorded)
		})
	}
}
