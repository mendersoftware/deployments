#!/bin/sh -e

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

exec /usr/bin/deployments -config /etc/deployments/config.yaml "$@"
