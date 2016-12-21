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

package model

import (
	"time"

	"github.com/mendersoftware/deployments/resources/deployments"
)

// Storage for Deployment type
type DeploymentsStorage interface {
	Insert(deployment *deployments.Deployment) error
	Delete(id string) error
	FindByID(id string) (*deployments.Deployment, error)
	FindUnfinishedByID(id string) (*deployments.Deployment, error)
	UpdateStats(id string, state_from, state_to string) error
	UpdateStatsAndFinishDeployment(id string, stats deployments.Stats) error
	Find(query deployments.Query) ([]*deployments.Deployment, error)
	Finish(id string, when time.Time) error
}
