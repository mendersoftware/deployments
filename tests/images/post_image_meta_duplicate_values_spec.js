var frisby = require('frisby');
var validator = require('validator');
var images = require("./images_endpoint.js")

frisby.create('Create image metadata - w/ common name + yocto_id')
    .post(images.endpoint, {
        name: "foo",
        device_type: "BB",
        yocto_id: "foo"
    }, {
        json: true
    })
    .expectStatus(201)
    .after(function(err, res, body) {
        frisby.create("create second identical name + yocto_id pair")
            .post(images.endpoint, {
                name: "foo",
                device_type: "RPi3",
                yocto_id: "foo"
            }, {
                json: true
            })
            .expectStatus(201)
            .toss()
    })
    .toss();
