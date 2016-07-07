var frisby = require('frisby');
var images = require("./images_endpoint.js")

frisby.create('GET invalid ID results in error')
    .get(images.endpoint + '/7514-df22-42cf-a13c-dac331e0552a ')
    .expectStatus(400)
    .expectHeaderContains('content-type', 'application/json')
    .expectJSON({
        Error: "ID is not UUIDv4"
    })
    .toss()
