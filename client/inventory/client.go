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

package inventory

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/pkg/errors"

	dconfig "github.com/mendersoftware/deployments/config"
	"github.com/mendersoftware/deployments/model"
)

const (
	searchURL      = "/api/internal/v2/inventory/tenants/:tenantId/filters/search"
	areInGroupURL  = "/api/internal/v1/inventory/devices/:tenantId/ingroup/:name"
	defaultTimeout = 10
)

// Errors
var (
	ErrFilterNotFound = errors.New("Filter with given ID not found in the inventory.")
)

// HTTPClient is the HTTP client used to send requests to the inventory service
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is the inventory client
type Client interface {
	Search(ctx context.Context, tenantId string, searchParams model.SearchParams) ([]model.InvDevice, int, error)
	AreDevicesInGroup(ctx context.Context, devices []string, group string, tenantId string) bool
}

// NewClient returns a new inventory client
func NewClient() Client {
	baseURL := config.Config.GetString(dconfig.SettingInventoryAddr)
	timeout := config.Config.GetString(dconfig.SettingInventoryTimeout)

	t, err := strconv.Atoi(timeout)
	if err != nil {
		t = defaultTimeout
	}

	return &client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: time.Duration(t) * time.Second},
	}
}

type client struct {
	baseURL    string
	httpClient HTTPClient
}

func (c *client) Search(ctx context.Context, tenantId string, searchParams model.SearchParams) ([]model.InvDevice, int, error) {
	l := log.FromContext(ctx)
	l.Debugf("Search")

	repl := strings.NewReplacer(":tenantId", tenantId)
	url := c.baseURL + repl.Replace(searchURL)

	payload, _ := json.Marshal(searchParams)
	req, err := http.NewRequest("POST", url, strings.NewReader(string(payload)))
	if err != nil {
		return nil, -1, err
	}
	req.Header.Set("Content-Type", "application/json")

	rsp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, -1, errors.Wrap(err, "search devices request failed")
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return nil, -1, errors.Errorf("search devices request failed with unexpected status %v", rsp.StatusCode)
	}

	devs := []model.InvDevice{}
	if err := json.NewDecoder(rsp.Body).Decode(&devs); err != nil {
		return nil, -1, errors.Wrap(err, "error parsing search devices response")
	}

	totalCountStr := rsp.Header.Get("X-Total-Count")
	totalCount, err := strconv.Atoi(totalCountStr)
	if err != nil {
		return nil, -1, errors.Wrap(err, "error parsing X-Total-Count header")
	}

	return devs, totalCount, nil
}

func (c *client) AreDevicesInGroup(ctx context.Context, devices []string, group string, tenantId string) bool {
	l := log.FromContext(ctx)
	l.Debugf("AreDevicesInGroup starting")

	repl := strings.NewReplacer(":tenantId", tenantId, ":name", group)
	url := c.baseURL + repl.Replace(areInGroupURL)

	deviceIds := model.DeviceIds{
		Devices: devices,
	}
	payload, _ := json.Marshal(deviceIds)
	req, err := http.NewRequest("POST", url, strings.NewReader(string(payload)))
	if err != nil {
		return false
	}

	req.Header.Set("Content-Type", "application/json")
	rsp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return false
	}

	return true
}
