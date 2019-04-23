FROM golang:1.11.4-alpine3.8 as builder
RUN apk update && apk upgrade && \
     apk add \
     xz-dev \
     musl-dev \
     gcc
RUN mkdir -p /go/src/github.com/mendersoftware/deployments
COPY . /go/src/github.com/mendersoftware/deployments
RUN cd /go/src/github.com/mendersoftware/deployments && env CGO_ENABLED=1 go build

FROM alpine:3.6
RUN apk update && apk upgrade && \
     apk add --no-cache ca-certificates xz
RUN mkdir -p /etc/deployments
EXPOSE 8080
COPY ./config.yaml /etc/deployments
COPY ./entrypoint.sh /entrypoint.sh
COPY --from=builder /go/src/github.com/mendersoftware/deployments/deployments /usr/bin
CMD ["./entrypoint.sh"]  
ENTRYPOINT ["/entrypoint.sh", "--config", "/etc/deployments/config.yaml"]