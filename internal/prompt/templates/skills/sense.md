# skill:sense

Use when you need to deeply analyze a problem, requirement, or situation before acting.

## SENSE Framework

### Subject（主体）
Identify who is doing this and why:
- Who are the key actors (user, system, team)?
- What are their motivations and goals?
- What constraints or pressures are they operating under?

### Perspective（视角）
Examine the problem from multiple angles:
- User perspective: what do they actually need vs. what they asked for?
- System perspective: technical constraints, dependencies, performance
- Business perspective: cost, timeline, risk tolerance
- Maintenance perspective: future developers reading this code

### Judgment（判断）
Draw conclusions based on evidence:
- Synthesize observations from Subject + Perspective
- Identify the core problem (not just the surface symptom)
- Propose a concrete action or design decision
- Justify why this is the right approach given the constraints

### Critique（批判）
Challenge your own judgment:
- What assumptions am I making that could be wrong?
- What edge cases or failure modes am I ignoring?
- What's the worst-case outcome if my judgment is incorrect?
- Is there a simpler solution I'm overlooking?

## Usage Pattern

```
S: [Who is doing X and why]
E: [From role/constraint perspective, the situation looks like...]
N: [Therefore, the right approach is...]
S: [But this could be wrong if...]
```
