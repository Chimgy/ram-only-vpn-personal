#!/bin/bash
# will need this to detect the platform eventually but not yet
cp ../build/bin/vpn-client linux/usr/local/bin/vpn-client
chmod +x linux/usr/local/bin/vpn-client
chmod +x linux/DEBIAN/postinst
dpkg-deb --build linux vpn-client.deb
