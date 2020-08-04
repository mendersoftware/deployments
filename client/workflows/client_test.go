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

package workflows

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mendersoftware/deployments/model"
	"github.com/stretchr/testify/assert"
)

func TestGenerateArtifactFails(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}
	srv := httptest.NewServer(http.HandlerFunc(handler))

	workflowsClient := NewClient().(*client)
	workflowsClient.baseURL = srv.URL

	multipartGenerateImage := &model.MultipartGenerateImageMsg{
		Name:                  "name",
		Description:           "description",
		DeviceTypesCompatible: []string{"Beagle Bone"},
		Type:                  "single_file",
		Args:                  "",
		Size:                  10,
		TenantID:              "tenant_id",
		ArtifactID:            "artifact_id",
		FileReader:            bytes.NewReader([]byte("123456790")),
	}

	ctx := context.Background()
	err := workflowsClient.StartGenerateArtifact(ctx, multipartGenerateImage)
	assert.Error(t, err)
	assert.EqualError(t, err, "failed to start workflow: generate_artifact")
}

func TestGenerateArtifactSuccessful(t *testing.T) {
	reqChan := make(chan *model.MultipartGenerateImageMsg, 1)
	handler := func(w http.ResponseWriter, r *http.Request) {
		var multipartGenerateImage model.MultipartGenerateImageMsg
		defer w.WriteHeader(http.StatusCreated)
		b, err := ioutil.ReadAll(r.Body)
		if !assert.NoError(t, err) {
			t.FailNow()
		}

		err = json.Unmarshal(b, &multipartGenerateImage)
		if !assert.NoError(t, err) {
			t.FailNow()
		}

		select {
		case reqChan <- &multipartGenerateImage:
		default:
			t.FailNow()
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(handler))

	workflowsClient := NewClient().(*client)
	workflowsClient.baseURL = srv.URL

	multipartGenerateImage := &model.MultipartGenerateImageMsg{
		Name:                  "name",
		Description:           "description",
		DeviceTypesCompatible: []string{"Beagle Bone"},
		Type:                  "single_file",
		Args:                  "args",
		Size:                  10,
		TenantID:              "tenant_id",
		ArtifactID:            "artifact_id",
		FileReader:            bytes.NewReader([]byte("123456790")),
	}

	ctx := context.Background()
	err := workflowsClient.StartGenerateArtifact(ctx, multipartGenerateImage)
	assert.Nil(t, err)
	select {
	case multipartGenerateImage = <-reqChan:

	default:
		panic("[PROG ERR] Did not receive any response from httptest handler")
	}
	assert.NoError(t, err)
	assert.Equal(t, "name", multipartGenerateImage.Name)
	assert.Equal(t, "description", multipartGenerateImage.Description)
	assert.Equal(t, int64(10), multipartGenerateImage.Size)
	assert.Len(t, multipartGenerateImage.DeviceTypesCompatible, 1)
	assert.Equal(t, "Beagle Bone", multipartGenerateImage.DeviceTypesCompatible[0])
	assert.Equal(t, "single_file", multipartGenerateImage.Type)
	assert.Equal(t, "args", multipartGenerateImage.Args)
	assert.Equal(t, "tenant_id", multipartGenerateImage.TenantID)
	assert.Equal(t, "artifact_id", multipartGenerateImage.ArtifactID)
}
