---
name: ui-test
description: Run UI tests against the live LittleLiver app using Chrome browser automation
user-invocable: true
---

# UI Test Runner for LittleLiver

You are a UI test runner for the LittleLiver app. You use Claude Chrome browser extension to execute test cases and report results.

## Step 1: Load Test Cases

Read the file `/home/ab/projects/LittleLiver/docs/UI_TEST_CASES.md` to get the full list of test cases.

If the file does not exist, tell the user:
> "No test cases file found at `docs/UI_TEST_CASES.md`. Please create it first with your test case definitions. Each test case should have an ID (e.g., TC-01-01), a category, name, preconditions, steps, and expected results."

Then stop.

## Step 2: Parse Arguments

The user may provide an argument after `/ui-test`:

- **No argument** -> Run ALL test cases
- **Category name** (e.g., `dashboard`, `medications`, `navigation`, `trends`, `settings`, `metrics`, `entry`) -> Run only tests in that category (match case-insensitively against test case category/group names)
- **Specific test ID** (e.g., `TC-04-01`) -> Run only that single test case

## Step 3: Confirm with User

Use the `AskUserQuestion` tool to ask the user:

1. **Target URL**: "Which URL should I test against? (default: https://littleliver.fly.dev)"
2. **Auth status**: "Are you currently logged in to the app in Chrome? Tests that require authentication will need you to be logged in. (yes/no)"
3. If the user is NOT logged in and auth-dependent tests are selected: "Some tests require authentication. I'll ask you to log in via Google OAuth when needed. OK to proceed? (yes/no)"

## Step 4: Group and Launch Tests

Organize the selected test cases into these groups for parallel execution via the Agent tool:

- **Group A: Dashboard & Summary Cards** — Tests related to the today dashboard, summary cards, alerts, stool color warnings
- **Group B: Metric Entry Forms** — Tests for logging feedings, stools, urine, weight, abdomen, temperature entries
- **Group C: Navigation & Layout** — Tests for page routing, header, baby selector, responsive layout, login/logout flow
- **Group D: Trends & Reports** — Tests for trend charts, clinical summary, date range filtering
- **Group E: Medications** — Tests for medication CRUD, scheduling, dose logging
- **Group F: Settings & Baby Management** — Tests for baby profile editing, invite codes, unlinking, account deletion, default calorie settings

### Subagent Instructions Template

For EACH group that has tests to run, launch a subagent using the Agent tool. Give each subagent these instructions (customized with its specific test cases):

---

**SUBAGENT PROMPT:**

```
You are a UI test executor for LittleLiver. You will test the app at [TARGET_URL] using Chrome browser automation.

## CRITICAL: Loading Chrome Tools

Before using ANY mcp__claude-in-chrome__* tool, you MUST first load it via ToolSearch:
- Call ToolSearch with query "select:mcp__claude-in-chrome__<tool_name>"
- Only THEN call the actual tool

You must do this for EVERY Chrome tool before first use. The tools will not work otherwise.

## Available Chrome Tools

- mcp__claude-in-chrome__tabs_context_mcp — Get current tab context (ALWAYS call this first)
- mcp__claude-in-chrome__tabs_create_mcp — Create a new tab
- mcp__claude-in-chrome__navigate — Navigate to a URL
- mcp__claude-in-chrome__read_page — Read page content, check element presence
- mcp__claude-in-chrome__find — Find/locate elements on page
- mcp__claude-in-chrome__computer — Click, type, take screenshots, scroll
- mcp__claude-in-chrome__javascript_tool — Execute JavaScript (check localStorage, console, DOM state)
- mcp__claude-in-chrome__form_input — Fill form fields
- mcp__claude-in-chrome__get_page_text — Get all text content from page
- mcp__claude-in-chrome__read_console_messages — Check for JS errors
- mcp__claude-in-chrome__read_network_requests — Monitor API calls

## Setup

1. Load and call mcp__claude-in-chrome__tabs_context_mcp to get current browser state
2. Load and call mcp__claude-in-chrome__tabs_create_mcp to create a fresh tab for testing
3. Load and call mcp__claude-in-chrome__navigate to go to [TARGET_URL]

## Test Execution

For each test case below, execute these steps:
1. Navigate to the correct page/state
2. Perform the test actions (click, type, verify)
3. Verify expected results using read_page, javascript_tool, or screenshots
4. Record PASS or FAIL with details
5. On FAILURE: take a screenshot using mcp__claude-in-chrome__computer with action "screenshot" for debugging

## Handling Human-Required Actions

If a test step requires something you cannot automate (Google OAuth login, camera-based photo upload, push notification interaction), use the AskUserQuestion tool:
- Clearly describe what you need the user to do
- Wait for confirmation
- Then verify the result of their action

## Test Cases to Execute

[INSERT FULL TEST CASES HERE — include ID, name, preconditions, steps, and expected results for each test]

## Output Format

After running all tests, output your results in this exact format:

RESULTS_START
| Test ID | Name | Result | Notes |
|---------|------|--------|-------|
| TC-XX-XX | Test name | PASS/FAIL | Details or error description |
RESULTS_END
```

---

### Important Notes for Subagent Dispatch

- **Include the FULL test case text** in each subagent's prompt — subagents do NOT have access to the test cases file
- **Include the confirmed TARGET_URL** in each subagent prompt
- **Sequential dependencies**: If tests within a group depend on each other (e.g., "create entry" then "verify it appears on dashboard"), keep them in the same subagent and note they must run in order
- **Group B creates data** that Group A may need — if running both, launch Group B first or note to Group A that it should verify existing data rather than expecting specific entries
- Launch independent groups in parallel using separate Agent tool calls

## Step 5: Collect and Report Results

After all subagents complete, parse their RESULTS_START/RESULTS_END output and compile a unified summary:

```
# UI Test Results — [date] — [TARGET_URL]

## Summary
- Total: X tests
- Passed: X
- Failed: X
- Skipped: X

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
