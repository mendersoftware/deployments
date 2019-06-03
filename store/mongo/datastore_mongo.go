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
	"crypto/tls"
	"net"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/mendersoftware/go-lib-micro/config"
	mstore "github.com/mendersoftware/go-lib-micro/store"
	"github.com/pkg/errors"

	dconfig "github.com/mendersoftware/deployments/config"
	"github.com/mendersoftware/deployments/migrations"
	"github.com/mendersoftware/deployments/model"
	mimages "github.com/mendersoftware/deployments/resources/images/mongo"
)

const (
	DatabaseName     = "deployment_service"
	CollectionLimits = "limits"
)

var (
	ErrLimitNotFound = errors.New("limit not found")
)

type DataStoreMongo struct {
	session *mgo.Session
}

func NewDataStoreMongoWithSession(session *mgo.Session) *DataStoreMongo {
	return &DataStoreMongo{
		session: session,
	}
}

func NewMongoSession(c config.Reader) (*mgo.Session, error) {

	dialInfo, err := mgo.ParseURL(c.GetString(dconfig.SettingMongo))
	if err != nil {
		return nil, errors.Wrap(err, "failed to open mgo session")
	}

	// Set 10s timeout - same as set by Dial
	dialInfo.Timeout = 10 * time.Second

	username := c.GetString(dconfig.SettingDbUsername)
	if username != "" {
		dialInfo.Username = username
	}

	passward := c.GetString(dconfig.SettingDbPassword)
	if passward != "" {
		dialInfo.Password = passward
	}

	if c.GetBool(dconfig.SettingDbSSL) {
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {

			// Setup TLS
			tlsConfig := &tls.Config{}
			tlsConfig.InsecureSkipVerify = c.GetBool(dconfig.SettingDbSSLSkipVerify)

			conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
			return conn, err
		}
	}

	masterSession, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open mgo session")
	}

	// Validate connection
	if err := masterSession.Ping(); err != nil {
		return nil, errors.Wrap(err, "failed to open mgo session")
	}

	// force write ack with immediate journal file fsync
	masterSession.SetSafe(&mgo.Safe{
		W: 1,
		J: true,
	})

	return masterSession, nil
}

func (db *DataStoreMongo) GetReleases(ctx context.Context, filt *model.ReleaseFilter) ([]model.Release, error) {
	session := db.session.Copy()
	defer session.Close()

	match := db.matchFromFilt(filt)

	group := bson.M{
		"$group": bson.M{
			"_id": "$" + mimages.StorageKeySoftwareImageName,
			"name": bson.M{
				"$first": "$" + mimages.StorageKeySoftwareImageName,
			},
			"artifacts": bson.M{
				"$push": "$$ROOT",
			},
		},
	}

	sort := bson.M{
		"$sort": bson.M{
			"name": -1,
		},
	}

	var pipe []bson.M

	if match != nil {
		pipe = []bson.M{
			match,
			group,
			sort,
		}
	} else {
		pipe = []bson.M{
			group,
			sort,
		}
	}

	results := []model.Release{}

	err := session.DB(mstore.DbFromContext(ctx, mimages.DatabaseName)).
		C(mimages.CollectionImages).Pipe(&pipe).All(&results)
	if err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return results, nil
}

func (db *DataStoreMongo) matchFromFilt(f *model.ReleaseFilter) bson.M {
	if f == nil {
		return nil
	}

	return bson.M{
		"$match": bson.M{
			mimages.StorageKeySoftwareImageName: f.Name,
		},
	}
}

// limits
//
func (db *DataStoreMongo) GetLimit(ctx context.Context, name string) (*model.Limit, error) {

	session := db.session.Copy()
	defer session.Close()

	var limit model.Limit
	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionLimits).FindId(name).One(&limit); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, ErrLimitNotFound
		}
		return nil, err
	}

	return &limit, nil
}

func (db *DataStoreMongo) ProvisionTenant(ctx context.Context, tenantId string) error {
	session := db.session.Copy()
	defer session.Close()

	dbname := mstore.DbNameForTenant(tenantId, migrations.DbName)

	return migrations.MigrateSingle(ctx, dbname, migrations.DbVersion, session, true)
}
