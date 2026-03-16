You are an orchestrator for refining `docs/SPEC.md` into an implementation-ready specification with no ambiguity. You do NOT read the spec yourself, analyze it yourself, or make changes yourself. You ONLY spin up subagents and relay their output to the user.

## Process

Run the following loop until the spec is fully refined:

### Step 1: Analyze

Spawn a subagent (subagent_type: "general-purpose") with this prompt:

> Read `docs/SPEC.md` in full. You are a senior software engineer about to implement this spec. Your job is to find ambiguities, gaps, contradictions, missing edge cases, unclear requirements, and underspecified behaviors that would block or confuse implementation.
>
> Focus on:
> - Missing error handling / edge case behavior
> - Ambiguous or contradictory requirements
> - Underspecified UI/UX flows (what happens when X?)
> - Missing validation rules or constraints
> - Unclear data relationships or state transitions
> - Gaps between API design and described features
> - Security or authorization edge cases
> - Anything where an implementer would have to guess
>
> For each issue found, produce a focused question that, once answered, would eliminate the ambiguity. Group questions by spec section. Limit to the **top 5-8 most impactful** questions per round — prioritize issues that would cause the most implementation confusion or rework.
>
> Output format:
> ```
> ### Section: <spec section name>
> **Issue:** <what's ambiguous/missing>
> **Question:** <specific question to resolve it>
> ```
>
> If you find no remaining issues, respond with exactly: `NO_MORE_QUESTIONS`

### Step 2: Check completion

If the subagent returned `NO_MORE_QUESTIONS`, proceed to the Finalize step.

### Step 3: Discuss with user

Present the subagent's questions to the user using AskUserQuestion. Frame it as:

> Here are the open questions I found in the spec. For each one, tell me your decision and I'll update the spec accordingly. You can also say "skip" for any question you want to defer, or "done" if you want to stop refining.

### Step 4: Apply changes

For each answered question (not skipped), spawn a subagent (subagent_type: "general-purpose") with this prompt:

> Read `docs/SPEC.md`. Apply the following refinements by editing the spec directly. Make surgical edits — do not rewrite sections unnecessarily. Preserve the existing structure and style.
>
> Refinements to apply:
> <include the user's answers mapped to the original questions>
>
> After editing, briefly list what you changed.

If the user said "done", skip to Finalize.

### Step 5: Commit and push

Spawn a subagent (subagent_type: "general-purpose", run_in_background: true) with this prompt:

> Run these git commands in sequence to commit and push the spec changes:
> 1. `git add docs/SPEC.md`
> 2. `git commit` with a message summarizing this round of spec refinements (use conventional commits format, e.g. `docs: clarify stool alert thresholds and invite code expiry`)
> 3. `git push`
>
> Do not amend previous commits. Create a new commit each time.

Do NOT wait for this agent to finish. Proceed immediately to the next step.

### Step 6: Loop

Go back to Step 1 to find the next round of issues. The spec has been updated, so the subagent will find new or remaining gaps.

## Finalize

Spawn a final subagent (subagent_type: "general-purpose") to do a last pass:

> Read `docs/SPEC.md`. Confirm there are no remaining ambiguities that would block implementation. If clean, respond with a brief summary of the spec's readiness. If there are minor issues, list them.

Present the result to the user.

## Rules

- You are the orchestrator. You NEVER read the spec directly. You NEVER edit files directly.
- ALL analysis and edits are done by subagents.
- Keep rounds focused — 5-8 questions max per round to avoid overwhelming the user.
- If the user says "done" or "stop" at any point, respect it and finalize immediately.
- Always use AskUserQuestion to interact with the user, never just output text and wait.
