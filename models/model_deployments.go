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
	"fmt"

	"github.com/mendersoftware/artifacts/models/images"
	"github.com/mendersoftware/artifacts/models/users"
	"gopkg.in/mgo.v2"
)

type FindImageByApplicationAndModeler interface {
	FindImageByApplicationAndModel(user users.UserI, version, model string) (*images.ImageMeta, error)
}

type DeploymentsModel struct {
	session     *mgo.Session
	imageFinder FindImageByApplicationAndModeler
}

func NewDeploymentModel(session *mgo.Session, imageFinder FindImageByApplicationAndModeler) *DeploymentsModel {
	return &DeploymentsModel{
		session:     session,
		imageFinder: imageFinder,
	}
}

func (d *DeploymentsModel) NewObject() interface{} {
	return NewDeploymentConstructor()
}

func (d *DeploymentsModel) Validate(deployment interface{}) error {
	return deployment.(*DeploymentConstructor).Validate()
}

// TODO: Check if there are any matching images, if not there is no point to be able to do deployment (DO: then images get presistence)
// TODO: Assign image to the device, depending on it's model (based on uploaded images)

// TODO: Check with inventory if the device ID exists and is bootstrapped.
// TODO: Check device model in the inventory
func (d *DeploymentsModel) Create(obj interface{}) (string, error) {
	constructorData := obj.(*DeploymentConstructor)

	deployment := NewDeployment()
	deployment.Name = constructorData.Name
	deployment.Version = constructorData.Version

	// Generate deployment for each specified device.
	deviceDeployments := make([]interface{}, 0, len(constructorData.Devices))
	for _, id := range constructorData.Devices {

		model, err := d.CheckModel(id)
		if err != nil {
			return "", err
		}

		image, err := d.AssignImage(*deployment.Version, model)
		if err != nil {
			return "", err
		}

		dd := NewDeviceDeployment(id, *deployment.Id)
		dd.Model = &model
		dd.Image = image
		dd.Created = deployment.Created

		if dd.Image == nil {
			fmt.Println("Image NULL")
			status := DeviceDeploymentStatusNoImage
			dd.Status = &status
		}

		deviceDeployments = append(deviceDeployments, dd)
	}

	// New database session for handling connection
	session := d.session.Copy()
	defer session.Close()

	// Store deployment
	if err := session.DB(DatabaseName).C(DeploymentsCollection).Insert(deployment); err != nil {
		return "", err
	}

	// Store devices
	if err := session.DB(DatabaseName).C(DevicesCollection).Insert(deviceDeployments...); err != nil {
		// Ignore output and continue (remove as much as we can)
		session.DB(DatabaseName).C(DeploymentsCollection).RemoveId(*deployment.Id)
		for _, deployment := range deviceDeployments {
			// Ignore output and continue (remove as much as we can)
			session.DB(DatabaseName).C(DeploymentsCollection).RemoveId(*deployment.(*DeviceDeployment).Id)
		}
		return "", err
	}

	return *deployment.Id, nil
}

// TODO: This should be provided as a part of inventory service driver (dependency)
// TODO: Model is hardcoded
func (d *DeploymentsModel) CheckModel(deviceId string) (string, error) {
	return "BB-8", nil
}

// TODO: Mess with the old vs new image types, need to migrate to SoftwareImage
// TODO: User management
func (d *DeploymentsModel) AssignImage(version, model string) (*SoftwareImage, error) {
	image, err := d.imageFinder.FindImageByApplicationAndModel(users.NewDummyUser(), version, model)
	if err != nil {
		return nil, err
	}

	if image == nil {
		return nil, nil
	}

	softwareImage := NewSoftwareImage(image.Id, image.YoctoId)
	if image.Checksum != "" {
		softwareImage.Checksum = &image.Checksum
	}

	return softwareImage, nil
}

func (d *DeploymentsModel) FindOne(id string) (interface{}, error) {
	if id == "" {
		return nil, ErrInvalidId
	}

	session := d.session.Copy()
	defer session.Close()

	deployment := Deployment{}
	if err := session.DB(DatabaseName).C(DeploymentsCollection).FindId(id).One(&deployment); err != nil {
		return nil, err
	}

	// TODO: CHECK STATUS (check from highest to lowest if has at least one entry)

	return deployment, nil
}
