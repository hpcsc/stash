# US-001: Sign in with your account

## Decisions

- **Keep the bank-account example**; add sign-in as a new `identity` use case (no touching
  the template's `account` package).
- **Issue our own session token** after validating the Google ID token; the middleware
  validates our token. Matches the `SignedIn`/`SignedOut` events and "stay signed in".
- **Backend-only scope** — the Flutter web/Android app and browser extension live elsewhere;
  this story delivers the endpoints and the auth gate they will call.

## Event model (source of truth)

`docs/event-model-stash.emod` — Identity context, `Account` aggregate:

- `SignIn → SignedIn`, idempotent (re-authentication works without a new event).
- `SignOut → SignedOut`, invariant `SignedOutOnce` (not signed in).

## Codebase patterns to follow

- **Aggregate**: struct embedding `domain.AggregateBase`; methods validate then
  `record(domain.NewEvent(&event.Xxx{...}))`; `Apply(domain.Event)` switch mutates state.
  See `internal/use_cases/account/account.go`.
- **Events**: plain structs with JSON tags + `EventType() string` returning
  `Namespace + ".xxx"`; registered via `RegisterEvents(reg)`.
  See `internal/use_cases/account/event/events.go` and `events.go`.
- **Commands**: plain structs. See `internal/use_cases/account/command/command.go`.
- **Command handler**: sub-package with `Register(bus, repo)` and a `handler` whose
  `Handle(ctx, cmd)` loads the aggregate, calls the method, saves.
  See `internal/use_cases/account/open/handler.go`.
- **Repository**: `NewRepository(stream)` returning `Load`/`Save`.
  See `internal/use_cases/account/repository.go`.
- **API handler**: `route.Routable` with `Routes()`, using `route.Protected` / `route.Public`.
  See `internal/api/use_cases/account/handler.go`.
- **Registration**: `internal/use_cases/register.go` (`RegisterEvents`, `RegisterHandlers`)
  and `internal/api/use_cases/register.go` (`allRoutes`).
- **Auth middleware**: `internal/api/middleware/auth.go` (static token today).
- **Tests**: one umbrella `Test{Type}` with `t.Run` grouped by operation, `testify/require`,
  fresh fixtures per subtest. See `internal/use_cases/account/account_test.go`.

## Tasks (dependency order)

### Phase 1 — domain

1. **Add the identity events** `SignedIn` and `SignedOut`.
   - `internal/use_cases/identity/event/events.go` (structs + `EventType()`).
   - `internal/use_cases/identity/events.go` (`RegisterEvents`).
   - Decide the account identity: the Google subject (`sub`) is the natural key; settle
     whether `UserID` is the raw `sub` string or a derived UUID, and align the emod if it
     deviates from `uuid`.

2. **Add the identity commands** `SignIn` and `SignOut`.
   - `internal/use_cases/identity/command/command.go`.

3. **Add the `Account` aggregate** with `SignIn()` / `SignOut()` and the two invariants,
   plus `TestAccount`.
   - `internal/use_cases/identity/account.go`, `account_test.go`.
   - `SignedInOnce`: sign-in is idempotent — re-authentication issues a fresh session token without a new event. `SignedOutOnce`: sign-out is
     refused when not signed in. Stream prefix must not collide with the bank example's
     `account-<uuid>`; use `identity-<id>`.

4. **Add the repository.**
   - `internal/use_cases/identity/repository.go`.

5. **Add the sign-in and sign-out command handlers** and register them.
   - `internal/use_cases/identity/sign_in/handler.go`, `sign_out/handler.go`.

6. **Wire events + handlers** into `internal/use_cases/register.go`.

### Phase 2 — auth

7. **Issue and verify our own session token.** Add a small `session` package: issue a token
   on sign-in, verify it on each request. Choose opaque token (a `sessions` row) or signed
   JWT; either must support sign-out invalidating the session. Add a migration if stored.

8. **Validate a Google ID token.** Add OIDC validation (Google's issuer + audience, JWKS or
   discovery), returning the `sub`. Provider stays an interface so another OIDC provider can
   replace Google.

9. **Sign-in endpoint.** `POST /auth/signin` (or OIDC callback): validate Google ID token →
   dispatch `SignIn` → issue our session token → return it.

10. **Sign-out endpoint.** `POST /auth/signout`: dispatch `SignOut` → invalidate the session.

11. **Replace the static-token middleware** with session-token validation; refuse requests
    while signed out with a clear message. `internal/api/middleware/auth.go`.

### Phase 3 — wiring + tests

12. **Wire the API routes** into `internal/api/use_cases/register.go` (sign-in is public,
    sign-out and everything else protected).

13. **API and middleware tests.** Sign-in returns a session token; protected route is refused
    without one and with an invalid/expired one; sign-out then refuses the old token.
