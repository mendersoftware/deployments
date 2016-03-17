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

package deployments

import (
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	ErrMsgInvalidDeviceDeployment   = "Invalid device deployment"
	ErrMsgInvalidDeviceDeploymentID = "Invalid device deploymnt ID"
)

// DeviceDeploymentsStorage is a data layer for deployments based on MongoDB
type DeviceDeploymentsStorage struct {
	session *mgo.Session
}

// NewDeviceDeploymentsStorage new data layer object
func NewDeviceDeploymentsStorage(session *mgo.Session) *DeviceDeploymentsStorage {
	return &DeviceDeploymentsStorage{
		session: session,
	}
}

// InsertMany stores multiple device deployment objects.
// TODO: Handle error cleanup, multi insert is not atomic
func (d *DeviceDeploymentsStorage) InsertMany(deployments ...*DeviceDeployment) error {

	if len(deployments) == 0 {
		return nil
	}

	// Writing to another interface list addresses golang gatcha interface{} == []interface{}
	var list []interface{}
	for _, deployment := range deployments {
		if deployment == nil {
			return errors.New(ErrMsgInvalidDeviceDeployment)
		}

		if deployment.Id == nil || len(strings.TrimSpace(*deployment.Id)) == 0 {
			return errors.New(ErrMsgInvalidDeviceDeploymentID)
		}

		list = append(list, deployment)
	}

	if err := d.session.DB(DatabaseName).C(CollectionDeployments).Insert(list...); err != nil {
		return errors.Wrap(err, ErrMsgDatabaseError)
	}

	return nil
}

// ExistAssignedImageWithIDAndStatuses checks if image is used by deplyment with specified status.
func (d *DeviceDeploymentsStorage) ExistAssignedImageWithIDAndStatuses(imageID string, statuses ...string) (bool, error) {

	// Verify ID formatting
	if !govalidator.IsUUIDv4(imageID) {
		return false, errors.New(ErrMsgInvalidID)
	}

	query := bson.M{StorageKeyDeviceDeploymentAssignedImageId: imageID}

	if len(statuses) > 0 {
		query[StorageKeyDeviceDeploymentStatus] = bson.M{
			"$in": statuses,
		}
	}

	session := d.session.Copy()
	defer session.Close()

	// if found at least one then image in active deployment
	var tmp interface{}
	err := session.DB(DatabaseName).C(CollectionDevices).Find(query).Explain(&tmp)
	if err != nil && err.Error() == mgo.ErrNotFound.Error() {
		return false, nil
	}

	if err != nil {
		return false, errors.Wrap(err, ErrMsgDatabaseError)
	}

	return true, nil
}

// FindOldestDeploymentForDeviceIDWithStatuses find oldest deplyoment matching device id and one of specified statuses.
func (d *DeviceDeploymentsStorage) FindOldestDeploymentForDeviceIDWithStatuses(deviceID string, statuses ...string) (*DeviceDeployment, error) {

	// Verify ID formatting
	if !govalidator.IsUUIDv4(deviceID) {
		return nil, errors.New(ErrMsgInvalidID)
	}

	// 	session := d.session.Copy()
	// defer session.Close()

	// // Device should know only on deployments in progress and pending.
	// query := bson.M{
	// 	StorageKeyDeviceDeploymentDeviceId: id,
	// 	StorageKeyDeviceDeploymentStatus: bson.M{
	// 		"$in": []string{
	// 			DeviceDeploymentStatusPending,
	// 			DeviceDeploymentStatusInProgress,
	// 		},
	// 	},
	// }

	// // Select only the oldest one that have not been finished yet.
	// var deployment DeviceDeployment
	// err := session.DB(DatabaseName).C(CollectionDevices).Find(query).Sort("created").One(&deployment)
	// if err != nil {
	// 	// No updates found
	// 	if err.Error() == mgo.ErrNotFound.Error() {
	// 		return nil, nil
	// 	}

	// 	log.Error(err)
	// 	return nil, ErrWhileSearchingForDeviceDeployments
	// }

	return nil, nil
}
