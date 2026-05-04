#!/bin/bash
set -e

PLATFORM=$(uname -s)

if [ "$PLATFORM" = "Linux" ]; then
    cp ../build/bin/vpn-client linux/usr/local/bin/vpn-client
    chmod +x linux/usr/local/bin/vpn-client
    chmod +x linux/DEBIAN/postinst
    dpkg-deb --build linux vpn-client.deb
    echo "Built: vpn-client.deb"

elif [ "$PLATFORM" = "Darwin" ]; then
    # Copy the .app bundle into the pkg payload
    mkdir -p mac/root/Applications
    rm -rf mac/root/Applications/vpn-client.app
    cp -R ../build/bin/vpn-client.app mac/root/Applications/vpn-client.app

    # Generate component plist then disable relocation so the installer
    # always puts the app in /Applications, not next to an existing copy.
    pkgbuild --analyze --root mac/root mac/components.plist
    /usr/libexec/PlistBuddy -c \
        "Set :0:BundleIsRelocatable false" mac/components.plist

    pkgbuild \
        --root mac/root \
        --scripts mac/scripts \
        --component-plist mac/components.plist \
        --identifier com.ramonvpn.vpnclient \
        --version 1.0 \
        --install-location / \
        vpn-client.pkg

    echo "Built: vpn-client.pkg"

else
    echo "Unsupported platform: $PLATFORM"
    exit 1
fi
