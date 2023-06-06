// Copyright 2023 Northern.tech AS
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
	"fmt"
	"time"

	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	mopts "go.mongodb.org/mongo-driver/mongo/options"

	"github.com/mendersoftware/deployments/model"
)

type migration_1_2_15 struct {
	client *mongo.Client
	db     string
}

// Up intrduces special representation of artifact provides and index them.
func (m *migration_1_2_15) Up(from migrate.Version) error {
	ctx := context.Background()
	c := m.client.Database(m.db).Collection(CollectionImages)
	cr := m.client.Database(m.db).Collection(CollectionReleases)

	// create index for release name
	storage := NewDataStoreMongoWithClient(m.client)
	if err := storage.EnsureIndexes(
		m.db,
		CollectionReleases,
		IndexReleaseName,
	); err != nil {
		return err
	}

	cursor, err := c.Find(ctx, bson.M{})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var a model.Image
		err = cursor.Decode(&a)
		if err != nil {
			break
		}

		opt := &mopts.UpdateOptions{}
		upsert := true
		opt.Upsert = &upsert
		update := bson.M{
			"$set": bson.M{
				StorageKeyReleaseName:     a.ArtifactMeta.Name,
				StorageKeyReleaseModified: time.Now(),
			},
			"$push": bson.M{StorageKeyReleaseArtifacts: a},
		}

		res, err := cr.UpdateOne(
			ctx,
			bson.M{
				"$and": []bson.M{
					{
						StorageKeyReleaseName: a.ArtifactMeta.Name,
					},
					{
						StorageKeyReleaseArtifactsId: bson.M{
							"$ne": a.Id,
						},
					},
				}},
			update,
			opt,
		)
		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				continue
			}
			return err
		}

		if res.ModifiedCount != 1 && res.UpsertedCount != 1 {
			return errors.New(fmt.Sprintf("failed to update release %s", a.ArtifactMeta.Name))
		}
	}
	if err = cursor.Err(); err != nil {
		return errors.WithMessage(err, "failed to decode artifact")
	}

	return nil
}

func (m *migration_1_2_15) Version() migrate.Version {
	return migrate.MakeVersion(1, 2, 15)
}
