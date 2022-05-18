FROM golang:1.18.2-alpine3.15 as builder
WORKDIR /go/src/github.com/mendersoftware/deployments
RUN mkdir -p /etc_extra
RUN echo "nobody:x:65534:" > /etc_extra/group
RUN echo "nobody:!::0:::::" > /etc_extra/shadow
RUN echo "nobody:x:65534:65534:Nobody:/:" > /etc_extra/passwd
RUN chown -R nobody:nobody /etc_extra
RUN apk add --no-cache \
     xz-dev \
     musl-dev \
     gcc
COPY ./ .
RUN env CGO_ENABLED=1 go build

FROM alpine:3.15.4
RUN apk add --no-cache ca-certificates xz
EXPOSE 8080
COPY --from=builder /etc_extra/ /etc/
USER 65534
WORKDIR /etc/deployments
COPY --chown=nobody ./config.yaml .
COPY --chown=nobody ./entrypoint.sh /entrypoint.sh
COPY --from=builder --chown=nobody /go/src/github.com/mendersoftware/deployments/deployments /usr/bin/
CMD ["./entrypoint.sh"]  
ENTRYPOINT ["/entrypoint.sh", "--config", "/etc/deployments/config.yaml"]
