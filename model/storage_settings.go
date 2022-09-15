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

package model

import (
	"encoding/json"
	"io"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pkg/errors"
)

type StorageSettings struct {
	Region         string `json:"region" bson:"region"`
	Bucket         string `json:"bucket" bson:"bucket"`
	Uri            string `json:"uri" bson:"uri"`
	ExternalUri    string `json:"external_uri" bson:"external_uri"`
	Key            string `json:"key" bson:"key"`
	Secret         string `json:"secret" bson:"secret"`
	Token          string `json:"token" bson:"token"`
	ForcePathStyle bool   `json:"force_path_style" bson:"force_path_style"`
	UseAccelerate  bool   `json:"use_accelerate" bson:"use_accelerate"`
}

func ParseStorageSettingsRequest(source io.Reader) (*StorageSettings, error) {
	var s StorageSettings

	if err := json.NewDecoder(source).Decode(&s); err != nil {
		return nil, err
	}

	if s.Region != "" || s.Bucket != "" || s.Key != "" || s.Secret != "" {
		keys := []string{s.Region, s.Bucket, s.Key, s.Secret}
		for _, k := range keys {
			if k == "" {
				return nil, errors.New("Invalid input data.")
			}
		}
	}

	return &s, nil
}

// Validate checks structure according to valid tags
func (s StorageSettings) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.Region, validation.Required, validation.Length(5, 20)),
		validation.Field(&s.Bucket, validation.Required, validation.Length(5, 100)),
		validation.Field(&s.Key, validation.Required, validation.Length(5, 50)),
		validation.Field(&s.Secret, validation.Required, validation.Length(5, 100)),
		validation.Field(&s.Uri, validation.Length(3, 2000)),
		validation.Field(&s.ExternalUri, validation.Length(3, 2000)),
		validation.Field(&s.Token, validation.Length(5, 100)),
	)
}
