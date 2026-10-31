# US-013: Capture from the Firefox extension

## Decisions

- **One extension, both browsers.** The same `extension/` directory loads in Chrome and
  Firefox. The manifest declares both `background.scripts` and `background.service_worker`
  (the documented cross-browser MV3 fallback), so Chrome runs a service worker and Firefox
  runs an event page.
- **`browser.*` namespace, not `chrome.*`.** `src/browser.js` resolves the namespace at
  access time — `globalThis.browser` (Firefox) falling back to `globalThis.chrome`
  (Chrome) — so the modules stay identical and the tests can install either mock.
- **Redirect URI via `browser.identity.getRedirectURL()`.** Firefox requires the redirect
  URL its own API returns (`https://<id>.extensions.allizom.org/`); Chrome falls back to
  the constructed `https://<id>.chromiumapp.org/`. A stable Firefox ID comes from
  `browser_specific_settings.gecko.id`.
- **No new capture logic.** Text, link, and image capture reuse the Chrome paths unchanged.

## Codebase patterns to follow

- No build step; plain ES modules; `task extension:test` runs Vitest against the shared
  modules with a `chrome` or `browser` mock.

## Tasks (dependency order)

### Phase 1 — browser adapter
1. Add `src/browser.js` that resolves `browser`/`chrome` lazily.
   - `extension/src/browser.js`.

### Phase 2 — swap the namespace
2. Replace `chrome.*` with the adapter in `storage`, `auth`, `menu`, `dispatch`,
   `background`, and the popup.
   - `extension/src/*.js`, `extension/popup/popup.js`.

### Phase 3 — cross-browser manifest
3. Declare both `scripts` and `service_worker`, add `browser_specific_settings.gecko.id`.
   - `extension/manifest.json`.

### Phase 4 — tests + docs
4. Mock both namespaces; test the adapter and the Firefox redirect URL.
   - `extension/test/*`.
5. Setup doc for loading in Firefox and registering its redirect URI.
   - `docs/firefox-extension.md`.
