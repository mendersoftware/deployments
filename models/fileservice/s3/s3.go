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
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/mendersoftware/artifacts/models/fileservice"
)

const (
	ExpireMaxLimit = 7 * 24 * time.Hour
	ExpireMinLimit = 1 * time.Minute
)

// AWS S3 client. Implements FileServiceModelI
type SimpleStorageService struct {
	client *s3.S3
	bucket string
}

// NewSimpleStorageService create new S3 client model.
// AWS authentication keys are automatically reloaded from env variables.
func NewSimpleStorageServiceStatic(bucket, key, secret, region, token string) *SimpleStorageService {

	credentials := credentials.NewStaticCredentials(key, secret, token)
	config := aws.NewConfig().WithCredentials(credentials).WithRegion(region)

	sess := session.New(config)

	return &SimpleStorageService{
		client: s3.New(sess),
		bucket: bucket,
	}
}

// NewSimpleStorageService create new S3 client model.
// Use default authentication provides which looks at env variables,
// aws profile file and ec2 iam role
func NewSimpleStorageServiceDefaults(bucket, region string) *SimpleStorageService {

	sess := session.New(aws.NewConfig().WithRegion(region))

	return &SimpleStorageService{
		client: s3.New(sess),
		bucket: bucket,
	}
}

// makeFileId creates file s3 path based on object id and customer id.
// Current structure used is directory per customer id: <customerId>/<objectId>
func (s *SimpleStorageService) makeFileId(customerId, objectId string) string {
	return customerId + "/" + objectId
}

func (s *SimpleStorageService) Delete(customerId, objectId string) error {

	id := s.makeFileId(customerId, objectId)

	params := &s3.DeleteObjectInput{
		// Required
		Bucket: aws.String(s.bucket),
		Key:    aws.String(id),

		// Optional
		RequestPayer: aws.String(s3.RequestPayerRequester),
	}

	// ignore return response which contains charing info
	// and file versioning data which are not in interest
	_, err := s.client.DeleteObject(params)
	if err != nil {
		return err
	}

	return nil
}

func (s *SimpleStorageService) Exists(customerId, objectId string) (bool, error) {

	id := s.makeFileId(customerId, objectId)

	params := &s3.ListObjectsInput{
		// Required
		Bucket: aws.String(s.bucket),

		// Optional
		MaxKeys: aws.Int64(1),
		Prefix:  aws.String(id),
	}

	resp, err := s.client.ListObjects(params)
	if err != nil {
		return false, err
	}

	if len(resp.Contents) == 0 {
		return false, nil
	}

	// Note: Response should contain max 1 object (MaxKetys=1)
	// Double check if it's exact match as object search matches prefix.
	if resp.Contents[0].Key == aws.String("id") {
		return true, nil
	}

	return false, nil
}

// PutRequest duration is limited to 7 days (AWS limitation)
func (s *SimpleStorageService) PutRequest(customerId, objectId string, duration time.Duration) (*fileservice.Link, error) {

	if err := s.validateDurationLimits(duration); err != nil {
		return nil, err
	}

	id := s.makeFileId(customerId, objectId)

	params := &s3.PutObjectInput{
		// Required
		Bucket: aws.String(s.bucket),
		Key:    aws.String(id),
	}

	// Ignore out object
	req, _ := s.client.PutObjectRequest(params)

	uri, err := req.Presign(duration)
	if err != nil {
		return nil, err
	}

	return fileservice.NewLink(uri, req.Time), nil
}

// GetRequest duration is limited to 7 days (AWS limitation)
func (s *SimpleStorageService) GetRequest(customerId, objectId string, duration time.Duration) (*fileservice.Link, error) {

	if err := s.validateDurationLimits(duration); err != nil {
		return nil, err
	}

	id := s.makeFileId(customerId, objectId)

	params := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(id),
	}

	// Ignore out object
	req, _ := s.client.GetObjectRequest(params)

	uri, err := req.Presign(duration)
	if err != nil {
		return nil, err
	}

	return fileservice.NewLink(uri, req.Time), nil
}

func (s *SimpleStorageService) validateDurationLimits(duration time.Duration) error {
	if duration > ExpireMaxLimit || duration < ExpireMinLimit {
		return errors.New(fmt.Sprintf("Expire duration out of range: %d[ns] allowed %d-%d[ns]",
			duration, ExpireMinLimit, ExpireMaxLimit))
	}

	return nil
}
