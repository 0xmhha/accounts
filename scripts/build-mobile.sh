#!/usr/bin/env bash
#
# build-mobile.sh — build the native mobile artifacts from the Go `mobile`
# package via gomobile. Android is built by default; pass "ios" or "all".
#
# Requires: gomobile + gobind installed, `gomobile init` run once. For Android,
# ANDROID_HOME with an installed NDK. For iOS, a working Xcode.
set -euo pipefail

ACCOUNTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${OUT_DIR:-$ACCOUNTS_DIR/build}"
PKG="github.com/0xmhha/accounts/mobile"
TARGET="${1:-android}"

command -v gomobile >/dev/null || { echo "gomobile not found; go install golang.org/x/mobile/cmd/gomobile@latest && gomobile init" >&2; exit 1; }

mkdir -p "$OUT_DIR"
cd "$ACCOUNTS_DIR"

# gomobile bind needs x/mobile/bind resolvable; add temporarily, revert after.
GOTOOLCHAIN=local GOFLAGS=-mod=mod go get golang.org/x/mobile/bind >/dev/null 2>&1 || true
cleanup() { git -C "$ACCOUNTS_DIR" checkout go.mod go.sum >/dev/null 2>&1 || true; }
trap cleanup EXIT

build_android() {
  : "${ANDROID_HOME:=$HOME/Library/Android/sdk}"
  export ANDROID_HOME
  export ANDROID_NDK_HOME="${ANDROID_NDK_HOME:-$(ls -d "$ANDROID_HOME"/ndk/* 2>/dev/null | sort | tail -1)}"
  echo ">> Android AAR (NDK: $ANDROID_NDK_HOME)"
  GOTOOLCHAIN=local gomobile bind -target=android -androidapi 21 -o "$OUT_DIR/accounts.aar" "$PKG"
  echo ">> wrote $OUT_DIR/accounts.aar"
}

build_ios() {
  echo ">> iOS XCFramework"
  GOTOOLCHAIN=local gomobile bind -target=ios -o "$OUT_DIR/Accounts.xcframework" "$PKG"
  echo ">> wrote $OUT_DIR/Accounts.xcframework"
}

case "$TARGET" in
  android) build_android ;;
  ios) build_ios ;;
  all) build_android; build_ios ;;
  *) echo "usage: $0 [android|ios|all]" >&2; exit 2 ;;
esac
