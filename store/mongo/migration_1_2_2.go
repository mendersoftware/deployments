// Copyright 2019 Northern.tech AS
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
	"strings"

	"github.com/globalsign/mgo"
	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
)

type migration_1_2_2 struct {
	session *mgo.Session
	db      string
}

// Up drops index with len(name) > 127 chars in the 'deployments' collection
func (m *migration_1_2_2) Up(from migrate.Version) error {
	s := m.session.Copy()
	defer s.Close()

	indicesArrays := [][]string{
		StatusIndexes,
		DeviceIDStatusIndexes,
		DeploymentIdIndexes,
	}
	for _, index := range indicesArrays {
		// DropIndex will use the same rules for exploding the index name
		// as EnsureIndexKey previously used to create the 'long' index
		err := s.DB(m.db).
			C(CollectionDevices).
			DropIndex(index...)

		// 'ns not found' simply means the idx doesn't exist
		// DropIndex is just not idempotent, so force it
		if err != nil && err.Error() != "ns not found" && !strings.HasPrefix(err.Error(), "index not found with name") {
			return err
		}
	}

	indicesArrays = [][]string{
		DeploymentStatusFinishedIndex,
		DeploymentStatusPendingIndex,
		DeploymentCreatedIndex,
		DeploymentDeviceStatusRebootingIndex,
		DeploymentDeviceStatusPendingIndex,
		DeploymentDeviceStatusInstallingIndex,
		DeploymentDeviceStatusFinishedIndex,
	}
	for _, index := range indicesArrays {
		// DropIndex will use the same rules for exploding the index name
		// as EnsureIndexKey previously used to create the 'long' index
		err := s.DB(m.db).
			C(CollectionDeployments).
			DropIndex(index...)

		// 'ns not found' simply means the idx doesn't exist
		// DropIndex is just not idempotent, so force it
		if err != nil && err.Error() != "ns not found" && !strings.HasPrefix(err.Error(), "index not found with name") {
			return err
		}
	}

	// create the 'short' index
	storage := NewDataStoreMongoWithSession(m.session)
	return storage.DoEnsureAdditionalIndexing(m.db, m.session)
}

func (m *migration_1_2_2) Version() migrate.Version {
	return migrate.MakeVersion(1, 2, 2)
}
