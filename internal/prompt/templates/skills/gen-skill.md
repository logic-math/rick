# skill:gen-skill (Generate Skill from Act-Path)

Use when extracting reusable skills from observed agent execution patterns in act-path logs.

## Skill Generation Format

A skill file must contain three sections:

### 触发场景（Trigger Scenario）
When should this skill be applied?
- Describe the concrete situation where this pattern is useful
- Use "Use when..." format
- Be specific: "Use when debugging a test that passes locally but fails in CI" not "Use when debugging"
- Include the problem signals that indicate this skill is needed

### 预期效果（Expected Effect）
What outcome does applying this skill produce?
- Measurable result: "reduces debugging time from hours to minutes"
- Quality signal: "produces a test that actually exercises the code path"
- Failure prevention: "prevents X type of mistake"

### 核心内容（Core Content）
The reusable procedure or framework:
- Step-by-step instructions if sequential
- Decision tree if conditional
- Checklist if parallel
- Include concrete examples, not just abstract principles
- Reference actual code patterns where relevant

## Extraction Protocol from Act-Path

```
1. Read act-path-{taskID}.md for the completed task
2. Identify repeated tool call patterns (3+ similar sequences)
3. Find error→fix sequences (what went wrong and how it was resolved)
4. Check if the pattern is generalizable (applicable beyond this specific task)
5. Write the skill in the three-section format above
6. Add the skill to .rick/skills/ with descriptive filename
```

## Quality Criteria
- Trigger is specific enough to know when to apply it
- Content is actionable (not just principles, but steps)
- Effect is observable (you can tell if it worked)
- Length: 100-300 words (concise but complete)
