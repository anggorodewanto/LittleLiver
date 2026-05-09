# PRD: Imaging Studies (Lab Images)

**Status:** Draft
**Owner:** Anggoro Dewanto
**Last updated:** 2026-05-09

---

## 1. Problem & Context

Parents tracking their post-Kasai baby's care accumulate non-numeric lab/imaging artifacts: CT scans, ultrasounds, MRI films, HIDA scans, and multi-page radiology PDFs from the hospital. The app's existing `lab_results` entity is EAV-style for numeric tests (test name + value + unit + reference range) and does not fit imaging studies, which have no value/unit and instead consist of one or more image/PDF files plus narrative findings.

Today there is no place in LittleLiver to store these artifacts. Parents keep them scattered across phone galleries, email attachments, and printed paper — making them hard to surface during clinic visits and absent from the clinical PDF report shared with hepatologists.

This PRD adds a first-class **imaging studies** entity that stores the artifact files plus structured metadata, surfaces them inline in the labs view, and includes them in the generated clinical PDF.

---

## 2. Goals / Non-goals

### Goals

- Store imaging study artifacts (images + PDFs) with a single free-text `notes` field plus structured metadata (study type, study date).
- Reduce manual data-entry burden via Claude Vision auto-detect of study type, date, and findings on upload — auto-fill targets the single `notes` field.
- Integrate seamlessly into the existing labs view — single chronological list of all lab activity (numeric labs + imaging studies).
- Include imaging studies in the clinical PDF report (thumbnails + notes text).
- Enhanced in-app viewer: pinch-zoom, pan, swipe between files, inline multi-page PDF rendering via `pdf.js`.

### Non-goals

- DICOM viewer / native DICOM file support.
- Image annotation, markup, or measurement tools.
- Sharing/integration with hospital PACS or other clinical systems.
- Structured findings beyond a single free-text `notes` field (no codified observations, no separate `label`/`findings` split).
- Dashboard charting, count tiles, or alerts for imaging studies (the lab trends chart remains numeric-only).
- Formal evaluation harness for Vision auto-detect accuracy.

---

## 3. Target Users & Use Cases

**Users:** same as the rest of LittleLiver — post-Kasai parents (primary: the user, his wife). Personal-use app, two linked parents per baby with equal access.

**Use cases:**

- **Capture paper report:** parent photographs a printed radiology report at the clinic and stores it.
- **Capture printed films:** parent scans/photographs printed CT or ultrasound images.
- **Upload hospital PDF:** parent receives a multi-page PDF from the hospital portal/email and uploads it directly.
- **Reference during clinic visit:** parent opens the labs view at the next appointment, swipes through images to show the hepatologist.
- **Share via clinical PDF:** when sharing the auto-generated clinical PDF, imaging thumbnails + notes are included alongside numeric labs and other metrics.

---

## 4. User Stories / Key Flows

### 4.1 Upload — happy path (auto-detect)

1. In the labs view, parent taps **"Add imaging study"**.
2. File picker opens — parent selects 1–10 files (images and/or PDFs, mix allowed).
3. Each file uploads on selection via the existing `/upload` endpoint; per-file progress is shown. The **Save** button stays disabled until all uploads complete.
4. As soon as all uploads complete, the client immediately POSTs `/imaging-studies/extract` with the returned R2 keys; an **"Analyzing…"** spinner is shown.
5. Vision returns `{ suggested: { study_type, study_date, findings }, notes }`. The form pre-fills:
   - `study_type` (mapped to a quick-pick chip if it matches CT / Ultrasound / MRI; otherwise placed in the free-text field),
   - `study_date`,
   - `notes` (populated with the findings text).
6. Auto-filled fields are visually highlighted so the user knows to verify.
7. Parent reviews, edits if needed, taps **Save**.

**Invariant:** user-typed values always win — extraction never overwrites a non-empty field.

### 4.2 Upload — auto-detect failure

1. Steps 1–3 above complete normally.
2. Extraction fails (rate limit `429`, Vision/upstream error `502`, or a `200` with all-null suggestions).
3. Toast: **"Couldn't analyze, please fill manually."**
4. Form remains usable and empty (no fields auto-filled).
5. Parent fills required fields manually and saves. **Extraction failure never blocks save.**

### 4.3 Upload — mid-flow upload failure

1. One or more files in the multi-select fail to upload (network drop, server error).
2. Successfully-uploaded keys are retained in form state; failed file shows a retry control.
3. Parent retries the failed file (or removes it). Partial state persists until **Save** is pressed.
4. **Save** remains disabled until every file in the current selection has either uploaded successfully or been removed.
5. If the user closes the form without saving, successfully-uploaded R2 objects become orphans and are reaped by the existing 24h `linked_at IS NULL` cleanup cron — no special form-abandonment handling needed.

### 4.4 View — labs list

- Labs view shows imaging entries inlined chronologically with numeric labs (sorted by `timestamp` DESC).
- Frontend fetches `/labs` and `/imaging-studies` in parallel and merges by `timestamp` desc client-side. Each source has its own cursor.
- Cross-page merging is best-effort: a backdated entry on page 2 of one source may not appear in the correct chronological position relative to entries on page 1 of the other source. This matches the existing pagination semantics in SPEC.
- Imaging entries are visually distinguished by an icon/badge (e.g., `🖼️ CT`, `🖼️ Ultrasound`).

### 4.5 View — detail + enhanced viewer modal

- Tapping an imaging entry opens a detail screen showing thumbnails of all files in the study + metadata + notes.
- Tapping a thumbnail opens the **enhanced viewer modal**:
  - Pinch-zoom and pan on images.
  - Horizontal swipe between files in the same study.
  - Multi-page PDFs render inline as a vertical scrollable view via **`pdf.js`** (lazy-loaded so it does not bloat the initial bundle).
  - Modal close: swipe-down gesture or explicit X button.

### 4.6 Edit

- Either linked parent can edit any field (equal-access auth, matching the existing metric pattern).
- File set is replaced as a whole on `PUT` (full `photo_keys` replacement). On any change to `photo_keys`, the server sets `linked_at = NULL` on `photo_uploads` rows whose keys were removed; the existing cleanup cron deletes the underlying R2 objects after the >24h orphan window.
- `updated_by` is set on every `PUT`.

### 4.7 Delete

- Either linked parent can hard-delete an entry (matches SPEC default; medications are the only soft-delete exception).
- `ON DELETE CASCADE` on `baby_id` propagates baby-level deletes.
- The existing `photo_uploads` cleanup cron handles R2 object removal via the existing rule: `(linked_at IS NULL AND uploaded_at < NOW() - 24h) OR (baby_id IS NULL)`.

---

## 5. Functional Requirements

### 5.1 Data Model

New table `imaging_studies`:

```sql
CREATE TABLE imaging_studies (
    id           TEXT PRIMARY KEY,             -- ULID
    baby_id      TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by    TEXT REFERENCES users(id) NOT NULL,
    updated_by   TEXT REFERENCES users(id),
    study_date   TEXT NOT NULL,                -- naive YYYY-MM-DD
    study_type   TEXT NOT NULL,                -- "CT" | "Ultrasound" | "MRI" | free text
    notes        TEXT,                         -- single free-text field; Vision auto-fills with findings; user edits freely
    photo_keys   TEXT NOT NULL,                -- JSON array of R2 keys (1-10), mix of images + PDFs allowed; enforced server-side on create + update
    timestamp    DATETIME NOT NULL,            -- derived from study_date 12:00 in X-Timezone header at create time
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_imaging_studies_baby_timestamp ON imaging_studies(baby_id, timestamp DESC);
```

**Field rules:**

- `id` — ULID, generated server-side.
- `study_date` — stored as naive `YYYY-MM-DD` text (matches SPEC convention for date-only fields).
- `study_type` — required, free text. UI offers quick-picks (CT, Ultrasound, MRI) but accepts any string. HIDA is **not** a quick-pick — users free-text it.
- `notes` — single optional free-text field. **There is no separate `label` field and no separate `findings` field.** Vision auto-fill targets `notes`. Parent edits freely.
- `photo_keys` — JSON-encoded array of 1–10 R2 keys. Mix of images and PDFs allowed. Length and presence are enforced server-side on both create and update.
- `timestamp` — derived at create time as `study_date` at 12:00 local time in the `X-Timezone` request header. Used as the unified sort key in the merged labs list.
- `logged_by` — set at creation; never changes.
- `updated_by` — set on every `PUT` (matches SPEC pattern for all metric tables).
- `created_at`, `updated_at` — managed by SQLite defaults / app on update. App writes `updated_at = NOW()` on every PUT (matches SPEC pattern for all metric tables).

**Account-deletion anonymization (followup):** SPEC §2.2 anonymization list must add `imaging_studies.logged_by` and `imaging_studies.updated_by`. This is a **SPEC edit deferred** until explicitly approved by the user (per CLAUDE.md constraint that SPEC must not be modified without explicit ask). Tracked as a required followup, not implemented inline.

### 5.2 API Endpoints

All endpoints follow existing conventions: ULID IDs, cursor pagination (50/page, ULID DESC), equal-access auth (any linked parent of the baby), `X-Timezone` header for date/time interpretation, and signed-URL response shape that replaces raw `photo_keys` (TTL 1h).

| Method   | Path                                                  | Purpose                          |
|----------|-------------------------------------------------------|----------------------------------|
| `POST`   | `/api/babies/:id/imaging-studies`                     | Create entry                     |
| `GET`    | `/api/babies/:id/imaging-studies?from=&to=&cursor=`   | List, filtered by `timestamp`    |
| `GET`    | `/api/babies/:id/imaging-studies/:entryId`            | Detail                           |
| `PUT`    | `/api/babies/:id/imaging-studies/:entryId`            | Edit (full replacement)          |
| `DELETE` | `/api/babies/:id/imaging-studies/:entryId`            | Hard delete                      |
| `POST`   | `/api/babies/:id/imaging-studies/extract`             | Vision suggest (read-only)       |

**`POST /imaging-studies` (create):**

- Body: `{ "study_date": "YYYY-MM-DD", "study_type": "...", "notes": "..." | null, "photo_keys": ["...", "..."] }`
- Validates: `study_type` non-empty, `study_date` matches `YYYY-MM-DD`, `photo_keys` length 1–10.
- Sets `timestamp` from `study_date` 12:00 in `X-Timezone`.
- Marks each `photo_keys` row in `photo_uploads` as linked (matches SPEC §5.4).

**`GET /imaging-studies` (list):**

- Filters by `timestamp` (matches SPEC pattern for `from`/`to`).
- Cursor pagination, 50 per page, ULID DESC.
- `photo_keys` replaced with array of signed URLs (TTL 1h).

**`GET /imaging-studies/:entryId` (detail):**

- Returns entry with signed-URL `photo_keys`.

**`PUT /imaging-studies/:entryId` (edit):**

- Body shape mirrors create.
- On a change to `photo_keys`: server sets `linked_at = NULL` on `photo_uploads` rows whose keys were **removed** (mirrors SPEC §5.4). Cleanup cron deletes after 24h.
- `updated_by` set to the requesting user.
- If `study_date` changes on PUT, `timestamp` is recomputed as `study_date 12:00` in the PUT request's `X-Timezone` header.
- `logged_by` is immutable — only `updated_by`, `study_date`, `study_type`, `notes`, `photo_keys`, and the derived `timestamp` may change on PUT. (Matches SPEC §5.3 global rule.)

**`DELETE /imaging-studies/:entryId` (delete):**

- Hard delete. `ON DELETE CASCADE` on `baby_id` handles row removal when a baby is deleted.
- R2 cleanup via the existing cron: `(linked_at IS NULL AND uploaded_at < NOW() - 24h) OR (baby_id IS NULL)`.

**`POST /imaging-studies/extract` (Vision suggest):**

- Request: `{ "photo_keys": ["..."] }`
- Response: `{ "suggested": { "study_type": "CT" | null, "study_date": "2026-05-09" | null, "findings": "..." | null }, "notes": "..." | null }`
- Read-only — never writes to the DB.
- **Server-side cap: max 10 `photo_keys` per request** (matches `/labs/extract` cap).
- **Vision input:** for image keys, the server sends the resized "original" JPEG (existing pipeline). For PDF keys, the server sends the first-page rasterized JPEG thumbnail generated at upload time. Vision API never receives raw PDF bytes.
- **Rate limit: shared with `/labs/extract`, raised to 50 requests/hour/user** (was 10; raised to accommodate both flows).
- Status codes:
  - `200` with extracted suggestions on success.
  - `200` with all-null `suggested` if Vision yields nothing — client treats as failure UX.
  - `429` when the shared rate limit is exceeded.
  - `502` on Vision API / upstream error.
- Cost: ~$0.01–$0.05 per call. Uses the existing `ANTHROPIC_API_KEY` Fly secret.

**Auth:** all endpoints require equal-access baby auth (linked parent of the baby), identical to labs.

### 5.3 Upload Endpoint Changes

The existing `/upload` endpoint already accepts JPEG/PNG/HEIC, applies a **25 MB raw cap**, re-encodes via ImageMagick to a 2000px JPEG, and generates a 300px thumbnail. Fly VM is 512MB, so OOM safety is critical — there is no in-process Go image decoding.

This PRD extends `/upload` with separate per-content-type caps and a PDF code path:

- **Images:** **5 MB cap** (existing behavior; the image cap is **not** changed).
- **PDFs:** **20 MB cap**, separate code path. The 25 MB raw upper bound still applies as a hard ceiling.

**MIME allowlist (post-extension):** Server accepts MIME types: `image/jpeg`, `image/png`, `image/heic`, `application/pdf`. Other PDF-ish MIME types (e.g., `application/x-pdf`) are rejected.

**PDF processing:**

- Server detects `application/pdf` from MIME type.
- PDFs are **not** routed through the Go image decoder or the HEIC→JPEG conversion path.
- Server uses an **ImageMagick + Ghostscript subprocess** to rasterize the **first page only** to a 300px JPEG thumbnail. No in-process Go PDF decoder (no MuPDF/pdfium dependency).
- Original PDF is stored as-is in R2 with a key ending in `.pdf`.
- Thumbnail key follows the existing thumbnail naming pattern. PDF thumbnails follow the same `thumb_<id>.jpg` naming as image thumbnails — a PDF original at `photos/<id>.pdf` produces `photos/thumb_<id>.jpg` (always JPEG, even though the original is PDF).

**Container changes:**

- Add **Ghostscript** to the Docker image (small footprint; pairs with already-installed ImageMagick).

**OOM safety invariants preserved:**

- No in-process Go decoding of images or PDFs.
- All transcoding/rasterization happens in a bounded subprocess so peak memory stays outside the Go heap.

**Failure handling:** if Ghostscript rasterization fails on a malformed PDF, the upload still succeeds (the original PDF is stored), but no thumbnail is generated. Downstream consumers (clinical report, viewer) fall back to a generic PDF icon. Final fallback wiring is confirmed during implementation.

### 5.4 UI — Labs view

- Single chronological list mixes numeric `lab_results` and `imaging_studies`.
- Frontend fetches `/labs` and `/imaging-studies` in **parallel** and merges by `timestamp` desc client-side. Cursors are independent per source.
- Cross-page merge is **best-effort** — a backdated entry on a later page of one source may sort incorrectly relative to entries on an earlier page of the other source. This matches SPEC's existing pagination edge cases.
- Imaging row displays: icon/badge (`🖼️` + `study_type`), `study_date`, file count.
- Tap → detail view (thumbnails grid + metadata + notes).
- Tap a thumbnail → enhanced viewer modal.

### 5.5 UI — Add / edit form

- File picker accepts images (`image/*`) and PDFs (`application/pdf`), up to 10 files total.
- Each file uploads on selection via existing `/upload`; per-file progress is shown.
- **Save** button is disabled until **all** selected files have either uploaded successfully or been removed.
- As soon as all uploads complete, the client immediately POSTs `/imaging-studies/extract` with the returned keys; an **"Analyzing…"** spinner is shown.
- On success: pre-fill `study_type` (mapped to quick-pick chip when matching CT / Ultrasound / MRI), `study_date`, `notes` (with findings text). Auto-filled fields are visually highlighted to prompt verification.
- On failure (`429`, `502`, or all-null `suggested`): toast **"Couldn't analyze, please fill manually."** Form remains empty/usable.
- **Mid-flow upload failure:** form retains successfully-uploaded keys; failed file shows a retry control. Partial state persists until Save.
- **User-typed values always win** — extraction never overwrites a non-empty field.
- **Quick-pick chips:** **CT**, **Ultrasound**, **MRI**. HIDA is **not** a quick-pick (free-text only). A free-text "Other" input is always available alongside the chips.
- **Required fields:** at least 1 file, `study_date`, `study_type`.
- **Optional field:** `notes`.
- **No future-date validation on `study_date`** — matches the absence of future-date validation on other metrics.

### 5.6 UI — Enhanced viewer modal

- Touch-friendly: pinch-to-zoom, two-finger pan, single-finger pan when zoomed.
- Horizontal swipe navigates between files in the current study.
- Multi-page PDFs render inline with vertical page-scroll via **`pdf.js`** (frontend dep, ~1–2 MB minified, **lazy-loaded** so it does not bloat initial bundle).
- Modal close: swipe-down gesture or explicit X button.

### 5.7 Dashboard

- Imaging studies do **NOT** appear on the dashboard.
- No charts, no count tile, no alerts for imaging studies.
- The existing lab trends chart is unaffected (it remains numeric-only, sourced from `lab_results`).

### 5.8 Clinical PDF Report

- The existing photo appendix (currently stool/skin photos) is extended to include imaging studies whose `timestamp` falls within the report's date range.
- Per-study layout in the appendix: **one representative thumbnail (300px)** + `study_date` + `study_type` + `notes` text.
- Thumbnail source:
  - **Images:** the existing 300px JPEG thumbnail generated at upload.
  - **PDFs:** the first-page rasterization JPEG generated at upload (already stored).
  - **Malformed PDFs without a thumbnail:** generic PDF icon placeholder (final wiring confirmed during implementation).
- Studies are sorted **chronologically by `timestamp`** within the appendix.

---

## 6. Non-functional Requirements

### Performance & Limits

- Per-file upload caps: **images 5 MB**, **PDFs 20 MB**, with the existing 25 MB raw cap honored as the upper bound.
- Per-study file count: **1–10 files**, mix of images and PDFs allowed.
- Viewer modal opens within ~200ms of thumbnail tap (best-effort target on mobile PWA).

### OOM Safety (Fly 512MB VM)

- PDF rasterization runs in an ImageMagick + Ghostscript subprocess; peak memory bounded outside the Go heap.
- No in-process Go decoding of PDFs or images on the upload path.
- These invariants match the existing image upload pipeline and must not regress.

### Security & Auth

- Same auth as labs — only linked parents of the baby can read/write.
- Raw R2 keys are never returned to the client; only signed URLs (TTL 1h).
- Rate limit on Vision: shared 50 req/hr/user with `/labs/extract`.

### Vision API

- Cost: ~$0.01–$0.05 per request.
- Auth: existing `ANTHROPIC_API_KEY` Fly secret.

### Mobile-first

- Enhanced viewer is fully touch-driven (pinch zoom, pan, swipe between files, swipe-down to close).
- Form fully usable on mobile PWA; file picker uses the platform-native multi-select.

### Test Coverage (project standard)

- ≥ 90% backend, ≥ 90% frontend.
- **Strict Red-Green-Refactor TDD** (per CLAUDE.md): tests written before implementation; full suite green before commit.

---

## 7. Success Metrics

- A user can upload **and save** an imaging study end-to-end in **< 30 seconds** on a normal mobile connection (informal observation).
- Vision auto-detect produces useful suggestions on a typical hospital report image (informal observation — **no formal eval planned**).
- **Zero data loss:** an extraction failure never blocks save.
- All tests pass; coverage targets met before merge.

---

## 8. Constraints, Assumptions, Dependencies

### Dependencies

- **R2 photo infrastructure** — `photo_uploads` table, `POST /upload`, signed-URL helper, cleanup cron — extended in this PRD with PDF support.
- **Anthropic Claude Vision API** — already integrated for `/labs/extract`. `ANTHROPIC_API_KEY` Fly secret.
- **ImageMagick** subprocess — already in container.
- **Ghostscript** — **NEW** addition to the Docker image (small footprint, pairs with ImageMagick) for PDF first-page rasterization.
- **`pdf.js`** — **NEW** frontend dependency, **lazy-loaded** only when an imaging entry is opened.
- **Existing labs view component** — modified to merge `imaging_studies` into the chronological list.

### Assumptions

- Parents primarily upload **photographs of paper reports** (camera) or **PDFs from hospital email/portal**. DICOM/PACS integration is out of scope.
- Existing `photo_uploads` cleanup pattern is sufficient — no new cleanup job is needed for imaging studies.
- Personal-use app — no multi-tenant or scaling concerns.

### Constraints

- **Must not modify `docs/SPEC.md`** without explicit user approval (per CLAUDE.md).
- Stack is locked: Go + SQLite + SvelteKit + R2 + Fly.io.
- Minimal external dependencies. New deps in this PRD: Ghostscript (container), `pdf.js` (frontend, lazy-loaded).

---

## 9. Risks & Open Questions

### Risks

- **`pdf.js` bundle size impacts PWA:**
  *Mitigation:* lazy-load `pdf.js` only when an imaging entry is opened. ~1–2 MB minified is acceptable for a personal-use app on a non-critical path.
- **ImageMagick + Ghostscript PDF rasterize could be slow / memory-hungry on large multi-page PDFs:**
  *Mitigation:* rasterize **only the first page** at upload; cap PDFs at 20 MB; subprocess-bounded so the Go heap is unaffected.
- **Vision misclassifies type or date confidently:** user may not notice and save incorrect metadata.
  *Mitigation:* visually highlight auto-filled fields; required-field validation forces the user to acknowledge `study_type`.
- **Cross-page merge of labs + imaging in the client may show out-of-order entries near pagination boundaries:**
  *Mitigation:* accepted edge case. Matches existing SPEC pagination semantics.

### Open questions

_All previously-open items are now resolved (user approval captured 2026-05-09)._

- **SPEC amendments (bundled, approved).** The following SPEC edits are authorized and will land alongside Phase 1 implementation (SPEC describes what exists; edits apply when code lands):
  - **§2.2** — add `imaging_studies.logged_by` and `imaging_studies.updated_by` to the account-deletion anonymization list.
  - **§5.3** — add `/imaging-studies` to the metric endpoint list.
  - **§5.4** — extend "Photo support scope" to include `imaging_studies`; update upload limits to reflect 5 MB image / 20 MB PDF caps and the new `application/pdf` MIME allowlist.
  - **§5.6** — update `/labs/extract` rate limit from 10/hr/user to 50/hr/user (shared with `/imaging-studies/extract`).
  - **§8.2** — clinical report Photo appendix extended to include imaging-study thumbnails.
- **Future-date validation on `study_date`** — **decided: no validation** (matches other metrics).
- **Malformed-PDF thumbnail fallback** — **decided: generic PDF icon**. Upload still succeeds; no thumbnail stored; downstream (clinical report, detail view) renders a generic PDF icon placeholder.

---

## 10. Milestones / Scope Cuts

The work is split into three phases so the core entity ships fast and Vision integration follows once the data path and viewer are stable.

### Phase 1 — MVP (manual entry, images only)

- New `imaging_studies` table + migration.
- Full CRUD API (`POST` / `GET` list / `GET` detail / `PUT` / `DELETE`) — **no extract endpoint yet**.
- `/upload` extended with separate image cap (no functional change since 5 MB image cap already exists). PDF support **deferred to Phase 2**.
- Add/edit form, **manual entry only** (no Vision call).
- Mixed labs-list rendering via two parallel client-side calls and client-side merge by `timestamp`.
- Enhanced **image** viewer modal: zoom, pan, swipe between files. **No PDF support yet.**
- Clinical PDF report includes imaging thumbnails + notes (images only).
- Tests + ≥ 90% coverage.

### Phase 2 — PDF support

- `/upload` extended to accept `application/pdf` with the 20 MB cap and dedicated code path.
- **Ghostscript** added to the Docker image.
- PDF first-page rasterization → 300px JPEG thumbnail at upload.
- `pdf.js` integration — inline multi-page PDF viewer (lazy-loaded).
- Clinical PDF report renders PDF thumbnails alongside image thumbnails.

### Phase 3 — Vision auto-detect

- `POST /imaging-studies/extract` endpoint (Claude Vision; **shared 50 req/hr/user** rate limit with `/labs/extract`).
- Auto-on-upload UX in the form: spinner, pre-fill, highlight, toast-on-failure.
- **Invariant:** user-typed values always win — extraction never overwrites a non-empty field.

### Quick-pick study types (initial set)

**CT**, **Ultrasound**, **MRI**.
HIDA and other modalities use the free-text input. Quick-picks expand based on real usage.
