// Copyright 2020 Northern.tech AS
//
//    All Rights Reserved
package mongo

import (
	"context"
	"strings"
	"testing"

	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	"github.com/stretchr/testify/assert"

	"github.com/mendersoftware/deployments/model"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	errDupPrefix = `artifact has duplicate 'depends' attributes: `
	errDupSuffix = `: artifact not unique`
)

func TestMigration_1_2_3_DeviceTypeNameIndexReplaced(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMigration_1_2_3_DeviceTypeNameIndexReplace in short mode.")
	}

	//verify that, for 'old' artifacts, inserted before the migration:
	// - uniqueness of device_types_compatible and name will be preserved, even though the index
	//   was dropped (and values rewritten to 'depends')
	// - old artifacts won't prevent new ones from being inserted only based on device type + name
	//   (extra depends = no overlap)
	inputImages := []*model.Image{
		&model.Image{
			Id: "0cb87b3d-4f08-420b-b004-4347c07f70f6",
			ArtifactMeta: model.ArtifactMeta{
				Name:                  "release1",
				DeviceTypesCompatible: []string{"arm6"},
			},
		},
		&model.Image{
			Id: "0cb87b3d-4f08-420b-b004-4347c07f70f7",
			ArtifactMeta: model.ArtifactMeta{
				Name:                  "release1",
				DeviceTypesCompatible: []string{"arm7"},
			},
		},
		&model.Image{
			Id: "0cb87b3d-4f08-420b-b004-4347c07f70f8",
			ArtifactMeta: model.ArtifactMeta{
				Name:                  "release2",
				DeviceTypesCompatible: []string{"arm8", "arm9"},
			},
		},
	}

	testCases := map[string]struct {
		img     *model.Image
		errMsgs []string
	}{
		"conflict 1": {
			img: &model.Image{
				Id: "0cb87b3d-4f08-420b-b004-4347c07f70f9",
				ArtifactMeta: model.ArtifactMeta{
					Name:                  "release1",
					DeviceTypesCompatible: []string{"arm6"},
					Depends: bson.M{
						ArtifactDependsDeviceType: []interface{}{"arm6"},
					},
					DependsIdx: []bson.D{
						bson.D{
							bson.E{Key: ArtifactDependsDeviceType, Value: "arm6"},
						},
					},
				},
			},
			errMsgs: []string{
				`device_type: "arm6"`,
			},
		},
		"conflict 2": {
			img: &model.Image{
				Id: "0cb87b3d-4f08-420b-b004-4347c07f70f9",
				ArtifactMeta: model.ArtifactMeta{
					Name:                  "release1",
					DeviceTypesCompatible: []string{"arm7"},
					Depends: bson.M{
						ArtifactDependsDeviceType: []interface{}{"arm7"},
					},
					DependsIdx: []bson.D{
						bson.D{
							bson.E{Key: ArtifactDependsDeviceType, Value: "arm7"},
						},
					},
				},
			},
			errMsgs: []string{
				`device_type: "arm7"`,
			},
		},
		"conflict 3": {
			img: &model.Image{
				Id: "0cb87b3d-4f08-420b-b004-4347c07f70f9",
				ArtifactMeta: model.ArtifactMeta{
					Name:                  "release1",
					DeviceTypesCompatible: []string{"arm6", "arm7"},
					Depends: bson.M{
						ArtifactDependsDeviceType: []interface{}{"arm6"},
					},
					DependsIdx: []bson.D{
						bson.D{
							bson.E{Key: ArtifactDependsDeviceType, Value: "arm6"},
						},
						bson.D{
							bson.E{Key: ArtifactDependsDeviceType, Value: "arm5"},
						},
					},
				},
			},
			errMsgs: []string{
				`device_type: "arm6"`,
			},
		},
		"no conflict 1: different release": {
			img: &model.Image{
				Id: "0cb87b3d-4f08-420b-b004-4347c07f70f9",
				ArtifactMeta: model.ArtifactMeta{
					Name:                  "release2",
					DeviceTypesCompatible: []string{"arm6", "arm7"},
					Depends: bson.M{
						ArtifactDependsDeviceType: []interface{}{"arm6"},
					},
					DependsIdx: []bson.D{
						bson.D{

							bson.E{Key: ArtifactDependsDeviceType, Value: "arm6"},
						},
						bson.D{
							bson.E{Key: ArtifactDependsDeviceType, Value: "arm7"},
						},
					},
				},
			},
		},
		"no conflict 2: artifact has extra depends = no overlap": {
			img: &model.Image{
				Id: "0cb87b3d-4f08-420b-b004-4347c07f70f9",
				ArtifactMeta: model.ArtifactMeta{
					Name:                  "release1",
					DeviceTypesCompatible: []string{"arm6", "arm7"},
					Depends: bson.M{
						ArtifactDependsDeviceType: []interface{}{"arm6"},
						"checksum":                "1",
					},
					DependsIdx: []bson.D{
						bson.D{
							bson.E{Key: "checksum", Value: "1"},
							bson.E{Key: ArtifactDependsDeviceType, Value: "arm6"},
						},
						bson.D{
							bson.E{Key: "checksum", Value: "1"},
							bson.E{Key: ArtifactDependsDeviceType, Value: "arm7"},
						},
					},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Logf("test case: %s", name)

		db.Wipe()
		c := db.Client()

		ctx := context.TODO()

		store := NewDataStoreMongoWithClient(c)

		// bring db to version just-before new (1.2.2)
		migrations := []migrate.Migration{
			&migration_1_2_1{
				client: c,
				db:     DbName,
			},
			&migration_1_2_2{
				client: c,
				db:     DbName,
			},
		}

		m := migrate.SimpleMigrator{
			Client:      c,
			Db:          DbName,
			Automigrate: true,
		}

		err := m.Apply(ctx, migrate.MakeVersion(1, 2, 2), migrations)
		assert.NoError(t, err)

		// insert input images
		for _, i := range inputImages {
			err = store.InsertImage(ctx, i)
			assert.NoError(t, err)
		}

		// bring db to latest version (1.2.3)
		mnew := &migration_1_2_3{
			client: c,
			db:     DbName,
		}

		err = mnew.Up(migrate.MakeVersion(1, 2, 3))
		assert.NoError(t, err)

		// try insert image under test
		err = store.InsertImage(ctx, tc.img)
		if tc.errMsgs != nil {
			assert.NotNil(t, err)
			assertDupErr(t, err, tc.errMsgs)
		} else {
			assert.NoError(t, err)
		}

		all, _ := store.FindAll(ctx)
		for _, a := range all {
			assert.NotNil(t, a.ArtifactMeta.Depends)
			assert.NotNil(t, a.ArtifactMeta.DependsIdx)
		}
	}

}

func TestMigration_1_2_3_OverlappingDepends(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMigration_1_2_3_OverlappingDepends in short mode.")
	}

	// verify that, for new v3 artifacts, 'depends' uniqueness is
	// detected correctly, i.e. by checking overlaps via the
	// exploded depends index
	inputImages := []*model.Image{
		&model.Image{
			Id: "0cb87b3d-4f08-420b-b004-4347c07f70f9",
			ArtifactMeta: model.ArtifactMeta{
				Name:                  "release1",
				DeviceTypesCompatible: []string{"arm6"},
				Depends: bson.M{
					ArtifactDependsDeviceType: []interface{}{"arm6"},
					"checksum":                "1",
				},
				DependsIdx: []bson.D{
					bson.D{
						bson.E{Key: "checksum", Value: "1"},
						bson.E{Key: ArtifactDependsDeviceType, Value: "arm6"},
					},
				},
			},
		},
		&model.Image{
			Id: "0cb87b3d-4f08-420b-b004-4347c07f70fa",
			ArtifactMeta: model.ArtifactMeta{
				Name:                  "release1",
				DeviceTypesCompatible: []string{"arm7"},
				Depends: bson.M{
					ArtifactDependsDeviceType: []interface{}{"arm7"},
					"checksum":                "1",
				},
				DependsIdx: []bson.D{
					bson.D{
						bson.E{Key: "checksum", Value: "1"},
						bson.E{Key: ArtifactDependsDeviceType, Value: "arm7"},
					},
				},
			},
		},
		&model.Image{
			Id: "0cb87b3d-4f08-420b-b004-4347c07f70fb",
			ArtifactMeta: model.ArtifactMeta{
				Name:                  "release1",
				DeviceTypesCompatible: []string{"arm6"},
				Depends: bson.M{
					ArtifactDependsDeviceType: []interface{}{"arm6"},
					"checksum":                "2",
				},
				DependsIdx: []bson.D{
					bson.D{
						bson.E{Key: "checksum", Value: "2"},
						bson.E{Key: ArtifactDependsDeviceType, Value: "arm6"},
					},
				},
			},
		},
		&model.Image{
			Id: "0cb87b3d-4f08-420b-b004-4347c07f70fc",
			ArtifactMeta: model.ArtifactMeta{
				Name:                  "release1",
				DeviceTypesCompatible: []string{"arm8", "arm9"},
				Depends: bson.M{
					ArtifactDependsDeviceType: []interface{}{"arm8", "arm9"},
					"checksum":                "3",
					"foo":                     []interface{}{"foo1", "foo2"},
				},
				DependsIdx: []bson.D{
					bson.D{
						bson.E{Key: "checksum", Value: "3"},
						bson.E{Key: ArtifactDependsDeviceType, Value: "arm8"},
						bson.E{Key: "foo", Value: "foo1"},
					},

					bson.D{
						bson.E{Key: "checksum", Value: "3"},
						bson.E{Key: ArtifactDependsDeviceType, Value: "arm8"},
						bson.E{Key: "foo", Value: "foo2"},
					},
					bson.D{
						bson.E{Key: "checksum", Value: "3"},
						bson.E{Key: ArtifactDependsDeviceType, Value: "arm9"},
						bson.E{Key: "foo", Value: "foo1"},
					},
					bson.D{
						bson.E{Key: "checksum", Value: "3"},
						bson.E{Key: ArtifactDependsDeviceType, Value: "arm9"},
						bson.E{Key: "foo", Value: "foo2"},
					},
				},
			},
		},
	}

	testCases := map[string]struct {
		img     *model.Image
		errMsgs []string
	}{
		"conflict 1": {
			img: &model.Image{
				Id: "0cb87b3d-4f08-420b-b004-4347c07f70ff",
				ArtifactMeta: model.ArtifactMeta{
					Name:                  "release1",
					DeviceTypesCompatible: []string{"arm6", "arm7"},
					Depends: map[string]interface{}{
						ArtifactDependsDeviceType: []interface{}{"arm6", "arm7"},
						"checksum":                "2",
					},
					DependsIdx: []bson.D{
						bson.D{
							bson.E{
								Key: "checksum", Value: "2",
							},
							bson.E{
								Key: ArtifactDependsDeviceType, Value: "arm6",
							},
						},
						bson.D{
							bson.E{
								Key: "checksum", Value: "2",
							},
							bson.E{
								Key: ArtifactDependsDeviceType, Value: "arm7",
							},
						},
					},
				},
			},
			errMsgs: []string{
				`device_type: "arm6"`,
				`checksum: "2"`,
			},
		},
		"conflict 2": {
			img: &model.Image{
				Id: "0cb87b3d-4f08-420b-b004-4347c07f70ff",
				ArtifactMeta: model.ArtifactMeta{
					Name:                  "release1",
					DeviceTypesCompatible: []string{"arm6", "arm8"},
					Depends: map[string]interface{}{
						ArtifactDependsDeviceType: []interface{}{"arm6", "arm8"},
						"checksum":                "1",
					},
					DependsIdx: []bson.D{
						bson.D{
							bson.E{
								Key: "checksum", Value: "1",
							},
							bson.E{
								Key: ArtifactDependsDeviceType, Value: "arm6",
							},
						},
						bson.D{
							bson.E{
								Key: "checksum", Value: "1",
							},
							bson.E{
								Key: ArtifactDependsDeviceType, Value: "arm8",
							},
						},
					},
				},
			},
			errMsgs: []string{
				`device_type: "arm6"`,
				`checksum: "1"`,
			},
		},
		"conflict 3": {
			img: &model.Image{
				Id: "0cb87b3d-4f08-420b-b004-4347c07f70ff",
				ArtifactMeta: model.ArtifactMeta{
					Name:                  "release1",
					DeviceTypesCompatible: []string{"arm6"},
					Depends: map[string]interface{}{
						ArtifactDependsDeviceType: []interface{}{"arm6"},
						"checksum":                "1",
					},
					DependsIdx: []bson.D{
						bson.D{
							bson.E{
								Key: "checksum", Value: "1",
							},
							bson.E{
								Key: ArtifactDependsDeviceType, Value: "arm6",
							},
						},
					},
				},
			},
			errMsgs: []string{
				`device_type: "arm6"`,
				`checksum: "1"`,
			},
		},
		"conflict 4": {
			img: &model.Image{
				Id: "0cb87b3d-4f08-420b-b004-4347c07f70ff",
				ArtifactMeta: model.ArtifactMeta{
					Name:                  "release1",
					DeviceTypesCompatible: []string{"arm8"},
					Depends: map[string]interface{}{
						ArtifactDependsDeviceType: []interface{}{"arm8"},
						"checksum":                "3",
						"foo":                     "foo1",
					},
					DependsIdx: []bson.D{
						bson.D{
							bson.E{
								Key: "checksum", Value: "3",
							},
							bson.E{
								Key: ArtifactDependsDeviceType, Value: "arm8",
							},
							bson.E{
								Key: "foo", Value: "foo1",
							},
						},
					},
				},
			},
			errMsgs: []string{
				`device_type: "arm8"`,
				`checksum: "3"`,
				`foo: "foo1"`,
			},
		},
		"no conflict: overlap + extra param": {
			img: &model.Image{
				Id: "0cb87b3d-4f08-420b-b004-4347c07f70ff",
				ArtifactMeta: model.ArtifactMeta{
					Name:                  "release1",
					DeviceTypesCompatible: []string{"arm8"},
					Depends: map[string]interface{}{
						ArtifactDependsDeviceType: []interface{}{"arm8"},
						"checksum":                "3",
						"foo":                     "foo1",
						"bar":                     "bar1",
					},
					DependsIdx: []bson.D{
						bson.D{
							bson.E{
								Key: "checksum", Value: "1",
							},
							bson.E{
								Key: ArtifactDependsDeviceType, Value: "arm8",
							},
							bson.E{
								Key: "bar", Value: "bar1",
							},
							bson.E{
								Key: "foo", Value: "foo1",
							},
						},
					},
				},
			},
		},
		"no conflict: overlap but different release": {
			img: &model.Image{
				Id: "0cb87b3d-4f08-420b-b004-4347c07f70ff",
				ArtifactMeta: model.ArtifactMeta{
					Name:                  "release2",
					DeviceTypesCompatible: []string{"arm6"},
					Depends: map[string]interface{}{
						ArtifactDependsDeviceType: []interface{}{"arm6"},
						"checksum":                "1",
					},
					DependsIdx: []bson.D{
						bson.D{
							bson.E{
								Key: "checksum", Value: "1",
							},
							bson.E{
								Key: ArtifactDependsDeviceType, Value: "arm6",
							},
						},
					},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Logf("test case: %s", name)

		db.Wipe()
		c := db.Client()

		ctx := context.TODO()

		store := NewDataStoreMongoWithClient(c)

		// bring db to latest version (1.2.3)
		migrations := []migrate.Migration{
			&migration_1_2_1{
				client: c,
				db:     DbName,
			},
			&migration_1_2_2{
				client: c,
				db:     DbName,
			},
			&migration_1_2_3{
				client: c,
				db:     DbName,
			},
		}

		m := migrate.SimpleMigrator{
			Client:      c,
			Db:          DbName,
			Automigrate: true,
		}

		err := m.Apply(ctx, migrate.MakeVersion(1, 2, 3), migrations)
		assert.NoError(t, err)

		// insert input images
		for _, i := range inputImages {
			err = store.InsertImage(ctx, i)
			assert.NoError(t, err)
		}

		// try insert image under test
		err = store.InsertImage(ctx, tc.img)
		if tc.errMsgs != nil {
			assertDupErr(t, err, tc.errMsgs)
		} else {
			assert.NoError(t, err)
		}
	}
}

// assertDupErr verifies the error message w.r.t duplicated
// 'depends' values.
// it's not safe to compare messages verbatim because
// order of attributes is not guaranteed (maps underneath everything)
func assertDupErr(t *testing.T, err error, errMsgs []string) {
	if err == nil {
		assert.FailNow(t, "expected error to be non-nil")
	}

	msg := err.Error()
	assert.True(t, strings.Contains(msg, errDupPrefix))

	msg = strings.TrimPrefix(msg, errDupPrefix)
	msg = strings.TrimSuffix(msg, errDupSuffix)

	msgs := strings.Split(msg, ",")

	assert.Equal(t, len(errMsgs), len(msgs))

	for _, e := range errMsgs {
		found := false
		for _, m := range msgs {
			m = strings.TrimSpace(m)
			found = m == e
			if found {
				break
			}
		}
		assert.True(t, found)

	}
}
