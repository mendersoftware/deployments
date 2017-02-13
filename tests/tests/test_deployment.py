#!/usr/bin/python
# Copyright 2017 Mender Software AS
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

from os.path import basename
from collections import OrderedDict
from uuid import uuid4
from hashlib import sha256

import bravado
import requests

from client import Client, InventoryClient
from common import artifact_from_raw_data, artifact_from_data, Device


def make_upload_meta(meta):
    order = ['description', 'size', 'artifact']

    upload_meta = OrderedDict()
    for entry in order:
        if entry in meta:
            upload_meta[entry] = meta[entry]
    return upload_meta


class TestDeployment(Client):
    def setup(self):
        self.setup_swagger()

    def test_deployments_get(self):
        res = self.client.deployments.get_deployments(Authorization='foo').result()
        self.log.debug('result: %s', res)

        # try with bogus image ID
        try:
            res = self.client.deployments.get_deployments_id(Authorization='foo',
                                                         id='foo').result()
        except bravado.exception.HTTPError as e:
            assert e.response.status_code == 400
        else:
            raise AssertionError('expected to fail')

    def test_deployments_new_bogus(self):

        # NOTE: cannot make requests with arbitary data through swagger client,
        # so we'll use requests directly instead
        rsp = requests.post(self.make_api_url('/deployments'), data='foobar')
        assert 400 <= rsp.status_code < 500
        # some broken JSON now
        rsp = requests.post(self.make_api_url('/deployments'), data='{"foo": }',
                            headers={'Content-Type': 'application/json'})
        assert 400 <= rsp.status_code < 500

        NewDeployment = self.client.get_model('NewDeployment')

        baddeps = [
            NewDeployment(name='foobar', artifact_name='someartifact', devices=[]),
            NewDeployment(name='', artifact_name='someartifact', devices=['foo']),
            NewDeployment(name='adad', artifact_name='', devices=['foo']),
            NewDeployment(name='', artifact_name='', devices=['foo']),
        ]
        for newdep in baddeps:
            # try bogus image data
            try:
                res = self.client.deployments.post_deployments(Authorization='foo',
                                                               deployment=newdep).result()
            except bravado.exception.HTTPError as e:
                assert e.response.status_code == 400
            else:
                raise AssertionError('expected to fail')

    def test_deployments_new_valid(self):
        dev = Device()

        inv = InventoryClient()
        inv.report_attributes(dev.fake_token, [
            {
                'name': 'device_type',
                'value': 'hammer',
            },
        ])
