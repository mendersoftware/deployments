FROM golang:1.19.3-buster as builder
# Multiple architectures: we must have every library for every
# architecture for every arch build
RUN apt-get update && \
    apt-get -y upgrade && \
    apt-get install -y \
       ca-certificates \
       libssl-dev \
       liblzma-dev \
       gcc
WORKDIR /go/src/github.com/mendersoftware/deployments
RUN mkdir -p /scratch/etc/deployments /scratch/tmp /scratch/usr/bin \
    && echo 'nobody:x:65534:' > /scratch/etc/group \
    && echo 'nobody:!::0:::::' > /scratch/etc/shadow \
    && echo 'nobody:x:65534:65534:Nobody:/:' > /scratch/etc/passwd
COPY ./ .
RUN CGO_ENABLED=1 go build
RUN ldd ./deployments | sed -nE 's:^[^/]*(/[^ $]+).*$:\1:p' \
    | xargs tar -cvf libs.tar \
    && tar -xvf ./libs.tar -C /scratch \
    && install -D /etc/ssl/certs/ca-certificates.crt /scratch/etc/ssl/certs/ca-certificates.crt \
    && install -D config.yaml /scratch/etc/deployments/config.yaml \
    && install -sD deployments /scratch/usr/bin/deployments

FROM scratch
EXPOSE 8080
COPY --from=builder /scratch /
USER 65534:65534
WORKDIR /etc/deployments

ENTRYPOINT ["/usr/bin/deployments", "--config", "/etc/deployments/config.yaml"]
