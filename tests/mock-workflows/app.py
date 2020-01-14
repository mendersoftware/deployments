import binascii
import os
import time

from flask import Flask, jsonify

app = Flask(__name__)


def gen_random_object_id():
    timestamp = "{0:x}".format(int(time.time()))
    rest = binascii.b2a_hex(os.urandom(8)).decode("ascii")
    return timestamp + rest


@app.route("/api/v1/workflow/<name>", methods=["POST"])
def generate_artifact(name: str):
    response = {"id": gen_random_object_id(), "name": name}
    return jsonify(response), 201
