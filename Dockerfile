FROM alpine:3.4

RUN apk update && apk upgrade && \
     apk add ca-certificates && \
     rm -rf /var/cache/apk/*

RUN mkdir /etc/deployments

EXPOSE 8080

COPY ./config.yaml /etc/deployments/

ADD https://github.com/ufoscout/docker-compose-wait/releases/download/2.2.1/wait /wait-for-mongo
RUN chmod +x /wait-for-mongo

ENTRYPOINT ["/entrypoint.sh", "--config", "/etc/deployments/config.yaml"]

COPY ./entrypoint.sh /entrypoint.sh
COPY ./deployments /usr/bin/
