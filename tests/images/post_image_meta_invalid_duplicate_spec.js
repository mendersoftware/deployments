var frisby = require('frisby');
var validator = require('validator');
var images = require("./images_endpoint.js")

frisby.create('Create invalid image metadata: name + device_type is not unique')
    .post(images.endpoint, {
        name: "foo",
        device_type: "BBB",
        yocto_id: "foo1"
    }, {
        json: true
    })
    .expectStatus(201)
    .after(function(err, res, body) {
        frisby.create()
            .post(images.endpoint, {
                name: "foo",
                device_type: "BBB",
                yocto_id: "foo2"
            }, {
                json: true
            })
            .expectStatus(500)
            .expectJSON({
                Error: function(val) {
                    return Boolean(~val.search("dup key"))
                }
            })
            .toss()
    })
    .toss();
