# Deployments
[![Build Status](https://travis-ci.com/mendersoftware/deployments.svg?token=rx8YqsZ2ZyaopcMPmDmo&branch=master)](https://travis-ci.com/mendersoftware/deployments)
[![Coverage Status](https://coveralls.io/repos/github/mendersoftware/deployments/badge.svg?branch=master&t=n7mVVE)](https://coveralls.io/github/mendersoftware/deployments?branch=master)

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
    
### Docker Image

Prebuild docker images are available for each branch and tag. Available via [docker hub](https://hub.docker.com/r/mendersoftware/deployments/)

```
    docker pull mendersoftware/deployments:latest
    docke run -p 8080:8080 mendersoftware/deployments:latest 
```

### Source

Golang toolchain is required to build the application. [Installation instructions.](https://golang.org/doc/install)

Build instructions:

```
$ go get -u github.com/mendersoftware/deployments
$ cd $GOPATH/src/github.com/mendersoftware/deployments
$ go build
$ go test $(go list ./... | grep -v vendor)
```

Dependencies are managed using golang vendoring (GOVENDOREXPERIMENT)

## Configuration

Service is configured by providing configuration file. Supports JSON, TOML, YAML and HCL formatting.
Default configuration file is provided to be downloaded from [config.yaml](https://github.com/mendersoftware/deployments/blob/master/config.yaml).

Application requirements:
* Access to AWS S3 bucket, keys can be configured in several ways, documented in the configuration file.
* Access to MongoDB instance and configured in config file. [Installation instructions](https://www.mongodb.org/downloads#)

## API documentation

Application exposes REST API over HTTP protocol. Detailed documentation is specified in the following [document](https://github.com/mendersoftware/deployments/blob/master/docs/api_spec.md).
Format: [API Blueprint](https://apiblueprint.org)


## Logging

Apache style access log is provided on stderr.

