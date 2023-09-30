FROM --platform=$BUILDPLATFORM golang:1.19.3-buster as builder
ARG TARGETARCH
# Multiple architectures: we must have every library for every
# architecture for every arch build
RUN dpkg --add-architecture arm64
RUN dpkg --add-architecture amd64
RUN apt-get update && \
    apt-get -y upgrade && \
    apt-get install -y \
       ca-certificates \
       libssl-dev \
       libssl-dev:arm64 \
       libssl-dev:amd64 \
       liblzma-dev:arm64 \
       liblzma-dev:amd64 \
       musl-dev:arm64 \
       musl-dev:amd64 \
       gcc \
       gcc-aarch64-linux-gnu \
       dpkg-dev \
       binutils-aarch64-linux-gnu \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /go/src/github.com/mendersoftware/deployments
RUN mkdir -p /etc_extra
RUN echo 'nobody:x:65534:' > /etc_extra/group
RUN echo 'nobody:!::0:::::' > /etc_extra/shadow
RUN echo 'nobody:x:65534:65534:Nobody:/:' > /etc_extra/passwd
RUN chown -R 65534:65534 /etc_extra
RUN mkdir -p /tmp_extra && chown 65534:65534 /tmp_extra
COPY ./ .

# when building aarch64 we have to target aarch64-linux-gnu-gcc compiler
RUN if [ "$TARGETARCH" = "arm64" ]; then CC=aarch64-linux-gnu-gcc && CC_FOR_TARGET=gcc-aarch64-linux-gnu; fi && \
  CGO_ENABLED=1 GOOS=linux GOARCH=$TARGETARCH CC=$CC CC_FOR_TARGET=$CC_FOR_TARGET go build

FROM scratch
EXPOSE 8080
COPY --from=builder /etc_extra/ /etc/
COPY --from=builder /lib/aarch64-linux-gnu/libc.so.6 /lib/aarch64-linux-gnu/libc.so.6
COPY --from=builder /lib/x86_64-linux-gnu/libc.so.6 /lib/x86_64-linux-gnu/libc.so.6
COPY --from=builder /lib64/ld-linux-x86-64.so.2 /lib64/ld-linux-x86-64.so.2
COPY --from=builder /lib/ld-linux-aarch64.so.1 /lib/ld-linux-aarch64.so.1
COPY --from=builder /lib/x86_64-linux-gnu/liblzma.so.5 /lib/x86_64-linux-gnu/liblzma.so.5
COPY --from=builder /lib/aarch64-linux-gnu/liblzma.so.5 /lib/aarch64-linux-gnu/liblzma.so.5
COPY --from=builder /usr/lib/x86_64-linux-gnu/libssl.so.1.1 /usr/lib/x86_64-linux-gnu/libssl.so.1.1 
COPY --from=builder /usr/lib/aarch64-linux-gnu/libssl.so.1.1 /usr/lib/aarch64-linux-gnu/libssl.so.1.1
COPY --from=builder /usr/lib/x86_64-linux-gnu/libcrypto.so.1.1 /usr/lib/x86_64-linux-gnu/libcrypto.so.1.1
COPY --from=builder /usr/lib/aarch64-linux-gnu/libcrypto.so.1.1 /usr/lib/aarch64-linux-gnu/libcrypto.so.1.1
COPY --from=builder /lib/aarch64-linux-gnu/libpthread.so.0 /lib/aarch64-linux-gnu/libpthread.so.0
COPY --from=builder /lib/x86_64-linux-gnu/libpthread.so.0 /lib/x86_64-linux-gnu/libpthread.so.0
COPY --from=builder /lib/aarch64-linux-gnu/libdl.so.2 /lib/aarch64-linux-gnu/libdl.so.2
COPY --from=builder /lib/x86_64-linux-gnu/libdl.so.2 /lib/x86_64-linux-gnu/libdl.so.2
COPY --from=builder /lib/x86_64-linux-gnu/libnss_dns.so.2 /lib/x86_64-linux-gnu/libnss_dns.so.2
COPY --from=builder /lib/aarch64-linux-gnu/libnss_dns.so.2 /lib/aarch64-linux-gnu/libnss_dns.so.2
COPY --from=builder /lib/x86_64-linux-gnu/libresolv.so.2 /lib/x86_64-linux-gnu/libresolv.so.2
COPY --from=builder /lib/aarch64-linux-gnu/libresolv.so.2 /lib/aarch64-linux-gnu/libresolv.so.2
COPY --chown=nobody --from=builder /tmp_extra/ /tmp/
USER 65534
WORKDIR /etc/deployments
COPY --from=builder --chown=nobody /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --chown=nobody ./config.yaml .
COPY --from=builder --chown=nobody /go/src/github.com/mendersoftware/deployments/deployments /usr/bin/

ENTRYPOINT ["/usr/bin/deployments", "--config", "/etc/deployments/config.yaml"]
