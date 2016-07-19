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
	"net/http"
	"testing"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	"github.com/mendersoftware/deployments/resources/deployments"
	. "github.com/mendersoftware/deployments/resources/deployments/controller"
	"github.com/mendersoftware/deployments/resources/deployments/controller/mocks"
	"github.com/mendersoftware/deployments/resources/deployments/view"
	. "github.com/mendersoftware/deployments/utils/pointers"
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

func TestControllerGetDeploymentForDevice(t *testing.T) {

	t.Parallel()

	testCases := []struct {
		h.JSONResponseParams

		InputID string

		InputModelDeploymentInstructions *deployments.DeploymentInstructions
		InputModelError                  error

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
				OutputBodyObject: h.ErrorToErrStruct(errors.New("model error")),
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
			InputModelDeploymentInstructions: deployments.NewDeploymentInstructions("", nil, nil),
			JSONResponseParams: h.JSONResponseParams{
				OutputStatus:     http.StatusOK,
				OutputBodyObject: deployments.NewDeploymentInstructions("", nil, nil),
			},
			Headers: map[string]string{
				"Authorization": makeDeviceAuthHeader(`{"sub": "device-id-3"}`),
			},
		},
	}

	for _, testCase := range testCases {

		t.Logf("testing input ID: %v", testCase.InputID)
		deploymentModel := new(mocks.DeploymentsModel)
		deploymentModel.On("GetDeploymentForDevice", testCase.InputID).
			Return(testCase.InputModelDeploymentInstructions, testCase.InputModelError)

		router, err := rest.MakeRouter(
			rest.Get("/r/update",
				NewDeploymentsController(deploymentModel, new(view.DeploymentsView)).GetDeploymentForDevice))
		assert.NoError(t, err)

		api := rest.NewApi()
		api.SetApp(router)

		req := test.MakeSimpleRequest("GET", "http://localhost/r/update", nil)
		for k, v := range testCase.Headers {
			req.Header.Set(k, v)
		}
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
				OutputBodyObject: h.ErrorToErrStruct(errors.New("model error")),
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
				OutputStatus:     http.StatusOK,
				OutputBodyObject: &deployments.Deployment{Id: StringToPointer("id 123")},
			},
		},
	}

	for _, testCase := range testCases {

		deploymentModel := new(mocks.DeploymentsModel)
		deploymentModel.On("GetDeployment", testCase.InputID).
			Return(testCase.InputModelDeployment, testCase.InputModelError)

		router, err := rest.MakeRouter(
			rest.Get("/r/:id",
				NewDeploymentsController(deploymentModel, new(view.DeploymentsView)).GetDeployment))
		assert.NoError(t, err)

		api := rest.NewApi()
		api.SetApp(router)

		recorded := test.RunRequest(t, api.MakeHandler(),
			test.MakeSimpleRequest("GET", "http://localhost/r/"+testCase.InputID, nil))

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
				OutputBodyObject: h.ErrorToErrStruct(errors.New("model error")),
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

		api := rest.NewApi()
		api.SetApp(router)

		recorded := test.RunRequest(t, api.MakeHandler(),
			test.MakeSimpleRequest("POST", "http://localhost/r", testCase.InputBodyObject))

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
				OutputBodyObject: h.ErrorToErrStruct(errors.New("model error")),
			},
			Headers: map[string]string{
				"Authorization": makeDeviceAuthHeader(`{"sub": "device-id-4"}`),
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

		api := rest.NewApi()
		api.SetApp(router)

		req := test.MakeSimpleRequest("POST", "http://localhost/r/"+testCase.InputModelDeploymentID,
			testCase.InputBodyObject)
		for k, v := range testCase.Headers {
			req.Header.Set(k, v)
		}
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
				OutputBodyObject: h.ErrorToErrStruct(errors.New("storage issue")),
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

		api := rest.NewApi()
		api.SetApp(router)

		req := test.MakeSimpleRequest("POST", "http://localhost/r/"+testCase.InputModelDeploymentID,
			nil)
		recorded := test.RunRequest(t, api.MakeHandler(), req)

		h.CheckRecordedResponse(t, recorded, testCase.JSONResponseParams)
	}
}
