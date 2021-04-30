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


from bson.objectid import ObjectId
from common import api_client_int


class TestInternalApiStorageSettings:
    def test_ok(self, api_client_int):
        tenant_id = str(ObjectId())
        data = {
            "region": "region",
            "bucket": "bucket",
            "uri":    "https://example.com",
            "key":    "long_key",
            "secret": "secret",
            "token":  "token"
        }
        api_client_int.set_settings(tenant_id, data)
        rx_data = api_client_int.get_settings(tenant_id)
        assert data == rx_data

    def test_data_update(self, api_client_int):
        tenant_id = str(ObjectId())
        data1 = {
            "region": "region",
            "bucket": "bucket",
            "uri":    "https://example.com",
            "key":    "long_key",
            "secret": "secret",
            "token":  "token"
        }
        data2 = {
            "region": "region",
            "bucket": "new_bucket",
            "uri":    "https://example.com",
            "key":    "long_key",
            "secret": "secret",
            "token":  "token"
        }
        api_client_int.set_settings(tenant_id, data1)
        api_client_int.set_settings(tenant_id, data2)

    def test_update_to_empty_data_set(self, api_client_int):
        tenant_id = str(ObjectId())
        data1 = {
            "region": "region",
            "bucket": "bucket",
            "uri":    "https://example.com",
            "key":    "long_key",
            "secret": "secret",
            "token":  "token"
        }
        data2 = {
            "region": "",
            "bucket": "",
            "uri":    "",
            "key":    "",
            "secret": "",
            "token":  ""
        }
        api_client_int.set_settings(tenant_id, data1)
        api_client_int.set_settings(tenant_id, data2)

    def test_failed_data_key_length(self, api_client_int):
        tenant_id = str(ObjectId())
        # 'Key' is too short
        data = {
            "region": "region",
            "bucket": "bucket",
            "uri":    "https://example.com",
            "key":    "key",
            "secret": "secret",
            "token":  "token"
        }
        api_client_int.set_settings(tenant_id, data, 500)

    def test_failed_data_missing_bucket(self, api_client_int):
        tenant_id = str(ObjectId())
        # 'Bucket' key is missing
        data = {
            "region": "region",
            "uri":    "https://example.com",
            "key":    "long_key",
            "secret": "secret",
            "token":  "token"
        }
        api_client_int.set_settings(tenant_id, data, 400)
