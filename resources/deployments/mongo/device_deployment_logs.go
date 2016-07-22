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

package mongo

import (
	"github.com/asaskevich/govalidator"
	"github.com/mendersoftware/deployments/resources/deployments"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Database settings
const (
	// TODO: do we have any naming convention for mongo collections?
	CollectionDeviceDeploymentLogs = "devices.logs"
)

// Database keys
const (
	StorageKeyDeviceDeploymentLogMessages = "messages"
)

// Errors
var (
	ErrStorageInvalidLog = errors.New("Invalid deployment log")
)

// DeviceDeploymentLogsStorage is a data layer for deployment logs based on MongoDB
type DeviceDeploymentLogsStorage struct {
	session *mgo.Session
}

func NewDeviceDeploymentLogsStorage(session *mgo.Session) *DeviceDeploymentLogsStorage {
	return &DeviceDeploymentLogsStorage{
		session: session,
	}
}

func (d *DeviceDeploymentLogsStorage) SaveDeviceDeploymentLog(
	deviceID string, deploymentID string, log *deployments.DeploymentLog) error {

	if govalidator.IsNull(deviceID) ||
		govalidator.IsNull(deploymentID) {
		return ErrStorageInvalidID
	}
	if log == nil {
		return ErrStorageInvalidLog
	}

	session := d.session.Copy()
	defer session.Close()

	query := bson.M{
		StorageKeyDeviceDeploymentDeviceId:     deviceID,
		StorageKeyDeviceDeploymentDeploymentID: deploymentID,
	}

	// update log messages
	// if the deployment log is already present than messages will be overwritten
	update := bson.M{
		"$set": bson.M{
			StorageKeyDeviceDeploymentLogMessages: log.Messages,
		},
	}
	if _, err := session.DB(DatabaseName).C(CollectionDeviceDeploymentLogs).Upsert(query, update); err != nil {
		return err
	}

	return nil
}
