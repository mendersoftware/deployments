// Copyright 2017 Northern.tech AS
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

package controller_test

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/mendersoftware/go-lib-micro/requestlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/mendersoftware/deployments/model"
	. "github.com/mendersoftware/deployments/resources/limits/controller"
	"github.com/mendersoftware/deployments/resources/limits/controller/mocks"
	"github.com/mendersoftware/deployments/utils/restutil/view"
)

type routerTypeHandler func(pathExp string, handlerFunc rest.HandlerFunc) *rest.Route

func contextMatcher() interface{} {
	return mock.MatchedBy(func(_ context.Context) bool {
		return true
	})
}

func setUpRestTest(route string, routeType routerTypeHandler,
	handler func(w rest.ResponseWriter, r *rest.Request)) *rest.Api {

	router, _ := rest.MakeRouter(routeType(route, handler))
	api := rest.NewApi()
	api.Use(
		&requestlog.RequestLogMiddleware{
			BaseLogger: &logrus.Logger{Out: ioutil.Discard},
		},
		&requestid.RequestIdMiddleware{},
	)
	api.SetApp(router)

	return api
}

func TestGetLimits(t *testing.T) {

	testCases := []struct {
		name  string
		code  int
		body  string
		err   error
		limit *model.Limit
	}{
		{
			name: "storage",
			code: http.StatusOK,
			body: `{"limit":200,"usage":0}`,
			limit: &model.Limit{
				Name:  "storage",
				Value: 200,
			},
		},
		{
			name: "storage",
			code: http.StatusInternalServerError,
			err:  errors.New("failed"),
		},
		{
			name: "foobar",
			code: http.StatusBadRequest,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			limitsModel := &mocks.LimitsModel{}
			controller := NewLimitsController(limitsModel, new(view.RESTView))

			api := setUpRestTest("/api/0.0.1/limits/:name", rest.Get, controller.GetLimit)

			if tc.err != nil || tc.limit != nil {
				limitsModel.On("GetLimit", contextMatcher(), tc.name).
					Return(tc.limit, tc.err)
			}

			recorded := test.RunRequest(t, api.MakeHandler(),
				test.MakeSimpleRequest("GET", "http://localhost/api/0.0.1/limits/"+tc.name,
					nil))
			recorded.CodeIs(tc.code)
			if tc.code == http.StatusOK {
				assert.JSONEq(t, tc.body, recorded.Recorder.Body.String())
			}
			limitsModel.AssertExpectations(t)

		})
	}
}
