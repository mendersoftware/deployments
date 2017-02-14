#!/bin/bash -x

# tests are supposed to be located in the same directory as this file

DIR=$(readlink -f $(dirname $0))

export PYTHONDONTWRITEBYTECODE=1

HOST=${HOST="mender-deployments:8080"}
INVENTORY_HOST=${INVENTORY_HOST="mender-inventory:8080"}

# if we're running in a container, wait a little before starting tests
[ $$ -eq 1 ] && sleep 5

# some additional test binaries can be located in tests directory (eg.
# mender-artifact)
export PATH=$PATH:$DIR

py.test-3 -s --tb=short --api=0.0.1  --host $HOST \
          --inventory-host $INVENTORY_HOST \
          --spec $DIR/management_api.yml \
          --device-spec $DIR/devices_api.yml \
          --verbose --junitxml=$DIR/results.xml \
          $DIR/tests/test_*.py "$@"
