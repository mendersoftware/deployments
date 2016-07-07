var frisby = require('frisby');
var images = require("./images_endpoint.js")

frisby.create('Create image metadata and editting it to a duplicate fails')
    .post(images.endpoint, {
        name: "Install image - to edit",
        description: "A test image 1",
        checksum: "",
        device_type: "Beaglebone Black v3.1",
        yocto_id: "core-image-full-cmdline-20160330201409"
    }, {
        json: true
    })
    .after(function(err, res, body) {
        var id = res.headers.location.split("/").pop(-1)
        frisby.create("edit image metadata to an existing metadata")
            .put(images.endpoint + "/" + id,  {
                name: "Install image 1",
                description: "A Ţest image",
                checksum: "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e",
                device_type: "Beaglebone",
                yocto_id: "core-image-full-cmdline-20160330201408"
            }, {
                json: true
            })
            .expectStatus(500)
            .expectJSON({
                Error: function(val) {
                    return Boolean(~val.search("dup key"))
                }
            })
            .toss();
    })
    .toss();
