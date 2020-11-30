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
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	"github.com/pkg/errors"

	"github.com/mendersoftware/deployments/app"
	app_mocks "github.com/mendersoftware/deployments/app/mocks"
	"github.com/mendersoftware/deployments/model"
	store_mocks "github.com/mendersoftware/deployments/store/mocks"
	store_mongo "github.com/mendersoftware/deployments/store/mongo"
	"github.com/mendersoftware/deployments/utils/restutil/view"
	h "github.com/mendersoftware/deployments/utils/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPostArtifacts(t *testing.T) {
	type request struct {
		Description string `json:"description"`
		Size        int64  `json:"size"`
		ArtifactID  string `json:"artifact_id"`
	}

	var testConflictError = model.NewConflictError(
		store_mongo.ErrMsgConflictingDepends,
		`{meta_artifact.artifact_name: "foobar", `+
			`meta_artifact.depends_idx: {`+
			`"device_type": "arm6", "checksum": "2"}}`,
	)

	imageBody := []byte("123456790")

	testCases := []struct {
		requestBodyObject      []h.Part
		requestContentType     string
		responseCode           int
		responseBody           string
		appCreateImage         bool
		appCreateImageResponse string
		appCreateImageError    error
	}{
		{
			requestBodyObject:  []h.Part{},
			requestContentType: "",
			responseCode:       http.StatusBadRequest,
			responseBody:       "request Content-Type isn't multipart/form-data",
		},
		{
			requestBodyObject:  []h.Part{},
			requestContentType: "application/x-www-form-urlencoded",
			responseCode:       http.StatusBadRequest,
			responseBody:       "request Content-Type isn't multipart/form-data",
		},
		{
			requestBodyObject:  []h.Part{},
			requestContentType: "multipart/form-data",
			responseCode:       http.StatusBadRequest,
			responseBody:       ErrArtifactFileMissing.Error(),
		},
		{
			requestBodyObject: []h.Part{
				{
					FieldName:  "description",
					FieldValue: "description",
				},
				{
					FieldName:  "size",
					FieldValue: strconv.Itoa(len(imageBody)),
				},
				{
					FieldName:  "artifact_id",
					FieldValue: "wrong_uuidv4",
				},
				{
					FieldName:   "artifact",
					ContentType: "application/octet-stream",
					ImageData:   imageBody,
				},
			},
			requestContentType: "multipart/form-data",
			responseCode:       http.StatusBadRequest,
			responseBody:       "artifact_id is not a valid UUID",
		},
		{
			requestBodyObject: []h.Part{
				{
					FieldName:  "id",
					FieldValue: "5e2fbcf6a6a7eca56cbc9476",
				},
				{
					FieldName:  "artifact_id",
					FieldValue: "24436884-a710-4d20-aec4-82c89fbfe29e",
				},
				{
					FieldName:  "description",
					FieldValue: "description",
				},
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
			requestContentType:     "multipart/form-data",
			responseCode:           http.StatusCreated,
			responseBody:           "",
			appCreateImage:         true,
			appCreateImageResponse: "24436884-a710-4d20-aec4-82c89fbfe29e",
			appCreateImageError:    nil,
		},
		{
			requestBodyObject: []h.Part{
				{
					FieldName:  "id",
					FieldValue: "5e2fbcf6a6a7eca56cbc9476",
				},
				{
					FieldName:  "artifact_id",
					FieldValue: "24436884-a710-4d20-aec4-82c89fbfe29e",
				},
				{
					FieldName:  "description",
					FieldValue: "description",
				},
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
			requestContentType: "multipart/form-data",
			responseCode:       http.StatusConflict,
			// no slashes will be present in the real response - must be added
			// because we're comparing to body, _ := recorded.DecodedBody(), which does does funny formatting
			responseBody:           store_mongo.ErrMsgConflictingDepends,
			appCreateImage:         true,
			appCreateImageResponse: "24436884-a710-4d20-aec4-82c89fbfe29e",
			appCreateImageError:    testConflictError,
		},
	}

	store := &store_mocks.DataStore{}
	restView := new(view.RESTView)

	for i := range testCases {
		tc := testCases[i]

		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			app := &app_mocks.App{}

			if tc.appCreateImage {
				app.On("CreateImage",
					h.ContextMatcher(),
					mock.MatchedBy(func(msg *model.MultipartUploadMsg) bool {
						assert.Equal(t, msg.ArtifactID, tc.requestBodyObject[1].FieldValue)
						assert.Equal(t, msg.MetaConstructor.Description, tc.requestBodyObject[2].FieldValue)

						return true
					}),
				).Return(tc.appCreateImageResponse, tc.appCreateImageError)
			}

			d := NewDeploymentsApiHandlers(store, restView, app)
			api := setUpRestTest("/api/0.0.1/artifacts", rest.Post, d.NewImage)
			req := h.MakeMultipartRequest("POST", "http://localhost/api/0.0.1/artifacts",
				tc.requestContentType, tc.requestBodyObject)
			req.Header.Set("Authorization", HTTPHeaderAuthorizationBearer+" TOKEN")

			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.responseCode)
			if tc.responseBody == "" {
				recorded.BodyIs(tc.responseBody)
			} else {
				body, _ := recorded.DecodedBody()
				assert.Contains(t, string(body), tc.responseBody,
					`"%s" not in "%s"`, string(body), tc.responseBody)
			}

			if tc.appCreateImage {
				app.AssertExpectations(t)
			}
		})
	}

}

func TestPostArtifactsInternal(t *testing.T) {
	type request struct {
		Description string `json:"description"`
		Size        int64  `json:"size"`
		ArtifactID  string `json:"artifact_id"`
	}

	imageBody := []byte("123456790")
	var testConflictError = model.NewConflictError(
		store_mongo.ErrMsgConflictingDepends,
		`{meta_artifact.artifact_name: "foobar", `+
			`meta_artifact.depends_idx: {`+
			`"device_type": "arm6", "checksum": "2"}}`,
	)

	testCases := []struct {
		requestBodyObject      []h.Part
		requestContentType     string
		responseCode           int
		responseBody           string
		appCreateImage         bool
		appCreateImageResponse string
		appCreateImageError    error
	}{
		{
			requestBodyObject:  []h.Part{},
			requestContentType: "",
			responseCode:       http.StatusBadRequest,
			responseBody:       "request Content-Type isn't multipart/form-data",
		},
		{
			requestBodyObject:  []h.Part{},
			requestContentType: "application/x-www-form-urlencoded",
			responseCode:       http.StatusBadRequest,
			responseBody:       "request Content-Type isn't multipart/form-data",
		},
		{
			requestBodyObject:  []h.Part{},
			requestContentType: "multipart/form-data",
			responseCode:       http.StatusBadRequest,
			responseBody:       ErrArtifactFileMissing.Error(),
		},
		{
			requestBodyObject: []h.Part{
				{
					FieldName:  "description",
					FieldValue: "description",
				},
				{
					FieldName:  "size",
					FieldValue: strconv.Itoa(len(imageBody)),
				},
				{
					FieldName:  "artifact_id",
					FieldValue: "wrong_uuidv4",
				},
				{
					FieldName:   "artifact",
					ContentType: "application/octet-stream",
					ImageData:   imageBody,
				},
			},
			requestContentType: "multipart/form-data",
			responseCode:       http.StatusBadRequest,
			responseBody:       "artifact_id is not a valid UUID",
		},
		{
			requestBodyObject: []h.Part{
				{
					FieldName:  "id",
					FieldValue: "5e2fbcf6a6a7eca56cbc9476",
				},
				{
					FieldName:  "artifact_id",
					FieldValue: "24436884-a710-4d20-aec4-82c89fbfe29e",
				},
				{
					FieldName:  "description",
					FieldValue: "description",
				},
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
			requestContentType:     "multipart/form-data",
			responseCode:           http.StatusCreated,
			responseBody:           "",
			appCreateImage:         true,
			appCreateImageResponse: "24436884-a710-4d20-aec4-82c89fbfe29e",
			appCreateImageError:    nil,
		},
		{
			requestBodyObject: []h.Part{
				{
					FieldName:  "id",
					FieldValue: "5e2fbcf6a6a7eca56cbc9476",
				},
				{
					FieldName:  "artifact_id",
					FieldValue: "24436884-a710-4d20-aec4-82c89fbfe29e",
				},
				{
					FieldName:  "description",
					FieldValue: "description",
				},
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
			requestContentType: "multipart/form-data",
			responseCode:       http.StatusConflict,
			// no slashes will be present in the real response - must be added
			// because we're comparing to body, _ := recorded.DecodedBody(), which does does funny formatting
			responseBody:           store_mongo.ErrMsgConflictingDepends,
			appCreateImage:         true,
			appCreateImageResponse: "24436884-a710-4d20-aec4-82c89fbfe29e",
			appCreateImageError:    testConflictError,
		},
	}

	store := &store_mocks.DataStore{}
	restView := new(view.RESTView)

	for i := range testCases {
		tc := testCases[i]

		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			app := &app_mocks.App{}

			if tc.appCreateImage {
				app.On("CreateImage",
					h.ContextMatcher(),
					mock.MatchedBy(func(msg *model.MultipartUploadMsg) bool {
						assert.Equal(t, msg.ArtifactID, tc.requestBodyObject[1].FieldValue)
						assert.Equal(t, msg.MetaConstructor.Description, tc.requestBodyObject[2].FieldValue)

						return true
					}),
				).Return(tc.appCreateImageResponse, tc.appCreateImageError)
			}

			d := NewDeploymentsApiHandlers(store, restView, app)
			api := setUpRestTest("/api/0.0.1/tenants/:tenant/artifacts", rest.Post, d.NewImageForTenantHandler)
			req := h.MakeMultipartRequest("POST", "http://localhost/api/0.0.1/tenants/default/artifacts",
				tc.requestContentType, tc.requestBodyObject)
			req.Header.Set("Authorization", HTTPHeaderAuthorizationBearer+" TOKEN")

			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.responseCode)
			if tc.responseBody == "" {
				recorded.BodyIs(tc.responseBody)
			} else {
				body, _ := recorded.DecodedBody()
				assert.Contains(t, string(body), tc.responseBody)
			}

			if tc.appCreateImage {
				app.AssertExpectations(t)
			}
		})
	}
}

func TestPostArtifactsGenerate(t *testing.T) {
	type request struct {
		Name                  string `json:"name"`
		Description           string `json:"description"`
		Size                  int64  `json:"size"`
		DeviceTypesCompatible string `json:"device_types_compatible"`
		Type                  string `json:"type"`
		Args                  string `json:"args"`
	}

	imageBody := []byte("123456790")

	testCases := []struct {
		requestBodyObject        []h.Part
		requestContentType       string
		responseCode             int
		responseBody             string
		appGenerateImage         bool
		appGenerateImageResponse string
		appGenerateImageError    error
	}{
		{
			requestBodyObject:  []h.Part{},
			requestContentType: "",
			responseCode:       http.StatusBadRequest,
			responseBody:       "request Content-Type isn't multipart/form-data",
		},
		{
			requestBodyObject:  []h.Part{},
			requestContentType: "application/x-www-form-urlencoded",
			responseCode:       http.StatusBadRequest,
			responseBody:       "request Content-Type isn't multipart/form-data",
		},
		{
			requestBodyObject:  []h.Part{},
			requestContentType: "multipart/form-data",
			responseCode:       http.StatusBadRequest,
			responseBody:       "api: invalid form parameters:",
		},
		{
			requestBodyObject: []h.Part{
				{
					FieldName:  "name",
					FieldValue: "name",
				},
				{
					FieldName:  "type",
					FieldValue: "single_file",
				},
				{
					FieldName:   "file",
					ContentType: "application/octet-stream",
					ImageData:   imageBody,
				},
			},
			requestContentType: "multipart/form-data",
			responseCode:       http.StatusBadRequest,
			responseBody: "api: invalid form parameters: " +
				"device_types_compatible: non zero value required",
		},
		{
			requestBodyObject: []h.Part{
				{
					FieldName:  "name",
					FieldValue: "name",
				},
				{
					FieldName:  "type",
					FieldValue: "single_file",
				},
				{
					FieldName:  "device_types_compatible",
					FieldValue: "Beagle Bone",
				},
			},
			requestContentType: "multipart/form-data",
			responseCode:       http.StatusBadRequest,
			responseBody:       "api: invalid form parameters: missing 'file' section",
		},
		{
			requestBodyObject: []h.Part{
				{
					FieldName:  "name",
					FieldValue: "name",
				},
				{
					FieldName:  "device_types_compatible",
					FieldValue: "Beagle Bone",
				},
				{
					FieldName:   "file",
					ContentType: "application/octet-stream",
					ImageData:   imageBody,
				},
			},
			requestContentType: "multipart/form-data",
			responseCode:       http.StatusBadRequest,
			responseBody:       "api: invalid form parameters: type: non zero value required",
		},
		{
			requestBodyObject: []h.Part{
				{
					FieldName:  "name",
					FieldValue: "name",
				},
				{
					FieldName:  "description",
					FieldValue: "description",
				},
				{
					FieldName:  "size",
					FieldValue: strconv.Itoa(len(imageBody)),
				},
				{
					FieldName:  "device_types_compatible",
					FieldValue: "Beagle Bone",
				},
				{
					FieldName:  "type",
					FieldValue: "single_file",
				},
				{
					FieldName:  "args",
					FieldValue: "args",
				},
				{
					FieldName:   "file",
					ContentType: "application/octet-stream",
					ImageData:   imageBody,
				},
			},
			requestContentType:       "multipart/form-data",
			responseCode:             http.StatusCreated,
			responseBody:             "",
			appGenerateImage:         true,
			appGenerateImageResponse: "artifactID",
			appGenerateImageError:    nil,
		},
		{
			requestBodyObject: []h.Part{
				{
					FieldName:  "name",
					FieldValue: "name with spaces",
				},
				{
					FieldName:  "description",
					FieldValue: "description with spaces",
				},
				{
					FieldName:  "size",
					FieldValue: strconv.Itoa(len(imageBody)),
				},
				{
					FieldName:  "device_types_compatible",
					FieldValue: "Beagle Bone",
				},
				{
					FieldName:  "type",
					FieldValue: "single_file",
				},
				{
					FieldName:  "args",
					FieldValue: "arg1 arg2 arg3",
				},
				{
					FieldName:   "file",
					ContentType: "application/octet-stream",
					ImageData:   imageBody,
				},
			},
			requestContentType:       "multipart/form-data",
			responseCode:             http.StatusCreated,
			responseBody:             "",
			appGenerateImage:         true,
			appGenerateImageResponse: "artifactID",
			appGenerateImageError:    nil,
		},
		{
			requestBodyObject: []h.Part{
				{
					FieldName:  "name",
					FieldValue: "name",
				},
				{
					FieldName:  "description",
					FieldValue: "description",
				},
				{
					FieldName:  "size",
					FieldValue: strconv.Itoa(len(imageBody)),
				},
				{
					FieldName:  "device_types_compatible",
					FieldValue: "Beagle Bone",
				},
				{
					FieldName:  "type",
					FieldValue: "single_file",
				},
				{
					FieldName:  "args",
					FieldValue: "args",
				},
				{
					FieldName:   "file",
					ContentType: "application/octet-stream",
					ImageData:   imageBody,
				},
			},
			requestContentType:       "multipart/form-data",
			responseCode:             http.StatusUnprocessableEntity,
			responseBody:             "Artifact not unique",
			appGenerateImage:         true,
			appGenerateImageResponse: "",
			appGenerateImageError:    app.ErrModelArtifactNotUnique,
		},
		{
			requestBodyObject: []h.Part{
				{
					FieldName:  "name",
					FieldValue: "name",
				},
				{
					FieldName:  "description",
					FieldValue: "description",
				},
				{
					FieldName:  "size",
					FieldValue: strconv.Itoa(len(imageBody)),
				},
				{
					FieldName:  "device_types_compatible",
					FieldValue: "Beagle Bone",
				},
				{
					FieldName:  "type",
					FieldValue: "single_file",
				},
				{
					FieldName:  "args",
					FieldValue: "args",
				},
				{
					FieldName:   "file",
					ContentType: "application/octet-stream",
					ImageData:   imageBody,
				},
			},
			requestContentType:       "multipart/form-data",
			responseCode:             http.StatusBadRequest,
			responseBody:             "Artifact file too large",
			appGenerateImage:         true,
			appGenerateImageResponse: "",
			appGenerateImageError:    app.ErrModelArtifactFileTooLarge,
		},
		{
			requestBodyObject: []h.Part{
				{
					FieldName:  "name",
					FieldValue: "name",
				},
				{
					FieldName:  "description",
					FieldValue: "description",
				},
				{
					FieldName:  "size",
					FieldValue: strconv.Itoa(len(imageBody)),
				},
				{
					FieldName:  "device_types_compatible",
					FieldValue: "Beagle Bone",
				},
				{
					FieldName:  "type",
					FieldValue: "single_file",
				},
				{
					FieldName:  "args",
					FieldValue: "args",
				},
				{
					FieldName:   "file",
					ContentType: "application/octet-stream",
					ImageData:   imageBody,
				},
			},
			requestContentType:       "multipart/form-data",
			responseCode:             http.StatusBadRequest,
			responseBody:             "Cannot parse artifact file",
			appGenerateImage:         true,
			appGenerateImageResponse: "",
			appGenerateImageError:    app.ErrModelParsingArtifactFailed,
		},
		{
			requestBodyObject: []h.Part{
				{
					FieldName:  "name",
					FieldValue: "name",
				},
				{
					FieldName:  "description",
					FieldValue: "description",
				},
				{
					FieldName:  "size",
					FieldValue: strconv.Itoa(len(imageBody)),
				},
				{
					FieldName:  "device_types_compatible",
					FieldValue: "Beagle Bone",
				},
				{
					FieldName:  "type",
					FieldValue: "single_file",
				},
				{
					FieldName:  "args",
					FieldValue: "args",
				},
				{
					FieldName:   "file",
					ContentType: "application/octet-stream",
					ImageData:   imageBody,
				},
			},
			requestContentType:       "multipart/form-data",
			responseCode:             http.StatusInternalServerError,
			responseBody:             "internal error",
			appGenerateImage:         true,
			appGenerateImageResponse: "",
			appGenerateImageError:    errors.New("generic error"),
		},
	}

	store := &store_mocks.DataStore{}
	restView := new(view.RESTView)

	for i := range testCases {
		tc := testCases[i]

		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			app := &app_mocks.App{}

			if tc.appGenerateImage {
				app.On("GenerateImage",
					h.ContextMatcher(),
					mock.MatchedBy(func(msg *model.MultipartGenerateImageMsg) bool {
						assert.Equal(t, msg.Name, tc.requestBodyObject[0].FieldValue)
						assert.Equal(t, msg.Description, tc.requestBodyObject[1].FieldValue)
						assert.Equal(t, msg.DeviceTypesCompatible, []string{tc.requestBodyObject[3].FieldValue})
						assert.Equal(t, msg.Type, tc.requestBodyObject[4].FieldValue)
						assert.Equal(t, msg.Args, tc.requestBodyObject[5].FieldValue)
						assert.Equal(t, msg.Token, "TOKEN")

						return true
					}),
				).Return(tc.appGenerateImageResponse, tc.appGenerateImageError)
			}

			d := NewDeploymentsApiHandlers(store, restView, app)
			api := setUpRestTest("/api/0.0.1/artifacts/generate", rest.Post, d.GenerateImage)
			req := h.MakeMultipartRequest("POST", "http://localhost/api/0.0.1/artifacts/generate",
				tc.requestContentType, tc.requestBodyObject)
			req.Header.Set("Authorization", HTTPHeaderAuthorizationBearer+" TOKEN")

			recorded := test.RunRequest(t, api.MakeHandler(), req)
			recorded.CodeIs(tc.responseCode)
			if tc.responseBody == "" {
				recorded.BodyIs(tc.responseBody)
			} else {
				body, _ := recorded.DecodedBody()
				assert.Contains(t, string(body), tc.responseBody)
			}

			if tc.appGenerateImage {
				app.AssertExpectations(t)
			}
		})
	}

}
