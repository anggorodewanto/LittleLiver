# LittleLiver — Deployment Checklist

Interactive checklist for deploying to fly.io. Steps marked **YOU** require
human action (browser, credentials, CLI auth). Steps marked **CLAUDE** can be
done by Claude Code. Steps marked **TOGETHER** need coordination.

---

## Phase 1: External Services Setup (YOU)

### 1.1 — fly.io Account & CLI

- [ ] Install flyctl: `curl -L https://fly.io/install.sh | sh`
- [ ] Authenticate: `fly auth login`
- [ ] Create the app (if not already): `fly apps create littleliver`
      (Pick a different name if `littleliver` is taken — tell Claude the new name)
- [ ] Create persistent volume:
      `fly volumes create littleliver_data --region iad --size 1`
      (1 GB is plenty for SQLite personal use; adjust region if needed)

### 1.2 — Google OAuth Credentials

- [ ] Go to [Google Cloud Console](https://console.cloud.google.com/)
- [ ] Create a project (or use an existing one)
- [ ] Enable "Google Identity" / OAuth consent screen
  - App name: LittleLiver
  - User type: External (or Internal if using Workspace)
  - Scopes: `email`, `profile`, `openid`
  - Add your and your partner's Google emails as test users
    (required while app is in "Testing" status)
- [ ] Create OAuth 2.0 Client ID (Web application)
  - Authorized redirect URI: `https://<your-app>.fly.dev/auth/google/callback`
  - Note down: **Client ID** and **Client Secret**

### 1.3 — Anthropic API Key (for Lab Extraction)

- [ ] Go to [Anthropic Console](https://console.anthropic.com/)
- [ ] Create an account or log in
- [ ] Go to API Keys → Create Key
- [ ] Note down: **API Key** (starts with `sk-ant-...`)
- [ ] Add credit to your account (lab extraction costs ~$0.01–0.05 per extraction)

> This enables the AI-powered lab result extraction feature, which uses Claude's
> vision capabilities to read lab report photos and extract structured results.
> The feature is optional — if `ANTHROPIC_API_KEY` is not set, the extraction
> endpoint will return an error but all other features work normally.

### 1.4 — Cloudflare R2 Bucket

- [ ] Log into [Cloudflare Dashboard](https://dash.cloudflare.com/) → R2
- [ ] Create bucket named `littleliver-photos`
- [ ] Create R2 API token (Object Read & Write permissions for the bucket)
- [ ] Note down: **Account ID**, **Access Key ID**, **Secret Access Key**

### 1.5 — Generate Secrets (TOGETHER)

Tell Claude you're ready, and provide:
- Google Client ID
- Google Client Secret
- Anthropic API Key
- R2 Account ID, Access Key ID, Secret Access Key

Claude will generate the remaining secrets (SESSION_SECRET, VAPID keys) and
give you the full `fly secrets set` command to run.

---

## Phase 2: Configure & Deploy (TOGETHER)

### 2.1 — Set fly.io Secrets

Claude will prepare the command. **You run it** (it contains sensitive values):

```bash
fly secrets set \
  GOOGLE_CLIENT_ID="..." \
  GOOGLE_CLIENT_SECRET="..." \
  SESSION_SECRET="..." \
  ANTHROPIC_API_KEY="..." \
  R2_ACCOUNT_ID="..." \
  R2_ACCESS_KEY_ID="..." \
  R2_SECRET_ACCESS_KEY="..." \
  R2_BUCKET_NAME="littleliver-photos" \
  VAPID_PUBLIC_KEY="..." \
  VAPID_PRIVATE_KEY="..." \
  VAPID_SUBSCRIBER="mailto:you@example.com" \
  BASE_URL="https://littleliver.fly.dev"
```

### 2.2 — Verify App Name (CLAUDE)

If the fly.io app name differs from `littleliver`, Claude updates `fly.toml`.

### 2.3 — Deploy (YOU)

```bash
fly deploy
```

First deploy takes ~2-3 minutes (Docker build). Watch for:
- Frontend build stage succeeds
- Backend build stage succeeds
- Health check passes (`GET /health` returns 200)

### 2.4 — Verify Deployment (TOGETHER)

Claude will tell you what to check:

1. **Health check:** `curl https://<app>.fly.dev/health`
   → Should return `{"status":"ok"}`

2. **OAuth flow:** Open `https://<app>.fly.dev` in browser
   → Should redirect to Google login
   → After login, should show "Create Baby" or "Join with Invite Code"

3. **R2 connectivity:** Create a baby, log a stool entry with a photo
   → Photo should upload and display

4. **Push notifications:** Accept the notification permission prompt
   → Should register without errors in browser console

---

## Phase 3: Post-Deploy Verification (TOGETHER)

### 3.1 — Functional Smoke Test

- [ ] Login with Google account
- [ ] Create a baby profile
- [ ] Log a feeding (formula, 120mL) → verify calories calculated
- [ ] Log a stool (color rating 2) → verify acholic alert appears
- [ ] Log a temperature (38.5°C rectal) → verify fever alert appears
- [ ] Log a weight → check trends view for WHO percentile overlay
- [ ] Import lab results: tap "Import from Photo", take/select a lab report photo
  → verify extraction returns results → review and save
- [ ] Create a medication with schedule → verify it appears in "Upcoming"
- [ ] Generate a PDF report → verify it downloads with data
- [ ] Install as PWA (Add to Home Screen) → verify standalone mode

### 3.2 — Second Parent Flow

- [ ] Generate an invite code from Settings
- [ ] Open app on partner's phone, log in with their Google account
- [ ] Enter invite code → verify they see the baby and all data
- [ ] Partner logs an entry → verify you see it on your device

### 3.3 — Push Notification Test

- [ ] Create a medication with a schedule time ~2 minutes from now
- [ ] Wait for the notification to fire
- [ ] Tap notification → should open dose logging form
- [ ] Log dose → verify follow-up notification is suppressed

---

## Phase 4: Optional Polish (CLAUDE)

### 4.1 — Custom Domain

If you have a custom domain:
- [ ] `fly certs create yourdomain.com`
- [ ] Add CNAME record: `yourdomain.com → littleliver.fly.dev`
- [ ] Tell Claude the new domain → updates BASE_URL and OAuth redirect URI

### 4.2 — Publish OAuth App (YOU)

Once testing is complete:
- [ ] Go to Google Cloud Console → OAuth consent screen
- [ ] Click "Publish App" to move out of Testing status
  (Otherwise only test users can log in)

---

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| Health check fails after deploy | `fly logs` — check for migration or DB errors |
| OAuth redirect mismatch | Verify redirect URI in Google Console matches `BASE_URL + /auth/google/callback` |
| Photos don't upload | Check R2 credentials; `fly logs` for "R2 not configured" warning |
| Lab extraction fails | Check `ANTHROPIC_API_KEY` is set: `fly secrets list`. Check `fly logs` for Anthropic API errors |
| No push notifications | Check VAPID keys are set; browser must grant notification permission |
| DB lost after deploy | Verify volume is mounted: `fly volumes list` — should show `littleliver_data` |
| 502 after deploy | App may be starting — wait 10s. Check `fly logs` for startup errors |
| Session lost on redeploy | Sessions are in SQLite on persistent volume — should survive. Check volume mount. |
