// Copyright 2018 Northern.tech AS
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

	"github.com/globalsign/mgo"
	"github.com/mendersoftware/go-lib-micro/store"

	"github.com/mendersoftware/deployments/resources/limits"
	"github.com/mendersoftware/deployments/resources/limits/model"
)

// Database
const (
	DatabaseName     = "deployment_service"
	CollectionLimits = "limits"
)

// SoftwareImagesStorage is a data layer for SoftwareImages based on MongoDB
// Implements model.SoftwareImagesStorage
type LimitsStorage struct {
	session *mgo.Session
}

// NewSoftwareImagesStorage new data layer object
func NewLimitsStorage(session *mgo.Session) *LimitsStorage {
	return &LimitsStorage{
		session: session,
	}
}

// ImageByIdsAndDeviceType finds image with id from ids and targed device type
func (ls *LimitsStorage) GetLimit(ctx context.Context, name string) (*limits.Limit, error) {

	session := ls.session.Copy()
	defer session.Close()

	var limit limits.Limit
	if err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionLimits).FindId(name).One(&limit); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, model.ErrLimitNotFound
		}
		return nil, err
	}

	return &limit, nil
}
