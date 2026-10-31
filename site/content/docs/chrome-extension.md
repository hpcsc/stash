---
title: Chrome extension
weight: 3
---

The extension captures text, links, and images into the same Inbox as the web app, over the
same API and with the same Google account. It signs in with Google via
`chrome.identity.launchWebAuthFlow` and reuses the Web application OAuth client, so the API's
`GOOGLE_CLIENT_ID` and the extension's `googleClientId` are the same value. The same directory
also runs in Firefox — see [Firefox extension](/docs/firefox-extension/).

## 1. Google Cloud (one-time)

Reuse the Web application OAuth client (the value of `GOOGLE_CLIENT_ID`). Add one authorised
redirect URI for the extension:

1. Open the Web application client at https://console.cloud.google.com.
2. Under **Authorised redirect URIs**, add `https://<extension-id>.chromiumapp.org/`, where
   `<extension-id>` is the extension's ID from step 3.

## 2. Configure the extension

Set the client ID in `extension/src/config.js`:

```js
export const googleClientId = '<id>.apps.googleusercontent.com'
```

`apiBaseUrl` defaults to `http://127.0.0.1:3333`, matching the local API.

## 3. Load it

1. Open `chrome://extensions` and turn on **Developer mode**.
2. Click **Load unpacked** and select the `extension/` directory.
3. Note the **ID** on the extension card, and add its redirect URI in step 1.

The ID is stable while the directory stays in place. To pin it across machines, add a `key`
field to `manifest.json` (from the `chrome://extensions` "Pack extension" step).

## 4. Use it

- Click the toolbar icon and **Sign in with Google**.
- Right-click a selection, an image, or a page and choose the matching **Stash** item.

A green check on the toolbar icon means the item was saved; a red mark means it failed. The
items appear in the web Inbox.

## Test

```sh
cd extension && npm test
```

## Notes

- Images are downloaded by the extension and uploaded as bytes, so the item gets a thumbnail.
  An image that needs a login cookie may fail to download.
- `<all_urls>` host permission covers both the API and the image hosts.
