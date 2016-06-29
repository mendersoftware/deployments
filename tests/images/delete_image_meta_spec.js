var frisby = require('frisby');
var images = require("./images_endpoint.js")

frisby.create('POST valid image metadata, successfully DELETE it')
    .post(images.endpoint, {
        name: "Install image 2",
        description: "A test image 1",
        checksum: "1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2bff",
        device_type: "Beaglebone Black v3.1",
        yocto_id: "core-image-full-cmdline-20160330201409"
    }, {
        json: true
    })
    .expectStatus(201)
    .expectHeaderContains('content-type', 'application/json')
    .after(function(err, res, body) {
        var id = res.headers.location.split("/").pop(-1)
        frisby.create("DELETE valid ID")
            .delete(images.endpoint + '/' + id)
            .expectStatus(204)
            .after(function(err, res, body) {
                frisby.create("GET deleted ID")
                    .get(images.endpoint + '/' + id)
                    .expectStatus(404)
                    .toss()
            })
            .toss()
    })
    .toss()
