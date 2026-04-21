#!/bin/bash
set -e

# pack-initramfs.sh
# Packs initramfs/ into initramfs.cpio.gz and copies to pi-flash/
#
# Run after any changes to initramfs/ (e.g. after keygen.sh)
#
# Usage:
#   ./pack-initramfs.sh

cd "$(dirname "$0")"

echo "==> Packing initramfs..."
cd initramfs
find . | cpio -o -H newc | gzip > ../pi-flash/initramfs.cpio.gz
cd ..

echo "==> Done."