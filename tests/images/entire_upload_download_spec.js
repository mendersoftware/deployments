var frisby = require('frisby');
var images = require("./images_endpoint.js")

var uploadPayload = {data: Array(1024).join("foobar")}

frisby.create('Perform upload and download')
    .post(images.endpoint, {
        name: "Install image abcd 2",
        description: "This is a very nice description again",
        checksum: "1542850d66d8007d620e4050b57a5dc83f4a921d36ce9ce47d0d13c5d85f2bff",
        device_type: "Beaglebone Black 9000",
        yocto_id: "core-image-full-cmdline-20160331201499"
    }, {
        json: true
    })
    .expectStatus(201)
    .after(function(err, res, body) {
        var id = res.headers.location.split("/").pop(-1)
        frisby.create("Request upload link - 3")
            .get(images.endpoint + "/" + id + "/upload")
            .expectStatus(200)
            .after(function(err, res, body) {
                var uploadURI = JSON.parse(body).uri
                frisby.create("Upload some data to upload link")
                    .put(uploadURI, uploadPayload, {json: true})
                    .expectStatus(200)
                    .after(function(err, res, body) {
                        frisby.create("Get download link")
                            .get(images.endpoint + "/" + id + "/download")
                            .expectStatus(200)
                            .after(function(err, res, body) {
                                var downloadLink = JSON.parse(body).uri
                                frisby.create("perform download and verify content")
                                    .get(downloadLink)
                                    .expectStatus(200)
                                    .expectBodyContains(JSON.stringify(uploadPayload))
                                    .toss();
                            })
                            .toss();
                    })
                    .toss();
            })
            .toss()
    })
    .toss()
