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

	"github.com/pkg/errors"

	"gopkg.in/mgo.v2"
)

const (
	// Errors
	ErrMsgDatabaseError = "Database error"
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
func (d *DeploymentsStorage) Insert(deployment *Deployment) error {

	if deployment == nil {
		return errors.New(ErrMsgInvalidDeployment)
	}

	if deployment.Id == nil || len(strings.TrimSpace(*deployment.Id)) == 0 {
		return errors.New(ErrMsgInvalidDeploymentID)
	}

	session := d.session.Copy()
	defer session.Close()

	if err := session.DB(DatabaseName).C(CollectionDeployments).Insert(deployment); err != nil {
		return errors.Wrap(err, ErrMsgDatabaseError)
	}

	return nil
}

// Delete removed entry by ID
// Noop on ID not found
func (d *DeploymentsStorage) Delete(id string) error {

	if len(strings.TrimSpace(id)) == 0 {
		return errors.New(ErrMsgInvalidDeploymentID)
	}

	session := d.session.Copy()
	defer session.Close()

	err := session.DB(DatabaseName).C(CollectionDeployments).RemoveId(id)

	if err != nil && err.Error() == mgo.ErrNotFound.Error() {
		return nil
	}

	if err != nil {
		return errors.Wrap(err, ErrMsgDatabaseError)
	}

	return nil
}

func (d *DeploymentsStorage) FindByID(id string) (*Deployment, error) {

	if len(strings.TrimSpace(id)) == 0 {
		return nil, errors.New(ErrMsgInvalidID)
	}

	session := d.session.Copy()
	defer session.Close()

	var deployment *Deployment
	err := session.DB(DatabaseName).C(CollectionDeployments).FindId(id).One(&deployment)
	if err != nil && err.Error() == mgo.ErrNotFound.Error() {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, ErrMsgDatabaseError)
	}

	return deployment, nil
}
