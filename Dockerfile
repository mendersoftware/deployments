FROM alpine:3.4

RUN apk update && apk upgrade && \
     apk add ca-certificates && \
     rm -rf /var/cache/apk/*

RUN mkdir /etc/deployments

COPY ./config.yaml /etc/deployments/

ENTRYPOINT ["/entrypoint.sh"]

COPY ./entrypoint.sh /entrypoint.sh
COPY ./deployments /usr/bin/
