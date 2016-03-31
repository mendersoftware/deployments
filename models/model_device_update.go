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
package models

import (
	"time"

	"github.com/mendersoftware/artifacts/models/fileservice"
	"github.com/mendersoftware/artifacts/models/users"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type DeviceUpdateModel struct {
	session     *mgo.Session
	fileStorage fileservice.FileServiceModelI
}

func NewDeviceUpdateModel(session *mgo.Session, fileStorage fileservice.FileServiceModelI) *DeviceUpdateModel {
	return &DeviceUpdateModel{
		session:     session,
		fileStorage: fileStorage,
	}
}

// TODO: Not found -> No Content
// TODO: Authenticate device to request to access only it's updates
// TODO: Hardcoded 'admin' customer id , add support for user object
// TODO: Fake successful update, set status to success when device requests next update due to lack of mechanism for devcie to update status by itself.
func (d *DeviceUpdateModel) FindOne(id string) (interface{}, error) {
	if id == "" {
		return nil, ErrInvalidId
	}

	session := d.session.Copy()
	defer session.Close()

	// Device should know only on deployments in progress and pending.
	query := bson.M{
		"deviceid": id,
		"status": bson.M{
			"$in": []string{
				DeviceDeploymentStatusPending,
				DeviceDeploymentStatusInProgress,
			},
		},
	}

	// Select only the oldest one that have not been finished yet.
	var deployment DeviceDeployment
	err := session.DB(DatabaseName).C(DevicesCollection).Find(query).Sort("created").One(&deployment)
	if err != nil {
		// No updates found
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	link, err := d.fileStorage.GetRequest(users.NewDummyUser().GetCustomerID(), *deployment.Image.Id, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	type Image struct {
		*fileservice.Link
		*SoftwareImage
	}

	update := &struct {
		Id    *string `json:"id"`
		Image Image   `json:"image"`
	}{
		Id: deployment.Id,
		Image: Image{
			link,
			deployment.Image,
		},
	}

	// HACK for Demo/Testing
	if err := session.DB(DatabaseName).C(DevicesCollection).UpdateId(deployment.Id, bson.M{"$set": bson.M{"status": DeviceDeploymentStatusSuccess}}); err != nil {
		return nil, err
	}

	return update, nil
}
