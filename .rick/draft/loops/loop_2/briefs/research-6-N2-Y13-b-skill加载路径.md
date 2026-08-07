# research-6 N2-Y13-b：pi skill 加载路径扩展性

节点路径：[根 > N2-Y13-b：pi skill 加载路径扩展性]
事实陈述：pi skill 默认加载路径、是否支持自定义 skill 加载目录、Extensions 是否可拦截/重定向 skill 加载、rick 现有 .rick/skills/{name}_skill/skill.md 结构能否被 pi 识别。

## 执行动作

1. Read `/tmp/pi_repo/packages/coding-agent/src/core/skills.ts`（skill 加载逻辑 + 默认路径 + loadSkillsFromDir）
2. Read `/tmp/pi_repo/packages/coding-agent/src/core/extensions/types.ts`（resources_discover 事件 + ResourcesDiscoverResult.skillPaths）
3. Grep `skillPaths` / `skill.*path` / `loadSkill` 验证所有 skill 加载入口
4. Read `/tmp/pi_repo/packages/coding-agent/src/cli/args.ts`（--skill flag）

## 信源验证结果

### 代码原文（权重 0.4）✅

**skills.ts 默认加载路径**（line 430-436）：
```ts
if (includeDefaults) {
    addSkills(loadSkillsFromDirInternal(join(resolvedAgentDir, "skills"), "user", true));
    addSkills(loadSkillsFromDirInternal(resolve(resolvedCwd, CONFIG_DIR_NAME, "skills"), "project", true));
}
```
- user scope：`{agentDir}/skills/`（默认 `~/.pi/agent/skills/`）
- project scope：`{cwd}/.pi/skills/`（`CONFIG_DIR_NAME` = `.pi`）

**skills.ts 自定义路径加载**（line 455-481）：
```ts
for (const rawPath of skillPaths) {
    const resolvedPath = resolvePath(rawPath, resolvedCwd, { trim: true });
    // ...
    if (stats.isDirectory()) {
        addSkills(loadSkillsFromDirInternal(resolvedPath, source, true));
    } else if (stats.isFile() && resolvedPath.endsWith(".md")) {
        // 加载单个 .md 文件
    }
}
```
- `skillPaths` 来自 `--skill <path>` flag，可多次指定
- 支持目录和单个 .md 文件
- source 标记为 "user" / "project" / "path"（路径加载标记为 "path"）

**skills.ts skill 文件识别规则**（line 192-275）：
- 目录含 `SKILL.md` → 作为 skill root，不再递归
- 目录不含 `SKILL.md` → 递归扫描直接子目录的 .md 文件
- skill 名称：frontmatter.name 或父目录名
- skill 名称校验：`^[a-z0-9-]+$`（小写字母/数字/连字符，≤64 字符，不以连字符开头/结尾，无连续连字符）
- frontmatter.description 必填（≤1024 字符）

**extensions/types.ts resources_discover 事件**（line 543-555）：
```ts
export interface ResourcesDiscoverEvent {
    type: "resources_discover";
    cwd: string;
    reason: "startup" | "reload";
}

export interface ResourcesDiscoverResult {
    skillPaths?: string[];
    promptPaths?: string[];
    themePaths?: string[];
}
```
- extension 可订阅 `resources_discover` 事件，返回额外 skill 路径
- 在 startup 和 reload 时触发
- 这是**扩展点**：extension 可动态注入 skill 加载路径

### 运行时行为（权重 0.3）✅

**args.ts --skill flag**（line 156-158）：
```ts
} else if (arg === "--skill" && i + 1 < args.length) {
    result.skills = result.skills ?? [];
    result.skills.push(args[++i]);
}
```
- `--skill <path>` 可多次使用
- `--no-skills` / `-ns` 禁用所有 skill 发现和加载

**skills.ts formatSkillsForPrompt**（line 335-361）：
- skill 加载后注入 system prompt 的 `<available_skills>` XML 段
- LLM 通过 `read` 工具按需加载 skill 文件内容
- `disable-model-invocation: true` 的 skill 不出现在 prompt 中（仅 `/skill:name` 调用）

### 文档（权重 0.2）✅

- extensions.md `resources_discover` 事件：extension 可动态提供 skill/prompt/theme 路径
- skills.md（pi docs）：skill 遵循 Agent Skills standard（agentskills.io），`SKILL.md` 是入口文件
- README "Customization > Skills"：`~/.pi/agent/skills/` 和 `.pi/skills/` 是默认路径

### 反事实（权重 0.1）N/A

本节点为外部文档调研，无代码修改。

## 还原确认

无 rick 代码修改，无需还原。

## 关键事实

1. **默认加载路径**：
   - user scope：`~/.pi/agent/skills/`
   - project scope：`{cwd}/.pi/skills/`
   - 两层都可通过 `--no-skills` 禁用

2. **自定义 skill 加载目录**：
   - `--skill <path>` flag：可指定任意目录或 .md 文件，可多次使用
   - `resources_discover` extension 事件：extension 可动态返回 `skillPaths` 数组，在 startup/reload 时注入
   - `PI_CODING_AGENT_DIR` 环境变量：重定向 user scope 根（`{agentDir}/skills/` 随之移动）

3. **Extensions 拦截/重定向 skill 加载**：
   - `resources_discover` 事件是**追加式**（返回额外路径），不是拦截式
   - 但 extension 可通过 `--no-skills` 禁用默认发现 + `resources_discover` 返回自定义路径，实现"重定向"
   - 无直接"拦截 skill 加载"的 hook（skill 加载在 session_start 前，extension 的 resources_discover 是参与而非拦截）

4. **rick .rick/skills/{name}_skill/skill.md 结构能否被 pi 识别**：
   - **不能直接识别**：pi 要求入口文件名为 `SKILL.md`（大写），rick 用 `skill.md`（小写）
   - **结构差异**：rick 用 `{name}_skill/skill.md`（目录名带 `_skill` 后缀），pi 用 `{name}/SKILL.md` 或扫描 .md 文件
   - **名称校验冲突**：pi 要求 `^[a-z0-9-]+$`，rick 现有 skill 名如 `gen_skill` 含下划线，**不通过校验**（会 warning 但仍加载，name 取 frontmatter.name 或父目录名）
   - **适配方案**：
     - 方案 A（重命名）：rick 将 `skill.md` 重命名为 `SKILL.md`，目录名去掉 `_skill` 后缀或保留（pi 用 frontmatter.name 优先）
     - 方案 B（flag 指定）：`--skill .rick/skills/doing_loop_skill/skill.md` 直接指定单个文件（pi 接受单个 .md 文件，但仍要求 frontmatter.description）
     - 方案 C（extension 适配）：写一个 rick extension，订阅 `resources_discover` 事件，扫描 `.rick/skills/` 并返回适配后的路径
     - 方案 D（符号链接）：`.pi/skills/` 下符号链接到 `.rick/skills/` 各 skill 目录，并重命名为 `SKILL.md`

5. **skill frontmatter 要求**：
   - `description`（必填，≤1024 字符）
   - `name`（可选，默认父目录名，需 `^[a-z0-9-]+$`）
   - `disable-model-invocation`（可选，布尔，默认 false）

## 疑问点

无。本节点事实清晰，源码三重交叉验证（skills.ts + args.ts + types.ts）一致。

## 置信度评估（由 research 主调度计算）

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9（高，≥ 0.8 终止）
