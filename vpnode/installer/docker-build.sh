#!/bin/bash
# Runs inside the vpn-node-builder Docker container.
# The vpnode/ directory is mounted at /build.
set -e

# Strip Windows CRLF from all shell scripts — handles repos checked out on Windows
find /build -name "*.sh" | xargs sed -i 's/\r$//'

cd /build

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

# ── Signing keys (one-time) ──────────────────────────────────────────
if [ ! -f keys/signing.key ]; then
    echo "==> Generating signing keypair..."
    ./keygen.sh
fi

# ── Initramfs ────────────────────────────────────────────────────────
echo "==> Packing initramfs..."
./pack-initramfs.sh

# ── SSH password ─────────────────────────────────────────────────────
# TODO: implement SSH password setup
# SSH_PASS env var is available here when the user sets one in the wizard.
# Uncomment and test when ready:
#
# if [ -n "${SSH_PASS:-}" ]; then
#     HASH=$(openssl passwd -6 "$SSH_PASS")
#     printf 'root:%s:0:0:99999:7:::\n' "$HASH" > rootfs/etc/shadow
#     echo "==> SSH password configured"
# fi

# ── Rootfs ───────────────────────────────────────────────────────────
echo "==> Building and signing rootfs..."
./rebuild.sh

# ── Client app ───────────────────────────────────────────────────────
echo "==> Downloading vpn-client for ${CLIENT_OS}..."
mkdir -p /build/output
case "${CLIENT_OS}" in
  windows) CLIENT_FILE="vpn-client.exe" ;;
  darwin)  CLIENT_FILE="vpn-client-mac" ;;
  *)       CLIENT_FILE="vpn-client-linux" ;;
esac
curl -fL --progress-bar \
    -o "/build/output/${CLIENT_FILE}" \
    "https://github.com/Chimgy/ram-only-vpn-personal/releases/latest/download/${CLIENT_FILE}"
echo "==> Client written to output/${CLIENT_FILE}"

echo ""
echo "========================================="
echo " Build complete."
echo " pi-flash/  — flash to SD card"
echo " output/    — install on your machine"
echo "========================================="
