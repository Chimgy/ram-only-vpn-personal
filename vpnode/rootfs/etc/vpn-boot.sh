#!/bin/sh

# vpn-boot.sh
# VPN server boot script - runs as ::sysinit from inittab
# Sets up network, WireGuard server, NAT, and SSH
#
# All key material lives in /run/wg (tmpfs) and never touches disk
# Keys are generated fresh every boot and zeroed on shutdown
#
# QEMU vs Pi:
#   Pi 5:  eth0 (BCM ethernet)
#   QEMU:  eth0 or enp0s1 depending on virtio-net-device naming
#          check with: ip link show

log() { echo "[vpn-boot] $*"; }
fail() { log "ERROR: $*"; exit 1; }

# Step 1: network
log "Bringing up network..."
ip link set eth0 up || fail "Failed to bring up eth0"
udhcpc -i eth0 -q || fail "DHCP failed"
MY_IP=$(ip addr show eth0 | grep 'inet ' | awk '{print $2}' | cut -d/ -f1)
log "Network up: $MY_IP"

# DNS needed for ifconfig.me
log "Configuring DNS..."
echo "nameserver 8.8.8.8" > /etc/resolv.conf
echo "nameserver 1.1.1.1" >> /etc/resolv.conf

# Now sync clock so timestamps don't think its 1970....
log "Syncing clock via NTP..."
ntpd -n -q -p pool.ntp.org && log "Clock synced via NTP" || log "WARNING: NTP failed. Using Fallback Date"

# Step 2: WireGuard keypair into RAM
# Fresh keypair generated every boot
# Private key never written to disk (lives only in tmpfs)
# Public key must be distributed to clients after each reboot
# (see: ssh root@<ip> "cat /run/wg/server.pub")
#
# Production improvement: serve pubkey via HTTP on boot
# so clients can fetch it automatically 
# (RIGHT NOW SSH IS ENABLED BUT THAT IS OBVIOUSLY NOT GOING TO BE THE CASE IN PROD)

log "Generating WireGuard keypair in RAM..."
mkdir -p /run/wg
chmod 700 /run/wg

wg genkey > /run/wg/server.key
chmod 600 /run/wg/server.key
wg pubkey < /run/wg/server.key > /run/wg/server.pub
SERVER_PUBKEY=$(cat /run/wg/server.pub)
log "Server public key: $SERVER_PUBKEY"

# Step 3: WireGuard config into RAM
# Add one [Peer] block per client
# AllowedIPs = client's tunnel IP (must be unique per client)
#
# To add a client:
#   1. Generate keypair on client: wg genkey | tee client.key | wg pubkey
#   2. Add [Peer] block here with client pubkey
#   3. Give client: server pubkey + server LAN IP + their tunnel IP

# FOR REPRODUCING YOU WILL NEED TO CHANGE THIS wg0.conf TO MATCH YOUR LAN SET UP

cat > /run/wg/wg0.conf << WGEOF
[Interface]
PrivateKey = $(cat /run/wg/server.key)
ListenPort = 51820
WGEOF
chmod 600 /run/wg/wg0.conf

# Step 4: bring up WireGuard
log "Bringing up WireGuard server..."
ip link add dev wg0 type wireguard || fail "Failed to create wg0"
ip address add 10.8.0.1/24 dev wg0 || fail "Failed to assign tunnel IP"
wg setconf wg0 /run/wg/wg0.conf || fail "Failed to configure wg0"
ip link set up dev wg0 || fail "Failed to bring up wg0"
log "WireGuard listening on port 51820"

# Step 5: IP forwarding and NAT
# Enables client traffic to reach the internet via this server
# MASQUERADE: rewrites source IP so replies come back to us
# Alpine iptables uses nftables backend (requires kernel NF_TABLES=y)

log "Enabling IP forwarding and NAT..."
echo 1 > /proc/sys/net/ipv4/ip_forward

/usr/sbin/iptables -t nat -A POSTROUTING -s 10.8.0.0/24 -o eth0 -j MASQUERADE
/usr/sbin/iptables -A FORWARD -i wg0 -j ACCEPT
/usr/sbin/iptables -A FORWARD -i eth0 -o wg0 -m state --state RELATED,ESTABLISHED -j ACCEPT

log "NAT configured"

# Step 6: Run the n-api go file that will handle dynamic connections
# still needed for ssh
export VPN_LAN_IP=$MY_IP
# THIS IS AN IMPORTANT LINE LOL
export NODE_API_KEY=test123
/usr/local/bin/n-api &

# Step 6: SSH
# SSH host keys are baked into rootfs.squash at build time
# Allows remote management over LAN (not through VPN tunnel)
# Credentials: root / testpassword

log "Starting SSH..."

# 1. Create the privilege separation directory (Critical for OpenSSH)
mkdir -p /var/run/sshd
chmod 0755 /var/run/sshd

# 2. Ensure devpts is mounted for terminal access
mkdir -p /dev/pts
mount -t devpts devpts /dev/pts 2>/dev/null || log "devpts already mounted"

# 3. Start SSH with debug flags if you want to see why it fails in logs
log "Starting SSH..."
/usr/sbin/sshd && log "SSH ready: ssh root@$MY_IP" || log "WARNING: SSH failed to start"

# Step 7: ready
log "========================================="
log "VPN SERVER READY"
log "  LAN IP:      $MY_IP"
log "  Tunnel IP:   10.8.0.1/24"
log "  WireGuard:   port 51820"
log "  Public key:  $SERVER_PUBKEY"
log "========================================="