// Copyright 2020 Northern.tech AS
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
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	mapp "github.com/mendersoftware/deployments/app/mocks"
	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/utils/restutil/view"
	"github.com/mendersoftware/go-lib-micro/rest_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

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

			url := strings.Replace(ApiUrlInternalTenantDeployments, ":tenant", tc.tenant, 1)
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
