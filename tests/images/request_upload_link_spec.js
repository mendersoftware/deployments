var frisby = require('frisby');
var validator = require('validator')
var images = require("./images_endpoint.js")

frisby.create('Request upload link for image')
    .post(images.endpoint, {
        name: "Install image abc",
        description: "This is a very nice description",
        checksum: "1542850d66d8007d620e4050b57a5dc83f4a921d36ce9ce47d0d13c5d85f2bff",
        device_type: "Beaglebone Black 5000",
        yocto_id: "core-image-full-cmdline-20160330201499"
    }, {
        json: true
    })
    .expectStatus(201)
    .after(function(err, res, body) {
        var id = res.headers.location.split("/").pop(-1)
        frisby.create("Request upload link - 1")
            .get(images.endpoint + "/" + id + "/upload?expire=30")
            .expectStatus(200)
            .after(function(err, res, body) {
                jsonBody = JSON.parse(body)
                expect(validator.isURL(jsonBody.uri)).toBe(true)
                expect(validator.isDate(jsonBody.expire)).toBe(true)
            })
            .toss();
    })
    .toss();


frisby.create('Request upload link for image without expire parameter')
    .post(images.endpoint, {
        name: "Install image abcdefg",
        description: "This is a very nice description",
        checksum: "1542850d66d8007d620e4050b57a5dc83f4a921d36ce9ce47d0d13c5d85f2bff",
        device_type: "BBB 5000",
        yocto_id: "core-image-full-cmdline-20160330201499"
    }, {
        json: true
    })
    .expectStatus(201)
    .after(function(err, res, body) {
        var id = res.headers.location.split("/").pop(-1)
        frisby.create("Request upload link - 2")
            .get(images.endpoint + "/" + id + "/upload")
            .expectStatus(200)
            .after(function(err, res, body) {
                jsonBody = JSON.parse(body)
                expect(validator.isURL(jsonBody.uri)).toBe(true)
                expect(validator.isDate(jsonBody.expire)).toBe(true)
            })
            .toss();
    })
    .toss();
