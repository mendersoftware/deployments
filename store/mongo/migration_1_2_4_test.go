// Copyright 2020 Northern.tech AS
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
	"time"

	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/mendersoftware/deployments/model"
)

func TestMigration_1_2_4(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMigration_1_2_4 in short mode.")
	}
	ctx := context.Background()
	now := time.Now()

	testCases := map[string]struct {
		db         string
		id         string
		deployment interface{}
		devices    []interface{}

		outstatus        model.DeploymentStatus
		outerr           error
		ignoreMaxDevices bool
	}{
		"pending 1": {
			db: "deployments_service",
			id: "dep-pending-1",
			deployment: model.Deployment{
				Id: "dep-pending-1",
				Stats: model.Stats{
					model.DeviceDeploymentStatusPending:        3,
					model.DeviceDeploymentStatusDownloading:    0,
					model.DeviceDeploymentStatusInstalling:     0,
					model.DeviceDeploymentStatusRebooting:      0,
					model.DeviceDeploymentStatusSuccess:        0,
					model.DeviceDeploymentStatusAlreadyInst:    0,
					model.DeviceDeploymentStatusFailure:        0,
					model.DeviceDeploymentStatusNoArtifact:     0,
					model.DeviceDeploymentStatusAborted:        0,
					model.DeviceDeploymentStatusDecommissioned: 0,
				},
			},
			devices: []interface{}{
				model.DeviceDeployment{
					Id:           "1",
					DeploymentId: "dep-pending-1",
					Status:       "pending",
				},
				model.DeviceDeployment{
					Id:           "2",
					DeploymentId: "dep-pending-1",
					Status:       "pending",
				},
				model.DeviceDeployment{
					Id:           "3",
					DeploymentId: "dep-pending-1",
					Status:       "pending",
				},
			},
			outstatus: "pending",
		},
		"pending 2": {
			db: "deployments_service",
			id: "dep-pending-1",
			deployment: model.Deployment{
				Id: "dep-pending-1",
				Stats: model.Stats{
					model.DeviceDeploymentStatusPending:        1,
					model.DeviceDeploymentStatusDownloading:    0,
					model.DeviceDeploymentStatusInstalling:     0,
					model.DeviceDeploymentStatusRebooting:      0,
					model.DeviceDeploymentStatusSuccess:        0,
					model.DeviceDeploymentStatusAlreadyInst:    0,
					model.DeviceDeploymentStatusFailure:        0,
					model.DeviceDeploymentStatusNoArtifact:     0,
					model.DeviceDeploymentStatusAborted:        0,
					model.DeviceDeploymentStatusDecommissioned: 0,
				},
			},
			devices: []interface{}{
				model.DeviceDeployment{
					Id:           "1",
					DeploymentId: "dep-pending-1",
					Status:       "pending",
				},
			},
			outstatus: "pending",
		},
		"inprogress 1": {
			db: "deployments_service",
			id: "dep-inprog-1",
			deployment: model.Deployment{
				Id: "dep-inprog-1",
				Stats: model.Stats{
					model.DeviceDeploymentStatusPending:        2,
					model.DeviceDeploymentStatusDownloading:    1,
					model.DeviceDeploymentStatusInstalling:     0,
					model.DeviceDeploymentStatusRebooting:      0,
					model.DeviceDeploymentStatusSuccess:        0,
					model.DeviceDeploymentStatusAlreadyInst:    0,
					model.DeviceDeploymentStatusFailure:        1,
					model.DeviceDeploymentStatusNoArtifact:     0,
					model.DeviceDeploymentStatusAborted:        0,
					model.DeviceDeploymentStatusDecommissioned: 1,
				},
			},
			devices: []interface{}{
				model.DeviceDeployment{
					Id:           "1",
					DeploymentId: "dep-inprog-1",
					Status:       "pending",
				},
				model.DeviceDeployment{
					Id:           "2",
					DeploymentId: "dep-inprog-1",
					Status:       "downloading",
				},
				model.DeviceDeployment{
					Id:           "3",
					DeploymentId: "dep-inprog-1",
					Status:       "failure",
				},
				model.DeviceDeployment{
					Id:           "4",
					DeploymentId: "dep-inprog-1",
					Status:       "decommissioned",
				},
			},
			outstatus: "inprogress",
		},
		"inprogress 2": {
			db: "deployments_service",
			id: "dep-inprog-2",
			deployment: model.Deployment{
				Id: "dep-inprog-2",
				Stats: model.Stats{
					model.DeviceDeploymentStatusPending:        0,
					model.DeviceDeploymentStatusDownloading:    1,
					model.DeviceDeploymentStatusInstalling:     1,
					model.DeviceDeploymentStatusRebooting:      1,
					model.DeviceDeploymentStatusSuccess:        1,
					model.DeviceDeploymentStatusAlreadyInst:    0,
					model.DeviceDeploymentStatusFailure:        1,
					model.DeviceDeploymentStatusNoArtifact:     0,
					model.DeviceDeploymentStatusAborted:        0,
					model.DeviceDeploymentStatusDecommissioned: 0,
				},
			},
			devices: []interface{}{
				model.DeviceDeployment{
					Id:           "1",
					DeploymentId: "dep-inprog-2",
					Status:       "downloading",
				},
				model.DeviceDeployment{
					Id:           "2",
					DeploymentId: "dep-inprog-2",
					Status:       "installing",
				},
				model.DeviceDeployment{
					Id:           "3",
					DeploymentId: "dep-inprog-2",
					Status:       "rebooting",
				},
				model.DeviceDeployment{
					Id:           "4",
					DeploymentId: "dep-inprog-2",
					Status:       "success",
				},
			},
			outstatus: "inprogress",
		},
		"inprogress 3": {
			db: "deployments_service",
			id: "dep-inprog-3",
			deployment: model.Deployment{
				Id: "dep-inprog-3",
				Stats: model.Stats{
					model.DeviceDeploymentStatusPending:        1,
					model.DeviceDeploymentStatusDownloading:    0,
					model.DeviceDeploymentStatusInstalling:     0,
					model.DeviceDeploymentStatusRebooting:      0,
					model.DeviceDeploymentStatusSuccess:        2,
					model.DeviceDeploymentStatusAlreadyInst:    0,
					model.DeviceDeploymentStatusFailure:        1,
					model.DeviceDeploymentStatusNoArtifact:     0,
					model.DeviceDeploymentStatusAborted:        0,
					model.DeviceDeploymentStatusDecommissioned: 0,
				},
			},
			devices: []interface{}{
				model.DeviceDeployment{
					Id:           "1",
					DeploymentId: "dep-inprog-3",
					Status:       "pending",
				},
				model.DeviceDeployment{
					Id:           "2",
					DeploymentId: "dep-inprog-3",
					Status:       "success",
				},
				model.DeviceDeployment{
					Id:           "3",
					DeploymentId: "dep-inprog-3",
					Status:       "success",
				},
				model.DeviceDeployment{
					Id:           "4",
					DeploymentId: "dep-inprog-3",
					Status:       "failure",
				},
			},
			outstatus: "inprogress",
		},
		"finished (normally - pending down 0, all devs finished)": {
			db: "deployments_service",
			id: "finished-1",
			deployment: model.Deployment{
				Id: "finished-1",
				Stats: model.Stats{
					model.DeviceDeploymentStatusPending:        0,
					model.DeviceDeploymentStatusDownloading:    0,
					model.DeviceDeploymentStatusInstalling:     0,
					model.DeviceDeploymentStatusRebooting:      0,
					model.DeviceDeploymentStatusSuccess:        2,
					model.DeviceDeploymentStatusAlreadyInst:    0,
					model.DeviceDeploymentStatusFailure:        3,
					model.DeviceDeploymentStatusNoArtifact:     0,
					model.DeviceDeploymentStatusAborted:        0,
					model.DeviceDeploymentStatusDecommissioned: 0,
				},
			},
			devices: []interface{}{
				model.DeviceDeployment{
					Id:           "1",
					DeploymentId: "finished-1",
					Status:       "success",
				},
				model.DeviceDeployment{
					Id:           "2",
					DeploymentId: "finished-1",
					Status:       "success",
				},
				model.DeviceDeployment{
					Id:           "3",
					DeploymentId: "finished-1",
					Status:       "failure",
				},
				model.DeviceDeployment{
					Id:           "4",
					DeploymentId: "finished-1",
					Status:       "failure",
				},
			},
			outstatus: "finished",
		},
		"finished (with some noartifacts, decomms)": {
			db: "deployments_service",
			id: "finished-2",
			deployment: model.Deployment{
				Id: "finished-2",
				Stats: model.Stats{
					model.DeviceDeploymentStatusPending:        0,
					model.DeviceDeploymentStatusDownloading:    0,
					model.DeviceDeploymentStatusInstalling:     0,
					model.DeviceDeploymentStatusRebooting:      0,
					model.DeviceDeploymentStatusSuccess:        1,
					model.DeviceDeploymentStatusAlreadyInst:    0,
					model.DeviceDeploymentStatusFailure:        2,
					model.DeviceDeploymentStatusNoArtifact:     1,
					model.DeviceDeploymentStatusAborted:        0,
					model.DeviceDeploymentStatusDecommissioned: 1,
				},
			},
			devices: []interface{}{
				model.DeviceDeployment{
					Id:           "1",
					DeploymentId: "finished-2",
					Status:       "success",
				},
				model.DeviceDeployment{
					Id:           "2",
					DeploymentId: "finished-2",
					Status:       "failure",
				},
				model.DeviceDeployment{
					Id:           "3",
					DeploymentId: "finished-2",
					Status:       "failure",
				},
				model.DeviceDeployment{
					Id:           "4",
					DeploymentId: "finished-2",
					Status:       "decommissioned",
				},
				model.DeviceDeployment{
					Id:           "5",
					DeploymentId: "finished-2",
					Status:       "noartifact",
				},
			},
			outstatus: "finished",
		},
		"finished (via abort)": {
			db: "deployments_service",
			id: "finished-3",
			deployment: model.Deployment{
				Id: "finished-3",
				Stats: model.Stats{
					model.DeviceDeploymentStatusPending:        0,
					model.DeviceDeploymentStatusDownloading:    0,
					model.DeviceDeploymentStatusInstalling:     0,
					model.DeviceDeploymentStatusRebooting:      0,
					model.DeviceDeploymentStatusSuccess:        1,
					model.DeviceDeploymentStatusAlreadyInst:    0,
					model.DeviceDeploymentStatusFailure:        2,
					model.DeviceDeploymentStatusNoArtifact:     1,
					model.DeviceDeploymentStatusAborted:        0,
					model.DeviceDeploymentStatusDecommissioned: 1,
				},
			},
			devices: []interface{}{
				model.DeviceDeployment{
					Id:           "1",
					DeploymentId: "finished-3",
					Status:       "aborted",
				},
				model.DeviceDeployment{
					Id:           "2",
					DeploymentId: "finished-3",
					Status:       "aborted",
				},
				model.DeviceDeployment{
					Id:           "3",
					DeploymentId: "finished-3",
					Status:       "aborted",
				},
				model.DeviceDeployment{
					Id:           "4",
					DeploymentId: "finished-3",
					Status:       "aborted",
				},
				model.DeviceDeployment{
					Id:           "5",
					DeploymentId: "finished-3",
					Status:       "aborted",
				},
			},
			outstatus: "finished",
		},
		"finished (because of timestamp)": {
			db: "deployments_service",
			id: "finished-4",
			deployment: model.Deployment{
				Id:       "finished-4",
				Finished: &now,
				Stats: model.Stats{
					model.DeviceDeploymentStatusPending:        0,
					model.DeviceDeploymentStatusDownloading:    0,
					model.DeviceDeploymentStatusInstalling:     0,
					model.DeviceDeploymentStatusRebooting:      0,
					model.DeviceDeploymentStatusSuccess:        1,
					model.DeviceDeploymentStatusAlreadyInst:    0,
					model.DeviceDeploymentStatusFailure:        0,
					model.DeviceDeploymentStatusNoArtifact:     0,
					model.DeviceDeploymentStatusAborted:        0,
					model.DeviceDeploymentStatusDecommissioned: 0,
				},
			},
			devices: []interface{}{
				model.DeviceDeployment{
					Id:           "3",
					DeploymentId: "dep-inprog-3",
					Status:       "success",
				},
			},
			outstatus:        "finished",
			ignoreMaxDevices: true,
		},
	}

	for name, tc := range testCases {
		t.Logf("test case: %s", name)

		db.Wipe()
		c := db.Client()

		collDeps := c.Database(tc.db).Collection(CollectionDeployments)
		collDevDeps := c.Database(tc.db).Collection(CollectionDevices)

		// setup migrations up to 1.2.2
		migrations := []migrate.Migration{
			&migration_1_2_1{
				client: c,
				db:     tc.db,
			},
			&migration_1_2_2{
				client: c,
				db:     tc.db,
			},
		}

		m := migrate.SimpleMigrator{
			Client:      c,
			Db:          tc.db,
			Automigrate: true,
		}

		err := m.Apply(ctx, migrate.MakeVersion(1, 2, 2), migrations)
		assert.NoError(t, err)

		// setup test deployments
		_, err = collDeps.InsertOne(context.TODO(), tc.deployment, &options.InsertOneOptions{})
		assert.NoError(t, err)

		_, err = collDevDeps.InsertMany(context.TODO(), tc.devices, nil)
		assert.NoError(t, err)

		// apply 1.2.4
		migrations = []migrate.Migration{
			&migration_1_2_4{
				client: c,
				db:     tc.db,
			},
		}

		err = m.Apply(ctx, migrate.MakeVersion(1, 2, 4), migrations)
		assert.NoError(t, err)

		// verify statuses
		var out model.Deployment
		cur := collDeps.FindOne(ctx, bson.M{"_id": tc.id}, &options.FindOneOptions{})

		err = cur.Decode(&out)
		assert.NoError(t, err)

		assert.Equal(t, tc.outstatus, out.Status)

		if !tc.ignoreMaxDevices {
			assert.Equal(t, len(tc.devices), out.MaxDevices)
		}

		// verify index exists
		indexes := collDeps.Indexes()
		cursor, _ := indexes.List(ctx)

		var idxResults []bson.M
		err = cursor.All(context.TODO(), &idxResults)
		assert.NoError(t, err)

		found := false
		for _, i := range idxResults {
			if i["name"] == IndexDeploymentStatus {
				found = true
			}
		}

		assert.Equal(t, true, found)
	}
}
