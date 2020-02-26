FROM python:3

RUN pip3 install --quiet bravado==9.2.2 pymongo==3.6.1 pytest-ordering==0.5 minio==5.0.6 crypto pytest==3.10.1 twisted requests pyyaml tz

# Install musl runtime support to be able to execute the dynamically linked deployments binary
# See https://wiki.debian.org/musl
RUN apt-get -qq update && \
    apt-get -qy install musl && \
    echo "/lib/x86_64-linux-gnu" >> /etc/ld-musl-x86_64.d/x86_64-linux-musl.path && \
    ld-musl-config

RUN mkdir -p /testing

ENTRYPOINT ["bash", "/testing/run.sh"]
