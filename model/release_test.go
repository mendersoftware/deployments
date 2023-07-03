// Copyright 2023 Northern.tech AS
//
//	Licensed under the Apache License, Version 2.0 (the "License");
//	you may not use this file except in compliance with the License.
//	You may obtain a copy of the License at
//
//	    http://www.apache.org/licenses/LICENSE-2.0
//
//	Unless required by applicable law or agreed to in writing, software
//	distributed under the License is distributed on an "AS IS" BASIS,
//	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	See the License for the specific language governing permissions and
//	limitations under the License.

package model

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReleaseTags(t *testing.T) {

	t.Run("Validate", func(t *testing.T) {
		tooLong := make(Tags, TagsMaxPerRelease+1)
		for i := range tooLong {
			tooLong[i] = Tag(strconv.Itoa(i))
		}
		err := tooLong.Validate()
		assert.ErrorIs(t, err, ErrTooManyTags)

		invalidTags := Tags{"Va1_id", "invælid"}
		err = invalidTags.Validate()
		var charErr *InvalidCharacterError
		assert.ErrorAs(t, err, &charErr)

		tooLongTag := Tag(strings.Repeat("veryLongTag", 100))
		err = tooLongTag.Validate()
		assert.ErrorIs(t, err, ErrTagTooLong)

		var emptyTag Tag
		err = emptyTag.Validate()
		assert.ErrorIs(t, err, ErrTagEmpty)
	})

	t.Run("JSON serialization", func(t *testing.T) {
		var tags Tags
		b, _ := json.Marshal(tags)
		assert.Equal(t, []byte(`[]`), b)
		err := json.Unmarshal(
			[]byte(`["tag", "tag", "more-tags"]`),
			&tags,
		)
		assert.NoError(t, err)
		assert.Equal(t, Tags{"more-tags", "tag"}, tags)

		err = json.Unmarshal([]byte(`{}`), &tags)
		assert.Error(t, err)
	})
}
