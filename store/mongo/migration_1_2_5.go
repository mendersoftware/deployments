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
	"strings"

	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	"go.mongodb.org/mongo-driver/mongo"
)

type migration_1_2_5 struct {
	client *mongo.Client
	db     string
}

func (m *migration_1_2_5) Up(from migrate.Version) error {
	ctx := context.Background()
	storage := NewDataStoreMongoWithClient(m.client)
	coll := m.client.Database(m.db).Collection(CollectionDevices)
	if _, err := coll.Indexes().DropOne(ctx, IndexDeploymentDeviceStatusesName); err != nil && !strings.Contains(err.Error(), "NamespaceNotFound") {
		// Supress NamespaceNotFound errors - index is missing
		return err
	}
	if _, err := coll.Indexes().DropOne(ctx, IndexDeploymentDeviceDeploymentIdName); err != nil && !strings.Contains(err.Error(), "NamespaceNotFound") {
		// Supress NamespaceNotFound errors - index is missing
		return err
	}
	return storage.EnsureIndexes(m.db, CollectionDevices,
		DeviceIDCreatedStatusIndex,
		DeploymentIdIndexes,
	)
}

func (m *migration_1_2_5) Version() migrate.Version {
	return migrate.MakeVersion(1, 2, 5)
}
