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
	"flag"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/google/uuid"
	"github.com/mendersoftware/deployments/storage"
	"github.com/stretchr/testify/assert"
)

var (
	TEST_AZURE_CONNECTION_STRING = flag.String(
		"azure-connection-string",
		os.Getenv("TEST_AZURE_CONNECTION_STRING"),
		"Connection string for azure tests (env: TEST_AZURE_CONNECTION_STRING)",
	)
	TEST_AZURE_CONTAINER_NAME = flag.String(
		"azure-container-name",
		os.Getenv("TEST_AZURE_CONTAINER_NAME"),
		"Container name for azblob tests (env: TEST_AZURE_CONTAINER_NAME)",
	)
	TEST_AZURE_STORAGE_ACCOUNT_NAME = flag.String(
		"azure-account-name",
		os.Getenv("TEST_AZURE_STORAGE_ACCOUNT_NAME"),
		"The storage account name to use for testing "+
			"(env: TEST_AZURE_STORAGE_ACCOUNT_NAME)",
	)
	TEST_AZURE_STORAGE_ACCOUNT_KEY = flag.String(
		"azure-account-key",
		os.Getenv("TEST_AZURE_STORAGE_ACCOUNT_KEY"),
		"The storage account key to use for testing "+
			"(env: TEST_AZURE_STORAGE_ACCOUNT_KEY)",
	)
)

var azureOptions *Options

func initOptions() {
	opts := NewOptions().
		SetFilenameSuffix(".mender").
		SetContentType("vnd/testing").
		SetBufferSize(BufferSizeMin)
	if *TEST_AZURE_CONTAINER_NAME == "" {
		return
	} else if *TEST_AZURE_CONNECTION_STRING != "" {
		opts.SetConnectionString(*TEST_AZURE_CONNECTION_STRING)
	} else if *TEST_AZURE_STORAGE_ACCOUNT_NAME != "" && *TEST_AZURE_STORAGE_ACCOUNT_KEY != "" {
		opts.SetSharedKey(SharedKeyCredentials{
			AccountName: *TEST_AZURE_STORAGE_ACCOUNT_NAME,
			AccountKey:  *TEST_AZURE_STORAGE_ACCOUNT_KEY,
		})
	} else {
		return
	}
	azureOptions = opts
}

func TestMain(m *testing.M) {
	flag.Parse()
	initOptions()
	ec := m.Run()
	os.Exit(ec)
}

func TestObjectStorage(t *testing.T) {
	if azureOptions == nil {
		t.Skip("Requires env variables TEST_AZURE_CONTAINER_NAME and " +
			"either TEST_AZURE_CONNECTION_STRING or " +
			"TEST_AZURE_STORAGE_ACCOUNT_NAME and TEST_AZURE_STORAGE_ACCOUNT_KEY")
	}
	const (
		blobContent = `foobarbaz`
	)

	ctx := context.Background()

	pathPrefix := "test_" + uuid.NewString() + "/"

	c, err := New(
		ctx,
		*TEST_AZURE_CONTAINER_NAME,
		azureOptions,
	)
	if !assert.NoError(t, err) {
		return
	}
	t.Cleanup(func() {
		cc := c.(*client)
		cur := cc.ListBlobsFlat(&azblob.ContainerListBlobsFlatOptions{
			Prefix: &pathPrefix,
		})
		for cur.NextPage(ctx) {
			rsp := cur.PageResponse()
			if rsp.Segment != nil {
				for _, item := range rsp.Segment.BlobItems {
					if item.Name != nil {
						err = c.DeleteObject(ctx, *item.Name)
						if err != nil {
							t.Logf("Failed to delete blob %s: %s", *item.Name, err)
						}
					}
				}
			}
		}
		if err := cur.Err(); err != nil {
			t.Log("ERROR: Failed to clean up testing data:", err)
		}
	})
	err = c.PutObject(ctx, pathPrefix+"foo", strings.NewReader(blobContent))
	assert.NoError(t, err)

	stat, err := c.StatObject(ctx, pathPrefix+"foo")
	if assert.NoError(t, err) {
		assert.WithinDuration(t, time.Now(), *stat.LastModified, time.Second*10,
			"StatObject; last modified timestamp is not close to present time")
	}

	client := new(http.Client)

	// Test signed requests

	// Generate signed URL for object that does not exist
	_, err = c.GetRequest(context.Background(), pathPrefix+"not_found", time.Minute)
	assert.ErrorIs(t, err, storage.ErrObjectNotFound)

	link, err := c.GetRequest(context.Background(), pathPrefix+"foo", time.Minute)
	if assert.NoError(t, err) {
		req, err := http.NewRequest(link.Method, link.Uri, nil)
		if assert.NoError(t, err) {

			rsp, err := client.Do(req)
			assert.NoError(t, err)
			b, err := io.ReadAll(rsp.Body)
			assert.NoError(t, err)
			_ = rsp.Body.Close()
			assert.Equal(t, blobContent, string(b))
		}
	}

	link, err = c.DeleteRequest(context.Background(), pathPrefix+"foo", time.Minute)
	if assert.NoError(t, err) {
		req, err := http.NewRequest(link.Method, link.Uri, nil)
		if assert.NoError(t, err) {
			rsp, err := client.Do(req)
			if assert.NoError(t, err) {
				assert.Equal(t, http.StatusAccepted, rsp.StatusCode)
				_, err = c.StatObject(ctx, pathPrefix+"foo")
				assert.ErrorIs(t, err, storage.ErrObjectNotFound)
			}
		}
	}

	link, err = c.PutRequest(context.Background(), pathPrefix+"bar", time.Minute*5)
	if assert.NoError(t, err) {
		req, err := http.NewRequest(link.Method, link.Uri, strings.NewReader(blobContent))
		if assert.NoError(t, err) {
			for key, value := range link.Header {
				req.Header.Set(key, value)
			}
			rsp, err := client.Do(req)
			if assert.NoError(t, err) {
				assert.Equal(t, http.StatusCreated, rsp.StatusCode)
				stat, err = c.StatObject(ctx, pathPrefix+"bar")
				if assert.NoError(t, err) {
					assert.Equal(t, int64(len(blobContent)), *stat.Size)
				}

			}
		}
	}

	err = c.DeleteObject(ctx, pathPrefix+"baz")
	assert.ErrorIs(t, err, storage.ErrObjectNotFound)
	assert.Contains(t, err.Error(), storage.ErrObjectNotFound.Error())

	err = c.PutObject(ctx, pathPrefix+"baz", strings.NewReader(blobContent))
	if assert.NoError(t, err) {
		err = c.DeleteObject(ctx, pathPrefix+"baz")
		assert.NoError(t, err)
	}
}
