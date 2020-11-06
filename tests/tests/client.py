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
import os.path
import logging
import random
import subprocess

from datetime import datetime
from collections import OrderedDict
from contextlib import contextmanager

import requests
import pytest
import pytz

from bravado.swagger_model import load_file
from bravado.client import SwaggerClient, RequestsClient
from bravado.exception import HTTPUnprocessableEntity

DEPLOYMENTS_BASE_URL = "http://{}/api/{}/v1/deployments"


class BaseApiClient:

    def __init__(self, request):
        self.api_url = DEPLOYMENTS_BASE_URL.format(request.config.getoption("host"), "management")

    def make_api_url(self, path=None):
        if path is not None:
            return os.path.join(
                self.api_url, path if not path.startswith("/") else path[1:]
            )
        return self.api_url


class RequestsApiClient(requests.Session):
    # TODO: convert to make_session() helper
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.verify = False


class SwaggerApiClient(BaseApiClient):
    """Class that is based on swagger spec. Swagger support is initialized on call
    to setup_swagger(). Class has no constructor, hence can be used with Pytest"""

    config = {
        "also_return_response": True,
        "validate_responses": True,
        "validate_requests": False,
        "validate_swagger_spec": False,
        "use_models": True,
    }

    log = logging.getLogger("client.Client")
    spec_option = "spec"

    def __init__(self, request):
        self.spec = request.config.getoption(self.spec_option)

    def setup_swagger(self):
        self.http_client = RequestsClient()
        self.http_client.session.verify = False

        self.client = SwaggerClient.from_spec(
            load_file(self.spec), config=self.config, http_client=self.http_client
        )
        self.client.swagger_spec.api_url = self.make_api_url()


class ArtifactsClientError(Exception):
    def __init__(self, message="", response=None):
        self.response = response
        super().__init__(message)


class ArtifactsClient(SwaggerApiClient):

    def __init__(self, request):
        self.api_url = DEPLOYMENTS_BASE_URL.format(request.config.getoption("host"), "management")

    @staticmethod
    def make_upload_meta(meta):
        order = ["description", "size", "artifact_id", "artifact"]

        upload_meta = OrderedDict()
        for entry in order:
            if entry in meta:
                upload_meta[entry] = meta[entry]
        return upload_meta

    def add_artifact(self, description="", size=0, data=None):
        """Create new artifact with provided upload data. Data must be a file like
        object.

        Returns artifact ID or raises ArtifactsClientError if response checks
        failed
        """
        # prepare upload data for multipart/form-data
        files = ArtifactsClient.make_upload_meta(
            {
                "description": (None, description),
                "size": (None, str(size)),
                "artifact": ("firmware", data, "application/octet-stream", {}),
            }
        )
        rsp = requests.post(self.make_api_url("/artifacts"), files=files, verify=False)
        # should have be created
        try:
            assert rsp.status_code == 201
            loc = rsp.headers.get("Location", None)
            assert loc
        except AssertionError:
            raise ArtifactsClientError("add failed", rsp)

        loc = rsp.headers.get("Location", None)
        artid = os.path.basename(loc)
        return artid

    @staticmethod
    def make_generate_meta(meta):
        order = [
            "name",
            "description",
            "device_types_compatible",
            "type",
            "args",
            "file",
        ]

        upload_meta = OrderedDict()
        for entry in order:
            if entry in meta:
                upload_meta[entry] = meta[entry]
        return upload_meta

    def generate_artifact(
        self,
        name="",
        description="",
        device_types_compatible="",
        type="",
        args="",
        data=None,
    ):
        """Generate a new artifact with provided upload data.
        Data must be a file like object.

        Returns artifact ID or raises ArtifactsClientError if response checks
        failed
        """
        # prepare upload data for multipart/form-data
        files = ArtifactsClient.make_generate_meta(
            {
                "name": (None, name),
                "description": (None, description),
                "device_types_compatible": (None, device_types_compatible),
                "type": (None, type),
                "args": (None, args),
                "file": ("firmware", data, "application/octet-stream", {}),
            }
        )
        rsp = requests.post(
            self.make_api_url("/artifacts/generate"), files=files, verify=False
        )
        # should have be created
        try:
            assert rsp.status_code == 201
            loc = rsp.headers.get("Location", None)
            assert loc
        except AssertionError:
            raise ArtifactsClientError("add failed", rsp)

        loc = rsp.headers.get("Location", None)
        artid = os.path.basename(loc)
        return artid

    def delete_artifact(self, artid=""):
        # delete it now (NOTE: not using bravado as bravado does not support
        # DELETE)
        rsp = requests.delete(
            self.make_api_url("/artifacts/{}".format(artid)), verify=False
        )
        try:
            assert rsp.status_code == 204
        except AssertionError:
            raise ArtifactsClientError("delete failed", rsp)

    @contextmanager
    def with_added_artifact(self, description="", size=0, data=None):
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

    def __init__(self, request):
        self.api_url = DEPLOYMENTS_BASE_URL.format(pytest.config.getoption("host"), "management")


    def make_new_deployment(self, *args, **kwargs):
        NewDeployment = self.client.get_model("NewDeployment")
        return NewDeployment(*args, **kwargs)

    def add_deployment(self, dep):
        """Posts new deployment `dep`"""
        res = self.client.Management_API.Create_Deployment(
            Authorization="foo", deployment=dep
        ).result()
        adapter = res[1]
        loc = adapter.headers.get("Location", None)
        depid = os.path.basename(loc)

        self.log.debug("added new deployment with ID: %s", depid)
        return depid

    def abort_deployment(self, depid):
        """Abort deployment with `ID `depid`"""
        self.client.Management_API.Abort_Deployment(
            Authorization="foo", deployment_id=depid, Status={"status": "aborted"},
        ).result()

    @contextmanager
    def with_added_deployment(self, dep):
        """Acts as a context manager, adds artifact and yields artifact ID and deletes
        it upon completion"""
        depid = self.add_deployment(dep)
        yield depid
        try:
            self.abort_deployment(depid)
        except HTTPUnprocessableEntity:
            self.log.warning("deployment: %s already finished", depid)

    def verify_deployment_stats(self, depid, expected):
        stats = self.client.Management_API.Deployment_Status_Statistics(
            Authorization="foo", deployment_id=depid
        ).result()[0]
        stat_names = [
            "success",
            "pending",
            "failure",
            "downloading",
            "installing",
            "rebooting",
            "noartifact",
            "already-installed",
            "aborted",
        ]
        for s in stat_names:
            exp = expected.get(s, 0)
            current = getattr(stats, s) or 0
            assert exp == current


class DeviceClient(SwaggerApiClient):
    """Swagger based device API client. Can be used a Pytest base class"""

    spec_option = "device_spec"
    logger_tag = "client.DeviceClient"

    def __init__(self, request):
        self.api_url = DEPLOYMENTS_BASE_URL.format(request.config.getoption("host"), "devices")


    def get_next_deployment(self, token="", artifact_name="", device_type=""):
        """Obtain next deployment"""
        auth = "Bearer " + token
        res = self.client.Device_API.Check_Update(
            Authorization=auth, artifact_name=artifact_name, device_type=device_type,
        ).result()[0]
        return res

    def report_status(self, token="", devdepid=None, status=None):
        """Report device deployment status"""
        auth = "Bearer " + token
        res = self.client.Device_API.Update_Deployment_Status(
            Authorization=auth, id=devdepid, Status={"status": status}
        ).result()
        return res

    def upload_logs(self, token="", devdepid=None, logs=[]):
        auth = "Bearer " + token
        DeploymentLog = self.client.get_model("DeploymentLog")
        levels = ["info", "debug", "warn", "error", "other"]
        dl = DeploymentLog(
            messages=[
                {
                    "timestamp": pytz.utc.localize(datetime.now()),
                    "level": random.choice(levels),
                    "message": l,
                }
                for l in logs
            ]
        )
        res = self.client.Device_API.Report_Deployment_Log(
            Authorization=auth, id=devdepid, Log=dl
        ).result()
        return res


class SimpleDeviceClient(DeviceClient):
    """Simple device API client, cannot be used as Pytest tests base class"""

    def __init__(self):
        self.setup_swagger()


class InventoryClientError(Exception):
    pass


class InventoryClient(BaseApiClient, RequestsApiClient):


    def __init__(self, request):
        self.api_url = "http://%s/api/0.1.0/" % (pytest.config.getoption("inventory_host"))

    def report_attributes(self, devtoken, attributes):
        """Send device attributes to inventory service. Device is identified using
        authorization token passed in `devtoken`. Attributes can be a dict, a
        list, or anything else that can be serialized to JSON. Will raise
        InventoryClientError if request fails.

        """
        rsp = requests.patch(
            self.make_api_url("/attributes"),
            headers={"Authorization": "Bearer " + devtoken},
            json=attributes,
        )
        if rsp.status_code != 200:
            raise InventoryClientError(
                "request failed with status code {}".format(rsp.status_code)
            )


class CliClient:
    cmd = "/testing/deployments"

    def migrate(self, tenant=None):
        args = [self.cmd, "migrate"]

        if tenant is not None:
            args.extend(["--tenant", tenant])

        subprocess.run(args, check=True)


class InternalApiClient(SwaggerApiClient):
    spec_option = "internal_spec"
    logger_tag = "client.InternalApiClient"

    def __init__(self, request):
        self.api_url = DEPLOYMENTS_BASE_URL.format(request.config.getoption("host"), "internal")
        self.spec = request.config.getoption("spec")
        self.setup_swagger()

    def create_tenant(self, tenant_id):
        return self.client.tenants.Create_Tenant(
            tenant={"tenant_id": tenant_id}
        ).result()

    def add_artifact(
        self, tenant_id, description="", size=0, data=None, artifact_id=None
    ):
        """Create new artifact with provided upload data. Data must be a file like
        object.

        Returns artifact ID or raises ArtifactsClientError if response checks
        failed
        """
        # prepare upload data for multipart/form-data
        files = ArtifactsClient.make_upload_meta(
            {
                "artifact_id": artifact_id,
                "description": (None, description),
                "size": (None, str(size)),
                "artifact": ("firmware", data, "application/octet-stream", {}),
            }
        )
        url = self.make_api_url("/tenants/{}/artifacts".format(tenant_id))
        rsp = requests.post(url, files=files, verify=False)
        # should have been created
        try:
            assert rsp.status_code == 201
            loc = rsp.headers.get("Location", None)
            assert loc
        except AssertionError:
            raise ArtifactsClientError("add failed", rsp)
        # return the artifact id
        loc = rsp.headers.get("Location", None)
        artid = os.path.basename(loc)
        return artid
