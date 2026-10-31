---
title: Google sign-in setup
weight: 2
---

How to point Stash at a real Google OAuth client and test sign-in from the web app or the Android app.

The flow: the app obtains a Google ID token, posts it to `POST /auth/signin`, and the API
validates the token's audience (`aud`) against `GOOGLE_CLIENT_ID`, records `SignedIn`, and
returns a session token.

One Google client ID — a **Web application** OAuth client — is used everywhere: as the web
`clientId`, the Android `serverClientId`, and the API's `GOOGLE_CLIENT_ID`. Every platform then
produces an ID token with the same `aud`, and the API accepts them all.

The Google Cloud steps below have no public API, so they are done in the console; a helper
script prints the checklist and the run commands with the values filled in:

```sh
./scripts/google-auth-setup.sh                    # console checklist
./scripts/google-auth-setup.sh --client-id <id>   # wiring, given the client id
```

## 1. Google Cloud (one-time)

1. Create (or reuse) a project at https://console.cloud.google.com.
2. In the project sidebar, **APIs & Services → OAuth consent screen**: fill in the app name and
   support email, then set **Audience** to **External**.
3. On the **Clients** page, create two OAuth 2.0 client IDs:

   **Web application** — this is `GOOGLE_CLIENT_ID`.

   - Authorised JavaScript origins — pin the dev-server port so it is stable:
     - `http://localhost:8080`
     - `http://127.0.0.1:8080`

   **Android**

   - Package name `au.pnguyen.stash`.
   - SHA-1 of the signing key. For the debug key:

     ```sh
     cd app/android && ./gradlew signingReport
     ```

     Copy the **debug** `SHA1`.

## 2. Backend

Set the API's `GOOGLE_CLIENT_ID` to the web client ID in `server/.env.local`:

```sh
GOOGLE_CLIENT_ID=<id>.apps.googleusercontent.com
```

Start the stack (Postgres, migrations, `api`, `catchup`):

```sh
task run
```

The API listens on `:3333`. `SESSION_SECRET` can stay at its local-dev value.

## 3. Web (Svelte)

Set the Google client ID in `web/.env.local`:

```sh
VITE_GOOGLE_CLIENT_ID=<id>.apps.googleusercontent.com
```

Then run the dev server and open `http://127.0.0.1:8080`:

```sh
cd web
npm run dev -- --port 8080
```

The API base URL comes from `VITE_API_BASE_URL` (default `http://127.0.0.1:3333`).

## 4. Android

Two things to know first:

- The emulator cannot reach the host's `127.0.0.1`; use `10.0.2.2`.
- Google Sign-In needs Play services, so use a Google-APIs emulator image, not plain AOSP.

```sh
flutter run -d <emulator-id> \
  --dart-define=GOOGLE_CLIENT_ID=<id>.apps.googleusercontent.com \
  --dart-define=API_BASE_URL=http://10.0.2.2:3333
```

On a physical device, replace `10.0.2.2` with the machine's LAN (or Tailscale) IP.

## If it fails

| Symptom | Cause |
| --- | --- |
| "Google sign-in is not configured" | The Google client ID was not passed to the app. |
| `401 sign-in failed` from the API | The ID token's `aud` does not match `GOOGLE_CLIENT_ID`. Use the same web client ID for the web `clientId`, the Android `serverClientId`, and the API. |
| Web popup blocked / "origin not allowed" | The dev origin is not in the web client's authorised JavaScript origins. |
| Android sign-in error | SHA-1 mismatch with the Android client, or the emulator has no Play services. |
| "Failed to fetch" from the web app | CORS — the web origin is not in `ALLOWED_ORIGINS` (default `http://localhost:8080`, `http://127.0.0.1:8080`). |

The API starts even with the `local-dev-client-id` placeholder; it only refuses real tokens, so
step 2 is the piece that makes sign-in work.
