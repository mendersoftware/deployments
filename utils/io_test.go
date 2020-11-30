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

package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type NoopReader struct{}

func (r NoopReader) Read(b []byte) (int, error) {
	return len(b), nil
}

func TestLimitedReader(t *testing.T) {
	lr := &LimitedReader{
		R:          NoopReader{},
		N:          48,
		LimitError: errors.New("bogus error"),
	}
	b := make([]byte, 32)

	n, err := lr.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, 32, n)

	n, err = lr.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, 16, n)

	n, err = lr.Read(b)
	assert.EqualError(t, err, "bogus error")
	assert.Equal(t, 0, n)
}
