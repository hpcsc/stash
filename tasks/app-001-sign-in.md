# APP-001: Sign in from the app

## Decisions

- **Google sign-in via `google_sign_in`** (Android + Web). The app exchanges the Google ID
  token for our session token at `POST /auth/signin` (US-001), then stores that session token.
- **Session token in secure storage** (`flutter_secure_storage`): Android Keystore, Web
  localStorage. Kept behind a `TokenStore` interface so it is swappable and testable.
- **Session state via riverpod** (`AsyncNotifier`): a `SessionState` (signed-in/signed-out)
  drives the router's auth gate; the token is restored from storage on startup so the app
  stays signed in across restarts.
- **Google sign-in behind `IdTokenSource`** (mirrors the backend's `IDTokenVerifier`): a real
  `GoogleIdTokenSource` and a fake for tests, so the sign-in flow is testable without a Google
  account.
- **One Google client ID** (`GOOGLE_CLIENT_ID`) is used as the web `clientId` and the Android
  `serverClientId`, so both platforms produce an ID token whose `aud` matches the backend's
  `GOOGLE_CLIENT_ID` (the single audience the backend validates).

## Codebase patterns to follow

- Providers co-located with their type (`apiClientProvider` in `api/client.dart`, etc.).
- The backend envelope `{successful, messages, data}` is parsed by the client; failures throw
  `ApiException` carrying the backend's `messages`.
- Widget tests pump the app with a fake `TokenStore`/`IdTokenSource` and a mock HTTP client.

## Tasks (dependency order)

### Phase 1 — API client

1. Parse the response envelope; add `signIn`/`signOut`; add `googleClientId` config.
   - `lib/src/api/client.dart`, `lib/src/config.dart`, `test/api/client_test.dart`.

### Phase 2 — session + Google sign-in

2. Token storage behind `TokenStore`.
   - `lib/src/auth/token_store.dart`.
3. Google ID token behind `IdTokenSource`.
   - `lib/src/auth/id_token_provider.dart`.
4. Session state + controller + providers.
   - `lib/src/auth/session.dart`, `test/auth/session_test.dart`.

### Phase 3 — auth gate + UI

5. Router redirect on session state.
   - `lib/src/routes.dart`.
6. Sign-in screen + sign-out.
   - `lib/src/screens/sign_in_screen.dart`, `lib/src/screens/sign_out_button.dart`,
     `lib/main.dart`, `test/screens/sign_in_screen_test.dart`, `test/widget_test.dart`.

## Runtime configuration (outside the repo)

The "Continue with Google" flow needs a Google Cloud OAuth client:

1. Create a **Web application** OAuth client ID. This is the value of `GOOGLE_CLIENT_ID`.
   - Add the dev origin (e.g. `http://localhost:PORT`) to its authorised JS origins.
2. Create an **Android** OAuth client ID:
   - Package name `au.pnguyen.stash`, SHA-1 of the app's signing key.
3. Backend: set `GOOGLE_CLIENT_ID` in `server/.env.local`.
4. App: pass `--dart-define=GOOGLE_CLIENT_ID=<web client id>` when running/building.

Without a client ID the app fails the Google sign-in with a clear "not configured" message.
