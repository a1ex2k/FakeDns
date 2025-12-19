#!/bin/bash
# Usage: ./package_ipk.sh <arch> <version>

ARCH=$1
VERSION=$2
PACKAGE_NAME="fakedns"

if [ -z "$ARCH" ] || [ -z "$VERSION" ]; then
    echo "Usage: $0 <arch> <version>"
    exit 1
fi

# Maps GitHub/Go arch names to OpenWrt opkg names
PKG_ARCH=$ARCH
case "$ARCH" in
    "arm64")
        PKG_ARCH="aarch64_generic"
        ;;
    "amd64")
        PKG_ARCH="x86_64"
        ;;
    "mipsel_24kc")
        PKG_ARCH="mipsel_24kc"
        ;;
esac

CUR_DIR=$(pwd)
DIST_DIR="$CUR_DIR/dist"
BUILD_DIR="$CUR_DIR/ipk_build_root"

# 1. Setup Structure
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/usr/bin"
mkdir -p "$BUILD_DIR/etc/init.d"
mkdir -p "$BUILD_DIR/etc/config"
mkdir -p "$BUILD_DIR/CONTROL"
mkdir -p "$DIST_DIR"

echo "Creating OpenWrt IPK for $PACKAGE_NAME ($PKG_ARCH)..."

# 2. Copy Binary
if [ -f "bin/$ARCH/$PACKAGE_NAME" ]; then
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
    # Fix potential Windows line endings and ensure path is /etc/rc.common
    sed -i 's/\r$//' "$BUILD_DIR/etc/init.d/$PACKAGE_NAME"
    sed -i 's|/etc/init.d/rc.common|/etc/rc.common|g' "$BUILD_DIR/etc/init.d/$PACKAGE_NAME"
    chmod 755 "$BUILD_DIR/etc/init.d/$PACKAGE_NAME"
else
    echo "Error: $INIT_FILE not found!"
    exit 1
fi

# 3.5 Copy Configuration File (The "OpenWrt Way")
CONFIG_SRC="deployments/fakedns.uciconf"
if [ -f "$CONFIG_SRC" ]; then
    cp "$CONFIG_SRC" "$BUILD_DIR/etc/config/$PACKAGE_NAME"
    echo "/etc/config/$PACKAGE_NAME" > "$BUILD_DIR/CONTROL/conffiles"
else
    echo "Warning: $CONFIG_SRC not found, no default config will be bundled."
fi

# 4. Create Control File
cat <<EOT > "$BUILD_DIR/CONTROL/control"
Package: $PACKAGE_NAME
Version: $VERSION
Section: net
Priority: optional
Architecture: $PKG_ARCH
Maintainer: a1ex2k
Depends: nftables, kmod-nft-nat
Description: FakeDNS Core Server
 Redirects traffic via nftables DNAT for target subnets.
EOT

# 5. Create postinst (Simplified)
cat <<EOT > "$BUILD_DIR/CONTROL/postinst"
#!/bin/sh
if [ -z "\$IPKG_INSTROOT" ]; then
    # OpenWrt's opkg handles /etc/config automatically via the 'conffiles' list.
    # If the user modified the config, opkg installs the new one as fakedns-opkg.
    
    /etc/init.d/$PACKAGE_NAME enable
    /etc/init.d/$PACKAGE_NAME restart
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
fi
exit 0
EOT
chmod 755 "$BUILD_DIR/CONTROL/prerm"

# 7. Final Build Step
STAGING_DIR="$CUR_DIR/staging"
mkdir -p "$STAGING_DIR"

cd "$BUILD_DIR"
# Create data.tar.gz
tar --numeric-owner --owner=0 --group=0 --exclude="./CONTROL" -czf "$STAGING_DIR/data.tar.gz" .

# Create control.tar.gz
cd "$BUILD_DIR/CONTROL"
tar --numeric-owner --owner=0 --group=0 -czf "$STAGING_DIR/control.tar.gz" .

# Create the final .ipk
cd "$STAGING_DIR"
echo "2.0" > debian-binary
tar -czf "$DIST_DIR/${PACKAGE_NAME}_${VERSION}_${PKG_ARCH}.ipk" debian-binary control.tar.gz data.tar.gz

# 8. Cleanup
cd "$CUR_DIR"
rm -rf "$BUILD_DIR" "$STAGING_DIR"
echo "✅ IPK complete: dist/${PACKAGE_NAME}_${VERSION}_${PKG_ARCH}.ipk"