#!/bin/bash
# Run once after installing vpn-client.app on macOS.
# Grants the app permission to run wg-quick without a password prompt.

set -e

WG_QUICK=$(which wg-quick 2>/dev/null || echo "")

if [ -z "$WG_QUICK" ]; then
    echo "ERROR: wg-quick not found. Install WireGuard first:"
    echo "  brew install wireguard-tools"
    exit 1
fi

SUDOERS_FILE="/etc/sudoers.d/vpnclient"

echo "ALL ALL=(root) NOPASSWD: $WG_QUICK up /tmp/vpnclient.conf" | sudo tee "$SUDOERS_FILE" > /dev/null
echo "ALL ALL=(root) NOPASSWD: $WG_QUICK down /tmp/vpnclient.conf" | sudo tee -a "$SUDOERS_FILE" > /dev/null
sudo chmod 440 "$SUDOERS_FILE"

echo "Done. vpn-client can now connect without password prompts."
