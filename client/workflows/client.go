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
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/pkg/errors"

	dconfig "github.com/mendersoftware/deployments/config"
	"github.com/mendersoftware/deployments/model"
)

const (
	generateArtifactURL = "/api/workflow/generate_artifact"
	workflowTimeout     = 5 * time.Second
)

// HTTPClient is the HTTP client used to send requests to the workflows server
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is the workflows client
type Client interface {
	SetHTTPClient(httpClient HTTPClient)
	StartGenerateArtifact(ctx context.Context, multipartGenerateImageMsg *model.MultipartGenerateImageMsg) error
}

// NewClient returns a new workflows client
func NewClient() Client {
	workflowsBaseURL := config.Config.GetString(dconfig.SettingWorkflows)
	return &client{
		baseURL:    workflowsBaseURL,
		httpClient: &http.Client{Timeout: workflowTimeout},
	}
}

type client struct {
	baseURL    string
	httpClient HTTPClient
}

func (c *client) SetHTTPClient(httpClient HTTPClient) {
	c.httpClient = httpClient
}

func (c *client) StartGenerateArtifact(ctx context.Context, multipartGenerateImageMsg *model.MultipartGenerateImageMsg) error {
	l := log.FromContext(ctx)
	l.Debugf("Submit generate artifact: tenantID=%s, artifactID=%s",
		multipartGenerateImageMsg.TenantID, multipartGenerateImageMsg.ArtifactID)

	workflowsURL := c.baseURL + generateArtifactURL

	payload, _ := json.Marshal(multipartGenerateImageMsg)
	req, err := http.NewRequest("POST", workflowsURL, strings.NewReader(string(payload)))
	if err != nil {
		return err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to start workflow: generate_artifact")
	}
	if res.StatusCode != http.StatusCreated {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			body = []byte("<failed to read>")
		}
		l.Errorf("generate artifact failed with status %v, response text: %s",
			res.StatusCode, body)
		return errors.New("failed to start workflow: generate_artifact")
	}
	return nil
}
