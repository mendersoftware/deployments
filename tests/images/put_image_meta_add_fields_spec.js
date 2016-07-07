var frisby = require('frisby');
var images = require("./images_endpoint.js")

var createImageAndEditJSON2 = {
    name: "Install image minimum - to edit",
    device_type: "Beaglebone",
    yocto_id: "core-image-full-cmdline-20160330201408"
}

frisby.create('Create image metadata and edit it (additive)')
    .post(images.endpoint, createImageAndEditJSON2, {
        json: true
    })
    .after(function(err, res, body) {
        var id = res.headers.location.split("/").pop(-1)
        var edit2 = createImageAndEditJSON2
        edit2.checksum = "f56f463dd9ab083695e8ab07bacf2efe2eb4a1ae58a3a914835a827e8e1c23dd12aaba526ebf192ebafdf418b85913c6d7804614e665099e686276ecad97ba96"
        edit2.description = "An edited description"

        frisby.create("edit image metadata - additive")
            .put(images.endpoint + "/" + id, edit2, {
                json: true
            })
            .expectStatus(204)
            .after(function(err, res, body) {
                frisby.create("check if edit successfull - additive")
                    .get(images.endpoint + "/" + id)
                    .expectStatus(200)
                    .expectJSON(edit2)
                    .toss();
            })
            .toss();
    })
    .toss();
