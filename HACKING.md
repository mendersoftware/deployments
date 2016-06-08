# Deplyments service for Mender.io server

## Usecases
### Uploading YOCTO image
### Deploying image to devices
### Device checking for deployment 


## Dependencies

Service depends on following external services to be able to work properly.

### Mongo DB
Mongo DB is used as primary persistent storage for the service. Requires access to decicated
db cluster (can be single server as also HA/sharded cluster).

Access and address configured in [config.yaml](https://github.com/mendersoftware/deployments/blob/master/config.yaml)

### Amazon Simple Storge Service (S3)
Service require read/write access to AWS S3 bucket. S3 is used for storing and distrubuting files
such as YOCTO images to devices during deployments. 

Configured in [config.yaml](https://github.com/mendersoftware/deployments/blob/master/config.yaml) 

### Inventory Service
Currently service assumes that `inventory service` will be introduced in the future
as a source of information about types of devices. `TestDevice` type is not assumed for all
devices while scheduling deployment.  

