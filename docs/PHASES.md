# LittleLiver — Implementation Phases

Fine-grained, TDD-ready implementation phases. Each phase is small enough for a focused 1–3 hour session and produces buildable, testable proof of progress.

**Resolved decisions:**
- **HEIC photos:** Convert HEIC → JPEG on upload via ImageMagick; thumbnails via Go stdlib (`image` package). All stored images are JPEG/PNG.
- **PDF generation:** maroto v2 for layout; charts pre-rendered as PNG via go-echarts/gonum and embedded.
- **Frontend charts:** Chart.js.
- **WHO data:** `//go:embed` CSV files, parsed at startup.
- **Service worker (v1):** App shell caching only; API calls fail gracefully offline.
- **Push notifications:** Click opens `/log/med?medication_id=X` with medication pre-filled. No action buttons.
- **Rate limiting:** Per-session, uniform (100 req/min).
- **med_logs.updated_at:** `updated_at DATETIME DEFAULT CURRENT_TIMESTAMP`, set on PUT.
- **Medication detail endpoint:** `GET /api/babies/:id/medications/:id` for single-detail.
- **Cron frequency:** Cleanup cron runs **hourly**.
- **Breast-direct + cal_density:** Return 400 if `cal_density` is provided with no volume (breast-direct).
- **Push subscription uniqueness:** `UNIQUE` constraint on `push_subscriptions.endpoint`; upsert on conflict.
- **Notification click URL:** `/log/med?medication_id=X` pattern.
- **ImageMagick in Dockerfile:** Added to the runtime stage for HEIC→JPEG conversion.
- **Spec DDL amendments (to apply to `docs/SPEC.md` during implementation):** `med_logs.updated_at` column addition, `UNIQUE` constraint on `push_subscriptions.endpoint`, and ImageMagick addition to Dockerfile runtime stage.
- **WHO percentile endpoint:** `GET /api/who/percentiles?sex=&from_days=&to_days=` returns 5 percentile curves (3rd, 15th, 50th, 85th, 97th). Frontend calls this alongside the dashboard.
- **Detail endpoints:** Generic metric pattern includes `GET /:entryId` for single-entry detail.

---

## Infrastructure & Foundation

- [x] **Phase 1: Go backend scaffolding and project structure**
  **Depends on:** Nothing
  **What to build:** Initialize Go module at `backend/` with `cmd/server/main.go` entrypoint, `internal/` package structure (`auth/`, `handler/`, `model/`, `store/`, `storage/`, `notify/`, `report/`, `who/`), `migrations/` directory. Set up a basic HTTP server that listens on `:8080` and returns a health-check JSON response. Add `Makefile` or build script. Wire up `go vet` in CI-ready form.
  **TDD approach:** Write a test that starts the server, hits `GET /health`, and asserts a 200 JSON response. Then write the minimal `main.go` and router to make it pass.
  **Proof of progress:** `go test ./...` passes; `go build ./cmd/server` produces a binary; `GET /health` returns `{"status":"ok"}`.

- [x] **Phase 2: Svelte frontend scaffolding**
  **Depends on:** Nothing (can run in parallel with Phase 1)
  **What to build:** Initialize Svelte SPA with TypeScript at `frontend/`. Configure `svelte.config.js` for SPA mode (static adapter). Set up ESLint + Prettier. Create a minimal `App.svelte` shell with a placeholder route. Add API client stub in `lib/api.ts`. Add `npm test` script (Vitest).
  **TDD approach:** Write a Vitest test that renders the App component and checks for a placeholder heading. Write the minimal component to pass.
  **Proof of progress:** `npm test` passes; `npm run build` produces static output; `npm run dev` serves the shell app.

- [x] **Phase 3: SQLite connection, migration runner, and core schema**
  **Depends on:** Phase 1
  **What to build:** SQLite database connection in `internal/store/db.go` with WAL mode enabled. A file-based migration runner that applies `.sql` files from `migrations/` in order. First migration creates: `users` (with sentinel row), `babies`, `baby_parents`, `invites`, `sessions`, and `push_subscriptions` tables per the spec schema. `push_subscriptions` must include a `UNIQUE` constraint on `endpoint` to support upsert on conflict.
  **TDD approach:** Write tests that (1) apply migrations to an in-memory SQLite DB, (2) verify all tables exist with correct columns, (3) verify the sentinel `deleted_user` row exists, (4) verify the UNIQUE constraint on `push_subscriptions.endpoint`. Write the migration runner and SQL to pass.
  **Proof of progress:** `go test ./internal/store/...` passes; all core tables are created with correct constraints.

- [x] **Phase 4: ULID generation and domain model types**
  **Depends on:** Phase 1
  **What to build:** ULID generation utility in `internal/model/ulid.go`. Domain types for `User`, `Baby`, `BabyParent`, `Session`, `Invite` in `internal/model/`. Validation functions for enums (sex, feed_type, etc.).
  **TDD approach:** Write tests that (1) ULID generation produces valid, sortable IDs, (2) enum validators accept valid values and reject invalid ones. Write minimal code to pass.
  **Proof of progress:** `go test ./internal/model/...` passes with model types and validation working.

- [x] **Phase 5: Dockerfile and local docker-compose**
  **Depends on:** Phase 1, Phase 2
  **What to build:** Multi-stage `Dockerfile` per spec (frontend build, backend build, runtime stage). The runtime stage must explicitly install ImageMagick for HEIC→JPEG conversion. A `docker-compose.yml` for local development mounting a volume for SQLite. Verify the built image starts and serves the health endpoint.
  **TDD approach:** Build the image and run a smoke test (curl health endpoint from the container). This is an infrastructure phase — the "test" is a successful build + health check.
  **Proof of progress:** `docker build` succeeds; container starts and responds to `GET /health`; ImageMagick is available in the runtime image.

## Authentication & Sessions

- [x] **Phase 6: Google OAuth flow (login, callback, session creation)**
  **Depends on:** Phase 3, Phase 4
  **What to build:** `internal/auth/` package: Google OAuth redirect (`GET /auth/google/login`), callback handler (`GET /auth/google/callback`) that exchanges code for token, upserts user record, creates session in DB, sets HttpOnly/Secure/SameSite=Lax cookie. Session store (`internal/store/sessions.go`) with create/get/delete/extend operations. 30-day sliding window expiry.
  **TDD approach:** Write tests that (1) login endpoint returns a redirect to Google with correct params, (2) callback handler with a mocked Google token exchange creates a user and session, (3) session cookie is set correctly, (4) expired sessions are rejected with 401. Mock the Google HTTP client. Write handlers to pass.
  **Proof of progress:** Auth flow tests pass end-to-end with mocked Google; session creation and cookie setting verified.

- [x] **Phase 7: Auth middleware, CSRF, and `/api/me`**
  **Depends on:** Phase 6
  **What to build:** Auth middleware that validates session cookie, extends sliding window, extracts user into context, and updates user timezone from `X-Timezone` header. CSRF token endpoint (`GET /api/csrf-token`) using HMAC-SHA256 derivation from session token. CSRF validation middleware for state-changing requests. `GET /api/me` returning current user info + linked babies. `POST /auth/logout` clearing session.
  **TDD approach:** Write tests that (1) requests without session cookie get 401, (2) valid session extends expiry, (3) CSRF token is deterministic per session, (4) state-changing requests without valid CSRF get 403, (5) `/api/me` returns correct user data, (6) logout clears session. Write middleware and handlers to pass.
  **Proof of progress:** Full auth middleware chain tested; CSRF flow working; `/api/me` returns data.

## Test Utilities

- [x] **Phase 8: Test fixtures and helpers package**
  **Depends on:** Phase 7
  **What to build:** A `testutil` package (e.g., `internal/testutil/`) providing reusable test helpers: `createTestUser()`, `createTestBaby()`, `authenticatedRequest()`, `setupTestDB()`. These helpers encapsulate common setup logic (in-memory DB with migrations applied, user creation with session, baby creation with parent linkage, building authenticated HTTP requests with valid session cookies and CSRF tokens). This prevents each subsequent phase from reinventing test boilerplate.
  **TDD approach:** Write tests that (1) `setupTestDB()` returns a migrated in-memory DB, (2) `createTestUser()` inserts a user and returns it, (3) `createTestBaby()` creates a baby linked to the given user, (4) `authenticatedRequest()` produces a request with valid session cookie and CSRF token. Write the helpers to pass.
  **Proof of progress:** `go test ./internal/testutil/...` passes; helpers produce valid test fixtures.

## Integration Test — Auth

- [x] **Phase 9: Integration test — Auth flow**
  **Depends on:** Phase 8
  **What to build:** End-to-end test covering: Google OAuth login (mocked provider) → session cookie set → fetch CSRF token → make authorized state-changing request with CSRF → verify success → logout → verify 401. Test sliding window extension. Test expired session rejection.
  **TDD approach:** This IS the test phase. Write integration tests using `httptest.Server` with the full router stack. Verify the complete auth lifecycle.
  **Proof of progress:** Integration test passes covering full auth lifecycle.

## Rate Limiting

- [x] **Phase 10: Rate limiting middleware**
  **Depends on:** Phase 8
  **What to build:** Per-session rate limiting middleware (100 req/min uniform across all endpoints, returns 429 when exceeded). Wire into the middleware chain.
  **TDD approach:** Write tests that (1) requests under the limit succeed, (2) requests exceeding 100/min receive 429, (3) rate limit resets after the window, (4) different sessions have independent limits. Write rate limiter middleware.
  **Proof of progress:** Rate limiter tested; 429 responses verified for excess requests.

## Baby Profiles & Invite System

- [x] **Phase 11: Baby CRUD endpoints**
  **Depends on:** Phase 8
  **What to build:** `POST /api/babies` (create baby, auto-link creator as parent), `GET /api/babies` (list user's babies), `GET /api/babies/:id` (get baby details with authorization check), `PUT /api/babies/:id` (update baby info — name, DOB, sex, diagnosis_date, kasai_date, default_cal_per_feed, notes). Baby store layer in `internal/store/babies.go`. Authorization: only linked parents can access.
  **TDD approach:** Write tests that (1) creating a baby returns it with a ULID and links the creator, (2) listing babies returns only the user's babies, (3) getting a baby the user is not linked to returns 403, (4) updating baby fields persists correctly, (5) notes field is stored and returned correctly. Write store + handlers to pass.
  **Proof of progress:** All baby CRUD tests pass; baby creation flow works end-to-end in tests.

- [x] **Phase 12: Invite code generation and join flow**
  **Depends on:** Phase 11
  **What to build:** `POST /api/babies/:id/invite` — generates 6-digit code, hard-deletes prior codes for that baby, handles collision retry (up to 5 times). `POST /api/babies/join` — redeems code, links parent to baby. Edge cases: expired code, used code, already-linked parent (friendly message), last retry failure. Invite store in `internal/store/invites.go`.
  **TDD approach:** Write tests that (1) generating invite returns 6-digit code with 24h expiry, (2) prior codes are deleted on new generation, (3) join with valid code links parent, (4) join with expired/used/invalid code returns generic error, (5) already-linked parent gets friendly response, (6) collision retry works. Write store + handlers to pass.
  **Proof of progress:** Full invite flow tested: generate, join, edge cases.

- [x] **Phase 13: Self-unlink from baby**
  **Depends on:** Phase 12
  **What to build:** `DELETE /api/babies/:id/parents/me` — unlink self from baby; if last parent, delete baby + all data (CASCADE). Returns 204.
  **TDD approach:** Write tests that (1) unlinking with other parents remaining keeps baby, (2) last parent unlinking deletes baby and all associated data, (3) both cases return 204. Write handler to pass.
  **Proof of progress:** Self-unlink tests pass with correct cascade behavior verified.

- [x] **Phase 14: Account deletion**
  **Depends on:** Phase 12
  **What to build:** `DELETE /api/users/me` — full deletion cascade per spec: identify last-parent babies, delete them, delete invites created by the deleted user, anonymize `logged_by`/`updated_by` to `deleted_user`, anonymize `invites.used_by` to `deleted_user` for invites redeemed by the deleted user, delete user record. Returns 204. **Design note:** The deletion handler must iterate a configurable list of table names for `logged_by`/`updated_by` anonymization (table-driven design). Phase 25 (medications) and Phase 26 (med-logs) add their tables to this list when implemented.
  **TDD approach:** Write tests that (1) account deletion anonymizes correctly across all configured tables, (2) CASCADE cleanup works for sessions/baby_parents/push_subscriptions, (3) `invites.used_by` is anonymized to `deleted_user` on account deletion, (4) invites created by deleted user are hard-deleted, (5) table-driven anonymization iterates the configured table list, (6) returns 204. Write handler to pass.
  **Proof of progress:** Account deletion tests pass with correct cascade behavior and anonymization verified.

## Metric Logging — Core Metrics

- [x] **Phase 15: Reusable metric CRUD pattern and basic feeding CRUD**
  **Depends on:** Phase 11
  **What to build:** Migration for `feedings` table with `(baby_id, timestamp)` index. A reusable metric handler pattern: create, list, get-detail, update, delete — with baby authorization, cursor pagination, date filtering, and ULID ordering. The detail endpoint follows the pattern `GET /api/babies/:id/<metric>/:entryId` (e.g., `GET /api/babies/:id/feedings/:entryId`). All UPDATE queries must explicitly set `updated_at = CURRENT_TIMESTAMP`. Implement for feedings: `POST /api/babies/:id/feedings`, `GET /api/babies/:id/feedings` (list), `GET /api/babies/:id/feedings/:entryId` (single entry detail), `PUT /api/babies/:id/feedings/:entryId`, `DELETE /api/babies/:id/feedings/:entryId`. Basic feeding fields only (no calorie calculation in this phase).
  **Note:** All metric table migrations must include a `(baby_id, timestamp)` index. This is part of the reusable pattern.
  **TDD approach:** Write tests for (1) creating a feeding stores fields correctly, (2) cursor pagination returns correct pages, (3) date filtering with timezone, (4) get-detail returns single entry, (5) update sets `updated_at`, (6) delete removes entry, (7) baby authorization rejects unauthorized access, (8) verify the `(baby_id, timestamp)` index exists on the feedings table after migration. Write store + handlers.
  **Proof of progress:** Full feeding CRUD tested; reusable metric pattern established; pagination verified.

- [x] **Phase 16: Feeding calorie calculation**
  **Depends on:** Phase 15
  **What to build:** Caloric calculation logic layered onto feeding CRUD: formula-based with `cal_density`, breast_milk default 20 kcal/oz, breast-direct with `default_cal_per_feed`. `used_default_cal` tracking. Validation: return 400 if `cal_density` is provided with breast-direct (no volume). Recalculate endpoint on `PUT /api/babies/:id` with `recalculate_calories=true` query param.
  **TDD approach:** Write tests for (1) formula feeding with cal_density calculates calories correctly, (2) breast_milk defaults to 20 kcal/oz, (3) breast-direct uses default_cal_per_feed, (4) `used_default_cal` flag set correctly, (5) breast-direct with cal_density returns 400, (6) recalculate endpoint updates all affected entries, (7) normal `PUT /api/babies/:id` returns just the baby object, while PUT with `recalculate_calories=true` returns `{ "baby": {...}, "recalculated_count": N }`. Write calculation logic.
  **Proof of progress:** Calorie computation tested for all scenarios; recalculate endpoint working.

- [x] **Phase 17: Stool and urine entry endpoints**
  **Depends on:** Phase 15 (reuses metric pattern)
  **What to build:** Migration for `stools` and `urine` tables (both with `(baby_id, timestamp)` indexes). Stool endpoints with color_rating validation (1-7), color_label enum, consistency/volume_estimate enums. Urine endpoints with color enum. Both follow the same CRUD pattern established in Phase 15 (including single-entry detail endpoint).
  **TDD approach:** Write tests for (1) stool creation with valid/invalid color_rating, (2) urine creation with color enum validation, (3) list/get-detail/update/delete for both. Reuse pagination tests. Write handlers.
  **Proof of progress:** Stool and urine CRUD tested; color rating validation working.

- [x] **Phase 18: Weight, temperature, and abdomen endpoints**
  **Depends on:** Phase 15
  **What to build:** Migrations for `weights`, `temperatures`, `abdomen_observations` tables (all with `(baby_id, timestamp)` indexes). Endpoints for all three metric types. Weight: `weight_kg` to 2 decimals, `measurement_source` enum. Temperature: `value` to 1 decimal, `method` enum. Abdomen: `firmness` enum (required), `tenderness` boolean, `girth_cm` optional.
  **TDD approach:** Write tests for (1) weight creation with valid measurement_source, (2) temperature creation with method validation, (3) abdomen with required firmness validation, (4) CRUD operations including detail endpoints for all three. Write handlers.
  **Proof of progress:** Three metric types fully tested; enum validations working.

- [x] **Phase 19: Skin observations, bruising, and lab results endpoints**
  **Depends on:** Phase 15
  **What to build:** Migrations for `skin_observations`, `bruising`, `lab_results` tables (all with `(baby_id, timestamp)` indexes). Skin: `jaundice_level` enum, `scleral_icterus` boolean, `rashes` text field, `bruising` text field. Bruising: `location` required, `size_estimate` enum required, `size_cm` optional, `color` text field. Labs: EAV-style (`test_name`, `value`, `unit`, `normal_range`). CRUD for all three.
  **TDD approach:** Write tests for (1) skin creation with jaundice_level enum, rashes and bruising text fields, (2) bruising with size_estimate validation and color field, (3) lab result creation with arbitrary test_name, (4) CRUD including detail endpoints for all. Write handlers.
  **Proof of progress:** All three metric types tested; lab results working with EAV pattern.

- [x] **Phase 20: General notes endpoint and `logged_by`/`updated_by` handling**
  **Depends on:** Phase 15
  **What to build:** Migration for `general_notes` table (with `(baby_id, timestamp)` index). Notes endpoint: content required, category enum. Cross-cutting: verify `logged_by` is immutable on update, `updated_by` is set to editing user on PUT, any linked parent can edit/delete any entry.
  **TDD approach:** Write tests for (1) note creation with required content, (2) category enum validation, (3) `logged_by` unchanged after edit by different parent, (4) `updated_by` set correctly, (5) cross-parent edit authorization. Write handlers.
  **Proof of progress:** General notes working; edit authorization and audit fields verified across all metric types.

## Integration Tests — Baby Lifecycle

- [x] **Phase 21: Integration test — Multi-parent baby lifecycle**
  **Depends on:** Phase 13, Phase 15, Phase 16, Phase 20
  **What to build:** End-to-end test: User A creates baby → generates invite → User B joins with code → both users log entries (feedings, stools) → verify both can read all entries → User B unlinks → verify User B loses access → User A unlinks (last parent) → verify baby and all data deleted. Additionally, test the recalculate_calories flow: create baby, log breast-direct feedings, change default_cal_per_feed with `recalculate_calories=true`, verify all feeding calories updated.
  **TDD approach:** Write integration test with two test users, full HTTP request chain. Verify data isolation, cleanup, and calorie recalculation.
  **Proof of progress:** Integration test passes covering multi-parent baby lifecycle and calorie recalculation.

- [x] **Phase 22: Integration test — Account deletion anonymization**
  **Depends on:** Phase 14, Phase 15, Phase 17, Phase 18, Phase 20
  **What to build:** End-to-end test: A user logs entries across multiple metric types (feedings, stools, temperatures, weights), deletes account, and verify all `logged_by`/`updated_by` fields are anonymized to `deleted_user` across all metric tables while entries remain intact. (Medication-related tables are tested in Phase 35 after medications are implemented.)
  **TDD approach:** Write account deletion test that asserts anonymization across all metric tables and entry preservation.
  **Proof of progress:** Integration test passes verifying anonymization across all metric tables.

## Photo Upload System

- [x] **Phase 23: R2 storage client and photo upload endpoint**
  **Depends on:** Phase 11
  **What to build:** `internal/storage/r2.go` — S3-compatible client for Cloudflare R2 (upload, delete, generate signed URL). `photo_uploads` table migration. `POST /api/babies/:id/upload` — validates file size (5MB max), MIME type (JPEG/PNG/HEIC), converts HEIC to JPEG via ImageMagick if needed, generates thumbnail (~300px wide) using Go stdlib `image` package, stores both in R2, creates `photo_uploads` row, returns R2 key. Use an interface so tests can use an in-memory mock.
  **TDD approach:** Write tests with a mock R2 client that (1) rejects files over 5MB, (2) rejects invalid MIME types, (3) HEIC input is converted to JPEG before storage, (4) stores original + thumbnail, (5) creates `photo_uploads` row with correct keys, (6) returns R2 key. Write storage + handler.
  **Proof of progress:** Upload endpoint tested with mock R2; validation, HEIC conversion, and thumbnail generation verified.

- [x] **Phase 24: Photo linking, unlinking, and signed URL replacement**
  **Depends on:** Phase 23, Phase 17, Phase 18, Phase 19, Phase 20
  **What to build:** On metric create/update, validate `photo_keys` against `photo_uploads` table (matching baby_id), set `linked_at`. Enforce 4-photo limit. On update, unlink removed photos (`linked_at = NULL`). On metric read (list/detail), replace R2 keys with signed URLs (1hr TTL) — each photo becomes `{ url, thumbnail_url }` with both original and thumbnail signed URLs. Apply to: stools, abdomen, skin, bruising, general_notes.
  **TDD approach:** Write tests that (1) creating a stool with valid photo_keys sets linked_at, (2) invalid/wrong-baby photo_keys are rejected, (3) exceeding 4 photos is rejected, (4) removing a photo on update nulls linked_at, (5) list response contains signed URLs not raw keys, (6) each photo includes both `url` and `thumbnail_url`, (7) detail response also contains signed URLs with `url` and `thumbnail_url`. Write the linking/signing logic.
  **Proof of progress:** Full photo lifecycle tested: upload, link, unlink, signed URL replacement with thumbnails.

## Medications & Med-Logs

- [x] **Phase 25: Medication CRUD (create, list, detail, update/deactivate)**
  **Depends on:** Phase 15
  **What to build:** Migration for `medications` table. `POST /api/babies/:id/medications` — create with name, dose, frequency, schedule times array, timezone from `X-Timezone` header, `logged_by` set to current user. `GET /api/babies/:id/medications` — list all (active and inactive). `GET /api/babies/:id/medications/:id` — single medication detail. `PUT /api/babies/:id/medications/:id` — update fields including `active=false` for deactivation, timezone update, `updated_by` set to current user. No DELETE endpoint.
  **TDD approach:** Write tests that (1) creation stores schedule as JSON array, sets timezone, and records `logged_by`, (2) listing returns both active and inactive, (3) detail endpoint returns single medication, (4) deactivation sets `active=false`, (5) DELETE method returns 405, (6) frequency enum validation, (7) timezone can be updated via PUT, (8) `updated_by` is set on update. Write store + handlers.
  **Proof of progress:** Medication CRUD tested; deactivation flow verified; no delete confirmed; audit fields working.

- [x] **Phase 26: Med-log endpoints (dose logging, editing, deletion)**
  **Depends on:** Phase 25
  **What to build:** Migration for `med_logs` table with `updated_at DATETIME DEFAULT CURRENT_TIMESTAMP` column. `POST /api/babies/:id/med-logs` — log dose (given or skipped, mutually exclusive), set `logged_by` to current user. Validate `baby_id` matches `medications.baby_id`. `GET /api/babies/:id/med-logs` with medication_id/date filters. `GET /api/babies/:id/med-logs/:entryId` — single med-log detail. `PUT /api/babies/:id/med-logs/:entryId` sets `updated_by` to current user and `updated_at = CURRENT_TIMESTAMP`. `DELETE /api/babies/:id/med-logs/:entryId` for corrections. `scheduled_time` nullable for ad-hoc doses.
  **TDD approach:** Write tests that (1) logging as given sets `given_at` and `skipped=false`, (2) logging as skipped sets `skipped=true` and `given_at=null`, (3) baby_id mismatch is rejected, (4) edit sets `updated_by` and `updated_at`, (5) delete works, (6) filtering by medication_id and date range, (7) `logged_by` is immutable on update, (8) `GET /api/babies/:id/med-logs/:entryId` returns single med-log detail. Write handlers.
  **Proof of progress:** Med-log CRUD tested; mutual exclusivity of given/skipped enforced; audit fields verified.

## Dashboard & Alerts

- [x] **Phase 27: Dashboard essentials**
  **Depends on:** Phase 15, Phase 17, Phase 18, Phase 25, Phase 26
  **What to build:** `GET /api/babies/:id/dashboard?from=&to=` returning: `summary_cards` (total feeds, calories, wet diapers, stools with color indicator, last temp, last weight), `stool_color_trend` (always last 7 days regardless of params), `upcoming_meds` (active only, with countdown). Default to today when from/to omitted. All aggregation server-side.
  **TDD approach:** Write tests that (1) seed various metric entries, (2) verify summary_cards counts/values, (3) stool_color_trend is always 7 days regardless of params, (4) upcoming_meds excludes deactivated, (5) default date behavior. Write the aggregation query layer.
  **Proof of progress:** Dashboard essentials endpoint returns correct summary_cards, stool_color_trend, and upcoming_meds.

- [x] **Phase 28: Dashboard chart data series**
  **Depends on:** Phase 27, Phase 18, Phase 19
  **What to build:** Add `chart_data_series` to the dashboard response: `feeding_daily` (aggregated daily kcal/volume), `diaper_daily` (stool + urine counts), `temperature` (individual readings), `weight` (individual readings), `abdomen_girth` (individual readings), `stool_color` (color-coded scatter data), `lab_trends` (per-test time series). All respect from/to params.
  **TDD approach:** Write tests that (1) chart_data_series aggregates feeding data correctly, (2) diaper_daily combines stool + urine, (3) temperature/weight/abdomen return individual readings, (4) stool_color returns color-coded data, (5) lab_trends groups by test_name. Write the additional aggregation queries.
  **Proof of progress:** Full chart_data_series tested with various seeded data scenarios.

- [x] **Phase 29: Alert system (acholic stool, fever, jaundice, missed medication)**
  **Depends on:** Phase 27
  **What to build:** `active_alerts` array in dashboard response. Alert types: `acholic_stool` (color_rating <= 3, cleared by >= 4), `fever` (method-specific thresholds, single most recent temp), `jaundice_worsening` (severe level or scleral icterus), `missed_medication` (scheduled doses >30 min past due with no med_log within +/-30 min). Alerts are global (ignore from/to). Each alert: `{ entry_id, alert_type, method?, value, timestamp }`.
  **Note:** Extract the missed-medication +/-30 min suppression check as a shared utility in `internal/store/` so Phase 34 (scheduler) can reuse it.
  **TDD approach:** Write tests for each alert type: (1) acholic stool triggers alert, pigmented stool clears it, (2) fever per each method threshold, sub-threshold clears, (3) jaundice with severe/scleral, cleared by normal observation, (4) missed medication detection with the +/-30 min window, (5) shared suppression utility returns correct result for logged/unlogged doses. Write alert computation logic.
  **Proof of progress:** All four alert types tested with trigger and clear conditions; shared suppression utility tested independently.

## Integration Test — Dashboard

- [x] **Phase 30: Integration test — Dashboard aggregation**
  **Depends on:** Phase 29
  **What to build:** End-to-end test: Log diverse entries over multiple days (feedings with different types, stools with various colors, temperatures with fever, weights, med-logs with given/skipped). Hit dashboard endpoint for today and for 7-day range. Verify all summary_cards, chart_data_series, stool_color_trend, upcoming_meds, and active_alerts are correct.
  **TDD approach:** Write integration test with a rich set of seeded data. Assert every field of the dashboard response.
  **Proof of progress:** Integration test validates complete dashboard aggregation accuracy.

## WHO Growth Standards

- [x] **Phase 31: WHO growth data embedding and percentile calculation**
  **Depends on:** Phase 4
  **What to build:** `internal/who/` package. Embed WHO weight-for-age LMS CSV tables (0-24 months, male + female) using `//go:embed`, parse at startup. Function: given sex, age in days, weight_kg, compute z-score and percentile. Function to generate percentile curves (3rd, 15th, 50th, 85th, 97th) for a given sex and age range.
  **TDD approach:** Write tests that (1) known weight/age/sex inputs produce expected z-scores (validated against WHO reference), (2) edge cases (day 0, day 730), (3) percentile curve generation returns correct number of points. Write the LMS calculation code.
  **Proof of progress:** WHO percentile calculations match reference values; percentile curves generated.

- [x] **Phase 32: WHO percentile endpoint**
  **Depends on:** Phase 31, Phase 8
  **What to build:** `GET /api/who/percentiles?sex=&from_days=&to_days=` endpoint returning the 5 percentile curves (3rd, 15th, 50th, 85th, 97th) as arrays of `{ age_days, weight_kg }` points. Uses the `internal/who/` package from Phase 31.
  **TDD approach:** Write tests that (1) male percentile curves return expected values at known age points, (2) female curves are different from male, (3) invalid sex param returns 400, (4) curves span the requested day range. Write the handler.
  **Proof of progress:** WHO percentile endpoint returns correct curve data for both sexes.

## Push Notifications

- [x] **Phase 33: Web Push subscription management**
  **Depends on:** Phase 7, Phase 8
  **What to build:** `POST /api/push/subscribe` — register push subscription (endpoint, p256dh, auth keys). Upsert on conflict (leveraging the `UNIQUE` constraint on `push_subscriptions.endpoint`). `DELETE /api/push/subscribe` — unregister. `internal/notify/` package with VAPID key management and Web Push sending (using a library or stdlib). Push subscription store.
  **TDD approach:** Write tests that (1) subscribe stores subscription with correct fields, (2) duplicate endpoint upserts existing subscription, (3) unsubscribe deletes, (4) sending a push notification calls the correct endpoint with VAPID auth (mock HTTP). Write store + notify package.
  **Proof of progress:** Subscription CRUD tested; upsert on duplicate endpoint verified; push send verified with mock.

- [x] **Phase 34: Medication reminder scheduler**
  **Depends on:** Phase 25, Phase 26, Phase 29, Phase 33
  **What to build:** Scheduler goroutine in `internal/notify/scheduler.go`. Every minute: query active medications, compute "now" per medication timezone, check if any scheduled time is due (initial, +15 min, +30 min). Suppression check: look for med_log within +/-30 min of original scheduled_time. Send push to all subscribed devices for that baby's parents. Notification payload must include `scheduled_time` (UTC), `medication_id`, medication name, and click URL `/log/med?medication_id=X`. Stateless re-derivation (no tracking table).
  **TDD approach:** Write tests that (1) medication due now triggers notification, (2) medication with logged dose is suppressed, (3) +15 min follow-up fires if no log, (4) +30 min follow-up fires if still no log, (5) logged dose suppresses follow-ups, (6) timezone conversion is correct (e.g., med at 08:00 America/New_York), (7) inactive meds skipped, (8) notification payload includes `scheduled_time`, `medication_id`, medication name, and click URL `/log/med?medication_id=X`. Use mock time and mock push sender.
  **Proof of progress:** Scheduler logic fully tested with various timing scenarios.

## Integration Test — Medication Flow

- [x] **Phase 35: Integration test — Medication flow**
  **Depends on:** Phase 34
  **What to build:** End-to-end test: Create medication with schedule → scheduler fires notification at scheduled time (mock time) → parent logs dose as given → verify suppression of follow-ups → next dose: no log → verify +15 min follow-up fires → log as skipped → verify +30 min suppressed → verify adherence ratio in dashboard. Additionally, test account deletion with medication tables: create medication and log doses, delete the user's account, verify `medications.logged_by`/`updated_by` and `med_logs.logged_by`/`updated_by` are anonymized to `deleted_user`.
  **TDD approach:** Write integration test with mock time progression and mock push sender. Verify full notification lifecycle and medication-table anonymization on account deletion.
  **Proof of progress:** Integration test passes covering medication notification, adherence, and account deletion anonymization of medication tables.

## Cron Jobs

- [x] **Phase 36: Cron jobs (invite cleanup, session cleanup, photo cleanup)**
  **Depends on:** Phase 12, Phase 6, Phase 24
  **What to build:** Three periodic cleanup tasks running **hourly** (goroutines with tickers or a lightweight cron): (1) Delete all invite codes older than 24 hours, (2) Delete expired sessions, (3) Delete orphaned/cascade photo_uploads rows and their R2 objects (`linked_at IS NULL AND uploaded_at < NOW() - 24h` OR `baby_id IS NULL`).
  **TDD approach:** Write tests that (1) old invites are deleted, recent ones kept, (2) expired sessions are deleted, active ones kept, (3) unlinked photos older than 24h are cleaned up with R2 delete calls, (4) baby-deleted photos (baby_id IS NULL) are cleaned up. Use mock R2 for photo cleanup.
  **Proof of progress:** All three cleanup jobs tested; correct rows deleted, correct rows preserved.

## Integration Test — Photo Flow

- [x] **Phase 37: Integration test — Photo flow**
  **Depends on:** Phase 36
  **What to build:** End-to-end test: Upload photo → receive R2 key → create stool entry with photo_key → verify linked_at set → read entry, verify signed URL returned (both `url` and `thumbnail_url`) → update entry removing photo → verify linked_at nulled → wait for cleanup window → run cleanup → verify R2 delete called. Also test: 5MB limit rejection, invalid MIME rejection, 4-photo limit.
  **TDD approach:** Write integration test with mock R2 client. Verify complete photo lifecycle including cleanup.
  **Proof of progress:** Integration test passes covering upload, link, unlink, signed URL, and cleanup.

## PDF Reports

- [x] **Phase 38: PDF report — text sections**
  **Depends on:** Phase 27, Phase 28
  **What to build:** `internal/report/` package. `GET /api/babies/:id/report?from=&to=` generates a PDF using maroto v2. This phase covers the text/table content: header (baby info, age, days post-Kasai), summary section, stool color log table, temperature log with fever flags, feeding summary (avg daily volume/calories), medication adherence ratio, notable observations section.
  **TDD approach:** Write tests that (1) PDF is generated without error for a baby with seeded data, (2) PDF contains expected text content (parse or check byte markers), (3) empty date range produces a valid but minimal PDF. Write the maroto v2 skeleton and text section generation.
  **Proof of progress:** PDF generation test passes; generated file is a valid PDF with expected text sections.

- [x] **Phase 39: PDF report — charts and photos**
  **Depends on:** Phase 38, Phase 31, Phase 24
  **What to build:** Pre-render charts as PNG using go-echarts/gonum: stool color distribution chart, weight chart with WHO percentile bands, lab trends chart. Embed chart PNGs in the PDF. Photo appendix section: fetch thumbnails from R2 (using signed URLs), embed in PDF. Complete the report endpoint.
  **TDD approach:** Write tests that (1) chart PNGs are generated correctly, (2) weight chart includes WHO percentile data, (3) lab trends chart groups by test_name, (4) photo appendix includes thumbnails (mock R2 fetch), (5) full PDF with charts is a valid file. Write chart rendering and photo embedding.
  **Proof of progress:** Full PDF report with charts and photos generated; validated as complete.

## Integration Test — Report Generation

- [x] **Phase 40: Integration test — Report generation**
  **Depends on:** Phase 39
  **What to build:** End-to-end test: Seed a baby with 30 days of varied data (all metric types, photos, medications). Generate PDF report for the date range. Verify: PDF is valid, contains baby header info, stool log, weight chart data, lab values, feeding summary, medication adherence, photo appendix. Parse PDF text content to verify key values.
  **TDD approach:** Write integration test that generates a PDF and validates its content (text extraction or structural checks).
  **Proof of progress:** Integration test validates PDF report contains all expected sections and data.

## Frontend Views

- [x] **Phase 41: Frontend auth flow and API client**
  **Depends on:** Phase 2, Phase 7
  **What to build:** Login page with "Sign in with Google" button. OAuth redirect handling. API client in `lib/api.ts` that includes session cookie, fetches CSRF token, attaches `X-CSRF-Token` and `X-Timezone` headers. 401 response interceptor that redirects to login. Svelte store for current user state.
  **TDD approach:** Write tests that (1) login page renders sign-in button, (2) API client attaches correct headers, (3) 401 triggers redirect, (4) user store updates on successful `/api/me` fetch (mock fetch). Write components and API client.
  **Proof of progress:** Login page renders; API client tested with mocks; auth redirect works.

- [x] **Phase 42: Baby selector, create baby, and join flow UI**
  **Depends on:** Phase 41
  **What to build:** First-login screen (create baby or enter invite code). Baby creation form (name, DOB, sex, diagnosis date, kasai date). Invite code entry form. Baby selector dropdown in app header for multi-baby switching. Svelte store for active baby.
  **TDD approach:** Write tests that (1) first-login screen shows only create/join options, (2) baby form validates required fields, (3) invite code submission calls correct API, (4) baby selector renders linked babies and switches active baby. Write components.
  **Proof of progress:** Baby creation and join flows render correctly; baby switching works.

- [ ] **Phase 43: Quick-log buttons and simple metric entry forms**
  **Depends on:** Phase 42
  **What to build:** Today view quick-log buttons (Feed, Wet Diaper, Stool, Temp). Simple metric entry forms (no photo upload): feeding form (feed type selector, volume, cal density, duration, notes), urine form, temperature form (method selector), weight form.
  **TDD approach:** Write tests for each form: (1) renders required fields, (2) validates input, (3) submits correct payload to API, (4) quick-log buttons trigger correct form. Write form components.
  **Proof of progress:** Quick-log buttons and simple metric forms render and submit correctly.

- [ ] **Phase 44: Photo-capable and specialized metric entry forms**
  **Depends on:** Phase 43
  **What to build:** Stool form (CSS color swatches with labels — 7 tappable swatches, NOT reference images — consistency, volume estimate, photo upload, notes), abdomen form, skin form (with "consistent lighting recommended" hint displayed when photo upload is triggered), bruising form, lab entry form (with quick-pick buttons that pre-fill test_name and unit), general notes form (with category and multi-photo).
  **TDD approach:** Write tests for each form: (1) renders required fields, (2) validates input (e.g., color_rating 1-7), (3) submits correct payload to API, (4) photo upload flow integrates with upload endpoint, (5) stool form renders 7 tappable CSS color swatches with labels, (6) lab form quick-pick buttons render and selecting one pre-fills test_name and unit, (7) skin form displays "consistent lighting recommended" hint on photo upload trigger. Write form components.
  **Proof of progress:** All photo-capable and specialized forms render and submit correctly; stool color swatches display; lab quick-pick works; skin lighting hint appears.

- [ ] **Phase 45: Today dashboard view**
  **Depends on:** Phase 43, Phase 27, Phase 29
  **What to build:** Today view consuming `GET /api/babies/:id/dashboard` (no from/to = today). Summary cards (total feeds, calories, wet diapers, stools with color indicator, last temp, last weight). Stool color trend mini-chart (last 7 days). Upcoming medications with countdown. Alert banners with client-side dismissal (localStorage). Quick-log buttons.
  **TDD approach:** Write tests that (1) summary cards display correct values from mock API response, (2) stool color trend renders dots, (3) upcoming meds show countdown, (4) alert banners render for each type, (5) dismissal persists to localStorage and hides alert, (6) recovery clears dismissed IDs. Write dashboard component.
  **Proof of progress:** Today view renders all sections with mock data; alert dismissal works.

- [ ] **Phase 46: Trends view — core charts**
  **Depends on:** Phase 45, Phase 32
  **What to build:** Trends view with date range selector (7d/14d/30d/90d/custom). Charts using Chart.js: stool color scatter (color-coded, using CSS color swatches with labels), weight curve with WHO percentile bands (fetched from `GET /api/who/percentiles`), temperature line with fever threshold. All data from the same dashboard endpoint with different from/to.
  **TDD approach:** Write tests that (1) date range selector updates API call params, (2) each chart component renders with mock data (verify canvas element exists), (3) WHO percentile bands appear on weight chart (data fetched from WHO percentile endpoint), (4) fever threshold line appears on temperature chart, (5) stool color scatter uses correct color coding. Write chart wrapper components.
  **Proof of progress:** Trends view renders date range selector and 3 core chart types with mock data.

- [ ] **Phase 47: Trends view — additional charts**
  **Depends on:** Phase 46
  **What to build:** Additional Chart.js charts in the trends view: abdomen girth line, feeding bar chart (daily kcal), diaper daily counts (stool + urine), lab trends multi-line with normal range shading. All data from the same dashboard endpoint.
  **TDD approach:** Write tests that (1) abdomen girth chart renders with mock data, (2) feeding bar chart shows daily kcal, (3) diaper chart combines stool + urine counts, (4) lab trends groups by test_name with normal range shading. Write chart wrapper components.
  **Proof of progress:** All 4 additional chart types render with mock data; full trends view complete with 7 charts.

- [ ] **Phase 48: Medication management UI**
  **Depends on:** Phase 44, Phase 25, Phase 26
  **What to build:** Medication list view (active and inactive, with visual distinction). Create medication form (name with pre-populated suggestions, dose, frequency, schedule time picker). Edit medication form. Deactivation toggle. Med-log history view per medication. Dose logging from notification click (pre-filled medication via URL param `/log/med?medication_id=X`). Med Given quick-log button on the today view.
  **TDD approach:** Write tests that (1) medication list renders active/inactive correctly, (2) creation form submits correct schedule JSON, (3) deactivation updates display, (4) med-log list shows given/skipped status, (5) dose logging form pre-fills medication from URL param, (6) Med Given quick-log button renders and triggers dose logging form. Write components.
  **Proof of progress:** Full medication management UI working; dose logging from notification click functional; Med Given quick-log button working.

- [ ] **Phase 49: PWA setup (service worker, manifest, install prompt)**
  **Depends on:** Phase 2
  **What to build:** `manifest.json` with app name, icons, theme color, display standalone. Service worker for app shell caching only (HTML/JS/CSS) — API calls are not cached and fail gracefully when offline. Service worker `push` event handler that displays a notification with title/body from the push payload. Service worker `notificationclick` handler that opens `/log/med?medication_id=X` (URL from notification data). Install prompt handling. Push notification permission request and subscription registration (calls `POST /api/push/subscribe`).
  **TDD approach:** Unit tests using mocked `ServiceWorkerGlobalScope` (e.g., via `msw` or manual mocks). Test `push` event handler displays notification with correct title/body. Test `notificationclick` handler calls `clients.openWindow` with correct URL. Additional tests: (1) manifest is valid JSON with required fields, (2) service worker registers successfully (mock), (3) push subscription registration calls correct API endpoint.
  **Proof of progress:** App is installable as PWA; service worker caches app shell; push subscription registers; push and notificationclick handlers tested.

- [ ] **Phase 50: Report generation UI and PDF download**
  **Depends on:** Phase 38, Phase 39, Phase 42
  **What to build:** Report page with date range picker. "Generate Report" button that calls `GET /api/babies/:id/report?from=&to=` and triggers PDF download. Loading state during generation. Preview summary of what will be included.
  **TDD approach:** Write tests that (1) date range picker renders and validates, (2) generate button calls correct API endpoint, (3) PDF response triggers download (mock fetch), (4) loading state shows during generation. Write report page component.
  **Proof of progress:** Report page renders; PDF download triggered on button click with mock.

- [ ] **Phase 51: Settings, invite sharing, and account management UI**
  **Depends on:** Phase 42, Phase 13, Phase 14
  **What to build:** Settings page: baby settings (edit name/DOB/sex/dates, adjust default_cal_per_feed with recalculate option), invite code generation + share UI (display code, copy button, expiry countdown), self-unlink from baby (with confirmation dialog), account deletion (with confirmation dialog). Baby selector management.
  **TDD approach:** Write tests that (1) invite generation displays code and expiry, (2) unlink shows confirmation and calls DELETE, (3) account deletion shows confirmation and calls DELETE, (4) default_cal_per_feed edit with recalculate checkbox. Write settings components.
  **Proof of progress:** Settings page functional; invite sharing, unlink, and account deletion flows tested.

## Frontend E2E Tests

- [ ] **Phase 52: Frontend E2E tests (Playwright)**
  **Depends on:** Phase 51
  **What to build:** End-to-end tests using Playwright covering the critical user flow: login (mocked OAuth) → baby creation → metric entry (e.g., feeding + stool) → dashboard display (verify summary cards reflect entered data) → alert dismissal. Configure Playwright to run against the full stack (backend + frontend).
  **Note:** Depends on Phase 51 and transitively all prior frontend and backend phases.
  **TDD approach:** This IS the test phase. Write Playwright tests that exercise the full user journey. Mock the OAuth provider but use the real backend and database.
  **Proof of progress:** Playwright E2E tests pass covering the full user journey.

## Deployment & Production Readiness

- [ ] **Phase 53: fly.io deployment and production configuration**
  **Depends on:** Phase 5, all backend phases
  **What to build:** `fly.toml` per spec. Configure fly.io secrets for all env vars (Google OAuth, R2, VAPID, session secret). Deploy. Verify persistent volume mounts, SQLite database survives restarts. Set up custom domain and TLS. Verify health endpoint in production.
  **TDD approach:** Deploy and run a smoke test: hit health endpoint, verify OAuth redirect works, verify R2 connectivity. **Note:** This is a deployment verification phase — an exception to the TDD rule. The "test" is a successful deploy + smoke test.
  **Proof of progress:** App deployed to fly.io; health check passes; OAuth login flow works in production.

- [ ] **Phase 54: SQLite backup automation**
  **Depends on:** Phase 53
  **What to build:** Automated daily backup: SQLite `.backup` command copies DB to R2 via a cron/scheduled task. Verify backup restores correctly.
  **TDD approach:** Write a test that (1) backup produces a valid SQLite file, (2) backup file is stored in R2 (mock), (3) restored backup matches original data. Write backup script.
  **Proof of progress:** Backup file stored in R2; restore verified.
