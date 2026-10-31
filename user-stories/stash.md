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