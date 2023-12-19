FROM --platform=$BUILDPLATFORM golang:1.21.5 as builder
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
COPY ./ .

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target=/root/.cache/go-build \
    # when building aarch64 we have to target aarch64-linux-gnu-gcc compiler
    if [ "$TARGETARCH" = "arm64" ]; then \
    export CC=aarch64-linux-gnu-gcc; \
    export CC_FOR_TARGET=gcc-aarch64-linux-gnu; \
    export OBJDUMP=aarch64-linux-gnu-objdump; \
    export LD_LINUX="/lib/ld-linux-aarch64.so.1"; \
    export LIBS_PATH="/usr/lib/aarch64-linux-gnu/"; \
    else \
    export OBJDUMP=objdump; \
    export LD_LINUX="/lib64/ld-linux-x86-64.so.2"; \
    export CC=gcc; \
    export LIBS_PATH="/usr/lib/x86_64-linux-gnu/"; \
    fi && \
    CGO_ENABLED=1 GOOS=linux GOARCH=$TARGETARCH CC=$CC CC_FOR_TARGET=$CC_FOR_TARGET go build && \
    install -D /go/src/github.com/mendersoftware/deployments/deployments /mnt/usr/bin/deployments && \
    mkdir -p /mnt/etc /mnt/tmp && \
    echo 'nobody:x:65534:' > /mnt/etc/group && \
    echo 'nobody:!::0:::::' > /mnt/etc/shadow && \
    echo 'nobody:x:65534:65534:Nobody:/:' > /mnt/etc/passwd && \
    chown -R 65534:65534 /mnt && \
    install -D /etc/ssl/certs/ca-certificates.crt /mnt/etc/ssl/certs/ca-certificates.crt && \
    install -D ./config.yaml /mnt/etc/deployments/config.yaml && \
    install -D $LD_LINUX "/mnt${LD_LINUX}" && \
    $OBJDUMP -p ./deployments | sed -nE 's/^.*NEEDED.*?(lib.+$)/\1/p' | \
    while read lib; do install -D "${LIBS_PATH}${lib}" "/mnt${LIBS_PATH}${lib}"; done

FROM scratch
EXPOSE 8080
COPY --from=builder /mnt /
USER 65534
WORKDIR /etc/deployments

ENTRYPOINT ["/usr/bin/deployments", "--config", "/etc/deployments/config.yaml"]
