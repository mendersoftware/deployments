// Copyright 2019 Northern.tech AS
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
	"fmt"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/pkg/errors"
)

type LogMessage struct {
	Timestamp *time.Time `json:"timestamp" valid:"required"`
	Level     string     `json:"level" valid:"required"`
	Message   string     `json:"message" valid:"required"`
}

type DeploymentLog struct {
	// skip these 2 field when (un)marshalling to/from JSON
	DeviceID     string `json:"-" valid:"required"`
	DeploymentID string `json:"-" valid:"uuidv4,required"`

	Messages []LogMessage `json:"messages" valid:"required"`
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

	l.Timestamp = alm.Timestamp
	l.Level = alm.Level
	l.Message = alm.Message

	if err := l.Validate(); err != nil {
		return err
	}
	return nil
}

func (l LogMessage) Validate() error {
	_, err := govalidator.ValidateStruct(l)
	return err
}

func (l LogMessage) String() string {
	return fmt.Sprintf("%s %s: %s", l.Timestamp.UTC().String(), l.Level, l.Message)
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

func (d DeploymentLog) Validate() error {
	_, err := govalidator.ValidateStruct(d)
	return err
}
