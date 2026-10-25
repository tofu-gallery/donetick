# Tofu gallery copy

This repository is Tofu's copy of **donetick/donetick** (licence: AGPL-3.0), listed in Tofu's app gallery as **Shared chore chart**: Donetick, a chore and task tracker for a household or a small group, with repeating schedules, assignees who take turns, points and rewards, labels, projects and a calendar.

- `develop` mirrors the upstream repository and is never committed to by Tofu.
- `tofu` carries Tofu's own changes; tags named `pin/<date>` mark the versions the gallery points to. The current pin is `pin/2026-10-25`.
- A visitor who chooses this app gets a private snapshot of the pinned version in their own Tofu account. Their copy does not follow upstream updates.
- Upstream, server: https://github.com/donetick/donetick at v0.1.83, `549e44c836de6b54f026624ccce9d6e8df726a30` (2026-10-18), the latest stable release. `tofu` starts from that tag, which is on `develop`.
- Upstream, web app: https://github.com/donetick/frontend at `0a0850a734752fcdf50513c033ec5159a9a79821` (2026-10-18, app version 1.2.58), in `web/`. Upstream's release workflow builds the frontend's default branch into the server at release time; this is that branch's head when v0.1.83 was released, going by commit times (the next frontend commit is from 2026-10-19). It was not compared with the published v0.1.83 image.
- Why not `develop`'s head: at the time of this pin, `develop` (`584301c`, tagged v0.1.84-beta.3) does not compile (`internal/notifier/push/push.go:99: not enough return values`), and upstream published no release for that tag.

## How Tofu runs it

- Framework preset `container`, root folder: the repository root. Tofu builds the root `Dockerfile` and runs the image as a service, so there is no separate install command, build command or output directory.
- The `Dockerfile` has three stages:
  1. `node:22-bookworm-slim` builds the web app in `web/`: `npm ci --ignore-scripts --no-audit --no-fund`, `npx --no-install patch-package`, then `npx --no-install vite build --mode selfhosted` with `VITE_APP_API_URL=""` (the API is the same server) and `VITE_IS_SELF_HOSTED=true`.
  2. `golang:1.24-alpine` downloads and verifies the Go modules, replaces the placeholder `frontend/dist` with the web app built in stage 1, and builds the server with `CGO_ENABLED=0 go build`. The server embeds `frontend/dist` (`go:embed`) and serves it.
  3. `alpine` runs `/donetick` through `/tofu-entrypoint.sh`, with `/config/selfhosted.yaml` (see below) and `DT_ENV=selfhosted`.
- Database: PostgreSQL, the app's Tofu database, read from `DATABASE_URL`. Without `DATABASE_URL`, Donetick uses SQLite (`donetick.db` in the working directory), which is lost on every restart.
- Migrations: the server makes and updates its own tables at every start, before it starts listening: GORM's AutoMigrate, then Donetick's data migrations (recorded in the `migrations` table), then the SQL files in `internal/database/migrations` (recorded in `gorp_migrations`). There is no `supabase/migrations` folder, so Tofu applies no SQL of its own. These tables are made by the app, so the database's public Data API roles have no access to them, and they need no row-level security: the server's own connection is what reads them.
- Settings Tofu sets: `PORT` and `DATABASE_URL`. Settings the image sets: `DT_ENV=selfhosted` and `DT_SOURCE_URL` (this pin's source). Settings a gallery copy saves before its first deploy: `DT_SERVER_PUBLIC_HOST` and `DT_EMAIL_APP_HOST`, both the copy's own address, and `DT_SOURCE_URL`, this pin's exact source. Nothing has to be typed by hand.
- Settings an owner may add later: `DT_IS_USER_CREATION_DISABLED=true` to close sign-up (see "Accounts"), `DT_JWT_SECRET` as a hidden value (see "The sign-in secret"), the `DT_EMAIL_*` settings for an SMTP server, the `DT_STORAGE_*` settings for a file bucket, and `DT_TELEGRAM_TOKEN` or `DT_PUSHOVER_TOKEN` for notifications (Discord notifications use a webhook set inside the app). Every key of `/config/selfhosted.yaml` can be set as `DT_` plus the key's path in capitals with `_` for `.`, for example `DT_SERVER_RATE_LIMIT`, and such a setting wins over the file.

## What Tofu changes

Every file Tofu changed carries a dated `Tofu (2026-10-25)` note at the change; files Tofu added say so at their top.

- `web/` (new): the web app's source, copied unchanged from donetick/frontend at the commit above, without the parts the web build does not use: `android/`, `ios/`, `fastlane/`, `Gemfile`, `Gemfile.lock`, `.bundle/` (the mobile apps and their release tooling), `.github/`, `.husky/`, `e2e/`, `tests/`, `resources/`, `icons/`, `assets/` (app-store and README images), `scripts/` (release, secret-sync and translation-audit scripts), `capacitor.config.ts`, `ionic.config.json`, `wrangler.toml`, `crowdin.yml`, `bump-version.js`, `AUTOMATED_RELEASE.md` and `.npmrc`. Those stay available at that commit. Upstream builds the frontend from its own repository in its release workflow; a Tofu copy is built from this repository alone, so the source has to be here.
- `Dockerfile`:
  - The web stage above, and the Go stage puts its output in `frontend/dist` before building, as upstream's release workflow does.
  - `npm ci` runs without install scripts: they download the PostHog command-line tool (`@posthog/cli`) and the libvips image library (`sharp`, for icon generators), neither of which the web build uses, and print funding notices (`core-js`, `tesseract.js`). The app's own postinstall step, `patch-package`, runs on its own; its one patch is to an Android plugin file.
  - `/config/selfhosted.yaml` is a copy of upstream's `config/selfhosted.yaml.example`, shipped in the image. Upstream writes that file at the first start instead, with a random sign-in secret, which needs a writable `/config` and, where nothing is kept between starts, makes a new secret, signing everyone out, at every start. The secret in the shipped file is upstream's placeholder, which the server refuses; `tofu-entrypoint.sh` always sets `DT_JWT_SECRET`, which wins over the file.
  - It starts through `tofu-entrypoint.sh`, sets `DT_SOURCE_URL`, and the health check uses the port the server listens on (`PORT`, then `DT_SERVER_PORT`, then 2021).
- `tofu-entrypoint.sh` (new): sets `DT_SERVER_PORT` to Tofu's `PORT`; sets `DT_DATABASE_TYPE=postgres` when `DATABASE_URL` is set, unless `DT_DATABASE_TYPE` is set; derives `DT_JWT_SECRET` when it is not set (see "The sign-in secret"); then runs `/donetick`.
- `internal/database/database.go`: upstream connects to Postgres from host, port, user, password and database name with `sslmode=disable`. When `DATABASE_URL` is set and `DT_DATABASE_HOST` is not, the server now connects with `DATABASE_URL`, keeping its `sslmode` (Tofu's database takes TLS connections), and adds `default_query_exec_mode=cache_describe` unless the URL names a mode. Tofu's `DATABASE_URL` goes through a transaction pooler, which keeps no named prepared statements, and those are the default of pgx, the driver under GORM; with this mode every query runs on the unnamed statement. With `DT_DATABASE_HOST` set, the upstream connection is used as before. Test: `internal/database/pooler_dsn_test.go` (new).
- `main.go`: the origin of `server.public_host` (`DT_SERVER_PUBLIC_HOST`, the copy's own address) is always an allowed origin. Upstream's CORS check answers 403 to a request whose `Origin` is not on `server.cors_allow_origins`, unless it equals the scheme and `Host` header the server receives; a proxy that rewrites `Host` would otherwise make every sign-in from the copy's own address fail. Other origins are refused as before. Test: `public_origin_test.go` (new).
- The source link (AGPL-3.0 section 13):
  - `config/config.go` adds the `source_url` setting (`DT_SOURCE_URL`), empty by default, and `internal/resource/handler.go` returns it from `/api/v1/resource` as `source_url`, leaving the field out when it is empty. Test: `internal/resource/source_url_test.go` (new).
  - `web/src/components/SourceCodeLink.jsx` (new) is a "Source code" link to that address, or to https://github.com/tofu-gallery/donetick before the server has answered.
  - `web/src/views/Authorization/AuthFields.jsx`: the footer of the sign-in, sign-up and password pages shows it after "Privacy Policy · Terms of Use".
  - `web/src/views/components/NavBar.jsx`: signed in, it shows under the version line of the navigation (open the menu: "Expand menu" on a wide screen, "Open menu" on a phone).
  - `web/src/views/Landing/Footer.jsx` and `web/src/views/Landing/HomeHero.jsx`: upstream's landing page, which a copy shows at `/welcome` (and with `?site=marketing`), linked "Source Code", its GitHub icon and its "Github" button to donetick/donetick; they now open the same address.
- No analytics:
  - `web/index.html`: upstream's Plausible script tag, which loads `collector.wannaknow.link` when the page is served from donetick.com, is removed.
  - `web/src/analytics/posthogClient.js`: no longer reads `VITE_POSTHOG_KEY` or imports `posthog-js`, so the build leaves the PostHog SDK out and nothing is sent. The analytics and crash-report switches under Settings stay and have nothing to switch. OpenReplay is already commented out upstream, RevenueCat runs only in the native mobile apps, and the feedback and error-report forms send nothing unless a webhook address is built in (none is); otherwise they offer a pre-filled GitHub issue on upstream's repository, which the person opens themselves.
- `.dockerignore` (new): keeps a local `web/node_modules` and `web/dist` out of the build context.

## The sign-in secret

- Donetick signs its sign-in tokens with `DT_JWT_SECRET`, and also the links to files kept in local storage and the live-update connection's tokens. It refuses to start with a short or known-weak secret.
- On Tofu, when `DT_JWT_SECRET` is not set, `tofu-entrypoint.sh` sets it at each start to the SHA-256 of `donetick-jwt-secret:` followed by the whole `DATABASE_URL`, written as 64 hex characters. So a copy starts with nothing set by hand, and a restart keeps everyone signed in.
- Anything that changes `DATABASE_URL` changes the secret at the next start: a new database password, a different host, port or pooler address, other query options, or a database Tofu brings back under a different address. After that start, every sign-in token issued before is refused (`401`). The refresh tokens are random values stored hashed in the database, not signed with the secret, so a client holding one still gets a new sign-in token (checked through the API below; not checked in a browser).
- To keep the secret independent of the database address, set `DT_JWT_SECRET` as a hidden value on the app's Keys & settings page, at least 32 characters, for example the output of `openssl rand -base64 48`, and keep it unchanged afterwards. Setting it has the same effect as a change, once.
- The derived secret is only as secret as `DATABASE_URL`, which Tofu keeps as a hidden value.
- Without `DATABASE_URL` and without `DT_JWT_SECRET`, the entrypoint makes a random secret for that start only and says so in the log, as upstream does on a filesystem that keeps nothing.

## Accounts

- Donetick has its own accounts, not Tofu sign-in. Anyone who can open the address can sign up until the owner closes sign-up.
- There is no administrator of the whole app: every account that signs up gets its own circle (household) and is that circle's admin. Others join a circle with its invite code (Settings, Circle), and an account can also make sub-accounts for children.
- To close sign-up, set `DT_IS_USER_CREATION_DISABLED=true` and deploy again. The sign-in page then hides "Sign up", and the sign-up request gets `403`; existing accounts keep signing in. Make the owner's own account, and let the household sign up and join the circle, before closing it; or make the app private on Tofu.
- Password reset: no email server is set up, so a reset link is not emailed. It is printed in the app's runtime log (`=== PASSWORD RESET LINK ===`), starting with the copy's address (`DT_EMAIL_APP_HOST`). An owner who wants emails sets the `DT_EMAIL_*` settings for an SMTP server.

## What works differently on Tofu

- **Reminders are sent while the server runs.** The server checks every 3 minutes for notifications that have come due and sends those scheduled in the last 15 hours. A reminder that falls due while the app is asleep goes out at the first check after it wakes, if that is within 15 hours, and never after that. Notifications also need a channel the owner sets up: a Telegram or Pushover token, or a Discord webhook address entered in the app. Push notifications to the mobile apps need Firebase or Donetick Bridge, both off.
- **Files need a bucket.** Attachments and profile photos are kept in S3-compatible storage set with the `DT_STORAGE_*` settings, and this copy sets none, so uploading a file is expected to fail until the owner points it at a bucket (not tried). `DT_STORAGE_STORAGE_TYPE=local` keeps files on the container's disk, which is lost on every restart and deploy.
- **Live updates are optional.** Each browser can turn on "real-time updates" in Settings, which keeps a long-lived connection (server-sent events) open. Whether Tofu's hosting keeps such a connection open was not tested; without it, pages update when they are reloaded or refetched, as with the switch off.
- **Privacy Policy and Terms of Use** in the app's footers and pages are upstream's documents for the donetick.com service. They describe that service, not a copy run by its owner on Tofu.
- **Calendar feeds** (Settings, Calendar) give links on the copy's own address, `DT_SERVER_PUBLIC_HOST`; without it the server would name `http://localhost:2021`, its built-in default (from the code; not tried).

## Needs a real deploy to verify

- The image builds on Tofu's container hosting: `npm ci` from the lockfile on Linux (the lockfile lists the Linux builds of rollup, SWC and esbuild; the install was only run on macOS here), and the `--platform=$BUILDPLATFORM`, `TARGETOS` and `TARGETARCH` build arguments upstream's `Dockerfile` uses.
- The server answers `/` and `/api/v1/health` on `PORT`, the first start migrates Tofu's database through the pooler, and the runtime log shows no `prepared statement` errors (SQLSTATE 26000 or 42P05).
- Sign-up, sign-in, and creating and completing a chore work at the copy's address.
- Whether Tofu's proxy keeps the `Host` header. If it rewrites it, sign-in still works through `DT_SERVER_PUBLIC_HOST`, but a custom domain would have to be added to `DT_SERVER_CORS_ALLOW_ORIGINS` (with upstream's mobile-app origins `https://localhost,http://localhost,capacitor://localhost` if the mobile apps should connect).
- Whether live updates stay connected.
- The version shows as `dev`: Tofu's build may pass no `VERSION` or `COMMIT` build arguments.

## Licence and source code

- AGPL-3.0, for the server (`LICENSE.md`) and the web app (`web/LICENSE.md`). Both files and every copyright notice stay exactly as upstream has them.
- The Donetick name and logo belong to the upstream project; the gallery uses them only to say which app this is. The web app bundles upstream's translations (`web/public/locales`), icons and screenshots. No fonts are bundled, and the app loads nothing from other hosts.
- **Source offer (AGPL-3.0 section 13).** A modified copy that people use over a network has to offer them its own source code.
  - Every page before sign-in (sign-in, sign-up, password reset) and the signed-in navigation show a "Source code" link to `DT_SOURCE_URL`. The image sets it to https://github.com/tofu-gallery/donetick/tree/pin/2026-10-25, and a gallery copy saves it as a setting naming this pin's exact commit. This repository holds the server and the web app together, and it is public; a tag never moves.
  - The landing page's links to upstream's source open the same address. Its other links to upstream stay as they are: documentation, discussions, roadmap, releases, Docker Hub, the Home Assistant add-on and the licence file, and the feedback and error-report forms' pre-filled GitHub issue on donetick/donetick, which is where the app's own bugs are reported.
  - An owner who changes their copy's code runs a modified version of their own. They have to publish that source and set `DT_SOURCE_URL` to it on the app's Keys & settings page; the server reads it at start, so it takes effect with the next deploy.

## Verification

2026-10-25, on macOS arm64 against 127.0.0.1 only. No Docker, nothing deployed.

- Upstream's `develop` head (`584301c`) failed `go test .` with `internal/notifier/push/push.go:99:10: not enough return values`; v0.1.83 builds, so the pin starts there.
- `go vet . ./config/ ./internal/resource/ ./internal/database/` passed, and `go test . ./internal/resource/ ./internal/database/ -count=1` passed (upstream's tests in those packages and Tofu's three new ones), with Go 1.27.3 (`go.mod` asks for 1.24); `gofmt -l` was clean on the changed Go files.
- Tofu's own check on the pushed `tofu` branch (the GitHub import, then Tofu's static-lane and safety predicates) printed `safety ok (0 findings)` and `detect=container`; 813 files are imported, and the app is not on the static lane, as expected for an app that ships a `Dockerfile`.
- Built as the `Dockerfile` does, from the pushed branch's files (`git archive`), with Node 22.22 and npm 10.9: `npm ci --ignore-scripts --no-audit --no-fund` installed 1323 packages, `patch-package` applied its one patch, `vite build --mode selfhosted` built 248 files (7.1 MB, no source maps), which were copied to `frontend/dist`, and `CGO_ENABLED=0 go build` with the `Dockerfile`'s flags succeeded. The built web app holds no PostHog SDK and no Plausible tag.
- Run as the image would: the image's layout (`config/` with `selfhosted.yaml` copied from the example, `tofu-entrypoint.sh`, the binary), the `Dockerfile`'s `ENV DT_*` lines, `PORT`, and `DATABASE_URL=postgresql://donetick:<password>@127.0.0.1:<port>/donetick_r9?sslmode=require` pointing at a scratch PostgreSQL 14 with TLS on and every statement logged. The server answered in 3 s; `GET /` and `GET /login` answered 200 (`text/html`); `/api/v1/resource` returned `source_url` with this pin's address. The first start created 41 tables and recorded the 3 SQL migrations and 10 data migrations. Its log has two upstream lines a fresh database gives: `Error creating bot: Not Found` (no Telegram token) and one read of `bridge_settings` before that table exists; that read was the only database error of the whole run.
- Over the five starts below, the database log showed 2979 `execute <unnamed>` and no named prepared statement.
- Through the API: sign-up 201, sign-in 200, creating the chore "Water the plants (r9 check)" 200, and listing chores returned it.
- Browser (headless Chromium, 127.0.0.1 only, every other host blocked): the sign-in page's "Source code" link opens https://github.com/tofu-gallery/donetick/tree/pin/2026-10-25; signing in reached the app, and `/my/chores` showed the chore; the expanded desktop menu and the phone-width drawer both show the same link. No console errors, no failed request, and the app made no request to any other host. One run logged `AbortError: Transition was skipped. New ViewTransition started` once, from the browser's page transitions; a second run logged nothing.
- Restarted with the same `DATABASE_URL`: the earlier sign-in token still answered 200 and the chore was still there. With `DT_SERVER_PUBLIC_HOST=https://donetick-r9.example.test`, sign-in with that `Origin` answered 200, with `Origin: https://other.example.test` 403, and from the server's own origin 200.
- Restarted with `&application_name=r9check` added to `DATABASE_URL`: the earlier sign-in token answered 401, and its refresh token got a new one that answered 200.
- Restarted with `DT_IS_USER_CREATION_DISABLED=true`: `/api/v1/resource` said `is_user_creation_disabled: true`, a new sign-up answered 403, the existing account signed in (200), and a password reset printed `=== PASSWORD RESET LINK ===` with `https://donetick-r9.example.test/password/update?c=...` (`DT_EMAIL_APP_HOST`).
- Rebuilt the same way after the landing page change (`731c6d9`) and started on the same database: `/` and `/welcome` answered 200. In the browser, the landing page's "Source Code" link and GitHub icon link to https://github.com/tofu-gallery/donetick/tree/pin/2026-10-25, its "Github" button opened a tab for that address (blocked by the check, like every other host), and the links still on donetick/donetick are Discussions, GitHub Release, License and Changelog. No page error.
- The server, the scratch database and every build output were removed afterwards.
