FROM python:3

RUN pip3 install --quiet bravado==9.2.2 pymongo==3.6.1 pytest-ordering==0.5 minio crypto pytest==3.10.1 twisted requests pyyaml tz

RUN ln -s /lib64/ld-linux-x86-64.so.2 /lib/libc.musl-x86_64.so.1

RUN mkdir -p /testing

ENTRYPOINT ["bash", "/testing/run.sh"]
