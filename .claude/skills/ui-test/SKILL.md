---
name: ui-test
description: Run UI tests against the live LittleLiver app using Chrome browser automation with data seeding and cleanup
user-invocable: true
---

# UI Test Runner for LittleLiver

You are a UI test runner for the LittleLiver app. You use Chrome browser automation to execute test cases, seed required data via the API, run destructive tests, and clean up the database afterwards.

## Step 1: Load Test Cases

Read the file `docs/UI_TEST_CASES.md` to get the full list of test cases.

If the file does not exist, tell the user:
> "No test cases file found at `docs/UI_TEST_CASES.md`. Please create it first."

Then stop.

## Step 2: Parse Arguments

The user may provide an argument after `/ui-test`:

- **No argument** -> Run ALL test cases
- **Category name** (e.g., `dashboard`, `medications`, `navigation`, `trends`, `settings`, `metrics`, `entry`) -> Run only tests in that category
- **Specific test ID** (e.g., `TC-04-01`) -> Run only that single test case
- **`--seed-only`** -> Only seed data, don't run tests
- **`--cleanup-only`** -> Only run cleanup

## Step 3: Confirm with User

Use the `AskUserQuestion` tool to ask:

1. **Target URL**: "Which URL should I test against?" Options: `https://littleliver.fly.dev` (default), `http://localhost:5173`
2. **Auth status**: "Are you currently logged in to the app in Chrome?" (yes/no)
3. **Test mode**: "Which test mode?" Options:
   - `full` — Seed data, run ALL tests including destructive ones, clean up afterwards (Recommended)
   - `safe` — Read-only tests only, no data mutation, no cleanup needed
   - `seed+test` — Seed data and test but skip cleanup (inspect data afterwards)

If user is not logged in, prompt them to log in before proceeding.

## Step 4: Data Seeding (full and seed+test modes)

Before running tests, seed the database with test data via the API. Use `mcp__claude-in-chrome__javascript_tool` to make API calls from the authenticated browser session.

### Seeding Strategy

Use this JavaScript helper to make API calls from the browser (preserving the user's auth session):

```javascript
async function seedAPI(method, path, body) {
  const csrfResp = await fetch('/api/csrf-token', { credentials: 'include' });
  const { token } = await csrfResp.json();
  const opts = {
    method,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      'X-CSRF-Token': token,
      'X-Timezone': Intl.DateTimeFormat().resolvedOptions().timeZone
    }
  };
  if (body) opts.body = JSON.stringify(body);
  const resp = await fetch('/api' + path, opts);
  if (resp.status === 204) return null;
  return resp.json();
}
```

### Required Seed Data

Run these sequentially in a single javascript_tool call. Store returned IDs in variables for later cleanup.

**IMPORTANT:** Before seeding, GET the current state to avoid duplicating data. Query `/api/babies` first to check what exists.

#### 1. Ensure a test baby exists (or use existing one)

If user already has a baby, use it. Record the `babyId`.

If not, create one:
```javascript
const baby = await seedAPI('POST', '/babies', {
  name: 'UI Test Baby',
  date_of_birth: '2025-06-01',
  sex: 'female',
  diagnosis_date: '2025-06-15',
  kasai_date: '2025-06-20'
});
const babyId = baby.id;
```

#### 2. Seed metric entries for today (for dashboard tests)

```javascript
const now = new Date().toISOString();
const seededIds = { feedings: [], stools: [], urine: [], temperatures: [], weights: [], abdomen: [], skin: [], bruising: [], labs: [], notes: [], medications: [], medLogs: [] };

// Feeding
let r = await seedAPI('POST', `/babies/${babyId}/feedings`, {
  timestamp: now, feed_type: 'formula', volume_ml: 120, cal_density: 20
});
seededIds.feedings.push(r.id);

// Second feeding
r = await seedAPI('POST', `/babies/${babyId}/feedings`, {
  timestamp: now, feed_type: 'breast', duration_min: 15
});
seededIds.feedings.push(r.id);

// Urine (2 entries for wet diaper count)
r = await seedAPI('POST', `/babies/${babyId}/urine`, { timestamp: now, color: 'pale' });
seededIds.urine.push(r.id);
r = await seedAPI('POST', `/babies/${babyId}/urine`, { timestamp: now });
seededIds.urine.push(r.id);

// Stool with green color (rating 6 — safe, no alert)
r = await seedAPI('POST', `/babies/${babyId}/stools`, {
  timestamp: now, color_rating: 6, color_label: 'green', consistency: 'soft', volume_estimate: 'medium'
});
seededIds.stools.push(r.id);

// Stool with acholic color (rating 1 — triggers alert)
r = await seedAPI('POST', `/babies/${babyId}/stools`, {
  timestamp: now, color_rating: 1, color_label: 'white', consistency: 'soft'
});
seededIds.stools.push(r.id);

// Temperature (normal)
r = await seedAPI('POST', `/babies/${babyId}/temperatures`, {
  timestamp: now, value: 37.2, method: 'axillary'
});
seededIds.temperatures.push(r.id);

// Temperature (fever — triggers alert)
r = await seedAPI('POST', `/babies/${babyId}/temperatures`, {
  timestamp: now, value: 38.5, method: 'rectal'
});
seededIds.temperatures.push(r.id);

// Weight
r = await seedAPI('POST', `/babies/${babyId}/weights`, {
  timestamp: now, weight_kg: 5.45, measurement_source: 'clinic'
});
seededIds.weights.push(r.id);

// Abdomen
r = await seedAPI('POST', `/babies/${babyId}/abdomen`, {
  timestamp: now, appearance: 'soft', notes: 'Test entry'
});
seededIds.abdomen.push(r.id);

// Skin observation
r = await seedAPI('POST', `/babies/${babyId}/skin`, {
  timestamp: now, observations: 'Mild jaundice', affected_areas: 'face'
});
seededIds.skin.push(r.id);

// Bruising
r = await seedAPI('POST', `/babies/${babyId}/bruising`, {
  timestamp: now, location: 'left arm', size: 'small', color: 'purple'
});
seededIds.bruising.push(r.id);

// Lab result
r = await seedAPI('POST', `/babies/${babyId}/labs`, {
  timestamp: now, test_name: 'total_bilirubin', value: '2.1', unit: 'mg/dL', normal_range: '0.1-1.2'
});
seededIds.labs.push(r.id);

// General note
r = await seedAPI('POST', `/babies/${babyId}/notes`, {
  timestamp: now, content: 'Test note for UI testing', note_type: 'general'
});
seededIds.notes.push(r.id);
```

#### 3. Seed medications (for medication and dashboard tests)

```javascript
// Active medication with schedule
r = await seedAPI('POST', `/babies/${babyId}/medications`, {
  name: 'UDCA (ursodiol)', dose: '45mg', frequency: 'twice_daily',
  schedule_times: ['08:00', '20:00']
});
seededIds.medications.push(r.id);
const medId1 = r.id;

// Active medication — as needed
r = await seedAPI('POST', `/babies/${babyId}/medications`, {
  name: 'Vitamin D', dose: '400IU', frequency: 'as_needed'
});
seededIds.medications.push(r.id);

// Log a dose (given)
r = await seedAPI('POST', `/babies/${babyId}/med-logs`, {
  medication_id: medId1, skipped: false
});
seededIds.medLogs.push(r.id);

// Log a dose (skipped)
r = await seedAPI('POST', `/babies/${babyId}/med-logs`, {
  medication_id: medId1, skipped: true, skip_reason: 'Test skip'
});
seededIds.medLogs.push(r.id);
```

#### 4. Seed historical data for trends (past 14 days)

```javascript
for (let daysAgo = 1; daysAgo <= 14; daysAgo++) {
  const d = new Date();
  d.setDate(d.getDate() - daysAgo);
  const ts = d.toISOString();

  // Daily stool with varying color
  const rating = (daysAgo % 7) + 1;
  r = await seedAPI('POST', `/babies/${babyId}/stools`, {
    timestamp: ts, color_rating: rating, consistency: 'soft'
  });
  seededIds.stools.push(r.id);

  // Daily weight (gradual increase)
  r = await seedAPI('POST', `/babies/${babyId}/weights`, {
    timestamp: ts, weight_kg: 5.0 + (14 - daysAgo) * 0.03
  });
  seededIds.weights.push(r.id);

  // Daily temp
  r = await seedAPI('POST', `/babies/${babyId}/temperatures`, {
    timestamp: ts, value: 36.5 + Math.random() * 1.0, method: 'axillary'
  });
  seededIds.temperatures.push(r.id);

  // Daily feeding
  r = await seedAPI('POST', `/babies/${babyId}/feedings`, {
    timestamp: ts, feed_type: 'formula', volume_ml: 100 + daysAgo * 5
  });
  seededIds.feedings.push(r.id);

  // Daily urine
  r = await seedAPI('POST', `/babies/${babyId}/urine`, { timestamp: ts });
  seededIds.urine.push(r.id);
}
```

#### 5. Store seeded IDs for cleanup

After seeding, store the IDs in the browser's sessionStorage so the cleanup phase can access them:

```javascript
sessionStorage.setItem('ui_test_seeded', JSON.stringify({ babyId, seededIds, createdBaby: !!createdNewBaby }));
```

**Output** the babyId and counts of seeded items so the orchestrator can pass them to subagents.

## Step 5: Group and Launch Tests

Organize the selected test cases into these groups for parallel execution via the Agent tool:

- **Group A: Dashboard & Summary Cards** — Dashboard, summary cards, alerts, stool color trend, upcoming meds, quick log buttons
- **Group B: Metric Entry Forms** — All /log/* form rendering, validation, and submission tests
- **Group C: Navigation & Layout** — Login page, nav header, PWA, routing, cross-cutting (CSRF, timezone, etc.)
- **Group D: Trends & Reports** — Trend charts, date range selectors, report generation
- **Group E: Medications** — Medication CRUD, dose logging, scheduling, deactivate/reactivate
- **Group F: Settings & Baby Management** — Baby settings, invite codes, unlink, account deletion

### Subagent Instructions Template

For EACH group, launch a subagent using the Agent tool with these instructions:

---

**SUBAGENT PROMPT:**

```
You are a UI test executor for LittleLiver. Test the app at [TARGET_URL] using Chrome browser automation.

## CRITICAL: Loading Chrome Tools

Before using ANY mcp__claude-in-chrome__* tool, you MUST first load it via ToolSearch:
- Call ToolSearch with query "select:mcp__claude-in-chrome__<tool_name>"
- Only THEN call the actual tool

You must do this for EVERY Chrome tool before first use.

## Available Chrome Tools

- mcp__claude-in-chrome__tabs_context_mcp — Get current tab context (ALWAYS call first)
- mcp__claude-in-chrome__tabs_create_mcp — Create a new tab
- mcp__claude-in-chrome__navigate — Navigate to a URL
- mcp__claude-in-chrome__read_page — Read page content, check element presence
- mcp__claude-in-chrome__find — Find/locate elements on page
- mcp__claude-in-chrome__computer — Click, type, take screenshots, scroll
- mcp__claude-in-chrome__javascript_tool — Execute JavaScript (check localStorage, DOM, make API calls)
- mcp__claude-in-chrome__form_input — Fill form fields
- mcp__claude-in-chrome__get_page_text — Get all text content from page
- mcp__claude-in-chrome__read_console_messages — Check for JS errors
- mcp__claude-in-chrome__read_network_requests — Monitor API calls

## Setup

1. Load and call tabs_context_mcp to get current browser state
2. Load and call tabs_create_mcp to create a fresh tab
3. Load and call navigate to go to [TARGET_URL]

## Test Data Context

The following test data has been seeded and is available:
- Baby ID: [BABY_ID]
- Medications exist (UDCA twice daily, Vitamin D as-needed)
- Metric entries exist for today and past 14 days (feedings, stools, urine, temps, weights, etc.)
- An acholic stool (rating 1) and fever temp (38.5 rectal) exist to trigger alerts
- Med-logs exist (one given, one skipped)

## Test Mode: [full|safe]

[If full mode]: You MAY create, modify, and delete data as part of testing. All seeded data will be cleaned up after tests complete. When tests require form submission, go ahead and submit with test data. When tests require destructive actions (deactivate medication, dismiss alert), go ahead and perform them.

[If safe mode]: Do NOT submit forms or modify data. Only verify rendering, validation messages, and UI behavior.

## Making API Calls (for full mode)

Use this helper via javascript_tool when you need to create/delete data as part of testing:

```js
async function testAPI(method, path, body) {
  const csrfResp = await fetch('/api/csrf-token', { credentials: 'include' });
  const { token } = await csrfResp.json();
  const opts = { method, credentials: 'include', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': token, 'X-Timezone': Intl.DateTimeFormat().resolvedOptions().timeZone }};
  if (body) opts.body = JSON.stringify(body);
  const resp = await fetch('/api' + path, opts);
  if (resp.status === 204) return null;
  return resp.json();
}
```

## Test Execution

For each test case:
1. Navigate to the correct page/state
2. Perform the test actions (click, type, verify)
3. Verify expected results using read_page, javascript_tool, or screenshots
4. Record PASS or FAIL with details
5. On FAILURE: take a screenshot for debugging
6. If you created any additional data during testing, record the IDs for cleanup

## Handling Human-Required Actions

If a test requires something you cannot automate (Google OAuth, file picker), use AskUserQuestion to request the user's help. Wait for confirmation, then verify.

## Test Cases to Execute

[INSERT FULL TEST CASES HERE]

## Output Format

RESULTS_START
| Test ID | Name | Result | Notes |
|---------|------|--------|-------|
| TC-XX-XX | Test name | PASS/FAIL/SKIP | Details |
RESULTS_END

CREATED_IDS_START
[JSON array of any additional entry IDs created during testing, grouped by type: {"feedings": [...], "medications": [...], ...}]
CREATED_IDS_END
```

---

### Subagent Dispatch Rules

- **Include the FULL test case text** in each subagent's prompt
- **Include the confirmed TARGET_URL and BABY_ID** in each subagent prompt
- **Include the test mode** (full or safe)
- **Sequential dependencies**: Keep dependent tests in the same subagent in order
- **In full mode**: Subagents may submit forms and perform destructive actions
- **In full mode**: Subagents must report any IDs they created in CREATED_IDS_START/END
- Launch independent groups in parallel

## Step 6: Cleanup (full mode only)

After ALL subagents complete, clean up all seeded and test-created data. This ensures the database is returned to its pre-test state.

### Cleanup Strategy

Use `mcp__claude-in-chrome__javascript_tool` to delete all seeded entries via the API, in reverse order of creation (delete dependents first):

```javascript
// Retrieve seeded IDs
const stored = JSON.parse(sessionStorage.getItem('ui_test_seeded'));
const { babyId, seededIds, createdBaby } = stored;

// Also merge any IDs created by subagents during testing
// [Merge CREATED_IDS from subagent results here]

async function cleanupAPI(method, path) {
  const csrfResp = await fetch('/api/csrf-token', { credentials: 'include' });
  const { token } = await csrfResp.json();
  const resp = await fetch('/api' + path, {
    method, credentials: 'include',
    headers: { 'X-CSRF-Token': token, 'X-Timezone': Intl.DateTimeFormat().resolvedOptions().timeZone }
  });
  return resp.status;
}

// 1. Delete med-logs first (depends on medications)
for (const id of seededIds.medLogs) {
  await cleanupAPI('DELETE', `/babies/${babyId}/med-logs/${id}`);
}

// 2. Deactivate then we can leave medications (no delete endpoint)
//    Or if we created the baby, unlinking will cascade-delete everything
for (const id of seededIds.medications) {
  // Can't delete medications, but if we delete the baby they go too
}

// 3. Delete all metric entries
for (const type of ['feedings', 'stools', 'urine', 'temperatures', 'weights', 'abdomen', 'skin', 'bruising', 'labs', 'notes']) {
  for (const id of (seededIds[type] || [])) {
    await cleanupAPI('DELETE', `/babies/${babyId}/${type}/${id}`);
  }
}

// 4. If we created the test baby, unlink (cascade-deletes baby + all remaining data)
if (createdBaby) {
  await cleanupAPI('DELETE', `/babies/${babyId}/parents/me`);
}

// 5. Clear sessionStorage
sessionStorage.removeItem('ui_test_seeded');
```

### Cleanup Verification

After cleanup, verify the database is clean:
- If baby was created by us: `GET /api/babies` should not contain the test baby
- If using existing baby: `GET /api/babies/{id}/feedings?from=TODAY&to=TOMORROW` should not contain seeded entries

Report cleanup status.

## Step 7: Collect and Report Results

After all subagents complete and cleanup is done, compile the unified summary:

```
# UI Test Results -- [date] -- [TARGET_URL]

## Summary
- Total: X tests
- Passed: X
- Failed: X
- Skipped: X

## Cleanup
- Seeded entries deleted: X
- Test-created entries deleted: X
- Database state: CLEAN / PARTIAL (details)

## Results

| Test ID | Name | Result | Notes |
|---------|------|--------|-------|
| ... | ... | ... | ... |

## Failures (Details)

### TC-XX-XX: Test Name
- **Expected:** ...
- **Actual:** ...
- **Screenshot:** (taken on failure)
```

If any tests failed, ask the user if they'd like to re-run the failed tests.
