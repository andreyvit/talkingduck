#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
app=".tasks/desktop/TalkingDuck.app"
mkdir -p "$app/Contents/MacOS"
MACOSX_DEPLOYMENT_TARGET=13.0 CGO_CFLAGS="-mmacosx-version-min=13.0" CGO_LDFLAGS="-mmacosx-version-min=13.0" CGO_ENABLED=1 go build -o "$app/Contents/MacOS/talkingduck-desktop" ./desktop
cp desktop/build/Info.plist "$app/Contents/Info.plist"
codesign --force --sign - "$app"
printf '%s\n' "$PWD/$app"
