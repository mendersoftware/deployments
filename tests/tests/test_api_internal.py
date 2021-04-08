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

from uuid import uuid4

from bson.objectid import ObjectId
from common import (
    api_client_int,
    artifact_from_data,
    mongo,
    clean_db,
    clean_minio,
)
from client import SimpleArtifactsClient, ArtifactsClientError


class TestInternalApiTenantCreate:
    def test_create_ok(self, api_client_int, clean_db):
        _, r = api_client_int.create_tenant("foobar")
        assert r.status_code == 201

        assert "deployment_service-foobar" in clean_db.list_database_names()
        assert (
            "migration_info"
            in clean_db["deployment_service-foobar"].list_collection_names()
        )

    def test_create_twice(self, api_client_int, clean_db):
        _, r = api_client_int.create_tenant("foobar")
        assert r.status_code == 201

        # creating once more should not fail
        _, r = api_client_int.create_tenant("foobar")
        assert r.status_code == 201

    def test_create_empty(self, api_client_int):
        try:
            _, r = api_client_int.create_tenant("")
        except bravado.exception.HTTPError as e:
            assert e.response.status_code == 400

    @pytest.mark.usefixtures("clean_minio")
    def test_artifacts_valid(self, api_client_int, mongo):
        artifact_name = str(uuid4())
        description = "description for foo " + artifact_name
        device_type = "project-" + str(uuid4())
        data = b"foo_bar"

        tenant_id = str(ObjectId())
        _, r = api_client_int.create_tenant(tenant_id)
        assert r.status_code == 201

        # generate artifact
        with artifact_from_data(
            name=artifact_name, data=data, devicetype=device_type
        ) as art:
            artifacts_client = SimpleArtifactsClient()

            artifacts_client.log.info("uploading artifact")
            artid = api_client_int.add_artifact(
                tenant_id, description, art.size, art
            )
            assert artid is not None

            # verify the artifact has been stored correctly in mongodb
            artifact = mongo[
                "deployment_service-{}".format(tenant_id)
            ].images.find_one({"_id": artid})
            assert artifact is not None
            #
            assert artifact["_id"] == artid
            assert artifact["meta_artifact"]["name"] == artifact_name
            assert artifact["meta"]["description"] == description
            assert artifact["size"] == int(art.size)
            assert (
                device_type
                in artifact["meta_artifact"]["device_types_compatible"]
            )
            assert len(artifact["meta_artifact"]["updates"]) == 1
            update = artifact["meta_artifact"]["updates"][0]
            assert len(update["files"]) == 1
            uf = update["files"][0]
            assert uf["size"] == len(data)
            assert uf["checksum"]

    @pytest.mark.usefixtures("clean_minio")
    def test_artifacts_fails_invalid_artifact_id(self, api_client_int):
        artifact_name = str(uuid4())
        description = "description for foo " + artifact_name
        device_type = "project-" + str(uuid4())
        data = b"foo_bar"

        tenant_id = str(ObjectId())

        # generate artifact
        with artifact_from_data(
            name=artifact_name, data=data, devicetype=device_type
        ) as art:
            artifacts_client = SimpleArtifactsClient()

            artifacts_client.log.info("uploading artifact")
            with pytest.raises(ArtifactsClientError):
                api_client_int.add_artifact(
                    tenant_id, description, -1, art, "wrong_uuid4"
                )
