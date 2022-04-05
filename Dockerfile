FROM golang:1.17.8-alpine3.15 as builder
RUN apk add --no-cache \
     xz-dev \
     musl-dev \
     gcc \
     ca-certificates
WORKDIR /go/src/github.com/mendersoftware/deployments
COPY ./ .
RUN env CGO_ENABLED=0 go build

FROM scratch
WORKDIR /etc/deployments
EXPOSE 8080
COPY ./config.yaml .
COPY ./entrypoint.sh /entrypoint.sh
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/github.com/mendersoftware/deployments/deployments /usr/bin/
CMD ["./entrypoint.sh"]  
ENTRYPOINT ["/entrypoint.sh", "--config", "/etc/deployments/config.yaml"]
