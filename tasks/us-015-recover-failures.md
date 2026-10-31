# US-015: Recover gracefully from failures

## Decisions

- **Web app only.** The value is "I'm never left staring at nothing", which is the triage UI's
  concern. The server already recovers by restart (SIGTERM graceful shutdown in `cmd/api`,
  panic recovery + process restart in `cmd/catchup`); durable consumer retry-with-backoff is a
  separate concern and out of scope.
- **Every request gets a timeout.** JSON requests time out after 15s, multipart uploads after
  30s (a 25 MB upload over a slow link needs longer). A timeout resolves to a friendly "took
  too long" error, not an endless skeleton or "Saving…" spinner.
- **Transport failures become friendly `ApiError`s in `client.ts`.** A network failure
  (`fetch` rejects with `TypeError`) and a non-JSON response (a reverse-proxy 502/504 HTML
  page) both become a clear message instead of `TypeError: Failed to fetch` / `SyntaxError`.
- **401 while authenticated signs the user out.** A 401 on a request that carried a bearer
  token means the session expired; the client invokes a handler that clears the token so the
  app returns to SignIn instead of leaving the user behind dead error banners. A 401 on
  sign-in (no token yet) still surfaces the backend's message, as today.
- **Retry reloads the current view in place.** The Inbox, Library, and Trash error state gains
  a Retry button that re-runs the load, preserving the current search. The error banner is
  extracted into a shared `ErrorBanner.svelte` to remove the triplicated markup.
- **WebSocket re-sync is out of scope.** After a reconnect the client does not yet re-fetch;
  recovering from *stale* data belongs with offline/sync work (US-011), not "staring at
  nothing".

## Codebase patterns to follow

- `client.ts` already holds a module-level `token` and `setToken`; the unauthorized handler
  follows the same shape (`setUnauthorizedHandler`), registered by `session.ts` at module load
  so there is no import cycle.
- The `session.test.ts` mock of `../api/client` must gain `setUnauthorizedHandler` once
  `session.ts` calls it.

## Tasks (dependency order)

### Phase 1 — API client resilience
1. Add timeouts and friendly errors, and fire an unauthorized handler on 401.
   - `web/src/lib/api/client.ts` (constants, `setUnauthorizedHandler`, refactor `request` /
     `postForm` into a shared exchange that times out, parses the body safely, and maps
     failures).

### Phase 2 — session expiry
2. Register the unauthorized handler to clear the token and return to SignIn.
   - `web/src/lib/auth/session.ts`.

### Phase 3 — retry affordance
3. Extract a shared `ErrorBanner` component with an optional Retry button.
   - `web/src/lib/components/ErrorBanner.svelte`.
4. Use it in the Inbox, Library, and Trash; retry re-runs the load with the current search.
   - `web/src/lib/components/Inbox.svelte`, `Library.svelte`, `Trash.svelte` (add a `query`
     state to Library/Trash so retry keeps the search).

### Phase 4 — tests
5. Client tests: timeout, network failure, non-JSON response, 401 → handler + message.
   - `web/src/lib/api/client.test.ts`.
6. Session test: registering the unauthorized handler and invoking it clears the token.
   - `web/src/lib/auth/session.test.ts`.
