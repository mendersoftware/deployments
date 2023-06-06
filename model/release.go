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

import "time"

type Release struct {
	Name      string     `json:"Name" bson:"name"`
	Modified  *time.Time `json:"Modified" bson:"modified"`
	Artifacts []Image    `json:"Artifacts" bson:"artifacts"`
}

type ReleaseOrImageFilter struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	DeviceType  string `json:"device_type"`
	Page        int    `json:"page"`
	PerPage     int    `json:"per_page"`
	Sort        string `json:"sort"`
}
