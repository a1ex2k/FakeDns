#!/bin/bash
# Usage: ./deployments/build.sh <arch>
ARCH=$1
GO_ARCH=$ARCH
GOMIPS_FLAG=""

if [ "$ARCH" == "mipsel_24kc" ]; then
    GO_ARCH="mipsle"
    GOMIPS_FLAG="softfloat"
fi

echo "Building binaries for $ARCH..."

export GOOS=linux
export GOARCH=$GO_ARCH
export GOMIPS=$GOMIPS_FLAG
export CGO_ENABLED=0

go build -ldflags="-s -w" -o "bin/$ARCH/fakedns" ./cmd/fakedns
go build -ldflags="-s -w" -o "bin/$ARCH/fakedns-webui" ./cmd/fakedns-webui
echo "Build complete: bin/$ARCH/"