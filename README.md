# LittleLiver

Post-Kasai baby health tracking app for parents.

LittleLiver is a personal-use web app for parents to track daily health metrics of an infant recovering from the Kasai portoenterostomy procedure (biliary atresia). Both parents can log data from their phones, view trends on a dashboard, and generate printable clinical summaries for hepatologist appointments.

## Features

### Health Metric Tracking
- **Feedings** — Breast milk, formula, fortified, solids with automatic caloric calculation
- **Stool output** — Color rating (1-7) based on Infant Stool Color Card with photo uploads
- **Urine output** — Color tracking and optional volume
- **Weight** — Plotted against WHO growth percentiles (sex-specific)
- **Temperature** — Multiple measurement methods with fever threshold detection
- **Abdomen** — Firmness, tenderness, girth measurements with photos
- **Skin & jaundice** — Jaundice level, scleral icterus, rash/bruise notes
- **Bruising** — Location, size, color with photos (coagulopathy monitoring)
- **Lab results** — EAV-style storage with common Kasai test quick-picks
- **General notes** — Free-form with categories (behavior, sleep, vomiting, etc.)
- **Fluid I/O log** — Unified intake/output ledger, auto-linked from feedings and diapers
- **Head circumference & MUAC** — Growth tracking with WHO percentiles

### Clinical Alerts
- **Acholic stool warning** — Prominent alert when stool color rating is 1-3 (bile flow failure indicator)
- **Cholangitis warning** — Fever alert with method-specific thresholds prompting immediate medical contact

### Medications & Reminders
- Medication management with pre-populated suggestions (UDCA, Bactrim, vitamins, iron)
- Dose logging (given/skipped) with flexible scheduling (daily, twice daily, every X days, custom)
- Web Push notifications for medication reminders

### AI Lab Extraction
- Upload lab report images and auto-extract results via Claude Vision API
- Batch import of extracted lab values

### Reporting & Analytics
- Dashboard with aggregated metrics and active alerts
- WHO growth percentile charts (weight, head circumference)
- Trend visualization with Chart.js
- Printable PDF clinical summary with embedded charts and photos

### Multi-User & Multi-Baby
- Google OAuth 2.0 authentication
- Invite codes to share baby profiles between parents (24-hour expiry)
- Unlimited authorized parents per baby with equal access
- Switch between multiple baby profiles

### PWA
- Installable on mobile devices
- App shell caching for offline graceful degradation
- Push notification support via VAPID

### Photo Management
- Cloudflare R2 storage with signed URLs
- HEIC to JPEG auto-conversion
- Attach photos to stools, abdomen, skin, bruising, and notes

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go 1.26+ |
| Database | SQLite (WAL mode) |
| Frontend | SvelteKit (TypeScript, strict mode) |
| Charts | Chart.js |
| Photo storage | Cloudflare R2 |
| PDF reports | maroto v2 + go-echarts |
| Push notifications | Web Push (VAPID) |
| Lab extraction | Claude Vision API |
| Deployment | fly.io with Docker |

## Development Setup

### Prerequisites

- Go 1.26+
- Node 20+
- ImageMagick (for HEIC conversion, included in Docker image)

### Backend

```bash
cd backend
go test ./... -v -cover   # run tests
go vet ./...              # lint
make all                  # build
```

### Frontend

```bash
cd frontend
npm install
npm test          # run tests (Vitest)
npm run build     # build
```

### Testing

This project follows strict Red-Green-Refactor TDD with a 90%+ code coverage target. All tests must be green before committing.

## Environment Variables

### Required

| Variable | Purpose |
|----------|---------|
| `GOOGLE_CLIENT_ID` | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | Google OAuth client secret |
| `SESSION_SECRET` | Session signing key |
| `BASE_URL` | Application base URL |

### Optional

| Variable | Purpose |
|----------|---------|
| `DATABASE_PATH` | SQLite DB path (default: `littleliver.db`) |
| `PORT` | HTTP port (default: `8080`) |
| `R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `R2_BUCKET_NAME` | Cloudflare R2 photo storage |
| `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY` | Push notification keys |
| `ANTHROPIC_API_KEY` | Claude Vision lab extraction |
| `TEST_MODE` | Set to `1` to enable test login and in-memory photo store |

## Project Structure

```
LittleLiver/
├── backend/
│   ├── cmd/server/        # HTTP server entrypoint
│   ├── internal/
│   │   ├── auth/          # Google OAuth, sessions
│   │   ├── handler/       # HTTP handlers (29 files)
│   │   ├── store/         # Data access layer (22 files)
│   │   ├── model/         # Domain types, validation
│   │   ├── middleware/     # Auth, CSRF, rate limiting
│   │   ├── labextract/    # Claude Vision integration
│   │   ├── notify/        # Web Push, reminder scheduler
│   │   ├── report/        # PDF generation
│   │   ├── storage/       # R2 client
│   │   ├── who/           # WHO growth data
│   │   ├── cron/          # Cleanup jobs
│   │   └── backup/        # DB backup to R2
│   └── migrations/        # 14 SQL migrations
├── frontend/
│   ├── src/
│   │   ├── routes/        # SvelteKit pages
│   │   └── lib/
│   │       ├── components/ # 40+ Svelte components
│   │       ├── stores/    # State management
│   │       └── api.ts     # REST client
│   └── src/tests/         # Vitest tests
├── docs/
│   ├── SPEC.md            # Full product specification
│   └── PHASES.md          # Implementation phases
├── scripts/deploy.sh      # Deployment automation
├── Dockerfile             # Multi-stage build
├── fly.toml               # fly.io config
└── docker-compose.yml     # Local dev
```

## Deployment

Deployed on fly.io using a multi-stage Docker build. SQLite persists on a mounted volume.

```bash
./scripts/deploy.sh --smoke    # deploy with smoke tests
./scripts/deploy.sh --secrets  # deploy and set secrets
./scripts/deploy.sh --all      # full deploy (secrets + smoke)
```

## Specification

See `docs/SPEC.md` for the complete product specification.
