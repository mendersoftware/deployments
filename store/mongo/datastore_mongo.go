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
	"crypto/tls"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/mendersoftware/go-lib-micro/config"
	mstore "github.com/mendersoftware/go-lib-micro/store"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	mopts "go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	dconfig "github.com/mendersoftware/deployments/config"
	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/utils/mgoutils"
)

const (
	DatabaseName                   = "deployment_service"
	CollectionLimits               = "limits"
	CollectionImages               = "images"
	CollectionDeployments          = "deployments"
	CollectionDeviceDeploymentLogs = "devices.logs"
	CollectionDevices              = "devices"
)

var (
	// Indexes (version: 1.2.2)
	IndexUniqueNameAndDeviceTypeName          = "uniqueNameAndDeviceTypeIndex"
	IndexDeploymentArtifactName               = "deploymentArtifactNameIndex"
	IndexDeploymentDeviceStatusesName         = "deviceIdWithStatusByCreated"
	IndexDeploymentDeviceIdStatusName         = "devicesIdWithStatus"
	IndexDeploymentDeviceDeploymentIdName     = "devicesDeploymentId"
	IndexDeploymentStatusFinishedName         = "deploymentStatusFinished"
	IndexDeploymentStatusPendingName          = "deploymentStatusPending"
	IndexDeploymentCreatedName                = "deploymentCreated"
	IndexDeploymentDeviceStatusRebootingName  = "deploymentsDeviceStatusRebooting"
	IndexDeploymentDeviceStatusPendingName    = "deploymentsDeviceStatusPending"
	IndexDeploymentDeviceStatusInstallingName = "deploymentsDeviceStatusInstalling"
	IndexDeploymentDeviceStatusFinishedName   = "deploymentsFinished"

	// Indexes (version: 1.2.3)
	IndexArtifactNameDependsName = "artifactNameDepends"
	IndexNameAndDeviceTypeName   = "artifactNameAndDeviceTypeIndex"

	_false         = false
	_true          = true
	StorageIndexes = mongo.IndexModel{
		// NOTE: Keys should be bson.D as element
		//       order matters!
		Keys: bson.D{
			{Key: StorageKeyDeploymentName,
				Value: "text"},
			{Key: StorageKeyDeploymentArtifactName,
				Value: "text"},
		},
		Options: &mopts.IndexOptions{
			Background: &_false,
			Name:       &IndexDeploymentArtifactName,
		},
	}
	StatusIndexes = mongo.IndexModel{
		Keys: bson.D{
			{Key: StorageKeyDeviceDeploymentDeviceId,
				Value: 1},
			{Key: StorageKeyDeviceDeploymentStatus,
				Value: 1},
			{Key: StorageKeyDeploymentStatsCreated,
				Value: 1},
		},
		Options: &mopts.IndexOptions{
			Background: &_false,
			Name:       &IndexDeploymentDeviceStatusesName,
		},
	}
	DeviceIDStatusIndexes = mongo.IndexModel{
		Keys: bson.D{
			{Key: "deviceID", Value: 1},
			{Key: "status", Value: 1},
		},
		Options: &mopts.IndexOptions{
			Background: &_false,
			Name:       &IndexDeploymentDeviceIdStatusName,
		},
	}
	DeploymentIdIndexes = mongo.IndexModel{
		Keys: bson.D{
			{Key: "deploymentid", Value: 1},
		},
		Options: &mopts.IndexOptions{
			Background: &_false,
			Name:       &IndexDeploymentDeviceDeploymentIdName,
		},
	}
	DeploymentStatusFinishedIndex = mongo.IndexModel{
		Keys: bson.D{
			{Key: "stats.downloading", Value: 1},
			{Key: "stats.installing", Value: 1},
			{Key: "stats.pending", Value: 1},
			{Key: "stats.rebooting", Value: 1},
			{Key: "created", Value: -1},
		},
		Options: &mopts.IndexOptions{
			Background: &_false,
			Name:       &IndexDeploymentStatusFinishedName,
		},
	}
	DeploymentStatusPendingIndex = mongo.IndexModel{
		Keys: bson.D{
			{Key: "stats.aborted", Value: 1},
			{Key: "stats.already-installed", Value: 1},
			{Key: "stats.decommissioned", Value: 1},
			{Key: "stats.downloading", Value: 1},
			{Key: "stats.failure", Value: 1},
			{Key: "stats.installing", Value: 1},
			{Key: "stats.noartifact", Value: 1},
			{Key: "stats.rebooting", Value: 1},
			{Key: "stats.success", Value: 1},
			{Key: "created", Value: -1},
		},
		Options: &mopts.IndexOptions{
			Background: &_false,
			Name:       &IndexDeploymentStatusPendingName,
		},
	}
	DeploymentCreatedIndex = mongo.IndexModel{
		Keys: bson.D{
			{Key: "created", Value: -1},
		},
		Options: &mopts.IndexOptions{
			Background: &_false,
			Name:       &IndexDeploymentCreatedName,
		},
	}
	DeploymentDeviceStatusRebootingIndex = mongo.IndexModel{
		Keys: bson.D{
			{Key: "stats.rebooting", Value: 1},
		},
		Options: &mopts.IndexOptions{
			Background: &_false,
			Name:       &IndexDeploymentDeviceStatusRebootingName,
		},
	}
	DeploymentDeviceStatusPendingIndex = mongo.IndexModel{
		Keys: bson.D{
			{Key: "stats.pending", Value: 1},
		},
		Options: &mopts.IndexOptions{
			Background: &_false,
			Name:       &IndexDeploymentDeviceStatusPendingName,
		},
	}
	DeploymentDeviceStatusInstallingIndex = mongo.IndexModel{
		Keys: bson.D{
			{Key: "stats.installing", Value: 1},
		},
		Options: &mopts.IndexOptions{
			Background: &_false,
			Name:       &IndexDeploymentDeviceStatusInstallingName,
		},
	}
	DeploymentDeviceStatusFinishedIndex = mongo.IndexModel{
		Keys: bson.D{
			{Key: "finished", Value: 1},
		},
		Options: &mopts.IndexOptions{
			Background: &_false,
			Name:       &IndexDeploymentDeviceStatusFinishedName,
		},
	}
	UniqueNameVersionIndex = mongo.IndexModel{
		Keys: bson.D{
			{Key: StorageKeyImageName,
				Value: 1},
			{Key: StorageKeyImageDeviceTypes,
				Value: 1},
		},
		Options: &mopts.IndexOptions{
			Background: &_false,
			Name:       &IndexUniqueNameAndDeviceTypeName,
			Unique:     &_true,
		},
	}

	// 1.2.3
	IndexArtifactNameDepends = mongo.IndexModel{
		Keys: bson.D{
			{Key: StorageKeyImageName,
				Value: 1},
			{Key: StorageKeyImageDependsIdx,
				Value: 1},
		},
		Options: &mopts.IndexOptions{
			Background: &_false,
			Name:       &IndexArtifactNameDependsName,
			Unique:     &_true,
		},
	}
)

// Errors
var (
	ErrImagesStorageInvalidID           = errors.New("Invalid id")
	ErrImagesStorageInvalidArtifactName = errors.New("Invalid artifact name")
	ErrImagesStorageInvalidName         = errors.New("Invalid name")
	ErrImagesStorageInvalidDeviceType   = errors.New("Invalid device type")
	ErrImagesStorageInvalidImage        = errors.New("Invalid image")

	ErrStorageInvalidDeviceDeployment = errors.New("Invalid device deployment")

	ErrDeploymentStorageInvalidDeployment = errors.New("Invalid deployment")
	ErrStorageInvalidID                   = errors.New("Invalid id")
	ErrStorageNotFound                    = errors.New("Not found")
	ErrDeploymentStorageInvalidQuery      = errors.New("Invalid query")
	ErrDeploymentStorageCannotExecQuery   = errors.New("Cannot execute query")
	ErrStorageInvalidInput                = errors.New("invalid input")

	ErrLimitNotFound = errors.New("limit not found")
)

// Database keys
const (
	// Need to be kept in sync with structure filed names
	StorageKeyImageId          = "_id"
	StorageKeyImageDepends     = "meta_artifact.depends"
	StorageKeyImageDependsIdx  = "meta_artifact.depends_idx"
	StorageKeyImageSize        = "size"
	StorageKeyImageDeviceTypes = "meta_artifact.device_types_compatible"
	StorageKeyImageName        = "meta_artifact.name"

	StorageKeyDeviceDeploymentLogMessages = "messages"

	StorageKeyDeviceDeploymentAssignedImage   = "image"
	StorageKeyDeviceDeploymentAssignedImageId = StorageKeyDeviceDeploymentAssignedImage + "." + StorageKeyImageId
	StorageKeyDeviceDeploymentDeviceId        = "deviceid"
	StorageKeyDeviceDeploymentStatus          = "status"
	StorageKeyDeviceDeploymentSubState        = "substate"
	StorageKeyDeviceDeploymentDeploymentID    = "deploymentid"
	StorageKeyDeviceDeploymentFinished        = "finished"
	StorageKeyDeviceDeploymentIsLogAvailable  = "log"
	StorageKeyDeviceDeploymentArtifact        = "image"

	StorageKeyDeploymentName         = "deploymentconstructor.name"
	StorageKeyDeploymentArtifactName = "deploymentconstructor.artifactname"
	StorageKeyDeploymentStats        = "stats"
	StorageKeyDeploymentStatsCreated = "created"
	StorageKeyDeploymentFinished     = "finished"
	StorageKeyDeploymentArtifacts    = "artifacts"

	ArtifactDependsDeviceType = "device_types"
)

type DataStoreMongo struct {
	client *mongo.Client
}

func NewDataStoreMongoWithClient(client *mongo.Client) *DataStoreMongo {
	return &DataStoreMongo{
		client: client,
	}
}

func NewMongoClient(ctx context.Context, c config.Reader) (*mongo.Client, error) {

	clientOptions := mopts.Client()
	mongoURL := c.GetString(dconfig.SettingMongo)
	if !strings.Contains(mongoURL, "://") {
		return nil, errors.Errorf("Invalid mongoURL %q: missing schema.",
			mongoURL)
	}
	clientOptions.ApplyURI(mongoURL)

	username := c.GetString(dconfig.SettingDbUsername)
	if username != "" {
		credentials := mopts.Credential{
			Username: c.GetString(dconfig.SettingDbUsername),
		}
		password := c.GetString(dconfig.SettingDbPassword)
		if password != "" {
			credentials.Password = password
			credentials.PasswordSet = true
		}
		clientOptions.SetAuth(credentials)
	}

	if c.GetBool(dconfig.SettingDbSSL) {
		tlsConfig := &tls.Config{}
		tlsConfig.InsecureSkipVerify = c.GetBool(dconfig.SettingDbSSLSkipVerify)
		clientOptions.SetTLSConfig(tlsConfig)
	}

	// Set writeconcern to acknowlage after write has propagated to the
	// mongod instance and commited to the file system journal.
	var wc *writeconcern.WriteConcern
	wc.WithOptions(writeconcern.W(1), writeconcern.J(true))
	clientOptions.SetWriteConcern(wc)

	if clientOptions.ReplicaSet != nil {
		clientOptions.SetReadConcern(readconcern.Linearizable())
	}

	// Set 10s timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to connect to mongo server")
	}

	// Validate connection
	if err = client.Ping(ctx, nil); err != nil {
		return nil, errors.Wrap(err, "Error reaching mongo server")
	}

	return client, nil
}

func (db *DataStoreMongo) GetReleases(ctx context.Context, filt *model.ReleaseFilter) ([]model.Release, error) {
	var pipe []bson.D

	match := db.matchFromFilt(filt)

	project := bson.D{
		// Remove (possibly expensive) sub-document from pipeline
		{Key: "$project", Value: bson.M{StorageKeyImageDependsIdx: 0}},
	}

	group := bson.D{
		{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$" + StorageKeyImageName},
			{Key: "name", Value: bson.M{
				"$first": "$" + StorageKeyImageName}},
			{Key: "artifacts", Value: bson.M{"$push": "$$ROOT"}}},
		},
	}

	sort := bson.D{
		{Key: "$sort", Value: bson.M{
			"name": -1}},
	}

	if match != nil {
		pipe = []bson.D{
			match,
			project,
			group,
			sort,
		}
	} else {
		pipe = []bson.D{
			project,
			group,
			sort,
		}
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collImg := database.Collection(CollectionImages)

	results := []model.Release{}
	cursor, err := collImg.Aggregate(ctx, pipe)
	if err != nil {
		return nil, err
	}
	// NOTE: Call to cursor.All will automatically close cursor
	if err = cursor.All(ctx, &results); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return results, nil
}

func (db *DataStoreMongo) matchFromFilt(f *model.ReleaseFilter) bson.D {
	if f == nil {
		return nil
	}

	return bson.D{
		{Key: "$match", Value: bson.M{
			StorageKeyImageName: f.Name}},
	}
}

// limits
//
func (db *DataStoreMongo) GetLimit(ctx context.Context, name string) (*model.Limit, error) {

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collLim := database.Collection(CollectionLimits)

	limit := new(model.Limit)
	if err := collLim.FindOne(ctx, bson.M{"_id": name}).
		Decode(limit); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrLimitNotFound
		}
		return nil, err
	}

	return limit, nil
}

func (db *DataStoreMongo) ProvisionTenant(ctx context.Context, tenantId string) error {

	dbname := mstore.DbNameForTenant(tenantId, DbName)

	return MigrateSingle(ctx, dbname, DbVersion, db.client, true)
}

//images

// Ensure required indexes exists; create if not.
func (db *DataStoreMongo) ensureIndexing(ctx context.Context, client *mongo.Client) error {

	// Build index upfront - make sure this index is always on.
	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collImg := database.Collection(CollectionImages)
	indexes := collImg.Indexes()
	// NOTE: CreateIndex (CreateOne) doesn't create duplicates for mongodb
	//       version > 3.0, db.collection.ensureIndexing is an alias for
	//       db.collection.createIndex
	_, err := indexes.CreateOne(ctx, UniqueNameVersionIndex)

	return err
}

// Exists checks if object with ID exists
func (db *DataStoreMongo) Exists(ctx context.Context, id string) (bool, error) {
	var result interface{}

	if govalidator.IsNull(id) {
		return false, ErrImagesStorageInvalidID
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collImg := database.Collection(CollectionImages)

	if err := collImg.FindOne(ctx, bson.M{"_id": id}).
		Decode(&result); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Update provided Image
// Return false if not found
func (db *DataStoreMongo) Update(ctx context.Context,
	image *model.Image) (bool, error) {

	if err := image.Validate(); err != nil {
		return false, err
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collImg := database.Collection(CollectionImages)

	image.SetModified(time.Now())
	var result model.Image
	if err := collImg.FindOneAndUpdate(
		ctx, bson.M{"_id": image.Id}, image).
		Decode(&result); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// ImageByNameAndDeviceType finds image with specified application name and target device type
func (db *DataStoreMongo) ImageByNameAndDeviceType(ctx context.Context,
	name, deviceType string) (*model.Image, error) {

	if govalidator.IsNull(name) {
		return nil, ErrImagesStorageInvalidArtifactName
	}

	if govalidator.IsNull(deviceType) {
		return nil, ErrImagesStorageInvalidDeviceType
	}

	// equal to device type & software version (application name + version)
	query := bson.M{
		StorageKeyImageName:        name,
		StorageKeyImageDeviceTypes: deviceType,
	}

	// If multiple entries matches, pick the smallest one.
	findOpts := mopts.FindOne()
	findOpts.SetSort(bson.M{StorageKeyImageSize: 1})

	dbName := mstore.DbFromContext(ctx, DatabaseName)
	database := db.client.Database(dbName)
	collImg := database.Collection(CollectionImages)

	// Both we lookup unique object, should be one or none.
	var image model.Image
	if err := collImg.FindOne(ctx, query, findOpts).
		Decode(&image); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &image, nil
}

// ImageByIdsAndDeviceType finds image with id from ids and target device type
func (db *DataStoreMongo) ImageByIdsAndDeviceType(ctx context.Context,
	ids []string, deviceType string) (*model.Image, error) {

	if govalidator.IsNull(deviceType) {
		return nil, ErrImagesStorageInvalidDeviceType
	}

	if len(ids) == 0 {
		return nil, ErrImagesStorageInvalidID
	}

	query := bson.D{
		{Key: StorageKeyImageId, Value: bson.M{"$in": ids}},
		{Key: StorageKeyImageDeviceTypes, Value: deviceType},
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collImg := database.Collection(CollectionImages)

	// If multiple entries matches, pick the smallest one
	findOpts := mopts.FindOne()
	findOpts.SetSort(bson.M{StorageKeyImageSize: 1})

	// Both we lookup unique object, should be one or none.
	var image model.Image
	if err := collImg.FindOne(ctx, query, findOpts).
		Decode(&image); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &image, nil
}

// ImagesByName finds images with specified artifact name
func (db *DataStoreMongo) ImagesByName(
	ctx context.Context, name string) ([]*model.Image, error) {

	var images []*model.Image

	if govalidator.IsNull(name) {
		return nil, ErrImagesStorageInvalidName
	}

	// equal to artifact name
	query := bson.M{
		StorageKeyImageName: name,
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collImg := database.Collection(CollectionImages)
	cursor, err := collImg.Find(ctx, query)
	if err != nil {
		return nil, err
	}
	// Both we lookup unique object, should be one or none.
	if cursor.All(ctx, &images); err != nil {
		return nil, err
	}

	return images, nil
}

// Insert persists object
func (db *DataStoreMongo) InsertImage(ctx context.Context, image *model.Image) error {

	if image == nil {
		return ErrImagesStorageInvalidImage
	}

	if err := image.Validate(); err != nil {
		return err
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collImg := database.Collection(CollectionImages)

	_, err := collImg.InsertOne(ctx, image)
	if err != nil {
		if except, ok := err.(mongo.WriteException); ok {
			e := mgoutils.NewIndexError(except)
			if e == nil {
				return err
			}
			// Provide keys in a more readable format
			e.IndexConflict["artifact_name"] = e.
				IndexConflict[StorageKeyImageName]
			delete(e.IndexConflict, StorageKeyImageName)
			e.IndexConflict["depends"] = e.
				IndexConflict[StorageKeyImageDependsIdx]
			delete(e.IndexConflict, StorageKeyImageDependsIdx)
			return e
		}
	}

	return nil
}

// FindImageByID search storage for image with ID, returns nil if not found
func (db *DataStoreMongo) FindImageByID(ctx context.Context,
	id string) (*model.Image, error) {

	if govalidator.IsNull(id) {
		return nil, ErrImagesStorageInvalidID
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collImg := database.Collection(CollectionImages)

	var image model.Image
	if err := collImg.FindOne(ctx, bson.M{"_id": id}).
		Decode(&image); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &image, nil
}

// IsArtifactUnique checks if there is no artifact with the same artifactName
// supporting one of the device types from deviceTypesCompatible list.
// Returns true, nil if artifact is unique;
// false, nil if artifact is not unique;
// false, error in case of error.
func (db *DataStoreMongo) IsArtifactUnique(ctx context.Context,
	artifactName string, deviceTypesCompatible []string) (bool, error) {

	if govalidator.IsNull(artifactName) {
		return false, ErrImagesStorageInvalidArtifactName
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collImg := database.Collection(CollectionImages)

	query := bson.M{
		"$and": []bson.M{
			bson.M{
				StorageKeyImageName: artifactName,
			},
			{
				StorageKeyImageDeviceTypes: bson.M{
					"$in": deviceTypesCompatible},
			},
		},
	}

	// do part of the job manually
	// if candidate images have any extra 'depends' - guaranteed non-overlap
	// otherwise it's a match
	cur, err := collImg.Find(ctx, query)
	if err != nil {
		return false, err
	}

	var images []model.Image
	err = cur.All(ctx, &images)
	if err != nil {
		return false, err
	}

	for _, i := range images {
		// the artifact already has same name and overlapping dev type
		// if there are no more depends than dev type - it's not unique
		if len(i.ArtifactMeta.Depends) == 1 {
			if _, ok := i.ArtifactMeta.Depends["device_type"]; ok {
				return false, nil
			}
		} else if len(i.ArtifactMeta.Depends) == 0 {
			return false, nil
		}
		return false, err
	}

	return false, nil
}

// Delete image specified by ID
// Noop on if not found.
func (db *DataStoreMongo) DeleteImage(ctx context.Context, id string) error {

	if govalidator.IsNull(id) {
		return ErrImagesStorageInvalidID
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collImg := database.Collection(CollectionImages)

	if res, err := collImg.DeleteOne(ctx, bson.M{"_id": id}); err != nil {
		if res.DeletedCount == 0 {
			return nil
		}
		return err
	}

	return nil
}

// FindAll lists all images
func (db *DataStoreMongo) FindAll(ctx context.Context) ([]*model.Image, error) {

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collImg := database.Collection(CollectionImages)
	cursor, err := collImg.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}

	// NOTE: cursor.All closes the cursor before returning
	var images []*model.Image
	if err := cursor.All(ctx, &images); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return images, nil
}

// device deployment log
func (db *DataStoreMongo) SaveDeviceDeploymentLog(ctx context.Context,
	log model.DeploymentLog) error {

	if err := log.Validate(); err != nil {
		return err
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collLogs := database.Collection(CollectionDeviceDeploymentLogs)

	query := bson.D{
		{Key: StorageKeyDeviceDeploymentDeviceId,
			Value: log.DeviceID},
		{Key: StorageKeyDeviceDeploymentDeploymentID,
			Value: log.DeploymentID},
	}

	// update log messages
	// if the deployment log is already present than messages will be overwritten
	update := bson.D{
		{Key: "$set", Value: bson.M{
			StorageKeyDeviceDeploymentLogMessages: log.Messages,
		}},
	}
	updateOptions := mopts.Update()
	updateOptions.SetUpsert(true)
	if _, err := collLogs.UpdateOne(
		ctx, query, update, updateOptions); err != nil {
		return err
	}

	return nil
}

func (db *DataStoreMongo) GetDeviceDeploymentLog(ctx context.Context,
	deviceID, deploymentID string) (*model.DeploymentLog, error) {

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collLogs := database.Collection(CollectionDeviceDeploymentLogs)

	query := bson.M{
		StorageKeyDeviceDeploymentDeviceId:     deviceID,
		StorageKeyDeviceDeploymentDeploymentID: deploymentID,
	}

	var depl model.DeploymentLog
	if err := collLogs.FindOne(ctx, query).Decode(&depl); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &depl, nil
}

// device deployments

// InsertMany stores multiple device deployment objects.
// TODO: Handle error cleanup, multi insert is not atomic, loop into two-phase commits
func (db *DataStoreMongo) InsertMany(ctx context.Context,
	deployments ...*model.DeviceDeployment) error {

	if len(deployments) == 0 {
		return nil
	}

	// Writing to another interface list addresses golang gatcha interface{} == []interface{}
	var list []interface{}
	for _, deployment := range deployments {

		if deployment == nil {
			return ErrStorageInvalidDeviceDeployment
		}

		if err := deployment.Validate(); err != nil {
			return errors.Wrap(err, "Validating device deployment")
		}

		list = append(list, deployment)
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDevs := database.Collection(CollectionDevices)

	if _, err := collDevs.InsertMany(ctx, list); err != nil {
		return err
	}

	return nil
}

// ExistAssignedImageWithIDAndStatuses checks if image is used by deployment with specified status.
func (db *DataStoreMongo) ExistAssignedImageWithIDAndStatuses(ctx context.Context,
	imageID string, statuses ...string) (bool, error) {

	// Verify ID formatting
	if govalidator.IsNull(imageID) {
		return false, ErrStorageInvalidID
	}

	query := bson.M{StorageKeyDeviceDeploymentAssignedImageId: imageID}

	if len(statuses) > 0 {
		query[StorageKeyDeviceDeploymentStatus] = bson.M{
			"$in": statuses,
		}
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDevs := database.Collection(CollectionDevices)

	// if found at least one then image in active deployment
	var tmp interface{}
	if err := collDevs.FindOne(ctx, query).Decode(&tmp); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// FindOldestDeploymentForDeviceIDWithStatuses find oldest deployment matching device id and one of specified statuses.
func (db *DataStoreMongo) FindOldestDeploymentForDeviceIDWithStatuses(ctx context.Context,
	deviceID string, statuses ...string) (*model.DeviceDeployment, error) {

	// Verify ID formatting
	if govalidator.IsNull(deviceID) {
		return nil, ErrStorageInvalidID
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDevs := database.Collection(CollectionDevices)

	// Device should know only about deployments that are not finished
	query := bson.D{
		{Key: StorageKeyDeviceDeploymentDeviceId,
			Value: deviceID},
		{Key: StorageKeyDeviceDeploymentStatus,
			Value: bson.M{"$in": statuses}},
	}

	// Find the oldest one by sorting the creation timestamp
	// in ascending order.
	findOptions := mopts.FindOne()
	findOptions.SetSort(bson.M{"created": 1})

	// Select only the oldest one that have not been finished yet.
	var deployment *model.DeviceDeployment
	if err := collDevs.FindOne(ctx, query, findOptions).
		Decode(&deployment); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return deployment, nil
}

// FindAllDeploymentsForDeviceIDWithStatuses finds all deployments matching device id and one of specified statuses.
func (db *DataStoreMongo) FindAllDeploymentsForDeviceIDWithStatuses(ctx context.Context,
	deviceID string, statuses ...string) ([]model.DeviceDeployment, error) {

	// Verify ID formatting
	if govalidator.IsNull(deviceID) {
		return nil, ErrStorageInvalidID
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDevs := database.Collection(CollectionDevices)

	// Device should know only about deployments that are not finished
	query := bson.D{
		{Key: StorageKeyDeviceDeploymentDeviceId,
			Value: deviceID},
		{Key: StorageKeyDeviceDeploymentStatus,
			Value: bson.M{
				"$in": statuses,
			}},
	}

	var deployments []model.DeviceDeployment
	if cursor, err := collDevs.Find(ctx, query); err != nil {
		return nil, err
	} else {
		if err = cursor.All(ctx, &deployments); err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, nil
			}
		}
	}

	return deployments, nil
}

func (db *DataStoreMongo) UpdateDeviceDeploymentStatus(ctx context.Context,
	deviceID string, deploymentID string, ddStatus model.DeviceDeploymentStatus) (string, error) {

	// Verify ID formatting
	if govalidator.IsNull(deviceID) ||
		govalidator.IsNull(deploymentID) {
		return "", ErrStorageInvalidID
	}

	if ok, _ := govalidator.ValidateStruct(ddStatus); !ok {
		return "", ErrStorageInvalidInput
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDevs := database.Collection(CollectionDevices)

	// Device should know only about deployments that are not finished
	query := bson.D{
		{Key: StorageKeyDeviceDeploymentDeviceId, Value: deviceID},
		{Key: StorageKeyDeviceDeploymentDeploymentID, Value: deploymentID},
	}

	// update status field
	set := bson.M{
		StorageKeyDeviceDeploymentStatus: ddStatus.Status,
	}
	// and finish time if provided
	if ddStatus.FinishTime != nil {
		set[StorageKeyDeviceDeploymentFinished] = ddStatus.FinishTime
	}

	if ddStatus.SubState != nil {
		set[StorageKeyDeviceDeploymentSubState] = *ddStatus.SubState
	}

	update := bson.D{
		{Key: "$set", Value: set},
	}

	var old model.DeviceDeployment

	if err := collDevs.FindOneAndUpdate(ctx, query, update).
		Decode(&old); err != nil {
		if err == mongo.ErrNoDocuments {
			return "", ErrStorageNotFound
		}
		return "", err

	}

	return *old.Status, nil
}

func (db *DataStoreMongo) UpdateDeviceDeploymentLogAvailability(ctx context.Context,
	deviceID string, deploymentID string, log bool) error {

	// Verify ID formatting
	if govalidator.IsNull(deviceID) ||
		govalidator.IsNull(deploymentID) {
		return ErrStorageInvalidID
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDevs := database.Collection(CollectionDevices)

	selector := bson.D{
		{Key: StorageKeyDeviceDeploymentDeviceId,
			Value: deviceID},
		{Key: StorageKeyDeviceDeploymentDeploymentID,
			Value: deploymentID},
	}

	update := bson.D{
		{Key: "$set", Value: bson.M{
			StorageKeyDeviceDeploymentIsLogAvailable: log}},
	}

	// NOTE <Review> Perhaps this should be UpdateOne ?
	if res, err := collDevs.UpdateMany(ctx, selector, update); err != nil {
		return err
	} else if res.MatchedCount == 0 {
		return ErrStorageNotFound
	}

	return nil
}

// AssignArtifact assigns artifact to the device deployment
func (db *DataStoreMongo) AssignArtifact(ctx context.Context,
	deviceID string, deploymentID string, artifact *model.Image) error {

	// Verify ID formatting
	if govalidator.IsNull(deviceID) ||
		govalidator.IsNull(deploymentID) {
		return ErrStorageInvalidID
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDevs := database.Collection(CollectionDevices)

	selector := bson.D{
		{Key: StorageKeyDeviceDeploymentDeviceId,
			Value: deviceID},
		{Key: StorageKeyDeviceDeploymentDeploymentID,
			Value: deploymentID},
	}

	update := bson.D{
		{Key: "$set", Value: bson.M{
			StorageKeyDeviceDeploymentArtifact: artifact}},
	}

	// NOTE <Review> Perhaps this should be UpdateOne ?
	if res, err := collDevs.UpdateMany(ctx, selector, update); err != nil {
		return err
	} else if res.MatchedCount == 0 {
		return ErrStorageNotFound
	}

	return nil
}

func (db *DataStoreMongo) AggregateDeviceDeploymentByStatus(ctx context.Context,
	id string) (model.Stats, error) {

	if govalidator.IsNull(id) {
		return nil, ErrStorageInvalidID
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDevs := database.Collection(CollectionDevices)

	match := bson.D{
		{Key: "$match", Value: bson.M{
			StorageKeyDeviceDeploymentDeploymentID: id}},
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
	cursor, err := collDevs.Aggregate(ctx, pipeline)
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

//GetDeviceStatusesForDeployment retrieve device deployment statuses for a given deployment.
func (db *DataStoreMongo) GetDeviceStatusesForDeployment(ctx context.Context,
	deploymentID string) ([]model.DeviceDeployment, error) {

	var statuses []model.DeviceDeployment
	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDevs := database.Collection(CollectionDevices)

	query := bson.M{
		StorageKeyDeviceDeploymentDeploymentID: deploymentID,
	}

	cursor, err := collDevs.Find(ctx, query)
	if err != nil {
		return nil, err
	}
	if err = cursor.All(ctx, &statuses); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return statuses, nil
}

// Returns true if deployment of ID `deploymentID` is assigned to device with ID
// `deviceID`, false otherwise. In case of errors returns false and an error
// that occurred
func (db *DataStoreMongo) HasDeploymentForDevice(ctx context.Context,
	deploymentID string, deviceID string) (bool, error) {

	var dep model.DeviceDeployment
	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDevs := database.Collection(CollectionDevices)

	query := bson.D{
		{Key: StorageKeyDeviceDeploymentDeploymentID,
			Value: deploymentID},
		{Key: StorageKeyDeviceDeploymentDeviceId,
			Value: deviceID},
	}

	if err := collDevs.FindOne(ctx, query).Decode(&dep); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}

func (db *DataStoreMongo) GetDeviceDeploymentStatus(ctx context.Context,
	deploymentID string, deviceID string) (string, error) {

	var dep model.DeviceDeployment
	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDevs := database.Collection(CollectionDevices)

	query := bson.M{
		StorageKeyDeviceDeploymentDeploymentID: deploymentID,
		StorageKeyDeviceDeploymentDeviceId:     deviceID,
	}

	if err := collDevs.FindOne(ctx, query).Decode(&dep); err != nil {
		if err == mongo.ErrNoDocuments {
			return "", nil
		} else {
			return "", err
		}
	}

	return *dep.Status, nil
}

func (db *DataStoreMongo) AbortDeviceDeployments(ctx context.Context,
	deploymentId string) error {

	if govalidator.IsNull(deploymentId) {
		return ErrStorageInvalidID
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDevs := database.Collection(CollectionDevices)
	selector := bson.M{
		"$and": []bson.M{
			{
				StorageKeyDeviceDeploymentDeploymentID: deploymentId,
			},
			{
				StorageKeyDeviceDeploymentStatus: bson.M{
					"$in": model.ActiveDeploymentStatuses(),
				},
			},
		},
	}

	update := bson.M{
		"$set": bson.M{
			StorageKeyDeviceDeploymentStatus: model.DeviceDeploymentStatusAborted,
		},
	}

	if res, err := collDevs.UpdateMany(ctx, selector, update); err != nil {
		return err
	} else if res.MatchedCount == 0 {
		return ErrStorageInvalidID
	}

	return nil
}

func (db *DataStoreMongo) DecommissionDeviceDeployments(ctx context.Context,
	deviceId string) error {

	if govalidator.IsNull(deviceId) {
		return ErrStorageInvalidID
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDevs := database.Collection(CollectionDevices)
	selector := bson.M{
		"$and": []bson.M{
			{
				StorageKeyDeviceDeploymentDeviceId: deviceId,
			},
			{
				StorageKeyDeviceDeploymentStatus: bson.M{
					"$in": model.ActiveDeploymentStatuses(),
				},
			},
		},
	}

	update := bson.M{
		"$set": bson.M{
			StorageKeyDeviceDeploymentStatus: model.DeviceDeploymentStatusDecommissioned,
		},
	}

	if _, err := collDevs.UpdateMany(ctx, selector, update); err != nil {
		return err
	}

	return nil
}

// deployments

func (db *DataStoreMongo) EnsureIndexes(dbName string, collName string,
	indexes ...mongo.IndexModel) error {
	ctx := context.Background()
	dataBase := db.client.Database(dbName)

	coll := dataBase.Collection(collName)
	idxView := coll.Indexes()
	_, err := idxView.CreateMany(ctx, indexes)
	return err
}

// return true if required indexing was set up
func (db *DataStoreMongo) hasIndexing(ctx context.Context, client *mongo.Client) bool {

	var idx bson.M
	database := client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDpl := database.Collection(CollectionDeployments)
	idxView := collDpl.Indexes()

	cursor, err := idxView.List(ctx)
	if err != nil {
		// check failed, assume indexing is not there
		return false
	}

	has := map[string]bool{}
	for cursor.Next(ctx) {
		if err = cursor.Decode(&idx); err != nil {
			continue
		}
		if _, ok := idx["weights"]; ok {
			// text index
			for k := range idx["weights"].(bson.M) {
				has[k] = true
			}
		} else {
			for i := range idx["key"].(bson.M) {
				has[i] = true
			}

		}
	}
	if err != nil {
		return false
	}

	for _, key := range StorageIndexes.Keys.(bson.D) {
		_, ok := has[key.Key]
		if !ok {
			return false
		}
	}

	return true
}

// Insert persists object
func (db *DataStoreMongo) InsertDeployment(ctx context.Context, deployment *model.Deployment) error {

	if deployment == nil {
		return ErrDeploymentStorageInvalidDeployment
	}

	if err := deployment.Validate(); err != nil {
		return err
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDpl := database.Collection(CollectionDeployments)

	if _, err := collDpl.InsertOne(ctx, deployment); err != nil {
		return err
	}
	return nil
}

// Delete removed entry by ID
// Noop on ID not found
func (db *DataStoreMongo) DeleteDeployment(ctx context.Context, id string) error {

	if govalidator.IsNull(id) {
		return ErrStorageInvalidID
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDpl := database.Collection(CollectionDeployments)

	if _, err := collDpl.DeleteOne(ctx, bson.M{"_id": id}); err != nil {
		return err
	}

	return nil
}

func (db *DataStoreMongo) FindDeploymentByID(ctx context.Context, id string) (*model.Deployment, error) {

	if govalidator.IsNull(id) {
		return nil, ErrStorageInvalidID
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDpl := database.Collection(CollectionDeployments)

	deployment := new(model.Deployment)
	if err := collDpl.FindOne(ctx, bson.M{"_id": id}).
		Decode(deployment); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return deployment, nil
}

func (db *DataStoreMongo) FindUnfinishedByID(ctx context.Context,
	id string) (*model.Deployment, error) {

	if govalidator.IsNull(id) {
		return nil, ErrStorageInvalidID
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDpl := database.Collection(CollectionDeployments)

	var deployment *model.Deployment
	filter := bson.D{
		{Key: "_id", Value: id},
		{Key: StorageKeyDeploymentFinished, Value: nil},
	}
	if err := collDpl.FindOne(ctx, filter).
		Decode(&deployment); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return deployment, nil
}

func (db *DataStoreMongo) DeviceCountByDeployment(ctx context.Context,
	id string) (int, error) {

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDevs := database.Collection(CollectionDevices)

	filter := bson.M{
		"deploymentid": id,
	}

	deviceCount, err := collDevs.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	return int(deviceCount), nil
}

func (db *DataStoreMongo) UpdateStatsAndFinishDeployment(ctx context.Context,
	id string, stats model.Stats) error {

	if govalidator.IsNull(id) {
		return ErrStorageInvalidID
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDpl := database.Collection(CollectionDeployments)

	deployment, err := model.NewDeployment()
	if err != nil {
		return errors.Wrap(err, "failed to create deployment")
	}

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

	res, err := collDpl.UpdateOne(ctx, bson.M{"_id": id}, update)
	if res.MatchedCount == 0 {
		return ErrStorageInvalidID
	}
	return err
}

func (db *DataStoreMongo) UpdateStats(ctx context.Context, id string,
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

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDpl := database.Collection(CollectionDeployments)

	// note dot notation on embedded document
	update := bson.M{
		"$inc": bson.M{
			"stats." + state_from: -1,
			"stats." + state_to:   1,
		},
	}

	res, err := collDpl.UpdateOne(ctx, bson.M{"_id": id}, update)

	if res.MatchedCount == 0 {
		return ErrStorageInvalidID
	}

	return err
}

func buildStatusKey(status string) string {
	return StorageKeyDeploymentStats + "." + status
}

func buildStatusQuery(status model.StatusQuery) bson.M {

	gt0 := bson.M{"$gt": 0}
	eq0 := bson.M{"$eq": 0}
	notNull := bson.M{"$ne": nil}

	// empty query, catches StatusQueryAny
	stq := bson.M{}

	switch status {
	case model.StatusQueryInProgress:
		{
			// downloading, installing or rebooting are non 0, or
			// already-installed/success/failure/noimage >0 and pending > 0
			stq = bson.M{
				"$or": []bson.M{
					{
						buildStatusKey(model.DeviceDeploymentStatusDownloading): gt0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusInstalling): gt0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusRebooting): gt0,
					},
					{
						"$and": []bson.M{
							{
								buildStatusKey(model.DeviceDeploymentStatusPending): gt0,
							},
							{
								"$or": []bson.M{
									{
										buildStatusKey(model.DeviceDeploymentStatusAlreadyInst): gt0,
									},
									{
										buildStatusKey(model.DeviceDeploymentStatusSuccess): gt0,
									},
									{
										buildStatusKey(model.DeviceDeploymentStatusFailure): gt0,
									},
									{
										buildStatusKey(model.DeviceDeploymentStatusNoArtifact): gt0,
									},
								},
							},
						},
					},
				},
			}
		}

	case model.StatusQueryPending:
		{
			// all status counters, except for pending, are 0
			stq = bson.M{
				"$and": []bson.M{
					{
						buildStatusKey(model.DeviceDeploymentStatusDownloading): eq0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusInstalling): eq0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusRebooting): eq0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusSuccess): eq0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusAlreadyInst): eq0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusAborted): eq0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusDecommissioned): eq0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusFailure): eq0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusNoArtifact): eq0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusPending): gt0,
					},
				},
			}
		}
	case model.StatusQueryFinished:
		{
			stq = bson.M{StorageKeyDeploymentFinished: notNull}
		}
	}

	return stq
}

func (db *DataStoreMongo) Find(ctx context.Context,
	match model.Query) ([]*model.Deployment, error) {

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDpl := database.Collection(CollectionDeployments)

	andq := []bson.M{}

	// build deployment by name part of the query
	if match.SearchText != "" {
		// we must have indexing for text search
		if !db.hasIndexing(ctx, db.client) {
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
	if match.Status != model.StatusQueryAny {
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

	if match.CreatedAfter != nil && match.CreatedBefore != nil {
		query["created"] = bson.M{
			"$gte": match.CreatedAfter,
			"$lte": match.CreatedBefore,
		}
	} else if match.CreatedAfter != nil {
		query["created"] = bson.M{
			"$gte": match.CreatedAfter,
		}
	} else if match.CreatedBefore != nil {
		query["created"] = bson.M{
			"$lte": match.CreatedBefore,
		}
	}

	pipeline := []bson.D{
		bson.D{
			{Key: "$match", Value: query},
		},
		bson.D{
			{Key: "$sort", Value: bson.M{"created": -1}},
		},
	}
	if match.Skip > 0 {
		pipeline = append(pipeline,
			bson.D{{Key: "$skip", Value: match.Skip}})
	}
	if match.Limit > 0 {
		pipeline = append(pipeline,
			bson.D{{Key: "$limit", Value: match.Limit}})
	}

	var deployment []*model.Deployment
	cursor, err := collDpl.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	if err := cursor.All(ctx, &deployment); err != nil {
		return nil, err
	}

	return deployment, nil
}

func (db *DataStoreMongo) Finish(ctx context.Context, id string, when time.Time) error {
	if govalidator.IsNull(id) {
		return ErrStorageInvalidID
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDpl := database.Collection(CollectionDeployments)

	// note dot notation on embedded document
	update := bson.M{
		"$set": bson.M{
			StorageKeyDeploymentFinished: &when,
		},
	}

	res, err := collDpl.UpdateOne(ctx, bson.M{"_id": id}, update)

	if res.MatchedCount == 0 {
		return ErrStorageInvalidID
	}

	return err
}

// ExistUnfinishedByArtifactId checks if there is an active deployment that uses
// given artifact
func (db *DataStoreMongo) ExistUnfinishedByArtifactId(ctx context.Context,
	id string) (bool, error) {

	if govalidator.IsNull(id) {
		return false, ErrStorageInvalidID
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDpl := database.Collection(CollectionDeployments)

	var tmp interface{}
	query := bson.D{
		{Key: StorageKeyDeploymentFinished, Value: nil},
		{Key: StorageKeyDeploymentArtifacts, Value: id},
	}
	if err := collDpl.FindOne(ctx, query).Decode(&tmp); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// ExistByArtifactId check if there is any deployment that uses give artifact
func (db *DataStoreMongo) ExistByArtifactId(ctx context.Context,
	id string) (bool, error) {

	if govalidator.IsNull(id) {
		return false, ErrStorageInvalidID
	}

	database := db.client.Database(mstore.DbFromContext(ctx, DatabaseName))
	collDpl := database.Collection(CollectionDeployments)

	var tmp interface{}
	query := bson.D{
		{Key: StorageKeyDeploymentArtifacts, Value: id},
	}
	if err := collDpl.FindOne(ctx, query).Decode(&tmp); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
