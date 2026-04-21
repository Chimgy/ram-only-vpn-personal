#!/bin/bash
set -e

# configure-kernel.sh
# Configures the Pi 5 kernel for noram-vpn
# Run from linux/ directory after: make bcm2712_defconfig
#
# Usage:
#   source ../env.sh
#   make bcm2712_defconfig
#   cd .. && ./configure-kernel.sh && cd linux
#   make -j$(nproc) Image dtbs

CONF=.config
echo "==> Applying kernel config for noram-vpn..."

# Architecture
# For Pi 5: native page size is 16K
./scripts/config --disable CONFIG_ARM64_4K_PAGES
./scripts/config --enable  CONFIG_ARM64_16K_PAGES
# For QEMU: swap these two lines (QEMU only supports 4K)
# ./scripts/config --disable CONFIG_ARM64_16K_PAGES
# ./scripts/config --enable CONFIG_ARM64_4K_PAGES

# Filesystems
# overlayfs: RAM (tmpfs) upper layer over read-only squashfs lower
# squashfs: compressed read-only OS image format
./scripts/config --enable CONFIG_OVERLAY_FS
./scripts/config --enable CONFIG_SQUASHFS
./scripts/config --enable CONFIG_SQUASHFS_ZSTD

# Virtio drivers (QEMU only)
# Not needed on real Pi hardware
# Allows testing in QEMU with same kernel
# ./scripts/config --enable CONFIG_VIRTIO_BLK
# ./scripts/config --enable CONFIG_VIRTIO_PCI
# ./scripts/config --enable CONFIG_VIRTIO_MMIO
# ./scripts/config --enable CONFIG_VIRTIO_NET

# WireGuard
# IPV6 must be =y not =m — if =m it forces WireGuard to =m too
./scripts/config --enable CONFIG_IPV6
./scripts/config --enable CONFIG_NET_UDP_TUNNEL
./scripts/config --enable CONFIG_NET_IP_TUNNEL
./scripts/config --enable CONFIG_WIREGUARD

# WireGuard crypto primitives
./scripts/config --enable CONFIG_CRYPTO_LIB_CHACHA
./scripts/config --enable CONFIG_CRYPTO_LIB_POLY1305
./scripts/config --enable CONFIG_CRYPTO_LIB_CURVE25519
./scripts/config --enable CONFIG_CRYPTO_LIB_CHACHA20POLY1305
./scripts/config --enable CONFIG_CRYPTO_CHACHA20
./scripts/config --enable CONFIG_CRYPTO_POLY1305
./scripts/config --enable CONFIG_CRYPTO_CHACHA20POLY1305
./scripts/config --enable CONFIG_CRYPTO_CHACHA20_NEON
./scripts/config --enable CONFIG_CRYPTO_POLY1305_NEON

# Netfilter / NAT
# Required for iptables MASQUERADE (client traffic forwarding)
# Alpine 3.21 iptables uses nftables backend (iptables-nft)
# NF_CONNTRACK must be =y (not =m) or NAT silently fails
./scripts/config --enable CONFIG_BRIDGE
./scripts/config --enable CONFIG_NF_CONNTRACK
./scripts/config --enable CONFIG_NF_NAT
./scripts/config --enable CONFIG_NF_NAT_MASQUERADE

# nftables engine (Alpine iptables backend)
./scripts/config --enable CONFIG_NF_TABLES
./scripts/config --enable CONFIG_NF_TABLES_INET
./scripts/config --enable CONFIG_NF_TABLES_IPV4
./scripts/config --enable CONFIG_NF_TABLES_IPV6
./scripts/config --enable CONFIG_NFT_COMPAT
./scripts/config --enable CONFIG_NFT_NAT
./scripts/config --enable CONFIG_NFT_MASQ

# Legacy iptables (IPv4)
./scripts/config --enable CONFIG_IP_NF_IPTABLES
./scripts/config --enable CONFIG_IP_NF_FILTER
./scripts/config --enable CONFIG_IP_NF_NAT
./scripts/config --enable CONFIG_IP_NF_TARGET_MASQUERADE

# Legacy iptables (IPv6)
./scripts/config --enable CONFIG_IP6_NF_IPTABLES
./scripts/config --enable CONFIG_IP6_NF_FILTER
./scripts/config --enable CONFIG_IP6_NF_NAT
./scripts/config --enable CONFIG_IP6_NF_TARGET_MASQUERADE

# Resolve dependencies
echo "==> Running olddefconfig to resolve dependencies..."
make ARCH=arm64 CROSS_COMPILE=aarch64-linux-gnu- olddefconfig

# Verify critical configs
echo "==> Verifying critical configs..."
FAIL=0
for cfg in \
    CONFIG_OVERLAY_FS \
    CONFIG_SQUASHFS \
    CONFIG_IPV6 \
    CONFIG_WIREGUARD \
    CONFIG_NF_CONNTRACK \
    CONFIG_NF_NAT \
    CONFIG_NF_TABLES \
    CONFIG_NFT_COMPAT; do
    val=$(grep "^$cfg=" .config | cut -d= -f2)
    if [ "$val" = "y" ]; then
        echo "  [OK] $cfg=y"
    else
        echo "  [FAIL] $cfg=$val (expected y)"
        FAIL=1
    fi
done

if [ $FAIL -eq 1 ]; then
    echo "==> Some configs failed verification. Check dependencies."
    exit 1
fi

echo "==> All configs verified. Ready to build:"
echo "    make ARCH=arm64 CROSS_COMPILE=aarch64-linux-gnu- -j\$(nproc) Image dtbs"