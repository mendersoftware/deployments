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
	"fmt"

	"github.com/asaskevich/govalidator"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Database settings
const (
	CollectionDevices = "devices"
)

// Errors
var (
	ErrStorageInvalidDeviceDeployment = errors.New("Invalid device deployment")
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
// TODO: Handle error cleanup, multi insert is not atomic, loop into two-phase commits
func (d *DeviceDeploymentsStorage) InsertMany(deployments ...*DeviceDeployment) error {

	if len(deployments) == 0 {
		return nil
	}

	// Writing to another interface list addresses golang gatcha interface{} == []interface{}
	var list []interface{}
	for _, deployment := range deployments {

		if deployment == nil {
			return ErrStorageInvalidDeviceDeployment
		}

		if err := deployment.Validate(); err != nil {
			return errors.Wrap(err, "Validating device deployment")
		}

		list = append(list, deployment)
	}

	if err := d.session.DB(DatabaseName).C(CollectionDeployments).Insert(list...); err != nil {
		return err
	}

	return nil
}

// ExistAssignedImageWithIDAndStatuses checks if image is used by deplyment with specified status.
func (d *DeviceDeploymentsStorage) ExistAssignedImageWithIDAndStatuses(imageID string, statuses ...string) (bool, error) {

	// Verify ID formatting
	if govalidator.IsNull(imageID) {
		return false, ErrStorageInvalidID
	}

	query := bson.M{StorageKeyDeviceDeploymentAssignedImageId: imageID}

	if len(statuses) > 0 {
		query[StorageKeyDeviceDeploymentStatus] = bson.M{
			"$in": statuses,
		}
	}

	fmt.Println(query)

	session := d.session.Copy()
	defer session.Close()

	// if found at least one then image in active deployment
	var tmp interface{}
	err := session.DB(DatabaseName).C(CollectionDevices).Find(query).One(&tmp)
	fmt.Println(err)
	fmt.Println(tmp)
	if err != nil && err.Error() == mgo.ErrNotFound.Error() {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

// FindOldestDeploymentForDeviceIDWithStatuses find oldest deplyoment matching device id and one of specified statuses.
func (d *DeviceDeploymentsStorage) FindOldestDeploymentForDeviceIDWithStatuses(deviceID string, statuses ...string) (*DeviceDeployment, error) {

	// Verify ID formatting
	if govalidator.IsNull(deviceID) {
		return nil, ErrStorageInvalidID
	}

	session := d.session.Copy()
	defer session.Close()

	// Device should know only about deployments that are not finished
	query := bson.M{
		StorageKeyDeviceDeploymentDeviceId: deviceID,
		StorageKeyDeviceDeploymentStatus:   bson.M{"$in": statuses},
	}

	// Select only the oldest one that have not been finished yet.
	var deployment *DeviceDeployment
	if err := session.DB(DatabaseName).C(CollectionDevices).Find(query).Sort("created").One(deployment); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}

		return nil, err
	}

	return deployment, nil
}
