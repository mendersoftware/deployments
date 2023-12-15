// Copyright 2023 Northern.tech AS
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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/mendersoftware/deployments/app"
	mapp "github.com/mendersoftware/deployments/app/mocks"
	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/store"
	"github.com/mendersoftware/deployments/utils/restutil/view"
	h "github.com/mendersoftware/deployments/utils/testing"
	"github.com/mendersoftware/go-lib-micro/identity"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/mendersoftware/go-lib-micro/rest_utils"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
)

func TestAlive(t *testing.T) {
	t.Parallel()

	req, _ := http.NewRequest("GET", "http://localhost"+ApiUrlInternalAlive, nil)
	d := NewDeploymentsApiHandlers(nil, nil, nil)
	api := setUpRestTest(ApiUrlInternalAlive, rest.Get, d.AliveHandler)
	recorded := test.RunRequest(t, api.MakeHandler(), req)
	recorded.CodeIs(http.StatusNoContent)
}

func TestHealthCheck(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Name string

		AppError     error
		ResponseCode int
		ResponseBody interface{}
	}{{
		Name:         "ok",
		ResponseCode: http.StatusNoContent,
	}, {
		Name:         "error: app unhealthy",
		AppError:     errors.New("*COUGH! COUGH!*"),
		ResponseCode: http.StatusServiceUnavailable,
		ResponseBody: rest_utils.ApiError{
			Err:   "*COUGH! COUGH!*",
			ReqId: "test",
		},
	}}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			app := &mapp.App{}
			app.On("HealthCheck", mock.MatchedBy(
				func(ctx interface{}) bool {
					if _, ok := ctx.(context.Context); ok {
						return true
					}
					return false
				}),
			).Return(tc.AppError)
			d := NewDeploymentsApiHandlers(nil, nil, app)
			api := setUpRestTest(
				ApiUrlInternalHealth,
				rest.Get,
				d.HealthHandler,
			)
			req, _ := http.NewRequest(
				"GET",
				"http://localhost"+ApiUrlInternalHealth,
				nil,
			)
			req.Header.Set("X-MEN-RequestID", "test")
			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.ResponseCode)
			if tc.ResponseBody != nil {
				b, _ := json.Marshal(tc.ResponseBody)
				assert.JSONEq(t, string(b), recorded.Recorder.Body.String())
			} else {
				recorded.BodyIs("")
			}
		})
	}
}

func TestDeploymentsPerTenantHandler(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		tenant       string
		queryString  string
		appError     error
		query        *model.Query
		deployments  []*model.Deployment
		count        int64
		responseCode int
		responseBody interface{}
	}{
		"ok": {
			tenant: "tenantID",
			query: &model.Query{
				Limit: rest_utils.PerPageDefault + 1,
				Sort:  model.SortDirectionDescending,
			},
			deployments:  []*model.Deployment{},
			count:        0,
			responseCode: http.StatusOK,
			responseBody: []*model.Deployment{},
		},
		"ok with pagination": {
			tenant:      "tenantID",
			queryString: rest_utils.PerPageName + "=50&" + rest_utils.PageName + "=2",
			query: &model.Query{
				Skip:  50,
				Limit: 51,
				Sort:  model.SortDirectionDescending,
			},
			deployments:  []*model.Deployment{},
			count:        0,
			responseCode: http.StatusOK,
			responseBody: []*model.Deployment{},
		},
		"ko, missing tenant ID": {
			tenant:       "",
			responseCode: http.StatusBadRequest,
			responseBody: rest_utils.ApiError{
				Err:   "missing tenant ID",
				ReqId: "test",
			},
		},
		"ko, error in pagination": {
			tenant:       "tenantID",
			queryString:  rest_utils.PerPageName + "=a",
			responseCode: http.StatusBadRequest,
			responseBody: rest_utils.ApiError{
				Err:   "Can't parse param per_page",
				ReqId: "test",
			},
		},
		"ko, error in filters": {
			tenant:       "tenantID",
			queryString:  "created_before=a",
			responseCode: http.StatusBadRequest,
			responseBody: rest_utils.ApiError{
				Err:   "timestamp parsing failed for created_before parameter: invalid timestamp: a",
				ReqId: "test",
			},
		},
		"ko, error in LookupDeployment": {
			tenant: "tenantID",
			query: &model.Query{
				Limit: rest_utils.PerPageDefault + 1,
				Sort:  model.SortDirectionDescending,
			},
			appError:     errors.New("generic error"),
			deployments:  []*model.Deployment{},
			count:        0,
			responseCode: http.StatusBadRequest,
			responseBody: rest_utils.ApiError{
				Err:   "generic error",
				ReqId: "test",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			app := &mapp.App{}
			if tc.query != nil {
				app.On("LookupDeployment",
					mock.MatchedBy(func(ctx context.Context) bool {
						return true
					}),
					*tc.query,
				).Return(tc.deployments, tc.count, tc.appError)
			}
			defer app.AssertExpectations(t)

			restView := new(view.RESTView)
			d := NewDeploymentsApiHandlers(nil, restView, app)
			api := setUpRestTest(
				ApiUrlInternalTenantDeployments,
				rest.Get,
				d.DeploymentsPerTenantHandler,
			)

			url := strings.Replace(ApiUrlInternalTenantDeployments, "#tenant", tc.tenant, 1)
			if tc.queryString != "" {
				url = url + "?" + tc.queryString
			}
			req, _ := http.NewRequest(
				"GET",
				"http://localhost"+url,
				bytes.NewReader([]byte("")),
			)
			req.Header.Set("X-MEN-RequestID", "test")
			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.responseCode)
			if tc.responseBody != nil {
				b, _ := json.Marshal(tc.responseBody)
				assert.JSONEq(t, string(b), recorded.Recorder.Body.String())
			} else {
				recorded.BodyIs("")
			}
		})
	}
}

func TestUploadLink(t *testing.T) {
	t.Parallel()

	type testCase struct {
		Name string

		App func(t *testing.T) *mapp.App

		StatusCode        int
		BodyAssertionFunc func(t *testing.T, body string) bool
	}
	testCases := []testCase{{
		Name: "ok",

		App: func(t *testing.T) *mapp.App {
			app := new(mapp.App)
			expire := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
			app.On("UploadLink", contextMatcher(), mock.AnythingOfType("time.Duration"), false).
				Return(&model.UploadLink{
					ArtifactID: "00000000-0000-0000-0000-000000000000",
					Link: model.Link{
						Uri:    "http://localhost:8080",
						Method: "PUT",
						Expire: expire,
					},
				}, nil)

			return app
		},

		StatusCode: http.StatusOK,
		BodyAssertionFunc: func(t *testing.T, body string) bool {
			expire := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
			expectedBody, _ := json.Marshal(model.UploadLink{
				ArtifactID: "00000000-0000-0000-0000-000000000000",
				Link: model.Link{
					Uri:    "http://localhost:8080",
					Method: "PUT",
					Expire: expire,
				},
			})
			return assert.Equal(t, string(expectedBody), body, "unexpected HTTP body")
		},
	}, {
		Name: "error/generating signed URL",

		App: func(t *testing.T) *mapp.App {
			app := new(mapp.App)
			app.On("UploadLink", contextMatcher(), mock.AnythingOfType("time.Duration"), false).
				Return(nil, errors.New("error generating URL"))

			return app
		},

		StatusCode: http.StatusInternalServerError,
		BodyAssertionFunc: func(t *testing.T, body string) bool {
			return true
		},
	}, {
		Name: "error/not found",

		App: func(t *testing.T) *mapp.App {
			app := new(mapp.App)
			app.On("UploadLink", contextMatcher(), mock.AnythingOfType("time.Duration"), false).
				Return(nil, nil)

			return app
		},

		StatusCode: http.StatusNotFound,
		BodyAssertionFunc: func(t *testing.T, body string) bool {
			return true
		},
	}}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			req, _ := http.NewRequest(
				http.MethodPost,
				"https://localhost:8443"+ApiUrlManagementArtifactsDirectUpload,
				nil)
			app := tc.App(t)
			defer app.AssertExpectations(t)

			conf := NewConfig().
				SetEnableDirectUpload(true)
			apiHandler, err := NewHandler(
				ctx,
				app,
				nil,
				conf,
			)
			if err != nil {
				panic(err)
			}
			w := httptest.NewRecorder()
			apiHandler.ServeHTTP(w, req)

			assert.Equal(t, tc.StatusCode, w.Code, "Unexpected HTTP status code")
			tc.BodyAssertionFunc(t, w.Body.String())
		})
	}
}

func TestCompleteUpload(t *testing.T) {
	t.Parallel()

	const sampleID = "a5522c47-3c99-459b-ae6b-6049c744db7f"

	type testCase struct {
		Name string

		ID  string
		App func(t *testing.T) *mapp.App

		StatusCode        int
		BodyAssertionFunc func(t *testing.T, body string) bool
	}
	testCases := []testCase{{
		Name: "ok",

		ID: sampleID,
		App: func(t *testing.T) *mapp.App {
			app := new(mapp.App)
			app.On("CompleteUpload", contextMatcher(), sampleID, false, mock.AnythingOfType("*model.DirectUploadMetadata")).
				Return(nil)
			return app
		},

		StatusCode: http.StatusAccepted,
		BodyAssertionFunc: func(t *testing.T, body string) bool {
			return assert.Empty(t, body, body, "expected body to be empty")
		},
	}, {
		Name: "error/internal",

		ID: sampleID,
		App: func(t *testing.T) *mapp.App {
			app := new(mapp.App)
			app.On("CompleteUpload", contextMatcher(), sampleID, false, mock.AnythingOfType("*model.DirectUploadMetadata")).
				Return(errors.New("internal error"))

			return app
		},

		StatusCode: http.StatusInternalServerError,
		BodyAssertionFunc: func(t *testing.T, body string) bool {
			return assert.Regexp(t,
				`"error":"internal server error"`,
				string(body),
				"unexpected error response body",
			)
		},
	}, {
		Name: "error/not found",

		ID: sampleID,
		App: func(t *testing.T) *mapp.App {
			mockApp := new(mapp.App)
			mockApp.On("CompleteUpload", contextMatcher(), sampleID, false, mock.AnythingOfType("*model.DirectUploadMetadata")).
				Return(app.ErrUploadNotFound)
			return mockApp
		},

		StatusCode: http.StatusNotFound,
		BodyAssertionFunc: func(t *testing.T, body string) bool {
			return true
		},
	}}
	pathGen := func(id string) string {
		return strings.ReplaceAll(
			ApiUrlManagementArtifactsCompleteUpload, "#id", id,
		)
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			req, _ := http.NewRequest(
				http.MethodPost,
				"https://localhost:8443"+pathGen(tc.ID),
				nil,
			)
			app := tc.App(t)
			defer app.AssertExpectations(t)

			conf := NewConfig().
				SetEnableDirectUpload(true)
			apiHandler, err := NewHandler(
				ctx,
				app,
				nil,
				conf,
			)
			if err != nil {
				panic(err)
			}

			w := httptest.NewRecorder()
			apiHandler.ServeHTTP(w, req)

			assert.Equal(t, tc.StatusCode, w.Code, "Unexpected HTTP status code")
			tc.BodyAssertionFunc(t, w.Body.String())
		})
	}
}

func TestPostDeployment(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Name      string
		InputBody interface{}

		AppError               error
		ResponseCode           int
		ResponseLocationHeader string
		ResponseBody           interface{}
	}{{
		Name: "ok, device list",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
			Devices:      []string{"f826484e-1157-4109-af21-304e6d711560"},
		},
		ResponseCode:           http.StatusCreated,
		ResponseLocationHeader: "./management/v1/deployments/deployments/foo",
	}, {
		Name: "ok, all devices",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
			AllDevices:   true,
		},
		ResponseCode:           http.StatusCreated,
		ResponseLocationHeader: "./management/v1/deployments/deployments/foo",
	}, {
		Name:         "error: empty payload",
		ResponseCode: http.StatusBadRequest,
		ResponseBody: rest_utils.ApiError{
			Err:   "Validating request body: JSON payload is empty",
			ReqId: "test",
		},
	}, {
		Name: "error: app error",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
			AllDevices:   true,
		},
		AppError:     errors.New("some error"),
		ResponseCode: http.StatusInternalServerError,
		ResponseBody: rest_utils.ApiError{
			Err:   "internal error",
			ReqId: "test",
		},
	}, {
		Name: "error: app error: no devices",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
			AllDevices:   true,
		},
		AppError:     app.ErrNoDevices,
		ResponseCode: http.StatusBadRequest,
		ResponseBody: rest_utils.ApiError{
			Err:   app.ErrNoDevices.Error(),
			ReqId: "test",
		},
	}, {
		Name: "error: conflict",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
			Devices:      []string{"f826484e-1157-4109-af21-304e6d711560"},
			AllDevices:   true,
		},
		ResponseCode: http.StatusBadRequest,
		ResponseBody: rest_utils.ApiError{
			Err:   "Validating request body: Invalid deployments definition: list of devices provided togheter with all_devices flag",
			ReqId: "test",
		},
	}, {
		Name: "error: no devices",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
		},
		ResponseCode: http.StatusBadRequest,
		ResponseBody: rest_utils.ApiError{
			Err:   "Validating request body: Invalid deployments definition: provide list of devices or set all_devices flag",
			ReqId: "test",
		},
	}}
	var constructor *model.DeploymentConstructor
	for _, tc := range testCases {
		if tc.InputBody != nil {
			constructor = tc.InputBody.(*model.DeploymentConstructor)
		} else {
			constructor = nil
		}
		t.Run(tc.Name, func(t *testing.T) {
			app := &mapp.App{}
			app.On("CreateDeployment", mock.MatchedBy(
				func(ctx interface{}) bool {
					if _, ok := ctx.(context.Context); ok {
						return true
					}
					return false
				}),
				constructor,
			).Return("foo", tc.AppError)
			restView := new(view.RESTView)
			d := NewDeploymentsApiHandlers(nil, restView, app)
			api := setUpRestTest(
				ApiUrlManagementDeployments,
				rest.Post,
				d.PostDeployment,
			)

			req := test.MakeSimpleRequest(
				"POST",
				"http://localhost"+ApiUrlManagementDeployments,
				tc.InputBody,
			)
			req.Header.Set("X-MEN-RequestID", "test")
			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.ResponseCode)
			if tc.ResponseLocationHeader != "" {
				recorded.HeaderIs("Location", tc.ResponseLocationHeader)
			}
			if tc.ResponseBody != nil {
				b, _ := json.Marshal(tc.ResponseBody)
				assert.JSONEq(t, string(b), recorded.Recorder.Body.String())
			} else {
				recorded.BodyIs("")
			}
		})
	}
}

func TestPostDeploymentToGroup(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Name       string
		InputBody  interface{}
		InputGroup string

		AppError               error
		ResponseCode           int
		ResponseLocationHeader string
		ResponseBody           interface{}
	}{{
		Name: "ok",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
		},
		InputGroup:             "baz",
		ResponseCode:           http.StatusCreated,
		ResponseLocationHeader: "./management/v1/deployments/deployments/foo",
	}, {
		Name:         "error: empty payload",
		InputGroup:   "baz",
		ResponseCode: http.StatusBadRequest,
		ResponseBody: rest_utils.ApiError{
			Err:   "Validating request body: JSON payload is empty",
			ReqId: "test",
		},
	}, {
		Name: "error: conflict",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
			Devices:      []string{"f826484e-1157-4109-af21-304e6d711560"},
			AllDevices:   true,
		},
		InputGroup:   "baz",
		ResponseCode: http.StatusBadRequest,
		ResponseBody: rest_utils.ApiError{
			Err:   "Validating request body: The deployment for group constructor should have neither list of devices nor all_devices flag set",
			ReqId: "test",
		},
	}, {
		Name: "error: app error",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
		},
		InputGroup:   "baz",
		AppError:     errors.New("some error"),
		ResponseCode: http.StatusInternalServerError,
		ResponseBody: rest_utils.ApiError{
			Err:   "internal error",
			ReqId: "test",
		},
	}, {
		Name: "error: app error: no devices",
		InputBody: &model.DeploymentConstructor{
			Name:         "foo",
			ArtifactName: "bar",
		},
		InputGroup:   "baz",
		AppError:     app.ErrNoDevices,
		ResponseCode: http.StatusBadRequest,
		ResponseBody: rest_utils.ApiError{
			Err:   app.ErrNoDevices.Error(),
			ReqId: "test",
		},
	}}
	var constructor *model.DeploymentConstructor
	for _, tc := range testCases {
		if tc.InputBody != nil {
			constructor = tc.InputBody.(*model.DeploymentConstructor)
			constructor.Group = tc.InputGroup
		} else {
			constructor = nil
		}
		t.Run(tc.Name, func(t *testing.T) {
			app := &mapp.App{}
			app.On("CreateDeployment", mock.MatchedBy(
				func(ctx interface{}) bool {
					if _, ok := ctx.(context.Context); ok {
						return true
					}
					return false
				}),
				constructor,
			).Return("foo", tc.AppError)
			restView := new(view.RESTView)
			d := NewDeploymentsApiHandlers(
				nil,
				restView,
				app,
				NewConfig().SetEnableDirectUpload(true),
			)
			api := setUpRestTest(
				ApiUrlManagementDeploymentsGroup,
				rest.Post,
				d.DeployToGroup,
			)

			req := test.MakeSimpleRequest(
				"POST",
				"http://localhost"+ApiUrlManagementDeployments+"/group/"+tc.InputGroup,
				tc.InputBody,
			)
			req.Header.Set("X-MEN-RequestID", "test")
			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.ResponseCode)
			if tc.ResponseLocationHeader != "" {
				recorded.HeaderIs("Location", tc.ResponseLocationHeader)
			}
			if tc.ResponseBody != nil {
				b, _ := json.Marshal(tc.ResponseBody)
				assert.JSONEq(t, string(b), recorded.Recorder.Body.String())
			} else {
				recorded.BodyIs("")
			}
		})
	}
}

func TestControllerPostConfigurationDeployment(t *testing.T) {

	t.Parallel()

	testCases := map[string]struct {
		h.JSONResponseParams

		InputBodyObject interface{}

		InputTenantID                           string
		InputDeviceID                           string
		InputDeploymentID                       string
		InputCreateConfigurationDeploymentError error
	}{
		"ok": {
			InputBodyObject: &model.ConfigurationDeploymentConstructor{
				Name:          "NYC Production",
				Configuration: []byte("App 123"),
			},
			InputTenantID:     "foo",
			InputDeviceID:     "bar",
			InputDeploymentID: "baz",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusCreated,
				OutputBodyObject: nil,
				OutputHeaders:    map[string]string{"Location": "./deployments/baz"},
			},
		},
		"ok, object configuration encoding": {
			InputBodyObject: map[string]interface{}{
				"name":          "NYC Production",
				"configuration": map[string]interface{}{"App": "123"},
			},
			InputTenantID:     "foo",
			InputDeviceID:     "bar",
			InputDeploymentID: "baz",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusCreated,
				OutputBodyObject: nil,
				OutputHeaders:    map[string]string{"Location": "./deployments/baz"},
			},
		},
		"ko, empty body": {
			InputBodyObject:   nil,
			InputTenantID:     "foo",
			InputDeviceID:     "bar",
			InputDeploymentID: "baz",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Validating request body: JSON payload is empty")),
			},
		},
		"ko, empty deployment": {
			InputBodyObject:   &model.ConfigurationDeploymentConstructor{},
			InputTenantID:     "foo",
			InputDeviceID:     "bar",
			InputDeploymentID: "baz",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Validating request body: configuration: cannot be blank; name: cannot be blank.")),
			},
		},
		"ko, internal error": {
			InputBodyObject: &model.ConfigurationDeploymentConstructor{
				Name:          "foo",
				Configuration: []byte("bar"),
			},
			InputTenantID:                           "foo",
			InputDeviceID:                           "bar",
			InputDeploymentID:                       "baz",
			InputCreateConfigurationDeploymentError: errors.New("model error"),
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusInternalServerError,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("internal error")),
			},
		},
		"ko, conflict": {
			InputBodyObject: &model.ConfigurationDeploymentConstructor{
				Name:          "foo",
				Configuration: []byte("bar"),
			},
			InputTenantID:                           "foo",
			InputDeviceID:                           "bar",
			InputDeploymentID:                       "baz",
			InputCreateConfigurationDeploymentError: app.ErrDuplicateDeployment,
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusConflict,
				OutputBodyObject: h.ErrorToErrStruct(app.ErrDuplicateDeployment),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(fmt.Sprintf("test case: %s", name), func(t *testing.T) {
			restView := new(view.RESTView)
			app := &mapp.App{}

			d := NewDeploymentsApiHandlers(nil, restView, app)

			app.On("CreateDeviceConfigurationDeployment",
				h.ContextMatcher(), mock.AnythingOfType("*model.ConfigurationDeploymentConstructor"),
				tc.InputDeviceID, tc.InputDeploymentID).
				Return(tc.InputDeploymentID, tc.InputCreateConfigurationDeploymentError)

			api := setUpRestTest(
				ApiUrlInternalDeviceConfigurationDeployments,
				rest.Post,
				d.PostDeviceConfigurationDeployment,
			)

			uri := strings.Replace(ApiUrlInternalDeviceConfigurationDeployments, "#tenant", tc.InputTenantID, 1)
			uri = strings.Replace(uri, "#device_id", tc.InputDeviceID, 1)
			uri = strings.Replace(uri, "#deployment_id", tc.InputDeploymentID, 1)

			req := test.MakeSimpleRequest("POST", "http://localhost"+uri, tc.InputBodyObject)
			req.Header.Add(requestid.RequestIdHeader, "test")

			recorded := test.RunRequest(t, api.MakeHandler(), req)

			h.CheckRecordedResponse(t, recorded, tc.JSONResponseParams)
		})
	}
}

type brokenReader struct{}

func (r brokenReader) Read(b []byte) (int, error) {
	return 0, errors.New("rekt")
}

func TestDownloadConfiguration(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name string

		Config  *Config
		Request *http.Request
		App     *mapp.App // mock App

		// Response parameters
		StatusCode int    // Response StatusCode
		Error      error  // Error message in case of non-2XX response.
		Body       []byte // The Body on 2XX responses.
		Headers    http.Header
	}{{
		Name: "ok",

		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				FMTConfigURL(
					"http", "localhost",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
					"Bagelbone",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				),
				nil,
			)
			sig := model.NewRequestSignature(req, []byte("test"))
			sig.SetExpire(time.Now().Add(time.Minute))
			signature := sig.HMAC256()
			q := req.URL.Query()
			q.Set(
				model.ParamSignature,
				base64.RawURLEncoding.EncodeToString(signature))
			req.URL.RawQuery = q.Encode()
			return req
		}(),
		Config: NewConfig().
			SetPresignExpire(time.Minute).
			SetPresignSecret([]byte("test")).
			SetPresignHostname("localhost").
			SetPresignScheme("http"),
		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("GenerateConfigurationImage",
				contextMatcher(),
				"Bagelbone",
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
			).Return(bytes.NewReader([]byte("*Just imagine an artifact here*")), nil)
			return app
		}(),

		Headers: http.Header{
			"Content-Disposition": []string{"attachment; filename=\"artifact.mender\""},
			"Content-Type":        []string{app.ArtifactContentType},
			"Content-Length":      []string{"31"},
		},
		StatusCode: http.StatusOK,
		Body:       []byte("*Just imagine an artifact here*"),
	}, {
		Name: "ok, multi-tenant",

		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				FMTConfigURL(
					"http", "localhost",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
					"Bagelbone",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				),
				nil,
			)
			sig := model.NewRequestSignature(req, []byte("test"))
			sig.SetExpire(time.Now().Add(time.Minute))
			q := req.URL.Query()
			q.Set("tenant_id", "123456789012345678901234")
			req.URL.RawQuery = q.Encode()
			signature := sig.HMAC256()
			q.Set(
				model.ParamSignature,
				base64.RawURLEncoding.EncodeToString(signature))
			req.URL.RawQuery = q.Encode()
			return req
		}(),
		Config: NewConfig().
			SetPresignExpire(time.Minute).
			SetPresignSecret([]byte("test")).
			SetPresignHostname("localhost").
			SetPresignScheme("http"),
		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("GenerateConfigurationImage",
				contextMatcher(),
				"Bagelbone",
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
			).Return(bytes.NewReader([]byte("*Just imagine an artifact here*")), nil)
			return app
		}(),

		Headers: http.Header{
			"Content-Disposition": []string{"attachment; filename=\"artifact.mender\""},
			"Content-Type":        []string{app.ArtifactContentType},
			"Content-Length":      []string{"31"},
		},
		StatusCode: http.StatusOK,
		Body:       []byte("*Just imagine an artifact here*"),
	}, {
		Name: "error, signing configured incorrectly",

		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				FMTConfigURL(
					"http", "localhost",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
					"Bagelbone",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				),
				nil,
			)
			return req
		}(),
		App: new(mapp.App),

		StatusCode: http.StatusNotFound,
		Error:      errors.New("Resource not found"),
	}, {
		Name: "error, invalid request",

		Config: NewConfig().
			SetPresignSecret([]byte("test")),
		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				FMTConfigURL(
					"http", "localhost",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
					"Bagelbone",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				),
				nil,
			)
			return req
		}(),
		App: new(mapp.App),

		StatusCode: http.StatusBadRequest,
		Error: errors.New("invalid request parameters: " +
			"x-men-expire: required key is missing; " +
			"x-men-signature: required key is missing.",
		),
	}, {
		Name: "error, signature expired",

		Config: NewConfig().
			SetPresignSecret([]byte("test")),
		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				FMTConfigURL(
					"http", "localhost",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
					"Bagelbone",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				),
				nil,
			)
			sig := model.NewRequestSignature(req, []byte("test"))
			sig.SetExpire(time.Now().Add(-time.Second))
			sig.PresignURL()
			return req
		}(),
		App: new(mapp.App),

		StatusCode: http.StatusForbidden,
		Error:      model.ErrLinkExpired,
	}, {
		Name: "error, signature invalid",

		Config: NewConfig().
			SetPresignSecret([]byte("test")),
		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				FMTConfigURL(
					"http", "localhost",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
					"Bagelbone",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				),
				nil,
			)
			sig := model.NewRequestSignature(req, []byte("wrong_key"))
			sig.SetExpire(time.Now().Add(time.Minute))
			sig.PresignURL()
			return req
		}(),
		App: new(mapp.App),

		StatusCode: http.StatusForbidden,
		Error:      errors.New("signature invalid"),
	}, {
		Name: "error, deployment not found",

		Config: NewConfig().
			SetPresignSecret([]byte("test")),
		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				FMTConfigURL(
					"http", "localhost",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
					"Bagelbone",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				),
				nil,
			)
			sig := model.NewRequestSignature(req, []byte("test"))
			sig.SetExpire(time.Now().Add(time.Minute))
			sig.PresignURL()
			return req
		}(),
		App: func() *mapp.App {
			appl := new(mapp.App)
			appl.On("GenerateConfigurationImage",
				contextMatcher(),
				"Bagelbone",
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
			).Return(nil, app.ErrModelDeploymentNotFound)
			return appl
		}(),

		StatusCode: http.StatusNotFound,
		Error: errors.Errorf(
			"deployment with id '%s' not found",
			uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")),
		),
	}, {
		Name: "error, internal error",

		Config: NewConfig().
			SetPresignSecret([]byte("test")),
		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				FMTConfigURL(
					"http", "localhost",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
					"Bagelbone",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				),
				nil,
			)
			sig := model.NewRequestSignature(req, []byte("test"))
			sig.SetExpire(time.Now().Add(time.Minute))
			sig.PresignURL()
			return req
		}(),
		App: func() *mapp.App {
			appl := new(mapp.App)
			appl.On("GenerateConfigurationImage",
				contextMatcher(),
				"Bagelbone",
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
			).Return(nil, errors.New("internal error"))
			return appl
		}(),

		StatusCode: http.StatusInternalServerError,
		Error:      errors.New("internal error"),
	}, {
		Name: "error, broken artifact reader",
		Config: NewConfig().
			SetPresignSecret([]byte("test")),
		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				FMTConfigURL(
					"http", "localhost",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
					"Bagelbone",
					uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				),
				nil,
			)
			sig := model.NewRequestSignature(req, []byte("test"))
			sig.SetExpire(time.Now().Add(time.Minute))
			sig.PresignURL()
			return req
		}(),
		App: func() *mapp.App {
			appl := new(mapp.App)
			appl.On("GenerateConfigurationImage",
				contextMatcher(),
				"Bagelbone",
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("deployment")).String(),
			).Return(brokenReader{}, nil)
			return appl
		}(),

		StatusCode: http.StatusInternalServerError,
		Error:      errors.New("internal error"),
	}}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			defer tc.App.AssertExpectations(t)
			reqClone := tc.Request.Clone(context.Background())
			handlers := NewDeploymentsApiHandlers(nil, &view.RESTView{}, tc.App, tc.Config)
			routes := NewDeploymentsResourceRoutes(handlers)
			router, _ := rest.MakeRouter(routes...)
			api := rest.NewApi()
			api.SetApp(router)
			handler := api.MakeHandler()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, tc.Request)

			assert.Equal(t, tc.StatusCode, w.Code)
			if tc.Error != nil {
				var apiErr rest_utils.ApiError
				err := json.Unmarshal(w.Body.Bytes(), &apiErr)
				if assert.NoError(t, err) {
					assert.EqualError(t, &apiErr, tc.Error.Error())
				}
			} else {
				assert.Equal(t, w.Body.Bytes(), tc.Body)
				model.NewRequestSignature(reqClone, []byte("test"))
				rspHdr := w.Header()
				for key := range tc.Headers {
					if assert.Contains(t,
						rspHdr,
						key,
						"missing expected header",
					) {
						assert.Equal(t,
							tc.Headers.Get(key),
							rspHdr.Get(key),
						)
					}
				}
			}
		})
	}
}

func TestGetDeploymentForDevice(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name string

		Request        *http.Request
		App            *mapp.App
		IsConfig       bool
		XForwardedHost string

		StatusCode int
		Error      error
	}{{
		Name: "ok",

		Request: func() *http.Request {
			req, _ := http.NewRequestWithContext(
				identity.WithContext(context.Background(), &identity.Identity{
					Subject:  uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
					IsDevice: true,
				}),
				http.MethodGet,
				"http://localhost"+ApiUrlDevicesDeploymentsNext+
					"?device_type=bagelShins&artifact_name=bagelOS1.0.1",
				nil,
			)
			return req
		}(),
		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("GetDeploymentForDeviceWithCurrent",
				contextMatcher(),
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				&model.DeploymentNextRequest{
					DeviceProvides: &model.InstalledDeviceDeployment{
						ArtifactName: "bagelOS1.0.1",
						DeviceType:   "bagelShins",
					},
				},
			).Return(&model.DeploymentInstructions{
				ID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("deployment")).String(),
				Artifact: model.ArtifactDeploymentInstructions{
					ArtifactName:          "bagelOS1.1.0",
					DeviceTypesCompatible: []string{"bagelShins", "raspberryPlanck"},
					Source: model.Link{
						Uri:    "https://localhost/bucket/head/bagelOS1.0.1",
						Expire: time.Now().Add(time.Hour),
					},
				},
			}, nil)
			return app
		}(),

		StatusCode: http.StatusOK,
		Error:      nil,
	}, {
		Name: "ok, POST",

		Request: func() *http.Request {
			b, _ := json.Marshal(model.InstalledDeviceDeployment{
				ArtifactName: "bagelOS1.0.1",
				DeviceType:   "bagelBone",
			})
			req, _ := http.NewRequestWithContext(
				identity.WithContext(context.Background(), &identity.Identity{
					Subject:  uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
					IsDevice: true,
				}),
				http.MethodPost,
				"http://localhost"+ApiUrlDevicesDeploymentsNext+
					"?device_type=bagelShins&artifact_name=bagelOS1.0.1",
				bytes.NewReader(b),
			)
			return req
		}(),
		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("GetDeploymentForDeviceWithCurrent",
				contextMatcher(),
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				&model.DeploymentNextRequest{
					DeviceProvides: &model.InstalledDeviceDeployment{
						ArtifactName: "bagelOS1.0.1",
						DeviceType:   "bagelBone",
					},
				},
			).Return(&model.DeploymentInstructions{
				ID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("deployment")).String(),
				Artifact: model.ArtifactDeploymentInstructions{
					ArtifactName:          "bagelOS1.1.0",
					DeviceTypesCompatible: []string{"bagelBone"},
					Source: model.Link{
						Uri:    "https://localhost/bucket/head/bagelOS1.0.1",
						Expire: time.Now().Add(time.Hour),
					},
				},
			}, nil)
			return app
		}(),

		StatusCode: http.StatusOK,
		Error:      nil,
	}, {
		Name: "ok, configuration deployment",

		Request: func() *http.Request {
			req, _ := http.NewRequestWithContext(
				identity.WithContext(context.Background(), &identity.Identity{
					Subject:  uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
					IsDevice: true,
				}),
				http.MethodGet,
				"http://localhost"+ApiUrlDevicesDeploymentsNext+
					"?device_type=bagelShins&artifact_name=bagelOS1.0.1",
				nil,
			)
			return req
		}(),
		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("GetDeploymentForDeviceWithCurrent",
				contextMatcher(),
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				&model.DeploymentNextRequest{
					DeviceProvides: &model.InstalledDeviceDeployment{
						ArtifactName: "bagelOS1.0.1",
						DeviceType:   "bagelShins",
					},
				},
			).Return(&model.DeploymentInstructions{
				ID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("deployment")).String(),
				Artifact: model.ArtifactDeploymentInstructions{
					ArtifactName:          "bagelOS1.1.0",
					DeviceTypesCompatible: []string{"bagelShins", "raspberryPlanck"},
				},
				Type: model.DeploymentTypeConfiguration,
			}, nil)
			return app
		}(),
		IsConfig: true,

		StatusCode: http.StatusOK,
		Error:      nil,
	}, {
		Name: "ok, configuration deployment w/tenant",

		Request: func() *http.Request {
			req, _ := http.NewRequestWithContext(
				identity.WithContext(context.Background(), &identity.Identity{
					Subject:  uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
					IsDevice: true,
					Tenant:   "12456789012345678901234",
				}),
				http.MethodGet,
				"http://localhost"+ApiUrlDevicesDeploymentsNext+
					"?device_type=bagelShins&artifact_name=bagelOS1.0.1",
				nil,
			)
			return req
		}(),
		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("GetDeploymentForDeviceWithCurrent",

				contextMatcher(),
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				&model.DeploymentNextRequest{
					DeviceProvides: &model.InstalledDeviceDeployment{
						ArtifactName: "bagelOS1.0.1",
						DeviceType:   "bagelShins",
					},
				},
			).Return(&model.DeploymentInstructions{
				ID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("deployment")).String(),
				Artifact: model.ArtifactDeploymentInstructions{
					ArtifactName:          "bagelOS1.1.0",
					DeviceTypesCompatible: []string{"bagelShins", "raspberryPlanck"},
				},
				Type: model.DeploymentTypeConfiguration,
			}, nil)
			return app
		}(),
		IsConfig: true,

		StatusCode: http.StatusOK,
		Error:      nil,
	}, {
		Name:           "ok, configuration deployment with X-Forwarded-Host",
		XForwardedHost: "hosted.mender.io",

		Request: func() *http.Request {
			req, _ := http.NewRequestWithContext(
				identity.WithContext(context.Background(), &identity.Identity{
					Subject:  uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
					IsDevice: true,
					Tenant:   "12456789012345678901234",
				}),
				http.MethodGet,
				"http://localhost"+ApiUrlDevicesDeploymentsNext+
					"?device_type=bagelShins&artifact_name=bagelOS1.0.1",
				nil,
			)
			req.Header.Add(hdrForwardedHost, "hosted.mender.io")
			return req
		}(),
		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("GetDeploymentForDeviceWithCurrent",

				contextMatcher(),
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				&model.DeploymentNextRequest{
					DeviceProvides: &model.InstalledDeviceDeployment{
						ArtifactName: "bagelOS1.0.1",
						DeviceType:   "bagelShins",
					},
				},
			).Return(&model.DeploymentInstructions{
				ID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("deployment")).String(),
				Artifact: model.ArtifactDeploymentInstructions{
					ArtifactName:          "bagelOS1.1.0",
					DeviceTypesCompatible: []string{"bagelShins", "raspberryPlanck"},
				},
				Type: model.DeploymentTypeConfiguration,
			}, nil)
			return app
		}(),
		IsConfig: true,

		StatusCode: http.StatusOK,
		Error:      nil,
	}, {
		Name:           "ko, configuration deployment without X-Forwarded-Host nor presign host config",
		XForwardedHost: "hosted.mender.io",

		Request: func() *http.Request {
			req, _ := http.NewRequestWithContext(
				identity.WithContext(context.Background(), &identity.Identity{
					Subject:  uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
					IsDevice: true,
					Tenant:   "12456789012345678901234",
				}),
				http.MethodGet,
				"http://localhost"+ApiUrlDevicesDeploymentsNext+
					"?device_type=bagelShins&artifact_name=bagelOS1.0.1",
				nil,
			)
			return req
		}(),
		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("GetDeploymentForDeviceWithCurrent",

				contextMatcher(),
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				&model.DeploymentNextRequest{
					DeviceProvides: &model.InstalledDeviceDeployment{
						ArtifactName: "bagelOS1.0.1",
						DeviceType:   "bagelShins",
					},
				},
			).Return(&model.DeploymentInstructions{
				ID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("deployment")).String(),
				Artifact: model.ArtifactDeploymentInstructions{
					ArtifactName:          "bagelOS1.1.0",
					DeviceTypesCompatible: []string{"bagelShins", "raspberryPlanck"},
				},
				Type: model.DeploymentTypeConfiguration,
			}, nil)
			return app
		}(),
		IsConfig: true,

		StatusCode: http.StatusInternalServerError,
		Error:      errors.New("internal error"),
	}, {
		Name: "error, missing identity",

		Request: func() *http.Request {
			req, _ := http.NewRequest(
				http.MethodGet,
				"http://localhost"+ApiUrlDevicesDeploymentsNext+
					"?device_type=bagelShins&artifact_name=bagelOS1.0.1",
				nil,
			)
			return req
		}(),
		App: new(mapp.App),

		StatusCode: http.StatusBadRequest,
		Error:      ErrMissingIdentity,
	}, {
		Name: "error, invalid POST schema",

		Request: func() *http.Request {
			req, _ := http.NewRequestWithContext(
				identity.WithContext(context.Background(), &identity.Identity{
					Subject:  uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
					IsDevice: true,
					Tenant:   "12456789012345678901234",
				}),
				http.MethodPost,
				"http://localhost"+ApiUrlDevicesDeploymentsNext,
				bytes.NewReader([]byte("Lorem ipsum...")),
			)
			return req
		}(),
		App: new(mapp.App),

		StatusCode: http.StatusBadRequest,
		Error:      errors.New("invalid schema: invalid character 'L' looking for beginning of value"),
	}, {
		Name: "error, missing parameters",

		Request: func() *http.Request {
			req, _ := http.NewRequestWithContext(
				identity.WithContext(context.Background(), &identity.Identity{
					Subject:  uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
					IsDevice: true,
					Tenant:   "12456789012345678901234",
				}),
				http.MethodGet,
				"http://localhost"+ApiUrlDevicesDeploymentsNext,
				nil,
			)
			return req
		}(),
		App: new(mapp.App),

		StatusCode: http.StatusBadRequest,
		Error:      errors.New("artifact_name: cannot be blank; device_type: cannot be blank."),
	}, {
		Name: "error, internal app error",

		Request: func() *http.Request {
			req, _ := http.NewRequestWithContext(
				identity.WithContext(context.Background(), &identity.Identity{
					Subject:  uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
					IsDevice: true,
					Tenant:   "12456789012345678901234",
				}),
				http.MethodGet,
				"http://localhost"+ApiUrlDevicesDeploymentsNext+
					"?device_type=bagelShins&artifact_name=bagelOS1.0.1",
				nil,
			)
			return req
		}(),
		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("GetDeploymentForDeviceWithCurrent",

				contextMatcher(),
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				&model.DeploymentNextRequest{
					DeviceProvides: &model.InstalledDeviceDeployment{
						ArtifactName: "bagelOS1.0.1",
						DeviceType:   "bagelShins",
					},
				},
			).Return(nil, errors.New("mongo: internal error"))
			return app
		}(),

		StatusCode: http.StatusInternalServerError,
		Error:      errors.New("internal error"),
	}, {
		Name: "error, internal app error",

		Request: func() *http.Request {
			req, _ := http.NewRequestWithContext(
				identity.WithContext(context.Background(), &identity.Identity{
					Subject:  uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
					IsDevice: true,
					Tenant:   "12456789012345678901234",
				}),
				http.MethodGet,
				"http://localhost"+ApiUrlDevicesDeploymentsNext+
					"?device_type=bagelShins&artifact_name=bagelOS1.0.1",
				nil,
			)
			return req
		}(),
		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("GetDeploymentForDeviceWithCurrent",

				contextMatcher(),
				uuid.NewSHA1(uuid.NameSpaceOID, []byte("device")).String(),
				&model.DeploymentNextRequest{
					DeviceProvides: &model.InstalledDeviceDeployment{
						ArtifactName: "bagelOS1.0.1",
						DeviceType:   "bagelShins",
					},
				},
			).Return(nil, nil)
			return app
		}(),

		StatusCode: http.StatusNoContent,
	}}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			defer tc.App.AssertExpectations(t)
			config := NewConfig().
				SetPresignScheme("https").
				SetPresignSecret([]byte("test")).
				SetPresignExpire(time.Hour)
			if tc.XForwardedHost == "" {
				config = config.SetPresignHostname("localhost")
			}

			handlers := NewDeploymentsApiHandlers(nil, &view.RESTView{}, tc.App, config)
			routes := NewDeploymentsResourceRoutes(handlers)
			router, _ := rest.MakeRouter(routes...)
			api := rest.NewApi()
			api.SetApp(router)
			handler := api.MakeHandler()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, tc.Request)

			assert.Equal(t, tc.StatusCode, w.Code)
			if tc.Error != nil {
				var apiErr rest_utils.ApiError
				err := json.Unmarshal(w.Body.Bytes(), &apiErr)
				if assert.NoError(t, err) {
					assert.EqualError(t, &apiErr, tc.Error.Error())
				}
			} else if tc.StatusCode == 204 {
				assert.Equal(t, []byte(nil), w.Body.Bytes())
			} else {
				if !assert.NotNil(t, w.Body.Bytes()) {
					return
				}
				var instr model.DeploymentInstructions
				json.Unmarshal(w.Body.Bytes(), &instr) //nolint: errcheck
				link, err := url.Parse(instr.Artifact.Source.Uri)
				if tc.IsConfig {
					assert.NoError(t, err)
					assert.Equal(t, "https", link.Scheme)
					if tc.XForwardedHost != "" {
						assert.Equal(t, tc.XForwardedHost, link.Host)
					} else {
						assert.Equal(t, "localhost", link.Host)
					}
					q := link.Query()
					expire, err := time.Parse(time.RFC3339, q.Get(model.ParamExpire))
					if assert.NoError(t, err) {
						assert.WithinDuration(t, time.Now().Add(time.Hour), expire, time.Minute)
					}
				}
				assert.WithinDuration(t, time.Now().Add(time.Hour), instr.Artifact.Source.Expire, time.Minute)
			}
		})
	}
}

func TestGetTenantStorageSettings(t *testing.T) {
	testCases := map[string]struct {
		tenantID   string
		settings   *model.StorageSettings
		err        error
		httpStatus int
	}{
		"ok": {
			tenantID: "",
			settings: &model.StorageSettings{
				Region: "region",
				Key:    "key",
				Secret: "secret",
				Bucket: "bucket",
			},
			httpStatus: http.StatusOK,
		},
		"ok multi-tenant": {
			tenantID: "tenant1",
			settings: &model.StorageSettings{
				Region: "region",
				Key:    "key",
				Secret: "secret",
				Bucket: "bucket",
			},
			httpStatus: http.StatusOK,
		},
		"error": {
			tenantID:   "",
			err:        errors.New("generic error"),
			httpStatus: http.StatusInternalServerError,
		},
		"error multi-tenant": {
			tenantID:   "tenant1",
			err:        errors.New("generic error"),
			httpStatus: http.StatusInternalServerError,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			app := &mapp.App{}
			app.On("GetStorageSettings",
				mock.MatchedBy(func(ctx context.Context) bool { return true }),
			).Return(tc.settings, tc.err)

			restView := new(view.RESTView)
			d := NewDeploymentsApiHandlers(nil, restView, app)
			api := setUpRestTest(
				ApiUrlInternalTenantStorageSettings,
				rest.Get,
				d.GetTenantStorageSettingsHandler,
			)
			url := strings.Replace(ApiUrlInternalTenantStorageSettings, "#tenant", tc.tenantID, -1)
			req, _ := http.NewRequest(
				"GET",
				"http://localhost"+url,
				nil,
			)
			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.httpStatus)

			if tc.httpStatus == http.StatusOK {
				settings := &model.StorageSettings{}
				err := json.Unmarshal(recorded.Recorder.Body.Bytes(), settings)
				assert.NoError(t, err)
				assert.Equal(t, settings, tc.settings)
			}
		})
	}
}

func TestPutTenantStorageSettings(t *testing.T) {
	testCases := map[string]struct {
		tenantID   string
		settings   *model.StorageSettings
		err        error
		httpStatus int
	}{
		"ok": {
			tenantID: "",
			settings: &model.StorageSettings{
				Region: "region",
				Key:    "secretkey",
				Secret: "secret",
				Bucket: "bucket",
				Uri:    "https://example.com",
				Token:  "token",
			},
			httpStatus: http.StatusNoContent,
		},
		"ok external-uri": {
			tenantID: "",
			settings: &model.StorageSettings{
				Region:      "region",
				Key:         "secretkey",
				Secret:      "secret",
				Bucket:      "bucket",
				Uri:         "https://example.com",
				ExternalUri: "https://external.example.com",
				Token:       "token",
			},
			httpStatus: http.StatusNoContent,
		},
		"ok multi-tenant": {
			tenantID: "tenant1",
			settings: &model.StorageSettings{
				Region:      "region",
				Key:         "secretkey",
				Secret:      "secret",
				Bucket:      "bucket",
				Uri:         "https://example.com",
				ExternalUri: "https://external.example.com",
				Token:       "token",
			},
			httpStatus: http.StatusNoContent,
		},
		"error no data": {
			tenantID:   "",
			settings:   nil,
			httpStatus: http.StatusNoContent,
		},
		"error no data multi-tenant": {
			tenantID:   "tenant1",
			settings:   nil,
			httpStatus: http.StatusNoContent,
		},
		"error invalid data": {
			tenantID: "",
			settings: &model.StorageSettings{
				Region: "region",
				Key:    "secretkey",
				Bucket: "bucket",
			},
			httpStatus: http.StatusBadRequest,
		},
		"error invalid data multi-tenant": {
			tenantID: "tenant1",
			settings: &model.StorageSettings{
				Region: "region",
				Key:    "secretkey",
				Bucket: "bucket",
			},
			httpStatus: http.StatusBadRequest,
		},
		"error app err": {
			tenantID: "",
			settings: &model.StorageSettings{
				Region:      "region",
				Key:         "secretkey",
				Secret:      "secret",
				Bucket:      "bucket",
				Uri:         "https://example.com",
				ExternalUri: "https://external.example.com",
				Token:       "token",
			},
			err:        errors.New("generic error"),
			httpStatus: http.StatusInternalServerError,
		},
		"error app err multi-tenant": {
			tenantID: "tenant1",
			settings: &model.StorageSettings{
				Region:      "region",
				Key:         "secretkey",
				Secret:      "secret",
				Bucket:      "bucket",
				Uri:         "https://example.com",
				ExternalUri: "https://external.example.com",
				Token:       "token",
			},
			err:        errors.New("generic error"),
			httpStatus: http.StatusInternalServerError,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			app := &mapp.App{}
			app.On("SetStorageSettings",
				mock.MatchedBy(func(ctx context.Context) bool { return true }),
				tc.settings,
			).Return(tc.err)

			restView := new(view.RESTView)
			d := NewDeploymentsApiHandlers(nil, restView, app)
			api := setUpRestTest(
				ApiUrlInternalTenantStorageSettings,
				rest.Put,
				d.PutTenantStorageSettingsHandler,
			)
			body, _ := json.Marshal(tc.settings)
			url := strings.Replace(ApiUrlInternalTenantStorageSettings, "#tenant", tc.tenantID, -1)
			req, _ := http.NewRequest(
				http.MethodPut,
				"http://localhost"+url,
				bytes.NewBuffer(body),
			)

			recorded := test.RunRequest(t, api.MakeHandler(), req)
			if recorded.Recorder.Code != tc.httpStatus {
				fmt.Println(recorded.Recorder.Body)
			}
			recorded.CodeIs(tc.httpStatus)
		})
	}
}

func TestLookupDeployment(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Name         string
		appError     error
		query        *model.Query
		deployments  []*model.Deployment
		count        int64
		sort         string
		ResponseCode int
	}{
		{
			Name: "ok, discending",
			query: &model.Query{
				Limit: rest_utils.PerPageDefault + 1,
				Sort:  model.SortDirectionDescending,
			},
			deployments:  []*model.Deployment{},
			count:        0,
			sort:         model.SortDirectionDescending,
			ResponseCode: http.StatusOK,
		},
		{
			Name: "ok, ascending",
			query: &model.Query{
				Limit: rest_utils.PerPageDefault + 1,
				Sort:  model.SortDirectionAscending,
			},
			deployments:  []*model.Deployment{},
			count:        0,
			sort:         model.SortDirectionAscending,
			ResponseCode: http.StatusOK,
		},
		{
			Name: "ok, default",
			query: &model.Query{
				Limit: rest_utils.PerPageDefault + 1,
				Sort:  model.SortDirectionDescending,
			},
			deployments:  []*model.Deployment{},
			count:        0,
			ResponseCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			app := &mapp.App{}
			app.On("LookupDeployment",
				mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}),
				*tc.query,
			).Return(tc.deployments, tc.count, tc.appError)
			restView := new(view.RESTView)
			d := NewDeploymentsApiHandlers(nil, restView, app)
			api := setUpRestTest(
				ApiUrlManagementDeployments,
				rest.Get,
				d.LookupDeployment,
			)
			url := "http://localhost" + ApiUrlManagementDeployments
			if tc.sort != "" {
				url = "http://localhost" + ApiUrlManagementDeployments + "?sort=" + tc.sort
			}
			req := test.MakeSimpleRequest(
				"GET",
				url,
				"",
			)
			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.ResponseCode)
		})
	}
}

func TestAbortDeviceDeployments(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		deviceID                  string
		abortDeviceDeploymentsErr error
		responseCode              int
	}{
		"ok": {
			deviceID:     "1",
			responseCode: http.StatusNoContent,
		},
		"ok, not found": {
			deviceID:                  "1",
			abortDeviceDeploymentsErr: app.ErrStorageNotFound,
			responseCode:              http.StatusNoContent,
		},
		"ko": {
			deviceID:                  "1",
			abortDeviceDeploymentsErr: errors.New("internal error"),
			responseCode:              http.StatusInternalServerError,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			app := &mapp.App{}
			app.On("AbortDeviceDeployments",
				mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}),
				tc.deviceID,
			).Return(tc.abortDeviceDeploymentsErr)

			restView := new(view.RESTView)
			d := NewDeploymentsApiHandlers(nil, restView, app)
			api := setUpRestTest(
				ApiUrlManagementDeploymentsDeviceId,
				rest.Delete,
				d.AbortDeviceDeployments,
			)
			url := "http://localhost" + ApiUrlManagementDeploymentsDeviceId
			url = strings.Replace(url, "#id", tc.deviceID, 1)
			req := test.MakeSimpleRequest("DELETE", url, "")

			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.responseCode)
		})
	}
}

func TestDeleteDeviceDeploymentsHistory(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		deviceID                   string
		deleteDeviceDeploymentsErr error
		responseCode               int
	}{
		"ok": {
			deviceID:     "1",
			responseCode: http.StatusNoContent,
		},
		"ok, not found": {
			deviceID:                   "1",
			deleteDeviceDeploymentsErr: app.ErrStorageNotFound,
			responseCode:               http.StatusNoContent,
		},
		"ko": {
			deviceID:                   "1",
			deleteDeviceDeploymentsErr: errors.New("internal error"),
			responseCode:               http.StatusInternalServerError,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			app := &mapp.App{}
			app.On("DeleteDeviceDeploymentsHistory",
				mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}),
				tc.deviceID,
			).Return(tc.deleteDeviceDeploymentsErr)

			restView := new(view.RESTView)
			d := NewDeploymentsApiHandlers(nil, restView, app)
			api := setUpRestTest(
				ApiUrlManagementDeploymentsDeviceHistory,
				rest.Delete,
				d.DeleteDeviceDeploymentsHistory,
			)
			url := "http://localhost" + ApiUrlManagementDeploymentsDeviceHistory
			url = strings.Replace(url, "#id", tc.deviceID, 1)
			req := test.MakeSimpleRequest("DELETE", url, "")

			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.responseCode)
		})
	}
}

func TestGetDeploymentsStats(t *testing.T) {
	t.Parallel()

	testSHA := uuid.NewSHA1(uuid.NameSpaceOID, []byte("deploymentid1")).String()

	testCases := map[string]struct {
		deploymentIDs         model.DeploymentIDs
		responseCode          int
		mockedDeploymentStats []*model.DeploymentStats
		mockedError           error
	}{
		"OK - default success case": {
			deploymentIDs: model.DeploymentIDs{[]string{testSHA}},
			responseCode:  http.StatusOK,
			mockedDeploymentStats: []*model.DeploymentStats{
				{
					ID:    testSHA,
					Stats: model.NewDeviceDeploymentStats(),
				},
			},
		},
		"Error - malformed UUID": {
			deploymentIDs: model.DeploymentIDs{[]string{"imnotauuid"}},
			responseCode:  http.StatusBadRequest,
			mockedError:   nil,
		},
		"Error - database error": {
			deploymentIDs: model.DeploymentIDs{[]string{testSHA}},
			responseCode:  http.StatusInternalServerError,
			mockedError:   errors.New("checking deployment statistics for IDs"),
		},
		"Error - no deploymentStats found": {
			deploymentIDs: model.DeploymentIDs{[]string{testSHA}},
			responseCode:  http.StatusNotFound,
			mockedError:   app.ErrModelDeploymentNotFound,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			app := &mapp.App{}
			app.On("GetDeploymentsStats",
				mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}),
				tc.deploymentIDs.IDs[0],
			).Return(tc.mockedDeploymentStats, tc.mockedError)

			restView := new(view.RESTView)
			d := NewDeploymentsApiHandlers(nil, restView, app)
			api := setUpRestTest(
				ApiUrlManagementMultipleDeploymentsStatistics,
				rest.Post,
				d.GetDeploymentsStats,
			)
			url := "http://localhost" + ApiUrlManagementMultipleDeploymentsStatistics
			req := test.MakeSimpleRequest("POST", url, tc.deploymentIDs)

			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.responseCode)
			recorded.ContentTypeIsJson()
			res := []*model.DeploymentStats{}
			recorded.DecodeJsonPayload(&res)
			if tc.responseCode == http.StatusOK {
				assert.Equal(t, tc.mockedDeploymentStats, res, "Unexpected response body")
			}
		})
	}
}

func str2ptr(s string) *string {
	return &s
}

func TestListDeviceDeployments(t *testing.T) {
	const deviceID = "d50eda0d-2cea-4de1-8d42-9cd3e7e86701"
	t.Parallel()
	testCases := map[string]struct {
		deviceID     string
		status       string
		limit        int
		query        *store.ListQueryDeviceDeployments
		responseCode int
		deployments  []model.DeviceDeploymentListItem
		count        int
		err          error
	}{
		"ok": {
			deviceID: deviceID,
			query: &store.ListQueryDeviceDeployments{
				DeviceID: deviceID,
				Limit:    DefaultPerPage,
			},
			responseCode: http.StatusOK,
			deployments: []model.DeviceDeploymentListItem{
				{
					Id: "d50eda0d-2cea-4de1-8d42-9cd3e7e86701",
				},
			},
			count: 1,
		},
		"ok, device ID not UUID": {
			deviceID: "foo",
			query: &store.ListQueryDeviceDeployments{
				DeviceID: "foo",
				Limit:    DefaultPerPage,
			},
			responseCode: http.StatusOK,
			deployments: []model.DeviceDeploymentListItem{
				{
					Id: "d50eda0d-2cea-4de1-8d42-9cd3e7e86701",
				},
			},
			count: 1,
		},
		"ok, no records": {
			deviceID: deviceID,
			query: &store.ListQueryDeviceDeployments{
				DeviceID: deviceID,
				Limit:    DefaultPerPage,
			},
			responseCode: http.StatusOK,
			deployments:  []model.DeviceDeploymentListItem{},
			count:        0,
		},
		"ok, filter by status": {
			deviceID: deviceID,
			status:   "pending",
			query: &store.ListQueryDeviceDeployments{
				DeviceID: deviceID,
				Limit:    DefaultPerPage,
				Status:   str2ptr("pending"),
			},
			responseCode: http.StatusOK,
			deployments: []model.DeviceDeploymentListItem{
				{
					Id: "d50eda0d-2cea-4de1-8d42-9cd3e7e86701",
				},
			},
			count: 1,
		},
		"ok, custom limit": {
			deviceID: deviceID,
			limit:    10,
			query: &store.ListQueryDeviceDeployments{
				DeviceID: deviceID,
				Limit:    10,
			},
			responseCode: http.StatusOK,
			deployments: []model.DeviceDeploymentListItem{
				{
					Id: "d50eda0d-2cea-4de1-8d42-9cd3e7e86701",
				},
			},
			count: 1,
		},
		"ko, too high per_page": {
			deviceID:     deviceID,
			limit:        MaximumPerPageListDeviceDeployments + 1,
			responseCode: http.StatusBadRequest,
		},
		"ko, empty device ID": {
			deviceID:     "",
			responseCode: http.StatusNotFound,
			err:          errors.New("error"),
		},
		"ko, wrong limit": {
			deviceID:     deviceID,
			limit:        -10,
			responseCode: http.StatusBadRequest,
		},
		"ko, wrong status": {
			deviceID:     deviceID,
			status:       "dummy",
			responseCode: http.StatusBadRequest,
		},
		"ko, error": {
			deviceID: deviceID,
			query: &store.ListQueryDeviceDeployments{
				DeviceID: deviceID,
				Limit:    DefaultPerPage,
			},
			responseCode: http.StatusInternalServerError,
			count:        -1,
			err:          errors.New("error"),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			app := &mapp.App{}
			if tc.query != nil {
				app.On("GetDeviceDeploymentListForDevice",
					mock.MatchedBy(func(ctx context.Context) bool {
						return true
					}),
					*tc.query,
				).Return(
					tc.deployments,
					tc.count,
					tc.err,
				)
			}

			restView := new(view.RESTView)
			d := NewDeploymentsApiHandlers(nil, restView, app)
			api := setUpRestTest(
				ApiUrlManagementDeploymentsDeviceId,
				rest.Get,
				d.ListDeviceDeployments,
			)
			url := "http://localhost" + ApiUrlManagementDeploymentsDeviceId
			url = strings.Replace(url, "#id", tc.deviceID, 1)
			if tc.status != "" {
				url = url + "?status=" + tc.status
			}
			if tc.limit != 0 {
				url = url + fmt.Sprintf("?per_page=%d", tc.limit)
			}
			req := test.MakeSimpleRequest("GET", url, nil)

			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.responseCode)
			recorded.ContentTypeIsJson()
			res := []model.DeviceDeploymentListItem{}
			recorded.DecodeJsonPayload(&res)
			if tc.responseCode == http.StatusOK {
				assert.Equal(t, tc.deployments, res, "Unexpected response body")
			}
		})
	}
}

func TestListDeviceDeploymentsInternal(t *testing.T) {
	const deviceID = "d50eda0d-2cea-4de1-8d42-9cd3e7e86701"
	const tenantID = "tenant_id"
	t.Parallel()
	testCases := map[string]struct {
		deviceID     string
		status       string
		limit        int
		query        *store.ListQueryDeviceDeployments
		responseCode int
		deployments  []model.DeviceDeploymentListItem
		count        int
		err          error
	}{
		"ok": {
			deviceID: deviceID,
			query: &store.ListQueryDeviceDeployments{
				DeviceID: deviceID,
				Limit:    DefaultPerPage,
			},
			responseCode: http.StatusOK,
			deployments: []model.DeviceDeploymentListItem{
				{
					Id: "d50eda0d-2cea-4de1-8d42-9cd3e7e86701",
				},
			},
			count: 1,
		},
		"ok, device ID not UUID": {
			deviceID: "foo",
			query: &store.ListQueryDeviceDeployments{
				DeviceID: "foo",
				Limit:    DefaultPerPage,
			},
			responseCode: http.StatusOK,
			deployments: []model.DeviceDeploymentListItem{
				{
					Id: "d50eda0d-2cea-4de1-8d42-9cd3e7e86701",
				},
			},
			count: 1,
		},
		"ok, no records": {
			deviceID: deviceID,
			query: &store.ListQueryDeviceDeployments{
				DeviceID: deviceID,
				Limit:    DefaultPerPage,
			},
			responseCode: http.StatusOK,
			deployments:  []model.DeviceDeploymentListItem{},
			count:        0,
		},
		"ok, filter by status": {
			deviceID: deviceID,
			status:   "pending",
			query: &store.ListQueryDeviceDeployments{
				DeviceID: deviceID,
				Limit:    DefaultPerPage,
				Status:   str2ptr("pending"),
			},
			responseCode: http.StatusOK,
			deployments: []model.DeviceDeploymentListItem{
				{
					Id: "d50eda0d-2cea-4de1-8d42-9cd3e7e86701",
				},
			},
			count: 1,
		},
		"ok, custom limit": {
			deviceID: deviceID,
			limit:    10,
			query: &store.ListQueryDeviceDeployments{
				DeviceID: deviceID,
				Limit:    10,
			},
			responseCode: http.StatusOK,
			deployments: []model.DeviceDeploymentListItem{
				{
					Id: "d50eda0d-2cea-4de1-8d42-9cd3e7e86701",
				},
			},
			count: 1,
		},
		"ko, too high per_page": {
			deviceID:     deviceID,
			limit:        MaximumPerPageListDeviceDeployments + 1,
			responseCode: http.StatusBadRequest,
		},
		"ko, wrong device ID": {
			deviceID:     "",
			responseCode: http.StatusNotFound,
			err:          errors.New("error"),
		},
		"ok, wrong limit": {
			deviceID:     deviceID,
			limit:        -10,
			responseCode: http.StatusBadRequest,
		},
		"ok, wrong status": {
			deviceID:     deviceID,
			status:       "dummy",
			responseCode: http.StatusBadRequest,
		},
		"ko, error": {
			deviceID: deviceID,
			query: &store.ListQueryDeviceDeployments{
				DeviceID: deviceID,
				Limit:    DefaultPerPage,
			},
			responseCode: http.StatusInternalServerError,
			count:        -1,
			err:          errors.New("error"),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			app := &mapp.App{}
			if tc.query != nil {
				app.On("GetDeviceDeploymentListForDevice",
					mock.MatchedBy(func(ctx context.Context) bool {
						id := identity.FromContext(ctx)
						assert.NotNil(t, id)
						assert.Equal(t, tenantID, id.Tenant)
						return true
					}),
					*tc.query,
				).Return(
					tc.deployments,
					tc.count,
					tc.err,
				)
			}

			restView := new(view.RESTView)
			d := NewDeploymentsApiHandlers(nil, restView, app)
			api := setUpRestTest(
				ApiUrlInternalTenantDeploymentsDevice,
				rest.Get,
				d.ListDeviceDeploymentsInternal,
			)
			url := "http://localhost" + ApiUrlInternalTenantDeploymentsDevice
			url = strings.Replace(url, "#tenant", tenantID, 1)
			url = strings.Replace(url, "#id", tc.deviceID, 1)
			if tc.status != "" {
				url = url + "?status=" + tc.status
			}
			if tc.limit != 0 {
				url = url + fmt.Sprintf("?per_page=%d", tc.limit)
			}
			req := test.MakeSimpleRequest("GET", url, nil)

			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.responseCode)
			recorded.ContentTypeIsJson()
			res := []model.DeviceDeploymentListItem{}
			recorded.DecodeJsonPayload(&res)
			if tc.responseCode == http.StatusOK {
				assert.Equal(t, tc.deployments, res, "Unexpected response body")
			}
		})
	}
}

func TestListDeviceDeploymentsByIDsInternal(t *testing.T) {
	const ID = "d50eda0d-2cea-4de1-8d42-9cd3e7e86701"
	const tenantID = "tenant_id"
	t.Parallel()
	testCases := map[string]struct {
		ID           string
		status       string
		limit        int
		query        *store.ListQueryDeviceDeployments
		responseCode int
		deployments  []model.DeviceDeploymentListItem
		count        int
		err          error
	}{
		"ok": {
			ID: ID,
			query: &store.ListQueryDeviceDeployments{
				IDs:   []string{ID},
				Limit: DefaultPerPage,
			},
			responseCode: http.StatusOK,
			deployments: []model.DeviceDeploymentListItem{
				{
					Id: "d50eda0d-2cea-4de1-8d42-9cd3e7e86701",
				},
			},
			count: 1,
		},
		"ok, no records": {
			ID: ID,
			query: &store.ListQueryDeviceDeployments{
				IDs:   []string{ID},
				Limit: DefaultPerPage,
			},
			responseCode: http.StatusOK,
			deployments:  []model.DeviceDeploymentListItem{},
			count:        0,
		},
		"ok, filter by status": {
			ID:     ID,
			status: "pending",
			query: &store.ListQueryDeviceDeployments{
				IDs:    []string{ID},
				Limit:  DefaultPerPage,
				Status: str2ptr("pending"),
			},
			responseCode: http.StatusOK,
			deployments: []model.DeviceDeploymentListItem{
				{
					Id: "d50eda0d-2cea-4de1-8d42-9cd3e7e86701",
				},
			},
			count: 1,
		},
		"ok, custom limit": {
			ID:    ID,
			limit: 10,
			query: &store.ListQueryDeviceDeployments{
				IDs:   []string{ID},
				Limit: 10,
			},
			responseCode: http.StatusOK,
			deployments: []model.DeviceDeploymentListItem{
				{
					Id: "d50eda0d-2cea-4de1-8d42-9cd3e7e86701",
				},
			},
			count: 1,
		},
		"ko, too high per_page": {
			ID:           ID,
			limit:        MaximumPerPageListDeviceDeployments + 1,
			responseCode: http.StatusBadRequest,
		},
		"ko, wrong ID": {
			responseCode: http.StatusBadRequest,
			err:          errors.New("error"),
		},
		"ko, wrong limit": {
			ID:           ID,
			limit:        -10,
			responseCode: http.StatusBadRequest,
		},
		"ko, wrong status": {
			ID:           ID,
			status:       "dummy",
			responseCode: http.StatusBadRequest,
		},
		"ko, error": {
			ID: ID,
			query: &store.ListQueryDeviceDeployments{
				IDs:   []string{ID},
				Limit: DefaultPerPage,
			},
			responseCode: http.StatusInternalServerError,
			count:        -1,
			err:          errors.New("error"),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			app := &mapp.App{}
			if tc.query != nil {
				app.On("GetDeviceDeploymentListForDevice",
					mock.MatchedBy(func(ctx context.Context) bool {
						id := identity.FromContext(ctx)
						assert.NotNil(t, id)
						assert.Equal(t, tenantID, id.Tenant)
						return true
					}),
					*tc.query,
				).Return(
					tc.deployments,
					tc.count,
					tc.err,
				)
			}

			restView := new(view.RESTView)
			d := NewDeploymentsApiHandlers(nil, restView, app)
			api := setUpRestTest(
				ApiUrlInternalTenantDeploymentsDevices,
				rest.Get,
				d.ListDeviceDeploymentsByIDsInternal,
			)
			url := "http://localhost" + ApiUrlInternalTenantDeploymentsDevices
			url = strings.Replace(url, "#tenant", tenantID, 1) + "?"
			if tc.ID != "" {
				url = url + "id=" + tc.ID
			}
			if tc.status != "" {
				url = url + "&status=" + tc.status
			}
			if tc.limit != 0 {
				url = url + fmt.Sprintf("&per_page=%d", tc.limit)
			}
			req := test.MakeSimpleRequest("GET", url, nil)

			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.responseCode)
			recorded.ContentTypeIsJson()
			res := []model.DeviceDeploymentListItem{}
			recorded.DecodeJsonPayload(&res)
			if tc.responseCode == http.StatusOK {
				assert.Equal(t, tc.deployments, res, "Unexpected response body")
			}
		})
	}
}

func TestNewConfig(t *testing.T) {
	conf := NewConfig()

	conf.SetDisableNewReleasesFeature(false)
	assert.False(t, conf.DisableNewReleasesFeature)

	conf.SetDisableNewReleasesFeature(true)
	assert.True(t, conf.DisableNewReleasesFeature)
}
