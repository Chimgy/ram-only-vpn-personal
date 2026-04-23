#!/bin/bash
set -e

# pack-initramfs.sh
cd "$(dirname "$0")"

# Ensure the target directory exists
mkdir -p pi-flash

echo "==> Packing initramfs..."

# 1. Enter the root
cd initramfs

# 2. Force creation of mount points just in case they were missing
mkdir -p proc sys dev mnt tmp

# 3. Pack it
# --null + -print0: Handles weird characters/spaces
# --quiet: Keeps the cpio log out of our binary stream
find . -print0 | cpio --null -ov -H newc --quiet | gzip -9 > ../pi-flash/initramfs.cpio.gz

cd ..

echo "==> Done. File size: $(du -sh pi-flash/initramfs.cpio.gz | cut -f1)"