#!/bin/sh
# Builds the universal app signed with a Developer ID, has Apple notarize
# its disk image, and staples the ticket to the image and the app.
#
#   MACOS_SIGNING_IDENTITY="Developer ID Application: Name (TEAMID)" scripts/release.sh
#
# The notary credentials come from the keychain profile NOTARY_PROFILE
# ("notary" unless set), stored once with:
#
#   xcrun notarytool store-credentials notary
#
# Notarizing lives here rather than in mygo.json, so that builds from source
# without a Developer ID keep working.
#
# The build also signs the update that installed apps download, with the key
# in MYGO_UPDATE_KEY_FILE (by default where docs/releasing.md keeps it), or
# in MYGO_UPDATER_PRIVATE_KEY itself. The update's archive holds the app
# before stapling, which is fine: the updater downloads it without the
# quarantine that would have Gatekeeper look for the ticket.
set -eu
cd "$(dirname "$0")/.."

identity=${MACOS_SIGNING_IDENTITY:?set MACOS_SIGNING_IDENTITY to "Developer ID Application: Name (TEAMID)"}
profile=${NOTARY_PROFILE:-notary}
version=$(sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' mygo.json)
out=build/darwin-universal
dmg=$out/kai-xin-guo-$version.dmg
manifest=$out/update-darwin-universal.json

if [ "${SKIP_BUILD:-}" = "" ]; then
	if [ "${MYGO_UPDATER_PRIVATE_KEY:-}" = "" ]; then
		key=${MYGO_UPDATE_KEY_FILE:-"$HOME/Library/Application Support/mygo/update-keys/kaixinguo/mygo-update.key"}
		if [ ! -r "$key" ]; then
			echo "no update signing key at $key: set MYGO_UPDATE_KEY_FILE (see docs/releasing.md)" >&2
			exit 1
		fi
		MYGO_UPDATER_PRIVATE_KEY=$(cat "$key")
		export MYGO_UPDATER_PRIVATE_KEY
	fi
	go tool mygo build -platform darwin/universal -sign "$identity"
fi
if [ ! -f "$manifest" ]; then
	echo "no $manifest: installed apps could not update to this build" >&2
	exit 1
fi

echo "Notarizing $dmg (usually a few minutes)…"
result=$(xcrun notarytool submit "$dmg" --keychain-profile "$profile" --wait --output-format json)
echo "$result"
case $result in
*'"status":"Accepted"'*) ;;
*)
	echo "notarization failed; see: xcrun notarytool log <id> --keychain-profile $profile" >&2
	exit 1
	;;
esac

xcrun stapler staple "$dmg"
xcrun stapler staple "$out/开心果.app"
(cd "$out" && shasum -a 256 "$(basename "$dmg")" >"$(basename "$dmg").sha256")

# Gatekeeper's verdict on the image as a download.
spctl -a -vv -t open --context context:primary-signature "$dmg"

# What the GitHub release holds: the installer, and the update, its deltas
# from earlier versions and the manifest that installed apps read, listed in
# upload.txt for gh release create.
uploads=$out/upload.txt
: >"$uploads"
for f in "$dmg" "$dmg.sha256" "$out"/kai-xin-guo-"$version"-darwin-universal.tar.gz "$out"/*-to-"$version"-darwin-universal.delta "$manifest"; do
	if [ -f "$f" ]; then
		echo "$f" >>"$uploads"
	fi
done
echo "Ready to upload ($uploads):"
sed 's/^/  /' "$uploads"
