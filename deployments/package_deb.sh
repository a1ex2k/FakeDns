#!/bin/bash
# Usage: ./package_deb.sh <arch> <version>
# This script bundles both fakedns and fakedns-webui into one .deb

ARCH=$1
VERSION=$2
PACKAGE_NAME="fakedns-full"

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

# 2. Copy Binaries
for COMPONENT in "fakedns" "fakedns-webui"; do
    if [ -f "bin/$ARCH/$COMPONENT" ]; then
        cp "bin/$ARCH/$COMPONENT" "$ROOT/usr/bin/"
        chmod +x "$ROOT/usr/bin/$COMPONENT"
        echo "Added Binary: $COMPONENT"
    else
        echo "Error: bin/$ARCH/$COMPONENT not found!"
        exit 1
    fi

    # 3. Copy Systemd Services
    SERVICE_FILE="deployments/$COMPONENT/$COMPONENT.service"
    if [ -f "$SERVICE_FILE" ]; then
        cp "$SERVICE_FILE" "$ROOT/lib/systemd/system/"
        echo "📄 Added Service: $COMPONENT.service"
    fi
done

# 4. Create Control File
cat <<EOT > "$ROOT/DEBIAN/control"
Package: $PACKAGE_NAME
Version: $VERSION
Section: net
Priority: optional
Architecture: $ARCH
Maintainer: a1ex2k
Depends: nftables, curl
Description: FakeDNS Suite (Core + WebUI)
 Redirects traffic via nftables DNAT and provides a management UI.
EOT

# 5. Create postinst Script (To enable/start services on install)
cat <<EOT > "$ROOT/DEBIAN/postinst"
#!/bin/sh
set -e
if [ "\$1" = "configure" ]; then
    systemctl daemon-reload >/dev/null 2>&1 || true
    
    # Enable and Start both services
    for svc in fakedns fakedns-webui; do
        systemctl enable "\$svc" >/dev/null 2>&1 || true
        systemctl start "\$svc" >/dev/null 2>&1 || true
    done
fi
exit 0
EOT
chmod 755 "$ROOT/DEBIAN/postinst"

# 6. Create prerm Script (To stop/disable services on removal)
cat <<EOT > "$ROOT/DEBIAN/prerm"
#!/bin/sh
set -e
if [ "\$1" = "remove" ]; then
    for svc in fakedns fakedns-webui; do
        systemctl stop "\$svc" >/dev/null 2>&1 || true
        systemctl disable "\$svc" >/dev/null 2>&1 || true
    done
fi
exit 0
EOT
chmod 755 "$ROOT/DEBIAN/prerm"

# 7. Build Package
dpkg-deb --build --root-owner-group "$ROOT" "$DIST_DIR/${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"

rm -rf "$ROOT"
echo "Package complete: dist/${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"