package handler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// seedTasksJSON 造一个 doing/tasks.json（名字避开 learning_test.go 里既有的
// writeTasksJSON(t, dir, tasks) 辅助，避免同包符号冲突）。
func seedTasksJSON(t *testing.T, rickDir, jobID, body string) string {
	t.Helper()
	dir := filepath.Join(rickDir, "jobs", jobID, "doing")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "tasks.json")
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readStatuses(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Tasks []struct {
			TaskID string `json:"task_id"`
			Status string `json:"status"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, tk := range doc.Tasks {
		out[tk.TaskID] = tk.Status
	}
	return out
}

// TestNormalizeRunningTasks 覆盖「人工恢复 doing 前的归一化」：
// 只把遗留 running → pending（确定性门禁把遗留 running 判为 zombie，归一化否则
// 续跑第一轮就失败），success/error/pending 原样保留，幂等，且不动无关文件。
func TestNormalizeRunningTasks(t *testing.T) {
	rickDir := t.TempDir()
	path := seedTasksJSON(t, rickDir, "job_1", `{"version":"1.0","tasks":[
		{"task_id":"task1","status":"success","commit_hash":"abc"},
		{"task_id":"task2","status":"running"},
		{"task_id":"task3","status":"error"},
		{"task_id":"task4","status":"pending"},
		{"task_id":"task5","status":"running","attempts":2}
	]}`)

	moved, err := NormalizeRunningTasks(rickDir, "job_1")
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if len(moved) != 2 || moved[0] != "task2" || moved[1] != "task5" {
		t.Fatalf("moved = %v, want [task2 task5]", moved)
	}
	st := readStatuses(t, path)
	want := map[string]string{"task1": "success", "task2": "pending", "task3": "error", "task4": "pending", "task5": "pending"}
	for id, w := range want {
		if st[id] != w {
			t.Fatalf("%s status = %q, want %q（只应把 running 改为 pending）", id, st[id], w)
		}
	}
	// 备份存在（写盘前留痕）
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("缺 tasks.json.bak: %v", err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("落盘留下 .tmp 残留（应原子替换）")
	}
	// 未知字段保留（tasks.json 由 hook 写，字段面比本包更宽）
	data, _ := os.ReadFile(path)
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["version"] != "1.0" {
		t.Fatalf("顶层字段被丢弃: %v", raw)
	}

	// 幂等：已经无 running → 不再改文件（mtime 不变、不产生新备份内容）
	before, _ := os.Stat(path)
	moved2, err := NormalizeRunningTasks(rickDir, "job_1")
	if err != nil {
		t.Fatalf("second normalize: %v", err)
	}
	if len(moved2) != 0 {
		t.Fatalf("第二次 normalize moved = %v, want empty（幂等）", moved2)
	}
	after, _ := os.Stat(path)
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("无变化时不应重写 tasks.json（避免扰动 watcher/mtime）")
	}
}

// TestNormalizeRunningTasks_Errors 覆盖错误分支：参数缺失 / 文件不存在。
func TestNormalizeRunningTasks_Errors(t *testing.T) {
	if _, err := NormalizeRunningTasks("", "job_1"); err == nil {
		t.Fatal("空 rickDir 必须报错")
	}
	if _, err := NormalizeRunningTasks(t.TempDir(), ""); err == nil {
		t.Fatal("空 jobID 必须报错")
	}
	if _, err := NormalizeRunningTasks(t.TempDir(), "job_missing"); err == nil {
		t.Fatal("tasks.json 不存在必须报错（调用方据此提示无法恢复）")
	}
}
