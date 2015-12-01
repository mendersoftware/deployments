[![Build Status](https://travis-ci.com/mendersoftware/artifacts.svg?token=rx8YqsZ2ZyaopcMPmDmo&branch=master)](https://travis-ci.com/mendersoftware/artifacts)

# Artifacts

Service responsible for artifact management and distribution.

## Version 0.0.1 Features:
* Create image metadata
* List image metadata
* Get image metadata
* Edit image metadata
* Delete image (from metadata and S3)
* Generate TTLd link for uploading image file to S3
* Generate TTLd link for downloading image file from S3

# Usage

Manual how to use and operate the service.

## CLI

```
NAME:
   artifacts - Archifact management service.

USAGE:
   artifacts [global options] command [command options] [arguments...]

VERSION:
   0.0.1

AUTHOR(S):
   Maciej Mrowiec <maciej.mrowiec@mender.io>  <contact@mender.io>

COMMANDS:
   help, h	Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --certificate                    HTTPS certificate filename. [$MENDER_ARTIFACT_CERT]
   --key                            HTTPS private key filename. [$MENDER_ARTIFACT_CERT_KEY]
   --https                          Serve under HTTPS. Requires key and cerfiticate. [$MENDER_ARTIFACT_HTTPS]
   --listen "localhost:8080"        TCP network address. [$MENDER_ARTIFACT_LISTEN]
   --env "dev"                      Environment prod|dev [$MENDER_ARTIFACT_ENV]
   --aws-id 				        AWS access id key with S3 read/write permissions for specified bucket (required). [$AWS_ACCESS_KEY_ID]
   --aws_secret 			        AWS secret key with S3 read/write permissions for specified bucket (required). [$AWS_SECRET_ACCESS_KEY]
   --bucket "mender-file-storage"   S3 bucket name for image storage. [$MENDER_S3_BUCKET]
   --aws-region "eu-west-1"         AWS region. [$AWS_REGION]
   --help, -h                       show help
   --version, -v                    print the version

```
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

* **uuidgen** tool on the path

# API

REST API documentation for **artifacts** service.

## Errors

Common error response:

```
{
    "error": "Specific error message"
}
```

## Dates

Dates are formatted with [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt) standard.

## Content type

If Content-Type is provided be the client request it is expected to be set to 'application/json'.

## Resources

Available resources, operations and their responses.

### Lookup available images.

**URI:** Get /api/0.0.1/images/

### Insert image meta data.

**URI:** Post /api/0.0.1/images/

Content-Type: application/json

Body Payload (example):

```
{
  "name":"MyName",
  "md5": "ui2ehu2h3823",
  "model": "model1"
}
```

Note:
* Name is required field and has to be unique.
* Response new object as payload + Location header. HTTP CREATED
* Unknown fields are ignored.

Output payload contains created object.
```
{
    "name": "MyName1",
    "description": "",
    "md5": "ui2ehu2h3823",
    "model": "model1",
    "id": "0C13A0E6-6B63-475D-8260-EE42A590E8FF",
    "verified": false,
    "modified": "2015-10-06T15:43:42.663180816+02:00"
}
```

201 Created on Success


Location header set with path to new resource.
Example:

```
/api/0.0.1/images/0C13A0E6-6B63-475D-8260-EE42A590E8FF
```

500 Internal Server Error on error

### Get image meta data.

**URI:** Get /api/0.0.1/images/:id

Json object or 404

### Edit image meta data.

**URI:** Put /api/0.0.1/images/:id

Body Payload (example):

```
{
  "name":"MyName",
  "md5": "ui2ehu2h3823",
  "model": "model1"
}
```

Note:
* Name is required field and has to be unique.
* Unknown fields are ignored.

### Delete image meta data.

**URI:** Delete /api/0.0.1/images/:id

Removes also file (download links will throw not found resource)

### Get image dowload URL

Request presigned HTTP GET request for image file. Request is valid for **expire** time.

**URI:** GET /api/0.0.1/images/:id/download

Supported query parameters:

* expire - Request expire time in [minutes]. Min=1, Max=10080 (1 week). (REQUIRED)

**Response Details:**

URI can be userd as HTTP GET request.

Supported HTTP headers:

* Range - Downloads the specified range bytes of an object. [RFC2616](http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.35)
* If-Modified-Since - Return the object only if it has been modified since the specified time, otherwise return a 304 (not modified).
* If-Unmodified-Since - Return the object only if it has not been modified since the specified time, otherwise return a 412 (precondition failed).

For more information about using presigned requests [visit](http://docs.aws.amazon.com/AmazonS3/latest/API/RESTObjectGET.html)

### Get image upload URL

Request presigned HTTP PUT to upload file. Request is valid for **expire** time.

URI: GET /api/0.0.1/images/:id/download

Supported query parameters:

* expire - Request expire time in [minutes]. Min=1, Max=10080 (1 week). (REQUIRED)

Response:

**Response Details:**

URI can be used as HTTP PUT request.

Supported HTTP headers:


* Content-MD5 - The base64-encoded 128-bit MD5 digest of the message (without the headers) according to RFC 1864. This header can be used as a message integrity check to verify that the data is the same data that was originally sent. Although it is optional, we recommend using the Content-MD5 mechanism as an end-to-end integrity check.

For more information about using presigned requests [visit](http://docs.aws.amazon.com/AmazonS3/latest/API/RESTObjectPUT.html)



## Notes for later

Partial download (HTTP Range):

```
curl -i --header "Range: bytes=10-20"  'http://akjdhaksjdh'

HTTP/1.1 206 Partial Content
Accept-Ranges: bytes
Content-Length: 11
Content-Range: bytes 10-20/22
Content-Type: text/plain; charset=utf-8
Last-Modified: Mon, 21 Sep 2015 13:49:50 GMT
Date: Mon, 21 Sep 2015 15:25:32 GMT

binary-payload
```

Don't send payload if not modified (HTTP If-Modified-Since):

```
curl -i -H "If-Modified-Since: Mon, 21 Sep 2015 14:49:49 GMT" http://localhost:8080/api/0.0.1/artifacts/images/test

HTTP/1.1 304 Not Modified
Date: Mon, 21 Sep 2015 15:23:15 GMT
```

Context-type auto detected.

Serves files directly from ./storage directory (for now).

# Development

Make sure you have golang/git enviroment installed.
GOPATH environment variable set corectly.

```
go get github.com/mendersoftware/artifatcs
go build
go test ./...
```

All dependencies are vendored using [godep](https://github.com/tools/godep) tool.

## Additional information

Golang has a strict directory tree for package import purposes.
If you want to work with fork of this project. Your fork has to be cloned to following directory (origin project directory tree):

```
$GOPATH/src/github.com/mendersoftware/artifacts/
```

**go get** tool usually does this for you, however if you use your private fork,
this tool would git clone to `$GOPATH/src/github.com/user123/mender-services/` and all imports of local packages inside if this project would still point to `$GOPATH/src/github.com/mendersoftware/mender-services/`. In case of having project in both directories, it may create confusing situation when modifications in fork sub package will not show up in local build of the forked version because it uses package from origin directory.
