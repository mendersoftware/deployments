var frisby = require('frisby');
var images = require("./images_endpoint.js")

var createImageAndEditJSON3 = {
    name: "Install image",
    description: "This will be editted out!",
    checksum: "1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2bff",
    device_type: "Beaglebone Black 2",
    yocto_id: "core-image-full-cmdline-20160330201408"
}

frisby.create('Create image metadata and edit it (subtractive)')
    .post(images.endpoint, createImageAndEditJSON3, {
        json: true
    })
    .after(function(err, res, body) {
        var id = res.headers.location.split("/").pop(-1)
        var edit3 = createImageAndEditJSON3
        delete edit3.checksum
        delete edit3.description

        frisby.create("edit image metadata - subtractive")
            .put(images.endpoint + "/" + id, edit3, {
                json: true
            })
            .expectStatus(204)
            .after(function(err, res, body) {
                frisby.create("check if edit successfull - subtractive")
                    .get(images.endpoint + "/" + id)
                    .expectStatus(200)
                    .expectJSON(edit3)
                    .toss();
            })
            .toss();
    })
    .toss();
