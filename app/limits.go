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

package app

import (
	"context"

	"github.com/pkg/errors"

	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/store"
	"github.com/mendersoftware/deployments/store/mongo"
)

type App interface {
	GetLimit(ctx context.Context, name string) (*model.Limit, error)
	ProvisionTenant(ctx context.Context, tenant_id string) error
}

type Deployments struct {
	db store.DataStore
}

func NewDeployments(storage store.DataStore) *Deployments {
	return &Deployments{
		db: storage,
	}
}

func (d *Deployments) GetLimit(ctx context.Context, name string) (*model.Limit, error) {
	limit, err := d.db.GetLimit(ctx, name)
	if err == mongo.ErrLimitNotFound {
		return &model.Limit{
			Name:  name,
			Value: 0,
		}, nil

	} else if err != nil {
		return nil, errors.Wrap(err, "failed to obtain limit from storage")
	}
	return limit, nil
}

func (d *Deployments) ProvisionTenant(ctx context.Context, tenant_id string) error {
	if err := d.db.ProvisionTenant(ctx, tenant_id); err != nil {
		return errors.Wrap(err, "failed to provision tenant")
	}

	return nil
}
