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

package s3

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/mendersoftware/deployments/resources/images"
	"github.com/mendersoftware/deployments/resources/images/model"
	"github.com/pkg/errors"
)

const (
	ExpireMaxLimit = 7 * 24 * time.Hour
	ExpireMinLimit = 1 * time.Minute
)

// SimpleStorageService - AWS S3 client.
// Data layer for file storage.
// Implements model.FileStorage interface
type SimpleStorageService struct {
	client *s3.S3
	bucket string
}

// NewSimpleStorageServiceStatic create new S3 client model.
// AWS authentication keys are automatically reloaded from env variables.
func NewSimpleStorageServiceStatic(bucket, key, secret, region, token, uri string) *SimpleStorageService {

	credentials := credentials.NewStaticCredentials(key, secret, token)
	config := aws.NewConfig().WithCredentials(credentials).WithRegion(region)

	if len(uri) > 0 {
		sslDisabled := !strings.HasPrefix(uri, "https://")
		config = config.WithDisableSSL(sslDisabled).WithEndpoint(uri)
	}

	sess := session.New(config)

	return &SimpleStorageService{
		client: s3.New(sess),
		bucket: bucket,
	}
}

// NewSimpleStorageServiceDefaults create new S3 client model.
// Use default authentication provides which looks at env variables,
// Aws profile file and ec2 iam role
func NewSimpleStorageServiceDefaults(bucket, region string) *SimpleStorageService {

	sess := session.New(aws.NewConfig().WithRegion(region))

	return &SimpleStorageService{
		client: s3.New(sess),
		bucket: bucket,
	}
}

// Delete removes delected file from storage.
// Noop if ID does not exist.
func (s *SimpleStorageService) Delete(objectID string) error {

	params := &s3.DeleteObjectInput{
		// Required
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectID),

		// Optional
		RequestPayer: aws.String(s3.RequestPayerRequester),
	}

	// ignore return response which contains charing info
	// and file versioning data which are not in interest
	_, err := s.client.DeleteObject(params)
	if err != nil {
		return errors.Wrap(err, "Removing file")
	}

	return nil
}

// Exists check if selected object exists in the storage
func (s *SimpleStorageService) Exists(objectID string) (bool, error) {

	params := &s3.ListObjectsInput{
		// Required
		Bucket: aws.String(s.bucket),

		// Optional
		MaxKeys: aws.Int64(1),
		Prefix:  aws.String(objectID),
	}

	resp, err := s.client.ListObjects(params)
	if err != nil {
		return false, errors.Wrap(err, "Searching for file")
	}

	if len(resp.Contents) == 0 {
		return false, nil
	}

	// Note: Response should contain max 1 object (MaxKetys=1)
	// Double check if it's exact match as object search matches prefix.
	if *resp.Contents[0].Key == objectID {
		return true, nil
	}

	return false, nil
}

// PutRequest duration is limited to 7 days (AWS limitation)
func (s *SimpleStorageService) PutRequest(objectID string, duration time.Duration) (*images.Link, error) {

	if err := s.validateDurationLimits(duration); err != nil {
		return nil, err
	}

	params := &s3.PutObjectInput{
		// Required
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectID),
	}

	// Ignore out object
	req, _ := s.client.PutObjectRequest(params)

	uri, err := req.Presign(duration)
	if err != nil {
		return nil, errors.Wrap(err, "Signing PUT request")
	}

	return images.NewLink(uri, req.Time.Add(req.ExpireTime)), nil
}

// GetRequest duration is limited to 7 days (AWS limitation)
func (s *SimpleStorageService) GetRequest(objectID string, duration time.Duration) (*images.Link, error) {

	if err := s.validateDurationLimits(duration); err != nil {
		return nil, err
	}

	params := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectID),
	}

	// Ignore out object
	req, _ := s.client.GetObjectRequest(params)

	uri, err := req.Presign(duration)
	if err != nil {
		return nil, errors.Wrap(err, "Signing GET request")
	}

	return images.NewLink(uri, req.Time.Add(req.ExpireTime)), nil
}

func (s *SimpleStorageService) validateDurationLimits(duration time.Duration) error {
	if duration > ExpireMaxLimit || duration < ExpireMinLimit {
		return fmt.Errorf("Expire duration out of range: allowed %d-%d[ns]",
			ExpireMinLimit, ExpireMaxLimit)
	}

	return nil
}

// LastModified returns last file modification time.
// If object not found return ErrFileStorageFileNotFound
func (s *SimpleStorageService) LastModified(objectID string) (time.Time, error) {

	params := &s3.ListObjectsInput{
		// Required
		Bucket: aws.String(s.bucket),

		// Optional
		MaxKeys: aws.Int64(1),
		Prefix:  aws.String(objectID),
	}

	resp, err := s.client.ListObjects(params)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "Searching for file")
	}

	if len(resp.Contents) == 0 {
		return time.Time{}, model.ErrFileStorageFileNotFound
	}

	// Note: Response should contain max 1 object (MaxKetys=1)
	// Double check if it's exact match as object search matches prefix.
	if *resp.Contents[0].Key != objectID {
		return time.Time{}, model.ErrFileStorageFileNotFound
	}

	return *resp.Contents[0].LastModified, nil
}
