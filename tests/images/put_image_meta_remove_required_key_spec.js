var frisby = require('frisby');
var images = require("./images_endpoint.js")

frisby.create('Create image metadata and attempt to remove required field')
    .post(images.endpoint, {
        "name": "another test image",
        "yocto_id": "core-file-image2016010101010101",
        "device_type": "Orange Pi 2 w/ WiFi"
    }, {
        json: true
    })
    .expectStatus(201)
    .after(function(err, res, body) {
        var id = res.headers.location.split("/").pop(-1)
        frisby.create("edit image metadata to an existing metadata")
            .put(images.endpoint + "/" + id, {
                "name": "another test image",
                "yocto_id": "core-file-image2016010101010101"
            }, {
                json: true
            })
            .expectStatus(400)
            .expectJSON({
                Error: function(val) {
                    return Boolean(~val.search("DeviceType: non zero value required"))
                }
            })
            .toss();
    })
    .toss();
