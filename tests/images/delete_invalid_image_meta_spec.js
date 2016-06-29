var frisby = require('frisby');
var images = require("./images_endpoint.js")


frisby.create('DELETE invalid ID results in error')
    .delete(images.endpoint + '/851675c4-df6X-42cf-a33c-cfa471e0524d')
    .expectStatus(400)
    .expectHeaderContains('content-type', 'application/json')
    .expectJSON({
        Error: "ID is not UUIDv4"
    })
