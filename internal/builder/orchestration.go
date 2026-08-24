package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunquan/rick/internal/workspace"
)

// orchTask is the minimal task node for the doing orchestration: id + direct
// dependencies. It is computed at prompt-generation time (rick-side) because the
// pi workflowScript sandbox has no filesystem access — "skip completed" must be
// resolved before the prompt is assembled.
type orchTask struct {
	ID           string
	Dependencies []string
	WriteDomain  []string // # 写域 声明（目录以 / 结尾=前缀语义；空=未声明）
}

// parseWriteDomain extracts the "# 写域" section as a path list.
func parseWriteDomain(taskMD string) []string {
	section := extractSection(taskMD, "# 写域")
	var out []string
	for _, line := range strings.Split(section, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "[") || strings.HasPrefix(t, "#") {
			continue // 跳过说明性文字
		}
		t = strings.TrimPrefix(t, "- ")
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// writeDomainConflict deterministically reports whether two declared write
// domains overlap: path equality, or directory-prefix containment either way.
func writeDomainConflict(a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	if strings.HasSuffix(a, "/") && strings.HasPrefix(b, a) {
		return true
	}
	if strings.HasSuffix(b, "/") && strings.HasPrefix(a, b) {
		return true
	}
	return false
}

// levelWriteDomainGate deterministically validates same-layer write-domain
// disjointness: a layer with >1 task requires every task to declare a
// non-empty write domain, and all pairs must be conflict-free.
func levelWriteDomainGate(level []string, domains map[string][]string) error {
	if len(level) <= 1 {
		return nil // 单 task 层无需写域声明（向后兼容存量 plan）
	}
	for _, id := range level {
		if len(domains[id]) == 0 {
			return fmt.Errorf("同层 %s 未声明 # 写域（同层多 task 并行的前提：全员声明写域且两两不相交）——请补 plan/task%s.md 的 # 写域 节", id, strings.TrimPrefix(id, "task"))
		}
	}
	for i := 0; i < len(level); i++ {
		for j := i + 1; j < len(level); j++ {
			for _, a := range domains[level[i]] {
				for _, b := range domains[level[j]] {
					if writeDomainConflict(a, b) {
						return fmt.Errorf("同层写域冲突：%s 的 %q 与 %s 的 %q 重叠——补依赖使其分层，或收敛写域", level[i], a, level[j], b)
					}
				}
			}
		}
	}
	return nil
}

// buildOrchestrationSection renders the pi workflowScript orchestration for the
// pending (non-success) tasks in topological order. When doing/tasks.json does
// not exist yet, it falls back to scanning plan/task*.md (all pending).
func buildOrchestrationSection(doingDir, planDir string) string {
	tasks, satisfied := pendingOrchTasksWithSatisfied(doingDir, planDir)
	if len(tasks) == 0 {
		return "## pi 编排（workflowScript）\n\n所有 task 均已完成（status=success），无待执行的编排。\n"
	}
	// 拓扑门禁：环检测 + 依赖引用存在性。校验失败直接在编排段报错拒绝派发
	//（plan 错误不该留到运行时炸）。
	levels, err := topoLevelsOrch(tasks, satisfied)
	if err != nil {
		// 环是最小必要校验（编排无法渲染）；其余结构校验（写域互斥/gate 存在性）
		// 全部下沉到 rick-gates hook 的 pipeline_gate 工具（确定性逻辑集中在 hook）。
		return fmt.Sprintf("## pi 编排（workflowScript）\n\n⛔ 拓扑门禁失败：%v\n\n请先修正 plan 的依赖关系（task*.md 的 # 依赖关系 节），再重新 `rick doing`。\n", err)
	}
	section := renderPipelineGateStep(doingDir, planDir)
	return section + renderWorkflowSection(levels, planDir, doingDir)
}

// renderPipelineGateStep renders the pre-flight structural gate instruction:
// parent 调 pipeline_gate 工具（rick-gates hook 注册）做确定性结构校验
// （分层 DAG / 同层写域两两不相交 / gate{N}.py 存在），⛔ 才继续派发。
func renderPipelineGateStep(doingDir, planDir string) string {
	return fmt.Sprintf("## 第 0 步：流水线结构门禁（执行任何派发之前）\n\n调用 `pipeline_gate` 工具（rick-gates hook）：\n\n```\n{ doing_dir: %q, plan_dir: %q }\n```\n\nhook 确定性校验：分层 DAG 无环且依赖引用存在 / 同层多 task 全员声明 `# 写域` 且两两不相交 / 每层 `gate{N}.py` 存在。**⛔ 通过才继续步骤 ①**；失败按报错修正 plan 后重调。\n\n", doingDir, planDir)
}

// pendingOrchTasks returns pending tasks (status != success) with dependencies,
// reading doing/tasks.json first and falling back to plan/task*.md.
func pendingOrchTasks(doingDir, planDir string) []orchTask {
	tasks, _ := pendingOrchTasksWithSatisfied(doingDir, planDir)
	return tasks
}

// pendingOrchTasksWithSatisfied also returns the set of task ids that are
// already success (satisfied deps — legal references that must not trip the
// dependency-existence gate).
func pendingOrchTasksWithSatisfied(doingDir, planDir string) ([]orchTask, map[string]bool) {
	if doingDir != "" {
		if tj, err := workspace.LoadTasksJSON(filepath.Join(doingDir, "tasks.json")); err == nil {
			out := make([]orchTask, 0, len(tj.Tasks))
			satisfied := make(map[string]bool, len(tj.Tasks))
			for _, t := range tj.Tasks {
				if t.Status == "success" {
					satisfied[t.TaskID] = true
					continue
				}
				wd := readTaskWriteDomain(planDir, t.TaskID)
				out = append(out, orchTask{ID: t.TaskID, Dependencies: t.Dependencies, WriteDomain: wd})
			}
			return out, satisfied
		}
	}
	return scanPlanOrchTasks(planDir), map[string]bool{}
}

// readTaskWriteDomain reads the # 写域 section of plan/<id>.md (best effort —
// absent file or section returns nil).
func readTaskWriteDomain(planDir, id string) []string {
	data, err := os.ReadFile(filepath.Join(planDir, id+".md"))
	if err != nil {
		return nil
	}
	return parseWriteDomain(string(data))
}

// scanPlanOrchTasks scans plan/task*.md and extracts the # 依赖关系 section for
// each (all pending — no tasks.json yet).
func scanPlanOrchTasks(planDir string) []orchTask {
	entries, err := os.ReadDir(planDir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "task") || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	var out []orchTask
	for _, name := range names {
		id := strings.TrimSuffix(name, ".md")
		data, err := os.ReadFile(filepath.Join(planDir, name))
		if err != nil {
			out = append(out, orchTask{ID: id})
			continue
		}
		out = append(out, orchTask{ID: id, Dependencies: parseDepsSection(string(data)), WriteDomain: parseWriteDomain(string(data))})
	}
	return out
}

// parseDepsSection extracts the "# 依赖关系" section content and splits it into
// task ids (comma-separated). "无"/"无依赖"/"none"/"-" and empties are dropped.
func parseDepsSection(content string) []string {
	section := extractSection(content, "# 依赖关系")
	if section == "" {
		return nil
	}
	var deps []string
	for _, part := range strings.Split(section, ",") {
		t := strings.TrimSpace(part)
		if t == "" || isNoDepToken(t) {
			continue
		}
		deps = append(deps, t)
	}
	return deps
}

// extractSection returns the content under a markdown heading until the next
// same-or-higher-level heading (mirrors the deleted parser's minimal behavior).
func extractSection(content, heading string) string {
	lines := strings.Split(content, "\n")
	var out []string
	found := false
	level := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == heading {
			found = true
			level = headingLevel(trimmed)
			continue
		}
		if !found {
			continue
		}
		if strings.HasPrefix(trimmed, "#") && headingLevel(trimmed) <= level {
			break
		}
		out = append(out, line)
	}
	for len(out) > 0 && strings.TrimSpace(out[0]) == "" {
		out = out[1:]
	}
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

func headingLevel(line string) int {
	return len(line) - len(strings.TrimLeft(line, "#"))
}

func isNoDepToken(s string) bool {
	trimmed := strings.Trim(s, "()（）")
	switch strings.ToLower(trimmed) {
	case "无", "无依赖", "none", "null", "nil", "n/a", "na", "-":
		return true
	}
	return false
}

// topoLevelsOrch computes Kahn topological LEVELS over the pending tasks:
// level 0 = no pending dependencies; level n = all deps in levels < n.
// Same-level tasks are mutually independent — safe to fan out in parallel.
// Validates dependency references first: a dep that is neither pending nor
// already satisfied (success) is a plan error (gate failure).
func topoLevelsOrch(tasks []orchTask, satisfied map[string]bool) ([][]string, error) {
	if len(tasks) == 0 {
		return nil, nil
	}
	known := make(map[string]bool, len(tasks))
	for _, t := range tasks {
		known[t.ID] = true
	}
	// 依赖引用存在性门禁：依赖必须在 pending 集（层间约束）或 satisfied 集
	//（已完成，忽略）——两者都不是 = plan 引用了不存在的 task。
	for _, t := range tasks {
		for _, d := range t.Dependencies {
			if !known[d] && !satisfied[d] {
				return nil, fmt.Errorf("task %s 依赖不存在的 %q（plan/task*.md 的 # 依赖关系 引用错误）", t.ID, d)
			}
		}
	}
	inDegree := make(map[string]int, len(tasks))
	dependents := make(map[string][]string, len(tasks))
	for _, t := range tasks {
		for _, d := range t.Dependencies {
			if satisfied[d] {
				continue // 已完成的依赖不构成层间约束
			}
			inDegree[t.ID]++
			dependents[d] = append(dependents[d], t.ID)
		}
	}
	var level []string
	for _, t := range tasks {
		if inDegree[t.ID] == 0 {
			level = append(level, t.ID)
		}
	}
	sort.Strings(level)
	var levels [][]string
	for len(level) > 0 {
		levels = append(levels, level)
		var next []string
		for _, cur := range level {
			for _, dep := range dependents[cur] {
				inDegree[dep]--
				if inDegree[dep] == 0 {
					next = append(next, dep)
				}
			}
		}
		sort.Strings(next)
		level = next
	}
	seen := 0
	for _, lv := range levels {
		seen += len(lv)
	}
	if seen != len(tasks) {
		return nil, fmt.Errorf("cycle detected")
	}
	return levels, nil
}

// estimateTimeoutMs estimates a per-task worker timeout from the task's own
// workload signal (v4.3.1 动态超时): 关键结果条数 × 8min + 测试场景数 × 4min
// + base 15min, clamped to [20min, 90min]. Falls back to 45min when the task
// file is unreadable.
func estimateTimeoutMs(taskFile string) int {
	const min, max, base = 20 * 60_000, 90 * 60_000, 15 * 60_000
	data, err := os.ReadFile(taskFile)
	if err != nil {
		return 45 * 60_000
	}
	s := string(data)
	krCount := countListItems(extractSection(s, "# 关键结果"))
	testCount := countListItems(extractSection(s, "# 测试方法"))
	ms := base + krCount*8*60_000 + testCount*4*60_000
	if ms < min {
		ms = min
	}
	if ms > max {
		ms = max
	}
	return ms
}

// countListItems counts markdown list entries (`- ` / `1. `) in a section.
func countListItems(section string) int {
	if section == "" {
		return 0
	}
	n := 0
	for _, line := range strings.Split(section, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") ||
			(strings.HasPrefix(t, "1. ") && !strings.Contains(t, "#")) {
			n++
		}
	}
	return n
}

// renderWorkflowSection renders the layered-pipeline orchestration prompt
// (v4.4.2 测试收敛到层门禁). 每层 4 步：
//   ① 门禁判别力验证：跑 gate{N}.py 应为红（该层模块集成测试此刻必然失败
//      ——门禁有判别力的证明；门禁携带的测试是模块级集成测试，human 确认）
//   ② runs.all 并行 impl-worker：按 # 任务目标/# 关键结果 实现，按
//      # 测试方法 **自测**（过程性验证，不落盘专门测试脚本；测试资产只有
//      门禁的模块集成测试）；写域互斥；不碰 git
//   ③ 层门禁提交：parent 调 level_complete（gate_cmd 必填）→ hook 跑
//      gate{N}.py → 绿 → 单次 commit → tasks.json 批量
//   ④ debug 压缩传递：该层 debug 摘要注入下层 worker（前层教训）
// loop 与 pipeline 正交：worker 按匹配的项目 loop 执行；parent 结对导航。
func renderWorkflowSection(levels [][]string, planDir, doingDir string) string {
	var sb strings.Builder
	sb.WriteString("## pi 编排（doing pipeline：分层 DAG + 层门禁）\n\n")
	sb.WriteString(fmt.Sprintf("你是本次 job 的 parent 编排者（结对导航员）。task 已按依赖拓扑分为 %d 层 pipeline（层间递进，层内写域独立可并行）。**逐层执行**以下 4 步。测试资产只有一层：每层门禁（gate{N}.py）携带的**模块级集成测试**（human 在 plan 阶段确认）；task 级只做自测（过程性）。\n\n", len(levels)))

	for li, level := range levels {
		gatePath := filepath.Join(planDir, "gates", fmt.Sprintf("gate%d.py", li+1))
		sb.WriteString(fmt.Sprintf("### 第 %d 层（互相独立、写域互斥的 task：%s）\n\n", li+1, strings.Join(level, ", ")))

		// 步骤 ① 门禁判别力验证
		sb.WriteString(fmt.Sprintf("**步骤 ① 门禁判别力验证**：先跑本层门禁 `python3 %s`——此时应**失败（红）**：本层模块集成测试此刻必然失败，这是门禁有判别力的证明（若一开始就绿：门禁无效或本层已完成，停下来查）。红 → 继续步骤 ②。\n\n", gatePath))

		// 步骤 ② impl-worker 并行（含自测）
		sb.WriteString(fmt.Sprintf("**步骤 ② 并行派发 impl-worker**（写域 = 各 task 的 # 写域 声明，两两互斥；**不碰 git、不调用任何提交工具**；若该项目有匹配的 loop（先验知识区），按其工作方法执行）：\n\n```javascript\nconst workflowScript = `\nconst L%d_impl = await runs.all([\n", li))
		for _, id := range level {
			taskFile := filepath.Join(planDir, id+".md")
			tmo := estimateTimeoutMs(taskFile)
			sb.WriteString(fmt.Sprintf("  { key: '%s-impl', agent: 'worker', timeoutMs: %d, task: '实现：按 %s 的 # 任务目标/# 关键结果 写实现（只写 # 写域 声明的路径），并按 # 测试方法 **自测**（过程性验证——自测代码可写在写域内随交付，或跑通即弃；不落盘到共享测试目录）。自测全绿后回执：改动文件清单 + 自测结果。禁止 git 操作' },\n",
				id, tmo, taskFile))
		}
		sb.WriteString(fmt.Sprintf("]);\nreturn L%d_impl.map(r => r.output);\n`;\n```\n\n", li))

		// 步骤 ③ 层门禁提交
		taskListJSON := "[" + strings.Join(mapStr(level, func(s string) string { return `\"` + s + `\"` }), ", ") + "]"
		sb.WriteString(fmt.Sprintf("**步骤 ③ 层门禁提交**——你（parent）直接调用 `level_complete` 工具：\n\n```\n{ level_tasks: %s, doing_dir: %q, gate_cmd: \"python3 %s\", summary: \"<一句话本层摘要>\" }\n```\n\nhook 确定性执行该层 human 确认的门禁（模块集成测试）→ exit 0 → `git add -A` 单次 commit（feat(layer): %s）→ tasks.json 批量写 success+commit_hash。门禁失败 → 拒绝提交并输出失败详情（只需重派失败 task 的 impl-worker，修复后重试本步骤）。\n\n",
			taskListJSON, doingDir, gatePath, strings.Join(level, "+")))

		// 步骤 ④ debug 压缩传递
		if li+1 < len(levels) {
			sb.WriteString(fmt.Sprintf("**步骤 ④ debug 压缩传递**：把本层 `doing/debug/` 的新增 bug*.md 压缩为一行一条的摘要清单（现象+解法+来源文件名），作为**共享上下文**注入第 %d 层所有 worker 的 task 文本末尾（「前层教训」段），避免重复踩坑。\n\n", li+2))
		} else {
			sb.WriteString("**步骤 ④ debug 归档**：最后一层完成后，把全程 `doing/debug/` 摘要归档到 `doing/debug-summary.md`（learning 阶段的行为轨迹分析输入之一）。\n\n")
		}
	}

	sb.WriteString("**契约**：worker 永不手工 git commit；tasks.json 只由 hook 写；门禁（gate{N}.py 及其集成测试）是 human 确认的层验收唯一标准，agent 不得修改。\n\n")
	sb.WriteString("**监督（结对导航）**：步骤 ② 运行期间用 `{action:'status', view:'transcript', id:'<runId>'}` tail worker 实时轨迹；方向偏离用 `{action:'steer'}` 纠偏、死循环用 `{action:'stop'}` 止损重派。你不在执行层，而在全局层——保持过程符合全局目标。\n")
	return sb.String()
}

func mapStr(xs []string, f func(string) string) []string {
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = f(x)
	}
	return out
}

// EnsureTasksJSON writes the initial doing/tasks.json draft when it does not
// exist yet: task list + dependencies (from plan/task*.md) + status=pending.
// It is idempotent (existing tasks.json is left untouched).
func EnsureTasksJSON(doingDir, planDir string) error {
	if doingDir == "" || planDir == "" {
		return fmt.Errorf("doingDir and planDir cannot be empty")
	}
	tasksJSONPath := filepath.Join(doingDir, "tasks.json")
	if _, err := os.Stat(tasksJSONPath); err == nil {
		return nil
	}
	tasks := scanPlanOrchTasks(planDir)
	if len(tasks) == 0 {
		return fmt.Errorf("no task*.md files found in plan directory %s", planDir)
	}
	if err := os.MkdirAll(doingDir, 0755); err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("{\n  \"version\": \"1.0\",\n  \"tasks\": [\n")
	for i, t := range tasks {
		if i > 0 {
			b.WriteString(",\n")
		}
		deps := "[]"
		if len(t.Dependencies) > 0 {
			deps = "["
			for j, d := range t.Dependencies {
				if j > 0 {
					deps += ", "
				}
				deps += fmt.Sprintf("%q", d)
			}
			deps += "]"
		}
		b.WriteString(fmt.Sprintf("    {\"task_id\": %q, \"task_name\": %q, \"task_file\": %q, \"status\": \"pending\", \"dependencies\": %s, \"attempts\": 0}",
			t.ID, t.ID, t.ID+".md", deps))
	}
	b.WriteString("\n  ]\n}\n")
	return os.WriteFile(tasksJSONPath, []byte(b.String()), 0644)
}
