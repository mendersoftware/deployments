var frisby = require('frisby');
var validator = require('validator');
var images = require("./images_endpoint.js")

frisby.create('POST valid image (with all parameters) metadata and verify location header in response')
    .post(images.endpoint, {
        name: "Install image 1",
        description: "A Ţest image",
        checksum: "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e",
        device_type: "Beaglebone",
        yocto_id: "core-image-full-cmdline-20160330201408",
    }, {
        json: true
    })
    .expectStatus(201)
    .expectHeaderContains('content-type', 'application/json')
    .expectHeaderToMatch("location", /\/api\/\d.\d.\d\/images\/[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$/g)
    .after(function(err, res, body) {
        var id = res.headers.location.split("/").pop(-1)
        expect(validator.isUUID(id, 4)).toBe(true)
    })
    .toss()
