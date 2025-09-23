#!/bin/bash

set -e

APP_NAME="is16kbReady"
VERSION="1.0.0"
BUILD_DIR="dist"

echo "Building $APP_NAME v$VERSION..."

rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR

platforms=(
    "windows/amd64"
    "windows/386"
    "linux/amd64"
    "linux/386"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

for platform in "${platforms[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"
    
    output_name="$APP_NAME-$GOOS-$GOARCH"
    if [ $GOOS = "windows" ]; then
        output_name+=".exe"
    fi
    
    echo "Building for $GOOS/$GOARCH..."
    
    env GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "-s -w" -o "$BUILD_DIR/$output_name" main.go
    
    if [ $? -ne 0 ]; then
        echo "Failed to build for $GOOS/$GOARCH"
        exit 1
    fi
done

echo ""
echo "Build completed successfully!"
echo "Binaries created in $BUILD_DIR:"
ls -la $BUILD_DIR/
