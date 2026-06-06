# RFC: Go Codebase Refactor

**Status**: Pending  
**Date**: 2026-06-05  
**Trigger**: Deep code scan revealed divergence between Go source and `.rick/` context after three-layer restructure

---

## 1. Dead Code Cleanup — workspace injection layer

### Problem

After moving `tools/` into `.rick/tools/` and merging `skills/` into `wiki/`, the following Go code became dead (always returns empty / writes to non-existent paths):

| Symbol | File | Issue |
|--------|------|-------|
| `LoadToolsList(projectRoot)` | `internal/workspace/tools.go` | Scans `projectRoot/tools/` — path deleted, always returns `[]` |
| `LoadSkillsIndex(rickDir)` | `internal/workspace/skills.go` | Reads `rickDir/skills/index.md` — file deleted |
| `LoadSkillsList(rickDir)` | `internal/workspace/skills.go` | Scans `rickDir/skills/*.py` — directory deleted |
| `GenerateSkillsIndex` / `GenerateSkillsREADME` | `internal/workspace/skills.go` | Writes to `rickDir/skills/` — directory deleted |
| `SkillsDirName = "skills"` | `internal/workspace/paths.go` | Constant for deleted directory |
| `formatToolsSection` | `internal/prompt/doing_prompt.go` | Calls `LoadToolsList(projectRoot)` — always empty |
| `formatSkillsSection` | `internal/prompt/doing_prompt.go` | Calls `LoadSkillsIndex` — always empty |

### Fix

1. Delete `internal/workspace/skills.go` entirely (`SkillInfo`, `LoadSkillsList`, `LoadSkillsIndex`, `GenerateSkillsIndex`, `GenerateSkillsREADME`, `extractSkillDescription`)
2. Remove `SkillsDirName = "skills"` from `paths.go`
3. Update `LoadToolsList(projectRoot string)` → `LoadToolsList(rickDir string)`, change internal path to `filepath.Join(rickDir, "tools")`
4. Remove `formatSkillsSection` from `doing_prompt.go`; update `formatToolsSection` to pass `rickDir` instead of `projectRoot`
5. Clean up dead tests: `TestLoadSkillsList_*`, `TestGenerateSkillsIndex_*` in `skills_test.go`
6. Update `integration_rfc001_test.go` and `tests/tools_integration_test.sh` path assertions

---

## 2. Implement `rick tools merge`

### Problem

`rick tools merge <job_id>` is referenced in SPEC and wiki docs but never implemented. `tools.go` only registers `plan_check`, `doing_check`, `learning_check`, `dream_check`. No `tools_merge.go` exists.

### Fix

Create `internal/cmd/tools_merge.go` and register `NewMergeCmd()`:

**Flow:**
1. Check `learning/SUMMARY.md` first line is `<!-- APPROVED: true -->` (safety gate)
2. Get current git branch
3. Create + switch to `learning/job_N` branch
4. Copy `learning/wiki/*` → `.rick/wiki/`
5. Copy `learning/tools/*` → `.rick/tools/`
6. Copy `learning/OKR.md` → `.rick/OKR.md` (if exists)
7. Copy `learning/SPEC.md` → `.rick/SPEC.md` (if exists)
8. Regenerate `.rick/wiki/README.md`
9. `git commit -m "learning: merge job_N knowledge"`
10. Switch back to original branch; output structured summary

Human then runs: `git merge --no-ff learning/job_N && git branch -D learning/job_N`

---

## 3. RED Verification — implement or remove

### Problem

SPEC previously described a RED verification loop in `RunTask`:
- After test script generation, run the script **before** implementation
- If `pass == true` (unexpected green) → append RED warning to `debug.md`, regenerate test script
- Max 2 retries, then warn and proceed

`internal/executor/runner.go` has no such logic. The RED warnings visible in `executor/debug.md` are from test-suite artifacts, not production code.

### Fix Options

**Option A — Implement:** Insert RED check between `GenerateTestWithAgent` and `agentExecutor.Execute` in `RunTask`. Add `maxREDRetries = 2` constant and `appendREDWarning` helper. Add `TestRunTask_RED*` tests.

**Option B — Remove from scope:** Keep SPEC clean (already removed this cycle). Close RFC without action.

*Recommendation: implement Option A — it directly protects test quality and aligns with TDD embedded in `doing.md` prompts.*

---

## 4. Register or Remove `easy.go`

### Problem

`internal/cmd/easy.go` implements `runEasyMode(requirement string)` — a one-shot mode that skips plan/doing/learning separation. It has a corresponding `internal/prompt/easy_prompt.go`. Neither is registered in `root.go`, making both files unreachable dead code.

### Fix Options

**Option A — Register:** Add `NewEasyCmd()` to `easy.go` and register in `root.go`. Update SPEC with `rick easy` command spec.

**Option B — Delete:** Remove `easy.go` and `easy_prompt.go` if the one-shot mode is not part of the product roadmap.

---

## Suggested Job Breakdown

| Task | Estimated effort |
|------|-----------------|
| Dead code cleanup (items 1) | 1 job, ~5 tasks |
| Implement `rick tools merge` (item 2) | 1 job, ~4 tasks |
| RED verification (item 3) | 1 job, ~3 tasks |
| Decide + act on `easy.go` (item 4) | 1 task (decision) |
