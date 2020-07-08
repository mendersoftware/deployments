#!/usr/bin/python
# Copyright 2018 Northern.tech AS
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#        https://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either exprsps or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.
import io
import pytest

import bravado
import requests

from client import DeploymentsClient
from common import (
    artifacts_added_from_data,
    clean_db,
    clean_minio,
    MinioClient,
    mongo,
)


class TestRelease(DeploymentsClient):
    m = MinioClient()

    def setup(self):
        self.setup_swagger()

    def test_releases_no_artifacts(self):
        rsp = self.client.Management_API.List_Releases(Authorization="foo").result()
        assert len(rsp[0]) == 0

    @pytest.mark.usefixtures("clean_minio", "clean_db")
    def test_get_all_releses(self):
        with artifacts_added_from_data(
            [
                ("foo", "device-type-1"),
                ("foo", "device-type-2"),
                ("bar", "device-type-2"),
            ]
        ):
            rsp = self.client.Management_API.List_Releases(Authorization="foo").result()
            res = rsp[0]
            assert len(res) == 2
            release1 = res[0]
            release2 = res[1]
            assert release1.Name == "foo"
            assert len(release1.Artifacts) == 2

            r1a1 = release1.Artifacts[0]
            r1a2 = release1.Artifacts[1]
            assert r1a1["name"] == "foo"
            assert r1a1["device_types_compatible"] == ["device-type-1"]
            assert r1a2["name"] == "foo"
            assert r1a2["device_types_compatible"] == ["device-type-2"]

            assert release2.Name == "bar"
            assert len(release2.Artifacts) == 1
            r2a = release2.Artifacts[0]
            assert r2a["name"] == "bar"
            assert r2a["device_types_compatible"] == ["device-type-2"]

    @pytest.mark.usefixtures("clean_minio", "clean_db")
    def test_get_releses_by_name(self):
        with artifacts_added_from_data(
            [
                ("foo", "device-type-1"),
                ("foo", "device-type-2"),
                ("bar", "device-type-2"),
            ]
        ):
            rsp = self.client.Management_API.List_Releases(
                Authorization="foo", name="bar"
            ).result()
            res = rsp[0]
            assert len(res) == 1
            release = res[0]
            assert release.Name == "bar"
            assert len(release.Artifacts) == 1
            artifact = release.Artifacts[0]
            assert artifact["name"] == "bar"
            assert artifact["device_types_compatible"] == ["device-type-2"]

    @pytest.mark.usefixtures("clean_minio", "clean_db")
    def test_get_releses_by_name_no_result(self):
        with artifacts_added_from_data(
            [
                ("foo", "device-type-1"),
                ("foo", "device-type-2"),
                ("bar", "device-type-2"),
            ]
        ):
            rsp = self.client.Management_API.List_Releases(
                Authorization="foo", name="baz"
            ).result()
            res = rsp[0]
            assert len(res) == 0
