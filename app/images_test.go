// Copyright 2019 Northern.tech AS
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

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/mendersoftware/deployments/client/workflows"
	workflows_mocks "github.com/mendersoftware/deployments/client/workflows/mocks"
	"github.com/mendersoftware/deployments/model"
	fs_mocks "github.com/mendersoftware/deployments/s3/mocks"
	"github.com/mendersoftware/deployments/store/mocks"
	h "github.com/mendersoftware/deployments/utils/testing"
	"github.com/mendersoftware/go-lib-micro/identity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGenerateImageError(t *testing.T) {
	db := mocks.DataStore{}
	fs := &fs_mocks.FileStorage{}
	d := NewDeployments(&db, fs, ArtifactContentType)

	testCases := []struct {
		multipartGenerateImage *model.MultipartGenerateImageMsg
		expectedError          error
	}{
		{
			multipartGenerateImage: nil,
			expectedError:          ErrModelMultipartUploadMsgMalformed,
		},
		{
			multipartGenerateImage: &model.MultipartGenerateImageMsg{
				Size: MaxImageSize + 1,
			},
			expectedError: ErrModelArtifactFileTooLarge,
		},
	}

	ctx := context.Background()
	for i := range testCases {
		tc := testCases[i]
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			artifactID, err := d.GenerateImage(ctx, tc.multipartGenerateImage)

			assert.Equal(t, artifactID, "")
			assert.Error(t, err)
			assert.EqualError(t, err, tc.expectedError.Error())
		})
	}
}

func TestGenerateImageArtifactIsNotUnique(t *testing.T) {
	db := mocks.DataStore{}
	fs := &fs_mocks.FileStorage{}
	d := NewDeployments(&db, fs, ArtifactContentType)

	db.On("IsArtifactUnique",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("[]string"),
	).Return(false, nil)

	multipartGenerateImage := &model.MultipartGenerateImageMsg{
		Name:                  "name",
		Description:           "description",
		DeviceTypesCompatible: []string{"Beagle Bone"},
		Type:                  "single_file",
		Args:                  "",
		Size:                  10,
		FileReader:            nil,
	}

	ctx := context.Background()
	artifactID, err := d.GenerateImage(ctx, multipartGenerateImage)

	assert.Equal(t, artifactID, "")
	assert.Error(t, err)
	assert.EqualError(t, err, ErrModelArtifactNotUnique.Error())

	db.AssertExpectations(t)
}

func TestGenerateImageErrorWhileCheckingIfArtifactIsNotUnique(t *testing.T) {
	db := mocks.DataStore{}
	fs := &fs_mocks.FileStorage{}
	d := NewDeployments(&db, fs, ArtifactContentType)

	db.On("IsArtifactUnique",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("[]string"),
	).Return(false, errors.New("error"))

	multipartGenerateImage := &model.MultipartGenerateImageMsg{
		Name:                  "name",
		Description:           "description",
		DeviceTypesCompatible: []string{"Beagle Bone"},
		Type:                  "single_file",
		Args:                  "",
		Size:                  10,
		FileReader:            nil,
	}

	ctx := context.Background()
	artifactID, err := d.GenerateImage(ctx, multipartGenerateImage)

	assert.Equal(t, artifactID, "")
	assert.Error(t, err)
	assert.EqualError(t, err, "Fail to check if artifact is unique: error")

	db.AssertExpectations(t)
}

func TestGenerateImageErrorWhileUploading(t *testing.T) {
	db := mocks.DataStore{}
	fs := &fs_mocks.FileStorage{}
	d := NewDeployments(&db, fs, ArtifactContentType)

	fs.On("UploadArtifact",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("int64"),
		mock.AnythingOfType("*io.LimitedReader"),
		mock.AnythingOfType("string"),
	).Return(errors.New("error while uploading"))

	db.On("IsArtifactUnique",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("[]string"),
	).Return(true, nil)

	multipartGenerateImage := &model.MultipartGenerateImageMsg{
		Name:                  "name",
		Description:           "description",
		DeviceTypesCompatible: []string{"Beagle Bone"},
		Type:                  "single_file",
		Args:                  "",
		Size:                  10,
		FileReader:            bytes.NewReader([]byte("123456790")),
	}

	ctx := context.Background()
	artifactID, err := d.GenerateImage(ctx, multipartGenerateImage)

	assert.Equal(t, artifactID, "")
	assert.Error(t, err)
	assert.EqualError(t, err, "error while uploading")

	db.AssertExpectations(t)
	fs.AssertExpectations(t)
}

func TestGenerateImageErrorWhileStartingWorkflow(t *testing.T) {
	db := mocks.DataStore{}
	fs := &fs_mocks.FileStorage{}
	d := NewDeployments(&db, fs, ArtifactContentType)

	mockHTTPClient := &workflows_mocks.HTTPClientMock{}
	mockHTTPClient.On("Do",
		mock.AnythingOfType("*http.Request"),
	).Return(&http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       ioutil.NopCloser(strings.NewReader("")),
	}, nil)

	workflowsClient := workflows.NewClient()
	workflowsClient.SetHTTPClient(mockHTTPClient)
	d.SetWorkflowsClient(workflowsClient)

	fs.On("UploadArtifact",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("int64"),
		mock.AnythingOfType("*io.LimitedReader"),
		mock.AnythingOfType("string"),
	).Return(nil)

	fs.On("Delete",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
	).Return(nil)

	db.On("IsArtifactUnique",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("[]string"),
	).Return(true, nil)

	multipartGenerateImage := &model.MultipartGenerateImageMsg{
		Name:                  "name",
		Description:           "description",
		DeviceTypesCompatible: []string{"Beagle Bone"},
		Type:                  "single_file",
		Args:                  "",
		Size:                  10,
		FileReader:            bytes.NewReader([]byte("123456790")),
	}

	ctx := context.Background()
	artifactID, err := d.GenerateImage(ctx, multipartGenerateImage)

	assert.Equal(t, artifactID, "")
	assert.Error(t, err)
	assert.EqualError(t, err, "failed to start workflow: generate_artifact")

	db.AssertExpectations(t)
	fs.AssertExpectations(t)
	mockHTTPClient.AssertExpectations(t)
}

func TestGenerateImageErrorWhileStartingWorkflowAndFailsWhenCleaningUp(t *testing.T) {
	db := mocks.DataStore{}
	fs := &fs_mocks.FileStorage{}
	d := NewDeployments(&db, fs, ArtifactContentType)

	mockHTTPClient := &workflows_mocks.HTTPClientMock{}
	mockHTTPClient.On("Do",
		mock.AnythingOfType("*http.Request"),
	).Return(&http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       ioutil.NopCloser(strings.NewReader("")),
	}, nil)

	workflowsClient := workflows.NewClient()
	workflowsClient.SetHTTPClient(mockHTTPClient)
	d.SetWorkflowsClient(workflowsClient)

	fs.On("UploadArtifact",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("int64"),
		mock.AnythingOfType("*io.LimitedReader"),
		mock.AnythingOfType("string"),
	).Return(nil)

	fs.On("Delete",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
	).Return(errors.New("unable to remove the file"))

	db.On("IsArtifactUnique",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("[]string"),
	).Return(true, nil)

	multipartGenerateImage := &model.MultipartGenerateImageMsg{
		Name:                  "name",
		Description:           "description",
		DeviceTypesCompatible: []string{"Beagle Bone"},
		Type:                  "single_file",
		Args:                  "",
		Size:                  10,
		FileReader:            bytes.NewReader([]byte("123456790")),
	}

	ctx := context.Background()
	artifactID, err := d.GenerateImage(ctx, multipartGenerateImage)

	assert.Equal(t, artifactID, "")
	assert.Error(t, err)
	assert.EqualError(t, err, "unable to remove the file: failed to start workflow: generate_artifact")

	db.AssertExpectations(t)
	fs.AssertExpectations(t)
	mockHTTPClient.AssertExpectations(t)
}

func TestGenerateImageSuccessful(t *testing.T) {
	db := mocks.DataStore{}
	fs := &fs_mocks.FileStorage{}
	d := NewDeployments(&db, fs, ArtifactContentType)

	mockHTTPClient := &workflows_mocks.HTTPClientMock{}
	mockHTTPClient.On("Do",
		mock.MatchedBy(func(req *http.Request) bool {
			b, err := ioutil.ReadAll(req.Body)
			if err != nil {
				return false
			}
			multipartGenerateImage := &model.MultipartGenerateImageMsg{}
			err = json.Unmarshal(b, &multipartGenerateImage)
			if err != nil {
				return false
			}
			assert.Equal(t, "name", multipartGenerateImage.Name)
			assert.Equal(t, "description", multipartGenerateImage.Description)
			assert.Equal(t, int64(10), multipartGenerateImage.Size)
			assert.Len(t, multipartGenerateImage.DeviceTypesCompatible, 1)
			assert.Equal(t, "Beagle Bone", multipartGenerateImage.DeviceTypesCompatible[0])
			assert.Equal(t, "single_file", multipartGenerateImage.Type)
			assert.Equal(t, "args", multipartGenerateImage.Args)
			assert.Empty(t, multipartGenerateImage.TenantID)
			assert.NotEmpty(t, multipartGenerateImage.ArtifactID)
			return true
		}),
	).Return(&http.Response{
		StatusCode: http.StatusCreated,
		Body:       ioutil.NopCloser(strings.NewReader("")),
	}, nil)

	workflowsClient := workflows.NewClient()
	workflowsClient.SetHTTPClient(mockHTTPClient)
	d.SetWorkflowsClient(workflowsClient)

	fs.On("UploadArtifact",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("int64"),
		mock.AnythingOfType("*io.LimitedReader"),
		mock.AnythingOfType("string"),
	).Return(nil)

	db.On("IsArtifactUnique",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("[]string"),
	).Return(true, nil)

	multipartGenerateImage := &model.MultipartGenerateImageMsg{
		Name:                  "name",
		Description:           "description",
		DeviceTypesCompatible: []string{"Beagle Bone"},
		Type:                  "single_file",
		Args:                  "args",
		Size:                  10,
		FileReader:            bytes.NewReader([]byte("123456790")),
	}

	ctx := context.Background()
	artifactID, err := d.GenerateImage(ctx, multipartGenerateImage)

	assert.NotEqual(t, artifactID, "")
	assert.Nil(t, err)

	db.AssertExpectations(t)
	fs.AssertExpectations(t)
	mockHTTPClient.AssertExpectations(t)
}

func TestGenerateImageSuccessfulWithTenant(t *testing.T) {
	db := mocks.DataStore{}
	fs := &fs_mocks.FileStorage{}
	d := NewDeployments(&db, fs, ArtifactContentType)

	mockHTTPClient := &workflows_mocks.HTTPClientMock{}
	mockHTTPClient.On("Do",
		mock.MatchedBy(func(req *http.Request) bool {
			b, err := ioutil.ReadAll(req.Body)
			if err != nil {
				return false
			}
			multipartGenerateImage := &model.MultipartGenerateImageMsg{}
			err = json.Unmarshal(b, &multipartGenerateImage)
			if err != nil {
				return false
			}
			assert.Equal(t, "name", multipartGenerateImage.Name)
			assert.Equal(t, "description", multipartGenerateImage.Description)
			assert.Equal(t, int64(10), multipartGenerateImage.Size)
			assert.Len(t, multipartGenerateImage.DeviceTypesCompatible, 1)
			assert.Equal(t, "Beagle Bone", multipartGenerateImage.DeviceTypesCompatible[0])
			assert.Equal(t, "single_file", multipartGenerateImage.Type)
			assert.Equal(t, "args", multipartGenerateImage.Args)
			assert.Equal(t, "tenant_id", multipartGenerateImage.TenantID)
			assert.NotEmpty(t, multipartGenerateImage.ArtifactID)
			return true
		}),
	).Return(&http.Response{
		StatusCode: http.StatusCreated,
		Body:       ioutil.NopCloser(strings.NewReader("")),
	}, nil)

	workflowsClient := workflows.NewClient()
	workflowsClient.SetHTTPClient(mockHTTPClient)
	d.SetWorkflowsClient(workflowsClient)

	fs.On("UploadArtifact",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("int64"),
		mock.AnythingOfType("*io.LimitedReader"),
		mock.AnythingOfType("string"),
	).Return(nil)

	db.On("IsArtifactUnique",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("[]string"),
	).Return(true, nil)

	multipartGenerateImage := &model.MultipartGenerateImageMsg{
		Name:                  "name",
		Description:           "description",
		DeviceTypesCompatible: []string{"Beagle Bone"},
		Type:                  "single_file",
		Args:                  "args",
		Size:                  10,
		FileReader:            bytes.NewReader([]byte("123456790")),
	}

	ctx := context.Background()
	identityObject := &identity.Identity{Tenant: "tenant_id"}
	ctxWithIdentity := identity.WithContext(ctx, identityObject)
	artifactID, err := d.GenerateImage(ctxWithIdentity, multipartGenerateImage)

	assert.NotEqual(t, artifactID, "")
	assert.Nil(t, err)

	db.AssertExpectations(t)
	fs.AssertExpectations(t)
	mockHTTPClient.AssertExpectations(t)
}
