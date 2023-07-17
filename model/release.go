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
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	TagsMaxPerRelease = 20  // Maximum number of tags per release.
	TagsMaxUnique     = 100 // Maximum number of unique tags.
)

var (
	ErrTooManyTags = errors.New(
		"the total number of tags per exceeded maximum of " +
			strconv.Itoa(TagsMaxPerRelease),
	)
	ErrTooManyUniqueTags = errors.New(
		"the total number of unique tags (" +
			strconv.Itoa(TagsMaxUnique) +
			") has been exceeded",
	)
)

type Tags []Tag

func (tags Tags) Validate() (err error) {
	if len(tags) > TagsMaxPerRelease {
		return ErrTooManyTags
	}
	for _, tag := range tags {
		if err = tag.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (tags Tags) MarshalJSON() ([]byte, error) {
	if len(tags) == 0 {
		return []byte{'[', ']'}, nil
	}
	return json.Marshal([]Tag(tags))
}

func (tags *Tags) Dedup() {
	// Deduplicate tags:
	s := *tags
	sort.Slice(s, func(i, j int) bool {
		return s[i] < s[j]
	})
	i, j := 0, 0
	for j < len(s) {
		if s[i] != s[j] {
			i++
			s[i] = s[j]
		}
		j++
	}
	*tags = s[:i+1]
}

func (tags *Tags) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, (*[]Tag)(tags))
	if err != nil {
		return err
	}
	tags.Dedup()
	return nil
}

type Tag string

const (
	TagMaxLength = 1024
)

var (
	ErrTagEmpty   = errors.New("tag cannot be empty")
	ErrTagTooLong = errors.New("tag must be less than " +
		strconv.Itoa(TagMaxLength) +
		" characters")
)

type InvalidCharacterError struct {
	Source string
	Char   rune
}

func (err *InvalidCharacterError) Error() string {
	return fmt.Sprintf(`invalid character '%c' in string "%s"`, err.Char, err.Source)
}

func (tag Tag) Validate() error {
	if len(tag) < 1 {
		return ErrTagEmpty
	} else if len(tag) > TagMaxLength {
		return ErrTagTooLong
	}
	for _, c := range tag { // [A-Za-z0-9-_.]
		if c >= 'A' && c <= 'Z' {
			continue
		} else if c >= 'a' && c <= 'z' {
			continue
		} else if c >= '0' && c <= '9' {
			continue
		} else if c == '-' || c == '_' || c == '.' {
			continue
		} else {
			return &InvalidCharacterError{
				Source: string(tag),
				Char:   c,
			}
		}
	}
	return nil
}

func (tag *Tag) UnmarshalJSON(b []byte) error {
	// Convert tag to lower case
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	*tag = Tag(strings.ToLower(s))
	return nil
}

type Notes string

var (
	NotesLengthMaximumCharacters = 1024

	ErrReleaseNotesTooLong  = errors.New("release notes too long")
	ErrCharactersNotAllowed = errors.New("release notes contain characters which are not allowed")
)

type InvalidCharError struct {
	Offset int
	Char   byte
}

func (err *InvalidCharError) Error() string {
	return fmt.Sprintf(`invalid character '%c' at offset %d`, err.Char, err.Offset)
}

func IsNotGraphic(r rune) bool {
	return !unicode.IsGraphic(r)
}

func (n Notes) Validate() error {
	length := len(n)
	if length > NotesLengthMaximumCharacters {
		return ErrReleaseNotesTooLong
	}
	if i := strings.IndexFunc(string(n), IsNotGraphic); i > 0 {
		return &InvalidCharError{
			Char:   n[i],
			Offset: i,
		}
	}

	return nil
}

type Release struct {
	Name      string     `json:"Name" bson:"_id"`
	Modified  *time.Time `json:"Modified,omitempty" bson:"modified,omitempty"`
	Artifacts []Image    `json:"Artifacts" bson:"artifacts"`
	Tags      Tags       `json:"tags" bson:"tags,omitempty"`
	Notes     Notes      `json:"notes" bson:"notes,omitempty"`
}

type ReleasePatch struct {
	Notes Notes `json:"notes" bson:"notes,omitempty"`
}

func (r ReleasePatch) Validate() error {
	return r.Notes.Validate()
}

type ReleaseOrImageFilter struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	DeviceType  string `json:"device_type"`
	Page        int    `json:"page"`
	PerPage     int    `json:"per_page"`
	Sort        string `json:"sort"`
}
