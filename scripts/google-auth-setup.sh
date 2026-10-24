#!/usr/bin/env bash
set -euo pipefail

# Prints the values and commands for wiring Stash to a real Google OAuth
# client. The Google Cloud side (project, consent screen, OAuth clients) has no
# public API, so those few steps stay in the console; everything around them is
# here: the Android SHA-1, the origin list, and the commands to run.
#
# Usage:
#   scripts/google-auth-setup.sh                      # checklist for the console
#   scripts/google-auth-setup.sh --client-id <id>     # the wiring, given the id

package_name="${PACKAGE_NAME:-au.pnguyen.stash}"
web_port="${WEB_PORT:-8080}"
api_port="${API_PORT:-3333}"

client_id=""
if [[ "${1:-}" == "--client-id" ]]; then
  client_id="${2:?usage: --client-id <id>.apps.googleusercontent.com}"
elif [[ -n "${GOOGLE_CLIENT_ID:-}" && "${GOOGLE_CLIENT_ID}" != "local-dev-client-id" ]]; then
  client_id="$GOOGLE_CLIENT_ID"
fi

if [[ -n "$client_id" ]] && [[ ! "$client_id" =~ ^[0-9]+-[a-z0-9]+\.apps\.googleusercontent\.com$ ]]; then
  echo "error: '$client_id' does not look like a Google OAuth client ID" >&2
  exit 1
fi

sha1=""
if command -v keytool >/dev/null 2>&1 && [[ -f "$HOME/.android/debug.keystore" ]]; then
  sha1="$(keytool -list -v -keystore "$HOME/.android/debug.keystore" \
    -alias androiddebugkey -storepass android -keypass android 2>/dev/null \
    | awk -F': ' '/^SHA1:/{print $2; exit}')"
fi
if [[ -z "$sha1" ]]; then
  sha1="(run: cd app/android && ./gradlew signingReport)"
fi

echo "Android debug SHA-1: $sha1"
echo

if [[ -z "$client_id" ]]; then
  echo "In the Google Cloud console — no API exists for these steps:"
  echo "  1. Create or pick a project."
  echo "  2. APIs & Services > OAuth consent screen: app name + support email, Audience = External."
  echo "  3. Clients page > Web application: note its client ID; add the origins:"
  echo "       http://localhost:$web_port"
  echo "       http://127.0.0.1:$web_port"
  echo "  4. Clients page > Android:"
  echo "       package name: $package_name"
  echo "       SHA-1:        <the debug SHA-1 above>"
  echo
  echo "Then re-run with the web client ID to print the wiring:"
  echo "  $0 --client-id <id>.apps.googleusercontent.com"
else
  echo "Backend — set in server/.env.local:"
  echo "  GOOGLE_CLIENT_ID=$client_id"
  echo
  echo "Run the app:"
  echo "  web:     cd app && flutter run -d chrome --web-port $web_port --dart-define=GOOGLE_CLIENT_ID=$client_id"
  echo "  android: cd app && flutter run -d <emulator> --dart-define=GOOGLE_CLIENT_ID=$client_id --dart-define=API_BASE_URL=http://10.0.2.2:$api_port"
fi
