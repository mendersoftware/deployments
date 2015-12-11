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
