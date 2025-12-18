#!/bin/bash
# Usage: ./package.sh <component> <arch> <version> <type>
# Example: ./package.sh fakedns amd64 1.0.0 deb
# Example: ./package.sh fakedns-webui mipsel_24kc 1.0.0 ipk

#!/bin/bash

COMPONENT=$1
ARCH=$2
VERSION=$3
TYPE=$4

# Setup working directories
CUR_DIR=$(pwd)
DIST_DIR="$CUR_DIR/dist"
ROOT="$CUR_DIR/tmp_${COMPONENT}_${TYPE}_${ARCH}"

rm -rf "$ROOT"
mkdir -p "$ROOT/usr/bin"
mkdir -p "$ROOT/lib/systemd/system"
mkdir -p "$ROOT/CONTROL"
mkdir -p "$DIST_DIR"

echo "Packaging $COMPONENT ($ARCH) into $TYPE..."

# Copy Binaries
if [ -f "bin/$ARCH/$COMPONENT" ]; then
    cp "bin/$ARCH/$COMPONENT" "$ROOT/usr/bin/"
    chmod +x "$ROOT/usr/bin/$COMPONENT"
else
    echo "❌ Error: Binary bin/$ARCH/$COMPONENT not found!"
    exit 1
fi

# Copy Services
SERVICE_FILE="deployments/$COMPONENT/$COMPONENT.service"
if [ -f "$SERVICE_FILE" ]; then
    cp "$SERVICE_FILE" "$ROOT/lib/systemd/system/"
    echo "📄 Included service: $SERVICE_FILE"
fi

# Architecture Mapping
PKG_ARCH=$ARCH
if [ "$TYPE" == "deb" ] && [ "$ARCH" == "mipsel_24kc" ]; then
    PKG_ARCH="mipsel"
fi

# Dependencies
if [ "$COMPONENT" == "fakedns" ]; then
    [ "$TYPE" == "deb" ] && DEPS="Depends: nftables" || DEPS="Depends: nftables, kmod-nft-core, kmod-nft-nat"
    DESC="FakeDNS Core Server"
else
    DEPS="Depends: fakedns"
    DESC="Web Interface for FakeDNS"
fi

# Create Control File
cat <<EOT > "$ROOT/CONTROL/control"
Package: $COMPONENT
Version: $VERSION
Section: net
Priority: optional
Architecture: $PKG_ARCH
Maintainer: a1ex2k
$DEPS
Description: $DESC
EOT

# Final Build Step
if [ "$TYPE" == "deb" ]; then
    mv "$ROOT/CONTROL" "$ROOT/DEBIAN"
    dpkg-deb --build "$ROOT" "$DIST_DIR/${COMPONENT}_${VERSION}_${PKG_ARCH}.deb"
else
    # Manual .ipk creation
    cd "$ROOT"
    tar -cvzf control.tar.gz -C CONTROL . > /dev/null 2>&1
    tar -cvzf data.tar.gz . --exclude=CONTROL --exclude=control.tar.gz --exclude=data.tar.gz > /dev/null 2>&1
    echo "2.0" > debian-binary
    ar r "$DIST_DIR/${COMPONENT}_${VERSION}_${PKG_ARCH}.ipk" debian-binary control.tar.gz data.tar.gz > /dev/null 2>&1
    cd "$CUR_DIR"
fi

rm -rf "$ROOT"
echo "✅ Done: dist/${COMPONENT}_${VERSION}_${PKG_ARCH}.$TYPE"