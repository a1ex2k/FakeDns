#!/bin/bash
# Usage: ./deployments/build.sh <arch> <component>

ARCH=$1
COMPONENT=$2
GO_ARCH=$ARCH
GOMIPS_VAL=""

# Mapping OpenWrt arch names to Go Standard arch names
if [ "$ARCH" == "mipsel_24kc" ]; then
    GO_ARCH="mipsle"
    GOMIPS_VAL="softfloat"
fi

mkdir -p "bin/$ARCH"
echo "Building $COMPONENT for $ARCH (GOARCH=$GO_ARCH)..."

export GOOS=linux
export GOARCH=$GO_ARCH
export CGO_ENABLED=0

if [ -n "$GOMIPS_VAL" ]; then
    export GOMIPS=$GOMIPS_VAL
else
    unset GOMIPS
fi

go build -ldflags="-s -w" -o "bin/$ARCH/$COMPONENT" "./cmd/$COMPONENT"

if [ $? -eq 0 ]; then
    echo "Build complete: bin/$ARCH/$COMPONENT"
else
    echo "Build failed!"
    exit 1
fi