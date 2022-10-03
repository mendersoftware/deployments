// Copyright 2022 Northern.tech AS
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

package azblob

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/storage"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

const (
	headerBlobType             = "x-ms-blob-type"
	headerMSContentDisposition = "x-ms-blob-content-disposition"

	blobTypeBlock = "BlockBlob"
)

type client struct {
	DefaultClient *azblob.ContainerClient
	fileSuffix    *string
	contentType   *string
	bufferSize    int
}

func New(ctx context.Context, bucket string, opts ...*Options) (storage.ObjectStorage, error) {
	var (
		err error
		cc  *azblob.ContainerClient
	)
	opt := NewOptions(opts...)
	if opt.ConnectionString != nil {
		cc, err = azblob.NewContainerClientFromConnectionString(
			*opt.ConnectionString, bucket, &azblob.ClientOptions{},
		)
		if err != nil {
			return nil, err
		}
	} else if sk := opt.SharedKey; sk != nil {
		var (
			containerURL string
			azCred       *azblob.SharedKeyCredential
		)
		containerURL, azCred, err = sk.azParams(bucket)
		if err == nil {
			cc, err = azblob.NewContainerClientWithSharedKey(
				containerURL,
				azCred,
				&azblob.ClientOptions{},
			)
		}
	}
	if err != nil {
		return nil, err
	}
	objectStorage := &client{
		DefaultClient: cc,
		fileSuffix:    opt.FilenameSuffix,
		contentType:   opt.ContentType,
	}
	if cc != nil {
		if err := objectStorage.HealthCheck(ctx); err != nil {
			return nil, err
		}
	}
	return objectStorage, nil
}

func (c *client) clientFromContext(
	ctx context.Context,
) (client *azblob.ContainerClient, err error) {
	client = c.DefaultClient
	if settings := storage.SettingsFromContext(ctx); settings != nil {
		if err = settings.Validate(); err != nil {
			return nil, err
		} else if settings.ConnectionString != nil {
			client, err = azblob.NewContainerClientFromConnectionString(
				*settings.ConnectionString,
				settings.Bucket,
				&azblob.ClientOptions{},
			)
		} else {
			var (
				containerURL string
				azCreds      *azblob.SharedKeyCredential
			)
			creds := SharedKeyCredentials{
				AccountName: settings.Key,
				AccountKey:  settings.Secret,
			}
			if settings.Uri != "" {
				creds.URI = &settings.Uri
			}

			containerURL, azCreds, err = creds.azParams(settings.Bucket)
			if err == nil {
				client, err = azblob.NewContainerClientWithSharedKey(
					containerURL,
					azCreds,
					&azblob.ClientOptions{},
				)
			}
		}
	}
	return client, err
}

func (c *client) HealthCheck(ctx context.Context) error {
	azClient, err := c.clientFromContext(ctx)
	if err != nil {
		return OpError{
			Op:     OpHealthCheck,
			Reason: err,
		}
	} else if azClient == nil {
		return nil
	}
	_, err = azClient.GetProperties(ctx, &azblob.ContainerGetPropertiesOptions{})
	if err != nil {
		return OpError{
			Op:     OpHealthCheck,
			Reason: err,
		}
	}
	return nil
}

func (c *client) PutObject(
	ctx context.Context,
	objectPath string,
	src io.Reader,
) error {
	azClient, err := c.clientFromContext(ctx)
	if err != nil {
		return OpError{
			Op:     OpPutObject,
			Reason: err,
		}
	} else if azClient == nil {
		return nil
	}
	bc, err := azClient.NewBlockBlobClient(objectPath)
	if err != nil {
		return OpError{
			Op:      OpPutObject,
			Message: "failed to initialize blob client",
			Reason:  err,
		}
	}
	var blobOpts = azblob.UploadStreamOptions{
		HTTPHeaders: &azblob.BlobHTTPHeaders{
			BlobContentType: c.contentType,
		},
	}
	if c.fileSuffix != nil {
		filename := path.Base(objectPath) + *c.fileSuffix
		disp := fmt.Sprintf(
			`attachment; filename="%s"`, filename,
		)
		blobOpts.HTTPHeaders.BlobContentDisposition = &disp
	}
	blobOpts.BufferSize = c.bufferSize
	_, err = bc.UploadStream(ctx, src, blobOpts)
	if err != nil {
		return OpError{
			Op:      OpPutObject,
			Message: "failed to upload object to blob",
			Reason:  err,
		}
	}
	return err
}

func (c *client) DeleteObject(
	ctx context.Context,
	path string,
) error {
	azClient, err := c.clientFromContext(ctx)
	if err != nil {
		return OpError{
			Op:     OpDeleteObject,
			Reason: err,
		}
	} else if azClient == nil {
		return nil
	}
	bc, err := azClient.NewBlockBlobClient(path)
	if err != nil {
		return OpError{
			Op:      OpDeleteObject,
			Message: "failed to initialize blob client",
			Reason:  err,
		}
	}
	_, err = bc.Delete(ctx, &azblob.BlobDeleteOptions{
		DeleteSnapshots: azblob.DeleteSnapshotsOptionTypeInclude.ToPtr(),
	})
	var storageErr *azblob.StorageError
	if errors.As(err, &storageErr) {
		if storageErr.ErrorCode == azblob.StorageErrorCodeBlobNotFound {
			err = storage.ErrObjectNotFound
		}
	}
	if err != nil {
		return OpError{
			Op:      OpDeleteObject,
			Message: "failed to delete object",
			Reason:  err,
		}
	}
	return nil
}

func (c *client) StatObject(
	ctx context.Context,
	path string,
) (*storage.ObjectInfo, error) {
	azClient, err := c.clientFromContext(ctx)
	if err != nil {
		return nil, OpError{
			Op:     OpStatObject,
			Reason: err,
		}
	} else if azClient == nil {
		return nil, nil
	}
	bc, err := azClient.NewBlockBlobClient(path)
	if err != nil {
		return nil, OpError{
			Op:      OpStatObject,
			Message: "failed to initialize blob client",
			Reason:  err,
		}
	}
	rsp, err := bc.GetProperties(ctx, &azblob.BlobGetPropertiesOptions{})
	var storageErr *azblob.StorageError
	if errors.As(err, &storageErr) {
		if storageErr.ErrorCode == azblob.StorageErrorCodeBlobNotFound {
			err = storage.ErrObjectNotFound
		}
	}
	if err != nil {
		return nil, OpError{
			Op:      OpStatObject,
			Message: "failed to retrieve object properties",
			Reason:  err,
		}
	}
	return &storage.ObjectInfo{
		Path:         path,
		LastModified: rsp.LastModified,
		Size:         rsp.ContentLength,
	}, nil
}

func buildSignedURL(
	blobURL string,
	SASParams azblob.SASQueryParameters,
) (string, error) {
	baseURL, err := url.Parse(blobURL)
	if err != nil {
		return "", err
	}
	qSAS, err := url.ParseQuery(SASParams.Encode())
	if err != nil {
		return "", err
	}
	q := baseURL.Query()
	for key, values := range qSAS {
		for _, value := range values {
			q.Add(key, value)
		}
	}
	baseURL.RawQuery = q.Encode()
	return baseURL.String(), nil
}

func (c *client) GetRequest(
	ctx context.Context,
	path string,
	duration time.Duration,
) (*model.Link, error) {
	azClient, err := c.clientFromContext(ctx)
	if err != nil {
		return nil, OpError{
			Op:     OpGetRequest,
			Reason: err,
		}
	} else if azClient == nil {
		return nil, nil
	}
	// Check if object exists
	bc, err := azClient.NewBlockBlobClient(path)
	if err != nil {
		return nil, OpError{
			Op:      OpGetRequest,
			Message: "failed to initialize blob client",
			Reason:  err,
		}
	}
	_, err = bc.GetProperties(ctx, &azblob.BlobGetPropertiesOptions{})
	var storageErr *azblob.StorageError
	if errors.As(err, &storageErr) {
		if storageErr.ErrorCode == azblob.StorageErrorCodeBlobNotFound {
			err = storage.ErrObjectNotFound
		}
	}
	if err != nil {
		return nil, OpError{
			Op:      OpGetRequest,
			Message: "failed to check preconditions",
			Reason:  err,
		}
	}
	now := time.Now().UTC()
	exp := now.Add(duration)
	qParams, err := bc.GetSASToken(azblob.BlobSASPermissions{Read: true}, now, exp)
	if err != nil {
		return nil, OpError{
			Op:      OpGetRequest,
			Message: "failed to generate SAS token",
			Reason:  err,
		}
	}
	uri, err := buildSignedURL(bc.URL(), qParams)
	if err != nil {
		return nil, OpError{
			Op:      OpGetRequest,
			Message: "failed to create pre-signed URL",
			Reason:  err,
		}
	}
	return &model.Link{
		Uri:    uri,
		Expire: exp,
		Method: http.MethodGet,
	}, nil
}

func (c *client) DeleteRequest(
	ctx context.Context,
	path string,
	duration time.Duration,
) (*model.Link, error) {
	azClient, err := c.clientFromContext(ctx)
	if err != nil {
		return nil, OpError{
			Op:     OpGetRequest,
			Reason: err,
		}
	} else if azClient == nil {
		return nil, nil
	}
	bc, err := azClient.NewBlobClient(path)
	if err != nil {
		return nil, OpError{
			Op:      OpDeleteRequest,
			Message: "failed to initialize blob client",
			Reason:  err,
		}
	}
	now := time.Now().UTC()
	exp := now.Add(duration)
	qParams, err := bc.GetSASToken(azblob.BlobSASPermissions{Delete: true}, now, exp)
	if err != nil {
		return nil, OpError{
			Op:      OpDeleteRequest,
			Message: "failed to generate SAS token",
			Reason:  err,
		}
	}
	uri, err := buildSignedURL(bc.URL(), qParams)
	if err != nil {
		return nil, OpError{
			Op:      OpDeleteRequest,
			Message: "failed to create pre-signed URL",
			Reason:  err,
		}
	}
	return &model.Link{
		Uri:    uri,
		Expire: exp,
		Method: http.MethodDelete,
	}, nil
}

func (c *client) PutRequest(
	ctx context.Context,
	objectPath string,
	duration time.Duration,
) (*model.Link, error) {
	azClient, err := c.clientFromContext(ctx)
	if err != nil {
		return nil, OpError{
			Op:     OpGetRequest,
			Reason: err,
		}
	} else if azClient == nil {
		return nil, nil
	}
	bc, err := azClient.NewBlobClient(objectPath)
	if err != nil {
		return nil, OpError{
			Op:      OpPutRequest,
			Message: "failed to initialize blob client",
			Reason:  err,
		}
	}
	now := time.Now().UTC()
	exp := now.Add(duration)
	qParams, err := bc.GetSASToken(azblob.BlobSASPermissions{
		Create: true,
		Write:  true,
	}, now, exp)
	if err != nil {
		return nil, OpError{
			Op:      OpPutRequest,
			Message: "failed to generate SAS token",
			Reason:  err,
		}
	}
	uri, err := buildSignedURL(bc.URL(), qParams)
	if err != nil {
		return nil, OpError{
			Op:      OpPutRequest,
			Message: "failed to create pre-signed URL",
			Reason:  err,
		}
	}
	hdrs := map[string]string{
		headerBlobType: blobTypeBlock,
	}
	if c.fileSuffix != nil {
		filename := path.Base(objectPath) + *c.fileSuffix
		hdrs[headerMSContentDisposition] = fmt.Sprintf(
			`attachment; filename="%s"`, filename,
		)
	}
	return &model.Link{
		Uri:    uri,
		Expire: exp,
		Method: http.MethodPut,
		Header: hdrs,
	}, nil
}
