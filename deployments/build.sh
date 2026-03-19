#!/bin/bash
# Usage: ./deployments/build.sh <arch> <component>

ARCH=$1
COMPONENT=$2

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

if [ -n "$CXX" ]; then
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
CXX_FLAGS="${CXXFLAGS:--O3 -DNDEBUG -flto -pipe -std=c++20 -pthread}"

echo "Compiling C++ core with: $CXX_BIN"
$CXX_BIN $CXX_FLAGS -I"$CPP_DIR" -o "bin/$ARCH/$COMPONENT" "$CPP_DIR"/*.cpp
if [ $? -eq 0 ]; then
    echo "Build complete: bin/$ARCH/$COMPONENT"
    exit 0
fi

echo "Build failed!"
exit 1
