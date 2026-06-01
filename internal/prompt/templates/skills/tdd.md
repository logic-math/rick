# skill:tdd (Test-Driven Development)

Use when implementing any new function, feature, or bug fix that requires code changes.

## The Iron Law: RED → GREEN → REFACTOR

### RED: Write a Failing Test First
- Write the test BEFORE writing implementation code
- The test must fail for the right reason (not compile error, not wrong assertion)
- Test name should document the behavior: `TestUserService_GetByID_ReturnsNotFoundWhenMissing`
- One behavior per test — do not test multiple scenarios in one test function

### GREEN: Write the Minimum Code to Pass
- Write ONLY enough code to make the failing test pass
- Do not add features not required by current tests
- It's OK if the code is ugly or duplicated at this stage
- Run the test — it must pass (green) before proceeding

### REFACTOR: Clean Up Without Breaking Tests
- Remove duplication, improve naming, extract functions
- Run tests after EVERY change — if anything breaks, revert
- Refactor implementation AND tests separately, never both at once
- When done: all tests still pass

## Cycle Discipline

```
1. Write ONE failing test
2. Run test → must see RED
3. Write minimum code
4. Run test → must see GREEN
5. Refactor
6. Run test → must still be GREEN
7. Repeat
```

## Critical Rules
- Never write implementation before tests
- Never skip the RED phase (if test passes immediately, the test is wrong)
- Never refactor when tests are RED
- Test the behavior (what), not the implementation (how)
