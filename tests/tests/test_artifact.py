#!/usr/bin/python
# Copyright 2017 Northern.tech AS
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#        https://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.
import io
import pytest
from os.path import basename
from uuid import uuid4
from hashlib import sha256

import bravado
import requests

from client import ArtifactsClient
from common import artifact_from_raw_data, artifact_from_data, clean_minio, MinioClient


class TestArtifact(ArtifactsClient):
    m = MinioClient()

    def setup(self):
        self.setup_swagger()

    def test_artifacts_all(self):
        res = self.client.artifacts.get_artifacts().result()
        self.log.debug('result: %s', res)

    @pytest.mark.usefixtures("clean_minio")
    def test_artifacts_new_bogus_empty(self):
        # try bogus image data
        try:
            res = self.client.artifacts.post_artifacts(Authorization='foo',
                                                       size=100,
                                                       artifact=''.encode(),
                                                       description="bar").result()
        except bravado.exception.HTTPError as e:

            assert sum(1 for x in self.m.list_objects("mender-artifact-storage")) == 0
            assert e.response.status_code == 400
        else:
            raise AssertionError('expected to fail')

    @pytest.mark.usefixtures("clean_minio")
    def test_artifacts_new_bogus_data(self):
        with artifact_from_raw_data(b'foo_bar') as art:
            files = ArtifactsClient.make_upload_meta({
                'description': 'bar',
                'size': str(art.size),
                'artifact': ('firmware', art, 'application/octet-stream', {}),
            })

            rsp = requests.post(self.make_api_url('/artifacts'), files=files)

            assert sum(1 for x in self.m.list_objects("mender-artifact-storage")) == 0
            assert rsp.status_code == 400



    @pytest.mark.usefixtures("clean_minio")
    def test_artifacts_valid(self):
        artifact_name = str(uuid4())
        description = 'description for foo ' + artifact_name
        device_type = 'project-' + str(uuid4())
        data = b'foo_bar'

        # generate artifact
        with artifact_from_data(name=artifact_name, data=data, devicetype=device_type) as art:
            self.log.info("uploading artifact")
            artid = self.add_artifact(description, art.size, art)

            # artifacts listing should not be empty now
            res = self.client.artifacts.get_artifacts().result()
            self.log.debug('result: %s', res)
            assert len(res[0]) > 0

            res = self.client.artifacts.get_artifacts_id(Authorization='foo',
                                                         id=artid).result()[0]
            self.log.info('artifact: %s', res)

            # verify its data
            assert res.id == artid
            assert res.name == artifact_name
            assert res.description == description
            assert res.size == int(art.size)
            assert device_type in res.device_types_compatible
            assert len(res.updates) == 1
            update = res.updates[0]
            assert len(update.files) == 1
            uf = update.files[0]
            assert uf.size == len(data)
            assert uf.checksum
            # TODO: verify uf signature once it's supported
            # assert uf.signature

            # try to fetch the update
            res = self.client.artifacts.get_artifacts_id_download(Authorization='foo',
                                                                  id=artid).result()[0]
            self.log.info('download result %s', res)
            assert res.uri
            # fetch it now (disable SSL verification)
            rsp = requests.get(res.uri, verify=False, stream=True)

            assert rsp.status_code == 200
            assert sum(1 for x in self.m.list_objects("mender-artifact-storage")) == 1

            # receive artifact and compare its checksum
            dig = sha256()
            while True:
                rspdata = rsp.raw.read()
                if rspdata:
                    dig.update(rspdata)
                else:
                    break

            self.log.info('artifact checksum %s expecting %s', dig.hexdigest(), art.checksum)
            assert dig.hexdigest() == art.checksum

            # delete it now
            self.delete_artifact(artid)

            # should be unavailable now
            try:
                res = self.client.artifacts.get_artifacts_id(Authorization='foo',
                                                             id=artid).result()
            except bravado.exception.HTTPError as e:
                assert e.response.status_code == 404
            else:
                raise AssertionError('expected to fail')


    def test_single_artifact(self):
        # try with bogus image ID
        try:
            res = self.client.artifacts.get_artifacts_id(Authorization='foo',
                                                         id='foo').result()
        except bravado.exception.HTTPError as e:
            assert e.response.status_code == 400
        else:
            raise AssertionError('expected to fail')

        # try with nonexistent image ID
        try:
            res = self.client.artifacts.get_artifacts_id(Authorization='foo',
                                                         id=uuid4()).result()
        except bravado.exception.HTTPError as e:
            assert e.response.status_code == 404
        else:
            raise AssertionError('expected to fail')
