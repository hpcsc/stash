# US-012: Capture from the Chrome extension

## Decisions

- **Manifest V3, plain ES modules, no build step.** Chrome loads `extension/` unpacked;
  the service worker and popup import the shared `src/` modules directly. Tests run with
  Vitest against the same modules, mocking `chrome` and `fetch`.
- **Google sign-in via `chrome.identity.launchWebAuthFlow`.** The extension asks for an ID
  token (`response_type=id_token`, `scope=openid email profile`, a `nonce`) and posts it to
  the existing `POST /auth/signin`, reusing the Web OAuth client ID. The redirect URI is
  `https://<extension-id>.chromiumapp.org/`, added to the Web client in Google Cloud.
- **Session token in `chrome.storage.local`**, read by the capture requests as the bearer.
- **Three context-menu items** on install: selection (`text`), image (`image`), page
  (`link`). The image item fetches the image bytes with `<all_urls>` host permission and
  uploads them as multipart, so it becomes a real `image` item with a thumbnail.
- **Feedback through the action badge** — `✓` on success, `!` on failure — cleared when the
  popup opens. No notification icons.

## Codebase patterns to follow

- The backend envelope `{successful, messages, data}` is parsed by the client; failures
  throw `ApiError` carrying `messages`, mirroring the web client.
- `apiBaseUrl` and `googleClientId` live in `src/config.js`, editable for local dev.

## Tasks (dependency order)

### Phase 1 — extension shell
1. Manifest, config, and token storage.
   - `extension/manifest.json`, `extension/src/config.js`, `extension/src/storage.js`.

### Phase 2 — API + auth
2. API client: `saveText`, `saveLink`, `saveImage`.
   - `extension/src/api.js`.
3. Google sign-in via `launchWebAuthFlow`, and sign-out.
   - `extension/src/auth.js`.

### Phase 3 — capture + UI
4. Context menus and the click dispatch.
   - `extension/src/menu.js`, `extension/src/background.js`.
5. Popup for sign-in/status.
   - `extension/popup/popup.html`, `extension/popup/popup.js`, `extension/popup/popup.css`.

### Phase 4 — tests + docs
6. Vitest for `api`, `auth`, and `menu`, with a `chrome` mock.
   - `extension/test/*`, `extension/package.json`, `extension/vitest.config.js`.
7. Setup doc (Google Cloud redirect URI, load unpacked, client ID).
   - `docs/chrome-extension.md`.
