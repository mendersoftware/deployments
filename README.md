# Artifacts
[![Build Status](https://travis-ci.com/mendersoftware/artifacts.svg?token=rx8YqsZ2ZyaopcMPmDmo&branch=master)](https://travis-ci.com/mendersoftware/artifacts)
[![Coverage Status](https://coveralls.io/repos/mendersoftware/artifacts/badge.svg?branch=master&service=github&t=xZ0vYT)](https://coveralls.io/github/mendersoftware/artifacts?branch=master)

Service responsible for software deployment and image management.

    [Installation](#Installation)
    [Configuration](#Configuration)
    [API documentation](#API documentation)
    [Logging](#Logging)

## Installation
    Install instructions.

### Binaries (Linux x64)
    Latest build of binaries for Linux 64bit are [available](LINK).
    ```
        wget <link>
    ```

### Source
    Golang toolchain is required to build the application. [Installation instructions.](https://golang.org/doc/install)

    In addition for managing dependencies it depends on [godep](https://github.com/tools/godep) tool. Can be installed with following command:

    ```
        go get -u github.com/tools/godep
    ```

    Build instructions:

    ```
    $ go get -u github.com/mendersoftware/artifatcs
    $ cd $GOPATH/src/github.com/mendersoftware/artifatcs
    $ godep go build
    $ godep go test ./...
    ```

## Configuration
    Service is configured by providing configuration file. Supports JSON, TOML, YAML and HCL formatting.
    Default configuration file is provided to be downloaded from [config.yaml](https://github.com/mendersoftware/artifacts/blob/master/config.yaml).

    Application requirements:
    * Access to AWS S3 bucket, keys can be configured in several ways, documented in the configuration file.
    * Access to MongoDB instance and configured in config file. [Installation instructions](https://www.mongodb.org/downloads#)

## API documentation
    Application exposes REST API over HTTP protocol. Detailed documentation is specified in the following [document](https://github.com/mendersoftware/artifacts/blob/master/docs/api_spec.md).
    Format: [API Blueprint](https://apiblueprint.org)


## Logging
    Apache style error log is provided on stderr.
