# skill:debug (Systematic Debugging)

Use when encountering a bug, unexpected behavior, or test failure that is not immediately obvious.

## Four-Phase Systematic Debugging

### Phase 1: Information Collection（信息收集）
Gather all available evidence before forming hypotheses:
- Read the FULL error message and stack trace (not just the last line)
- Reproduce the failure consistently — if it's flaky, that's a separate bug
- Identify exact inputs and environment that trigger the failure
- Check recent changes: `git log --oneline -10`, `git diff HEAD~1`
- Collect: error message, stack trace, relevant logs, system state

**Output**: A precise problem statement: "When I call X with Y, I expect Z but get W"

### Phase 2: Hypothesis（假设）
Form testable explanations:
- List 2-3 candidate root causes ordered by likelihood
- Each hypothesis must be falsifiable (can be proven wrong)
- Consider: off-by-one, nil pointer, type mismatch, race condition, wrong config
- Don't fix anything yet — premature fixes hide the real cause

**Output**: "My top hypothesis is [X] because [evidence]. I can verify by [test]."

### Phase 3: Verification（验证）
Test one hypothesis at a time:
- Add targeted logging or print statements to confirm/deny
- Write a minimal reproduction case (smallest code that triggers the bug)
- Use debugger or unit test to isolate the failing component
- If hypothesis is wrong, eliminate it and test the next one

**Output**: Confirmed root cause with evidence: "The bug is [X] as proven by [observation]."

### Phase 4: Fix（修复）
Apply the minimal correct fix:
- Fix the root cause, not just the symptom
- Write a regression test that would have caught this bug
- Verify the fix with: original reproduction case passes, existing tests still pass
- Clean up any debug logging added in Phase 3

**Output**: Commit message: "fix: [what was wrong] - [how it was fixed]"

## Anti-patterns
- Guessing and randomly changing code without forming hypotheses
- Fixing the symptom (e.g., catching an exception) instead of the root cause
- Adding workarounds without understanding why the bug occurred
