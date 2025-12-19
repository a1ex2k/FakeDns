#!/bin/bash
# Usage: ./deployments/build.sh <arch> <component>

ARCH=$1
COMPONENT=$2
GO_ARCH=$ARCH
GOMIPS_VAL=""

case "$ARCH" in
    "mipsel_24kc")
        GO_ARCH="mipsle"
        GOMIPS_VAL="softfloat"
        ;;
    "arm64")
        GO_ARCH="arm64"
        ;;
    "amd64")
        GO_ARCH="amd64"
        ;;
esac

mkdir -p "bin/$ARCH"
export GOOS=linux
export GOARCH=$GO_ARCH
export CGO_ENABLED=0

if [ -n "$GOMIPS_VAL" ]; then export GOMIPS=$GOMIPS_VAL; else unset GOMIPS; fi

go build -ldflags="-s -w" -o "bin/$ARCH/$COMPONENT" "./cmd/$COMPONENT"

if [ $? -eq 0 ]; then
    echo "Build complete: bin/$ARCH/$COMPONENT"
else
    echo "Build failed!"
    exit 1
fi