#!/bin/bash

# Configuration
APP_NAME="DuDe"
VERSION="${VERSION:-$(git describe --tags --abbrev=0 2>/dev/null)}"
VERSION="${VERSION#v}"
DIST_DIR="./dist"
BIN_DIR="./build/bin"

# Exit on error
set -e

if [ -z "$VERSION" ]; then
    echo "Set VERSION or create a Git tag before packaging."
    exit 1
fi

echo "🚀 Starting High-Performance Build for $APP_NAME v$VERSION..."

# Clean up previous distribution
if [ -d "$DIST_DIR" ]; then
    echo "🧹 Cleaning old dist folder..."
    rm -rf "$DIST_DIR"
fi
mkdir -p "$DIST_DIR"

# 1. Windows Build (AMD64)
echo "📦 Building Windows..."
wails build -clean -platform windows/amd64 -upx -upxflags "--best" -ldflags "-s -w" -webview2 embed
mv "$BIN_DIR/$APP_NAME.exe" "$DIST_DIR/${APP_NAME}.exe"

# 2. Linux Build (AMD64)
echo "📦 Building Linux..."
wails build -platform linux/amd64 -ldflags "-s -w"
mv "$BIN_DIR/$APP_NAME" "$DIST_DIR/${APP_NAME}"

# --- ARCHIVING PHASE ---
echo "📦 Packaging binaries for distribution..."

cd "$DIST_DIR"

# Zip Windows
zip -q "${APP_NAME}_v${VERSION}_windows.zip" "${APP_NAME}.exe"

# Tar Linux
tar -czf "${APP_NAME}_v${VERSION}_linux.tar.gz" "${APP_NAME}"

echo "------------------------------------------------"
echo "✅ Build and Packaging Complete!"
echo "📂 Files available in: $DIST_DIR"
ls -lh
