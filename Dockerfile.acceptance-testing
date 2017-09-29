FROM alpine:3.4

RUN apk update && apk upgrade && \
     apk add ca-certificates && \
     rm -rf /var/cache/apk/*

RUN mkdir /etc/deployments

EXPOSE 8080

COPY ./entrypoint.sh /usr/bin/
COPY ./deployments-test /usr/bin/deployments
COPY ./config.yaml /usr/bin/

STOPSIGNAL SIGINT
ENV DEPLOYMENTS_MENDER_GATEWAY http://mender-inventory:8080
ENTRYPOINT ["/usr/bin/entrypoint.sh", "-test.coverprofile=/testing/coverage-acceptance.txt", "-acceptance-tests", "-test.run=TestRunMain", "-cli-args=server --automigrate"]
