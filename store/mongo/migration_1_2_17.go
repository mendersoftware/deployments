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

	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	mopts "go.mongodb.org/mongo-driver/mongo/options"
)

type migration_1_2_17 struct {
	client *mongo.Client
	db     string
}

// Up creates an index for filtering update types
func (m *migration_1_2_17) Up(from migrate.Version) error {
	ctx := context.Background()
	idxReleases := m.client.
		Database(m.db).
		Collection(CollectionReleases).
		Indexes()

	_, err := idxReleases.CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{
				Key:   StorageKeyReleaseArtifactsUpdateTypes,
				Value: 1,
			},
		},
		Options: mopts.Index().
			SetName(IndexNameReleaseUpdateTypes).
			SetSparse(true),
	})
	if err != nil {
		return fmt.Errorf("mongo(1.2.17): failed to create index: %w", err)
	}
	return nil
}

func (m *migration_1_2_17) Version() migrate.Version {
	return migrate.MakeVersion(1, 2, 17)
}
