#!/bin/sh

# vpn-shutdown.sh
# Runs on poweroff/reboot via ::shutdown in inittab
# Zeros all key material before the system halts
#
# Two-pass wipe (zero then random) is defence
# against cold-boot attacks (tmpfs pages should
# be freed by the kernel anyway but explicit zeroing ensures
# no key material lingers in memory)

log() { echo "[vpn-shutdown] $*"; }

# Bring down WireGuard
if ip link show wg0 > /dev/null 2>&1; then
    ip link delete dev wg0
    log "wg0 removed"
fi

# Zero key material
if [ -f /run/wg/server.key ]; then
    LEN=$(wc -c < /run/wg/server.key)
    dd if=/dev/zero   of=/run/wg/server.key bs=1 count=$LEN conv=notrunc 2>/dev/null
    dd if=/dev/urandom of=/run/wg/server.key bs=1 count=$LEN conv=notrunc 2>/dev/null
    rm -f /run/wg/server.key /run/wg/wg0.conf
    log "Server key zeroed and removed"
fi

log "Shutdown clean"