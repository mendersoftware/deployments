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

package model

import (
	"context"

	"github.com/pkg/errors"

	"github.com/mendersoftware/deployments/resources/limits"
)

type LimitsModel struct {
	storage LimitsStorage
}

func NewLimitsModel(storage LimitsStorage) *LimitsModel {
	return &LimitsModel{
		storage: storage,
	}
}

func (lm *LimitsModel) GetLimit(ctx context.Context, name string) (*limits.Limit, error) {
	limit, err := lm.storage.GetLimit(ctx, name)
	if err == ErrLimitNotFound {
		return &limits.Limit{
			Name:  name,
			Value: 0,
		}, nil

	} else if err != nil {
		return nil, errors.Wrap(err, "failed to obtain limit from storage")
	}
	return limit, nil
}
