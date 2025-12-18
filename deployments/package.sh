#!/bin/bash
# Usage: ./package.sh <component> <arch> <version> <type>
# Example: ./package.sh fakedns amd64 1.0.0 deb
# Example: ./package.sh fakedns-webui mipsel_24kc 1.0.0 ipk

COMPONENT=$1
ARCH=$2
VERSION=$3
TYPE=$4

ROOT="tmp_${COMPONENT}_${TYPE}_${ARCH}"
DIST_DIR="dist"
rm -rf "$ROOT"
mkdir -p "$ROOT/usr/bin"
mkdir -p "$ROOT/lib/systemd/system"
mkdir -p "$ROOT/CONTROL"
mkdir -p "$DIST_DIR"

echo "Packaging $COMPONENT ($ARCH) into $TYPE..."

if [ -f "bin/$ARCH/$COMPONENT" ]; then
    cp "bin/$ARCH/$COMPONENT" "$ROOT/usr/bin/"
    chmod +x "$ROOT/usr/bin/$COMPONENT"
else
    echo "Error: Binary bin/$ARCH/$COMPONENT not found! Run build.sh first."
    exit 1
fi

SERVICE_FILE="deployments/$COMPONENT/$COMPONENT.service"
if [ -f "$SERVICE_FILE" ]; then
    cp "$SERVICE_FILE" "$ROOT/lib/systemd/system/"
    echo "Included service file from $SERVICE_FILE"
else
    echo "Error: Service file not found at $SERVICE_FILE."
    exit 1
fi

PKG_ARCH=$ARCH
if [ "$TYPE" == "deb" ] && [ "$ARCH" == "mipsel_24kc" ]; then
    PKG_ARCH="mipsel"
fi

if [ "$COMPONENT" == "fakedns" ]; then
    if [ "$TYPE" == "deb" ]; then
        DEPS="Depends: nftables"
    else
        DEPS="Depends: nftables, kmod-nft-core, kmod-nft-nat"
    fi
    DESC="FakeDNS Core Server (Netlink nftables implementation)"
else
    DEPS="Depends: fakedns"
    DESC="Web Interface for FakeDNS"
fi

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

if [ "$TYPE" == "deb" ]; then
    mv "$ROOT/CONTROL" "$ROOT/DEBIAN"
    dpkg-deb --build "$ROOT" "$DIST_DIR/${COMPONENT}_${VERSION}_${PKG_ARCH}.deb"
    
else
    cd "$ROOT"
    tar -cvzf control.tar.gz -C CONTROL . > /dev/null 2>&1
    tar -cvzf data.tar.gz . --exclude=CONTROL --exclude=control.tar.gz --exclude=data.tar.gz > /dev/null 2>&1
    echo "2.0" > debian-binary
    ar r "../../$DIST_DIR/${COMPONENT}_${VERSION}_${PKG_ARCH}.ipk" debian-binary control.tar.gz data.tar.gz > /dev/null 2>&1 
    cd ..
fi

rm -rf "$ROOT"
echo "Done: $DIST_DIR/${COMPONENT}_${VERSION}_${PKG_ARCH}.$TYPE"
