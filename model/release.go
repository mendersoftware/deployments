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
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	TagsPerReleaseMax = 20  // Maximum number of tags per release.
	TagsMaxUniqueKeys = 100 // Maximum number of unique tag keys.
)

var (
	ErrTooManyTags = errors.New(
		"the total number of tags per exceeded maximum of " +
			strconv.Itoa(TagsPerReleaseMax),
	)
	ErrTooManyUniqueTags = errors.New(
		"the total number of unique tags (" +
			strconv.Itoa(TagsMaxUniqueKeys) +
			") has been exceeded",
	)
)

type Tags []Tag

func (tags Tags) Validate() (err error) {
	if len(tags) > TagsPerReleaseMax {
		return ErrTooManyTags
	}
	return validation.Validate([]Tag(tags))
}

func (tags Tags) MarshalJSON() ([]byte, error) {
	if len(tags) == 0 {
		return []byte{'[', ']'}, nil
	}
	return json.Marshal([]Tag(tags))
}

func (tags *Tags) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, (*[]Tag)(tags))
	if err != nil {
		return err
	}
	// Deduplicate tag keys:
	s := *tags
	sort.SliceStable(s, func(i, j int) bool {
		return strings.Compare(s[i].Key, s[j].Key) < 0
	})
	i := 1
	for i < len(s) {
		p := s[i]
		if p.Key == s[i-1].Key {
			// Swap duplicate and shrink slice
			lastIdx := len(s) - 1
			s[i] = s[lastIdx]
			s[lastIdx] = p
			s = s[:lastIdx]
		} else {
			i++
		}
	}
	*tags = s
	return nil
}

type Tag struct {
	Key   string `json:"key" bson:"key"`
	Value string `json:"value" bson:"value"`
}

func (tag Tag) Validate() error {
	return validation.ValidateStruct(&tag,
		validation.Field(&tag.Key, validation.Required, lengthIn0To200),
		validation.Field(&tag.Value, lengthLessThan4096),
	)
}

type Release struct {
	Name      string     `json:"Name" bson:"_id"`
	Modified  *time.Time `json:"Modified,omitempty" bson:"modified,omitempty"`
	Artifacts []Image    `json:"Artifacts" bson:"artifacts"`
	Tags      Tags       `json:"tags" bson:"tags,omitempty"`
}

type ReleaseOrImageFilter struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	DeviceType  string `json:"device_type"`
	Page        int    `json:"page"`
	PerPage     int    `json:"per_page"`
	Sort        string `json:"sort"`
}
