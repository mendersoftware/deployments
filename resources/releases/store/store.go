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
package store

import (
	"context"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	mstore "github.com/mendersoftware/go-lib-micro/store"

	"github.com/mendersoftware/deployments/model"
	mimages "github.com/mendersoftware/deployments/resources/images/mongo"
)

type Store interface {
	GetReleases(ctx context.Context, filt *model.ReleaseFilter) ([]model.Release, error)
}

type store struct {
	session *mgo.Session
}

func NewStore(session *mgo.Session) *store {
	return &store{
		session: session,
	}
}

func (s *store) GetReleases(ctx context.Context, filt *model.ReleaseFilter) ([]model.Release, error) {
	session := s.session.Copy()
	defer session.Close()

	match := s.matchFromFilt(filt)

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

func (s *store) matchFromFilt(f *model.ReleaseFilter) bson.M {
	if f == nil {
		return nil
	}

	return bson.M{
		"$match": bson.M{
			mimages.StorageKeySoftwareImageName: f.Name,
		},
	}
}
