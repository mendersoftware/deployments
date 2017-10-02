// Copyright 2017 Northern.tech AS
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

package model_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/mendersoftware/deployments/resources/limits"
	. "github.com/mendersoftware/deployments/resources/limits/model"
	"github.com/mendersoftware/deployments/resources/limits/model/mocks"
)

func TestGetLimit(t *testing.T) {
	testCases := []struct {
		name string

		getLimit *limits.Limit
		getErr   error

		expected *limits.Limit
		err      error
	}{
		{
			name: "foo",
			getLimit: &limits.Limit{
				Name:  "foo",
				Value: 123,
			},
			expected: &limits.Limit{
				Name:  "foo",
				Value: 123,
			},
		},
		{
			name:   "not-found",
			getErr: ErrLimitNotFound,
			expected: &limits.Limit{
				Name:  "not-found",
				Value: 0,
			},
		},
		{
			name:   "not-found",
			getErr: errors.New("error"),
			err:    errors.New("failed to obtain limit from storage: error"),
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			ls := mocks.LimitsStorage{}
			ls.On("GetLimit",
				mock.MatchedBy(
					func(_ context.Context) bool {
						return true
					}),
				tc.name).Return(tc.getLimit, tc.getErr)

			lm := NewLimitsModel(&ls)

			ctx := context.Background()
			lim, err := lm.GetLimit(ctx, tc.name)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error())
			} else {
				assert.NoError(t, err)
			}
			if tc.expected != nil {
				assert.Equal(t, tc.expected, lim)
			}

			ls.AssertExpectations(t)
		})
	}
}
