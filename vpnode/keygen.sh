#!/bin/bash
# Generate signing keypair for rootfs verification (this is your personal stamp of approval).
# Should only have to do this ONCE for initial setup.
# If this key doesn't match the one baked into initramfs key it will refuse to boot into the unstamped rootfs.

mkdir -p keys
openssl genpkey -algorithm ed25519 -out keys/signing.key
openssl pkey -in keys/signing.key -pubout -out keys/signing.pub
echo "Done. Keep keys/signing.key secret (never commit it)."

# Bake the new pubkey into initramfs (this is what the init ram filesystem will check before trusting and mounting rootfs).
cp keys/signing.pub initramfs/etc/ospkg_signing_root.pem
echo "ospkg_signing_root.pem updated in initramfs/etc, rebuild initramfs before flashing."
