// Copyright 2022 Northern.tech AS
//
//	Licensed under the Apache License, Version 2.0 (the "License");
//	you may not use this file except in compliance with the License.
//	You may obtain a copy of the License at
//
//	    http://www.apache.org/licenses/LICENSE-2.0
//
//	Unless required by applicable law or agreed to in writing, software
//	distributed under the License is distributed on an "AS IS" BASIS,
//	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	See the License for the specific language governing permissions and
//	limitations under the License.

package workflows

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/mendersoftware/go-lib-micro/identity"
	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/mendersoftware/go-lib-micro/rest_utils"
	"github.com/pkg/errors"

	dconfig "github.com/mendersoftware/deployments/config"
	"github.com/mendersoftware/deployments/model"
)

const (
	healthURL                     = "/api/v1/health"
	generateArtifactURL           = "/api/v1/workflow/generate_artifact"
	reindexReportingURL           = "/api/v1/workflow/reindex_reporting"
	reindexReportingDeploymentURL = "/api/v1/workflow/reindex_reporting_deployment"
	defaultTimeout                = 5 * time.Second
)

// Client is the workflows client
//
//go:generate ../../utils/mockgen.sh
type Client interface {
	CheckHealth(ctx context.Context) error
	StartGenerateArtifact(
		ctx context.Context,
		multipartGenerateImageMsg *model.MultipartGenerateImageMsg,
	) error
	StartReindexReporting(c context.Context, device string) error
	StartReindexReportingDeployment(c context.Context, device, deployment, id string) error
}

// NewClient returns a new workflows client
func NewClient() Client {
	workflowsBaseURL := config.Config.GetString(dconfig.SettingWorkflows)
	return &client{
		baseURL:    workflowsBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

type client struct {
	baseURL    string
	httpClient *http.Client
}

func (c *client) CheckHealth(ctx context.Context) error {
	var (
		apiErr rest_utils.ApiError
		client http.Client
	)

	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}
	req, _ := http.NewRequestWithContext(
		ctx, "GET", c.baseURL+healthURL, nil,
	)

	rsp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode >= http.StatusOK && rsp.StatusCode < 300 {
		return nil
	}
	decoder := json.NewDecoder(rsp.Body)
	err = decoder.Decode(&apiErr)
	if err != nil {
		return errors.Errorf("health check HTTP error: %s", rsp.Status)
	}
	return &apiErr
}

func (c *client) StartGenerateArtifact(
	ctx context.Context,
	multipartGenerateImageMsg *model.MultipartGenerateImageMsg,
) error {
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
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			body = []byte("<failed to read>")
		}
		l.Errorf("generate artifact failed with status %v, response text: %s",
			res.StatusCode, body)
		return errors.New("failed to start workflow: generate_artifact")
	}
	return nil
}

func (c *client) StartReindexReporting(ctx context.Context, device string) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}
	id := identity.FromContext(ctx)
	if id == nil {
		return errors.New("workflows: context lacking tenant identity")
	}
	wflow := ReindexWorkflow{
		RequestID: requestid.FromContext(ctx),
		TenantID:  id.Tenant,
		DeviceID:  device,
		Service:   ServiceDeployments,
	}
	payload, _ := json.Marshal(wflow)
	req, err := http.NewRequestWithContext(ctx,
		"POST",
		c.baseURL+reindexReportingURL,
		bytes.NewReader(payload),
	)
	if err != nil {
		return errors.Wrap(err, "workflows: error preparing HTTP request")
	}

	req.Header.Set("Content-Type", "application/json")

	rsp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "workflows: failed to trigger reporting reindex")
	}
	defer rsp.Body.Close()

	if rsp.StatusCode < 300 {
		return nil
	}

	if rsp.StatusCode == http.StatusNotFound {
		workflowURIparts := strings.Split(reindexReportingURL, "/")
		workflowName := workflowURIparts[len(workflowURIparts)-1]
		return errors.New(`workflows: workflow "` + workflowName + `" not defined`)
	}

	return errors.Errorf(
		"workflows: unexpected HTTP status from workflows service: %s",
		rsp.Status,
	)
}

func (c *client) StartReindexReportingDeployment(ctx context.Context,
	device, deployment, id string) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}
	ident := identity.FromContext(ctx)
	if ident == nil {
		return errors.New("workflows: context lacking tenant identity")
	}
	wflow := ReindexDeploymentWorkflow{
		RequestID:    requestid.FromContext(ctx),
		TenantID:     ident.Tenant,
		DeviceID:     device,
		DeploymentID: deployment,
		ID:           id,
		Service:      ServiceDeployments,
	}
	payload, _ := json.Marshal(wflow)
	req, err := http.NewRequestWithContext(ctx,
		"POST",
		c.baseURL+reindexReportingDeploymentURL,
		bytes.NewReader(payload),
	)
	if err != nil {
		return errors.Wrap(err, "workflows: error preparing HTTP request")
	}

	req.Header.Set("Content-Type", "application/json")

	rsp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "workflows: failed to trigger reporting reindex deployment")
	}
	defer rsp.Body.Close()

	if rsp.StatusCode < 300 {
		return nil
	}

	if rsp.StatusCode == http.StatusNotFound {
		workflowURIparts := strings.Split(reindexReportingDeploymentURL, "/")
		workflowName := workflowURIparts[len(workflowURIparts)-1]
		return errors.New(`workflows: workflow "` + workflowName + `" not defined`)
	}

	return errors.Errorf(
		"workflows: unexpected HTTP status from workflows service: %s",
		rsp.Status,
	)
}
