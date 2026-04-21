#!/bin/bash
set -e

# setup-ssh.sh
# Sets the root SSH password for the VPN node
# Run before rebuild.sh, password gets baked into rootfs.squash
#
# Usage:
#   ./setup-ssh.sh

cd "$(dirname "$0")"

read -s -p "Enter root SSH password: " PASSWORD
echo
read -s -p "Confirm password: " PASSWORD2
echo

if [ "$PASSWORD" != "$PASSWORD2" ]; then
    echo "Passwords don't match."
    exit 1
fi

HASH=$(openssl passwd -6 "$PASSWORD")

cat > rootfs/etc/shadow << EOF
root:$HASH:0:0:99999:7:::
EOF

echo "==> Password set. Run rebuild.sh to bake into rootfs."