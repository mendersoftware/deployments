#!/bin/bash

if [ ! -d "/tmp/mender-artifact-storage" ]; then
    mkdir /tmp/mender-artifact-storage
fi

(cd /tmp/ && fakes3 -r . -p 4567 > /dev/null &)
./deployments -config tests/config.testing.yaml &
