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
	"github.com/mendersoftware/artifacts/images"
	"github.com/pkg/errors"
)

type ImageByNameAndDeviceTyper interface {
	ImageByNameAndDeviceType(name, deviceType string) (*images.SoftwareImage, error)
}

type GetDeviceTyper interface {
	GetDeviceType(deviceID string) (string, error)
}

type ImageBasedDeviceDeployment struct {
	images  ImageByNameAndDeviceTyper
	devices GetDeviceTyper
}

func NewImageBasedDeviceDeployment(images ImageByNameAndDeviceTyper, devices GetDeviceTyper) *ImageBasedDeviceDeployment {
	return &ImageBasedDeviceDeployment{
		images:  images,
		devices: devices,
	}
}

func (d *ImageBasedDeviceDeployment) Generate(deviceID string, deployment *Deployment) (*DeviceDeployment, error) {

	if err := deployment.Validate(); err != nil {
		return nil, errors.Wrap(err, "Validating deployment")
	}

	deviceType, err := d.devices.GetDeviceType(deviceID)
	if err != nil {
		return nil, errors.Wrap(err, "Checking device type")
	}

	image, err := d.images.ImageByNameAndDeviceType(*deployment.ArtifactName, deviceType)
	if err != nil {
		return nil, errors.Wrap(err, "Assigning image targeted for device type")
	}

	deviceDeployment := NewDeviceDeployment(deviceID, *deployment.Id)
	deviceDeployment.DeviceType = &deviceType
	deviceDeployment.Image = image
	deviceDeployment.Created = deployment.Created

	// If not having appropriate image, set noimage status
	if deviceDeployment.Image == nil {
		status := DeviceDeploymentStatusNoImage
		deviceDeployment.Status = &status
	}

	return deviceDeployment, nil
}

type InventoryWithHardcodedType struct {
	deviceType string
}

func NewInventoryWithHardcodedType(deviceType string) *InventoryWithHardcodedType {
	return &InventoryWithHardcodedType{
		deviceType: deviceType,
	}
}

func (i *InventoryWithHardcodedType) GetDeviceType(deviceID string) (string, error) {
	return i.deviceType, nil
}
