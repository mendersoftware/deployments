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
)

// Database settings
const (
	DatabaseName          = "deployment_service"
	CollectionDeployments = "deployments"
)

// Errors
var (
	ErrDeploymentStorageInvalidDeployment = errors.New("Invalid deployment")
	ErrStorageInvalidID                   = errors.New("Invalid id")
)

// DeploymentsStorage is a data layer for deployments based on MongoDB
type DeploymentsStorage struct {
	session *mgo.Session
}

// NewDeploymentsStorage new data layer object
func NewDeploymentsStorage(session *mgo.Session) *DeploymentsStorage {
	return &DeploymentsStorage{
		session: session,
	}
}

// Insert persists object
func (d *DeploymentsStorage) Insert(deployment *deployments.Deployment) error {

	if deployment == nil {
		return ErrDeploymentStorageInvalidDeployment
	}

	if err := deployment.Validate(); err != nil {
		return err
	}

	session := d.session.Copy()
	defer session.Close()

	return session.DB(DatabaseName).C(CollectionDeployments).Insert(deployment)
}

// Delete removed entry by ID
// Noop on ID not found
func (d *DeploymentsStorage) Delete(id string) error {

	if govalidator.IsNull(id) {
		return ErrStorageInvalidID
	}

	session := d.session.Copy()
	defer session.Close()

	if err := session.DB(DatabaseName).C(CollectionDeployments).RemoveId(id); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil
		}
		return err
	}

	return nil
}

func (d *DeploymentsStorage) FindByID(id string) (*deployments.Deployment, error) {

	if govalidator.IsNull(id) {
		return nil, ErrStorageInvalidID
	}

	session := d.session.Copy()
	defer session.Close()

	var deployment *deployments.Deployment
	if err := session.DB(DatabaseName).C(CollectionDeployments).
		FindId(id).One(&deployment); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return deployment, nil
}
