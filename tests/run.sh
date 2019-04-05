#!/bin/bash -x

# tests are supposed to be located in the same directory as this file

DIR=$(readlink -f $(dirname $0))

export PYTHONDONTWRITEBYTECODE=1

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

ls -asl $DIR
echo $PATH
mender-artifact
file $DIR/mender-artifact

py.test -s --tb=short --host $HOST \
          --inventory-host $INVENTORY_HOST \
          --spec $DIR/management_api.yml \
          --device-spec $DIR/devices_api.yml \
          --internal-spec $DIR/internal_api.yml \
          --verbose --junitxml=$DIR/results.xml \
          $DIR/tests/test_*.py "$@"
