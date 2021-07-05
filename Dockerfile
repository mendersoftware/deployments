FROM golang:1.14-alpine3.12 as builder
RUN apk add --no-cache \
     xz-dev \
     musl-dev \
     gcc
WORKDIR /go/src/github.com/mendersoftware/deployments
COPY ./ .
RUN env CGO_ENABLED=1 go build

FROM alpine:3.14.0
RUN apk add --no-cache ca-certificates xz
RUN mkdir -p /etc/deployments
EXPOSE 8080
COPY ./config.yaml /etc/deployments
COPY ./entrypoint.sh /entrypoint.sh
COPY --from=builder /go/src/github.com/mendersoftware/deployments/deployments /usr/bin
CMD ["./entrypoint.sh"]  
ENTRYPOINT ["/entrypoint.sh", "--config", "/etc/deployments/config.yaml"]
