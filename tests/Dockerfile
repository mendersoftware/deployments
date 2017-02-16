FROM ubuntu:16.04

RUN apt-get -y -qq update && apt-get -qq -y install \
    python3-pip \
    python3-pytest python3-crypto python3-twisted python3-requests python3-yaml \
    python3-tz && \
    rm -rf /var/lib/apt/lists/* && \
    pip3 install --quiet bravado

RUN mkdir -p /testing
ENTRYPOINT ["bash", "/testing/run.sh"]
