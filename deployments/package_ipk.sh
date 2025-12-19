#!/bin/bash
# Usage: ./package_ipk.sh <arch> <version>

ARCH=$1
VERSION=$2
PACKAGE_NAME="fakedns"

CUR_DIR=$(pwd)
DIST_DIR="$CUR_DIR/dist"
ROOT="$CUR_DIR/tmp_ipk_${PACKAGE_NAME}_${ARCH}"

# 1. Setup Structure
rm -rf "$ROOT"
mkdir -p "$ROOT/usr/bin"
mkdir -p "$ROOT/etc/init.d"
mkdir -p "$ROOT/CONTROL"
mkdir -p "$DIST_DIR"

echo "Creating OpenWrt IPK for $PACKAGE_NAME ($ARCH)..."

# 2. Copy Binary
if [ -f "bin/$ARCH/$PACKAGE_NAME" ]; then
    cp "bin/$ARCH/$PACKAGE_NAME" "$ROOT/usr/bin/"
    chmod +x "$ROOT/usr/bin/$PACKAGE_NAME"
else
    echo "Error: bin/$ARCH/$PACKAGE_NAME not found!"
    exit 1
fi

# 3. Copy OpenWrt Init Script
INIT_FILE="deployments/$PACKAGE_NAME/$PACKAGE_NAME.init"
if [ -f "$INIT_FILE" ]; then
    cp "$INIT_FILE" "$ROOT/etc/init.d/$PACKAGE_NAME"
    chmod 755 "$ROOT/etc/init.d/$PACKAGE_NAME"
    echo "Added Init Script: $PACKAGE_NAME"
else
    echo "Error: $INIT_FILE not found!"
    exit 1
fi

# 4. Create Control File
cat <<EOT > "$ROOT/CONTROL/control"
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

# 5. Create postinst (Enable and start service on install)
cat <<EOT > "$ROOT/CONTROL/postinst"
#!/bin/sh
# Check if we are on a real system, not during image building
if [ -z "\$IPKG_INSTROOT" ]; then
    /etc/init.d/$PACKAGE_NAME enable
    /etc/init.d/$PACKAGE_NAME start
    echo "$PACKAGE_NAME enabled and started."
fi
exit 0
EOT
chmod 755 "$ROOT/CONTROL/postinst"

# 6. Create prerm (Stop and disable service before removal)
cat <<EOT > "$ROOT/CONTROL/prerm"
#!/bin/sh
if [ -z "\$IPKG_INSTROOT" ]; then
    /etc/init.d/$PACKAGE_NAME stop
    /etc/init.d/$PACKAGE_NAME disable
    echo "$PACKAGE_NAME stopped and disabled."
fi
exit 0
EOT
chmod 755 "$ROOT/CONTROL/prerm"

# 7. Final Build Step (Assemble IPK)
cd "$ROOT"
tar -cvzf control.tar.gz -C CONTROL . > /dev/null 2>&1
tar -cvzf data.tar.gz . --exclude=CONTROL --exclude=control.tar.gz --exclude=data.tar.gz > /dev/null 2>&1

echo "2.0" > debian-binary
ar r "$DIST_DIR/${PACKAGE_NAME}_${VERSION}_${ARCH}.ipk" debian-binary control.tar.gz data.tar.gz > /dev/null 2>&1

cd "$CUR_DIR"
rm -rf "$ROOT"
echo "IPK complete: dist/${PACKAGE_NAME}_${VERSION}_${ARCH}.ipk"