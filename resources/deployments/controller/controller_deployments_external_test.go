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
