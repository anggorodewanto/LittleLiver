# LittleLiver

Post-Kasai baby health tracking app for parents.

LittleLiver helps parents track daily health metrics for babies recovering from the Kasai procedure (biliary atresia). It provides structured logging of stool color, medication adherence, weight, fever, and other vitals, with dashboards and reports to share with medical teams.

## Tech Stack

- **Backend:** Go, SQLite
- **Frontend:** Svelte (TypeScript, strict mode)
- **Storage:** Cloudflare R2 (photo uploads)
- **Deployment:** fly.io with Docker

## Development Setup

### Prerequisites

- Go 1.26+
- Node 20+

### Backend

```bash
cd backend
go test ./... -v -cover   # run tests
make all                  # build
```

### Frontend

```bash
cd frontend
npm install
npm test          # run tests
npm run build     # build
```

## Deployment

Deployed on fly.io using Docker. See `Dockerfile` and `fly.toml` for configuration.

## Specification

See `docs/SPEC.md` for the complete product specification.
