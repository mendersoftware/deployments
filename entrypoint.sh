#!/bin/sh -e
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

CERTS_DIR=/etc/ssl/certs
CERTS_BUNDLE=$CERTS_DIR/ca-certificates.crt

if [ -n "$STORAGE_BACKEND_CERT" -a -e "$STORAGE_BACKEND_CERT" ]; then
    cat "$STORAGE_BACKEND_CERT" >> $CERTS_BUNDLE
    wheredir=$(dirname "$STORAGE_BACKEND_CERT")
    if [ "$wheredir" != $CERTS_DIR ]; then
        cp "$STORAGE_BACKEND_CERT" $CERTS_DIR
    fi
    # storage certificate may or may not have been in CERTS_DIR already, just to
    # be safe, run c_rehash so that other tools work too
    c_rehash $CERTS_DIR
fi

exec deployments "$@"
