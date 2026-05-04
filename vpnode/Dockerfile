FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    # Pi 5 arm64 cross-compiler
    gcc-aarch64-linux-gnu \
    make bc bison flex libssl-dev \
    # Rootfs
    squashfs-tools \
    # Initramfs
    cpio gzip \
    # Signing + SSH password hashing
    openssl \
    # Kernel download from GitHub Releases
    curl \
    # n-api cross-compile
    golang-go \
    # rebuild.sh uses sudo mksquashfs
    sudo \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build
