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
}

// buildOrchestrationSection renders the pi workflowScript orchestration for the
// pending (non-success) tasks in topological order. When doing/tasks.json does
// not exist yet, it falls back to scanning plan/task*.md (all pending).
func buildOrchestrationSection(doingDir, planDir string) string {
	tasks := pendingOrchTasks(doingDir, planDir)
	ids, err := topoSortOrch(tasks)
	if err != nil {
		// A cycle in the pending set is a plan error; surface it inline rather
		// than failing prompt generation so the agent can report it.
		return fmt.Sprintf("## pi 编排（workflowScript）\n\n无法生成编排：依赖存在环（%v）。请先修正 plan 的依赖关系。\n", err)
	}
	if len(ids) == 0 {
		return "## pi 编排（workflowScript）\n\n所有 task 均已完成（status=success），无待执行的编排。\n"
	}
	return renderWorkflowSection(ids, planDir)
}

// pendingOrchTasks returns pending tasks (status != success) with dependencies,
// reading doing/tasks.json first and falling back to plan/task*.md.
func pendingOrchTasks(doingDir, planDir string) []orchTask {
	if doingDir != "" {
		if tj, err := workspace.LoadTasksJSON(filepath.Join(doingDir, "tasks.json")); err == nil {
			out := make([]orchTask, 0, len(tj.Tasks))
			for _, t := range tj.Tasks {
				if t.Status == "success" {
					continue
				}
				out = append(out, orchTask{ID: t.TaskID, Dependencies: t.Dependencies})
			}
			return out
		}
	}
	return scanPlanOrchTasks(planDir)
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
		out = append(out, orchTask{ID: id, Dependencies: parseDepsSection(string(data))})
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

// topoSortOrch is an inlined Kahn topological sort over the pending tasks.
func topoSortOrch(tasks []orchTask) ([]string, error) {
	if len(tasks) == 0 {
		return nil, nil
	}
	pending := make(map[string]bool, len(tasks))
	inDegree := make(map[string]int, len(tasks))
	dependents := make(map[string][]string, len(tasks))
	for _, t := range tasks {
		pending[t.ID] = true
	}
	for _, t := range tasks {
		for _, d := range t.Dependencies {
			if pending[d] {
				inDegree[t.ID]++
				dependents[d] = append(dependents[d], t.ID)
			}
		}
	}
	var queue []string
	for _, t := range tasks {
		if inDegree[t.ID] == 0 {
			queue = append(queue, t.ID)
		}
	}
	sort.Strings(queue)

	var result []string
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		result = append(result, cur)
		for _, dep := range dependents[cur] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
		sort.Strings(queue)
	}
	if len(result) != len(tasks) {
		return nil, fmt.Errorf("cycle detected")
	}
	return result, nil
}

// renderWorkflowSection renders the workflowScript + runs.run orchestration for
// the given topological task order.
func renderWorkflowSection(ids []string, planDir string) string {
	var sb strings.Builder
	sb.WriteString("## pi 编排（workflowScript）\n\n")
	sb.WriteString("你是本次 job 的 parent 编排者（单写者）。必须调用 `subagent` 工具，把下面的 `workflowScript` 作为其 `workflowScript` 参数触发编排执行（一次触发，内部按依赖拓扑顺序执行，`await` 强制顺序，被依赖 task 先执行；每个 task 完成后由 worker 立即 commit 并把 commit_hash 回传给 parent 写入 tasks.json）。\n\n")
	sb.WriteString("```javascript\n")
	sb.WriteString("const workflowScript = `\n")
	for _, id := range ids {
		taskFile := filepath.Join(planDir, id+".md")
		sb.WriteString(fmt.Sprintf("const %s = await runs.run('%s', {agent:'worker', task:'按 %s 执行 %s，完成后 commit 并把 commit_hash 回传给 parent 写入 tasks.json'});\n",
			id, id, taskFile, id))
	}
	sb.WriteString("return {")
	for i, id := range ids {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(id + ": " + id + ".output")
	}
	sb.WriteString("};\n")
	sb.WriteString("`;\n")
	sb.WriteString("```\n\n")
	sb.WriteString("把上述 `workflowScript` 作为 `subagent` 工具调用的 `workflowScript` 参数触发编排执行。\n")
	return sb.String()
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
