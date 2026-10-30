#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
mkdir -p .tasks/desktop
build_dir=$(mktemp -d "$PWD/.tasks/desktop/build.XXXXXX")
trap 'rm -rf "$build_dir"' EXIT HUP INT TERM
app="$build_dir/TalkingDuck.app"
mkdir -p "$app/Contents/MacOS"
MACOSX_DEPLOYMENT_TARGET=13.0 CGO_CFLAGS="-mmacosx-version-min=13.0" CGO_LDFLAGS="-mmacosx-version-min=13.0" CGO_ENABLED=1 go build -o "$app/Contents/MacOS/talkingduck-desktop" ./desktop
cp desktop/build/Info.plist "$app/Contents/Info.plist"
codesign --force --sign - "$app"
destination="$HOME/Applications/TalkingDuck.app"
mkdir -p "$HOME/Applications"
ditto "$app" "$destination"
codesign --verify --strict "$destination"
/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister -f "$destination"
printf 'Installed %s\nLaunch: open "%s"\n' "$destination" "$destination"
