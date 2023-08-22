// Copyright 2023 Northern.tech AS
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

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/store"
	"github.com/mendersoftware/deployments/store/mocks"
)

func TestReplaceReleaseTags(t *testing.T) {
	t.Parallel()

	type testCase struct {
		Name string

		context.Context
		ReleaseName string
		Tags        model.Tags

		GetDatabase func(t *testing.T, self *testCase) *mocks.DataStore

		Error error
	}
	testCases := []testCase{{
		Name: "ok",

		Context:     context.Background(),
		ReleaseName: "foobar",
		Tags:        model.Tags{"foo", "baz"},

		GetDatabase: func(t *testing.T, self *testCase) *mocks.DataStore {
			ds := new(mocks.DataStore)
			ds.On("ReplaceReleaseTags", self.Context, self.ReleaseName, self.Tags).
				Return(nil)
			return ds
		},
	}, {
		Name: "error/not found",

		Context:     context.Background(),
		ReleaseName: "foobar",
		Tags:        model.Tags{"foo", "baz"},

		GetDatabase: func(t *testing.T, self *testCase) *mocks.DataStore {
			ds := new(mocks.DataStore)
			ds.On("ReplaceReleaseTags", self.Context, self.ReleaseName, self.Tags).
				Return(store.ErrNotFound)
			return ds
		},
		Error: ErrReleaseNotFound,
	}, {
		Name: "error/too many unique keys",

		Context:     context.Background(),
		ReleaseName: "foobar",
		Tags:        model.Tags{"foo", "baz"},

		GetDatabase: func(t *testing.T, self *testCase) *mocks.DataStore {
			ds := new(mocks.DataStore)
			ds.On("ReplaceReleaseTags", self.Context, self.ReleaseName, self.Tags).
				Return(model.ErrTooManyUniqueTags)
			return ds
		},
		Error: model.ErrTooManyUniqueTags,
	}, {
		Name: "error/internal error",

		Context:     context.Background(),
		ReleaseName: "foobar",
		Tags:        model.Tags{"foo", "baz"},

		GetDatabase: func(t *testing.T, self *testCase) *mocks.DataStore {
			ds := new(mocks.DataStore)
			ds.On("ReplaceReleaseTags", self.Context, self.ReleaseName, self.Tags).
				Return(errors.New("internal error with sensitive info"))
			return ds
		},
		Error: ErrModelInternal,
	}}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			ds := tc.GetDatabase(t, &tc)
			defer ds.AssertExpectations(t)

			app := NewDeployments(ds, nil, 0, false)

			err := app.ReplaceReleaseTags(tc.Context, tc.ReleaseName, tc.Tags)
			if tc.Error != nil {
				assert.ErrorIs(t, err, tc.Error)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestListReleaseTags(t *testing.T) {
	t.Parallel()

	type testCase struct {
		Name string

		context.Context

		GetDatabase func(t *testing.T, self *testCase) *mocks.DataStore

		Tags  model.Tags
		Error error
	}
	testCases := []testCase{{
		Name: "ok",

		Context: context.Background(),
		Tags:    model.Tags{"field1", "field2"},

		GetDatabase: func(t *testing.T, self *testCase) *mocks.DataStore {
			ds := new(mocks.DataStore)
			ds.On("ListReleaseTags", self.Context).
				Return(self.Tags, nil)
			return ds
		},
	}, {
		Name: "error/internal error",

		Context: context.Background(),

		GetDatabase: func(t *testing.T, self *testCase) *mocks.DataStore {
			ds := new(mocks.DataStore)
			ds.On("ListReleaseTags", self.Context).
				Return(nil, errors.New("internal error with sensitive info"))
			return ds
		},
		Error: ErrModelInternal,
	}}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			ds := tc.GetDatabase(t, &tc)
			defer ds.AssertExpectations(t)

			app := NewDeployments(ds, nil, 0, false)

			tags, err := app.ListReleaseTags(tc.Context)
			if tc.Error != nil {
				assert.ErrorIs(t, err, tc.Error)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Tags, tags)
			}
		})
	}
}

func TestGetReleasesUpdateTypes(t *testing.T) {
	t.Parallel()

	type testCase struct {
		Name string

		context.Context

		GetDatabase func(t *testing.T, self *testCase) *mocks.DataStore

		Types []string
		Error error
	}
	testCases := []testCase{{
		Name: "ok",

		Context: context.Background(),
		Types:   []string{"field1", "field2"},

		GetDatabase: func(t *testing.T, self *testCase) *mocks.DataStore {
			ds := new(mocks.DataStore)
			ds.On("GetUpdateTypes", self.Context).
				Return(self.Types, nil)
			return ds
		},
	}, {
		Name: "error/internal error",

		Context: context.Background(),

		GetDatabase: func(t *testing.T, self *testCase) *mocks.DataStore {
			ds := new(mocks.DataStore)
			ds.On("GetUpdateTypes", self.Context).
				Return([]string{}, errors.New("internal error with sensitive info"))
			return ds
		},
		Error: ErrModelInternal,
	}}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			ds := tc.GetDatabase(t, &tc)
			defer ds.AssertExpectations(t)

			app := NewDeployments(ds, nil, 0, false)

			tags, err := app.GetReleasesUpdateTypes(tc.Context)
			if tc.Error != nil {
				assert.ErrorIs(t, err, tc.Error)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Types, tags)
			}
		})
	}
}

func TestUpdateRelease(t *testing.T) {
	t.Parallel()

	type testCase struct {
		Name string

		context.Context
		ReleaseName string
		Release     model.ReleasePatch

		GetDatabase func(t *testing.T, self *testCase) *mocks.DataStore

		Error error
	}
	testCases := []testCase{
		{
			Name: "ok",

			Context:     context.Background(),
			ReleaseName: "foobar",
			Release:     model.ReleasePatch{Notes: "New Release fixes 2023"},

			GetDatabase: func(t *testing.T, self *testCase) *mocks.DataStore {
				ds := new(mocks.DataStore)
				ds.On("UpdateRelease", self.Context, self.ReleaseName, self.Release).
					Return(nil)
				return ds
			},
		},
		{
			Name: "error/not found",

			Context:     context.Background(),
			ReleaseName: "foobar",
			Release:     model.ReleasePatch{Notes: "New Release fixes 2023"},

			GetDatabase: func(t *testing.T, self *testCase) *mocks.DataStore {
				ds := new(mocks.DataStore)
				ds.On("UpdateRelease", self.Context, self.ReleaseName, self.Release).
					Return(store.ErrNotFound)
				return ds
			},
			Error: ErrReleaseNotFound,
		},
		{
			Name: "error/internal error",

			Context:     context.Background(),
			ReleaseName: "foobar",
			Release:     model.ReleasePatch{Notes: "New Release fixes 2023"},

			GetDatabase: func(t *testing.T, self *testCase) *mocks.DataStore {
				ds := new(mocks.DataStore)
				ds.On("UpdateRelease", self.Context, self.ReleaseName, self.Release).
					Return(errors.New("internal error with sensitive info"))
				return ds
			},
			Error: ErrModelInternal,
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			ds := tc.GetDatabase(t, &tc)
			defer ds.AssertExpectations(t)

			app := NewDeployments(ds, nil, 0, false)

			err := app.UpdateRelease(tc.Context, tc.ReleaseName, tc.Release)
			if tc.Error != nil {
				assert.ErrorIs(t, err, tc.Error)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
