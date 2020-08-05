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

package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

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

func TestGenerateImageErrorS3GetRequest(t *testing.T) {
	db := mocks.DataStore{}
	fs := &fs_mocks.FileStorage{}
	d := NewDeployments(&db, fs, ArtifactContentType)

	fs.On("UploadArtifact",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("*io.LimitedReader"),
		mock.AnythingOfType("string"),
	).Return(nil)

	db.On("IsArtifactUnique",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("[]string"),
	).Return(true, nil)

	fs.On("GetRequest",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("time.Duration"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
	).Return(nil, errors.New("error get request"))

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
	assert.EqualError(t, err, "error get request")

	db.AssertExpectations(t)
	fs.AssertExpectations(t)
}

func TestGenerateImageErrorS3DeleteRequest(t *testing.T) {
	db := mocks.DataStore{}
	fs := &fs_mocks.FileStorage{}
	d := NewDeployments(&db, fs, ArtifactContentType)

	fs.On("UploadArtifact",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("*io.LimitedReader"),
		mock.AnythingOfType("string"),
	).Return(nil)

	db.On("IsArtifactUnique",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("[]string"),
	).Return(true, nil)

	fs.On("GetRequest",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("time.Duration"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
	).Return(&model.Link{
		Uri: "GET",
	}, nil)

	fs.On("DeleteRequest",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("time.Duration"),
	).Return(nil, errors.New("error delete request"))

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
	assert.EqualError(t, err, "error delete request")

	db.AssertExpectations(t)
	fs.AssertExpectations(t)
}

func TestGenerateImageErrorWhileStartingWorkflow(t *testing.T) {
	generateErr := errors.New("failed to start workflow: generate_artifact")
	db := mocks.DataStore{}
	fs := &fs_mocks.FileStorage{}
	d := NewDeployments(&db, fs, ArtifactContentType)

	fs.On("GetRequest",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("time.Duration"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
	).Return(&model.Link{
		Uri: "GET",
	}, nil)

	fs.On("DeleteRequest",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("time.Duration"),
		mock.AnythingOfType("string"),
	).Return(&model.Link{
		Uri: "DELETE",
	}, nil)

	workflowsClient := &workflows_mocks.Client{}
	workflowsClient.On("StartGenerateArtifact",
		h.ContextMatcher(),
		mock.AnythingOfType("*model.MultipartGenerateImageMsg"),
	).Return(generateErr)
	d.SetWorkflowsClient(workflowsClient)

	fs.On("UploadArtifact",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
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
	assert.EqualError(t, err, generateErr.Error())

	db.AssertExpectations(t)
	fs.AssertExpectations(t)
	workflowsClient.AssertExpectations(t)
}

func TestGenerateImageErrorWhileStartingWorkflowAndFailsWhenCleaningUp(t *testing.T) {
	db := mocks.DataStore{}
	fs := &fs_mocks.FileStorage{}
	d := NewDeployments(&db, fs, ArtifactContentType)

	workflowsClient := &workflows_mocks.Client{}
	d.SetWorkflowsClient(workflowsClient)

	workflowsClient.On("StartGenerateArtifact",
		h.ContextMatcher(),
		mock.AnythingOfType("*model.MultipartGenerateImageMsg"),
	).Return(errors.New("failed to start workflow: generate_artifact"))

	fs.On("GetRequest",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("time.Duration"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
	).Return(&model.Link{
		Uri: "GET",
	}, nil)

	fs.On("DeleteRequest",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("time.Duration"),
		mock.AnythingOfType("string"),
	).Return(&model.Link{
		Uri: "DELETE",
	}, nil)

	fs.On("UploadArtifact",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
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
	workflowsClient.AssertExpectations(t)
}

func TestGenerateImageSuccessful(t *testing.T) {
	ctx := context.Background()
	db := mocks.DataStore{}
	fs := &fs_mocks.FileStorage{}
	d := NewDeployments(&db, fs, ArtifactContentType)

	multipartGenerateImage := &model.MultipartGenerateImageMsg{
		Name:                  "name",
		Description:           "description",
		DeviceTypesCompatible: []string{"Beagle Bone"},
		Type:                  "single_file",
		Args:                  "args",
		Size:                  10,
		FileReader:            bytes.NewReader([]byte("123456790")),
	}

	workflowsClient := &workflows_mocks.Client{}
	d.SetWorkflowsClient(workflowsClient)

	workflowsClient.On("StartGenerateArtifact",
		h.ContextMatcher(),
		multipartGenerateImage,
	).Return(nil)

	fs.On("GetRequest",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("time.Duration"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
	).Return(&model.Link{
		Uri: "GET",
	}, nil)

	fs.On("DeleteRequest",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("time.Duration"),
		mock.AnythingOfType("string"),
	).Return(&model.Link{
		Uri: "DELETE",
	}, nil)

	fs.On("UploadArtifact",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("*io.LimitedReader"),
		mock.AnythingOfType("string"),
	).Return(nil)

	db.On("IsArtifactUnique",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("[]string"),
	).Return(true, nil)

	artifactID, err := d.GenerateImage(ctx, multipartGenerateImage)

	assert.NotEqual(t, artifactID, "")
	assert.Nil(t, err)

	db.AssertExpectations(t)
	fs.AssertExpectations(t)
	workflowsClient.AssertExpectations(t)
}

func TestGenerateImageSuccessfulWithTenant(t *testing.T) {
	ctx := context.Background()
	db := mocks.DataStore{}
	fs := &fs_mocks.FileStorage{}
	d := NewDeployments(&db, fs, ArtifactContentType)

	multipartGenerateImage := &model.MultipartGenerateImageMsg{
		Name:                  "name",
		Description:           "description",
		DeviceTypesCompatible: []string{"Beagle Bone"},
		Type:                  "single_file",
		Args:                  "args",
		Size:                  10,
		FileReader:            bytes.NewReader([]byte("123456790")),
	}

	workflowsClient := &workflows_mocks.Client{}
	d.SetWorkflowsClient(workflowsClient)
	workflowsClient.On("StartGenerateArtifact",
		h.ContextMatcher(), multipartGenerateImage,
	).Return(nil)

	fs.On("GetRequest",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("time.Duration"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
	).Return(&model.Link{
		Uri: "GET",
	}, nil)

	fs.On("DeleteRequest",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("time.Duration"),
		mock.AnythingOfType("string"),
	).Return(&model.Link{
		Uri: "DELETE",
	}, nil)

	fs.On("UploadArtifact",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("*io.LimitedReader"),
		mock.AnythingOfType("string"),
	).Return(nil)

	db.On("IsArtifactUnique",
		h.ContextMatcher(),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("[]string"),
	).Return(true, nil)

	identityObject := &identity.Identity{Tenant: "tenant_id"}
	ctxWithIdentity := identity.WithContext(ctx, identityObject)
	artifactID, err := d.GenerateImage(ctxWithIdentity, multipartGenerateImage)

	assert.NotEqual(t, artifactID, "")
	assert.Nil(t, err)

	db.AssertExpectations(t)
	fs.AssertExpectations(t)
	workflowsClient.AssertExpectations(t)
}
