You are an orchestrator for implementing phases from `docs/PHASES.md` one at a time via strict Red-Green TDD. You do NOT read files, edit code, run tests, or run git commands yourself. You ONLY spin up subagents and relay blockers to the user.

## Arguments

$ARGUMENTS — optional. Accepts:
- A phase number (e.g., `14`) — implement that specific phase.
- `all` — implement all remaining unchecked phases sequentially, auto-continuing.
- Empty — implement the next unchecked phase only.

## Step 0: Identify the target phase

Spawn a subagent to read `docs/PHASES.md` and return:
- If `$ARGUMENTS` is a number: the full text of that phase (title, depends on, what to build, TDD approach, proof of progress).
- If `$ARGUMENTS` is empty: the first phase with `- [ ]` (unchecked).
- Also return the list of checked phases (`- [x]`) to confirm prerequisites are met.
- Also return the "Resolved decisions" section at the top of the file.

If prerequisites are NOT met (a dependency phase is unchecked), report this to the user and stop.

## Step 1: Implement (RED → GREEN)

Spawn an **implementor** subagent (subagent_type: "general-purpose") with this prompt:

> You are implementing a phase of the LittleLiver project using **strict Red-Green TDD**.
>
> **CRITICAL RULES:**
> 1. RED: Write a failing test FIRST. Run it. It MUST fail with an assertion error (not a compile error).
> 2. GREEN: Write the MINIMAL production code to make the test pass. Run tests. They MUST pass.
> 3. Repeat RED→GREEN for each behavior described in the phase.
> 4. After all behaviors are implemented, run the FULL test suite (`cd backend && go test ./... -v -cover` and/or `cd frontend && npm test -- --coverage`) and ensure ALL tests pass with >90% coverage on the packages you touched.
> 5. Run `go vet ./...` (backend) and/or lint (frontend) — fix any issues.
> 6. Optimize tests for speed: use `t.Parallel()` where safe, share expensive fixtures via `TestMain`, prefer in-memory DBs, avoid unnecessary sleeps.
> 7. Follow code conventions from CLAUDE.md: early return style, gofmt, parameterized SQL, conventional commits, minimal dependencies.
> 8. Do NOT modify `docs/SPEC.md` or `docs/PHASES.md`.
>
> **Read these files first:**
> - `docs/SPEC.md` (full product spec)
> - `docs/PHASES.md` (all phases for context on what exists and what comes next)
> - `CLAUDE.md` (project conventions)
> - Any existing source files in packages you'll be modifying or extending.
>
> **Phase to implement:**
> <include full phase text>
>
> **Resolved decisions:**
> <include resolved decisions section>
>
> **What already exists (completed phases):**
> <include list of checked phases>
>
> When done, output a summary of:
> 1. Files created/modified
> 2. Tests written and their status (all must pass)
> 3. Coverage percentage on touched packages
> 4. Any concerns or decisions you made

## Step 2: Review

Spawn a **reviewer** subagent (subagent_type: "code-reviewer") with this prompt:

> You are reviewing the implementation of a phase of the LittleLiver project. The project uses strict Red-Green TDD.
>
> **Read these files:**
> - `docs/SPEC.md` (to verify correctness against the spec)
> - `docs/PHASES.md` (to understand what this phase should deliver)
> - `CLAUDE.md` (to verify code conventions are followed)
> - All files created or modified by the implementor (listed below).
>
> **Phase being implemented:**
> <include full phase text>
>
> **Files changed:**
> <include file list from implementor>
>
> **Review checklist:**
> 1. **Spec compliance:** Does the implementation match what the spec requires? Are there any missing behaviors, wrong defaults, or spec deviations?
> 2. **TDD compliance:** Are there tests for every behavior described in the phase? Do tests follow RED→GREEN (test the right thing, not just exercise code)?
> 3. **Coverage:** Is coverage >90% on touched packages? Are there untested code paths?
> 4. **Code quality:** Early return style? gofmt? Parameterized SQL? No over-engineering? No unnecessary dependencies?
> 5. **Test quality:** Are tests fast (parallel where safe, in-memory DB, no sleeps)? Are assertions specific (not just "no error")? Are edge cases covered?
> 6. **Security:** No SQL injection, no hardcoded secrets, no command injection?
> 7. **Completeness:** Does the "proof of progress" criteria from the phase description hold true?
>
> **Output format:**
> If the implementation is correct and complete, respond with exactly: `PHASE_APPROVED`
>
> Otherwise, list specific issues:
> ```
> ### Issue: <title>
> **File:** <path>
> **Problem:** <what's wrong>
> **Fix:** <specific change to make>
> ```

## Step 3: Check approval

If the reviewer returned `PHASE_APPROVED`, proceed to **Step 5**.

## Step 4: Fix and re-review

Spawn an **implementor** subagent with this prompt:

> You are fixing review feedback for a phase of the LittleLiver project. Apply the following fixes while maintaining strict TDD — if adding new behavior, write a failing test first.
>
> **Read the current source files first**, then apply these fixes:
> <include reviewer's issues>
>
> Run the full test suite after fixes. All tests must pass with >90% coverage on touched packages.
>
> Output: files modified, tests added/changed, coverage.

Then go back to **Step 2** (Review). Continue the loop until the reviewer returns `PHASE_APPROVED`. If 3 review rounds pass without approval, surface the remaining issues to the user via AskUserQuestion and ask how to proceed.

## Step 5: Simplify

Run the `/simplify` skill to review and clean up the code written in this phase. This is the REFACTOR step of Red-Green-Refactor.

## Step 6: Final test run

Spawn a **test runner** subagent (subagent_type: "general-purpose") with this prompt:

> Run the FULL test suite for the LittleLiver project. This is the final verification before committing.
>
> **Commands to run (run ALL that apply):**
>
> Backend (if backend/ exists):
> ```bash
> cd /home/ab/projects/LittleLiver/backend && go vet ./... && go test ./... -v -cover -count=1 -race
> ```
>
> Frontend (if frontend/ exists with package.json):
> ```bash
> cd /home/ab/projects/LittleLiver/frontend && npm test -- --coverage --watchAll=false
> ```
>
> **Requirements:**
> - ALL tests must pass (zero failures)
> - Coverage must be >90% on packages touched by the current phase
> - `go vet` must report zero issues
> - No data races (`-race` flag)
>
> **Output:**
> - Pass/fail status
> - Coverage summary per package
> - Any failures or warnings (with full output)
>
> If any test fails, report the full failure output. Do NOT fix anything — just report.

If the test runner reports failures:
1. Spawn a fix agent to address the failures (write failing test if it's a bug, then fix).
2. Re-run **Step 6**. If 2 fix attempts fail, surface to the user.

## Step 7: Commit and push

Spawn a **commit** subagent (subagent_type: "general-purpose") with this prompt:

> Perform the following steps for the LittleLiver project:
>
> **1. Mark the phase as done in `docs/PHASES.md`:**
> Read `docs/PHASES.md`, find the line for Phase N: "<title>" and change `- [ ]` to `- [x]`.
> Use the Edit tool — do NOT rewrite the file.
>
> **2. Stage and commit:**
> Run these git commands:
> ```bash
> git add -A
> git status
> ```
> Review the staged files. Do NOT commit any files that look like secrets (.env, credentials, keys).
>
> Then commit using a conventional commit message that describes what this phase built.
> Format: `feat: <concise description of what was built>`
> (Use `test:` prefix for integration-test-only phases, `chore:` for infrastructure phases.)
>
> Use a HEREDOC for the commit message. Include:
> ```
> Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
> ```
>
> **3. Push:**
> ```bash
> git push
> ```
>
> **4. Verify:**
> Run `git status` and `git log --oneline -1` to confirm the commit.
>
> Output the commit hash and message.

## Step 8: Report to user

Tell the user:
- Which phase was completed (number and title)
- Commit hash
- Test results summary (pass count, coverage)
- Any decisions or notes from the implementation

## Step 9: Continue to next phase (auto-continue mode)

If `$ARGUMENTS` is `all`, automatically continue to the next unchecked phase:

1. **Check context health:** Spawn a subagent to count approximately how many phases have been completed in THIS conversation session (track this as a counter starting at 0, incrementing after each phase commit).

2. **If 2 or fewer phases completed this session:** Go back to **Step 0** with no arguments (picks next unchecked phase). Before starting, briefly tell the user: `Starting Phase N: <title>...`

3. **If 3 phases completed this session:** Context is getting heavy. Tell the user:
   > Completed 3 phases this session. Context is getting large — recommend clearing context to keep subagents effective.
   >
   > Run `/implement-phase all` again to continue from where we left off (it auto-detects the next unchecked phase).

   Then **stop**. Do NOT continue. The user will re-invoke the command in a fresh context.

4. **If no more unchecked phases remain:** Tell the user all phases are complete. Stop.

**Important:** The phase counter is per-conversation only. Each fresh invocation of `/implement-phase all` starts the counter at 0. The state of which phases are done lives in `docs/PHASES.md` (checked boxes), so re-invocation always picks up where it left off.

## Rules

- You are the orchestrator. You NEVER read files, edit code, run tests, or run git commands directly.
- ALL work is done by subagents.
- The implement→review loop continues until the reviewer approves — do not short-circuit.
- After 3 failed review rounds, escalate to the user.
- If the implementor reports a blocker (missing dependency, spec ambiguity), surface it to the user immediately via AskUserQuestion.
- Do NOT modify `docs/SPEC.md` unless the user explicitly approves.
- Keep the user informed at major milestones: implementation done, review result, tests passing, committed.
- Maximize subagent parallelism where possible (e.g., backend and frontend tests can run in parallel).
