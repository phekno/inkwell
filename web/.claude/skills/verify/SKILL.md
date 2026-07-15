---
name: verify
description: Drive the inkwell web UI end-to-end without real credentials — fake Cognito session + Playwright-mocked API against the Vite dev server.
---

# Verifying the inkwell web client

The app is gated by Cognito (aws-amplify v6) and talks to the prod API
(`VITE_API_URL` in `web/.env`). No test account exists, but Amplify only
checks token **expiry** client-side, so a forged unexpired JWT gets past the
router guard, and Playwright route interception supplies the backend.

## Recipe

1. `cd web && npm run dev -- --port 5199 --strictPort` (background).
2. Playwright (already in `~/Library/Caches/ms-playwright`): before
   `page.goto`, seed localStorage via `addInitScript` with keys
   `CognitoIdentityServiceProvider.<VITE_COGNITO_CLIENT_ID>.LastAuthUser`
   and `.<user>.idToken` / `.accessToken` / `.refreshToken` / `.clockDrift`.
   Tokens: `base64url(header).base64url(claims).garbage` with `exp` in the
   future; id token needs an `email` claim (shown in the header).
3. `page.route('https://journal.phekno.com/api/**', ...)` — fulfill GET
   `/entries` (list), GET/PATCH/DELETE `/entries/:id`. Include CORS headers
   (`access-control-allow-origin: *` etc.) and answer OPTIONS preflights
   with 204 — the dev origin is cross-origin to the API.
4. Real API quirk to mimic: PATCH responses carry `title: ""`/`folder: ""`;
   the app only trusts `updated_at` from them.

## Gotchas

- Theme: set `localStorage['inkwell-theme']` in the same init script.
  Don't toggle `.dark` mid-test and screenshot immediately — `.btn-term`
  has `transition: color 0.15s`, so text looks invisible mid-transition;
  start the session in the theme you want or wait ~400ms.
- Entry rows/folders are all `<button>`s; select by text
  (`aside button:has-text("...")`). Moving an entry collapses nothing, but
  the destination folder starts collapsed — expand before asserting on it.
