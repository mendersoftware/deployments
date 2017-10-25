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

	"github.com/mendersoftware/deployments/resources/tenants/store"
)

type Model interface {
	ProvisionTenant(ctx context.Context, tenant_id string) error
}

type model struct {
	store store.Store
}

func NewModel(store store.Store) *model {
	return &model{
		store: store,
	}
}

func (m *model) ProvisionTenant(ctx context.Context, tenant_id string) error {
	if err := m.store.ProvisionTenant(ctx, tenant_id); err != nil {
		return errors.Wrap(err, "failed to provision tenant")
	}

	return nil
}
