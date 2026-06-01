# skill:evolve-skills (Skill Evolution Decision Logic)

Use when reviewing the .rick/skills/ directory to decide which skills to keep, upgrade, or deprecate.

## Evolution Decision Logic

Decisions are based on `run_log` statistics for each skill:

### 保留（Keep）
Criteria: trigger_count ≥ 3 AND error_count < (trigger_count / 3)

The skill is working — it's applied frequently and rarely causes errors. Keep it as-is.

### 升级（Upgrade）
Criteria: The skill is effective (produces correct results) BUT the description is unclear, incomplete, or hard to understand when re-reading.

Signs that upgrade is needed:
- The trigger scenario is too vague (skill gets applied in wrong situations)
- The core content lacks concrete steps or examples
- The expected effect is unmeasurable

Action: Rewrite the description/content to be more precise. Do NOT change what the skill does, only how it's documented.

### 淘汰（Deprecate）
Criteria: trigger_count = 0 OR error_count ≥ (trigger_count / 2)

Two failure modes:
- trigger_count = 0: The skill was never needed, or the trigger scenario doesn't match real usage → remove
- error_count ≥ trigger_count/2: The skill causes errors more than half the time → it's harmful, remove or rewrite from scratch

## Audit Process

```
1. List all skills in .rick/skills/index.md
2. For each skill, read its run_log entry
3. Apply the decision logic above
4. For "upgrade" skills: rewrite in-place, bump version comment
5. For "deprecate" skills: remove file, update index.md
6. Run tests to ensure no broken references
```

## Edge Cases
- New skills (trigger_count = 0, just created): exempt from deprecation for 3 runs
- Skills with no run_log entry: treat as trigger_count = 0
