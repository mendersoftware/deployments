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

package controller

import (
	"encoding/json"

	"github.com/mendersoftware/deployments/resources/deployments"
	"github.com/pkg/errors"
)

var (
	ErrBadStatus = errors.New("unknown status value")
)

type statusReport struct {
	Status string
}

func containsString(what string, in []string) bool {
	found := false
	for _, v := range in {
		if what == v {
			found = true
			break
		}
	}
	return found
}

func (s *statusReport) UnmarshalJSON(raw []byte) error {
	type auxStatusReport statusReport
	var temp auxStatusReport

	err := json.Unmarshal(raw, &temp)
	if err != nil {
		return err
	}

	valid := []string{
		deployments.DeviceDeploymentStatusDownloading,
		deployments.DeviceDeploymentStatusInstalling,
		deployments.DeviceDeploymentStatusRebooting,
		deployments.DeviceDeploymentStatusSuccess,
		deployments.DeviceDeploymentStatusFailure,
		deployments.DeviceDeploymentStatusAlreadyInst,
	}

	if !containsString(temp.Status, valid) {
		return ErrBadStatus
	}

	// all good
	s.Status = temp.Status

	return nil
}
