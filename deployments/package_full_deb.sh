#!/bin/bash
# Usage: ./package_deb.sh <arch> <version>
# This script packages the C++ FakeDNS core into one .deb

ARCH=$1
VERSION=$2
PACKAGE_NAME="fakedns-lite"

CUR_DIR=$(pwd)
DIST_DIR="$CUR_DIR/dist"
ROOT="$CUR_DIR/tmp_deb_combined_${ARCH}"

# 1. Setup Structure
rm -rf "$ROOT"
mkdir -p "$ROOT/usr/bin"
mkdir -p "$ROOT/lib/systemd/system"
mkdir -p "$ROOT/DEBIAN"
mkdir -p "$DIST_DIR"

echo "Creating All-in-One Debian Package for $ARCH..."

# 2. Copy Binary
COMPONENT="fakedns-lite"
if [ -f "bin/$ARCH/$COMPONENT" ]; then
    cp "bin/$ARCH/$COMPONENT" "$ROOT/usr/bin/"
    chmod +x "$ROOT/usr/bin/$COMPONENT"
    echo "Added Binary: $COMPONENT"
else
    echo "Error: bin/$ARCH/$COMPONENT not found!"
    exit 1
fi

# 3. Copy Systemd Service
SERVICE_FILE="deployments/$COMPONENT/$COMPONENT.service"
if [ -f "$SERVICE_FILE" ]; then
    cp "$SERVICE_FILE" "$ROOT/lib/systemd/system/"
    echo "Added Service: $COMPONENT.service"
fi

# 4. Create Control File
cat <<EOT > "$ROOT/DEBIAN/control"
Package: $PACKAGE_NAME
Version: $VERSION
Section: net
Priority: optional
Architecture: $ARCH
Maintainer: a1ex2k
Depends: nftables, curl, libstdc++6
Description: FakeDNS Core Server (C++)
 Redirects traffic via nftables DNAT.
EOT

# 5. Create postinst Script (To enable/start service on install)
cat <<EOT > "$ROOT/DEBIAN/postinst"
#!/bin/sh
set -e
if [ "\$1" = "configure" ]; then
    systemctl daemon-reload >/dev/null 2>&1 || true
    systemctl enable "fakedns-lite" >/dev/null 2>&1 || true
    systemctl start "fakedns-lite" >/dev/null 2>&1 || true
fi
exit 0
EOT
chmod 755 "$ROOT/DEBIAN/postinst"

# 6. Create prerm Script (To stop/disable service on removal)
cat <<EOT > "$ROOT/DEBIAN/prerm"
#!/bin/sh
set -e
if [ "\$1" = "remove" ]; then
    systemctl stop "fakedns-lite" >/dev/null 2>&1 || true
    systemctl disable "fakedns-lite" >/dev/null 2>&1 || true
fi
exit 0
EOT
chmod 755 "$ROOT/DEBIAN/prerm"

# 7. Build Package
dpkg-deb --build --root-owner-group "$ROOT" "$DIST_DIR/${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"

rm -rf "$ROOT"
echo "Package complete: dist/${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"
