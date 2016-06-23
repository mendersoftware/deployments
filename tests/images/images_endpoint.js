var SERVER_PROTOCOL = process.env.SERVER_PROTOCOL || "http"
var SERVER_IP = process.env.SERVER_IP || "127.0.0.1"
var SERVER_PORT = process.env.SERVER_PORT || "8080"
var API_VERSION = process.env.API_VERSION || "0.0.1"

var host = SERVER_PROTOCOL + "://" + SERVER_IP + ":" + SERVER_PORT + "/api/" + API_VERSION + "/"
var imagesEndpoint = host + "images"

exports.endpoint = imagesEndpoint
