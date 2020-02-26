// Copyright 2020 Northern.tech AS
//
//    All Rights Reserved
package mongo

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendersoftware/go-lib-micro/mongo/doc"
	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type migration_1_2_3 struct {
	client *mongo.Client
	db     string
}

// Up intrduces a unique index on artifact depends_idx and name to ensure unique depends in a release, also:
// - drops index on DeviceTypesCompatible, superseded by the above
// - rewrites DeviceTypesCompatible to 'depends.device_type' - even for v1, v2 artifacts
func (m *migration_1_2_3) Up(from migrate.Version) error {
	ctx := context.Background()
	c := m.client.Database(m.db).Collection(CollectionImages)
	storage := NewDataStoreMongoWithClient(m.client)

	// drop old device type + name index
	_, err := c.Indexes().DropOne(ctx, IndexUniqueNameAndDeviceTypeName)

	// the index might not be there - was created only on image inserts (not upfront)
	if err != nil &&
		!strings.Contains(err.Error(), "index not found with name") &&
		!strings.Contains(err.Error(), "ns not found") {
		return err
	}

	// transform existing device_types_compatible in v1 and v2 artifacts into 'depends.device_type'
	artifacts, err := storage.FindAll(ctx)
	if err != nil {
		return err
	}

	for _, a := range artifacts {
		// storage.Update is broken, do it manually via the driver
		dtypes := make([]interface{}, len(a.ArtifactMeta.DeviceTypesCompatible))

		for i, d := range a.ArtifactMeta.DeviceTypesCompatible {
			dtypes[i] = interface{}(d)
		}

		depends := bson.M{
			ArtifactDependsDeviceType: dtypes,
		}

		dependsIdx, err := doc.UnwindMap(depends)
		if err != nil {
			return err
		}

		up := bson.M{
			"$set": bson.M{
				StorageKeyImageDepends:    depends,
				StorageKeyImageDependsIdx: dependsIdx,
			},
		}

		res, err := c.UpdateOne(ctx, bson.M{"_id": a.Id}, up)
		if err != nil {
			return err
		}

		if res.MatchedCount != 1 {
			return errors.New(fmt.Sprintf("failed to update artifact %s: not found", a.Id))
		}
	}

	// create new artifact depends + name index
	err = storage.EnsureIndexes(m.db,
		CollectionImages,
		IndexArtifactNameDepends)

	return nil
}

func (m *migration_1_2_3) Version() migrate.Version {
	return migrate.MakeVersion(1, 2, 3)
}
