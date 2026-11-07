---
title: Chrome extension
weight: 3
---

The extension captures text, links, and images into the same Inbox as the web app, over the
same API and with the same Google account. It is distributed as an unpacked package from
GitHub Releases rather than the Chrome Web Store.

The same source also builds a Firefox package — see
[Firefox extension](/docs/firefox-extension/).

## 1. Install from a release

1. Open the [latest release](https://github.com/hpcsc/stash/releases/latest).
2. Download `stash-chrome-<version>.zip` and unzip it.
3. Open `chrome://extensions` and turn on **Developer mode**.
4. Click **Load unpacked** and select the unzipped folder — the one containing `manifest.json`.

Chrome does not install `.crx` files from outside the Web Store on Windows or macOS, so
**Load unpacked** is the supported channel. The extension stays installed across restarts.

The package carries a fixed manifest `key`, so its ID is always
`kdfpkmjkeamfogldcphiajinboajjced` no matter where the folder was unzipped. That keeps the
Google sign-in redirect URI stable across machines and updates.

To update, unzip the new release over the old folder and click **Reload** on the extension
card. Developer-mode installs do not update themselves.

The Chrome Web Store rejects a manifest that contains `key` and would assign a different ID.
This package is for GitHub distribution; a store listing would need a separate build without
the key.

### Brave

Brave uses Chromium's extension model, so it installs the same Chrome package. Open
`brave://extensions` instead of `chrome://extensions`; the other steps are the same. Brave
derives the same extension ID and redirect URI from the fixed `key`, so a self-hoster's OAuth
client needs no change.

Brave on Android and iOS does not support extensions.

## 2. Point it at your server

1. Click the toolbar icon, then **Settings** (or right-click the icon and choose **Options**).
2. Enter the server URL and click **Save**.
   - Hosted: `https://stash.saphix.app`, the default.
   - Local: `http://127.0.0.1:3333`.
   - A server under a path also works, for example `https://example.com/stash`.
3. Click **Test connection** to read the server's public config.
4. Click **Sign in with Google** in the popup.

The extension reads the Google client ID from the server's public `GET /api/auth/config`. A
server can change `GOOGLE_CLIENT_ID` without a new extension build.

Changing the server clears the stored session, because a token from one server is not valid
on another.

## 3. Self-hosting: allow the redirect

Google sign-in needs the extension's redirect URI on the OAuth client that the server's
`GOOGLE_CLIENT_ID` belongs to. Chrome derives it from the extension ID, so it is fixed:

```
https://kdfpkmjkeamfogldcphiajinboajjced.chromiumapp.org/
```

Add it under **Authorised redirect URIs** for that client in the Google Cloud console. The
extension's settings page shows the same value, so it can be copied from there.

## 4. Use it

- Right-click a selection, an image, or a page and choose the matching **Stash** item.
- A green check on the toolbar icon means the item was saved; a red mark means it failed.
- The items appear in the web Inbox.

## Build and release

```sh
cd extension
npm ci
npm test
npm run build
```

The build writes `extension/dist/stash-chrome-<version>.zip` and an unpacked copy in
`extension/dist/chrome/`. The version comes from `package.json`, or from `--version`:

```sh
npm run build -- --version 1.2.3
```

Pushing a `v*` tag runs `.github/workflows/extension.yml`, which tests, builds, lints, and
attaches the zip to a GitHub Release. Release assets are the distributable channel: unlike
workflow artifacts they do not expire and do not require a GitHub login.

A push to `main` that changes `extension/**` publishes a prerelease instead. It tags the
commit `v<next>-<run>.g<sha>`, for example `v1.2.4-42.g4829f92`, and marks the release as a
prerelease, so the latest release still points at the last stable tag. The browser version
takes the latest stable version plus the run number as a fourth part, for example `1.2.3.42`:
it installs over `1.2.3`, and stable `1.2.4` installs over it. Only the 5 newest prereleases
are kept.

Stable tags must be numeric (`v1.2.3`), because Chrome and Firefox reject a version with a
suffix such as `-rc.1`. Use the prerelease channel for builds before the stable tag: open the
Releases page and pick the newest one marked **Pre-release**.

`extension/chrome-key.pub` holds the public key behind the fixed ID. The build injects it as
`manifest.key`. Do not regenerate it — a new key means a new extension ID, and every
self-hoster would have to add a new redirect URI to their Google OAuth client.

## Test

```sh
cd extension && npm test
```

## Notes

- Images are downloaded by the extension and uploaded as bytes, so the item gets a thumbnail.
  An image that needs a login cookie may fail to download.
- `<all_urls>` host permission covers both the API and the image hosts.
- The extension talks only to the server you configure. Nothing is sent to the extension's
  author.
