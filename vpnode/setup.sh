#!/bin/bash
set -e

# setup.sh
# Full build pipeline for ram-only-vpn-personal (and my very word explanation sry)
# Builds kernel, signs rootfs, packs initramfs, and places everything in pi-flash/
# so all you have to do is put it on your pi's SD card and turn it on.
#
# This script is the driver code for the following (modular) shell scripts (look inside each one for more detail)
# found in the vpnode directory:
# 	- env.sh  						1(Use everytime you reconfigure and make the kernel img)
#	- configure-kernel.sh 			1("")
# 	- keygen.sh  					2(Use only if you dont have keys/ set up)
# 	- initramfs-pack.sh 			3(Use whenever you change initramfs)
# 	- rebuild.sh 					4(Use whenever you change rootfs)
# 
# This script should really only be used for inital setup. Using the individual scripts will work.
# 
# 
# Prerequisites:
#  	To build the kernel image you need to have cross compiler etc to make it arm64 architecture
#   Run these if you don't have: ( if you're not sure just run the code and it will tell you your missing them )
# 	sudo apt install gcc-aarch64-linux-gnu bc bison flex libssl-dev make
#   Run from vpnode/ directory
#
# Usage:
#   ./setup.sh


# Prereq check:
echo "==> Checking dependencies..."
MISSING=0
for cmd in aarch64-linux-gnu-gcc make openssl mksquashfs cpio gzip; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "  [MISSING] $cmd"
        MISSING=1
    else
        echo "  [OK] $cmd"
    fi
done
if [ $MISSING -eq 1 ]; then
    echo ""
    echo "Install missing deps:"
    echo "  sudo apt install gcc-aarch64-linux-gnu bc bison flex libssl-dev make squashfs-tools"
    exit 1
fi

# 1 Set environment to build for arm64
echo ""
echo "==> Loading environment..."
source env.sh

# 2 Build the Kernel, First configure (no modules on inbuilt for ram only)
echo ""
echo "==> Configuring kernel..."
cd linux
make bcm2712_defconfig
../configure-kernel.sh
cd ..

# 2.5 Actually build the kernel 
echo ""
echo "==> Building Kernel..."
cd linux
make ARCH=arm64 CROSS_COMPILE=aarch64-linux-gnu- -j$(nproc) Image dtbs
cd ..
echo ""
echo "==> Copying kernel and DTB to pi-flash/..."
cp linux/arch/arm64/boot/Image pi-flash/kernel8.img
cp linux/arch/arm64/boot/dts/broadcom/bcm2712-rpi-5-b.dtb pi-flash/

# 3 Signing Keys (baking them into initramfs, your perso stamp that says trust this rootfs I made it)
echo ""
echo "==> Generating signing keypair..."
# Run without sudo, look inside keygen.sh for more info 
./keygen.sh

# 4 Initramfs
echo ""
echo "==> Packing initramfs..."
./pack-initramfs.sh

# 5 Rootfs

echo ""
echo "==> Building and signing rootfs..."
./rebuild.sh

# Fin
echo ""
echo "-----------------------------------------"
echo "pi-flash/ is ready to flash:"
echo ""
ls pi-flash/
echo ""
echo "Flash your SD card then boot your Pi."
echo "-----------------------------------------"