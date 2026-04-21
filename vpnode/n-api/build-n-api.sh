#!/bin/bash
set -e
# build.sh
# Cross-compiles n-api for arm64
# Run this if you change n-api, then run rebuild.sh
#
# Requires: go

echo "==> Building n-api"
GOOS=linux GOARCH=arm64 go build -o n-api .
echo ""
echo "==> Copying n-api to rootfs..."
cd ..
cp n-api/n-api rootfs/usr/local/bin/n-api
echo "==> Done. Run rebuild.sh to bake into rootfs.squash"