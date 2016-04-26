FROM iron/base

COPY ./artifacts /usr/bin/

RUN mkdir /etc/artifacts
COPY ./config.yaml /etc/artifacts/

ENTRYPOINT ["/usr/bin/artifacts", "-config", "/etc/artifacts/config.yaml"]
