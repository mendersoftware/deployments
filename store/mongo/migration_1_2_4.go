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
	"reflect"

	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/mendersoftware/deployments/model"
)

type migration_1_2_4 struct {
	client *mongo.Client
	db     string
}

func (m *migration_1_2_4) Up(from migrate.Version) error {
	ctx := context.TODO()
	l := log.FromContext(ctx)

	coll := m.client.Database(m.db).Collection(CollectionDeployments)

	// we'll be iterating and modifying - ensure every doc is handled exactly once
	fopts := options.FindOptions{}
	fopts.SetSnapshot(true)

	cur, err := coll.Find(context.Background(), bson.D{}, &fopts)
	if err != nil {
		return errors.Wrap(err, "failed to get deployments")
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var dep model.Deployment
		err := cur.Decode(&dep)
		if err != nil {
			return errors.Wrap(err, "failed to get deployment")
		}
		l.Infof("processing deployment %s with stats %v", *dep.Id, dep.Stats)

		newstats, err := m.aggregateDeviceStatuses(ctx, *dep.Id)
		l.Infof("computed stats: %v", dep.Stats)

		if !reflect.DeepEqual(newstats, dep.Stats) {
			l.Warnf("stats don't match, will overwrite")
		}

		// substitute stats to recalc status with deployment.GetStatus
		dep.Stats = newstats
		status := dep.GetStatus()

		res, err := coll.UpdateOne(ctx, bson.M{"_id": *dep.Id},
			bson.M{
				"$set": bson.M{
					StorageKeyDeploymentStats:  newstats,
					StorageKeyDeploymentStatus: status,
				},
			})

		if err != nil {
			return errors.Wrapf(err, "failed to update deployment %s", *dep.Id)
		}

		if res.MatchedCount == 0 {
			return errors.Wrapf(err, "can't find deployment for update: %s", *dep.Id)
		}

		l.Infof("processing deployment %s: success", *dep.Id)
	}

	if err := cur.Err(); err != nil {
		l.Warnf("cursor error after processing: %v", err)
		return err
	}

	// have an index on just the plain Deployment.Status field
	// for easy querying by status
	storage := NewDataStoreMongoWithClient(m.client)
	if err := storage.EnsureIndexes(m.db, CollectionDeployments,
		DeploymentStatusIndex); err != nil {
		return err
	}

	return nil
}

// aggregateDeviceStatuses calculates:
// - stats
// - statistics
// for deployment 'depId', based on individual device statuses
// it mirrors store.AggregateDeviceDeploymentByStatus, and freezes
// it's implementation in case it changes/is removed
// note that device statuses are the best bet as a single source of
// truth on deployment status (used for all GETs at the time of writing this migration)
func (m *migration_1_2_4) aggregateDeviceStatuses(ctx context.Context, depId string) (model.Stats, error) {
	deviceDeployments := m.client.Database(m.db).Collection(CollectionDevices)

	match := bson.D{
		{Key: "$match", Value: bson.M{
			StorageKeyDeviceDeploymentDeploymentID: depId}},
	}
	group := bson.D{
		{Key: "$group", Value: bson.D{
			{Key: "_id",
				Value: "$" + StorageKeyDeviceDeploymentStatus},
			{Key: "count",
				Value: bson.M{"$sum": 1}}},
		},
	}
	pipeline := []bson.D{
		match,
		group,
	}

	var results []struct {
		Name  string `bson:"_id"`
		Count int
	}

	cursor, err := deviceDeployments.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	if err := cursor.All(ctx, &results); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	raw := model.NewDeviceDeploymentStats()
	for _, res := range results {
		raw[res.Name] = res.Count
	}
	return raw, nil
}

func (m *migration_1_2_4) Version() migrate.Version {
	return migrate.MakeVersion(1, 2, 4)
}

func statKey(counter string) string {
	return StorageKeyDeploymentStats + "." + counter
}
