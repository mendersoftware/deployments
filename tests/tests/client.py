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
import os.path
import logging

from collections import OrderedDict
from contextlib import contextmanager

import requests
import pytest

from bravado.swagger_model import load_file
from bravado.client import SwaggerClient, RequestsClient


API_URL = "http://%s/api/%s/" % \
          (pytest.config.getoption("host"), \
           pytest.config.getoption("api"))


class BaseApiClient:
    api_url = API_URL

    def make_api_url(self, path):
        return os.path.join(self.api_url,
                            path if not path.startswith("/") else path[1:])


class RequestsApiClient(requests.Session):
    # TODO: convert to make_session() helper
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.verify = False


class SwaggerApiClient(BaseApiClient):
    """Class that it based on swagger spec. Swagger support is initialized on call
    to setup_swagger(). Class has no constructor, hence can be used with Pytest"""

    config = {
        'also_return_response': True,
        'validate_responses': True,
        'validate_requests': False,
        'validate_swagger_spec': False,
        'use_models': True,
    }

    log = logging.getLogger('client.Client')
    spec_option = 'spec'

    def setup_swagger(self):
        self.http_client = RequestsClient()
        self.http_client.session.verify = False

        spec = pytest.config.getoption(self.spec_option)
        self.client = SwaggerClient.from_spec(load_file(spec),
                                              config=self.config,
                                              http_client=self.http_client)
        self.client.swagger_spec.api_url = self.api_url


class ArtifactsClientError(Exception):
    def __init__(self, message='', response=None):
        self.response = response
        super().__init__(message)


class ArtifactsClient(SwaggerApiClient):
    @staticmethod
    def make_upload_meta(meta):
        order = ['description', 'size', 'artifact']

        upload_meta = OrderedDict()
        for entry in order:
            if entry in meta:
                upload_meta[entry] = meta[entry]
        return upload_meta


    def add_artifact(self, description='', size=0, data=None):
        """Create new artifact with provided upload data. Data must be a file like
        object.

        Returns artifact ID or raises ArtifactsClientError if response checks
        failed
        """
        # prepare upload data for multipart/form-data
        files = ArtifactsClient.make_upload_meta({
            'description': description,
            'size': str(size),
            'artifact': ('firmware', data, 'application/octet-stream', {}),
        })
        rsp = requests.post(self.make_api_url('/artifacts'), files=files, verify=False)
        # should have be created
        try:
            assert rsp.status_code == 201
            loc = rsp.headers.get('Location', None)
            assert loc
        except AssertionError:
            raise ArtifactsClientError('add failed failed', rsp)

        loc = rsp.headers.get('Location', None)
        artid = os.path.basename(loc)
        return artid

    def delete_artifact(self, artid=''):
        # delete it now (NOTE: not using bravado as bravado does not support
        # DELETE)
        rsp = requests.delete(self.make_api_url('/artifacts/{}'.format(artid)), verify=False)
        try:
            assert rsp.status_code == 204
        except AssertionError:
            raise ArtifactsClientError('delete failed', rsp)

    @contextmanager
    def with_added_artifact(self, description='', size=0, data=None):
        """Acts as a context manager, adds artifact and yields artifact ID and deletes
        it upon completion"""
        artid = self.add_artifact(description=description, size=size, data=data)
        yield artid
        self.delete_artifact(artid)


class SimpleArtifactsClient(ArtifactsClient):
    """Simple swagger based client for artifacts. Cannot be used as Pytest base class"""
    def __init__(self):
        self.setup_swagger()


class DeploymentsClient(SwaggerApiClient):
    def make_new_deployment(self, *args, **kwargs):
        NewDeployment = self.client.get_model('NewDeployment')
        return NewDeployment(*args, **kwargs)


class DeviceClient(SwaggerApiClient):
    """Swagger based device API client. Can be used a Pytest base class"""
    spec_option = 'device_spec'
    logger_tag = 'client.DeviceClient'

    def get_next_deployment(self, token='', artifact_name='', device_type=''):
        auth = 'Bearer ' + token
        res = self.client.device.get_device_deployments_next(Authorization=auth,
                                                             artifact_name=artifact_name,
                                                             device_type=device_type).result()[0]
        return res

class SimpleDeviceClient(DeviceClient):
    """Simple device API client, cannot be used as Pytest tests base class"""
    def __init__(self):
        self.setup_swagger()


class InventoryClientError(Exception):
    pass


class InventoryClient(BaseApiClient, RequestsApiClient):

    api_url = "http://%s/api/0.1.0/" % \
          (pytest.config.getoption("inventory_host"))

    def report_attributes(self, devtoken, attributes):
        """Send device attributes to inventory service. Device is identified using
        authorization token passed in `devtoken`. Attributes can be a dict, a
        list, or anything else that can be serialized to JSON. Will raise
        InventoryClientError if request fails.

        """
        rsp = requests.patch(self.make_api_url('/attributes'),
                             headers={
                                 'Authorization': 'Bearer ' + devtoken,
                             },
                             json=attributes)
        if rsp.status_code != 200:
            raise InventoryClientError(
                'request failed with status code {}'.format(rsp.status_code))
