FROM iron/base

COPY ./deployments /usr/bin/

RUN mkdir /etc/deployments
COPY ./config.yaml /etc/deployments/

ENTRYPOINT ["/usr/bin/deployments", "-config", "/etc/deployments/config.yaml"]
