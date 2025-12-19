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
        # Already an OpenWrt name
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
mkdir -p "$BUILD_DIR/CONTROL"
mkdir -p "$DIST_DIR"

echo "Creating OpenWrt IPK for $PACKAGE_NAME ($PKG_ARCH)..."

# 2. Copy Binary (Uses original $ARCH to find the file)
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
    chmod 755 "$BUILD_DIR/etc/init.d/$PACKAGE_NAME"
    echo "Added Init Script: $PACKAGE_NAME"
else
    echo "Error: $INIT_FILE not found!"
    exit 1
fi

# 4. Create Control File (Uses translated $PKG_ARCH)
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

# 7. Final Build Step
STAGING_DIR="$CUR_DIR/staging"
mkdir -p "$STAGING_DIR"
cd "$BUILD_DIR"

tar --numeric-owner --owner=0 --group=0 -czf "$STAGING_DIR/data.tar.gz" . --exclude=CONTROL
cd "$BUILD_DIR/CONTROL"
tar --numeric-owner --owner=0 --group=0 -czf "$STAGING_DIR/control.tar.gz" .

cd "$STAGING_DIR"
echo "2.0" > debian-binary
tar -czf "$DIST_DIR/${PACKAGE_NAME}_${VERSION}_${PKG_ARCH}.ipk" debian-binary control.tar.gz data.tar.gz

cd "$CUR_DIR"
rm -rf "$BUILD_DIR" "$STAGING_DIR"
echo "✅ IPK complete: dist/${PACKAGE_NAME}_${VERSION}_${PKG_ARCH}.ipk"