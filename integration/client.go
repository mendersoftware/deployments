// Copyright 2017 Northern.tech AS
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

package integration

import (
	"errors"
	"net/http"

	"github.com/asaskevich/govalidator"
)

// MenderAPIOption is the type of constructor options for NewMenderAPI
type MenderAPIOption func(*MenderAPI) error

type MenderAPI struct {
	client *http.Client
	uri    string
}

func NewMenderAPI(uri string, options ...MenderAPIOption) (*MenderAPI, error) {

	if !govalidator.IsURL(uri) {
		return nil, errors.New("invalid server uri")
	}

	api := &MenderAPI{uri: uri}

	// Default http client
	api.client = &http.Client{}

	// Apply all user provided configuration
	for _, option := range options {
		err := option(api)
		if err != nil {
			return nil, err
		}
	}

	return api, nil
}

func WithHTTPClient(client *http.Client) MenderAPIOption {
	return func(api *MenderAPI) error {
		if client != nil {
			api.client = client
		}
		return nil
	}
}
