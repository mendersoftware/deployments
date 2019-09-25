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
	"context"
	"testing"

	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	"github.com/stretchr/testify/assert"
)

func TestMigration_1_2_2(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMigration_1_2_2 in short mode.")
	}

	testCases := map[string]struct {
		// ST or MT naming convention
		db    string
		dbVer string

		err error
	}{
		"ST, no index, 0.0.0": {
			db:    "deployments_service",
			dbVer: "",
		},
		"MT, no index, 0.0.0": {
			db:    "deployments_service-59afdb71c704db002a86ad95",
			dbVer: "",
		},
	}

	for name, tc := range testCases {
		t.Logf("test case: %s", name)

		db.Wipe()
		s := db.Session()

		// setup
		// setup existing migrations
		if tc.dbVer != "" {
			ver, err := migrate.NewVersion(tc.dbVer)
			assert.NoError(t, err)
			migrate.UpdateMigrationInfo(*ver, s, tc.db)
		}

		migrations := []migrate.Migration{
			&migration_1_2_1{
				session: s,
				db:      tc.db,
			},
			&migration_1_2_2{
				session: s,
				db:      tc.db,
			},
		}

		m := migrate.SimpleMigrator{
			Session:     s,
			Db:          tc.db,
			Automigrate: true,
		}

		err := m.Apply(context.Background(), migrate.MakeVersion(1, 2, 2), migrations)
		assert.NoError(t, err)

		devicesCollectionIndicesNames := []string{
			IndexDeploymentDeviceStatusesStr,
			IndexDeploymentDeviceIdStatusStr,
			IndexDeploymentDeviceDeploymentIdStr,
		}
		// verify new indices present
		idxs, err := s.DB(tc.db).C(CollectionDevices).Indexes()
		assert.NoError(t, err)

		for _, indexName := range devicesCollectionIndicesNames {
			hasNew := hasIndex(indexName, idxs)
			assert.True(t, hasNew)
		}

		deploymentsCollectionIndicesNames := []string{
			IndexDeploymentStatusFinishedStr,
			IndexDeploymentStatusPendingStr,
			IndexDeploymentCreatedStr,
			IndexDeploymentDeviceStatusRebootingStr,
			IndexDeploymentDeviceStatusPendingStr,
			IndexDeploymentDeviceStatusInstallingStr,
			IndexDeploymentDeviceStatusFinishedStr,
		}
		// verify new indices present
		idxs, err = s.DB(tc.db).C(CollectionDeployments).Indexes()
		assert.NoError(t, err)
		for _, indexName := range deploymentsCollectionIndicesNames {
			hasNew := hasIndex(indexName, idxs)
			assert.True(t, hasNew)
		}

		s.Close()
	}
}
