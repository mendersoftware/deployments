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

package controller

import (
	"encoding/json"

	"github.com/asaskevich/govalidator"
	"github.com/pkg/errors"

	"github.com/mendersoftware/deployments/model"
)

var (
	ErrBadStatus = errors.New("unknown status value")
)

type statusReport struct {
	Status   string
	SubState *string `json:"substate" valid:"length(0|200)"`
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
		model.DeviceDeploymentStatusDownloading,
		model.DeviceDeploymentStatusInstalling,
		model.DeviceDeploymentStatusRebooting,
		model.DeviceDeploymentStatusSuccess,
		model.DeviceDeploymentStatusFailure,
		model.DeviceDeploymentStatusAlreadyInst,
	}

	if !containsString(temp.Status, valid) {
		return ErrBadStatus
	}

	if ok, err := govalidator.ValidateStruct(temp); !ok {
		return err
	}

	// all good
	s.Status = temp.Status
	s.SubState = temp.SubState

	return nil
}
