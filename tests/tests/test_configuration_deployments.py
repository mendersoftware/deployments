#!/usr/bin/python
# Copyright 2021 Northern.tech AS
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.

import bravado
import pytest
import requests
import os

from uuid import uuid4

from bson.objectid import ObjectId
from common import (
    api_client_int,
    mongo,
    clean_db,
)


class TestInternalApiPostConfigurationDeployment:
    def test_ok(self, api_client_int, clean_db, mongo):
        tenant_id = str(ObjectId())
        _, r = api_client_int.create_tenant(tenant_id)
        assert r.status_code == 201

        deployment_id = "foo"
        deployment_id = str(uuid4())
        device_id = "bar"
        url = api_client_int.make_api_url(
            "/tenants/{}/configuration/deployments/{}/devices/{}".format(
                tenant_id, deployment_id, device_id
            )
        )
        configuration_deployment = {"name": "foo", "configuration": '{"foo":"bar"}'}
        rsp = requests.post(url, json=configuration_deployment, verify=False)
        assert rsp.status_code == 201
        loc = rsp.headers.get("Location", None)
        assert loc
        api_deployment_id = os.path.basename(loc)
        assert api_deployment_id == deployment_id

        # verify the deployment has been stored correctly in mongodb
        deployment = mongo[
            "deployment_service-{}".format(tenant_id)
        ].deployments.find_one({"_id": deployment_id})
        assert deployment is not None
        assert deployment["type"] == "configuration"
        assert deployment["configuration"]

    def test_fail_missing_name(self, api_client_int, clean_db):
        tenant_id = str(ObjectId())
        _, r = api_client_int.create_tenant(tenant_id)
        assert r.status_code == 201

        deployment_id = "foo"
        deployment_id = str(uuid4())
        device_id = "bar"
        url = api_client_int.make_api_url(
            "/tenants/{}/configuration/deployments/{}/devices/{}".format(
                tenant_id, deployment_id, device_id
            )
        )
        configuration_deployment = {"configuration": '{"foo":"bar"}'}
        rsp = requests.post(url, json=configuration_deployment, verify=False)
        assert rsp.status_code == 400

    def test_fail_missing_configuration(self, api_client_int, clean_db):
        tenant_id = str(ObjectId())
        _, r = api_client_int.create_tenant(tenant_id)
        assert r.status_code == 201

        deployment_id = "foo"
        deployment_id = str(uuid4())
        device_id = "bar"
        url = api_client_int.make_api_url(
            "/tenants/{}/configuration/deployments/{}/devices/{}".format(
                tenant_id, deployment_id, device_id
            )
        )
        configuration_deployment = {"name": "foo"}
        rsp = requests.post(url, json=configuration_deployment, verify=False)
        assert rsp.status_code == 400

    def test_fail_wrong_deployment_id(self, api_client_int, clean_db):
        tenant_id = str(ObjectId())
        _, r = api_client_int.create_tenant(tenant_id)
        assert r.status_code == 201

        deployment_id = "foo"
        deployment_id = "baz"
        device_id = "bar"
        url = api_client_int.make_api_url(
            "/tenants/{}/configuration/deployments/{}/devices/{}".format(
                tenant_id, deployment_id, device_id
            )
        )
        configuration_deployment = {"name": "foo", "configuration": '{"foo":"bar"}'}
        rsp = requests.post(url, json=configuration_deployment, verify=False)
        assert rsp.status_code == 400

    def test_fail_duplicate_deployment(self, api_client_int, clean_db):
        tenant_id = str(ObjectId())
        _, r = api_client_int.create_tenant(tenant_id)
        assert r.status_code == 201

        deployment_id = "foo"
        deployment_id = str(uuid4())
        device_id = "bar"
        url = api_client_int.make_api_url(
            "/tenants/{}/configuration/deployments/{}/devices/{}".format(
                tenant_id, deployment_id, device_id
            )
        )
        configuration_deployment = {"name": "foo", "configuration": '{"foo":"bar"}'}
        rsp = requests.post(url, json=configuration_deployment, verify=False)
        assert rsp.status_code == 201
        loc = rsp.headers.get("Location", None)
        assert loc
        api_deployment_id = os.path.basename(loc)
        assert api_deployment_id == deployment_id

        rsp = requests.post(url, json=configuration_deployment, verify=False)
        assert rsp.status_code == 409
