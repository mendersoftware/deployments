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

package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"

	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/go-lib-micro/identity"
)

const (
	ExpireMaxLimit                 = 7 * 24 * time.Hour
	ExpireMinLimit                 = 1 * time.Minute
	ErrCodeBucketAlreadyOwnedByYou = "BucketAlreadyOwnedByYou"
)

// Errors specific to interface
var (
	ErrFileStorageFileNotFound = errors.New("File not found")
)

// FileStorage allows to store and manage large files
type FileStorage interface {
	Delete(ctx context.Context, objectId string) error
	Exists(ctx context.Context, objectId string) (bool, error)
	LastModified(ctx context.Context, objectId string) (time.Time, error)
	PutRequest(ctx context.Context, objectId string,
		duration time.Duration) (*model.Link, error)
	GetRequest(ctx context.Context, objectId string,
		duration time.Duration, responseContentType string) (*model.Link, error)
	DeleteRequest(ctx context.Context, objectId string,
		duration time.Duration) (*model.Link, error)
	UploadArtifact(ctx context.Context, objectId string,
		artifact io.Reader, contentType string) error
}

// SimpleStorageService - AWS S3 client.
// Data layer for file storage.
// Implements model.FileStorage interface
type SimpleStorageService struct {
	client      *s3.S3
	bucket      string
	tagArtifact bool
}

// NewSimpleStorageServiceStatic create new S3 client model.
// AWS authentication keys are automatically reloaded from env variables.
func NewSimpleStorageServiceStatic(bucket, key, secret, region, token, uri string, tag_artifact, forcePathStyle bool) (*SimpleStorageService, error) {
	credentials := credentials.NewStaticCredentials(key, secret, token)
	config := aws.NewConfig().WithCredentials(credentials).WithRegion(region)

	if len(uri) > 0 {
		sslDisabled := !strings.HasPrefix(uri, "https://")
		config = config.WithDisableSSL(sslDisabled).WithEndpoint(uri)
	}

	// Amazon S3 will no longer support path-style API requests starting September 30th, 2020
	// S3 buckets created after September 30, 2020 will support only virtual-hosted style requests
	// Setting S3ForcePathStyle to false forces virtual-hosted style.
	config.S3ForcePathStyle = aws.Bool(forcePathStyle)

	sess := session.New(config)

	client := s3.New(sess)

	// minio requires explicit bucket creation
	cparams := &s3.CreateBucketInput{
		Bucket: aws.String(bucket), // Required
	}

	ctx := context.Background()

	// timeout set to 5 seconds
	var cancelFn func()
	ctxWithTimeout, cancelFn := context.WithTimeout(ctx, 5*time.Second)
	defer cancelFn()

	_, err := client.CreateBucketWithContext(ctxWithTimeout, cparams)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() != ErrCodeBucketAlreadyOwnedByYou {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return &SimpleStorageService{
		client:      client,
		bucket:      bucket,
		tagArtifact: tag_artifact,
	}, nil
}

// NewSimpleStorageServiceDefaults create new S3 client model.
// Use default authentication provides which looks at env variables,
// Aws profile file and ec2 iam role
func NewSimpleStorageServiceDefaults(bucket, region string) (*SimpleStorageService, error) {

	sess := session.New(aws.NewConfig().WithRegion(region))
	client := s3.New(sess)

	// minio requires explicit bucket creation
	cparams := &s3.CreateBucketInput{
		Bucket: aws.String(bucket), // Required
	}

	ctx := context.Background()

	// timeout set to 5 seconds
	var cancelFn func()
	ctxWithTimeout, cancelFn := context.WithTimeout(ctx, 5*time.Second)
	defer cancelFn()

	_, err := client.CreateBucketWithContext(ctxWithTimeout, cparams)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() != ErrCodeBucketAlreadyOwnedByYou {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return &SimpleStorageService{
		client: client,
		bucket: bucket,
	}, nil
}

func getArtifactByTenant(ctx context.Context, objectID string) string {
	if id := identity.FromContext(ctx); id != nil && len(id.Tenant) > 0 {
		return fmt.Sprintf("%s/%s", id.Tenant, objectID)
	}

	return objectID
}

// Delete removes deleted file from storage.
// Noop if ID does not exist.
func (s *SimpleStorageService) Delete(ctx context.Context, objectID string) error {
	objectID = getArtifactByTenant(ctx, objectID)

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
func (s *SimpleStorageService) Exists(ctx context.Context, objectID string) (bool, error) {
	objectID = getArtifactByTenant(ctx, objectID)

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

func fillBuffer(b []byte, r io.Reader) (int, error) {
	var offset int
	var err error
	for n := 0; offset < len(b) && err == nil; offset += n {
		n, err = r.Read(b[offset:])
	}
	return offset, err
}

// uploadMultipart uploads an artifact using the multipart API.
func (s *SimpleStorageService) uploadMultipart(
	ctx context.Context,
	buf []byte,
	objectPath string,
	artifact io.Reader,
	contentType string,
) error {
	const maxPartNum = 10000
	var partNum int64 = 1
	var rspUpload *s3.UploadPartOutput

	// Pre-allocate 100 completed part (generous guesstimate)
	completedParts := make([]*s3.CompletedPart, 0, 100)
	// expiresAt is the maximum time the s3 service will cache the uploaded
	// parts. All request will be presigned relative to this time.
	expiresAt := time.Now().Add(10 * time.Minute)
	requestOptions := func(req *request.Request) {
		// This will pre-sign the request for the given duration.
		exp := expiresAt.Sub(time.Now())
		req.ExpireTime = exp
	}

	// Initiate Multipart upload
	createParams := &s3.CreateMultipartUploadInput{
		Bucket:      &s.bucket,
		Key:         &objectPath,
		ContentType: &contentType,
		Expires:     &expiresAt,
	}
	rspCreate, err := s.client.CreateMultipartUploadWithContext(
		ctx, createParams, requestOptions,
	)
	if err != nil {
		return err
	}
	uploadParams := &s3.UploadPartInput{
		Bucket:     &s.bucket,
		Key:        &objectPath,
		UploadId:   rspCreate.UploadId,
		PartNumber: &partNum,
	}

	// Upload the first chunk already stored in buffer
	r := bytes.NewReader(buf)
	// Readjust upload parameters
	uploadParams.Body = r
	rspUpload, err = s.client.UploadPartWithContext(
		ctx,
		uploadParams,
		requestOptions,
	)
	if err != nil {
		return err
	}
	part := partNum
	completedParts = append(
		completedParts,
		&s3.CompletedPart{
			ETag:       rspUpload.ETag,
			PartNumber: &part,
		},
	)

	// The following is loop is very similar to io.Copy except the
	// destination is the s3 bucket.
	for partNum++; partNum < maxPartNum; partNum++ {
		// Read next chunk from stream (fill the whole buffer)
		offset, eRead := fillBuffer(buf, artifact)
		if offset > 0 {
			r := bytes.NewReader(buf[:offset])
			// Readjust upload parameters
			uploadParams.Body = r
			rspUpload, err = s.client.UploadPartWithContext(
				ctx,
				uploadParams,
				requestOptions,
			)
			if err != nil {
				break
			}
			part := partNum
			completedParts = append(
				completedParts,
				&s3.CompletedPart{
					ETag:       rspUpload.ETag,
					PartNumber: &part,
				},
			)
		} else {
			// Read did not return any bytes
			break
		}
		if eRead != nil {
			err = eRead
			break
		}
	}
	if err == nil || err == io.EOF {
		// Complete upload
		uploadParams := &s3.CompleteMultipartUploadInput{
			Bucket:   &s.bucket,
			Key:      &objectPath,
			UploadId: rspCreate.UploadId,
			MultipartUpload: &s3.CompletedMultipartUpload{
				Parts: completedParts,
			},
		}
		_, err = s.client.CompleteMultipartUploadWithContext(
			ctx,
			uploadParams,
			requestOptions,
		)
	} else {
		// Abort multipart upload!
		uploadParams := &s3.AbortMultipartUploadInput{
			Bucket:   &s.bucket,
			Key:      &objectPath,
			UploadId: rspCreate.UploadId,
		}
		_, err = s.client.AbortMultipartUploadWithContext(
			ctx,
			uploadParams,
			requestOptions,
		)
	}
	return err

}

// UploadArtifact uploads given artifact into the file server (AWS S3 or minio)
// using objectID as a key. If the artifact is larger than 5 MiB, the file is
// uploaded using the s3 multipart API, otherwise the object is created in a
// single request.
func (s *SimpleStorageService) UploadArtifact(
	ctx context.Context,
	objectID string,
	artifact io.Reader,
	contentType string,
) error {
	// NOTE: This size along with the 10000 part limit sets the ultimate
	//       limit on the upload size. (currently at ~97.5GiB)
	const multipartSize = 10 * 1024 * 1024 // 10MiB (must be at least 5MiB)

	objectID = getArtifactByTenant(ctx, objectID)

	buf := make([]byte, multipartSize)
	n, err := fillBuffer(buf, artifact)

	// If only one part, use PutObject API.
	if n < len(buf) || err == io.EOF {
		// Ordinary single-file upload
		uploadParams := &s3.PutObjectInput{
			Body:   bytes.NewReader(buf[:n]),
			Bucket: &s.bucket,
			Key:    &objectID,
		}
		_, err := s.client.PutObjectWithContext(
			ctx,
			uploadParams,
			func(req *request.Request) {
				// ExpireTime will presign URI to expire after
				// 5 minutes
				req.ExpireTime = time.Minute * 5
			},
		)
		return err
	}
	return s.uploadMultipart(ctx, buf, objectID, artifact, contentType)
}

// PutRequest duration is limited to 7 days (AWS limitation)
func (s *SimpleStorageService) PutRequest(ctx context.Context, objectID string,
	duration time.Duration) (*model.Link, error) {

	objectID = getArtifactByTenant(ctx, objectID)

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

	return model.NewLink(uri, req.Time.Add(req.ExpireTime)), nil
}

// GetRequest duration is limited to 7 days (AWS limitation)
func (s *SimpleStorageService) GetRequest(ctx context.Context, objectID string,
	duration time.Duration, responseContentType string) (*model.Link, error) {

	if err := s.validateDurationLimits(duration); err != nil {
		return nil, err
	}

	objectID = getArtifactByTenant(ctx, objectID)

	params := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectID),
	}

	if responseContentType != "" {
		params.ResponseContentType = &responseContentType
	}

	// Ignore out object
	req, _ := s.client.GetObjectRequest(params)

	uri, err := req.Presign(duration)
	if err != nil {
		return nil, errors.Wrap(err, "Signing GET request")
	}

	return model.NewLink(uri, req.Time.Add(req.ExpireTime)), nil
}

// DeleteRequest returns a presigned deletion request
func (s *SimpleStorageService) DeleteRequest(ctx context.Context, objectID string,
	duration time.Duration) (*model.Link, error) {

	if err := s.validateDurationLimits(duration); err != nil {
		return nil, err
	}

	objectID = getArtifactByTenant(ctx, objectID)

	params := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectID),
	}

	// Ignore out object
	req, _ := s.client.DeleteObjectRequest(params)

	uri, err := req.Presign(duration)
	if err != nil {
		return nil, errors.Wrap(err, "Signing DELETE request")
	}

	return model.NewLink(uri, req.Time.Add(req.ExpireTime)), nil
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
func (s *SimpleStorageService) LastModified(ctx context.Context, objectID string) (time.Time, error) {

	objectID = getArtifactByTenant(ctx, objectID)

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
		return time.Time{}, ErrFileStorageFileNotFound
	}

	// Note: Response should contain max 1 object (MaxKetys=1)
	// Double check if it's exact match as object search matches prefix.
	if *resp.Contents[0].Key != objectID {
		return time.Time{}, ErrFileStorageFileNotFound
	}

	return *resp.Contents[0].LastModified, nil
}
