// Copyright 2021 Northern.tech AS
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

package app

import (
	"context"
	"testing"

	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/store/mocks"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetStorageSettings(t *testing.T) {
	testCases := map[string]struct {
		tenantID string
		settings *model.StorageSettings
		err      error
	}{
		"ok": {
			settings: &model.StorageSettings{
				Region: "region",
				Key:    "secretkey",
				Secret: "secret",
				Bucket: "bucket",
				Uri:    "https://example.com",
				Token:  "token",
			},
		},
		"error": {
			settings: &model.StorageSettings{
				Region: "region",
				Key:    "secretkey",
				Secret: "secret",
				Bucket: "bucket",
				Uri:    "https://example.com",
				Token:  "token",
			},
			err: errors.New("generic error"),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			db := mocks.DataStore{}
			db.On("GetStorageSettings",
				mock.MatchedBy(func(ctx context.Context) bool { return true }),
			).Return(tc.settings, tc.err)
			ctx := context.Background()

			ds := &Deployments{
				db: &db,
			}

			settings, err := ds.GetStorageSettings(ctx)

			if tc.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tc.settings, settings)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestSetStorageSettings(t *testing.T) {
	testCases := map[string]struct {
		tenantID string
		settings *model.StorageSettings
		err      error
	}{
		"ok": {
			settings: &model.StorageSettings{
				Region: "region",
				Key:    "secretkey",
				Secret: "secret",
				Bucket: "bucket",
				Uri:    "https://example.com",
				Token:  "token",
			},
		},
		"error invalid data": {
			settings: &model.StorageSettings{
				Region: "r",
				Key:    "k",
			},
			err: errors.New("generic error"),
		},
		"error failed db call": {
			settings: &model.StorageSettings{
				Region: "region",
				Key:    "secretkey",
				Secret: "secret",
				Bucket: "bucket",
				Uri:    "https://example.com",
				Token:  "token",
			},
			err: errors.New("generic error"),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			db := mocks.DataStore{}
			db.On("SetStorageSettings",
				mock.MatchedBy(func(ctx context.Context) bool { return true }),
				tc.settings,
			).Return(tc.err)
			ds := &Deployments{
				db: &db,
			}
			ctx := context.Background()

			err := ds.SetStorageSettings(ctx, tc.settings)

			if tc.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
