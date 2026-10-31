---
title: Firefox extension
weight: 4
---

The same `extension/` directory runs in Firefox as in Chrome. The manifest declares both a
`background.service_worker` (used by Chrome) and `background.scripts` (used by Firefox), and
the source resolves `browser.*` (Firefox) falling back to `chrome.*` (Chrome).

Sign-in reuses the same Web application OAuth client and `GOOGLE_CLIENT_ID` as the web and
Android apps, but the redirect URI differs per browser: Firefox uses its own redirect URL
derived from the extension's ID.

## 1. Set the extension ID

`extension/manifest.json` already sets a stable Firefox ID so the redirect URL stays the same
across reinstalls:

```json
"browser_specific_settings": { "gecko": { "id": "stash@example.com" } }
```

Change `stash@example.com` to an ID you own before publishing.

## 2. Add the redirect URI in Google Cloud

Firefox requires the redirect URL returned by its own API, which is
`https://<id>.extensions.allizom.org/`. Get the exact value by loading the extension and
reading `browser.identity.getRedirectURL()` from the extension console (or run it in
`about:debugging`'s console). Add that value under **Authorised redirect URIs** on the Web
application OAuth client, alongside the Chrome one from
[Chrome extension](/docs/chrome-extension/).

## 3. Configure the extension

Set the client ID in `extension/src/config.js` if you have not already:

```js
export const googleClientId = '<id>.apps.googleusercontent.com'
```

`apiBaseUrl` defaults to `http://127.0.0.1:3333`, matching the local API.

## 4. Load it

1. Open `about:debugging#/runtime/this-firefox`.
2. Click **Load Temporary Add-on…** and select `extension/manifest.json`.

Temporary add-ons are removed when Firefox restarts; use `web-ext run` (from the `extension/`
directory) for a persistent dev loop, or package with `web-ext build`.

## 5. Use it

Same as Chrome: click the toolbar icon and **Sign in with Google**, then right-click a
selection, an image, or a page.

## Test

```sh
cd extension && npm test
```

The tests exercise both namespaces: `chrome` and `browser` mocks.
