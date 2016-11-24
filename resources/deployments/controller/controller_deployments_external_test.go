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

package controller_test

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	"github.com/mendersoftware/deployments/resources/deployments"
	. "github.com/mendersoftware/deployments/resources/deployments/controller"
	"github.com/mendersoftware/deployments/resources/deployments/controller/mocks"
	"github.com/mendersoftware/deployments/resources/deployments/view"
	"github.com/mendersoftware/deployments/resources/images"
	. "github.com/mendersoftware/deployments/utils/pointers"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/mendersoftware/go-lib-micro/requestlog"
	"github.com/stretchr/testify/assert"

	h "github.com/mendersoftware/deployments/utils/testing"
)

// Notice: 	Controller tests are not pure unit tests,
// 			they are more of integration test beween controller and view
//			testing actuall HTTP endpoint input/reponse

func makeDeviceAuthHeader(claim string) string {
	return fmt.Sprintf("Bearer foo.%s.bar",
		base64.StdEncoding.EncodeToString([]byte(claim)))
}

func makeApi(router rest.App) *rest.Api {
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

func TestControllerGetDeploymentForDevice(t *testing.T) {

	t.Parallel()

	image := images.NewSoftwareImage(
		&images.SoftwareImageMetaConstructor{
			Name:        "foo-image",
			Description: "foo-image-desc",
		},
		&images.SoftwareImageMetaYoctoConstructor{
			YoctoId: "yocto-id",
		})

	testCases := []struct {
		h.JSONResponseParams

		InputID string
		Params  url.Values

		InputModelDeploymentInstructions *deployments.DeploymentInstructions
		InputModelError                  error

		InputModelUpdateStatusDeviceID     string
		InputModelUpdateStatusDeploymentId string
		InputModelUpdateStatusStatus       string
		InputModelUpdateStatusError        error

		Headers map[string]string
	}{
		{
			InputID: "malformed-token",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("failed to decode claims: invalid character ':' after top-level value")),
			},
			Headers: map[string]string{
				// fabricate bad token - malformed JSON
				"Authorization": makeDeviceAuthHeader(`"sub": "device"}`),
			},
		},
		{
			InputID:         "device-id-1",
			InputModelError: errors.New("model error"),
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusInternalServerError,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("internal error")),
			},
			Headers: map[string]string{
				"Authorization": makeDeviceAuthHeader(`{"sub": "device-id-1"}`),
			},
		},
		{
			InputID: "device-id-2",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus: http.StatusNoContent,
			},
			Headers: map[string]string{
				"Authorization": makeDeviceAuthHeader(`{"sub": "device-id-2"}`),
			},
		},
		{
			InputID: "device-id-3",
			InputModelDeploymentInstructions: deployments.NewDeploymentInstructions("foo-1",
				&images.Link{}, image),

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus: http.StatusOK,
				OutputBodyObject: deployments.NewDeploymentInstructions("foo-1",
					&images.Link{}, image),
			},
			Headers: map[string]string{
				"Authorization": makeDeviceAuthHeader(`{"sub": "device-id-3"}`),
			},
		},
		{
			InputID: "device-id-3",
			InputModelDeploymentInstructions: deployments.NewDeploymentInstructions("foo-1",
				&images.Link{}, image),

			InputModelUpdateStatusDeploymentId: "foo-1",
			InputModelUpdateStatusDeviceID:     "device-id-3",
			InputModelUpdateStatusStatus:       deployments.DeviceDeploymentStatusAlreadyInst,

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus: http.StatusNoContent,
			},
			Params: url.Values{
				GetDeploymentForDeviceQueryArtifact: []string{"yocto-id"},
			},
			Headers: map[string]string{
				"Authorization": makeDeviceAuthHeader(`{"sub": "device-id-3"}`),
			},
		},
	}

	for tidx, testCase := range testCases {

		t.Logf("testing input ID: %v tc: %d", testCase.InputID, tidx)
		deploymentModel := new(mocks.DeploymentsModel)
		deploymentModel.On("GetDeploymentForDevice", testCase.InputID).
			Return(testCase.InputModelDeploymentInstructions, testCase.InputModelError)

		deploymentModel.On("UpdateDeviceDeploymentStatus",
			testCase.InputModelUpdateStatusDeploymentId,
			testCase.InputModelUpdateStatusDeviceID,
			testCase.InputModelUpdateStatusStatus).
			Return(testCase.InputModelUpdateStatusError)

		router, err := rest.MakeRouter(
			rest.Get("/r/update",
				NewDeploymentsController(deploymentModel, new(view.DeploymentsView)).GetDeploymentForDevice))
		assert.NoError(t, err)

		api := makeApi(router)

		vals := testCase.Params.Encode()
		req := test.MakeSimpleRequest("GET", "http://localhost/r/update?"+vals, nil)
		for k, v := range testCase.Headers {
			req.Header.Set(k, v)
		}
		req.Header.Add(requestid.RequestIdHeader, "test")
		recorded := test.RunRequest(t, api.MakeHandler(), req)

		h.CheckRecordedResponse(t, recorded, testCase.JSONResponseParams)
	}
}

func TestControllerGetDeployment(t *testing.T) {

	t.Parallel()

	testCases := []struct {
		h.JSONResponseParams

		InputID string

		InputModelDeployment *deployments.Deployment
		InputModelError      error
	}{
		{
			InputID: "broken_id",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(ErrIDNotUUIDv4),
			},
		},
		{
			InputID:         "f826484e-1157-4109-af21-304e6d711560",
			InputModelError: errors.New("model error"),
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusInternalServerError,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("internal error")),
			},
		},
		{
			InputID: "f826484e-1157-4109-af21-304e6d711560",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusNotFound,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Resource not found")),
			},
		},
		{
			InputID: "f826484e-1157-4109-af21-304e6d711560",

			InputModelDeployment: &deployments.Deployment{Id: StringToPointer("id 123")},

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus: http.StatusOK,
				OutputBodyObject: &struct {
					deployments.Deployment
					Status string `json:"string"`
				}{
					Deployment: deployments.Deployment{
						Id: StringToPointer("id 123"),
					},
					Status: "pending",
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Logf("testing %v", testCase.InputID)

		deploymentModel := new(mocks.DeploymentsModel)
		deploymentModel.On("GetDeployment", testCase.InputID).
			Return(testCase.InputModelDeployment, testCase.InputModelError)

		router, err := rest.MakeRouter(
			rest.Get("/r/:id",
				NewDeploymentsController(deploymentModel, new(view.DeploymentsView)).GetDeployment))
		assert.NoError(t, err)

		api := makeApi(router)

		req := test.MakeSimpleRequest("GET", "http://localhost/r/"+testCase.InputID, nil)
		req.Header.Add(requestid.RequestIdHeader, "test")
		recorded := test.RunRequest(t, api.MakeHandler(), req)

		h.CheckRecordedResponse(t, recorded, testCase.JSONResponseParams)
	}
}

func TestControllerPostDeployment(t *testing.T) {

	t.Parallel()

	testCases := []struct {
		h.JSONResponseParams

		InputBodyObject interface{}

		InputModelID    string
		InputModelError error
	}{
		{
			InputBodyObject: nil,
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Validating request body: JSON payload is empty")),
			},
		},
		{
			InputBodyObject: deployments.NewDeploymentConstructor(),
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New(`Validating request body: Name: non zero value required;ArtifactName: non zero value required;Devices: non zero value required;`)),
			},
		},
		{
			InputBodyObject: &deployments.DeploymentConstructor{
				Name:         StringToPointer("NYC Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"f826484e-1157-4109-af21-304e6d711560"},
			},
			InputModelError: errors.New("model error"),
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusInternalServerError,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("internal error")),
			},
		},
		{
			InputBodyObject: &deployments.DeploymentConstructor{
				Name:         StringToPointer("NYC Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"f826484e-1157-4109-af21-304e6d711560"},
			},
			InputModelID: "1234",
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusCreated,
				OutputBodyObject: nil,
				OutputHeaders:    map[string]string{"Location": "http://localhost/r/1234"},
			},
		},
	}

	for _, testCase := range testCases {

		deploymentModel := new(mocks.DeploymentsModel)

		deploymentModel.On("CreateDeployment", testCase.InputBodyObject).
			Return(testCase.InputModelID, testCase.InputModelError)

		router, err := rest.MakeRouter(
			rest.Post("/r",
				NewDeploymentsController(deploymentModel, new(view.DeploymentsView)).PostDeployment))
		assert.NoError(t, err)

		api := makeApi(router)

		req := test.MakeSimpleRequest("POST", "http://localhost/r", testCase.InputBodyObject)
		req.Header.Add(requestid.RequestIdHeader, "test")
		recorded := test.RunRequest(t, api.MakeHandler(), req)

		h.CheckRecordedResponse(t, recorded, testCase.JSONResponseParams)
	}
}

func TestControllerPutDeploymentStatus(t *testing.T) {

	t.Parallel()

	type report struct {
		Status string `json:"status"`
	}

	testCases := []struct {
		h.JSONResponseParams

		InputBodyObject interface{}

		InputModelDeploymentID string
		InputModelDeviceID     string
		InputModelStatus       string
		InputModelError        error

		Headers map[string]string
	}{
		{
			// empty status report body
			InputBodyObject: nil,

			InputModelDeploymentID: "f826484e-1157-4109-af21-304e6d711560",
			InputModelDeviceID:     "device-id-1",
			InputModelStatus:       "none",

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("JSON payload is empty")),
			},
			Headers: map[string]string{
				"Authorization": makeDeviceAuthHeader(`{"sub": "device-id-1"}`),
			},
		},
		{
			// all correct
			InputBodyObject:        &report{Status: "installing"},
			InputModelDeploymentID: "f826484e-1157-4109-af21-304e6d711560",
			InputModelDeviceID:     "device-id-2",
			InputModelStatus:       "installing",

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusNoContent,
				OutputBodyObject: nil,
			},
			Headers: map[string]string{
				"Authorization": makeDeviceAuthHeader(`{"sub": "device-id-2"}`),
			},
		},
		{
			// no authorization
			InputBodyObject:        &report{Status: "installing"},
			InputModelDeploymentID: "f826484e-1157-4109-af21-304e6d711560",
			InputModelDeviceID:     "device-id-3",
			InputModelStatus:       "installing",

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("malformed authorization data")),
			},
		},
		{
			// no authorization
			InputBodyObject:        &report{Status: "success"},
			InputModelDeploymentID: "f826484e-1157-4109-af21-304e6d711560",
			InputModelDeviceID:     "device-id-4",
			InputModelStatus:       "success",
			InputModelError:        errors.New("model error"),

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusInternalServerError,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("internal error")),
			},
			Headers: map[string]string{
				"Authorization": makeDeviceAuthHeader(`{"sub": "device-id-4"}`),
			},
		},
		{
			// aborted -> installing, forbidden
			InputBodyObject:        &report{Status: "installing"},
			InputModelDeploymentID: "f826484e-1157-4109-af21-304e6d711560",
			InputModelDeviceID:     "device-id-2",
			InputModelStatus:       "installing",
			InputModelError:        ErrDeploymentAborted,

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusConflict,
				OutputBodyObject: h.ErrorToErrStruct(ErrDeploymentAborted),
			},
			Headers: map[string]string{
				"Authorization": makeDeviceAuthHeader(`{"sub": "device-id-2"}`),
			},
		},
		{
			// change to aborted forbidden
			InputBodyObject:        &report{Status: "aborted"},
			InputModelDeploymentID: "f826484e-1157-4109-af21-304e6d711560",
			InputModelDeviceID:     "device-id-2",
			InputModelStatus:       "aborted",

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(ErrBadStatus),
			},
			Headers: map[string]string{
				"Authorization": makeDeviceAuthHeader(`{"sub": "device-id-2"}`),
			},
		},
	}

	for _, testCase := range testCases {

		t.Logf("testing %s %s %s %v",
			testCase.InputModelDeploymentID, testCase.InputModelDeviceID,
			testCase.InputModelStatus, testCase.InputModelError)
		deploymentModel := new(mocks.DeploymentsModel)

		deploymentModel.On("UpdateDeviceDeploymentStatus",
			testCase.InputModelDeploymentID,
			testCase.InputModelDeviceID, testCase.InputModelStatus).
			Return(testCase.InputModelError)

		router, err := rest.MakeRouter(
			rest.Post("/r/:id",
				NewDeploymentsController(deploymentModel,
					new(view.DeploymentsView)).PutDeploymentStatusForDevice))
		assert.NoError(t, err)

		api := makeApi(router)

		req := test.MakeSimpleRequest("POST", "http://localhost/r/"+testCase.InputModelDeploymentID,
			testCase.InputBodyObject)
		for k, v := range testCase.Headers {
			req.Header.Set(k, v)
		}
		req.Header.Add(requestid.RequestIdHeader, "test")
		recorded := test.RunRequest(t, api.MakeHandler(), req)

		h.CheckRecordedResponse(t, recorded, testCase.JSONResponseParams)
	}
}

func TestControllerGetDeploymentStats(t *testing.T) {

	t.Parallel()

	testCases := []struct {
		h.JSONResponseParams

		InputModelDeploymentID string
		InputModelStats        deployments.Stats
		InputModelError        error
	}{
		{
			InputModelDeploymentID: "f826484e-1157-4109-af21-304e6d711560",

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusNotFound,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Resource not found")),
			},
		},
		{
			InputModelDeploymentID: "f826484e-1157-4109-af21-304e6d711560",
			InputModelError:        errors.New("storage issue"),

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusInternalServerError,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("internal error")),
			},
		},
		{
			InputModelDeploymentID: "23bbc7ba-3278-4b1c-a345-4080afe59e96",
			InputModelStats: deployments.Stats{
				deployments.DeviceDeploymentStatusSuccess:     12,
				deployments.DeviceDeploymentStatusFailure:     2,
				deployments.DeviceDeploymentStatusDownloading: 1,
				deployments.DeviceDeploymentStatusRebooting:   3,
				deployments.DeviceDeploymentStatusInstalling:  1,
				deployments.DeviceDeploymentStatusPending:     2,
				deployments.DeviceDeploymentStatusNoImage:     0,
				deployments.DeviceDeploymentStatusAlreadyInst: 0,
				deployments.DeviceDeploymentStatusAborted:     0,
			},

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus: http.StatusOK,
				OutputBodyObject: deployments.Stats{
					deployments.DeviceDeploymentStatusSuccess:     12,
					deployments.DeviceDeploymentStatusFailure:     2,
					deployments.DeviceDeploymentStatusDownloading: 1,
					deployments.DeviceDeploymentStatusRebooting:   3,
					deployments.DeviceDeploymentStatusInstalling:  1,
					deployments.DeviceDeploymentStatusPending:     2,
					deployments.DeviceDeploymentStatusNoImage:     0,
					deployments.DeviceDeploymentStatusAlreadyInst: 0,
					deployments.DeviceDeploymentStatusAborted:     0,
				},
			},
		},
	}

	for _, testCase := range testCases {

		t.Logf("testing %s %v", testCase.InputModelDeploymentID, testCase.InputModelError)
		deploymentModel := new(mocks.DeploymentsModel)

		deploymentModel.On("GetDeploymentStats", testCase.InputModelDeploymentID).
			Return(testCase.InputModelStats, testCase.InputModelError)

		router, err := rest.MakeRouter(
			rest.Post("/r/:id",
				NewDeploymentsController(deploymentModel,
					new(view.DeploymentsView)).GetDeploymentStats))

		assert.NoError(t, err)

		api := makeApi(router)

		req := test.MakeSimpleRequest("POST", "http://localhost/r/"+testCase.InputModelDeploymentID,
			nil)
		req.Header.Add(requestid.RequestIdHeader, "test")
		recorded := test.RunRequest(t, api.MakeHandler(), req)

		h.CheckRecordedResponse(t, recorded, testCase.JSONResponseParams)
	}
}

func TestControllerGetDeviceStatusesForDeployment(t *testing.T) {
	t.Parallel()

	// common device status list for all tests

	statuses := []deployments.DeviceDeployment{
		*deployments.NewDeviceDeployment("device0001", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
		*deployments.NewDeviceDeployment("device0002", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
		*deployments.NewDeviceDeployment("device0003", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
	}

	testCases := map[string]struct {
		h.JSONResponseParams

		deploymentID  string
		modelStatuses []deployments.DeviceDeployment
		modelErr      error
	}{
		"existing deployment and statuses": {
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusOK,
				OutputBodyObject: statuses,
			},
			deploymentID:  "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			modelStatuses: statuses,
			modelErr:      nil,
		},
		"deployment ID format error": {
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("ID is not UUIDv4")),
			},
			deploymentID:  "30b3e62c9ec24312a7facff24cc7397a",
			modelStatuses: nil,
			modelErr:      nil,
		},
		"model error: deployment doesn't exist": {
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusNotFound,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Deployment not found")),
			},
			deploymentID:  "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			modelStatuses: nil,
			modelErr:      ErrModelDeploymentNotFound,
		},
		"unknown model error": {
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusInternalServerError,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("internal error")),
			},
			deploymentID:  "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			modelStatuses: nil,
			modelErr:      errors.New("some unknown error"),
		},
	}

	for id, tc := range testCases {
		t.Logf("test case: %s", id)

		deploymentModel := new(mocks.DeploymentsModel)
		deploymentModel.On("GetDeviceStatusesForDeployment", tc.deploymentID).
			Return(tc.modelStatuses, tc.modelErr)

		router, err := rest.MakeRouter(
			rest.Get("/r/:id",
				NewDeploymentsController(deploymentModel, new(view.DeploymentsView)).GetDeviceStatusesForDeployment))

		assert.NoError(t, err)

		api := makeApi(router)

		req := test.MakeSimpleRequest("GET", "http://localhost/r/"+tc.deploymentID, nil)
		req.Header.Add(requestid.RequestIdHeader, "test")
		recorded := test.RunRequest(t, api.MakeHandler(), req)

		h.CheckRecordedResponse(t, recorded, tc.JSONResponseParams)
	}
}

func TestControllerLookupDeployment(t *testing.T) {

	t.Parallel()

	someDeployments := []*deployments.Deployment{
		&deployments.Deployment{
			DeploymentConstructor: &deployments.DeploymentConstructor{
				Name:         StringToPointer("zen"),
				ArtifactName: StringToPointer("baz"),
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},
			Id: StringToPointer("a108ae14-bb4e-455f-9b40-2ef4bab97bb7"),
		},
		&deployments.Deployment{
			DeploymentConstructor: &deployments.DeploymentConstructor{
				Name:         StringToPointer("foo"),
				ArtifactName: StringToPointer("bar"),
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},
			Id: StringToPointer("e8c32ff6-7c1b-43c7-aa31-2e4fc3a3c130"),
		},
	}

	testCases := []struct {
		h.JSONResponseParams

		SearchStatus string

		InputModelQuery       deployments.Query
		InputModelError       error
		InputModelDeployments []*deployments.Deployment
	}{
		{
			InputModelQuery: deployments.Query{
				SearchText: " ",
			},
			InputModelError: errors.New("bad query"),

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("bad query")),
			},
		},
		{
			SearchStatus:    "badstatus",
			InputModelError: errors.New("bad query"),

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("unknown status badstatus")),
			},
		},
		{
			InputModelQuery: deployments.Query{
				SearchText: "foo-not-found",
			},
			InputModelDeployments: []*deployments.Deployment{},

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusOK,
				OutputBodyObject: []*deployments.Deployment{},
			},
		},
		{
			InputModelQuery: deployments.Query{
				SearchText: "foo",
				Status:     deployments.StatusQueryInProgress,
			},
			SearchStatus:          "inprogress",
			InputModelDeployments: someDeployments,

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus: http.StatusOK,
				OutputBodyObject: []struct {
					deployments.Deployment
					Status string `json:"status"`
				}{
					{
						Deployment: deployments.Deployment{
							DeploymentConstructor: &deployments.DeploymentConstructor{
								Name:         StringToPointer("zen"),
								ArtifactName: StringToPointer("baz"),
							},
							Id: StringToPointer("a108ae14-bb4e-455f-9b40-2ef4bab97bb7"),
						},
						Status: "finished",
					},
					{
						Deployment: deployments.Deployment{
							DeploymentConstructor: &deployments.DeploymentConstructor{
								Name:         StringToPointer("foo"),
								ArtifactName: StringToPointer("bar"),
							},
							Id: StringToPointer("e8c32ff6-7c1b-43c7-aa31-2e4fc3a3c130"),
						},
						Status: "finished",
					},
				},
			},
		},
	}

	for _, testCase := range testCases {

		t.Logf("testing %+v %s", testCase.InputModelQuery, testCase.InputModelError)
		deploymentModel := new(mocks.DeploymentsModel)

		deploymentModel.On("LookupDeployment", testCase.InputModelQuery).
			Return(testCase.InputModelDeployments, testCase.InputModelError)

		router, err := rest.MakeRouter(
			rest.Get("/r",
				NewDeploymentsController(deploymentModel,
					new(view.DeploymentsView)).LookupDeployment))

		assert.NoError(t, err)

		api := makeApi(router)

		u := url.URL{
			Scheme: "http",
			Host:   "localhost",
			Path:   "/r",
		}
		q := u.Query()
		q.Set("search", testCase.InputModelQuery.SearchText)
		if testCase.SearchStatus != "" {
			q.Set("status", testCase.SearchStatus)
		}
		u.RawQuery = q.Encode()
		t.Logf("query: %s", u.String())
		req := test.MakeSimpleRequest("GET", u.String(), nil)
		req.Header.Add(requestid.RequestIdHeader, "test")
		recorded := test.RunRequest(t, api.MakeHandler(), req)

		h.CheckRecordedResponse(t, recorded, testCase.JSONResponseParams)
	}
}

func TestParseLookupQuery(t *testing.T) {
	testCases := []struct {
		vals  url.Values
		query deployments.Query
		err   error
	}{
		{
			vals: url.Values{
				"search": []string{"foo"},
				"status": []string{"inprogress"},
			},
			query: deployments.Query{
				SearchText: "foo",
				Status:     deployments.StatusQueryInProgress,
			},
		},
		{
			vals: url.Values{
				"search": []string{"foo"},
				"status": []string{"bar"},
			},
			err: errors.New("unknown status bar"),
		},
		{
			vals: url.Values{
				"search": []string{"foo"},
				"status": []string{"finished"},
			},
			query: deployments.Query{
				SearchText: "foo",
				Status:     deployments.StatusQueryFinished,
			},
		},
		{
			vals: url.Values{
				"search": []string{"foo"},
				"status": []string{"pending"},
			},
			query: deployments.Query{
				SearchText: "foo",
				Status:     deployments.StatusQueryPending,
			},
		},
		{
			vals: url.Values{
				"search": []string{"foo"},
			},
			query: deployments.Query{
				SearchText: "foo",
				Status:     deployments.StatusQueryAny,
			},
		},
		{
			vals: url.Values{},
			query: deployments.Query{
				SearchText: "",
				Status:     deployments.StatusQueryAny,
			},
		},
		{
			vals: url.Values{
				"status": []string{"pending"},
			},
			query: deployments.Query{
				SearchText: "",
				Status:     deployments.StatusQueryPending,
			},
		},
	}
	for _, tc := range testCases {
		t.Logf("testing: %v", tc.vals)

		q, err := ParseLookupQuery(tc.vals)
		if tc.err != nil {
			assert.Error(t, err)
			assert.EqualError(t, tc.err, err.Error())
		} else {
			assert.Equal(t, tc.query, q)
		}
	}
}

func TestControllerPutDeploymentLog(t *testing.T) {

	t.Parallel()

	type log struct {
		Messages string `json:"messages"`
	}

	tref := time.Now()

	messages := []deployments.LogMessage{
		{
			Timestamp: &tref,
			Message:   "foo",
			Level:     "notice",
		},
	}
	testCases := []struct {
		h.JSONResponseParams
		InputBodyObject interface{}

		InputModelDeploymentID string
		InputModelDeviceID     string
		InputModelMessages     []deployments.LogMessage
		InputModelError        error

		Headers map[string]string
	}{
		{
			// empty log body
			InputBodyObject: nil,

			InputModelDeploymentID: "f826484e-1157-4109-af21-304e6d711560",
			InputModelDeviceID:     "device-id-1",
			InputModelMessages:     nil,

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("JSON payload is empty")),
			},
			Headers: map[string]string{
				"Authorization": makeDeviceAuthHeader(`{"sub": "device-id-1"}`),
			},
		},
		{
			// all correct
			InputBodyObject: &deployments.DeploymentLog{
				Messages: messages,
			},
			InputModelDeploymentID: "f826484e-1157-4109-af21-304e6d711560",
			InputModelDeviceID:     "device-id-2",
			InputModelMessages:     messages,

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusNoContent,
				OutputBodyObject: nil,
			},
			Headers: map[string]string{
				"Authorization": makeDeviceAuthHeader(`{"sub": "device-id-2"}`),
			},
		},
		{
			// no authorization
			InputBodyObject: &deployments.DeploymentLog{
				Messages: messages,
			},
			InputModelDeploymentID: "f826484e-1157-4109-af21-304e6d711560",
			InputModelDeviceID:     "device-id-3",
			InputModelMessages:     messages,

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("malformed authorization data")),
			},
		},
		{
			// model error
			InputBodyObject: &deployments.DeploymentLog{
				Messages: messages,
			},
			InputModelDeploymentID: "f826484e-1157-4109-af21-304e6d711560",
			InputModelDeviceID:     "device-id-4",
			InputModelError:        errors.New("model error"),
			InputModelMessages:     messages,

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusInternalServerError,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("internal error")),
			},
			Headers: map[string]string{
				"Authorization": makeDeviceAuthHeader(`{"sub": "device-id-4"}`),
			},
		},
		{
			// deployment not assigned to device
			InputBodyObject: &deployments.DeploymentLog{
				Messages: messages,
			},
			InputModelDeploymentID: "f826484e-1157-4109-af21-304e6d711560",
			InputModelDeviceID:     "device-id-5",
			InputModelError:        ErrModelDeploymentNotFound,
			InputModelMessages:     messages,

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusNotFound,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Deployment not found")),
			},
			Headers: map[string]string{
				"Authorization": makeDeviceAuthHeader(`{"sub": "device-id-5"}`),
			},
		},
	}

	for _, testCase := range testCases {
		t.Logf("testing %s %s %v %v",
			testCase.InputModelDeploymentID, testCase.InputModelDeviceID,
			testCase.InputBodyObject, testCase.InputModelError)
		deploymentModel := new(mocks.DeploymentsModel)

		deploymentModel.On("SaveDeviceDeploymentLog",
			testCase.InputModelDeviceID,
			testCase.InputModelDeploymentID,
			testCase.InputModelMessages).
			Return(testCase.InputModelError)

		router, err := rest.MakeRouter(
			rest.Put("/r/:id",
				NewDeploymentsController(deploymentModel,
					new(view.DeploymentsView)).PutDeploymentLogForDevice))
		assert.NoError(t, err)

		api := makeApi(router)

		req := test.MakeSimpleRequest("PUT", "http://localhost/r/"+testCase.InputModelDeploymentID,
			testCase.InputBodyObject)
		for k, v := range testCase.Headers {
			req.Header.Set(k, v)
		}
		req.Header.Add(requestid.RequestIdHeader, "test")
		recorded := test.RunRequest(t, api.MakeHandler(), req)

		h.CheckRecordedResponse(t, recorded, testCase.JSONResponseParams)
	}
}

func parseTime(t *testing.T, value string) *time.Time {
	tm, err := time.Parse(time.RFC3339, value)
	if assert.NoError(t, err) == false {
		t.Fatalf("failed to parse time %s", value)
	}

	return &tm
}

func TestControllerGetDeploymentLog(t *testing.T) {

	t.Parallel()

	type log struct {
		Messages string `json:"messages"`
	}

	tref := parseTime(t, "2006-01-02T15:04:05-07:00")

	messages := []deployments.LogMessage{
		{
			Timestamp: tref,
			Message:   "foo",
			Level:     "notice",
		},
		{
			Timestamp: tref,
			Message:   "zed zed zed",
			Level:     "debug",
		},
		{
			Timestamp: tref,
			Message:   "bar bar bar",
			Level:     "info",
		},
	}

	testCases := []struct {
		h.JSONResponseParams

		InputModelDeploymentLog *deployments.DeploymentLog
		InputModelDeploymentID  string
		InputModelDeviceID      string
		InputModelMessages      []deployments.LogMessage
		InputModelError         error

		Body string
	}{
		{
			// all correct
			InputModelDeploymentLog: &deployments.DeploymentLog{
				DeploymentID: "f826484e-1157-4109-af21-304e6d711560",
				DeviceID:     "device-id-1",
				Messages:     messages,
			},
			InputModelDeploymentID: "f826484e-1157-4109-af21-304e6d711560",
			InputModelDeviceID:     "device-id-1",
			InputModelMessages:     messages,

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusOK,
				OutputBodyObject: nil,
			},
			Body: `2006-01-02 22:04:05 +0000 UTC notice: foo
2006-01-02 22:04:05 +0000 UTC debug: zed zed zed
2006-01-02 22:04:05 +0000 UTC info: bar bar bar
`,
		},
		{
			// model error
			InputModelDeploymentLog: nil,
			InputModelDeploymentID:  "f826484e-1157-4109-af21-304e6d711560",
			InputModelDeviceID:      "device-id-4",
			InputModelError:         errors.New("model error"),
			InputModelMessages:      messages,

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusInternalServerError,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("internal error")),
			},
		},
		{
			// deployment not assigned to device
			InputModelDeploymentLog: nil,
			InputModelDeploymentID:  "f826484e-1157-4109-af21-304e6d711560",
			InputModelDeviceID:      "device-id-5",
			InputModelError:         nil,
			InputModelMessages:      messages,

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusNotFound,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Resource not found")),
			},
		},
	}

	for _, testCase := range testCases {
		t.Logf("testing %s %s %v",
			testCase.InputModelDeploymentID, testCase.InputModelDeviceID,
			testCase.InputModelError)
		deploymentModel := new(mocks.DeploymentsModel)

		deploymentModel.On("GetDeviceDeploymentLog",
			testCase.InputModelDeviceID,
			testCase.InputModelDeploymentID).
			Return(testCase.InputModelDeploymentLog, testCase.InputModelError)

		router, err := rest.MakeRouter(
			rest.Get("/r/:id/:devid",
				NewDeploymentsController(deploymentModel,
					new(view.DeploymentsView)).GetDeploymentLogForDevice))
		assert.NoError(t, err)

		api := makeApi(router)

		req := test.MakeSimpleRequest("GET", "http://localhost/r/"+
			testCase.InputModelDeploymentID+"/"+testCase.InputModelDeviceID,
			nil)
		req.Header.Add(requestid.RequestIdHeader, "test")
		recorded := test.RunRequest(t, api.MakeHandler(), req)
		if testCase.JSONResponseParams.OutputStatus != http.StatusOK {
			h.CheckRecordedResponse(t, recorded, testCase.JSONResponseParams)
		} else {
			assert.Equal(t, testCase.Body, recorded.Recorder.Body.String())
			assert.Equal(t, http.StatusOK, recorded.Recorder.Code)
			assert.Equal(t, "text/plain", recorded.Recorder.HeaderMap.Get("Content-Type"))
			t.Logf("content:\n%s", recorded.Recorder.Body)
		}
	}
}

func TestControllerAbortDeployment(t *testing.T) {

	t.Parallel()

	type report struct {
		Status string `json:"status"`
	}

	testCases := []struct {
		h.JSONResponseParams

		InputBodyObject interface{}

		InputModelDeploymentID              string
		InputModelStatus                    string
		InputModelDeploymentFinishedFlag    bool
		InputModelIsDeploymentFinishedError error
		InputModelError                     error
	}{
		{
			// empty body
			InputBodyObject: nil,

			InputModelDeploymentID:           "f826484e-1157-4109-af21-304e6d711560",
			InputModelStatus:                 "none",
			InputModelDeploymentFinishedFlag: false,

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("JSON payload is empty")),
			},
		},
		{
			// wrong status
			InputBodyObject:                  &report{Status: "finished"},
			InputModelDeploymentID:           "f826484e-1157-4109-af21-304e6d711560",
			InputModelStatus:                 "finished",
			InputModelDeploymentFinishedFlag: false,

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusBadRequest,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Unexpected deployment status")),
			},
		},
		{
			// deployment finished already
			InputBodyObject:                  &report{Status: "aborted"},
			InputModelDeploymentID:           "f826484e-1157-4109-af21-304e6d711560",
			InputModelStatus:                 "aborted",
			InputModelDeploymentFinishedFlag: true,

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusUnprocessableEntity,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("Deployment already finished")),
			},
		},
		{
			// checking if deploymen was finished error
			InputBodyObject:                     &report{Status: "aborted"},
			InputModelDeploymentID:              "f826484e-1157-4109-af21-304e6d711560",
			InputModelStatus:                    "aborted",
			InputModelDeploymentFinishedFlag:    true,
			InputModelIsDeploymentFinishedError: errors.New("IsDeploymentFinished error"),

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusInternalServerError,
				OutputBodyObject: h.ErrorToErrStruct(errors.New("internal error")),
			},
		},
		{
			// all correct
			InputBodyObject:                  &report{Status: "aborted"},
			InputModelDeploymentID:           "f826484e-1157-4109-af21-304e6d711560",
			InputModelStatus:                 "aborted",
			InputModelDeploymentFinishedFlag: false,

			JSONResponseParams: h.JSONResponseParams{
				OutputStatus: http.StatusNoContent,
			},
		},
	}

	for _, testCase := range testCases {

		t.Logf("testing %s %s %t %v %v",
			testCase.InputModelDeploymentID, testCase.InputModelStatus,
			testCase.InputModelDeploymentFinishedFlag, testCase.InputModelIsDeploymentFinishedError,
			testCase.InputModelError)
		deploymentModel := new(mocks.DeploymentsModel)

		deploymentModel.On("AbortDeployment", testCase.InputModelDeploymentID).
			Return(testCase.InputModelError)

		deploymentModel.On("IsDeploymentFinished", testCase.InputModelDeploymentID).
			Return(testCase.InputModelDeploymentFinishedFlag, testCase.InputModelIsDeploymentFinishedError)

		router, err := rest.MakeRouter(
			rest.Post("/r/:id",
				NewDeploymentsController(deploymentModel,
					new(view.DeploymentsView)).AbortDeployment))
		assert.NoError(t, err)

		api := makeApi(router)

		req := test.MakeSimpleRequest("POST", "http://localhost/r/"+testCase.InputModelDeploymentID,
			testCase.InputBodyObject)
		req.Header.Add(requestid.RequestIdHeader, "test")
		recorded := test.RunRequest(t, api.MakeHandler(), req)

		h.CheckRecordedResponse(t, recorded, testCase.JSONResponseParams)
	}
}
