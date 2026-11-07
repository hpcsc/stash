# US-023: Add a selection to an existing note

Appending is a client-side composition over the edit capability from US-021: read the note,
put a blank line and the captured selection after it, save with the version that was read.
No new backend command is needed.

## Decisions

- **Append is load, concatenate, save.** `newBody = note.body.trimEnd() + "\n\n" + selection`.
  The save uses `PATCH /items/{id}` with the note's `version`, so a note edited between the
  read and the write is refused with `409` instead of losing that edit.
- **Capture before opening the picker.** Opening a page drops the selection on the source tab.
  The context-menu click therefore reads the selection first and stages it in
  `browser.storage.session`. The picker reads the staged capture; it never tries to re-read the
  original page.
- **Staged captures expire.** Store `{ markdown, images, tabUrl, capturedAt }` and treat it as
  expired after ten minutes. A stale stage is cleared with a message asking the user to select
  again. Closing the picker tab also clears it, so a later append cannot silently use an old
  selection.
- **A dedicated picker page, opened in a tab.** A service worker cannot show a list. Use
  `browser.tabs.create({ url: browser.runtime.getURL('picker/picker.html') })`; no `tabs`
  permission is required for that. A tab has room for a note list, unlike a popup.
- **Only notes, never the Trash.** List with `state=inbox` and `state=kept`, then keep
  `type === 'note'`. Filtering by type client-side avoids changing the `View.List` signature;
  revisit if a user has enough items that the list needs server-side filtering.
- **Show the note, not just its id.** Each row shows the opening text (markdown syntax and
  images stripped, truncated) and when the note was first saved. Newest first.
- **Read the note again before saving.** Use `GET /items/{id}` rather than the body in the list
  response, so the concatenation starts from the current text. `version` comes from the same
  response.
- **A conflict keeps the selection.** On `409`, show "the note changed, try again" with Retry
  and Cancel. Retry re-reads the note and appends the staged capture again; the staged capture
  is only cleared on success or cancel.
- **Image fallback is shared with US-022.** Reuse the download and token-resolution helpers so
  a failed image becomes its absolute URL and the rest of the selection still lands.
- **The badge reports the append, not the staging.** The context-menu handler must not mark
  the capture saved when it has only staged it; the picker marks success or failure.
- **The menu label distinguishes the two selection actions.** "Save selection to Stash" stays
  as it is; add "Add selection to a note…".

## Codebase patterns to follow

- Menus: `extension/src/menu.js`; click routing and badges: `extension/src/dispatch.js`
  (`handleMenuClick`, `markSaved`, `markFailed`).
- API calls and envelope parsing: `extension/src/api.js` (`postJson`, `saveText`, `saveImage`).
- Settings and token: `extension/src/settings.js`, `extension/src/storage.js`.
- Existing page script and styles: `extension/popup/*`, `extension/options/*`; mirror their
  plain-module, no-framework style.
- Build output: `extension/scripts/build.mjs` (`runtimeDirs`) and `extension/scripts/manifest.mjs`.
- Edit endpoint and version guard from US-021: `PATCH /items/{id}`, `NoteEdited`, `ErrNoteChanged`.
- List endpoint: `GET /items?state=inbox&state=kept`, returning each item's `body`, `version`,
  and `capturedAt`.

## Tasks (dependency order)

### Phase 1 — staging and menu

1. Add the `append-selection` menu item with contexts `['selection']` and the title
   "Add selection to a note…".
   - `extension/src/menu.js`
2. In `dispatch.js`, route `append-selection` to a new `stageSelection(info, tab)`:
   - require a token as `handleMenuClick` does;
   - reuse the US-022 selection reader and its `info.selectionText` fallback;
   - write the staged capture to `browser.storage.session` under a single key;
   - open the picker page;
   - do not call `markSaved()` yet.
   - `extension/src/dispatch.js`
3. Add a `storage.session` wrapper and the expiry check, so both the dispatcher and the picker
   use one definition of "staged" and "expired".
   - `extension/src/staged.js`
4. Add `listItems(states)` and `getItem(id)` to the extension API client.
   - `extension/src/api.js`

### Phase 2 — picker

5. Create the picker page and its styles: a heading, the note list, an empty state, an error
   state, and Cancel. It must explain which note will receive the selection.
   - `extension/picker/picker.html`, `extension/picker/picker.css`
6. Create the picker logic module, testable without a DOM:
   - read and validate the staged capture;
   - load notes and filter to notes;
   - build the appended body;
   - save via `editNote(id, body, version, images)`;
   - map `409` to a retry state and keep the staged capture;
   - clear the stage on success or cancel.
   - `extension/src/append.js`
7. Wire the page to the module. Render each row from stripped opening text and the saved date.
   The row action is the append; a selected row shows progress and then success or the
   conflict message with Retry and Cancel.
   - `extension/picker/picker.js`
8. Add `'picker'` to `runtimeDirs` so the folder is copied into both builds and included in
   the zip.
   - `extension/scripts/build.mjs`

### Phase 3 — API and conflict handling

9. Add `editNote(id, markdown, version, images)` to the extension client: JSON when there are
   no images, multipart (`markdown`, `version`, repeated `images`) when there are. Parse the
   envelope and surface `409` as a distinct error so the picker can offer Retry.
   - `extension/src/api.js`
10. On `409`, re-read the note with `getItem(id)` and offer a single Retry that rebuilds the
    body from the fresh text. Do not retry automatically, because the note may have changed in
    a way that makes the appended text unwanted.

### Phase 4 — tests and docs

11. Extend the browser mock with `storage.session` and `tabs.create`; stub `fetch` for the
    list, the note, and the save.
    - `extension/test/webext-mock.js`
12. Tests for `append.js`: appends after the existing text with a blank line; preserves image
    order; keeps the staged capture on `409`; clears it on success and cancel; refuses an
    expired stage; excludes dropped items.
    - `extension/test/append.test.js`
13. Dispatch tests: `append-selection` stages the selection, opens the picker, and does not
    show the saved badge.
    - `extension/test/dispatch.test.js`
14. Document the new action and the ten-minute staging window.
    - `docs/chrome-extension.md`, `docs/firefox-extension.md`
    - `site/content/docs/chrome-extension.md`, `site/content/docs/firefox-extension.md`
