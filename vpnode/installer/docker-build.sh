#!/bin/bash
# Runs inside the vpn-node-builder Docker container.
# Source files are baked into the image at /build.
# Output is written to /output (mounted from the host).
set -e

# Strip Windows CRLF from all shell scripts
find /build -name "*.sh" | xargs sed -i 's/\r$//'

cd /build

# ── Write node config from env vars ──────────────────────────────────
echo "==> Writing node config..."
mkdir -p /build/rootfs/etc/n-api
{
    echo "NODE_API_KEY=${NODE_API_KEY}"
    echo "API_PORT=8080"
    if [ -n "${DUCKDNS_TOKEN:-}" ]; then
        echo "DUCKDNS_TOKEN=${DUCKDNS_TOKEN}"
        echo "DUCKDNS_DOMAIN=${DUCKDNS_DOMAIN}"
    fi
    if [ -n "${STATIC_IP:-}" ]; then
        echo "STATIC_IP=${STATIC_IP}"
    fi
} > /build/rootfs/etc/n-api/config.env
chmod 600 /build/rootfs/etc/n-api/config.env

# ── Kernel (pre-built, downloaded from GitHub Releases) ──────────────
mkdir -p pi-flash
if [ ! -f pi-flash/kernel8.img ] || [ ! -f pi-flash/bcm2712-rpi-5-b.dtb ]; then
    echo "==> Downloading pre-built kernel from GitHub Releases..."
    curl -fL --progress-bar \
        -o pi-flash/kernel8.img \
        "https://github.com/Chimgy/ram-only-vpn-personal/releases/download/v1.0.0/kernel8.img"
    curl -fL --progress-bar \
        -o pi-flash/bcm2712-rpi-5-b.dtb \
        "https://github.com/Chimgy/ram-only-vpn-personal/releases/download/v1.0.0/bcm2712-rpi-5-b.dtb"
    echo "==> Kernel downloaded"
else
    echo "==> Kernel already present in pi-flash/, skipping download"
fi

# ── Cross-compile n-api for arm64 ────────────────────────────────────
echo "==> Building n-api for arm64..."
cd n-api
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o ../rootfs/usr/local/bin/n-api .
cd ..
echo "==> n-api binary written to rootfs/usr/local/bin/n-api"

# ── Signing keys ──────────────────────────────────────────────────────
echo "==> Generating signing keypair..."
./keygen.sh

# ── Initramfs ────────────────────────────────────────────────────────
echo "==> Packing initramfs..."
./pack-initramfs.sh

# ── Rootfs ───────────────────────────────────────────────────────────
echo "==> Building and signing rootfs..."
./rebuild.sh

# ── Copy output to /output (mounted from host) ────────────────────────
echo "==> Copying output..."
mkdir -p /output
cp -r /build/pi-flash /output/pi-flash

echo ""
echo "========================================="
echo " Build complete."
echo " pi-flash/  — flash ALL files to a FAT32 SD card"
echo "========================================="
