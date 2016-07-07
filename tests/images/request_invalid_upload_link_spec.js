var frisby = require('frisby');
var images = require("./images_endpoint.js")

frisby.create('Request upload link for invalid image')
    .get(images.endpoint + "/2a5a4a61-344f-4b75-8355-20d294715490/upload?expire=3")
    .expectStatus(404)
    .toss();

frisby.create('Request upload link for invalid image - no expire')
    .get(images.endpoint + "/2a5a4a61-344f-4b75-8355-20d294715490/upload")
    .expectStatus(404)
    .toss();
