#!/bin/bash
# Usage: ./package_ipk.sh <arch> <version>

ARCH=$1
VERSION=$2
PACKAGE_NAME="fakedns"

if [ -z "$ARCH" ] || [ -z "$VERSION" ]; then
    echo "Usage: $0 <arch> <version>"
    exit 1
fi

CUR_DIR=$(pwd)
DIST_DIR="$CUR_DIR/dist"
# Use a cleaner temporary directory name
BUILD_DIR="$CUR_DIR/ipk_build_root"

# 1. Setup Structure
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/usr/bin"
mkdir -p "$BUILD_DIR/etc/init.d"
mkdir -p "$BUILD_DIR/CONTROL"
mkdir -p "$DIST_DIR"

echo "Creating OpenWrt IPK for $PACKAGE_NAME ($ARCH)..."

# 2. Copy Binary
if [ -f "bin/$ARCH/fakedns" ]; then
    cp "bin/$ARCH/$PACKAGE_NAME" "$BUILD_DIR/usr/bin/"
    chmod +x "$BUILD_DIR/usr/bin/$PACKAGE_NAME"
else
    echo "Error: bin/$ARCH/$PACKAGE_NAME not found!"
    exit 1
fi

# 3. Copy OpenWrt Init Script
INIT_FILE="deployments/$PACKAGE_NAME/$PACKAGE_NAME.init"
if [ -f "$INIT_FILE" ]; then
    cp "$INIT_FILE" "$BUILD_DIR/etc/init.d/$PACKAGE_NAME"
    chmod 755 "$BUILD_DIR/etc/init.d/$PACKAGE_NAME"
    echo "Added Init Script: $PACKAGE_NAME"
else
    echo "Error: $INIT_FILE not found!"
    exit 1
fi

# 4. Create Control File
cat <<EOT > "$BUILD_DIR/CONTROL/control"
Package: $PACKAGE_NAME
Version: $VERSION
Section: net
Priority: optional
Architecture: $ARCH
Maintainer: a1ex2k
Depends: nftables, kmod-nft-core, kmod-nft-nat, kmod-nft-conntrack
Description: FakeDNS Core Server
 Redirects traffic via nftables DNAT for target subnets.
EOT

# 5. Create postinst
cat <<EOT > "$BUILD_DIR/CONTROL/postinst"
#!/bin/sh
if [ -z "\$IPKG_INSTROOT" ]; then
    /etc/init.d/$PACKAGE_NAME enable
    /etc/init.d/$PACKAGE_NAME start
    echo "$PACKAGE_NAME enabled and started."
fi
exit 0
EOT
chmod 755 "$BUILD_DIR/CONTROL/postinst"

# 6. Create prerm
cat <<EOT > "$BUILD_DIR/CONTROL/prerm"
#!/bin/sh
if [ -z "\$IPKG_INSTROOT" ]; then
    /etc/init.d/$PACKAGE_NAME stop
    /etc/init.d/$PACKAGE_NAME disable
    echo "$PACKAGE_NAME stopped and disabled."
fi
exit 0
EOT
chmod 755 "$BUILD_DIR/CONTROL/prerm"

# 7. Final Build Step (The Fix for 22.03+)
STAGING_DIR="$CUR_DIR/staging"
mkdir -p "$STAGING_DIR"
cd "$BUILD_DIR"

# Create data.tar.gz (Everything except the CONTROL folder)
tar --numeric-owner --owner=0 --group=0 -czf "$STAGING_DIR/data.tar.gz" . --exclude=CONTROL
cd "$BUILD_DIR/CONTROL"
tar --numeric-owner --owner=0 --group=0 -czf "$STAGING_DIR/control.tar.gz" .

# Create the final .ipk
cd "$STAGING_DIR"
echo "2.0" > debian-binary

# IMPORTANT: The file order MUST be: debian-binary, control.tar.gz, data.tar.gz
tar -czf "$DIST_DIR/${PACKAGE_NAME}_${VERSION}_${ARCH}.ipk" debian-binary control.tar.gz data.tar.gz

# 8. Cleanup
cd "$CUR_DIR"
rm -rf "$BUILD_DIR" "$STAGING_DIR"
echo "IPK complete: dist/${PACKAGE_NAME}_${VERSION}_${ARCH}.ipk"