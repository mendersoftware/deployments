var frisby = require('frisby');
var images = require("./images_endpoint.js")

frisby.create('GET unused ID results in error')
    .get(images.endpoint + '/75167dc4-df22-42cf-a13c-dac331e0552a')
    .expectStatus(404)
    .expectHeaderContains('content-type', 'application/json')
    .toss()
