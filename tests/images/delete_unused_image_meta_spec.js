var frisby = require('frisby');
var images = require("./images_endpoint.js")

frisby.create('DELETE unused ID results in error')
    .delete(images.endpoint + '/851675c4-df63-42cf-a33c-cfa471e0524d')
    .expectStatus(404)
    .expectHeaderContains('content-type', 'application/json')
    .toss()
