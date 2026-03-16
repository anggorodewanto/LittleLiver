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
- A baby profile has one or more **authorized parents** (Google account IDs).
- All authorized parents have equal read/write access to all data for that baby.
- Invite flow: Parent A creates baby → receives a single-use invite code → Parent B enters code on first login → linked.

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
| Caloric density (kcal/oz) | number | Relevant for fortified feeds (e.g., 24 or 30 cal/oz) |
| Duration (min) | number | For breastfeeding sessions |
| Notes | text | Free-form (e.g., "tolerated well", "vomited after") |

### 3.2 Urine Output (multiple entries per day)

| Field | Type | Notes |
|-------|------|-------|
| Timestamp | datetime | Auto-filled, editable |
| Wet diaper | boolean | Simple yes/count tracker |
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

Weight is plotted against **WHO Child Growth Standards** (weight-for-age percentiles, sex-specific).

### 3.5 Abdomen Circumference (1–2x/day)

| Field | Type | Notes |
|-------|------|-------|
| Timestamp | datetime | |
| Circumference (cm) | number | To 1 decimal place |
| Photo | image | Optional — for visual distension tracking |
| Notes | text | |

Increasing abdominal girth can indicate ascites or organomegaly — trend matters more than absolute number.

### 3.6 Temperature (multiple per day)

| Field | Type | Notes |
|-------|------|-------|
| Timestamp | datetime | |
| Temperature (°C) | number | To 1 decimal place |
| Method | enum | `rectal`, `axillary`, `ear`, `forehead` |
| Notes | text | |

**Alert logic:** If temperature ≥ 38.0°C (rectal) or ≥ 37.5°C (axillary), display a **cholangitis warning** banner: *"Fever detected. Contact your hepatology team immediately. Fever after Kasai can indicate cholangitis."*

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
| Size estimate | enum | `small_<1cm`, `medium_1-3cm`, `large_>3cm` |
| Photo | image | |
| Notes | text | |

New or worsening bruising can indicate vitamin K deficiency / coagulopathy.

### 3.9 Medications (log + reminders)

| Field | Type | Notes |
|-------|------|-------|
| Medication name | text | Pre-populated suggestions: `UDCA (ursodiol)`, `Sulfamethoxazole-Trimethoprim (Bactrim)`, `Vitamin A`, `Vitamin D`, `Vitamin E (TPGS)`, `Vitamin K`, `Iron`, `Other` |
| Dose | text | e.g., "50mg", "0.5mL" |
| Frequency | enum | `once_daily`, `twice_daily`, `three_times_daily`, `as_needed`, `custom` |
| Scheduled times | time[] | e.g., [08:00, 20:00] for twice daily |
| Given at | datetime | Logged when parent taps "given" |
| Skipped | boolean | With required reason text |
| Notes | text | e.g., "spit up half the dose" |

**Reminder system:** see §6 (Push Notifications).

### 3.10 Lab Results (per clinic visit, entered manually)

| Field | Type | Notes |
|-------|------|-------|
| Date | date | |
| Total bilirubin (mg/dL) | number | The key prognostic marker — goal is < 2.0 by 3 months post-Kasai |
| Direct bilirubin (mg/dL) | number | |
| ALT (U/L) | number | |
| AST (U/L) | number | |
| GGT (U/L) | number | |
| Albumin (g/dL) | number | |
| INR | number | Coagulation — elevated = concern |
| Platelets (×10³/µL) | number | Low = possible portal hypertension |
| Notes | text | |

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
```

### 5.2 Baby Profiles
```
POST   /api/babies                 → Create baby profile
GET    /api/babies                 → List my babies
GET    /api/babies/:id             → Get baby details
PUT    /api/babies/:id             → Update baby info (name, DOB, sex, diagnosis date, kasai date)
POST   /api/babies/:id/invite      → Generate invite code
POST   /api/babies/join             → Join baby profile via invite code
```

### 5.3 Metric Entries (pattern repeats for each metric type)
```
POST   /api/babies/:id/feedings              → Log feeding
GET    /api/babies/:id/feedings?from=&to=    → List feedings in range
PUT    /api/babies/:id/feedings/:entryId     → Edit entry
DELETE /api/babies/:id/feedings/:entryId     → Delete entry
```

Metric endpoints: `/feedings`, `/urine`, `/stools`, `/weights`, `/abdomen`, `/temperatures`, `/skin`, `/bruising`, `/medications`, `/med-logs`, `/labs`, `/notes`

### 5.4 Photos
```
POST   /api/upload                 → Upload photo → returns R2 URL
```
Photos are uploaded separately and their R2 key is stored in the relevant metric entry.

### 5.5 Medication Reminders
```
GET    /api/babies/:id/med-schedules         → List medication schedules
POST   /api/babies/:id/med-schedules         → Create/update schedule
DELETE /api/babies/:id/med-schedules/:sid     → Remove schedule
POST   /api/push/subscribe                    → Register push subscription (per device)
DELETE /api/push/subscribe                    → Unregister
```

### 5.6 Reports
```
GET    /api/babies/:id/dashboard?from=&to=   → Dashboard data (aggregated JSON for charts)
GET    /api/babies/:id/report?from=&to=      → Generate + download clinical PDF
```

---

## 6. Push Notifications (Medication Reminders)

### 6.1 Approach
- **Web Push API** with **VAPID** keys (generated server-side, stored in config).
- The Svelte PWA registers a push subscription on install and sends it to the backend.
- Each parent's device gets its own subscription — both parents receive reminders.

### 6.2 Reminder Logic
- The Go backend runs a **scheduler** (e.g., a goroutine with a ticker or a lightweight cron library).
- Every minute, it checks for medication schedules due within the next minute.
- Sends a push notification to all subscribed devices for that baby's parents.
- Notification includes: medication name, dose, and a "Log as given" action button (deep-links to the logging screen).

### 6.3 Notification Content
```
Title: "💊 UDCA — Time for dose"
Body:  "50mg for [Baby Name]. Tap to log."
Action: Opens app to medication logging screen with pre-filled medication.
```

### 6.4 Snooze / Acknowledgment
- If not logged within 15 minutes of scheduled time, send a follow-up reminder.
- Max 2 follow-ups per dose.

---

## 7. Dashboard (Parent-Facing)

The main screen parents see daily. Designed for quick data entry and at-a-glance status.

### 7.1 Today View
- **Summary cards** at top: total feeds today, total wet diapers, total stools (with color indicator), last temperature, last weight
- **Stool color trend** — last 7 days mini-chart with color-coded dots (red for acholic, green for pigmented)
- **Upcoming medications** — next due med with countdown
- **Quick-log buttons** — large tap targets for: Feed, Diaper (wet), Diaper (stool), Temp, Medication Given
- **Alert banners** — cholangitis warning (fever), acholic stool warning

### 7.2 Trends View
Selectable date range (7d / 14d / 30d / 90d / custom). Charts for:
- **Stool color over time** — scatter plot, color-coded by stool color rating
- **Weight curve** — with WHO percentile bands (3rd, 15th, 50th, 85th, 97th) overlaid
- **Temperature** — line chart with fever threshold line
- **Abdomen circumference** — line chart
- **Feeding volume / caloric intake** — daily aggregated bar chart
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
7. **Medication adherence** — % of scheduled doses logged as given
8. **Notable observations** — any flagged notes, bruising entries, photos (thumbnails)
9. **Photo appendix** — selected stool/skin photos in chronological order (optional, parent-selectable before export)

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
    id          TEXT PRIMARY KEY,  -- UUID
    google_id   TEXT UNIQUE NOT NULL,
    email       TEXT NOT NULL,
    name        TEXT NOT NULL,
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
    notes           TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Parent ↔ Baby link
CREATE TABLE baby_parents (
    baby_id     TEXT REFERENCES babies(id),
    user_id     TEXT REFERENCES users(id),
    role        TEXT DEFAULT 'parent',
    joined_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (baby_id, user_id)
);

-- Invite codes
CREATE TABLE invites (
    code        TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id),
    created_by  TEXT REFERENCES users(id),
    used_by     TEXT REFERENCES users(id),
    expires_at  DATETIME NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Generic pattern for metric tables (feedings shown as example)
CREATE TABLE feedings (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) NOT NULL,
    logged_by   TEXT REFERENCES users(id) NOT NULL,
    timestamp   DATETIME NOT NULL,
    feed_type   TEXT NOT NULL,
    volume_ml   REAL,
    cal_density REAL,
    duration_min INTEGER,
    notes       TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE stools (
    id              TEXT PRIMARY KEY,
    baby_id         TEXT REFERENCES babies(id) NOT NULL,
    logged_by       TEXT REFERENCES users(id) NOT NULL,
    timestamp       DATETIME NOT NULL,
    color_rating    INTEGER NOT NULL CHECK (color_rating BETWEEN 1 AND 7),
    color_label     TEXT,
    consistency     TEXT,
    volume_estimate TEXT,
    photo_key       TEXT,           -- R2 object key
    notes           TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Similar tables: urine, weights, abdomen, temperatures, skin_observations,
-- bruising, lab_results, general_notes

-- Medications (definitions / schedules)
CREATE TABLE medications (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) NOT NULL,
    name        TEXT NOT NULL,
    dose        TEXT NOT NULL,
    frequency   TEXT NOT NULL,
    schedule    TEXT,              -- JSON array of times, e.g., ["08:00","20:00"]
    active      BOOLEAN DEFAULT TRUE,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Medication administration log
CREATE TABLE med_logs (
    id              TEXT PRIMARY KEY,
    medication_id   TEXT REFERENCES medications(id) NOT NULL,
    baby_id         TEXT REFERENCES babies(id) NOT NULL,
    logged_by       TEXT REFERENCES users(id) NOT NULL,
    scheduled_time  DATETIME,
    given_at        DATETIME,
    skipped         BOOLEAN DEFAULT FALSE,
    skip_reason     TEXT,
    notes           TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

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
- **CSRF protection** — token-based for state-changing requests.
- **Photo access** — R2 objects are private. Backend generates **signed URLs** (time-limited) for photo access. No public bucket access.
- **Input validation** — all inputs validated server-side. Parameterized SQL queries (no injection).
- **Rate limiting** — basic rate limiting on API endpoints (personal use, but good hygiene).
- **Invite codes** — single-use, expire after 24 hours.

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
- Report customization (date range, include/exclude photos)
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
