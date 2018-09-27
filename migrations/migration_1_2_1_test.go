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
package migrations

import (
	"context"
	"testing"

	"github.com/globalsign/mgo"
	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	"github.com/stretchr/testify/assert"

	d "github.com/mendersoftware/deployments/resources/deployments"
	dm "github.com/mendersoftware/deployments/resources/deployments/mongo"
)

const (
	OldIndexName = "deploymentconstructor.name_text_deploymentconstructor.artifactname_text"
)

func TestMigration_1_2_1(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMigration_1_2_1 in short mode.")
	}

	testCases := map[string]struct {
		// ST or MT naming convention
		db    string
		dbVer string

		// non-empty deployments also means the index is already present
		// must be nil for MT databases - the index name issue implies no deployments were created ever
		deployments []*d.Deployment

		err error
	}{
		"ST, no index, 0.0.0": {
			db:          "deployments_service",
			dbVer:       "",
			deployments: nil,
		},
		"ST, no index, 0.0.1": {
			db:          "deployments_service",
			dbVer:       "0.0.1",
			deployments: nil,
		},
		"ST, with index, 0.0.1": {
			db:    "deployments_service",
			dbVer: "0.0.1",
			deployments: []*d.Deployment{
				makeDeployment(t, "one", "artifact1"),
				makeDeployment(t, "two", "artifact2"),
			},
		},
		"MT, no index, 0.0.0": {
			db:          "deployments_service-59afdb71c704db002a86ad95",
			dbVer:       "",
			deployments: nil,
		},
		"MT, no index, 0.0.1": {
			db:          "deployments_service-59afdb71c704db002a86ad95",
			dbVer:       "0.0.1",
			deployments: nil,
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

		// setup existing deployments
		for _, d := range tc.deployments {
			err := insertDeployment(d, s, tc.db)
			assert.NoError(t, err)
		}

		migrations := []migrate.Migration{
			&migration_1_2_1{
				session: s,
				db:      tc.db,
			},
		}

		m := migrate.SimpleMigrator{
			Session:     s,
			Db:          tc.db,
			Automigrate: true,
		}

		//verify
		// for DBs with data - just in case verify the old/long index was created
		if tc.deployments != nil {
			idxs, err := s.DB(tc.db).C(dm.CollectionDeployments).Indexes()
			assert.NoError(t, err)
			hasOld := hasIndex(OldIndexName, idxs)
			assert.True(t, hasOld)
		}

		err := m.Apply(context.Background(), migrate.Version{1, 2, 1}, migrations)
		assert.NoError(t, err)

		// verify old index dropped
		idxs, err := s.DB(tc.db).C(dm.CollectionDeployments).Indexes()
		assert.NoError(t, err)
		hasOld := hasIndex(OldIndexName, idxs)
		assert.False(t, hasOld)

		// verify new index present - only if deployment inserted
		if tc.deployments != nil {
			hasNew := hasIndex(dm.IndexDeploymentArtifactNameStr, idxs)
			assert.NoError(t, err)
			assert.True(t, hasNew)
		}

		s.Close()
	}
}

// makeDeployments creates a bare-bones deployment struct
func makeDeployment(t *testing.T, name, artifactName string) *d.Deployment {
	d, err := d.NewDeploymentFromConstructor(
		&d.DeploymentConstructor{
			Name:         sptr(name),
			ArtifactName: sptr(artifactName)})

	assert.NoError(t, err)
	return d
}

// insertDeployment mimics the 0.0.1 method of inserting deployments (now deleted)
// creates the deployment name + artifact name index with a 'long' name
func insertDeployment(d *d.Deployment, session *mgo.Session, db string) error {
	err := session.DB(db).
		C(dm.CollectionDeployments).
		EnsureIndexKey(dm.StorageIndexes...)
	if err != nil {
		return err
	}

	return session.DB(db).C(dm.CollectionDeployments).Insert(d)
}

// find indexes by name - new and old indexes have the same key so it's not useful
// Name is what causes the issue with long index name, so use that
func hasIndex(name string, idxs []mgo.Index) bool {
	for _, i := range idxs {
		if i.Name == name {
			return true
		}
	}

	return false
}

func sptr(s string) *string {
	return &s
}
