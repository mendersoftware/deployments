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
import tempfile
import subprocess
import logging
import os
import abc
import random
import string
import json
import pytest

from hashlib import sha256
from contextlib import contextmanager
from base64 import urlsafe_b64encode
from client import CliClient
from pymongo import MongoClient

DB_NAME = "deployment_service"
DB_MIGRATION_COLLECTION = "migration_info"
DB_VERSION = "1.2.1"

class Artifact(metaclass=abc.ABCMeta):
    @abc.abstractproperty
    def size(self):
        pass

    @abc.abstractproperty
    def checksum(self):
        pass


class BytesArtifact(io.BytesIO, Artifact):

    def __init__(self, data):
        self._size = len(data)
        d = sha256()
        d.update(data)
        self._checksum = d.hexdigest()

        super().__init__(data)

    @property
    def size(self):
        return self._size

    @property
    def checksum(self):
        return self._checksum


class FileArtifact(io.RawIOBase, Artifact):
    def __init__(self, size, openedfile):
        self.file = openedfile
        self._size = size

        d = sha256()
        with open(openedfile.name, 'rb') as inf:
            fdata = inf.read()
            if fdata:
                d.update(fdata)

        self._checksum = d.hexdigest()

    def read(self, *args):
        return self.file.read(*args)

    @property
    def size(self):
        return self._size

    @property
    def checksum(self):
        return self._checksum


@contextmanager
def artifact_from_mender_file(path):
    with open(path, 'rb') as infile:
        sz = str(os.stat(path).st_size)
        yield FileArtifact(sz, infile)


@contextmanager
def artifact_from_raw_data(data):
    if type(data) is str:
        data = data.encode()
    yield BytesArtifact(data)


@contextmanager
def artifact_from_data(name='foo', data=None, devicetype='hammer'):
    with tempfile.NamedTemporaryFile(prefix='menderout') as tmender:
        logging.info('writing mender artifact to temp file %s', tmender.name)

        with tempfile.NamedTemporaryFile(prefix='menderin') as tdata:
            logging.info('writing update data to temp file %s', tdata.name)
            tdata.write(data)
            tdata.flush()

            cmd = 'mender-artifact write rootfs-image --device-type "{}" ' \
                  '--update "{}" --artifact-name "{}" --output-path "{}"'.format(
                      devicetype,
                      tdata.name,
                      name,
                      tmender.name,
                  )
            rc = subprocess.call(cmd, shell=True)
            if rc:
                logging.error('mender-artifact call \'%s\' failed with code %d', cmd, rc)
                raise RuntimeError('mender-artifact command \'{}\' failed with code {}'.format(cmd, rc))

            # bring up temp mender artifact
            with artifact_from_mender_file(tmender.name) as fa:
                yield fa


class Device:
    def __init__(self, device_type='hammer'):
        self.devid = ''.join([random.choice(string.ascii_letters + string.digits) \
                              for _ in range(10)])
        self.device_type = device_type

    @property
    def fake_token(self):
        claims = json.dumps({
            'sub': self.devid,
            'iss': 'Mender',
        })
        hdr = '{"typ": "JWT"}'
        signature = 'fake-signature'
        return '.'.join(urlsafe_b64encode(p.encode()).decode() \
                        for p in [hdr, claims, signature])

@pytest.fixture(scope="session")
def cli():
    return CliClient()

@pytest.fixture(scope="session")
def mongo():
    return MongoClient('mender-mongo-deployments:27017')

@pytest.yield_fixture(scope='function')
def clean_db(mongo):
    mongo_cleanup(mongo)
    yield
    mongo_cleanup(mongo)

def mongo_cleanup(mongo):
    dbs = mongo.database_names()
    dbs = [d for d in dbs if d not in ['local', 'admin']]
    for d in dbs:
        mongo.drop_database(d)

def make_tenant_db(tenant_id):
    return '{}-{}'.format(DB_NAME, tenant_id)
