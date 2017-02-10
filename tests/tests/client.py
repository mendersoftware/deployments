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

import requests
import pytest

from bravado.swagger_model import load_file
from bravado.client import SwaggerClient, RequestsClient


API_URL = "http://%s/api/%s/" % \
          (pytest.config.getoption("host"), \
           pytest.config.getoption("api"))


class Client:

    config = {
        'also_return_response': True,
        'validate_responses': True,
        'validate_requests': False,
        'validate_swagger_spec': False,
        'use_models': True,
    }

    logger_tag = 'client.Client'
    spec_option = 'spec'

    def setup(self):
        self.log = logging.getLogger(self.logger_tag)
        self.api_url = API_URL
        self.http_client = RequestsClient()
        self.http_client.session.verify = False

        spec = pytest.config.getoption(self.spec_option)
        self.client = SwaggerClient.from_spec(load_file(spec),
                                              config=self.config,
                                              http_client=self.http_client)
        self.client.swagger_spec.api_url = self.api_url

    def make_api_url(self, path):
        return os.path.join(self.api_url,
                            path if not path.startswith("/") else path[1:])

class DeviceClient(Client):

    spec_option = 'device-spec'
    logger_tag = 'client.DeviceClient'



class InventoryClientError(Exception):
    pass


class InventoryClient(Client):

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
