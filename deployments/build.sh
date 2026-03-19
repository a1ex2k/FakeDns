#!/usr/bin/env bash
# Usage: ./deployments/build.sh <arch> <component>

set -u

ARCH=${1:-}
COMPONENT=${2:-}

if [ -z "$ARCH" ] || [ -z "$COMPONENT" ]; then
    echo "Usage: $0 <arch> <component>"
    exit 1
fi

if [ "$COMPONENT" != "fakedns-lite" ]; then
    echo "Build failed: only component 'fakedns-lite' is supported in this C/C++ branch."
    exit 1
fi

mkdir -p "bin/$ARCH"

CPP_DIR="./cmd/fakedns_cpp"
if [ ! -d "$CPP_DIR" ]; then
    echo "Build failed: $CPP_DIR not found"
    exit 1
fi

if ! ls "$CPP_DIR"/*.cpp >/dev/null 2>&1; then
    echo "Build failed: no C++ sources found in $CPP_DIR"
    exit 1
fi

if [ -n "${CXX:-}" ]; then
    CXX_BIN="$CXX"
else
    case "$ARCH" in
        "amd64")
            CXX_BIN="g++"
            ;;
        "arm64")
            CXX_BIN="aarch64-linux-gnu-g++"
            ;;
        "mipsel_24kc")
            CXX_BIN="mipsel-openwrt-linux-musl-g++"
            ;;
        *)
            CXX_BIN="g++"
            ;;
    esac
fi
CXX_FLAGS="${CXXFLAGS:--O3 -DNDEBUG -flto -pipe -std=c++17 -pthread}"
EXTRA_LDFLAGS="${FAKEDNS_LDFLAGS:-}"

if [ "$ARCH" = "mipsel_24kc" ] && [ -z "${CXXFLAGS:-}" ]; then
    # Keep mipsel defaults conservative to avoid cross-toolchain incompatibilities.
    CXX_FLAGS="-O2 -DNDEBUG -pipe -std=c++17 -pthread"
fi

echo "Compiling C++ core with: $CXX_BIN"
if ! command -v "$CXX_BIN" >/dev/null 2>&1; then
    echo "Build failed: compiler '$CXX_BIN' not found in PATH"
    exit 1
fi

if [ "$ARCH" = "mipsel_24kc" ]; then
    TARGET_TRIPLE="$("$CXX_BIN" -dumpmachine 2>/dev/null || true)"
    if [[ "$TARGET_TRIPLE" != *"musl"* ]]; then
        echo "Build failed: mipsel_24kc requires OpenWrt musl toolchain."
        echo "Detected target triple: ${TARGET_TRIPLE:-unknown}"
        echo "Use mipsel-openwrt-linux-musl-g++ from the matching OpenWrt SDK."
        exit 1
    fi
fi

echo "Compiler version:"
"$CXX_BIN" --version | head -n 1 || true
echo "Compile flags: $CXX_FLAGS"
if [ -n "$EXTRA_LDFLAGS" ]; then
    echo "Extra linker flags: $EXTRA_LDFLAGS"
fi

if "$CXX_BIN" $CXX_FLAGS -I"$CPP_DIR" -o "bin/$ARCH/$COMPONENT" "$CPP_DIR"/*.cpp $EXTRA_LDFLAGS; then
    echo "Build complete: bin/$ARCH/$COMPONENT"
    exit 0
fi

echo "Build failed!"
exit 1
