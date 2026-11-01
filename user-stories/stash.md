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

### US-016: Deploy self-hosted with docker compose

**Description:** As a user, I want to run Stash on my own machine with docker compose so that
I keep control of my data.

**Acceptance Criteria:**
- [x] One `docker compose` command builds and starts the API, the consumer, PostgreSQL, the
  migration runner, and the web app.
- [x] The database and uploaded files live in named volumes and survive a container restart.
- [x] The web app is served same-origin and talks to the API through the same hostname.
- [x] The steps to run it and reach it over Tailscale are documented.

**Depends on:** US-015

### US-017: Deploy to a cloud provider

**Description:** As a user, I want to deploy Stash to a cloud provider so that I use it
without running a server at home.

**Acceptance Criteria:**
- [x] File bytes can be stored in an S3-compatible bucket, selected via config, alongside
  local disk.
- [x] The API and consumer images publish to a container registry.
- [x] A compose file runs the stack against managed Postgres and S3 storage, pulling published
  images.
- [x] The steps to deploy to a cloud provider are documented.

**Depends on:** US-016

### US-018: Give each user a private stash

**Description:** As a user, I want everything I save to belong to me alone so that anyone else
who signs in cannot see, change, or delete it.

**Acceptance Criteria:**
- [x] Signing in with a second Google account shows a separate, empty stash.
- [x] My Inbox, Library, Trash, search results, and tag list show only my own data.
- [x] Opening, keeping, dropping, restoring, or deleting someone else's item fails as if it
  did not exist.
- [x] Downloading someone else's file fails, even with its full URL.
- [x] Two users can each have a tag with the same name, and renaming mine never touches
  theirs.
- [x] The duplicate warning compares my captures only with my own earlier captures.
- [x] Realtime updates reach only the user whose item changed.
- [x] The stored history attributes each change to the user who made it, without changing the
  event payloads.

**Depends on:** US-003, US-004, US-005, US-006, US-007, US-008, US-009