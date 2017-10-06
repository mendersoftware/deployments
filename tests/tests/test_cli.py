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
import pytest

@pytest.fixture(scope='function')
def migrated_db(clean_db, mongo, request):
    ''' Init a default db to version passed in 'request'. '''
    version = request.param
    mongo_set_version(mongo, DB_NAME, version)


MIGRATED_TENANT_DBS={
    "tenant-stale-1": "0.0.1",
    "tenant-stale-2": "0.2.0",
    "tenant-stale-3": "1.0.0",
    "tenant-current": "1.1.0",
    "tenant-future": "2.0.0",
}

@pytest.fixture(scope='function')
def migrated_tenant_dbs(clean_db, mongo):
    ''' Init a set of tenant dbs to predefined versions. '''
    for tid, ver in MIGRATED_TENANT_DBS.items():
        mongo_set_version(mongo, make_tenant_db(tid), ver)

def mongo_set_version(mongo, dbname, version):
    major, minor, patch = [int(x) for x in version.split('.')]

    version = {
        "major": major,
        "minor": minor,
        "patch": patch,
    }

    mongo[dbname][DB_MIGRATION_COLLECTION].insert_one({"version": version})
