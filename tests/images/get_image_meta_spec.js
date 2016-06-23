var frisby = require('frisby');
var validator = require('validator');
var images = require("./images_endpoint.js")

frisby.create("Create image metadata and GET selected image")
    .post(images.endpoint, {
        name: "A test image 3",
        device_type: "BBB 1.3",
        yocto_id: "Foo123"
    }, {
        json: true
    })
    .expectStatus(201)
    .after(function(err, res, body) {
        var id = res.headers.location.split("/").pop(-1)
        frisby.create("GET selected image")
            .get(images.endpoint + '/' + id)
            .expectStatus(200)
            .expectJSON({
                name: "A test image 3",
                device_type: "BBB 1.3",
                yocto_id: "Foo123"
            })
            .toss()
    })
    .toss()
