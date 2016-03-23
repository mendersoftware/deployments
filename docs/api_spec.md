
# Common api information

Common information about all exposed API endpoints.

## Date format
Dates are formatted with [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt) standard.

## OPTIONS method support
Each endpoint supports OPTIONS HTTP method.

          Allow: GET,POST,OPTIONS

## Cross-origin resource sharing

CORS and none-CORS requests are supported. Details [wikipedia](https://en.wikipedia.org/wiki/Cross-origin_resource_sharing#Simple_example).

Allows `location` header exposure.

# Group Device

## List updates for device [GET /api/0.0.1/devices/{id}/update]

+ Parameters
    + id: `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` (string, required) -  Device ID

+ Response 200 (application/json)
    Next update for the device.

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "id": "/",
                "type": "object",
                "properties": {
                    "image": {
                        "id": "image",
                        "type": "object",
                        "properties": {
                            "uri": {
                                "id": "uri",
                                "type": "string"
                            },
                            "checksum": {
                                "id": "checksum",
                                "type": "string"
                            },
                            "id": {
                                "id": "id",
                                "type": "string"
                            },
                            "expire": {
                                "id": "expire",
                                "type": "string"
                            }
                        },
                        "required": [
                            "uri",
                            "id"
                        ]
                    },
                    "id": {
                        "id": "id",
                        "type": "string"
                    }
                },
                "required": [
                    "image",
                    "id"
                ]
            }

    + Body

            {
                "image": {
                    "uri": "https://aws.my_update_bucket.com/yocto_image123",
                    "checksum": "cc436f982bc60a8255fe1926a450db5f195a19ad",
                    "id": "f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
                    "expire": "2016-03-11T13:03:17.063493443Z"
                },
                "id": "w81s4fae-7dec-11d0-a765-00a0c91e6bf6"
            }

+ Response 204
    No updates for the device are available.

    + Body

+ Response 400 (application/json)
    Invalid request.

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

+ Response 404 (application/json)
    Resource not found.

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

+ Response 500 (application/json)
    Internal server error. Please retry in a while.

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

# Group Deployment

## Lookup deployments [GET /api/0.0.1/deployments{?status,name}]
Lookup deployments in the system, including active and history.

+ Parameters
    + status: `pending` (enum[string], optional) - Deployment status
        + Members
            + `pending` - Pending to start
            + `inprogress` - In progress
            + `success` - Finished with success
            + `failure` - Finished with failure
    + name: `Jonas fix` (string, optional) - Deployment name

+ Response 200 (application/json)
    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "created": {
                            "id": "created",
                            "type": "string"
                        },
                        "status": {
                            "id": "status",
                            "type": "string",
                            "enum": [
                                "inprogress",
                                "pending",
                                "success",
                                "failure"
                            ]
                        },
                        "name": {
                            "id": "name",
                            "type": "string"
                        },
                        "version": {
                            "id": "version",
                            "type": "string"
                        },
                        "id": {
                            "id": "id",
                            "type": "string"
                        },
                        "finished": {
                            "id": "finished",
                            "type": "string"
                        }
                    },
                    "required": [
                        "created",
                        "status",
                        "name",
                        "version",
                        "id"
                    ]
                }
            }

    + Body

            [
                {
                    "created": "2016-03-02 23:00:00 +0000 UTC",
                    "status": "inprogress",
                    "name": "production",
                    "version": "Application 0.0.1",
                    "id": "00a0c91e6-7dec-11d0-a765-f81d4faebf6",
                    "finished": "2016-03-02 24:00:00 +0000 UTC"
                }
            ]

+ Response 400 (application/json)
    Invalid request

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

+ Response 500 (application/json)
    Internal server error. Please retry in a while.

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

## Deploy software version [POST /api/0.0.1/deployments]
Deploy version of software to specified devices.

+ Request (application/json)
    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "name": {
                        "id": "name",
                        "type": "string"
                    },
                    "version": {
                        "id": "version",
                        "type": "string"
                    },
                    "devices": {
                        "id": "devices",
                        "type": "array",
                        "items": {
                            "type": "string"
                        }
                    }
                },
                "required": [
                    "name",
                    "version",
                    "devices"
                ]
            }

    + Body

            {
                "name": "Monthly update: January",
                "version": "MySecretApp v2",
                "devices": [
                    "00a0c91e6-7dec-11d0-a765-f81d4faebf6",
                    "50b0c91e6-1drc-51d0-a165-g81d4faebry"
                ]
            }

+ Response 201 (application/json)
    + Headers

            Location: /api/0.0.1/deployments/{id}

+ Response 400 (application/json)
    Bad request. The request could not be understood by the server due to malformed syntax.

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
                }

+ Response 404 (application/json)
    Resource not found

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

+ Response 500 (application/json)
    Internal server error. Please retry in a while.

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

## Manage deployment [/api/0.0.1/deployments/{id}]
Manage specific deployment.

### Status [GET]
Check status for specified deployment

+ Parameters
    + id (string,required) - Deployment identifier

+ Response 200 (application/json)
    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "created": {
                        "id": "created",
                        "type": "string"
                    },
                    "status": {
                        "id": "status",
                        "type": "string",
                        "enum": [
                        "inprogress",
                        "pending",
                        "success",
                        "failure"
                        ]
                    },
                    "name": {
                        "id": "name",
                        "type": "string",
                        "additionalItems": false
                    },
                    "version": {
                        "id": "version",
                        "type": "string"
                    },
                    "id": {
                        "id": "id",
                        "type": "string"
                    },
                    "finished": {
                        "id": "finished",
                        "type": "string"
                    }
                },
                "required": [
                    "created",
                    "status",
                    "name",
                    "version",
                    "id"
                ]
            }

    + Body

            {
                "created": "2016-03-02 23:00:00 +0000 UTC",
                "status": "inprogress",
                "name": "production",
                "version": "Application 0.0.1",
                "id": "00a0c91e6-7dec-11d0-a765-f81d4faebf6",
                "finished": "2016-03-02 24:00:00 +0000 UTC"
            }

+ Response 404 (application/json)
    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

+ Response 500 (application/json)
    Internal server error. Please retry in a while.

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

### Cancel [DELETE]
Cancel deployment.

+ Parameters
    + id: `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` (string,required) - Deployment identifier

+ Response 204

+ Response 404 (application/json)
    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

+ Response 500 (application/json)
    Internal server error. Please retry in a while.

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

### Statistics [GET /api/0.0.1/deployments/{deployment_id}/statistics]
Statistics for the deployment.

+ Parameters
    + deployment_id: `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` (string,required) - Deployment identifier

+ Response 200 (application/json)
    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "successful": {
                        "id": "successful",
                        "type": "integer",
                        "description": "Number of successful deployments"
                    },
                    "pending": {
                        "id": "pending",
                        "type": "integer",
                        "description": "Number of pending deployments"
                    },
                    "inprogress": {
                        "id": "inprogress",
                        "type": "integer",
                        "description": "Number of deployments in progress"
                    },
                    "failure": {
                        "id": "failure",
                        "type": "integer",
                        "description": "Number of failer deployments."
                    },
                    "noimage": {
                        "id": "noimage",
                        "type": "integer",
                        "description": "Do not have apropriate image for the device model."
                    }
                },
                "required": [
                    "successful",
                    "pending",
                    "inprogress",
                    "failure",
                    "noimage"
                ]
            }

    + Body

            {
                "successful": 3,
                "pending": 1,
                "inprogress": 23,
                "failure": 0,
                "noimage": 1
            }

+ Response 404 (application/json)
    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

+ Response 500 (application/json)
    Internal server error. Please retry in a while.

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

### List devices [GET /api/0.0.1/deployments/{deployment_id}/devices]
Device statuses for the deployment.

+ Parameters
    + deployment_id: `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` (string,required) - Deployment identifier

+ Response 200 (application/json)
    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "id": "/",
                "type": "array",
                "items": {
                    "id": "0",
                    "type": "object",
                    "properties": {
                        "id": {
                            "id": "id",
                            "type": "string"
                        },
                        "finished": {
                            "id": "finished",
                            "type": "string"
                        },
                        "status": {
                            "id": "status",
                            "type": "string",
                            "enum": [
                                "inprogress",
                                "pending",
                                "success",
                                "failure",
                                "noimage"
                            ]
                        },
                        "started": {
                            "id": "started",
                            "type": "string"
                        },
                        "model": {
                            "id": "model",
                            "type": "string"
                        },
                        "image_id": {
                            "id": "image_id",
                            "type": "string"
                        }
                    },
                    "required": [
                        "id",
                        "status",
                        "model"
                    ]
                }
            }

    + Body

            [
                {
                    "id": "00a0c91e6-7dec-11d0-a765-f81d4faebf6",
                    "finished": "2016-03-02 24:00:00 +0000 UTC",
                    "status": "pending",
                    "started": "2016-03-02 23:20:00 +0000 UTC",
                    "model": "Raspberry Pi 3",
                    "image_id": "60a0c91e6-7dec-11d0-a765-f81d4faebf6"
                }
            ]

+ Response 404 (application/json)
    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

+ Response 500 (application/json)
    Internal server error. Please retry in a while.

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

### Deployment log [GET /api/0.0.1/deployments/{deployment_id}/devices/{device_id}/log]
Device statuses for the deployment.

+ Parameters
    + deployment_id: `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` (string,required) - Deployment identifier
    + device_id: `00a0c91e6-7dec-11d0-a765-f81d4faebf6` (string, required) - Device identifier

+ Response 200 (application/text)

    ```
    Feb 23 15:43:38 mrowa BezelServices 255.10[92] <Error>: ASSERTION FAILED: dvcAddrRef != ((void *)0) -[DriverServicesgetDeviceAddress:] Feb 23 15:45:10 mrowa com.apple.WebKit.WebContent[606] <Error>: [15:45:10.933] FigAgglomeratorSetObjectForKey signalled err=-16020 (kFigStringConformerError_ParamErr) (NULL key) at /Library/Caches/com.apple.xbs/Sources/CoreMedia/CoreMedia-1731.15.33/Prototypes/LegibleOutput/FigAgglomerator.c line 92 Feb 23 15:45:18 mrowa com.apple.WebKit.WebContent[606] <Error>: [15:45:18.956] <<<< Boss >>>> figPlaybackBossPrerollCompleted: unexpected preroll-complete notification
    Feb 23 15:45:18 mrowa com.apple.WebKit.WebContent[606] <Error>: [15:45:18.957] <<<< Boss >>>> figPlaybackBossPrerollCompleted: unexpected preroll-complete notification
    Feb 23 15:45:40 mrowa syslogd[44] <Notice>: ASL Sender Statistics

    ```

+ Response 404 (application/json)
    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

+ Response 500 (application/json)
    Internal server error. Please retry in a while.

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

# Group YOCTO images
Manage YOCTO images.

## Lookup images [GET /api/0.0.1/images]
List all YOCTO images.

+ Response 200 (application/json)
    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "name": {
                            "id": "name",
                            "type": "string"
                        },
                        "description": {
                            "id": "description",
                            "type": "string"
                        },
                        "checksum": {
                            "id": "checksum",
                            "type": "string"
                        },
                        "model": {
                            "id": "model",
                            "type": "string"
                        },
                        "id": {
                            "id": "id",
                            "type": "string"
                        },
                        "verified": {
                            "id": "verified",
                            "type": "boolean"
                        },
                        "modified": {
                            "id": "modified",
                            "type": "string",
                            "description": "represent creation / last edition of any of the image properties, including image file upload or rewrite "
                        }
                    },
                    "required": [
                        "name",
                        "description",
                        "checksum",
                        "model",
                        "id",
                        "verified",
                        "modified"
                    ]
                }
            }

    + Body

            [
                {
                    "name": "MySecretApp v2",
                    "description": "Johns Monday test build",
                    "checksum": "cc436f982bc60a8255fe1926a450db5f195a19ad",
                    "model": "Beagle Bone",
                    "id": "0C13A0E6-6B63-475D-8260-EE42A590E8FF",
                    "verified": false,
                    "modified": "2016-03-02 23:20:00 +0000 UTC"
                }
            ]

+ Response 500 (application/json)
    Internal server error. Please retry in a while.

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

## Create image [POST /api/0.0.1/images]
Create YOCTO image. Afterwards upload link can be generated to upload image file.

+ Request (application/json)
    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "name": {
                        "id": "name",
                        "type": "string",
                        "description": "reqired to be uniqe across all images"
                    },
                    "description": {
                        "id": "description",
                        "type": "string"
                    },
                    "checksum": {
                        "id": "checksum",
                        "type": "string"
                    },
                    "model": {
                        "id": "model",
                        "type": "string"
                    }
                },
                "required": [
                    "name",
                    "model"
                ]
            }

    + Body

            {
                "name": "Application 1.1",
                "description": "Monday build for production",
                "checksum": "cc436f982bc60a8255fe1926a450db5f195a19ad",
                "model": "Beagle Bone"
            }

+ Response 201
    + Headers

            Location: /api/0.0.1/images/{id}

+ Response 404 (application/json)
    Resource not found

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

+ Response 500 (application/json)
    Internal server error. Please retry in a while.

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

## Manage images [/api/0.0.1/images/{id}]
Manage selected image

### Image details [GET]
Image datails.

+ Parameters
   + id: `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` (string,required) - Image ID

+ Response 200 (application/json)
    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "name": {
                        "id": "name",
                        "type": "string"
                    },
                    "description": {
                        "id": "description",
                        "type": "string"
                    },
                    "checksum": {
                        "id": "checksum",
                        "type": "string"
                    },
                    "model": {
                        "id": "model",
                        "type": "string"
                    },
                    "id": {
                        "id": "id",
                        "type": "string"
                    },
                    "verified": {
                        "id": "verified",
                        "type": "boolean"
                    },
                    "modified": {
                        "id": "modified",
                        "type": "string",
                        "description": "represent creation / last edition of any of the image properties, including image file upload or rewrite "
                    }
                },
                "required": [
                    "name",
                    "description",
                    "checksum",
                    "model",
                    "id",
                    "verified",
                    "modified"
                ]
            }

    + Body

            {
                "name": "MySecretApp v2",
                "description": "Johns Monday test build",
                "checksum": "cc436f982bc60a8255fe1926a450db5f195a19ad",
                "model": "Beagle Bone",
                "id": "0C13A0E6-6B63-475D-8260-EE42A590E8FF",
                "verified": false,
                "modified": "2016-03-02 23:20:00 +0000 UTC"
            }

+ Response 404 (application/json)
    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

+ Response 500 (application/json)
   Internal server error. Please retry in a while.

   + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

   + Body

            {
                "error": "Detailed error message"
            }

### Edit image [PUT]
Edit image information.

+ Parameters
   + id: `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` (string,required) - Image ID

+ Request (application/json)
    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "name": {
                        "id": "name",
                        "type": "string",
                        "description": "reqired to be uniqe across all images"
                    },
                    "description": {
                        "id": "description",
                        "type": "string"
                    },
                    "checksum": {
                        "id": "checksum",
                        "type": "string"
                    },
                    "model": {
                        "id": "model",
                        "type": "string"
                    }
                },
                "required": [
                    "name",
                ]
            }

    + Body

            {
                "name": "Application 1.1",
                "description": "Monday build for production",
                "checksum": "cc436f982bc60a8255fe1926a450db5f195a19ad",
                "model": "Beagle Bone"
            }

+ Response 204

+ Response 400 (application/json)
    Invalid request

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

+ Response 404 (application/json)
    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

+ Response 500 (application/json)
   Internal server error. Please retry in a while.

   + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

   + Body

            {
                "error": "Detailed error message"
            }

### Remove image [DELETE]
Remove YOCTO image.

+ Parameters
   + id: `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` (string,required) - Image ID

+ Response 204

+ Response 404 (application/json)
    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

+ Response 500 (application/json)
   Internal server error. Please retry in a while.

   + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

   + Body

            {
                "error": "Detailed error message"
            }

### Generate upload link [GET /api/0.0.1/images/{id}/upload{?expire}]
Generate signed URL for uploading image file.

URI can be used only with PUT HTTP method.
It is valid for specified period if time.

In case link is used multiple times to upload file, file will be overwritten.

+ Parameters
    + id: `0C13A0E6-6B63-475D-8260-EE42A590E8FF` (string, required) - Image ID
    + expire: 60 (number, required) - Link validity length in minutes. Min 1 minute, max 10080 (1 week)

+ Response 200 (application/json)
    + Headers

            Expires: Tue, 10 Nov 2009 23:00:00 GMT

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "uri": {
                        "id": "uri",
                        "type": "string"
                    },
                    "expire": {
                        "id": "expire",
                        "type": "string"
                    }
                },
                "required": [
                    "uri",
                    "expire"
                ]
            }

    + Body

            {
                "uri": "https://exmple.com/file123",
                "expire": "2016-03-11T13:03:17.063493443Z"
            }

+ Response 400 (application/json)
    Invalid request

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

+ Response 404 (application/json)
    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

+ Response 500 (application/json)
   Internal server error. Please retry in a while.

   + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

   + Body

            {
                "error": "Detailed error message"
            }

### Generate download link [GET /api/0.0.1/images/{id}/download{?expire}]
Generate signed URL for downloading image file.

URI can be used only with GET HTTP method.
Link supports such HTTP headers: `Range`, `If-Modified-Since`, `If-Unmodified-Since`
It is valid for specified period if time.

To be able to recieve download link, image file have to be uploaded first.

+ Parameters
    + id: `0C13A0E6-6B63-475D-8260-EE42A590E8FF` (string, required) - Image ID
    + expire: 60 (number, required) - Link validity length in minutes. Min 1 minute, max 10080 (1 week)

+ Response 200 (application/json)
    + Headers

            Expires: Tue, 10 Nov 2009 23:00:00 GMT

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "uri": {
                        "id": "uri",
                        "type": "string"
                    },
                    "expire": {
                        "id": "expire",
                        "type": "string"
                    }
                },
                "required": [
                    "uri",
                    "expire"
                ]
            }

    + Body

            {
                "uri": "https://exmple.com/file123",
                "expire": "2016-03-11T13:03:17.063493443Z"
            }

+ Response 400 (application/json)
    Invalid request

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

+ Response 404 (application/json)
    Resource not found. Could mean for not having access, image does not exist or file have not been uploaded.

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }

+ Response 500 (application/json)
    Internal server error. Please retry in a while.

    + Schema

            {
                "$schema": "http://json-schema.org/draft-04/schema#",
                "type": "object",
                "properties": {
                    "error": {
                        "id": "error",
                        "type": "string"
                    }
                }
            }

    + Body

            {
                "error": "Detailed error message"
            }
