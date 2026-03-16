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
- Any linked parent can generate invite codes. All invite codes have a **fixed 24-hour expiration**. Generating a new invite code **hard-deletes ALL prior codes** for that baby (used, expired, or unused) — only one active invite code per baby at a time. On code collision (uniqueness violation), retry with a new random code up to **5 times**. A cron job periodically deletes ALL invite codes older than 24 hours (both used and unused) across all babies, keeping active code count low and collision risk negligible. The server checks `used_at IS NOT NULL` as a rejection condition but returns the same generic "invalid or expired code" error for all failure cases.
- If an already-linked parent redeems an invite code for a baby they are already linked to, show a friendly "You're already linked to this baby" message (no error).
- **Self-unlink:** A parent can unlink themselves from a baby (but not other parents) via `DELETE /api/babies/:id/parents/me`. If the last remaining parent unlinks, the baby and all associated data are deleted. The endpoint always returns **204 No Content** regardless of whether the baby was also deleted. The frontend detects baby deletion by attempting to fetch baby data and receiving 404, then navigates to the baby list/creation screen.
- **Account deletion:** `DELETE /api/users/me` deletes the requesting user's account. Deletion order: (1) identify babies where the user is the last remaining parent; (2) delete those babies (triggering `ON DELETE CASCADE` for all associated data); (3) anonymize `logged_by`/`updated_by` to `'deleted_user'` across all metric tables (anonymization, not CASCADE); (4) delete the user record (`ON DELETE CASCADE` cleans up remaining `baby_parents` and `sessions`). Returns **204 No Content**.
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
| Volume (mL) | number | NULL for breast-direct feeds (see caloric calculation below). If a parent forgot to enter volume for pumped milk, they can edit the entry later to add it. |
| Caloric density (kcal/oz) | number | Optional. When omitted, standard defaults apply: ~20 kcal/oz for formula and breast milk. User can override per entry by providing an explicit value. Relevant for fortified feeds (e.g., 24 or 30 cal/oz). |
| Duration (min) | number | For breastfeeding sessions |
| Notes | text | Free-form (e.g., "tolerated well", "vomited after") |

**Caloric intake calculation:** All feed types (including `solid` and `other`) can optionally specify `cal_density` and `volume_ml`. When both are provided, calories are calculated using the standard formula: `kcal = volume_ml × (cal_density / 29.5735)` (where `1 oz = 29.5735 mL`). **Cal density auto-apply:** When `volume_ml` is provided but `cal_density` is omitted, the backend auto-applies a default of **~20 kcal/oz** for `breast_milk` and `formula` feed types. This is a type-based default — `used_default_cal` is NOT set for these entries. The type-based 20 kcal/oz value is baked into the `calories` column at insert time; if a parent needs to correct it, they edit the individual entry's `cal_density` field. No extra flag or batch-recalculation mechanism exists for type-based defaults. If neither `cal_density` nor the type-based default applies, and volume is missing, caloric intake is left null for that entry. **Breast-direct feed detection:** A feeding is considered "breast-direct" when `feed_type = 'breast_milk' AND volume_ml IS NULL`. No additional field is needed. For breast-direct feeds, a configurable default estimate is used: **~67 kcal per session** (based on an average ~100 mL intake at 20 kcal/oz: `100 × 20 / 29.5735 ≈ 67.6 kcal`). This default is stored as `default_cal_per_feed` on the baby profile and can be adjusted via `PUT /api/babies/:id`. Only breast-direct feeds with no volume use `default_cal_per_feed` and have `used_default_cal=true`.

**Denormalized `calories` column:** The computed caloric value is stored as a `calories REAL` column on the `feedings` table. This value is computed and stored on insert/update using the formula above (or the baby's `default_cal_per_feed` for breast-direct feeds without volume). A `used_default_cal BOOLEAN DEFAULT false` column tracks whether the feeding's calories were computed using the baby's `default_cal_per_feed`. When `default_cal_per_feed` is changed on the baby via `PUT /api/babies/:id`, the parent can trigger recalculation of all affected entries by including `recalculate_calories=true` as a query parameter (or body field). When set, the server recalculates all feeding entries where `used_default_cal = true` using the new default value, within the same request. The response includes `{ "recalculated_count": N }` with the number of updated entries.

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
| Timestamp | datetime | Auto-filled, editable (full datetime like all other metrics) |
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
| Scheduled times | time[] | e.g., [08:00, 20:00] for twice daily. Stored as **local time strings** (not UTC). Interpreted per the medication's stored timezone (see §6.2). `custom` = arbitrary user-defined list of daily times (functionally the same as other frequencies but with user-chosen times). `as_needed` = null/empty schedule array, no push notifications sent. |
| Timezone | text | IANA timezone (e.g., `America/New_York`), set at creation time from the creator's `X-Timezone` header. All notification scheduling uses this timezone, not the individual user's timezone. This prevents dose drift and double-dosing across timezone boundaries. |
| Given at | datetime | Set to `NOW()` when parent taps "given" (not `scheduled_time`). Null when skipped. |
| Skipped | boolean | Mutually exclusive with `given_at`: `skipped=true` → `given_at` is null; `skipped=false` → `given_at` is non-null. |
| Skip reason | text | Optional even when `skipped=true`. |
| Notes | text | e.g., "spit up half the dose" |

**Reminder system:** see §6 (Push Notifications).

### 3.10 Lab Results (per clinic visit, entered manually)

Stored as individual test entries using an EAV-style table (`test_name`, `value`, `unit`, `normal_range`, `notes`). Each row is one test result. Lab entries from the same visit share the exact same timestamp for implicit grouping (no explicit visit_id). The schema is generic to support any lab test.

The **UI** suggests common Kasai-relevant tests as quick-pick options: `total_bilirubin`, `direct_bilirubin`, `ALT`, `AST`, `GGT`, `albumin`, `INR`, `platelets`. Selecting a quick-pick pre-fills the `test_name` and `unit` fields. Parents can also enter arbitrary test names.

| Field | Type | Notes |
|-------|------|-------|
| Timestamp | datetime | Full datetime like all other metric types |
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
| Content | text | Free-form text (required) |
| Category | enum | `behavior`, `sleep`, `vomiting`, `irritability`, `skin`, `other` |
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
DELETE /api/users/me               → Delete own account (see §2.2 for cascade behavior)
```

### 5.2 Baby Profiles
```
POST   /api/babies                 → Create baby profile
GET    /api/babies                 → List my babies
GET    /api/babies/:id             → Get baby details
PUT    /api/babies/:id             → Update baby info (name, DOB, sex, diagnosis date, kasai date). Supports `?recalculate_calories=true` — see §3.1.
POST   /api/babies/:id/invite      → Generate invite code (any linked parent). Returns { "code": "483921", "expires_at": "2026-03-17T14:30:00Z" }. Fixed 24-hour expiration.
POST   /api/babies/join             → Join baby profile via invite code
DELETE /api/babies/:id/parents/me   → Unlink self from baby (last parent triggers baby + data deletion). Always returns 204.
```

### 5.3 Metric Entries (pattern repeats for each metric type)
```
POST   /api/babies/:id/feedings              → Log feeding
GET    /api/babies/:id/feedings?from=&to=&cursor=  → List feedings in range (from/to are YYYY-MM-DD calendar dates)
PUT    /api/babies/:id/feedings/:entryId     → Edit entry
DELETE /api/babies/:id/feedings/:entryId     → Hard-delete entry
```

Metric endpoints: `/feedings`, `/urine`, `/stools`, `/weights`, `/abdomen`, `/temperatures`, `/skin`, `/bruising`, `/medications`, `/med-logs`, `/labs`, `/notes`

**Photo signed URLs:** For metric types that support photos (see §5.4), the `photo_keys` array in API responses contains signed URLs (TTL: 1 hour) rather than raw R2 keys.

**Edit authorization:** Any linked parent can edit or delete any entry for that baby, regardless of who originally logged it (equal access). The `logged_by` field is immutable — it always reflects the original author. An `updated_by` field (nullable `TEXT REFERENCES users(id)`) is set to the editing user's ID on any update.

**Pagination & sort order:** All metric list endpoints use **cursor-based pagination**. Default **50 items per page**. The API returns entries in **ULID order** per cursor (`WHERE id < cursor ORDER BY id DESC`). The client passes `?cursor=<entryId>` for subsequent pages and treats the cursor as an opaque entry ID string. The response includes a `next_cursor` field (`null` if no more results). All entity IDs are **ULIDs** (Universally Unique Lexicographically Sortable Identifiers). **Frontend sort responsibility:** The frontend re-sorts each page by `timestamp DESC` for display. Cross-page gaps for backdated entries are acceptable — a backdated entry may not appear in its chronological position relative to entries on other pages. **Combined date + cursor:** When both `from`/`to` and `cursor` are provided, both conditions apply as an AND: `WHERE timestamp BETWEEN from AND to AND id < cursor ORDER BY id DESC LIMIT 50`. The date range narrows the result set; the cursor paginates within it.

**Deletes:** All metric entries are **hard-deleted** (no soft deletes). Medications are the exception — they can only be deactivated (`active=false`), never deleted, to preserve adherence history. Med-log entries support full `PUT` (edit) and `DELETE` (hard delete) — parents can correct mistakes freely, and adherence is calculated from current data only. Keep it simple.

**Date parameters:** `from` and `to` query parameters are **YYYY-MM-DD calendar date strings**. They filter against the entry's user-editable `timestamp` field. They are interpreted using the user's timezone (from `X-Timezone` header). Both bounds are inclusive — the range spans from 00:00:00 to 23:59:59 in the user's timezone. Note: date filtering uses the editable `timestamp`, while pagination order uses ULID (`WHERE id < cursor ORDER BY id DESC`). This means backdated entries may appear in a date range but at a different position than their chronological order would suggest. Additionally, backdated entries may split same-timestamp entries across page boundaries since the ULID cursor is creation-order, not timestamp-order. Both quirks are acceptable for a personal-use app — the ULID-based cursor is kept as-is.

**Timezone:** Every API request must include an `X-Timezone` header with the user's IANA timezone (e.g., `America/New_York`). The backend persists this on the user record (`timezone` column), updating it on every API call so it stays current. The user's timezone is used for interpreting date parameters (`from`/`to`). Medication scheduled times are interpreted per the medication's own stored timezone (set at creation from the creator's `X-Timezone` header) — see §3.9 and §6.2. No timezone is stored on the baby profile.

### 5.4 Photos
```
POST   /api/babies/:id/upload      → Upload photo (baby-level auth check) → returns R2 key
```

**Photo upload constraints:** Max file size: **5 MB**. Accepted MIME types: **JPEG, PNG, HEIC**. The server generates a thumbnail (~300px wide) stored alongside the original in R2 with a `thumb_` prefix (e.g., original key `photos/abc123.jpg` → thumbnail key `photos/thumb_abc123.jpg`). Thumbnails are used in dashboard display and PDF report embedding. The `photo_uploads` table includes a `thumbnail_key TEXT` column to store the thumbnail's R2 key.

**Photo upload flow:**
1. Client uploads the photo via `POST /api/babies/:id/upload`. The server validates size/type, stores the original and generated thumbnail in R2, creates a `photo_uploads` row (with both `r2_key` and `thumbnail_key`), and returns the **R2 key** in the response.
2. Client includes the R2 key(s) in the metric entry creation or update request body, in the `photo_keys` JSON array field.
3. Server validates that each R2 key in `photo_keys` exists in the `photo_uploads` table with a matching `baby_id`. If valid, the server sets `linked_at` on the corresponding `photo_uploads` rows.

**Photo support scope:** Only the following metric types support photos (`photo_keys` column): `stools`, `abdomen_observations`, `skin_observations`, `bruising`, and `general_notes`. Photos are explicitly NOT supported for: `feedings`, `weights`, `temperatures`, `urine`, `lab_results`, `med_logs`.

**Signed URLs on read:** When metric entries containing `photo_keys` are returned by the API (list or detail), the server replaces each R2 key with a **signed URL** (TTL: 1 hour). No separate photo URL endpoint is needed — clients always receive ready-to-use URLs.

Photos are stored as a **JSON array in a single `TEXT` column** (`photo_keys`) on the relevant metric entry — no join table. **Photo unlink on edit:** When a metric entry is updated and a photo key is removed from `photo_keys`, the server sets `linked_at = NULL` on the corresponding `photo_uploads` row. No synchronous R2 deletion occurs during PUT requests — the cleanup cron handles eventual deletion. **Photo cleanup (single cron job):** One combined cron job handles both orphan and cascade cleanup. It deletes `photo_uploads` rows (and their R2 objects) matching: `(linked_at IS NULL AND uploaded_at < NOW() - 24h) OR (baby_id IS NULL)`. The first condition catches unlinked/abandoned uploads; the second catches rows orphaned by baby deletion (`ON DELETE SET NULL`). One job, one schedule.

### 5.5 Medications & Reminders

The medication resource includes both the drug definition and its schedule (no separate `/med-schedules` endpoint). Deactivate a medication by setting `active=false` via `PUT /api/babies/:id/medications/:id`. When a medication is deactivated, the scheduler skips it on its next tick — no further notifications are sent for that medication. Deactivated medications are also excluded from the `upcoming_meds` section of the dashboard response.

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
GET    /api/babies/:id/dashboard?from=&to=   → Dashboard data (aggregated JSON for charts). When `from`/`to` are omitted, defaults to today. Trends view uses the same endpoint with different date ranges (e.g., `?from=2026-03-09&to=2026-03-16` for 7-day view). All aggregation is server-side. The response always returns the same structure regardless of date range — the frontend picks what to display based on context (today view vs trends view).

**Dashboard response schema:**
```json
{
  "summary_cards": {
    "total_feeds": 0,
    "total_calories": 0,
    "total_wet_diapers": 0,
    "total_stools": 0,
    "worst_stool_color": null,
    "last_temperature": null,
    "last_weight": null
  },
  "stool_color_trend": [],         // always last 7 days, regardless of from/to — frontend uses this for the Today View mini-chart
  "upcoming_meds": [],             // next due ACTIVE medications with countdown (deactivated medications excluded)
  "active_alerts": [],             // array of alert objects: { entry_id, alert_type, method?, value, timestamp } — always computed from the globally most recent entry of each alert type across ALL time, ignoring the from/to date range parameters (alerts are global state). See §7.1 for alert types and trigger conditions.
  "chart_data_series": {           // for the requested date range
    "feeding_daily": [],           // [{ date, total_volume_ml, total_calories, feed_count, by_type: { breast_milk, formula, solid, other } }] — daily aggregates
    "diaper_daily": [],            // [{ date, wet_count, stool_count }] — daily aggregates
    "temperature": [],             // [{ timestamp, value, method }] — individual readings, not aggregated
    "weight": [],                  // [{ timestamp, weight_kg, measurement_source }] — individual readings
    "abdomen_girth": [],           // [{ timestamp, girth_cm }] — individual readings
    "stool_color": [],             // [{ timestamp, color_score }] — individual readings for requested date range; frontend uses this (not stool_color_trend) for the Trends View
    "lab_trends": {}               // { [test_name]: [{ timestamp, test_name, value, unit }] } — individual results, grouped by test_name on frontend
  }
}
```
Frontend compares `active_alerts` with the local dismissed set and removes dismissed IDs for alerts that now have recovery entries.
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
- Every minute, the scheduler queries only **active** medications (`active=true`). It computes "today" **relative to each medication's own timezone** (the `timezone` column on the medication record, set at creation time from the creator's `X-Timezone` header). It looks **forward** (next minute) for initial notifications and **backward** (up to 30 minutes) for follow-ups. This means at 23:50 UTC, a medication in UTC+2 checks against the next calendar day's schedule in that timezone. Scheduled times are stored as local time strings and interpreted per the medication's timezone. All parents are notified based on this single timezone, preventing dose drift and double-dosing when parents are in different timezones.
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
- **Suppression check:** Before sending any notification (initial, +15 min follow-up, or +30 min follow-up), the server checks for any `med_log` for that `medication_id` (given OR skipped) within **±30 minutes of the original scheduled time** — not ±30 min of the follow-up firing time. The check is identical regardless of which notification tier is being evaluated. The check uses `given_at` for given doses and `created_at` for skipped doses. This is a simple per-medication check — it does not need to match a specific `scheduled_time` field on the `med_log`. If found, the notification is suppressed. **Edge case — edited `given_at`:** If a `med_log` entry is edited such that `given_at` moves outside the ±30 min suppression window, a late follow-up may fire. This is an acceptable edge case — the server uses the current state of `med_log` entries as-is with no special handling.
- No pre-created `med_log` rows — rows are only created when the parent logs a dose (given or skipped). The client passes `scheduled_time` (from the notification payload or from the medication's schedule). `scheduled_time` is nullable for ad-hoc doses not tied to a schedule.
- **Follow-ups:** Follow-up notifications are re-derived each minute by the scheduler (no separate notification queue table). Follow-up #1 fires at **+15 min** after the scheduled time; follow-up #2 fires at **+30 min** after the scheduled time. Each follow-up re-runs the suppression check before sending. **Suppression is re-evaluated each minute** — the scheduler is fully stateless with no tracking table. On each tick, it checks whether a qualifying `med_log` exists now; if so, the notification is suppressed. If a `med_log` is deleted after a notification was previously suppressed, a late follow-up may fire on the next tick. This edge case is acceptable — simplicity over perfect suppression tracking.
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
- **Alert banners** — each alert is an object: `{ entry_id, alert_type, method?, value, timestamp }`. Alert types and trigger conditions:
  - `acholic_stool` — stool entry with `color_rating <= 3`. Cleared when most recent stool has `color_rating >= 4`.
  - `fever` — temperature entry exceeding threshold for its method (see §3.6). Cleared when most recent temperature is sub-threshold.
  - `jaundice_worsening` — skin observation with `jaundice_level = 'severe_limbs_and_trunk'` OR `scleral_icterus = true`. Cleared when most recent skin observation has neither condition.
  - `missed_medication` — checks all scheduled doses in the **last 24 hours** (expanding each active medication's `schedule` array into concrete UTC datetimes using the medication's stored timezone) that are **>30 min past due** with no corresponding `med_log` (given or skipped) within ±30 min of the scheduled time. One alert per missed dose. Cleared when a `med_log` is logged for that scheduled time.
  Alerts are based on the **most recent entry of that type across all time**, regardless of age — there is no lookback window or auto-expiry. `active_alerts` is always computed globally, ignoring any `from`/`to` date range on the dashboard request. **Temperature alerts:** Only one temperature alert exists at a time, based on the **single most recent temperature entry** regardless of method. If that entry exceeds the threshold for its method, the alert fires. If the most recent entry is sub-threshold for its own method, there is no alert — regardless of prior readings by other methods. Since only the most recent entry matters, "recovery" simply means the newest temperature entry is sub-threshold for its own method. The `active_alerts` response from the dashboard includes the alerting entry's `method` so the frontend can display appropriate guidance (e.g., "Take another reading to confirm recovery"). Stool color rating 4+ clears acholic alerts. Alerts persist until a **recovery entry** is logged or **manually dismissed**. **Dismissal is per-user, stored as a set of dismissed entry IDs in client-side local storage** (not persisted in the database). When a recovery entry is logged, all entry IDs of that alert type are auto-removed from the dismissed set (effectively clearing stale alerts). New alarming entries add new IDs, creating new alerts regardless of prior dismissals. Other parents still see alerts independently. No additional DB table needed.

### 7.2 Trends View
Uses the same `GET /api/babies/:id/dashboard?from=&to=` endpoint with the desired date range. Selectable date range (7d / 14d / 30d / 90d / custom). Charts for:
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

-- Sentinel user for FK integrity after account deletion.
-- Pre-seeded at database initialization. logged_by/updated_by are set to this value
-- when a user account is deleted (see §2.2 account deletion ordering).
INSERT INTO users (id, google_id, email, name) VALUES ('deleted_user', '__sentinel__', '', 'Deleted User');

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
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- photo_uploads.baby_id uses ON DELETE SET NULL (not CASCADE). A single cleanup
-- cron job handles both orphan and cascade cleanup — see §5.4.

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
-- baby_id is intentionally denormalized (also available via medications.baby_id) for query
-- convenience. The server validates that med_logs.baby_id matches medications.baby_id on
-- insert and update.
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
    id              TEXT PRIMARY KEY,
    baby_id         TEXT REFERENCES babies(id) ON DELETE SET NULL,
    r2_key          TEXT NOT NULL,
    thumbnail_key   TEXT,              -- R2 key for ~300px wide thumbnail (e.g., "photos/thumb_abc123.jpg")
    uploaded_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
    linked_at       DATETIME           -- set when a metric entry references this photo
);

-- Sessions (server-side, survives restarts)
CREATE TABLE sessions (
    id          TEXT PRIMARY KEY,  -- ULID; this is the session cookie value
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT NOT NULL UNIQUE,  -- random secret used only for CSRF token derivation via HMAC-SHA256; NOT the cookie value
    expires_at  DATETIME NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
-- Sessions last 30 days with a sliding window: expires_at is reset on each API call.
-- No absolute maximum session lifetime — the 30-day sliding window is the sole expiry policy.
-- Expired sessions return HTTP 401. A cleanup cron deletes expired sessions periodically.

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

**Account deletion & foreign keys:** The `logged_by` and `updated_by` columns across all metric tables do NOT use `ON DELETE CASCADE`. Instead, the `DELETE /api/users/me` handler explicitly sets these to `'deleted_user'` before removing the user record. The `baby_parents` and `sessions` tables use `ON DELETE CASCADE` on `user_id` so they are cleaned up automatically.

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
- **Session cookies** — HttpOnly, Secure, SameSite=Lax. The cookie value is the `sessions.id` (ULID). The server looks up the session by this ID. Sessions last **30 days** with a **sliding window** — `expires_at` is reset on each API call. No absolute maximum session lifetime; the sliding window is the sole expiry policy. Expired sessions return **HTTP 401**. Frontend redirects to login on 401.
- **CSRF protection** — `GET /api/csrf-token` returns a per-session CSRF token **derived deterministically from the session's `token` column via HMAC-SHA256** with a server secret. The `token` column is a separate random secret used only for CSRF derivation — it is not the session cookie value. No extra storage column needed beyond what exists — the CSRF token is computed on the fly and is stable for the session lifetime. Client includes it as an `X-CSRF-Token` header on all state-changing requests. Server validates by re-deriving the expected CSRF token from the current session's `token` value and comparing.
- **Photo access** — R2 objects are private. Backend generates **signed URLs** (time-limited) for photo access. No public bucket access.
- **Input validation** — all inputs validated server-side. Parameterized SQL queries (no injection).
- **Rate limiting** — basic rate limiting on API endpoints (personal use, but good hygiene).
- **Invite codes** — 6-digit numeric strings (e.g., `"483921"`). Single-use, fixed **24-hour expiration**. Only one active (unused, unexpired) code per baby at a time; generating a new code hard-deletes ALL prior codes for that baby (used, expired, or unused). On uniqueness violation (code collision), retry with a new random code up to **5 times** before returning an error. With cron cleanup keeping active code count low, collision risk is negligible. The `POST /api/babies/:id/invite` response includes both the `code` and the `expires_at` timestamp. A cron job periodically deletes ALL codes older than 24 hours (both used and unused). The server checks `used_at IS NOT NULL` as a rejection condition but returns the same generic `"invalid or expired code"` error for all failure cases (expired, used, invalidated, nonexistent, race condition).

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
