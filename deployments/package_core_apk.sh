#!/usr/bin/env bash
# Usage: ./deployments/package_core_apk.sh <build-arch> <version> <path-to-openwrt-apk-tool>

set -euo pipefail

ARCH=${1:-}
VERSION=${2:-}
APK_TOOL=${3:-}
PACKAGE_NAME="fakedns-lite"

case "$ARCH" in
    "openwrt_arm64")
        PKG_ARCH="aarch64_generic"
        ;;
    "openwrt_mipsel_24kc")
        PKG_ARCH="mipsel_24kc"
        ;;
    "filogic")
        PKG_ARCH="aarch64_cortex-a53"
        ;;
    *)
        echo "Error: unsupported OpenWrt 25.12 APK target: $ARCH"
        echo "Supported targets: openwrt_arm64, openwrt_mipsel_24kc, filogic"
        exit 1
        ;;
esac

if [ -z "$VERSION" ] || [ -z "$APK_TOOL" ]; then
    echo "Usage: $0 <build-arch> <version> <path-to-openwrt-apk-tool>"
    exit 1
fi

if [ ! -x "$APK_TOOL" ]; then
    echo "Error: OpenWrt APK v3 tool not found or not executable: $APK_TOOL"
    exit 1
fi

CUR_DIR=$(pwd)
DIST_DIR="$CUR_DIR/dist"
BUILD_DIR="$CUR_DIR/apk_build_root"
SCRIPTS_DIR="$CUR_DIR/apk_scripts"

rm -rf "$BUILD_DIR" "$SCRIPTS_DIR"
mkdir -p "$BUILD_DIR/usr/bin" "$BUILD_DIR/etc/init.d" "$BUILD_DIR/etc/config" \
    "$BUILD_DIR/lib/apk/packages" "$SCRIPTS_DIR" "$DIST_DIR"

echo "Creating OpenWrt 25.12 APK v3 for $PACKAGE_NAME ($PKG_ARCH)..."

BINARY="bin/$ARCH/$PACKAGE_NAME"
INIT_FILE="deployments/$PACKAGE_NAME/$PACKAGE_NAME.init"
CONFIG_FILE="deployments/$PACKAGE_NAME/$PACKAGE_NAME.uciconf"

for file in "$BINARY" "$INIT_FILE" "$CONFIG_FILE"; do
    if [ ! -f "$file" ]; then
        echo "Error: required package file not found: $file"
        exit 1
    fi
done

install -m 0755 "$BINARY" "$BUILD_DIR/usr/bin/$PACKAGE_NAME"
sed 's/\r$//' "$INIT_FILE" | sed 's|/etc/init.d/rc.common|/etc/rc.common|g' > "$BUILD_DIR/etc/init.d/$PACKAGE_NAME"
chmod 0755 "$BUILD_DIR/etc/init.d/$PACKAGE_NAME"
install -m 0644 "$CONFIG_FILE" "$BUILD_DIR/etc/config/$PACKAGE_NAME"

printf '%s\n' "/etc/config/$PACKAGE_NAME" > "$BUILD_DIR/lib/apk/packages/$PACKAGE_NAME.conffiles"

cat > "$SCRIPTS_DIR/post-install" <<'EOF'
#!/bin/sh
[ "${IPKG_NO_SCRIPT}" = "1" ] && exit 0
[ -s "${IPKG_INSTROOT}/lib/functions.sh" ] || exit 0
/etc/init.d/fakedns-lite enable
/etc/init.d/fakedns-lite restart
EOF

cat > "$SCRIPTS_DIR/pre-deinstall" <<'EOF'
#!/bin/sh
[ -s "${IPKG_INSTROOT}/lib/functions.sh" ] || exit 0
/etc/init.d/fakedns-lite stop
/etc/init.d/fakedns-lite disable
EOF

chmod 0755 "$SCRIPTS_DIR/post-install" "$SCRIPTS_DIR/pre-deinstall"

"$APK_TOOL" mkpkg \
    --info "name:$PACKAGE_NAME" \
    --info "version:$VERSION-r1" \
    --info "arch:$PKG_ARCH" \
    --info "license:unknown" \
    --info "origin:FakeDns" \
    --info "description:FakeDNS Core Server with nftables DNAT support" \
    --info "maintainer:a1ex2k" \
    --info "depends:nftables kmod-nft-nat libstdcpp" \
    --script "post-install:$SCRIPTS_DIR/post-install" \
    --script "pre-deinstall:$SCRIPTS_DIR/pre-deinstall" \
    --files "$BUILD_DIR" \
    --output "$DIST_DIR/${PACKAGE_NAME}-${VERSION}-r1_${PKG_ARCH}.apk"

rm -rf "$BUILD_DIR" "$SCRIPTS_DIR"
echo "APK v3 complete: $DIST_DIR/${PACKAGE_NAME}-${VERSION}-r1_${PKG_ARCH}.apk"
