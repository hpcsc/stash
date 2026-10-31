### APP-002: See the Inbox and capture text

**Description:** As a user, I want to see my Inbox and save a piece of text so I can triage
it later.

**Acceptance Criteria:**
- [x] The Inbox lists my unsorted items, newest first.
- [x] I can save a piece of text and see it appear in the Inbox.
- [x] Tapping an Inbox item shows its full text and, when known, the app it came from.
- [x] An empty Inbox shows a friendly message.

**Depends on:** US-002

### US-009: Warn on duplicate captures

**Description:** As a user, I want to be warned when I capture text or a link I've already
saved so that I don't unknowingly keep copies.

**Acceptance Criteria:**
- [x] Capturing text I've already saved still creates the item but shows a notice that it
  looks like a duplicate.
- [x] Capturing a link I've already saved still creates the item but shows the same notice.
- [x] Capturing something new shows no such notice.

**Depends on:** US-003

### US-012: Capture from the Chrome extension

**Description:** As a user, I want to save selected text, the current page, or an image from
Chrome so that I can capture while browsing.

**Acceptance Criteria:**
- [x] Selecting text in Chrome and saving it adds the text to my Inbox.
- [x] Saving the current page adds the page as a link to my Inbox.
- [x] Saving a right-clicked image adds the image to my Inbox.

**Depends on:** US-003, US-004

### US-013: Capture from the Firefox extension

**Description:** As a user, I want to save selected text, the current page, or an image from
Firefox so that I capture no matter which browser I use.

**Acceptance Criteria:**
- [x] Selecting text in Firefox and saving it adds the text to my Inbox.
- [x] Saving the current page adds the page as a link to my Inbox.
- [x] Saving a right-clicked image adds the image to my Inbox.

**Depends on:** US-012

### US-015: Recover gracefully from failures

**Description:** As a user, I want the app to recover gracefully when a request fails so that
I'm never left staring at nothing.

**Acceptance Criteria:**
- [x] When the server can't be reached, the Inbox, Library, and Trash show a clear error with
  a Retry button instead of a blank screen.
- [x] When a request times out, the app shows a clear "took too long" message instead of an
  endless spinner.
- [x] Retrying after a failure reloads the current view without a full page refresh, keeping
  my search.
- [x] When my session expires, I'm returned to sign-in instead of being stuck behind error
  messages.

**Depends on:** US-005, US-006