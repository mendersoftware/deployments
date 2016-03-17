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
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/mendersoftware/artifacts/images"
	"github.com/pkg/errors"
)

const (
	DefaultUpdateDownloadLinkExpire = 24 * time.Hour
)

var (
	ErrMsgGeneratingImageDownloadLink   = "Generating download link for image."
	ErrMsgSearchingForDeviceDeployments = "Searching for device deployment."
)

type FindOldestDeploymentForDeviceIDWithStatuser interface {
	FindOldestDeploymentForDeviceIDWithStatuses(deviceID string, statuses ...string) (*DeviceDeployment, error)
}

type GetImageLinker interface {
	GetRequest(objectId string, duration time.Duration) (*images.Link, error)
}

type DeviceUpdateModel struct {
	deviceDeployments FindOldestDeploymentForDeviceIDWithStatuser
	getImageLink      GetImageLinker
}

func NewDeviceUpdateModel(deviceDeployments FindOldestDeploymentForDeviceIDWithStatuser, getImageLink GetImageLinker) *DeviceUpdateModel {
	return &DeviceUpdateModel{
		deviceDeployments: deviceDeployments,
		getImageLink:      getImageLink,
	}
}

// TODO: Introduce device-level access control.
// 		 Need to make sure device has access only to it's own updates.
func (d *DeviceUpdateModel) GetObject(deviceID string) (interface{}, error) {

	// Verify ID formatting
	if !govalidator.IsUUIDv4(deviceID) {
		return nil, errors.New(ErrMsgInvalidID)
	}

	deployment, err := d.deviceDeployments.FindOldestDeploymentForDeviceIDWithStatuses(deviceID, ActiveDeploymentStatuses()...)

	if err != nil {
		return nil, errors.Wrap(err, ErrMsgSearchingForDeviceDeployments)
	}

	if deployment == nil {
		return nil, nil
	}

	link, err := d.getImageLink.GetRequest(*deployment.Image.Id, DefaultUpdateDownloadLinkExpire)
	if err != nil {
		return nil, errors.Wrap(err, ErrMsgGeneratingImageDownloadLink)
	}

	type Image struct {
		*images.Link
		*images.SoftwareImage
	}

	// reponse object
	update := &struct {
		ID    *string `json:"id"`
		Image Image   `json:"image"`
	}{
		ID: deployment.Id,
		Image: Image{
			link,
			deployment.Image,
		},
	}

	return update, nil
}

// ActiveDeploymentStatuses lists statuses that represent deployment in active state (not finished).
func ActiveDeploymentStatuses() []string {
	return []string{
		DeviceDeploymentStatusPending,
		DeviceDeploymentStatusInProgress,
	}
}
