// Copyright 2022 Northern.tech AS
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

package model

import "time"

type DeviceDeploymentListItem struct {
	Id         string                     `json:"id"`
	Deployment *Deployment                `json:"deployment"`
	Device     *DeviceDeploymentWithImage `json:"device"`
}

type DeviceDeploymentWithImage struct {
	Active         bool                   `json:"-"`
	Created        *time.Time             `json:"created"`
	Finished       *time.Time             `json:"finished,omitempty"`
	Deleted        *time.Time             `json:"deleted,omitempty"`
	Status         DeviceDeploymentStatus `json:"status"`
	DeviceId       string                 `json:"id"`
	DeploymentId   string                 `json:"-"`
	Id             string                 `json:"-"`
	Image          *Image                 `json:"image,omitempty"`
	Request        *DeploymentNextRequest `json:"-"`
	IsLogAvailable bool                   `json:"log"`
	SubState       string                 `json:"substate,omitempty"`
}
