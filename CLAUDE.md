# CLAUDE.md -- Project Context (Authoritative)

> **Note:** This is a **project-specific** CLAUDE.md. It applies only to this project.
> For personal preferences across all projects, use `~/.claude/CLAUDE.md`.

---

## Authority & Usage Rules

**This document is the AUTHORITATIVE source for understanding this project.**

Treat this file as the **single source of truth**. Do NOT rediscover information that is already documented here.

**Required behavior:**
* **Trust this file over repository inference** -- If something is documented here, accept it as fact
* **Do NOT rescan the repository** to rediscover high-level context already provided
* **Only read additional files when explicitly required for the task** -- Don't explore "just in case"
* **If something is not described here, assume it is out of scope** unless the task directly requires it

**Why this matters:**
This prevents token-heavy "verification scans" and repeated exploration. The information here is stable, intentional, and should eliminate the need to re-learn the project structure in every conversation.

---

## Project Summary

**Purpose:** A personal-use web application for parents to track daily health metrics of an infant recovering from the Kasai portoenterostomy procedure (biliary atresia). Enables both parents to log data from their phones, view trends on a dashboard, and generate printable clinical summaries for hepatologist appointments.

**Stack:**
* **Backend:** Go (stdlib + minimal dependencies), single binary
* **Database:** SQLite on fly.io persistent volume
* **Frontend:** Svelte SPA (TypeScript), lightweight bundle
* **Photo Storage:** Cloudflare R2 (S3-compatible, zero egress)
* **Auth:** Google OAuth 2.0 (sole identity provider)
* **Hosting:** fly.io (single machine, persistent volume, built-in TLS)
* **PWA:** Service worker + manifest (installable, push notifications via Web Push / VAPID)
* **Charts:** Chart.js or Apache ECharts (via Svelte wrapper)
* **PDF Reports:** Server-side Go PDF generation (e.g., `go-pdf` or `maroto`)

**Full Specification:** `docs/SPEC.md` -- the authoritative product spec. Read it for detailed field definitions, API endpoints, alert logic, and phased delivery plan.

---

## Development Methodology

### Red-Green TDD (STRICT)

This project follows **strict Red-Green-Refactor TDD**. This is non-negotiable.

**The cycle for every change:**

1. **RED** -- Write a failing test first. The test must fail for the right reason (not a compile error, but an actual assertion failure or missing behavior).
2. **GREEN** -- Write the **minimal** code to make the test pass. No more, no less.
3. **REFACTOR** -- Clean up the code while keeping all tests green. No new behavior in this step.

**Rules:**
* **Never write production code without a failing test first**
* **Never write more production code than is needed to pass the current failing test**
* **All tests must be green before any commit** -- no exceptions
* **Target 90%+ code coverage** across both backend and frontend
* **Run the full test suite before committing** -- if any test is red, fix it before committing

### Test Commands

**Backend (Go):**
```bash
cd backend && go test ./... -v -cover
```

**Frontend (Svelte/TypeScript):**
```bash
cd frontend && npm test
```

### What This Means in Practice

When asked to implement a feature:
1. Start by writing test(s) that define the expected behavior
2. Run the tests -- confirm they fail (RED)
3. Implement the minimum code to pass (GREEN)
4. Refactor if needed, re-run tests to confirm still green
5. Only then move to the next piece of behavior

When asked to fix a bug:
1. Write a test that reproduces the bug (fails)
2. Fix the bug (test passes)
3. Refactor if needed

---

## Architecture Overview

```
littleliver/
  backend/
    cmd/server/main.go            -> Entrypoint
    internal/
      auth/                       -> Google OAuth, sessions, middleware
      handler/                    -> HTTP handlers (REST JSON API)
      model/                      -> Domain types
      store/                      -> SQLite repository layer
      storage/                    -> R2/S3 photo upload
      notify/                     -> Web Push (VAPID)
      report/                     -> PDF generation
      who/                        -> WHO growth data + percentile calc
    migrations/                   -> SQL migration files
    go.mod
    go.sum
  frontend/
    src/
      routes/                     -> Svelte pages
      components/                 -> Reusable UI components
      lib/                        -> API client, stores, utils
      service-worker.ts           -> PWA + push notifications
    static/                       -> Icons, manifest.json
    package.json
    svelte.config.js
  docs/
    SPEC.md                       -> Product specification
  fly.toml
  Dockerfile
  CLAUDE.md                       -> You are here
```

**Data flow:** HTTP Request -> Middleware (auth, CSRF, rate limit) -> Handler -> Service/Store -> SQLite

**API style:** RESTful JSON. All endpoints require authentication (session cookie set after OAuth). Pattern: `/api/babies/:id/<metric>` for all metric types.

**Photo flow:** Client uploads to backend -> backend stores in R2 -> R2 object key saved in metric entry -> signed URLs generated for retrieval (no public bucket access).

---

## Domain Context

This is a **medical tracking app for post-Kasai biliary atresia infants**. Key domain concepts:

* **Stool color rating (1-7):** The most critical metric. Ratings 1-3 are **acholic** (no bile flow) and trigger alert banners. This is the primary indicator of Kasai procedure failure.
* **Cholangitis warning:** Fever >= 38.0C (rectal) or >= 37.5C (axillary) triggers an urgent warning. Fever after Kasai can indicate cholangitis (bile duct infection).
* **WHO growth percentiles:** Weight plotted against WHO Child Growth Standards (weight-for-age, sex-specific, 0-24 months). LMS method for z-score calculation.
* **Total bilirubin:** Key prognostic lab marker. Goal is < 2.0 mg/dL by 3 months post-Kasai.
* **Medication adherence:** Common post-Kasai meds include UDCA (ursodiol), Bactrim, fat-soluble vitamins (A, D, E-TPGS, K), and iron.

**Users:** Two parents per baby, equal access. Google OAuth only. Multi-baby support from day one.

**Invite system:** Parent A creates baby profile -> generates single-use invite code (expires 24h) -> Parent B enters code -> linked.

---

## Tracked Metrics

Eleven metric types, each with its own table and CRUD endpoints:

1. **Feedings** -- type, volume, caloric density, duration, notes
2. **Urine** -- wet diaper tracking, color
3. **Stools** -- color rating (1-7), consistency, volume, photo (R2), notes (CRITICAL metric)
4. **Weight** -- kg to 2 decimal places, source (home/clinic)
5. **Abdomen circumference** -- cm to 1 decimal place, optional photo
6. **Temperature** -- degrees C, method (rectal/axillary/ear/forehead)
7. **Skin/Jaundice** -- jaundice level, scleral icterus, photo
8. **Bruising** -- location, size, photo
9. **Medications** -- schedule definitions + administration log (given/skipped)
10. **Lab results** -- bilirubin, ALT, AST, GGT, albumin, INR, platelets
11. **General notes** -- categorized observations with up to 4 photos

---

## API Endpoints

All under session-authenticated context. Full details in `docs/SPEC.md` section 5.

* **Auth:** `/auth/google/login`, `/auth/google/callback`, `/auth/logout`, `/api/me`
* **Babies:** CRUD at `/api/babies`, invite at `/api/babies/:id/invite`, join at `/api/babies/join`
* **Metrics:** Pattern `/api/babies/:id/<metric>` with POST (create), GET (list with date range), PUT (edit), DELETE
* **Photos:** `POST /api/upload` returns R2 key
* **Med schedules:** CRUD at `/api/babies/:id/med-schedules`
* **Push:** `POST/DELETE /api/push/subscribe`
* **Dashboard:** `GET /api/babies/:id/dashboard?from=&to=`
* **Report:** `GET /api/babies/:id/report?from=&to=` (PDF download)

---

## Invariants, Conventions & Preferences

### Technical Invariants

These are **always true** and should never be re-inferred from code:

* **IDs:** UUIDs (TEXT) for all primary keys
* **Timestamps:** DATETIME type in SQLite, UTC always
* **Money/Measurements:** Weight in kg (2 decimal places), temperature in Celsius (1 decimal place), abdomen in cm (1 decimal place), volume in mL
* **Stool color ratings:** Integer 1-7 only. 1-3 = acholic (ALERT), 4-7 = pigmented (OK)
* **Migrations:** All schema changes via SQL migration files in `backend/migrations/` only (never direct DDL)
* **Photos:** Always stored in R2, never in SQLite. Only R2 object keys stored in DB. Access via signed URLs only.
* **Sessions:** HttpOnly, Secure, SameSite=Lax cookies
* **Invite codes:** Single-use, expire after 24 hours
* **SQL:** Parameterized queries only (no string interpolation, no injection risk)
* **CGO:** Required for SQLite (`CGO_ENABLED=1`)

### Code Conventions

* **Go:** Follow standard Go conventions (gofmt, effective Go)
* **Go error handling:** Early return style -- check error, return early, keep happy path unindented
* **Go project layout:** `cmd/` for entrypoints, `internal/` for private packages (not importable externally)
* **TypeScript:** Strict mode enabled in frontend
* **Svelte:** Component-based, reactive stores for shared state
* **API responses:** JSON, consistent error format across all endpoints
* **Testing:** Go stdlib `testing` package for backend, appropriate Svelte test tooling for frontend
* **Linting:** Use `go vet` and `golint` for Go; ESLint + Prettier for frontend

### Project Preferences

* **Testing:** Strict Red-Green TDD (see Development Methodology above). 90%+ coverage target.
* **API Design:** RESTful, plural nouns for resources (e.g., `/feedings`, `/stools`)
* **Commits:** All tests must pass before committing. Use conventional commits format (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`)
* **Minimal dependencies:** Go stdlib preferred where possible. Only add external deps when the stdlib alternative is significantly worse.
* **Single binary:** The Go backend serves both the API and the built Svelte static files
* **No over-engineering:** This is a personal-use app for two users. Simplicity over scalability.

---

## Boundaries & Constraints

### Off-Limits (Do Not Modify)

Unless explicitly requested:

* `/.git/` -- Git metadata
* `/docs/SPEC.md` -- Product specification (reference only, do not modify)
* Build outputs (`/frontend/build/`, `/backend/cmd/server/server`)
* `node_modules/` -- Frontend dependencies
* Generated/vendored files

### Behavioral Constraints

Unless explicitly requested, avoid:

* Full repository scans or architecture analysis
* Suggesting stack replacements or major refactors (the stack is chosen and locked)
* "Improving" working systems beyond the task scope
* Touching files outside the current task area
* Adding external Go dependencies without justification (stdlib first)

### Locked-In Assumptions

These are intentional; do not question or re-validate:

* **Go + SQLite + Svelte stack** -- chosen deliberately, not up for debate
* **Google OAuth only** -- no email/password, no other providers
* **fly.io hosting** -- single machine, persistent volume, no HA needed
* **Cloudflare R2** -- for photo storage, S3-compatible API
* **Single binary serving** -- Go serves static frontend files + API
* **Personal use scale** -- two parents, one baby (multi-baby supported but not high-traffic)

---

## Environment Variables

Required for production (see `docs/SPEC.md` section 11.3):

```
DATABASE_PATH=/data/littleliver.db
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
SESSION_SECRET=...
R2_ACCOUNT_ID=...
R2_ACCESS_KEY_ID=...
R2_SECRET_ACCESS_KEY=...
R2_BUCKET_NAME=littleliver-photos
R2_PUBLIC_URL=...
VAPID_PUBLIC_KEY=...
VAPID_PRIVATE_KEY=...
BASE_URL=https://littleliver.fly.dev
```

Never commit actual secrets. Use `.env.example` for documentation.

---

## Development Phases

From `docs/SPEC.md` section 14:

1. **Phase 1 (Core):** Scaffolding, OAuth, baby profiles, invite system, stool/feeding/temp logging, basic dashboard
2. **Phase 2 (Complete Tracking):** All remaining metrics, medication management, alert banners
3. **Phase 3 (Notifications + Charts):** PWA, Web Push, medication reminders, charting, WHO percentiles
4. **Phase 4 (Reports + Polish):** PDF generation, UI polish, offline resilience, backup automation, production deploy

---

## Documentation Map

* `CLAUDE.md` -- **Authoritative architectural overview and constraints** (you are here)
* `docs/SPEC.md` -- Full product specification (metrics, API, schema, deployment, phases)

---

## Working Approach

When completing tasks:

1. **Start here** -- Use CLAUDE.md as the authoritative context foundation (do NOT rescan the repository for information already documented)
2. **Follow TDD** -- Write failing test FIRST, then minimal code to pass, then refactor. No exceptions.
3. **Read targeted files** -- Only open files directly relevant to the task
4. **Minimal changes** -- Prefer focused edits over broad refactors
5. **Follow patterns** -- Match existing code style and architecture
6. **Run tests** -- Confirm all tests pass before considering a task complete
7. **Ask only when blocked** -- Proceed with constraints above if information seems missing but isn't required

**If the task truly cannot be completed without more info, ask specific questions.**

---

## Project Specific Information

### Alert Logic (must be implemented correctly)

* **Acholic stool alert:** Display prominent warning banner when stool color_rating <= 3 is logged. Message should advise contacting hepatology team.
* **Cholangitis/fever alert:** Display urgent warning when temperature >= 38.0C (rectal) or >= 37.5C (axillary). Message: "Fever detected. Contact your hepatology team immediately. Fever after Kasai can indicate cholangitis."

### Medication Reminder System

* Go backend scheduler (goroutine with ticker or lightweight cron)
* Checks every minute for medications due within the next minute
* Sends Web Push to all subscribed devices for that baby's parents
* Follow-up reminder if not logged within 15 minutes (max 2 follow-ups per dose)

### WHO Growth Standards

* Embedded in Go as LMS values (weight-for-age, 0-24 months, sex-specific)
* Source: WHO Anthro (https://www.who.int/tools/child-growth-standards/standards/weight-for-age)
* Percentile curves: 3rd, 15th, 50th, 85th, 97th

### SQLite Notes

* Indexes on `(baby_id, timestamp)` for all metric tables
* Full schema in `docs/SPEC.md` section 10
* Backup via SQLite `.backup` command for consistent snapshots
* Persistent volume at `/data` on fly.io

---
