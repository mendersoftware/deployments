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
	"context"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/mendersoftware/go-lib-micro/store"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/mendersoftware/deployments/resources/deployments"
	imagesMongo "github.com/mendersoftware/deployments/resources/images/mongo"
)

// Database settings
const (
	CollectionDevices = "devices"
)

// Database keys
const (
	StorageKeyDeviceDeploymentAssignedImage   = "image"
	StorageKeyDeviceDeploymentAssignedImageId = StorageKeyDeviceDeploymentAssignedImage + "." + imagesMongo.StorageKeySoftwareImageId
	StorageKeyDeviceDeploymentDeviceId        = "deviceid"
	StorageKeyDeviceDeploymentStatus          = "status"
	StorageKeyDeviceDeploymentDeploymentID    = "deploymentid"
	StorageKeyDeviceDeploymentFinished        = "finished"
	StorageKeyDeviceDeploymentIsLogAvailable  = "log"
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
func (d *DeviceDeploymentsStorage) InsertMany(ctx context.Context,
	deployments ...*deployments.DeviceDeployment) error {

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

	if err := d.session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Insert(list...); err != nil {
		return err
	}

	return nil
}

// ExistAssignedImageWithIDAndStatuses checks if image is used by deplyment with specified status.
func (d *DeviceDeploymentsStorage) ExistAssignedImageWithIDAndStatuses(ctx context.Context,
	imageID string, statuses ...string) (bool, error) {

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

	session := d.session.Copy()
	defer session.Close()

	// if found at least one then image in active deployment
	var tmp interface{}
	if err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Find(query).One(&tmp); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// FindOldestDeploymentForDeviceIDWithStatuses find oldest deployment matching device id and one of specified statuses.
func (d *DeviceDeploymentsStorage) FindOldestDeploymentForDeviceIDWithStatuses(ctx context.Context,
	deviceID string, statuses ...string) (*deployments.DeviceDeployment, error) {

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
	var deployment *deployments.DeviceDeployment
	if err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Find(query).Sort("created").One(&deployment); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return deployment, nil
}

// FindAllDeploymentsForDeviceIDWithStatuses finds all deployments matching device id and one of specified statuses.
func (d *DeviceDeploymentsStorage) FindAllDeploymentsForDeviceIDWithStatuses(ctx context.Context,
	deviceID string, statuses ...string) ([]deployments.DeviceDeployment, error) {

	// Verify ID formatting
	if govalidator.IsNull(deviceID) {
		return nil, ErrStorageInvalidID
	}

	session := d.session.Copy()
	defer session.Close()

	// Device should know only about deployments that are not finished
	query := bson.M{
		StorageKeyDeviceDeploymentDeviceId: deviceID,
		StorageKeyDeviceDeploymentStatus: bson.M{
			"$in": statuses,
		},
	}

	var deployments []deployments.DeviceDeployment
	if err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Find(query).All(&deployments); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return deployments, nil
}

func (d *DeviceDeploymentsStorage) UpdateDeviceDeploymentStatus(ctx context.Context,
	deviceID string, deploymentID string, status string, finishTime *time.Time) (string, error) {

	// Verify ID formatting
	if govalidator.IsNull(deviceID) ||
		govalidator.IsNull(deploymentID) ||
		govalidator.IsNull(status) {
		return "", ErrStorageInvalidID
	}

	session := d.session.Copy()
	defer session.Close()

	// Device should know only about deployments that are not finished
	query := bson.M{
		StorageKeyDeviceDeploymentDeviceId:     deviceID,
		StorageKeyDeviceDeploymentDeploymentID: deploymentID,
	}

	// update status field
	set := bson.M{
		StorageKeyDeviceDeploymentStatus: status,
	}
	// and finish time if provided
	if finishTime != nil {
		set[StorageKeyDeviceDeploymentFinished] = finishTime
	}

	update := bson.M{
		"$set": set,
	}

	var old deployments.DeviceDeployment

	// update and return the old status in one go
	change := mgo.Change{
		Update: update,
	}

	chi, err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Find(query).Apply(change, &old)

	if err != nil {
		return "", err
	}

	if chi.Updated == 0 {
		return "", mgo.ErrNotFound
	}

	return *old.Status, nil
}

func (d *DeviceDeploymentsStorage) UpdateDeviceDeploymentLogAvailability(ctx context.Context,
	deviceID string, deploymentID string, log bool) error {

	// Verify ID formatting
	if govalidator.IsNull(deviceID) ||
		govalidator.IsNull(deploymentID) {
		return ErrStorageInvalidID
	}

	session := d.session.Copy()
	defer session.Close()

	selector := bson.M{
		StorageKeyDeviceDeploymentDeviceId:     deviceID,
		StorageKeyDeviceDeploymentDeploymentID: deploymentID,
	}

	update := bson.M{
		"$set": bson.M{
			StorageKeyDeviceDeploymentIsLogAvailable: log,
		},
	}

	if err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Update(selector, update); err != nil {
		return err
	}

	return nil
}

func (d *DeviceDeploymentsStorage) AggregateDeviceDeploymentByStatus(ctx context.Context,
	id string) (deployments.Stats, error) {

	if govalidator.IsNull(id) {
		return nil, ErrStorageInvalidID
	}

	session := d.session.Copy()
	defer session.Close()

	match := bson.M{
		"$match": bson.M{
			StorageKeyDeviceDeploymentDeploymentID: id,
		},
	}
	group := bson.M{
		"$group": bson.M{
			"_id": "$" + StorageKeyDeviceDeploymentStatus,
			"count": bson.M{
				"$sum": 1,
			},
		},
	}
	pipe := []bson.M{
		match,
		group,
	}
	var results []struct {
		Name  string `bson:"_id"`
		Count int
	}
	err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Pipe(&pipe).All(&results)
	if err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	raw := deployments.NewDeviceDeploymentStats()
	for _, res := range results {
		raw[res.Name] = res.Count
	}
	return raw, nil
}

//GetDeviceStatusesForDeployment retrieve device deployment statuses for a given deployment.
func (d *DeviceDeploymentsStorage) GetDeviceStatusesForDeployment(ctx context.Context,
	deploymentID string) ([]deployments.DeviceDeployment, error) {

	session := d.session.Copy()
	defer session.Close()

	query := bson.M{
		StorageKeyDeviceDeploymentDeploymentID: deploymentID,
	}

	var statuses []deployments.DeviceDeployment

	err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Find(query).All(&statuses)
	if err != nil {
		return nil, err
	}

	return statuses, nil
}

// Returns true if deployment of ID `deploymentID` is assigned to device with ID
// `deviceID`, false otherwise. In case of errors returns false and an error
// that occurred
func (d *DeviceDeploymentsStorage) HasDeploymentForDevice(ctx context.Context,
	deploymentID string, deviceID string) (bool, error) {

	session := d.session.Copy()
	defer session.Close()

	query := bson.M{
		StorageKeyDeviceDeploymentDeploymentID: deploymentID,
		StorageKeyDeviceDeploymentDeviceId:     deviceID,
	}

	var dep deployments.DeviceDeployment
	err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Find(query).One(&dep)
	if err != nil {
		if err == mgo.ErrNotFound {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}

func (d *DeviceDeploymentsStorage) GetDeviceDeploymentStatus(ctx context.Context,
	deploymentID string, deviceID string) (string, error) {

	session := d.session.Copy()
	defer session.Close()

	query := bson.M{
		StorageKeyDeviceDeploymentDeploymentID: deploymentID,
		StorageKeyDeviceDeploymentDeviceId:     deviceID,
	}

	var dep deployments.DeviceDeployment
	err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Find(query).One(&dep)
	if err != nil {
		if err == mgo.ErrNotFound {
			return "", nil
		} else {
			return "", err
		}
	}

	return *dep.Status, nil
}

func (d *DeviceDeploymentsStorage) AbortDeviceDeployments(ctx context.Context,
	deploymentId string) error {

	if govalidator.IsNull(deploymentId) {
		return ErrStorageInvalidID
	}

	session := d.session.Copy()
	defer session.Close()
	selector := bson.M{
		"$and": []bson.M{
			{
				StorageKeyDeviceDeploymentDeploymentID: deploymentId,
			},
			{
				StorageKeyDeviceDeploymentStatus: bson.M{
					"$in": deployments.ActiveDeploymentStatuses(),
				},
			},
		},
	}

	update := bson.M{
		"$set": bson.M{
			StorageKeyDeviceDeploymentStatus: deployments.DeviceDeploymentStatusAborted,
		},
	}

	_, err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).UpdateAll(selector, update)

	if err == mgo.ErrNotFound {
		return ErrStorageInvalidID
	}

	return err
}

func (d *DeviceDeploymentsStorage) DecommissionDeviceDeployments(ctx context.Context,
	deviceId string) error {

	if govalidator.IsNull(deviceId) {
		return ErrStorageInvalidID
	}

	session := d.session.Copy()
	defer session.Close()
	selector := bson.M{
		"$and": []bson.M{
			{
				StorageKeyDeviceDeploymentDeviceId: deviceId,
			},
			{
				StorageKeyDeviceDeploymentStatus: bson.M{
					"$in": deployments.ActiveDeploymentStatuses(),
				},
			},
		},
	}

	update := bson.M{
		"$set": bson.M{
			StorageKeyDeviceDeploymentStatus: deployments.DeviceDeploymentStatusDecommissioned,
		},
	}

	_, err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).UpdateAll(selector, update)

	return err
}
