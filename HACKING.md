# Deplyments service for Mender.io server

## Usecases

Flow and descriptions of majour usecases.

### Uploading YOCTO image
### Deploying image to devices

User deploy image to specified group of devices. Deployment for each device is precomputed. 
Image is auto-assigned to device deploment based on artifact name and device type (fetched from inventory)

```
User             Deployment ser^ice         Collection: deployments    Collection: de^ice_deployments    Collection: images    In^entory Ser^ice

 +                        +                            +                               +                          +                    +
 |                        |                            |                               |                          |                    |
 | Deploy ^ersion 'X'     |                            |                               |                          |                    |
 | To de^ices 'Y','Z'     |                            |                               |                          |                    |
 +------------------------>                            |                               |                          |                    |
 |                        |                            |                               |                          |                    |
 |                   +----------------------------------------------------------------------------------------------------------------------+
 |                   |    | Check model of de^ice: Y   |                               |                          |                    |    |
 |                F  |    +------------------------------------------------------------------------------------------------------------>    |
 |                o  |    |                            |                               |                          |                    |    |
 |                R  |    | Model: BBB (mocked)        |                               |                          |                    |    |
 |                :  |    <------------------------------------------------------------------------------------------------------------+    |
 |                Y  |    |                            |                               |                          |                    |    |
 |                &  |    | Find image:                |                               |                          |                    |    |
 |                Z  |    | Version: 'X' Model: 'BBB'  |                               |                          |                    |    |
 |                   |    +--------------------------------------------------------------------------------------->                    |    |
 |                   |    |                            |                               |                          |                    |    |
 |                   |    | Image Metadata             |                               |                          |                    |    |
 |                   |    <---------------------------------------------------------------------------------------+                    |    |
 |                   |    |                            |                               |                          |                    |    |
 |                   +----------------------------------------------------------------------------------------------------------------------+
 |                        |                            |                               |                          |                    |
 |                        | Insert deployments         |                               |                          |                    |
 |                        | Id, high le^el info        |                               |                          |                    |
 |                        +----------------------------+                               |                          |                    |
 |                        |                            |                               |                          |                    |
 |                   +-----------------------------------------------------------------------+                    |                    |
 |                 F |    | Insert device_deployment   |                               |     |                    |                    |
 |                 O |    | with:                      |                               |     |                    |                    |
 |                 R |    | * deployment id            |                               |     |                    |                    |
 |                 : |    | * target de^ice id         |                               |     |                    |                    |
 |                 Y |    | * image  metadata          |                               |     |                    |                    |
 |                 & |    | * status: pending          |                               |     |                    |                    |
 |                 Z |    +------------------------------------------------------------>     |                    |                    |
 |                   |    |                            |                               |     |                    |                    |
 |                   +-----------------------------------------------------------------------+                    |                    |
 |                        |                            |                               |                          |                    |
 | Success: (HTTP Created)|                            |                               |                          |                    |
 | LINK:/deployments/{id} |                            |                               |                          |                    |
 <------------------------+                            |                               |                          |                    |
```

### Device checking for deployment

Device is peropdically sending GET request to check if there are any deplyoments.

```
De^ice                User             Deployment ser^ice         Collection: de^ice_deployments

  +                    +                        +                            +
  |                    |                        |                            |
  | I'm de^ice Y       |                        |                            |
  | Updates for me?    |                        |                            |
  +--------------------------------------------->                            |
  |                    |                        |                            |
  |                    |                        | Get me oldest not finished |
  |                    |                        | deployment for de^ice Y    |
  |                    |                        +---------------------------->
  |                    |                        |                            |
  |                    |                        | De^ice_update:             |
  |                    |                        | * deployment id            |
  |                    |                        | * target de^ice            |
  |                    |                        | * assigned image metadata  |
  |                    |                        | * status                   |
  |                    |                        <----------------------------+
  |                    |                        |                            |
  |                    |                        | Presign download link      |
  |                    |                        | for image.                 |
  |                    |                        |                            |
  |                    |                        | Package results            |
  |                    |                        |                            |
  | Install image G    |                        |                            |
  <---------------------------------------------+                            |
  |                    |                        |                            |
  +                    +                        +                            +
```

### Updating status of deplyoment

Notice: to be implemented

```
       De^ice                     Deployment Ser^ice            File ser^er

         +                                 +                         +
         | I'm de^ice X                    |                         |
         | Do I ha^e any update?           |                         |
         +--------------------------------->                         |
         |                                 |                         |
         | Update to image Y               |                         |
         | Your deployment ID: Z           |                         |
         <---------------------------------+                         |
         |                                 |                         |
         | I'm de^ice X                    |                         |
         | Set status of Z deployment      |                         |
         | to 'downloading'                |                         |
         +--------------------------------->                         |
         |                                 |                         |
         | Download image Y                |                         |
         +----------------------------------------------------------->
         |                                 |                         |
         | I'm de^ice X                    |                         |
         | Set status of Z deployment      |                         |
         | to 'ready to install'           |                         |
         |                                 |                         |
         +--------------------------------->                         |
         |                                 |                         |
         | I'm de^ice X                    |                         |
         | Set status of Z deployment      |                         |
         | to 'installing'                 |                         |
         +--------------------------------->                         |
         |                                 |                         |
+--------+--------+                        |                         |
|  Install image  |                        |                         |
+--------+--------+                        |                         |
         |                                 |                         |
         | I'm de^ice X                    |                         |
         | Set status of Z deployment      |                         |
         | to 'success'                    |                         |
         |                                 |                         |
         +--------------------------------->                         |
         |                                 |                         |
         |                                 |                         |
         |                                 |                         |
         +                                 +                         +
```

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

