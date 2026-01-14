#!/bin/bash

# Configuration
APP_NAME="DuDe"
VERSION="1.0.0" # You can sync this with your wails.json
DIST_DIR="./dist"
BIN_DIR="./build/bin"

# Exit on error
set -e

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

# 3. macOS Build (Universal)
# echo "📦 Building macOS Universal..."
# wails build -platform darwin/universal -ldflags "-s -w"
# # Wails creates a .app bundle for macOS
# mv "$BIN_DIR/$APP_NAME.app" "$DIST_DIR/$APP_NAME.app"

# --- ARCHIVING PHASE ---
echo "📦 Packaging binaries for distribution..."

cd "$DIST_DIR"

# Zip Windows
zip -q "${APP_NAME}_v${VERSION}_windows.zip" "${APP_NAME}.exe"

# Tar Linux
tar -czf "${APP_NAME}_v${VERSION}_linux.tar.gz" "${APP_NAME}"

# Zip macOS App Bundle
# zip -r -q "${APP_NAME}_v${VERSION}_macOS_universal.zip" "$APP_NAME.app"

# Cleanup raw binaries if you only want the archives
# rm -rf "$APP_NAME.app"

echo "------------------------------------------------"
echo "✅ Build and Packaging Complete!"
echo "📂 Files available in: $DIST_DIR"
ls -lh