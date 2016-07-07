var frisby = require('frisby');
var validator = require('validator');
var images = require("./images_endpoint.js")

frisby.create('POST valid image (with min. parameters) metadata and verify location header in response')
    .post(images.endpoint, {
        name: "Install image minimum",
        device_type: "Beaglebone",
        yocto_id: "core-image-full-cmdline-20160330201408"
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
