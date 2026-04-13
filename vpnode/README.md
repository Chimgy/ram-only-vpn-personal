# vpnode — RAM-Only VPN Server

A verified, RAM-only WireGuard VPN server for Raspberry Pi 5.

Every boot:
- OS image is cryptographically verified (SHA256 + Ed25519) before mounting
- Filesystem runs entirely from RAM (overlayfs over squashfs)
- WireGuard keypair generated fresh into tmpfs — never touches disk
- All state gone on poweroff — no logs, no keys, no trace

---

## Architecture

```
SD card (read-only after boot):
  kernel8.img         Pi kernel
  bcm2712-rpi-5-b.dtb Pi 5 device tree
  initramfs.cpio.gz   Early userspace (verifies OS before mounting)
  rootfs.squash       Signed, compressed OS image
  boot.json           SHA256 hash + Ed25519 signature

RAM (tmpfs, wiped on poweroff):
  /run/wg/            WireGuard keys and config
  /run/attest/        Boot attestation hash
  overlay upper       Any runtime writes
```

---

## Prerequisites

```
sudo apt install gcc-aarch64-linux-gnu make bc bison flex \
                 libssl-dev libncurses-dev

sudo apt install squashfs-tools cpio openssl qemu-system-arm \
                 qemu-user-static docker.io

sudo usermod -aG docker $USER && newgrp docker
```

---

## First Time Setup

### 1. Clone Pi kernel

```
git clone --depth=1 --branch rpi-6.12.y \
    https://github.com/raspberrypi/linux.git
```

### 2. Build static ARM64 busybox

```
wget https://busybox.net/downloads/busybox-1.36.1.tar.bz2
tar xjf busybox-1.36.1.tar.bz2
cd busybox-1.36.1

make ARCH=arm64 CROSS_COMPILE=aarch64-linux-gnu- defconfig
sed -i 's/# CONFIG_STATIC is not set/CONFIG_STATIC=y/' .config
make ARCH=arm64 CROSS_COMPILE=aarch64-linux-gnu- -j$(nproc)

cp busybox ../initramfs/bin/busybox
cd ..
```

### 3. Build static ARM64 OpenSSL

```
wget https://www.openssl.org/source/openssl-3.3.0.tar.gz
tar xzf openssl-3.3.0.tar.gz
cd openssl-3.3.0

./Configure linux-aarch64 \
    --cross-compile-prefix=aarch64-linux-gnu- \
    no-shared no-dso -static \
    --prefix=/tmp/openssl-arm64-static

make -j$(nproc)
make install_sw

cp /tmp/openssl-arm64-static/bin/openssl ../initramfs/bin/openssl
cd ..
```

### 4. Build Alpine ARM64 rootfs

```
docker run --platform linux/arm64 --name alpine-vpn alpine:3.21 \
    sh -c "apk add wireguard-tools openssl openssh iptables && echo done"

docker export alpine-vpn > rootfs-aarch64.tar
docker rm alpine-vpn

mkdir -p rootfs
sudo tar xf rootfs-aarch64.tar -C rootfs/
```

Configure rootfs:

```
# Set root password
sudo chroot rootfs /bin/sh -c "echo 'root:vpnprototype' | chpasswd"

# Generate SSH host keys
sudo ssh-keygen -t ed25519 -f rootfs/etc/ssh/ssh_host_ed25519_key -N ""
sudo ssh-keygen -t rsa -b 4096 -f rootfs/etc/ssh/ssh_host_rsa_key -N ""

# Allow root SSH login
sudo sh -c "printf 'PermitRootLogin yes\nPasswordAuthentication yes\n' >> rootfs/etc/ssh/sshd_config"
sudo sed -i 's/UsePAM yes/UsePAM no/' rootfs/etc/ssh/sshd_config
```

### 5. Generate signing keypair

```
openssl genpkey -algorithm ed25519 -out signing.key
openssl pkey -in signing.key -pubout -out signing.pub
cp signing.pub initramfs/etc/ospkg_signing_root.pem
```

> signing.key is your root of trust — keep it safe, never commit it.

### 6. Build kernel

```
source env.sh
cd linux
make bcm2712_defconfig
cd ..
./configure-kernel.sh
cd linux
make -j$(nproc) Image dtbs
cd ..
```

### 7. Build and sign OS image

```
./rebuild.sh
```

---

## Flashing to Pi 5

### Prepare SD card

```
# Find SD card device
lsblk

# Wipe and create single FAT32 partition
sudo fdisk /dev/sdX   # o -> n -> t -> 0b -> w
sudo mkfs.vfat -F 32 -n "VPNBOOT" /dev/sdX1
sudo mount /dev/sdX1 /mnt/pi_boot
```

### Copy files

```
sudo cp linux/arch/arm64/boot/Image /mnt/pi_boot/kernel8.img
sudo cp linux/arch/arm64/boot/dts/broadcom/bcm2712-rpi-5-b.dtb /mnt/pi_boot/
sudo cp initramfs.cpio.gz rootfs.squash boot.json /mnt/pi_boot/
sudo cp pi-flash/config.txt pi-flash/cmdline.txt /mnt/pi_boot/
sudo sync && sudo umount /mnt/pi_boot
```

### Boot and connect

```
# SSH into Pi after ~30 seconds
ssh root@<pi-lan-ip>
# password: vpnprototype

# Check WireGuard is running
wg show

# Get server public key for clients
cat /run/wg/server.pub
```

---

## Connecting a Client

Create a WireGuard config on the client machine:

```
[Interface]
PrivateKey = <client-private-key>
Address = 10.8.0.2/24

[Peer]
PublicKey = <get from: ssh root@pi-ip "cat /run/wg/server.pub">
Endpoint = <pi-lan-ip>:51820
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25
```

> The server generates a new keypair every boot.
> Run ssh root@pi-ip "cat /run/wg/server.pub" after each reboot
> to get the current public key.

```
sudo wg-quick up ./client.conf
ping 10.8.0.1
curl https://checkip.amazonaws.com
```

---

## Adding Clients to the Server

Edit rootfs/etc/vpn-boot.sh and add a [Peer] block per client:

```
[Peer]
PublicKey = <client-public-key>
AllowedIPs = 10.8.0.x/32
```

Then rebuild and reflash:

```
./rebuild.sh
sudo cp rootfs.squash boot.json /mnt/pi_boot/
sudo sync && sudo umount /mnt/pi_boot
```

---

## QEMU Testing (no Pi needed)

Switch kernel to 4K pages in configure-kernel.sh:

```
# swap these two lines:
./scripts/config --enable  CONFIG_ARM64_4K_PAGES
./scripts/config --disable CONFIG_ARM64_16K_PAGES
```

Also change BOOT_DEV in initramfs/init:

```
BOOT_DEV=/dev/vda
```

Create QEMU disk image:

```
dd if=/dev/zero of=boot.img bs=1M count=200
mkfs.ext4 boot.img
sudo mount -o loop boot.img /mnt/bootimg
sudo cp rootfs.squash boot.json /mnt/bootimg/
sudo umount /mnt/bootimg
```

Boot:

```
./boot.sh
```

Or manually:

```
qemu-system-aarch64 \
  -machine virt \
  -cpu cortex-a72 \
  -m 1G \
  -kernel linux/arch/arm64/boot/Image \
  -initrd initramfs.cpio.gz \
  -drive file=boot.img,format=raw,if=none,id=hd0 \
  -device virtio-blk-device,drive=hd0 \
  -netdev user,id=net0 \
  -device virtio-net-device,netdev=net0 \
  -append "console=ttyAMA0,115200 earlycon=pl011,0x9000000 rdinit=/init" \
  -nographic
```

QEMU controls:

```
Ctrl+A then X    exit
pkill qemu-system-aarch64    kill from another terminal
```

---

## Rebuild Reference

| Changed           | Command                                                                 |
|-------------------|-------------------------------------------------------------------------|
| rootfs files      | ./rebuild.sh                                                            |
| initramfs/init    | cd initramfs && find . | cpio -H newc -o | gzip > ../initramfs.cpio.gz  |
| kernel config     | source env.sh && cd linux && make bcm2712_defconfig && cd .. && ./configure-kernel.sh && cd linux && make -j$(nproc) Image dtbs |
| Flash rootfs      | sudo cp rootfs.squash boot.json /mnt/pi_boot/ && sudo sync             |
| Flash kernel      | sudo cp linux/arch/arm64/boot/Image /mnt/pi_boot/kernel8.img && sudo sync |

---

## Kernel Config Development

```
source env.sh
cd linux

# Check a config value
grep "CONFIG_WIREGUARD" .config
#   not set  -> disabled
#   =m       -> module (NOT usable, causes silent failures)
#   =y       -> built-in (always usable)

# Change values
./scripts/config --set-val CONFIG_WIREGUARD y
./scripts/config --enable  CONFIG_OVERLAY_FS
./scripts/config --disable CONFIG_ARM64_4K_PAGES

# Resolve dependencies
make oldconfig

# Show dependencies for a module
grep -A 20 "config WIREGUARD" drivers/net/Kconfig

# Interactive editor (last resort)
make menuconfig
```

> Common gotcha: if make oldconfig keeps reverting a config to =m,
> a dependency is still =m. See KERNELCONFIGCHANGES.md for the
> full dependency map and what was needed for this project.

---

## SSH Reference

```
ssh root@<pi-lan-ip>
# password: vpnprototype

cat /run/wg/server.pub                          # current WireGuard pubkey
wg show                                         # tunnel status
/usr/sbin/iptables -t nat -L POSTROUTING -v    # NAT rules
cat /run/attest/ospkg.sha256                    # verified boot hash
```

---

## Security Notes

- signing.key is the root of trust — keep it offline, never commit
- SSH host keys are baked into rootfs.squash — stable across boots
- WireGuard server key changes every boot — clients must update pubkey
- Root password vpnprototype is for development only
- /run/attest/ospkg.sha256 contains the verified boot hash
- Attestation submission to transparency log not yet implemented
- Pi EEPROM bootloader runs before verification — trust gap exists until Pi OTP secure boot is configured

---

## What's Not Done Yet

- [ ] Attestation submission to Rekor transparency log
- [ ] Automatic client pubkey distribution via HTTP on boot
- [ ] Client config auto-update when server key rotates
- [ ] Production hardening (SSH key auth only, no password)
- [ ] Pi OTP secure boot (closes EEPROM trust gap)
