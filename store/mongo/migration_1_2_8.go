// Copyright 2022 Northern.tech AS
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

	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mendersoftware/deployments/model"
)

type migration_1_2_8 struct {
	client *mongo.Client
	db     string
}

func (m *migration_1_2_8) Up(from migrate.Version) error {
	ctx := context.Background()
	storage := NewDataStoreMongoWithClient(m.client)
	collDeployments := storage.client.Database(m.db).Collection(CollectionDeployments)
	_, err := collDeployments.UpdateMany(ctx, bson.M{
		StorageKeyDeploymentStatus: bson.M{"$ne": string(model.DeploymentStatusFinished)},
	}, bson.M{
		"$set": bson.M{StorageKeyDeploymentActive: true},
	})
	if err == nil {
		err = storage.EnsureIndexes(m.db, CollectionDeployments,
			IndexDeploymentsActiveCreatedModel,
		)
	}
	return err
}

func (m *migration_1_2_8) Version() migrate.Version {
	return migrate.MakeVersion(1, 2, 8)
}
