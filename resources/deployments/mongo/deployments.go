// Copyright 2016 Mender Software AS
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
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/mendersoftware/go-lib-micro/store"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/mendersoftware/deployments/resources/deployments"
)

// Database settings
const (
	DatabaseName          = "deployment_service"
	CollectionDeployments = "deployments"
)

// Errors
var (
	ErrDeploymentStorageInvalidDeployment = errors.New("Invalid deployment")
	ErrStorageInvalidID                   = errors.New("Invalid id")
	ErrStorageNotFound                    = errors.New("Not found")
	ErrDeploymentStorageInvalidQuery      = errors.New("Invalid query")
	ErrDeploymentStorageCannotExecQuery   = errors.New("Cannot execute query")
	ErrStorageInvalidInput                = errors.New("invalid input")
)

const (
	StorageKeyDeploymentName         = "deploymentconstructor.name"
	StorageKeyDeploymentArtifactName = "deploymentconstructor.artifactname"
	StorageKeyDeploymentStats        = "stats"
	StorageKeyDeploymentFinished     = "finished"
)

var (
	StorageIndexes = []string{
		"$text:" + StorageKeyDeploymentName,
		"$text:" + StorageKeyDeploymentArtifactName,
	}
)

// DeploymentsStorage is a data layer for deployments based on MongoDB
type DeploymentsStorage struct {
	session *mgo.Session
}

// NewDeploymentsStorage new data layer object
func NewDeploymentsStorage(session *mgo.Session) *DeploymentsStorage {
	return &DeploymentsStorage{
		session: session,
	}
}

func (d *DeploymentsStorage) ensureIndexing(ctx context.Context, session *mgo.Session) error {
	return session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).
		EnsureIndexKey(StorageIndexes...)
}

// return true if required indexing was set up
func (d *DeploymentsStorage) hasIndexing(ctx context.Context, session *mgo.Session) bool {
	idxs, err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).Indexes()
	if err != nil {
		// check failed, assume indexing is not there
		return false
	}

	has := map[string]bool{}
	for _, idx := range idxs {
		for _, i := range idx.Key {
			has[i] = true
		}
	}

	for _, idx := range StorageIndexes {
		_, ok := has[idx]
		if !ok {
			return false
		}
	}
	return true
}

// Insert persists object
func (d *DeploymentsStorage) Insert(ctx context.Context, deployment *deployments.Deployment) error {

	if deployment == nil {
		return ErrDeploymentStorageInvalidDeployment
	}

	if err := deployment.Validate(); err != nil {
		return err
	}

	session := d.session.Copy()
	defer session.Close()

	if err := d.ensureIndexing(ctx, session); err != nil {
		return err
	}

	if err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).Insert(deployment); err != nil {
		return err
	}
	return nil
}

// Delete removed entry by ID
// Noop on ID not found
func (d *DeploymentsStorage) Delete(ctx context.Context, id string) error {

	if govalidator.IsNull(id) {
		return ErrStorageInvalidID
	}

	session := d.session.Copy()
	defer session.Close()

	if err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).RemoveId(id); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil
		}
		return err
	}

	return nil
}

func (d *DeploymentsStorage) FindByID(ctx context.Context, id string) (*deployments.Deployment, error) {

	if govalidator.IsNull(id) {
		return nil, ErrStorageInvalidID
	}

	session := d.session.Copy()
	defer session.Close()

	var deployment *deployments.Deployment
	if err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).FindId(id).One(&deployment); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return deployment, nil
}

func (d *DeploymentsStorage) FindUnfinishedByID(ctx context.Context,
	id string) (*deployments.Deployment, error) {

	if govalidator.IsNull(id) {
		return nil, ErrStorageInvalidID
	}

	session := d.session.Copy()
	defer session.Close()

	var deployment *deployments.Deployment
	filter := bson.M{
		"_id": id,
		StorageKeyDeploymentFinished: time.Time{},
	}
	if err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).Find(filter).One(&deployment); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return deployment, nil
}

func (d *DeploymentsStorage) UpdateStatsAndFinishDeployment(ctx context.Context,
	id string, stats deployments.Stats) error {

	if govalidator.IsNull(id) {
		return ErrStorageInvalidID
	}

	session := d.session.Copy()
	defer session.Close()
	deployment := deployments.NewDeployment()
	deployment.Stats = stats
	var update bson.M
	if deployment.IsFinished() {
		now := time.Now()

		update = bson.M{
			"$set": bson.M{
				StorageKeyDeploymentStats:    stats,
				StorageKeyDeploymentFinished: &now,
			},
		}
	} else {
		update = bson.M{
			"$set": bson.M{
				StorageKeyDeploymentStats: stats,
			},
		}
	}

	err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).UpdateId(id, update)
	if err == mgo.ErrNotFound {
		return ErrStorageInvalidID
	}

	return err
}

func (d *DeploymentsStorage) UpdateStats(ctx context.Context, id string,
	state_from, state_to string) error {

	if govalidator.IsNull(id) {
		return ErrStorageInvalidID
	}

	if govalidator.IsNull(state_from) {
		return ErrStorageInvalidInput
	}

	if govalidator.IsNull(state_to) {
		return ErrStorageInvalidInput
	}

	// does not need any extra operations
	// following query won't handle this case well and increase the state_to value
	if state_from == state_to {
		return nil
	}

	session := d.session.Copy()
	defer session.Close()

	// note dot notation on embedded document
	update := bson.M{
		"$inc": bson.M{
			"stats." + state_from: -1,
			"stats." + state_to:   1,
		},
	}

	err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).UpdateId(id, update)

	if err == mgo.ErrNotFound {
		return ErrStorageInvalidID
	}

	return err
}

func buildStatusKey(status string) string {
	return StorageKeyDeploymentStats + "." + status
}

func buildStatusQuery(status deployments.StatusQuery) bson.M {

	gt0 := bson.M{"$gt": 0}
	eq0 := bson.M{"$eq": 0}

	// empty query, catches StatusQueryAny
	stq := bson.M{}

	switch status {
	case deployments.StatusQueryInProgress:
		{
			// downloading, installing or rebooting are non 0, or
			// already-installed/success/failure/noimage >0 and pending > 0
			stq = bson.M{
				"$or": []bson.M{
					{
						buildStatusKey(deployments.DeviceDeploymentStatusDownloading): gt0,
					},
					{
						buildStatusKey(deployments.DeviceDeploymentStatusInstalling): gt0,
					},
					{
						buildStatusKey(deployments.DeviceDeploymentStatusRebooting): gt0,
					},
					{
						"$and": []bson.M{
							{
								buildStatusKey(deployments.DeviceDeploymentStatusPending): gt0,
							},
							{
								"$or": []bson.M{
									{
										buildStatusKey(deployments.DeviceDeploymentStatusAlreadyInst): gt0,
									},
									{
										buildStatusKey(deployments.DeviceDeploymentStatusSuccess): gt0,
									},
									{
										buildStatusKey(deployments.DeviceDeploymentStatusFailure): gt0,
									},
									{
										buildStatusKey(deployments.DeviceDeploymentStatusNoArtifact): gt0,
									},
								},
							},
						},
					},
				},
			}
		}

	case deployments.StatusQueryPending:
		{
			// all status counters, except for pending, are 0
			stq = bson.M{
				"$and": []bson.M{
					{
						buildStatusKey(deployments.DeviceDeploymentStatusDownloading): eq0,
					},
					{
						buildStatusKey(deployments.DeviceDeploymentStatusInstalling): eq0,
					},
					{
						buildStatusKey(deployments.DeviceDeploymentStatusRebooting): eq0,
					},
					{
						buildStatusKey(deployments.DeviceDeploymentStatusSuccess): eq0,
					},
					{
						buildStatusKey(deployments.DeviceDeploymentStatusAlreadyInst): eq0,
					},
					{
						buildStatusKey(deployments.DeviceDeploymentStatusAborted): eq0,
					},
					{
						buildStatusKey(deployments.DeviceDeploymentStatusDecommissioned): eq0,
					},
					{
						buildStatusKey(deployments.DeviceDeploymentStatusFailure): eq0,
					},
					{
						buildStatusKey(deployments.DeviceDeploymentStatusNoArtifact): eq0,
					},
					{
						buildStatusKey(deployments.DeviceDeploymentStatusPending): gt0,
					},
				},
			}
		}
	case deployments.StatusQueryFinished:
		{
			// downloading, installing, rebooting, pending status counters are 0
			stq = bson.M{
				"$and": []bson.M{
					{
						buildStatusKey(deployments.DeviceDeploymentStatusDownloading): eq0,
					},
					{
						buildStatusKey(deployments.DeviceDeploymentStatusInstalling): eq0,
					},
					{
						buildStatusKey(deployments.DeviceDeploymentStatusRebooting): eq0,
					},
					{
						buildStatusKey(deployments.DeviceDeploymentStatusPending): eq0,
					},
				},
			}
		}
	}

	return stq
}

func (d *DeploymentsStorage) Find(ctx context.Context,
	match deployments.Query) ([]*deployments.Deployment, error) {

	session := d.session.Copy()
	defer session.Close()

	andq := []bson.M{}

	// build deployment by name part of the query
	if match.SearchText != "" {
		// we must have indexing for text search
		if !d.hasIndexing(ctx, session) {
			return nil, ErrDeploymentStorageCannotExecQuery
		}

		tq := bson.M{
			"$text": bson.M{
				"$search": match.SearchText,
			},
		}

		andq = append(andq, tq)
	}

	// build deployment by status part of the query
	if match.Status != deployments.StatusQueryAny {
		stq := buildStatusQuery(match.Status)
		andq = append(andq, stq)
	}

	query := bson.M{}
	if len(andq) != 0 {
		// use search criteria if any
		query = bson.M{
			"$and": andq,
		}
	}
	var deployment []*deployments.Deployment
	err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).
		Find(&query).Skip(match.Skip).
		Limit(match.Limit).Sort("_id").
		All(&deployment)
	if err != nil {
		return nil, err
	}

	return deployment, nil
}

func (d *DeploymentsStorage) Finish(ctx context.Context, id string, when time.Time) error {
	if govalidator.IsNull(id) {
		return ErrStorageInvalidID
	}

	session := d.session.Copy()
	defer session.Close()

	// note dot notation on embedded document
	update := bson.M{
		"$set": bson.M{
			StorageKeyDeploymentFinished: &when,
		},
	}

	err := session.DB(store.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).UpdateId(id, update)

	if err == mgo.ErrNotFound {
		return ErrStorageInvalidID
	}

	return err
}
