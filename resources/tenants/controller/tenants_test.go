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

package controller

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	//	"strconv"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	deploymentsModel "github.com/mendersoftware/deployments/resources/deployments/model"
	"github.com/mendersoftware/deployments/utils/restutil/view"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/mendersoftware/go-lib-micro/requestlog"
	mt "github.com/mendersoftware/go-lib-micro/testing"
	"github.com/stretchr/testify/assert"

	imageController "github.com/mendersoftware/deployments/resources/images/controller"

	imageMock "github.com/mendersoftware/deployments/resources/images/controller/mocks"
	"github.com/mendersoftware/deployments/resources/tenants/model/mocks"
	h "github.com/mendersoftware/deployments/utils/testing"
	"github.com/stretchr/testify/mock"
)

type routerTypeHandler func(pathExp string, handlerFunc rest.HandlerFunc) *rest.Route

func contextMatcher() interface{} {
	return mock.MatchedBy(func(_ context.Context) bool {
		return true
	})
}

func setUpRestTest(route string, routeType routerTypeHandler,
	handler func(w rest.ResponseWriter, r *rest.Request)) http.Handler {

	router, _ := rest.MakeRouter(routeType(route, handler))
	api := rest.NewApi()
	api.Use(
		&requestlog.RequestLogMiddleware{
			BaseLogger: &logrus.Logger{Out: ioutil.Discard},
		},
		&requestid.RequestIdMiddleware{},
	)
	api.SetApp(router)

	rest.ErrorFieldName = "error"
	return api.MakeHandler()
}

func TestProvisionTenant(t *testing.T) {

	testCases := map[string]struct {
		req      *http.Request
		modelErr error
		checker  mt.ResponseChecker
	}{
		"ok": {
			req: test.MakeSimpleRequest("POST",
				"http://1.2.3.4/api/internal/v1/deployments/tenants",
				&NewTenantReq{TenantId: "foo"}),
			checker: mt.NewJSONResponse(
				http.StatusCreated,
				nil,
				nil),
		},
		"error: bad request": {
			req: test.MakeSimpleRequest("POST",
				"http://1.2.3.4/api/internal/v1/deployments/tenants",
				&NewTenantReq{}),
			checker: mt.NewJSONResponse(
				http.StatusBadRequest,
				nil,
				restError("tenant_id must be provided")),
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(fmt.Sprintf("%s", i), func(t *testing.T) {
			m := &mocks.Model{}
			deps := &deploymentsModel.DeploymentsModel{}

			m.On("ProvisionTenant", contextMatcher(), mock.AnythingOfType("string")).Return(tc.modelErr)

			imageModelMock := &imageMock.ImagesModel{}
			restView := new(view.RESTView)
			imgCtrl := imageController.NewSoftwareImagesController(imageModelMock, restView)

			c := NewController(m, deps, imageModelMock, imgCtrl, restView)

			api := setUpRestTest("/api/internal/v1/deployments/tenants", rest.Post, c.ProvisionTenantsHandler)

			tc.req.Header.Add(requestid.RequestIdHeader, "test")
			recorded := test.RunRequest(t, api, tc.req)
			mt.CheckResponse(t, tc.checker, recorded)
		})
	}
}

func restError(status string) map[string]interface{} {
	return map[string]interface{}{"error": status, "request_id": "test"}
}

func TestInteralTenantNewImage(t *testing.T) {
	t.Parallel()

	file := h.CreateValidImageFile()
	imageBody, err := ioutil.ReadAll(file)
	assert.NoError(t, err)
	assert.NotNil(t, imageBody)
	defer os.Remove(file.Name())
	defer file.Close()

	testCases := []struct {
		h.JSONResponseParams

		InputBodyObject []h.Part
		Tenant          string

		InputContentType string
		InputModelID     string
		InputModelError  error
	}{
		{
			InputBodyObject: nil,
			Tenant:          "foo",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("mime: no media type")),
			},
		},
		{
			InputBodyObject: []h.Part{
				{
					FieldName:  "size",
					FieldValue: strconv.Itoa(len(imageBody)),
				},
				{
					FieldName:   "artifact",
					ContentType: "application/octet-stream",
					ImageData:   imageBody,
				},
			},
			Tenant:           "foo",
			InputContentType: "multipart/form-data",
			InputModelID:     "1234",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusCreated,
				OutputBodyObject: nil,
				OutputHeaders:    map[string]string{"Location": "./r/tenants/foo/artifacts/1234"},
			},
		},
		{
			InputBodyObject:  []h.Part{},
			Tenant:           "",
			InputContentType: "multipart/form-data",
			InputModelID:     "1234",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("missing tenant id in path")),
			},
		},
	}

	for testCaseNumber, testCase := range testCases {

		t.Run(fmt.Sprintf("Test case number: %v", testCaseNumber+1), func(t *testing.T) {

			m := &mocks.Model{}
			imageModelMock := &imageMock.ImagesModel{}

			imageModelMock.On("CreateImage", h.ContextMatcher(),
				mock.AnythingOfType("*controller.MultipartUploadMsg")).
				Return(testCase.InputModelID, testCase.InputModelError)

			deps := &deploymentsModel.DeploymentsModel{}
			restView := new(view.RESTView)
			imgCtrl := imageController.NewSoftwareImagesController(imageModelMock, nil)
			c := NewController(m, deps, imageModelMock, imgCtrl, restView)

			api := setUpRestTest("/r/tenants/:tenant/artifacts", rest.Post, c.NewImageForTenantHandler)

			req := h.MakeMultipartRequest("POST", fmt.Sprintf("http://localhost/r/tenants/%s/artifacts", testCase.Tenant),
				testCase.InputContentType, testCase.InputBodyObject)
			req.Header.Add(requestid.RequestIdHeader, "test")

			recorded := test.RunRequest(t, api, req)

			h.CheckRecordedResponse(t, recorded, testCase.JSONResponseParams)
		})
	}
}
