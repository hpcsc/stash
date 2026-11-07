# Firefox extension

The same source builds a Firefox package. It captures text, links, and images into the same
Inbox as the web app, over the same API and with the same Google account.

Firefox release builds install only signed add-ons, so the GitHub release carries an AMO
**unlisted** XPI: signed by Mozilla, installable, but not listed in AMO search. See
[chrome-extension.md](chrome-extension.md) for the Chrome package.

## 1. Install from a release

1. Open the [latest release](https://github.com/hpcsc/stash/releases/latest).
2. Download `stash-firefox-<version>.xpi`.
3. Open `about:addons`, click the gear icon, choose **Install Add-on From File…**, and select
   the XPI.

If the release has no XPI, its signing secrets are not configured yet. Set them up under
[Signing](#4-signing), then re-run the tag workflow, or sign locally.

The `stash-firefox-<version>.zip` from `npm run build` is unsigned. Use it for
`about:debugging` temporary installs and for `web-ext run`, not for normal installation.

## 2. Point it at your server

1. Click the toolbar icon, then **Settings**.
2. Enter the server URL and click **Save**.
   - Hosted: `https://stash.saphix.app`, the default.
   - Local: `http://127.0.0.1:3333`.
3. Click **Test connection**, then **Sign in with Google** in the popup.

The client ID comes from the server's public `GET /api/auth/config`, so the server can change
`GOOGLE_CLIENT_ID` without a new extension build. Changing the server clears the stored
session.

Firefox 140 or later is required; Firefox for Android 142 or later.

## 3. Self-hosting: allow the redirect

The add-on ID is fixed at `stash@saphix.app`, so Firefox derives the same redirect URI on
every install:

```
https://aa9983a2262fb49a5f1a0291d8c833ab72951c3c.extensions.allizom.org/
```

Add it under **Authorised redirect URIs** on the OAuth client that the server's
`GOOGLE_CLIENT_ID` belongs to. Firefox hashes the add-on ID rather than using it directly;
the settings page shows the value from `browser.identity.getRedirectURL()`, which is the
source of truth.

## 4. Signing

Create an API key at <https://addons.mozilla.org/developers/addon/api/key/>, then add two
GitHub repository secrets:

| Secret | Value |
| --- | --- |
| `AMO_API_KEY` | the AMO API key (JWT issuer) |
| `AMO_API_SECRET` | the AMO API secret |

When both exist, the release workflow runs `web-ext sign --channel=unlisted` and attaches the
signed XPI. Without them it publishes only the Chrome zip and prints a notice.

To sign locally:

```sh
cd extension
npm ci
npm run build
WEB_EXT_API_KEY='<key>' WEB_EXT_API_SECRET='<secret>' npm run sign:firefox
```

The XPI lands in `extension/dist/signed/`. The first signing registers the add-on ID on AMO.

AMO rejects a version it has already signed, so every release needs a new version. The tag
supplies it: `v1.2.3` builds and signs version `1.2.3`.

Prereleases from `main` are signed too, under their four-part version such as `1.2.3.42`.
Deleting an old prerelease from GitHub does not remove its version from AMO; unlisted
versions stay there.

## Build and lint

```sh
cd extension
npm run build
npm run lint:firefox
```

The build writes `extension/dist/stash-firefox-<version>.zip` and an unpacked copy in
`extension/dist/firefox/`. `web-ext lint` validates the unpacked copy.

## Data disclosure

Firefox asks for consent at install time because captures leave the browser. The manifest
declares three required categories:

| Category | Why |
| --- | --- |
| `authenticationInfo` | the Stash session token |
| `browsingActivity` | the page URL of a saved page or image |
| `websiteContent` | the text, link, or image you choose to save |

Only what you explicitly capture is sent, and only to the server you configure in settings.

## Develop

1. Open `about:debugging#/runtime/this-firefox`.
2. Click **Load Temporary Add-on…** and select `extension/manifest.json`.

Temporary add-ons are removed when Firefox restarts. For a persistent dev loop:

```sh
cd extension
npm ci && npm run build
npx web-ext run --source-dir dist/firefox
```

## Test

```sh
cd extension && npm test
```

The tests exercise both namespaces: `chrome` and `browser` mocks.
