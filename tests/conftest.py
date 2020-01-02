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
import logging


def pytest_addoption(parser):
    parser.addoption(
        "--host", action="store", default="localhost", help="host running API"
    )
    parser.addoption(
        "--inventory-host",
        action="store",
        default="mender-inventory:8080",
        help="host running API",
    )
    parser.addoption("--spec", default="../docs/management_api.yml")
    parser.addoption("--device-spec", default="../docs/devices_api.yml")
    parser.addoption("--internal-spec", default="../docs/internal_api.yml")


def pytest_configure(config):
    lvl = logging.INFO
    if config.getoption("verbose"):
        lvl = logging.DEBUG
    logging.basicConfig(level=lvl)
    # configure bravado related loggers to be less verbose
    logging.getLogger("swagger_spec_validator").setLevel(logging.INFO)
    logging.getLogger("bravado_core").setLevel(logging.INFO)
