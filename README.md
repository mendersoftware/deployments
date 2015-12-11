[![Build Status](https://travis-ci.com/mendersoftware/artifacts.svg?token=rx8YqsZ2ZyaopcMPmDmo&branch=master)](https://travis-ci.com/mendersoftware/artifacts)
[![Coverage Status](https://coveralls.io/repos/mendersoftware/artifacts/badge.svg?branch=master&service=github&t=xZ0vYT)](https://coveralls.io/github/mendersoftware/artifacts?branch=master)

# Artifacts

Service responsible for artifact management and distribution.

## Usage

Manual how to use and operate the service.

```
$ artifacts --help
```

## Version 0.0.1 Features:
* Create image metadata
* List image metadata
* Get image metadata
* Edit image metadata
* Delete image (from metadata and S3)
* Generate TTLd link for uploading image file to S3
* Generate TTLd link for downloading image file from S3 

## Authentication (DUMMY)

Supports base auth. (Required)
Authenticates user if username and pass are the same.
Uses username as customerID.

## Logging

Apache style error log is provided on strerr.

## Compression

When executing in production environment responses are compressed with gzip if the request Accept-Encoding specifies support for gzip.

## Response format

When production environment is specified JSON is formatted as compact and pretty print in development environment.

## Panic recovery

In development environment in case of panic, stack trace is provided included in error response.

## Requirements:

* **uuidgen** installed and available on the path.

## Development

Golang dev environment required.

```
$ go get github.com/mendersoftware/artifatcs
$ cd $GOPATH/src/github.com/mendersoftware/artifatcs
$ godep restore
$ go build
$ go test ./...
```

All dependencies are vendored using [godep](https://github.com/tools/godep) tool.

## Testing

Run tests on current package:

```
$ go test
```

Run tests on all subdirectories:

```
$ go test ./...
```

Run benchmarks on local package:

```
$ go test -bench=.
```

Code coverage for package:

```
$ go test -cover
```

Note this include test coverage only for code in current package.
