// Copyright 2016 Mender Software AS
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

package deployments

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

type LogMessage struct {
	Timestamp *time.Time `json:"timestamp"`
	Level     string     `json:"level"`
	Message   string     `json:"message"`
}

type DeploymentLog struct {
	// skip these 2 field when (un)marshalling to/from JSON
	DeviceID     string `json:"-"`
	DeploymentID string `json:"-"`

	Messages []LogMessage `json:"messages"`
}

var (
	ErrInvalidDeploymentLog = errors.New("invalid deployment log")
	ErrInvalidLogMessage    = errors.New("invalid log message")
)

func (l *LogMessage) UnmarshalJSON(raw []byte) error {
	type AuxLogMessage LogMessage

	var alm AuxLogMessage

	if err := json.Unmarshal(raw, &alm); err != nil {
		return err
	}

	if alm.Timestamp == nil {
		return errors.Wrapf(ErrInvalidLogMessage, "no timestamp")
	}

	if alm.Level == "" {
		return errors.Wrapf(ErrInvalidLogMessage, "empty level")
	}

	if alm.Message == "" {
		return errors.Wrapf(ErrInvalidLogMessage, "empty message")
	}

	l.Timestamp = alm.Timestamp
	l.Level = alm.Level
	l.Message = alm.Message
	return nil
}

func (d *DeploymentLog) UnmarshalJSON(raw []byte) error {
	type AuxDeploymentLog DeploymentLog

	var adl AuxDeploymentLog

	if err := json.Unmarshal(raw, &adl); err != nil {
		return err
	}

	if adl.Messages == nil || len(adl.Messages) == 0 {
		return errors.Wrapf(ErrInvalidDeploymentLog, "no messages")
	}

	d.Messages = adl.Messages
	return nil
}
