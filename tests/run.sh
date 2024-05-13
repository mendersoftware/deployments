#!/bin/bash -x
# Copyright 2022 Northern.tech AS
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

# tests are supposed to be located in the same directory as this file

DIR=$(readlink -f $(dirname $0))

pip install boto3 # FIXME

export PYTHONDONTWRITEBYTECODE=1
export AWS_ENDPOINT_URL="http://minio:9000"
export AWS_ACCESS_KEY_ID="minio"
export AWS_SECRET_ACCESS_KEY="minio123"

HOST=${HOST="mender-deployments:8080"}
INVENTORY_HOST=${INVENTORY_HOST="mender-inventory:8080"}

# if we're running in a container, wait a little before starting tests
[ $$ -eq 1 ] && {
    echo "-- running in container, wait for other services"
    sleep 10
}

# some additional test binaries can be located in tests directory (eg.
# mender-artifact)
export PATH=$PATH:$DIR

py.test -s --tb=short --host $HOST \
          --inventory-host $INVENTORY_HOST \
          --spec $DIR/management_api.yml \
          --device-spec $DIR/devices_api.yml \
          --internal-spec $DIR/internal_api.yml \
          --mongo-url "mongodb://mender-mongo" \
          --s3-bucket "mender-artifact-storage" \
          --s3-key-id "minio" \
          --s3-secret-key "minio123" \
          --s3-endpoint-url="http://minio:9000" \
          --verbose --junitxml=$DIR/results.xml \
          $DIR/tests/test_*.py "$@"
