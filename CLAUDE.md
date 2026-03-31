# CLAUDE.md

## Project

LittleLiver — post-Kasai baby health tracking app. See `docs/SPEC.md` for full product details (stack, API, schema, metrics, deployment, phases). The spec is a living document — always read it for current project details.

---

## Development Methodology

### Red-Green TDD (STRICT)

This project follows **strict Red-Green-Refactor TDD**. Non-negotiable.

1. **RED** — Write a failing test first. Must fail for the right reason (assertion failure, not compile error).
2. **GREEN** — Write the **minimal** code to make the test pass.
3. **REFACTOR** — Clean up while keeping all tests green. No new behavior.

**Rules:**
* Never write production code without a failing test first
* Never write more production code than needed to pass the current failing test
* **All tests must be green before any commit** — no exceptions
* **Target 90%+ code coverage** across backend and frontend
* Run the full test suite before committing

### Test Commands

```bash
# Backend
cd backend && go test ./... -v -cover

# Frontend
cd frontend && npm test
```

### Bug Fixes

1. Write a test that reproduces the bug (fails)
2. Fix the bug (test passes)
3. Refactor if needed

### Worktree Workflow

Always work in a **git worktree** to avoid disturbing other agents working in the main tree. After completing work: commit, push the branch, then merge to main.

---

## Code Conventions

* **Go:** gofmt, early return style, `cmd/` for entrypoints, `internal/` for private packages
* **TypeScript:** strict mode
* **SQL:** parameterized queries only
* **Commits:** conventional commits (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`)
* **Dependencies:** Go stdlib preferred. Only add external deps when clearly justified.
* **Linting:** `go vet` for Go; ESLint + Prettier for frontend

---

## Constraints

* Do not modify `docs/SPEC.md` unless explicitly asked
* Do not suggest stack replacements — the stack is locked
* Minimal dependencies, no over-engineering — this is a personal-use app
