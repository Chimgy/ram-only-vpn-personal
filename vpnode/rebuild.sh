#!/bin/bash
set -e

# rebuild.sh
# Rebuilds rootfs.squash, signs it, updates boot.json and pi-flash/
# Run from vpnode/ directory
#
# Usage:
#   ./rebuild.sh           normal build, copies to pi-flash/
#   ./rebuild.sh --qemu    also updates boot.img for QEMU testing

cd "$(dirname "$0")"

QEMU_MODE=0
[ "$1" = "--qemu" ] && QEMU_MODE=1

echo "==> Building squashfs..."
sudo mksquashfs rootfs/ rootfs.squash -comp zstd -noappend

echo "==> Signing..."
HASH=$(sha256sum rootfs.squash | cut -d' ' -f1)
SIG=$(printf '%s' "$HASH" | openssl pkeyutl -sign -inkey signing.key -rawin | base64 -w0)

echo "==> Writing boot.json..."
cat > boot.json << JSONEOF
{
  "version": 1,
  "ospkg": {
    "filename": "rootfs.squash",
    "sha256": "$HASH",
    "compression": "zstd"
  },
  "signatures": [
    {
      "sig": "$SIG"
    }
  ]
}
JSONEOF

echo "==> Updating pi-flash/..."
sudo cp rootfs.squash boot.json pi-flash/

# QEMU 
# Updates boot.img for testing without real Pi hardware
# Requires boot.img to exist (created with: dd + mkfs.ext4)
if [ $QEMU_MODE -eq 1 ]; then
    echo "==> Updating boot.img for QEMU..."
    sudo mount -o loop boot.img /mnt/bootimg
    sudo cp rootfs.squash boot.json /mnt/bootimg/
    sudo umount /mnt/bootimg
    echo "==> QEMU image updated"
fi

echo "==> Done. Hash: $HASH"