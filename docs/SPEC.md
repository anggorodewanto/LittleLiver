# LittleLiver — Post-Kasai Baby Monitoring App

## Product Specification v1.0

---

## 1. Purpose

A personal-use web application for parents to track daily health metrics of an infant recovering from the Kasai portoenterostomy procedure (biliary atresia). The app enables both parents to log data from their phones, view trends on a dashboard, and generate printable clinical summaries for hepatologist appointments.

---

## 2. Users & Access Model

### 2.1 Authentication
- **Google OAuth 2.0** — each parent signs in with their own Google account.
- No email/password accounts. Google is the sole identity provider.
- On first login, a parent either **creates a new baby profile** or is **invited to an existing one** by the other parent via a share/invite code.

### 2.2 Authorization
- A baby profile has **unlimited authorized parents** (Google account IDs). No maximum.
- All authorized parents have equal read/write access to all data for that baby.
- Any linked parent can generate invite codes. All invite codes have a **fixed 24-hour expiration**. Generating a new invite code **hard-deletes ALL prior codes** for that baby (used, expired, or unused) — only one active invite code per baby at a time. A cron job periodically deletes ALL invite codes older than 24 hours (both used and unused) across all babies. The server checks `used_at IS NOT NULL` as a rejection condition but returns the same generic "invalid or expired code" error for all failure cases.
- If an already-linked parent redeems an invite code for a baby they are already linked to, show a friendly "You're already linked to this baby" message (no error).
- **Self-unlink:** A parent can unlink themselves from a baby (but not other parents) via `DELETE /api/babies/:id/parents/me`. If the last remaining parent unlinks, the baby and all associated data are deleted.
- **First login (no existing links):** The user sees only two options — "Create Baby" or "Enter Invite Code." There is no other entry path.

### 2.3 Multi-Baby Support
- Data model supports multiple children per parent from day one.
- A parent can switch between baby profiles via a selector in the app header.
- Each baby has its own independent dataset.

---

## 3. Tracked Metrics

### 3.1 Feeding / Intake (multiple entries per day)

| Field | Type | Notes |
|-------|------|-------|
| Timestamp | datetime | Auto-filled, editable |
| Feed type | enum | `breast_milk`, `formula`, `fortified_breast_milk`, `solid`, `other` |
| Volume (mL) | number | Optional for breast-direct feeds |
| Caloric density (kcal/oz) | number | Optional. When omitted, standard defaults apply: ~20 kcal/oz for formula and breast milk. User can override per entry by providing an explicit value. Relevant for fortified feeds (e.g., 24 or 30 cal/oz). |
| Duration (min) | number | For breastfeeding sessions |
| Notes | text | Free-form (e.g., "tolerated well", "vomited after") |

**Caloric intake calculation:** All feed types (including `solid` and `other`) can optionally specify `cal_density` and `volume_ml`. When both are provided, calories are calculated using the standard formula: `kcal = volume_ml × (cal_density / 29.5735)` (where `1 oz = 29.5735 mL`). If either field is missing, caloric intake is left null for that entry. For breast-direct feeds with no volume, a configurable default estimate is used: **~67 kcal per session** (based on an average ~100 mL intake at 20 kcal/oz: `100 × 20 / 29.5735 ≈ 67.6 kcal`). This default is stored as `default_cal_per_feed` on the baby profile and can be adjusted via `PUT /api/babies/:id`.

**Denormalized `calories` column:** The computed caloric value is stored as a `calories REAL` column on the `feedings` table. This value is computed and stored on insert/update using the formula above (or the baby's `default_cal_per_feed` for breast-direct feeds without volume). A `used_default_cal BOOLEAN DEFAULT false` column tracks whether the feeding's calories were computed using the baby's `default_cal_per_feed`. When `default_cal_per_feed` is changed on the baby via `PUT /api/babies/:id`, the response includes a count of affected entries. The parent can then choose to recalculate all existing feeding entries where `used_default_cal = true` — this is a parent-triggered action, not automatic.

### 3.2 Urine Output (multiple entries per day)

Each row represents a single wet diaper event (logged with a timestamp). Urine and stool are separate entries — for a combined diaper, the parent logs two entries (one urine, one stool).

| Field | Type | Notes |
|-------|------|-------|
| Timestamp | datetime | Auto-filled, editable |
| Color | enum | `clear`, `pale_yellow`, `dark_yellow`, `amber`, `brown` |
| Notes | text | |

### 3.3 Stool Output (multiple entries per day) ⭐ Critical

| Field | Type | Notes |
|-------|------|-------|
| Timestamp | datetime | Auto-filled, editable |
| Stool color rating | integer 1–7 | Based on standard Infant Stool Color Card. **1–3 = acholic (ALERT)**, 4–7 = pigmented (good) |
| Stool color reference | enum | `white`, `clay`, `pale_yellow`, `yellow`, `light_green`, `green`, `brown` — maps to 1–7 |
| Photo | image | Uploaded from camera/gallery. Stored in R2. |
| Consistency | enum | `watery`, `loose`, `soft`, `formed`, `hard` |
| Volume estimate | enum | `small`, `medium`, `large` |
| Notes | text | |

**Alert logic:** If stool color ≤ 3 is logged, the app should display a prominent warning banner suggesting the parent contact their hepatology team. This is the primary indicator of bile flow failure.

### 3.4 Weight (typically 1x/day or per clinic visit)

| Field | Type | Notes |
|-------|------|-------|
| Date | date | |
| Weight (kg) | number | To 2 decimal places (e.g., 4.35) |
| Measurement source | enum | `home_scale`, `clinic` |
| Notes | text | |

Weight is plotted against **WHO Child Growth Standards** (weight-for-age percentiles, sex-specific). **Percentile is NOT stored** — it is computed on-the-fly from WHO growth data based on the baby's age at the time of the weight entry.

### 3.5 Abdomen Observations (1–2x/day)

| Field | Type | Notes |
|-------|------|-------|
| Timestamp | datetime | |
| Firmness | enum | `soft`, `firm`, `distended` — required |
| Tenderness | boolean | Default false |
| Girth (cm) | number | To 1 decimal place. Optional |
| Photo | image | Optional — for visual distension tracking |
| Notes | text | |

Increasing abdominal girth can indicate ascites or organomegaly — trend matters more than absolute number. Firmness and tenderness provide qualitative context alongside the measurement.

### 3.6 Temperature (multiple per day)

| Field | Type | Notes |
|-------|------|-------|
| Timestamp | datetime | |
| Temperature (°C) | number | To 1 decimal place |
| Method | enum | `rectal`, `axillary`, `ear`, `forehead` |
| Notes | text | |

**Alert logic:** If temperature ≥ 38.0°C (rectal), ≥ 37.5°C (axillary), ≥ 38.0°C (ear), or ≥ 37.5°C (forehead), display a **cholangitis warning** banner: *"Fever detected. Contact your hepatology team immediately. Fever after Kasai can indicate cholangitis."*

### 3.7 Skin / Jaundice Observations (1–2x/day)

| Field | Type | Notes |
|-------|------|-------|
| Timestamp | datetime | |
| Jaundice level | enum | `none`, `mild_face`, `moderate_trunk`, `severe_limbs_and_trunk` |
| Scleral icterus | boolean | Yellowing of eyes |
| Photo | image | Consistent lighting recommended — app should note this |
| Notes | text | |

### 3.8 Bruising Observations (as needed)

| Field | Type | Notes |
|-------|------|-------|
| Timestamp | datetime | |
| Location on body | text | e.g., "left arm", "torso" |
| Size estimate | enum | `small_<1cm`, `medium_1-3cm`, `large_>3cm` — required |
| Size (cm) | number | Optional precise measurement |
| Color | text | e.g., "red", "purple", "yellow-green" |
| Photo | image | |
| Notes | text | |

New or worsening bruising can indicate vitamin K deficiency / coagulopathy.

### 3.9 Medications (log + reminders)

| Field | Type | Notes |
|-------|------|-------|
| Medication name | text | Pre-populated suggestions: `UDCA (ursodiol)`, `Sulfamethoxazole-Trimethoprim (Bactrim)`, `Vitamin A`, `Vitamin D`, `Vitamin E (TPGS)`, `Vitamin K`, `Iron`, `Other` |
| Dose | text | e.g., "50mg", "0.5mL" |
| Frequency | enum | `once_daily`, `twice_daily`, `three_times_daily`, `as_needed`, `custom` |
| Scheduled times | time[] | e.g., [08:00, 20:00] for twice daily. Stored as **local time strings** (not UTC). Interpreted per the medication's stored timezone (see §6.2). |
| Timezone | text | IANA timezone (e.g., `America/New_York`), set at creation time from the creator's `X-Timezone` header. All notification scheduling uses this timezone, not the individual user's timezone. This prevents dose drift and double-dosing across timezone boundaries. |
| Given at | datetime | Set to `NOW()` when parent taps "given" (not `scheduled_time`). Null when skipped. |
| Skipped | boolean | Mutually exclusive with `given_at`: `skipped=true` → `given_at` is null; `skipped=false` → `given_at` is non-null. |
| Skip reason | text | Optional even when `skipped=true`. |
| Notes | text | e.g., "spit up half the dose" |

**Reminder system:** see §6 (Push Notifications).

### 3.10 Lab Results (per clinic visit, entered manually)

Stored as individual test entries using an EAV-style table (`test_name`, `value`, `unit`, `normal_range`, `notes`). Each row is one test result. Entries on the same date are implicitly grouped (no explicit visit_id). The schema is generic to support any lab test.

The **UI** suggests common Kasai-relevant tests as quick-pick options: `total_bilirubin`, `direct_bilirubin`, `ALT`, `AST`, `GGT`, `albumin`, `INR`, `platelets`. Selecting a quick-pick pre-fills the `test_name` and `unit` fields. Parents can also enter arbitrary test names.

| Field | Type | Notes |
|-------|------|-------|
| Date | date | |
| Test name | text | Free-form; UI suggests common Kasai tests as quick-picks |
| Value | text | The result value |
| Unit | text | e.g., "mg/dL", "U/L", "×10³/µL" |
| Normal range | text | Optional, e.g., "0.1–1.2" |
| Notes | text | |

**Quick-pick reference (UI only, not a schema constraint):**
| Test | Typical Unit | Clinical Relevance |
|------|-------------|-------------------|
| total_bilirubin | mg/dL | Key prognostic marker — goal is < 2.0 by 3 months post-Kasai |
| direct_bilirubin | mg/dL | |
| ALT | U/L | |
| AST | U/L | |
| GGT | U/L | |
| albumin | g/dL | |
| INR | — | Coagulation — elevated = concern |
| platelets | ×10³/µL | Low = possible portal hypertension |

### 3.11 General Notes / Observations (as needed)

| Field | Type | Notes |
|-------|------|-------|
| Timestamp | datetime | |
| Category | enum | `behavior`, `sleep`, `vomiting`, `irritability`, `skin`, `other` |
| Text | text | |
| Photos | image[] | Up to 4 per entry |

---

## 4. Technology Stack

| Layer | Choice | Rationale |
|-------|--------|-----------|
| **Backend** | Go (stdlib + minimal deps) | Developer comfort, fast, single binary |
| **Database** | SQLite on fly.io persistent volume | Simple, no external deps, plenty for personal use |
| **Frontend** | Svelte SPA (TypeScript) | Lightweight, great DX, small bundle, excellent reactivity |
| **Photo storage** | Cloudflare R2 | S3-compatible, zero egress, generous free tier |
| **Auth** | Google OAuth 2.0 | Both parents have Google accounts |
| **Hosting** | fly.io | Free/cheap tier, native Go support, persistent volumes, built-in TLS |
| **PWA** | Service worker + manifest | Installable on Android home screen, push notification support |
| **Charts** | Chart.js or Apache ECharts (via Svelte wrapper) | Lightweight, good for medical time-series |
| **PDF reports** | Server-side Go PDF generation (e.g., `go-pdf` or `maroto`) | Printable clinical summaries |

### 4.1 Repository Structure

```
littleliver/
├── backend/
│   ├── cmd/server/main.go          # Entrypoint
│   ├── internal/
│   │   ├── auth/                    # Google OAuth, sessions, middleware
│   │   ├── handler/                 # HTTP handlers (REST JSON API)
│   │   ├── model/                   # Domain types
│   │   ├── store/                   # SQLite repository layer
│   │   ├── storage/                 # R2/S3 photo upload
│   │   ├── notify/                  # Web Push (VAPID)
│   │   ├── report/                  # PDF generation
│   │   └── who/                     # WHO growth data + percentile calc
│   ├── migrations/                  # SQL migration files
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── routes/                  # Svelte pages
│   │   ├── components/              # Reusable UI components
│   │   ├── lib/                     # API client, stores, utils
│   │   └── service-worker.ts        # PWA + push notifications
│   ├── static/                      # Icons, manifest.json
│   ├── package.json
│   └── svelte.config.js
├── fly.toml
├── Dockerfile
└── README.md
```

---

## 5. API Design

RESTful JSON API. All endpoints require authentication (session cookie set after OAuth).

### 5.1 Auth
```
GET    /auth/google/login          → Redirect to Google OAuth
GET    /auth/google/callback       → Handle OAuth callback, set session
POST   /auth/logout                → Clear session
GET    /api/me                     → Current user info + linked babies
GET    /api/csrf-token             → Get per-session CSRF token (include as X-CSRF-Token header on state-changing requests)
```

### 5.2 Baby Profiles
```
POST   /api/babies                 → Create baby profile
GET    /api/babies                 → List my babies
GET    /api/babies/:id             → Get baby details
PUT    /api/babies/:id             → Update baby info (name, DOB, sex, diagnosis date, kasai date)
POST   /api/babies/:id/invite      → Generate invite code (any linked parent). Returns { "code": "483921", "expires_at": "2026-03-17T14:30:00Z" }. Fixed 24-hour expiration.
POST   /api/babies/join             → Join baby profile via invite code
DELETE /api/babies/:id/parents/me   → Unlink self from baby (last parent triggers baby + data deletion)
```

### 5.3 Metric Entries (pattern repeats for each metric type)
```
POST   /api/babies/:id/feedings              → Log feeding
GET    /api/babies/:id/feedings?from=&to=&cursor=  → List feedings in range (from/to are YYYY-MM-DD calendar dates)
PUT    /api/babies/:id/feedings/:entryId     → Edit entry
DELETE /api/babies/:id/feedings/:entryId     → Hard-delete entry
```

Metric endpoints: `/feedings`, `/urine`, `/stools`, `/weights`, `/abdomen`, `/temperatures`, `/skin`, `/bruising`, `/medications`, `/med-logs`, `/labs`, `/notes`

**Edit authorization:** Any linked parent can edit or delete any entry for that baby, regardless of who originally logged it (equal access). The `logged_by` field is immutable — it always reflects the original author. An `updated_by` field (nullable `TEXT REFERENCES users(id)`) is set to the editing user's ID on any update.

**Pagination:** All metric list endpoints use **cursor-based pagination**, sorted newest-first. Default **50 items per page**. All entity IDs are **ULIDs** (Universally Unique Lexicographically Sortable Identifiers), which encode creation time and are naturally sortable newest-first. This means `WHERE id < cursor` with `ORDER BY id DESC` gives correct newest-first pagination without a separate timestamp sort. The client passes `?cursor=<entryId>` for subsequent pages and treats the cursor as an opaque entry ID string. The response includes a `next_cursor` field (`null` if no more results).

**Deletes:** All metric entries are **hard-deleted** (no soft deletes). Medications are the exception — they can only be deactivated (`active=false`), never deleted, to preserve adherence history. Med-log entries support full `PUT` (edit) and `DELETE` (hard delete) — parents can correct mistakes freely, and adherence is calculated from current data only. Keep it simple.

**Date parameters:** `from` and `to` query parameters are **YYYY-MM-DD calendar date strings**. They filter against the entry's user-editable `timestamp` field. They are interpreted using the user's timezone (from `X-Timezone` header). Both bounds are inclusive — the range spans from 00:00:00 to 23:59:59 in the user's timezone. Note: date filtering uses the editable `timestamp`, while pagination order uses ULID (`WHERE id < cursor ORDER BY id DESC`). This means backdated entries may appear in a date range but at a different position than their chronological order would suggest. This minor inconsistency is accepted.

**Timezone:** Every API request must include an `X-Timezone` header with the user's IANA timezone (e.g., `America/New_York`). The backend persists this on the user record (`timezone` column), updating it on every API call so it stays current. The user's timezone is used for interpreting date parameters (`from`/`to`). Medication scheduled times are interpreted per the medication's own stored timezone (set at creation from the creator's `X-Timezone` header) — see §3.9 and §6.2. No timezone is stored on the baby profile.

### 5.4 Photos
```
POST   /api/babies/:id/upload      → Upload photo (baby-level auth check) → returns R2 key
```

**Photo upload flow:**
1. Client uploads the photo via `POST /api/babies/:id/upload`. The server stores the file in R2, creates a `photo_uploads` row, and returns the **R2 key** in the response.
2. Client includes the R2 key(s) in the metric entry creation or update request body, in the `photo_keys` JSON array field.
3. Server validates that each R2 key in `photo_keys` exists in the `photo_uploads` table with a matching `baby_id`. If valid, the server sets `linked_at` on the corresponding `photo_uploads` rows.

Photos are stored as a **JSON array in a single `TEXT` column** (`photo_keys`) on the relevant metric entry — no join table. **Photo unlink on edit:** When a metric entry is updated and a photo key is removed from `photo_keys`, the server sets `linked_at = NULL` on the corresponding `photo_uploads` row. No synchronous R2 deletion occurs during PUT requests — the orphan cleanup cron handles eventual deletion. **Orphan cleanup:** A cron job deletes `photo_uploads` rows where `linked_at` is null and `uploaded_at` is older than 24 hours, and garbage-collects the corresponding R2 objects.

### 5.5 Medications & Reminders

The medication resource includes both the drug definition and its schedule (no separate `/med-schedules` endpoint). Deactivate a medication by setting `active=false` via `PUT /api/babies/:id/medications/:id`.

```
POST   /api/babies/:id/medications           → Create medication (name, dose, frequency, schedule times)
GET    /api/babies/:id/medications            → List medications (active and inactive)
PUT    /api/babies/:id/medications/:id        → Update medication (including set active=false to deactivate). No delete endpoint — medications can only be deactivated, never deleted, to preserve adherence history.
POST   /api/babies/:id/med-logs              → Log a dose (given or skipped). `given_at` and `skipped` are mutually exclusive: when logging as "given", the server sets `given_at` to `NOW()` (current time, not `scheduled_time`); when logging as "skipped", `given_at` is null. `skip_reason` is optional even when `skipped=true`. Client passes `scheduled_time` (a full UTC datetime computed by the server — see §6.4) from the notification payload or the medication's schedule; nullable for ad-hoc doses not tied to a schedule.
GET    /api/babies/:id/med-logs?medication_id=&from=&to=&cursor=  → List med-logs, filterable by medication and date range
PUT    /api/babies/:id/med-logs/:entryId     → Edit a med-log entry
DELETE /api/babies/:id/med-logs/:entryId     → Hard-delete a med-log entry
POST   /api/push/subscribe                    → Register push subscription (per device)
DELETE /api/push/subscribe                    → Unregister
```

### 5.6 Reports
```
GET    /api/babies/:id/dashboard?from=&to=   → Dashboard data (aggregated JSON for charts). Response includes an `active_alerts` array containing entry IDs that trigger alerts (based on the most recent entries of each alert type). Frontend compares this with the local dismissed set and removes dismissed IDs for alerts that now have recovery entries.
GET    /api/babies/:id/report?from=&to=      → Generate + download clinical PDF (always includes all photos within date range)
```

---

## 6. Push Notifications (Medication Reminders)

### 6.1 Approach
- **Web Push API** with **VAPID** keys (generated server-side, stored in config).
- The Svelte PWA registers a push subscription on install and sends it to the backend.
- Each parent's device gets its own subscription — both parents receive reminders.

### 6.2 Reminder Logic
- The Go backend runs a **scheduler** (e.g., a goroutine with a ticker or a lightweight cron library).
- Every minute, it checks for medication schedules due within the next minute. Scheduled times are stored as local time strings and interpreted per the **medication's stored timezone** (the `timezone` column on the medication record, set at creation time from the creator's `X-Timezone` header). All parents are notified based on this single timezone, preventing dose drift and double-dosing when parents are in different timezones.
- Sends a push notification to all subscribed devices for that baby's parents.
- Notification includes: medication name, dose, and a "Log as given" action button (deep-links to the logging screen).

### 6.3 Notification Content
```
Title: "💊 UDCA — Time for dose"
Body:  "50mg for [Baby Name]. Tap to log."
Action: Opens app to medication logging screen with pre-filled medication.
```

### 6.4 Suppression & Follow-ups
- `scheduled_time` is a **full UTC datetime**, computed by the server from the medication's local schedule times + the medication's stored timezone at the moment the notification fires. Both `given_at` and `scheduled_time` are UTC datetimes, making the ±30 min suppression comparison straightforward.
- **Suppression check:** Before sending any notification (initial or follow-up), the server checks for any `med_log` for that `medication_id` (given OR skipped) within **±30 minutes** of the scheduled time being checked. The check uses `given_at` for given doses and `created_at` for skipped doses. This is a simple per-medication check — it does not need to match a specific `scheduled_time` field on the `med_log`. If found, the notification is suppressed.
- No pre-created `med_log` rows — rows are only created when the parent logs a dose (given or skipped). The client passes `scheduled_time` (from the notification payload or from the medication's schedule). `scheduled_time` is nullable for ad-hoc doses not tied to a schedule.
- **Follow-ups:** Follow-up notifications are re-derived each minute by the scheduler (no separate notification queue table). Follow-up #1 fires at **+15 min** after the scheduled time; follow-up #2 fires at **+30 min** after the scheduled time. Each follow-up re-runs the suppression check before sending.
- **Missed notifications:** If the server was down and a scheduled time + 15 min or + 30 min has already passed, the follow-up is simply skipped. No backfill of missed notifications.
- Max **2 follow-ups** per dose.

---

## 7. Dashboard (Parent-Facing)

The main screen parents see daily. Designed for quick data entry and at-a-glance status.

### 7.1 Today View
- **Summary cards** at top: total feeds today, total wet diapers, total stools (with color indicator), last temperature, last weight
- **Stool color trend** — last 7 days mini-chart with color-coded dots (red for acholic, green for pigmented)
- **Upcoming medications** — next due med with countdown
- **Quick-log buttons** — large tap targets for: Feed, Diaper (wet), Diaper (stool), Temp, Medication Given
- **Alert banners** — cholangitis warning (fever), acholic stool warning. Alerts are based on the **most recent entry of that type**, regardless of age — there is no lookback window or auto-expiry. **Temperature alerts** track which measurement method triggered the alert. Only a same-method sub-threshold reading clears it. If a different method is logged below that method's threshold, the fever alert persists (e.g., if a rectal 38.5°C triggers an alert, only a rectal sub-38.0°C reading clears it; an axillary 37.0°C does not clear it). The `active_alerts` response from the dashboard includes the alerting entry's `method` so the frontend can display appropriate guidance (e.g., "Take another rectal reading to confirm recovery"). Stool color rating 4+ clears acholic alerts. Alerts persist until a **recovery entry** is logged or **manually dismissed**. **Dismissal is per-user, stored as a set of dismissed entry IDs in client-side local storage** (not persisted in the database). When a recovery entry is logged, all entry IDs of that alert type are auto-removed from the dismissed set (effectively clearing stale alerts). New alarming entries add new IDs, creating new alerts regardless of prior dismissals. Other parents still see alerts independently. No additional DB table needed.

### 7.2 Trends View
Selectable date range (7d / 14d / 30d / 90d / custom). Charts for:
- **Stool color over time** — scatter plot, color-coded by stool color rating
- **Weight curve** — with WHO percentile bands (3rd, 15th, 50th, 85th, 97th) overlaid
- **Temperature** — line chart with fever threshold line
- **Abdomen girth** — line chart
- **Feeding volume / caloric intake** — daily aggregated bar chart (kcal computed per §3.1 formula; breast-direct feeds use configurable default estimate)
- **Diaper counts** — daily wet + stool counts
- **Lab trends** — multi-line chart (bilirubin, ALT, AST, GGT) with normal range shading

---

## 8. Clinical Report (Hepatologist-Facing)

### 8.1 Format
- Server-generated **PDF** (Go library, not browser print).
- Clean, professional layout. Header with baby name, DOB, age, diagnosis/Kasai dates.

### 8.2 Content
1. **Summary section** — age, days post-Kasai, current weight + percentile, current medications
2. **Stool color log** — table of entries + color trend chart for the report period
3. **Weight chart** — with WHO percentiles
4. **Lab trends** — chart + table of values
5. **Temperature log** — flagging any fever episodes
6. **Feeding summary** — average daily volume/calories
7. **Medication adherence** — adherence = (given logs / total logs including skipped) for all medication types. No inferred expected doses — just the ratio of logged-as-given vs total logged entries
8. **Notable observations** — any flagged notes, bruising entries, photos (thumbnails)
9. **Photo appendix** — all stool/skin photos within the report date range in chronological order

---

## 9. WHO Growth Standards Integration

### 9.1 Data Source
- WHO Child Growth Standards weight-for-age tables (0–24 months, sex-specific).
- Stored as embedded Go data (LMS values for percentile calculation).
- Source: [WHO Anthro](https://www.who.int/tools/child-growth-standards/standards/weight-for-age)

### 9.2 Calculation
- Given baby's sex, exact age in days, and weight → compute z-score and percentile.
- Plot on chart with standard percentile curves (3rd, 15th, 50th, 85th, 97th).

---

## 10. Data Model (SQLite Schema — Simplified)

```sql
-- Parents
CREATE TABLE users (
    id          TEXT PRIMARY KEY,  -- ULID
    google_id   TEXT UNIQUE NOT NULL,
    email       TEXT NOT NULL,
    name        TEXT NOT NULL,
    timezone    TEXT,                  -- IANA timezone (e.g., "America/New_York"), updated on every API call via X-Timezone header
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Babies
CREATE TABLE babies (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    sex             TEXT NOT NULL CHECK (sex IN ('male', 'female')),
    date_of_birth   DATE NOT NULL,
    diagnosis_date  DATE,
    kasai_date      DATE,
    default_cal_per_feed REAL DEFAULT 67,  -- default kcal estimate for breast-direct feeds without volume
    notes           TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Parent ↔ Baby link
CREATE TABLE baby_parents (
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE,
    user_id     TEXT REFERENCES users(id),
    role        TEXT DEFAULT 'parent',
    joined_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (baby_id, user_id)
);

-- Invite codes (one active code per baby; generating a new code hard-deletes ALL prior codes for that baby)
-- A cron job periodically deletes ALL codes older than 24 hours (both used and unused).
-- Codes are 6-digit numeric strings (e.g., "483921")
-- All failure cases (expired, used, invalidated, nonexistent, race condition) return a generic "invalid or expired code" error
CREATE TABLE invites (
    code        TEXT PRIMARY KEY,  -- 6-digit numeric string
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE,
    created_by  TEXT REFERENCES users(id),
    used_by     TEXT REFERENCES users(id),
    used_at     DATETIME,              -- set when code is redeemed
    expires_at  DATETIME NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Generic pattern for metric tables (feedings shown as example)
CREATE TABLE feedings (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by   TEXT REFERENCES users(id) NOT NULL,
    updated_by  TEXT REFERENCES users(id),  -- set on edit, null initially
    timestamp   DATETIME NOT NULL,
    feed_type   TEXT NOT NULL,
    volume_ml   REAL,
    cal_density REAL,
    calories    REAL,              -- denormalized: computed on insert/update from cal_density × volume_ml / 29.5735, or default_cal_per_feed for breast-direct
    used_default_cal BOOLEAN DEFAULT FALSE,  -- true when calories computed using baby's default_cal_per_feed
    duration_min INTEGER,
    notes       TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE stools (
    id              TEXT PRIMARY KEY,
    baby_id         TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by       TEXT REFERENCES users(id) NOT NULL,
    updated_by      TEXT REFERENCES users(id),  -- set on edit, null initially
    timestamp       DATETIME NOT NULL,
    color_rating    INTEGER NOT NULL CHECK (color_rating BETWEEN 1 AND 7),
    color_label     TEXT,
    consistency     TEXT,
    volume_estimate TEXT,
    photo_keys      TEXT,           -- JSON array of R2 object keys
    notes           TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Urine (each row = one wet diaper event)
CREATE TABLE urine (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by   TEXT REFERENCES users(id) NOT NULL,
    updated_by  TEXT REFERENCES users(id),  -- set on edit, null initially
    timestamp   DATETIME NOT NULL,
    color       TEXT,               -- clear, pale_yellow, dark_yellow, amber, brown
    notes       TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE weights (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by   TEXT REFERENCES users(id) NOT NULL,
    updated_by  TEXT REFERENCES users(id),
    timestamp   DATETIME NOT NULL,
    weight_kg   REAL NOT NULL,
    measurement_source TEXT,        -- home_scale, clinic
    notes       TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE temperatures (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by   TEXT REFERENCES users(id) NOT NULL,
    updated_by  TEXT REFERENCES users(id),
    timestamp   DATETIME NOT NULL,
    value       REAL NOT NULL,
    method      TEXT NOT NULL,      -- rectal, axillary, ear, forehead
    notes       TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE abdomen_observations (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by   TEXT REFERENCES users(id) NOT NULL,
    updated_by  TEXT REFERENCES users(id),
    timestamp   DATETIME NOT NULL,
    firmness    TEXT NOT NULL,       -- soft, firm, distended
    tenderness  BOOLEAN DEFAULT FALSE,
    girth_cm    REAL,
    photo_keys  TEXT,               -- JSON array of R2 object keys
    notes       TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE skin_observations (
    id              TEXT PRIMARY KEY,
    baby_id         TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by       TEXT REFERENCES users(id) NOT NULL,
    updated_by      TEXT REFERENCES users(id),
    timestamp       DATETIME NOT NULL,
    jaundice_level  TEXT,            -- none, mild, moderate, severe
    scleral_icterus BOOLEAN DEFAULT FALSE,
    rashes          TEXT,
    bruising        TEXT,
    photo_keys      TEXT,            -- JSON array of R2 object keys
    notes           TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE bruising (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by   TEXT REFERENCES users(id) NOT NULL,
    updated_by  TEXT REFERENCES users(id),
    timestamp   DATETIME NOT NULL,
    location    TEXT NOT NULL,
    size_estimate TEXT NOT NULL,     -- small_<1cm, medium_1-3cm, large_>3cm
    size_cm     REAL,               -- optional precise measurement
    color       TEXT,
    photo_keys  TEXT,               -- JSON array of R2 object keys
    notes       TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE lab_results (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by   TEXT REFERENCES users(id) NOT NULL,
    updated_by  TEXT REFERENCES users(id),
    timestamp   DATETIME NOT NULL,
    test_name   TEXT NOT NULL,
    value       TEXT NOT NULL,
    unit        TEXT,
    normal_range TEXT,
    notes       TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE general_notes (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by   TEXT REFERENCES users(id) NOT NULL,
    updated_by  TEXT REFERENCES users(id),
    timestamp   DATETIME NOT NULL,
    content     TEXT NOT NULL,
    photo_keys  TEXT,               -- JSON array of R2 object keys
    category    TEXT,               -- behavior, sleep, vomiting, irritability, skin, other
    notes       TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- photo_uploads.baby_id uses ON DELETE SET NULL (not CASCADE). A cleanup job
-- finds photo_uploads rows with NULL baby_id, deletes the corresponding R2
-- objects, then deletes the rows.

-- Medications (definitions / schedules)
CREATE TABLE medications (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by   TEXT REFERENCES users(id) NOT NULL,
    updated_by  TEXT REFERENCES users(id),  -- set on edit, null initially
    name        TEXT NOT NULL,
    dose        TEXT NOT NULL,
    frequency   TEXT NOT NULL,
    schedule    TEXT,              -- JSON array of times, e.g., ["08:00","20:00"]
    timezone    TEXT,              -- IANA timezone, set at creation from creator's X-Timezone header; mutable via PUT (e.g., if family moves); used for all notification scheduling
    active      BOOLEAN DEFAULT TRUE,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Medication administration log
CREATE TABLE med_logs (
    id              TEXT PRIMARY KEY,
    medication_id   TEXT REFERENCES medications(id) ON DELETE CASCADE NOT NULL,
    baby_id         TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by       TEXT REFERENCES users(id) NOT NULL,
    updated_by      TEXT REFERENCES users(id),  -- set on edit, null initially
    scheduled_time  DATETIME,          -- full UTC datetime, computed by server from local schedule + user timezone; nullable for ad-hoc doses
    given_at        DATETIME,          -- set to NOW() when logging as given; null when skipped=true
    skipped         BOOLEAN DEFAULT FALSE,  -- mutually exclusive with given_at: skipped=true → given_at is null; skipped=false → given_at is non-null
    skip_reason     TEXT,              -- optional even when skipped=true
    notes           TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Photo upload staging (for orphan cleanup)
CREATE TABLE photo_uploads (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) ON DELETE SET NULL,
    r2_key      TEXT NOT NULL,
    uploaded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    linked_at   DATETIME           -- set when a metric entry references this photo
);

-- Sessions (server-side, survives restarts)
CREATE TABLE sessions (
    id          TEXT PRIMARY KEY,  -- ULID
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT NOT NULL UNIQUE,
    expires_at  DATETIME NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
-- A cleanup cron deletes expired sessions periodically.

-- Push subscriptions (per device)
CREATE TABLE push_subscriptions (
    id          TEXT PRIMARY KEY,
    user_id     TEXT REFERENCES users(id) NOT NULL,
    endpoint    TEXT NOT NULL,
    p256dh      TEXT NOT NULL,
    auth        TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

Indexes on `(baby_id, timestamp)` for all metric tables.

---

## 11. Deployment (fly.io)

### 11.1 Architecture
- Single fly.io machine (personal use, no HA needed).
- **Persistent volume** mounted at `/data` for SQLite database file.
- Go binary serves both the API and the built Svelte static files.
- TLS handled by fly.io edge.

### 11.2 Dockerfile
```dockerfile
# Stage 1: Build frontend
FROM node:20-slim AS frontend
WORKDIR /app/frontend
COPY frontend/ .
RUN npm ci && npm run build

# Stage 2: Build backend
FROM golang:1.22 AS backend
WORKDIR /app
COPY backend/ ./backend/
WORKDIR /app/backend
RUN CGO_ENABLED=1 go build -o /server ./cmd/server

# Stage 3: Runtime
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=backend /server /server
COPY --from=frontend /app/frontend/build /static
EXPOSE 8080
CMD ["/server"]
```

### 11.3 Configuration (Environment Variables)
```
DATABASE_PATH=/data/littleliver.db
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
SESSION_SECRET=...
R2_ACCOUNT_ID=...
R2_ACCESS_KEY_ID=...
R2_SECRET_ACCESS_KEY=...
R2_BUCKET_NAME=littleliver-photos
R2_PUBLIC_URL=...           # If using custom domain or R2 public access
VAPID_PUBLIC_KEY=...
VAPID_PRIVATE_KEY=...
BASE_URL=https://littleliver.fly.dev
```

### 11.4 fly.toml (Minimal)
```toml
app = "littleliver"
primary_region = "iad"      # Choose closest region

[http_service]
  internal_port = 8080
  force_https = true

[mounts]
  source = "littleliver_data"
  destination = "/data"
```

### 11.5 Backup Strategy
- Daily automated backup of SQLite database via `fly ssh` + cron job or a simple script that copies the DB file to R2.
- SQLite `.backup` command for consistent snapshots.

---

## 12. Security Considerations

- **HTTPS only** — enforced by fly.io.
- **Session cookies** — HttpOnly, Secure, SameSite=Lax.
- **CSRF protection** — `GET /api/csrf-token` returns a per-session token. Client includes it as an `X-CSRF-Token` header on all state-changing requests. Server validates per-session.
- **Photo access** — R2 objects are private. Backend generates **signed URLs** (time-limited) for photo access. No public bucket access.
- **Input validation** — all inputs validated server-side. Parameterized SQL queries (no injection).
- **Rate limiting** — basic rate limiting on API endpoints (personal use, but good hygiene).
- **Invite codes** — 6-digit numeric strings (e.g., `"483921"`). Single-use, fixed **24-hour expiration**. Only one active (unused, unexpired) code per baby at a time; generating a new code hard-deletes ALL prior codes for that baby (used, expired, or unused). The `POST /api/babies/:id/invite` response includes both the `code` and the `expires_at` timestamp. A cron job periodically deletes ALL codes older than 24 hours (both used and unused). The server checks `used_at IS NOT NULL` as a rejection condition but returns the same generic `"invalid or expired code"` error for all failure cases (expired, used, invalidated, nonexistent, race condition).

---

## 13. Stool Color Card Reference

The app should display a visual reference when logging stools:

| Rating | Color | Label | Clinical Meaning |
|--------|-------|-------|------------------|
| 1 | ⬜ White | `white` | Acholic — NO bile flow — **ALERT** |
| 2 | 🟡 Pale clay | `clay` | Acholic — minimal bile — **ALERT** |
| 3 | 🟡 Pale yellow | `pale_yellow` | Questionable — **ALERT** |
| 4 | 🟢 Yellow | `yellow` | Some bile present — monitor closely |
| 5 | 🟢 Light green | `light_green` | Good bile flow |
| 6 | 🟢 Green | `green` | Good bile flow |
| 7 | 🟤 Brown | `brown` | Normal bile flow |

The stool logging screen should show these colors as large tappable swatches with the reference images.

---

## 14. Development Phases

### Phase 1 — Core (Weeks 1–2)
- Project scaffolding (Go backend, Svelte frontend, Docker, fly.toml)
- Google OAuth + session management
- Baby profile CRUD + invite system
- Stool logging (with color card + photo upload to R2)
- Feeding logging
- Temperature logging
- Basic "today" dashboard

### Phase 2 — Complete Tracking (Weeks 3–4)
- All remaining metric types (urine, weight, abdomen, skin, bruising, labs, notes)
- Medication schedule management
- Medication logging
- Alert banners (fever, acholic stool)

### Phase 3 — Notifications + Charts (Weeks 5–6)
- PWA service worker + manifest
- Web Push notification registration
- Medication reminder scheduler
- Chart.js/ECharts integration for all trend views
- WHO growth percentile overlay

### Phase 4 — Reports + Polish (Weeks 7–8)
- PDF report generation (Go server-side)
- Report customization (date range selection)
- UI polish, loading states, offline resilience
- SQLite backup automation
- Deploy to fly.io production

---

## 15. Future Considerations (v2+)

- **Offline-first** — full offline data entry with sync when reconnected
- **Photo ML** — automatic stool color classification from photos (stretch goal)
- **Export** — CSV/JSON full data export for records
- **Multi-language** — if sharing with family/caregivers who speak another language
- **Doctor view** — read-only link for hepatologist to see live dashboard (no login required, token-based)
- **Integration** — direct upload of lab results from hospital portals (very stretch)

---

## 16. Naming

App name: **LittleLiver**. The name should be:
- Easy to remember
- Not alarming if seen on a phone's home screen
- Meaningful to the parents

---

*This spec is a living document. Last updated: March 2026.*
