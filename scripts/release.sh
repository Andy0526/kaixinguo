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
set -eu
cd "$(dirname "$0")/.."

identity=${MACOS_SIGNING_IDENTITY:?set MACOS_SIGNING_IDENTITY to "Developer ID Application: Name (TEAMID)"}
profile=${NOTARY_PROFILE:-notary}
version=$(sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' mygo.json)
out=build/darwin-universal
dmg=$out/kai-xin-guo-$version.dmg

if [ "${SKIP_BUILD:-}" = "" ]; then
	go tool mygo build -platform darwin/universal -sign "$identity"
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
echo "Ready: $dmg"
