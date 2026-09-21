package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// rsiTestWorkspace builds a throwaway workspace (<tmp>/.rick/jobs/job_1/doing/rsi)
// and chdirs into it (that's what workspace.GetJobDoingDir resolves against).
// Returns (workspace root, doing dir). Always restores cwd and the probe seam.
func rsiTestWorkspace(t *testing.T) (string, string) {
	t.Helper()
	tmp := t.TempDir()
	doing := filepath.Join(tmp, ".rick", "jobs", "job_1", "doing")
	if err := os.MkdirAll(filepath.Join(doing, "rsi"), 0o755); err != nil {
		t.Fatalf("mkdir doing/rsi: %v", err)
	}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	// 恢复 cwd + 全局 --job + 生产探测 seam，避免测试间串味
	prevJob := jobID
	prevProbe := rsiHealthProbe
	t.Cleanup(func() {
		_ = os.Chdir(orig)
		jobID = prevJob
		rsiHealthProbe = prevProbe
	})
	jobID = ""
	rsiHealthProbe = func(string) (string, error) { return "c539c60-260921192330", nil }
	return tmp, doing
}

// rsiWrite writes one evidence file.
func rsiWrite(t *testing.T, doing, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(doing, "rsi", name), []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// rsiWriteAll writes a fully compliant evidence set.
func rsiWriteAll(t *testing.T, doing string) {
	t.Helper()
	rsiWrite(t, doing, "dev-iterations.md", "# dev\nDEV_UP bin=... build_id=c539c60-260921192330 port=8414\n")
	rsiWrite(t, doing, "gates.md", "# gates\nGATE gate11 pass=true\nGATE gate12 pass=true\n")
	rsiWrite(t, doing, "approval.md", "APPROVED by=human at=2026-09-21T20:40:00+08:00\n")
	rsiWrite(t, doing, "release.md", "version=c539c60-260921192330\nrollback_point=bin/releases/a60035a-260921195959\nreleased=yes\n")
	rsiWrite(t, doing, "resume.md", "suspended=3\nresumed=3（逐条人工点击恢复）\n")
}

func rsiFind(res rsiCheckResult, name string) rsiCheck {
	for _, c := range res.Checks {
		if c.Name == name {
			return c
		}
	}
	return rsiCheck{Name: name, Detail: "<未找到该检查项>"}
}

// TestRSICheckAllCompliant 全合规 → pass，且 6 项与 loop 产出评估表一一对应。
func TestRSICheckAllCompliant(t *testing.T) {
	_, doing := rsiTestWorkspace(t)
	rsiWriteAll(t, doing)

	res := runRSICheck("job_1", doing, "http://127.0.0.1:18413")
	if !res.Pass {
		t.Fatalf("全合规证据应 pass，实际失败: %v", res.Errors)
	}
	if len(res.Checks) != 6 {
		t.Fatalf("检查项数 = %d，期望 6（与 rick-rsi-loop 产出评估表逐项对齐）", len(res.Checks))
	}
	want := []string{"dev-iterations", "gates", "approval", "release", "resume", "prod-health"}
	for i, name := range want {
		if res.Checks[i].Name != name {
			t.Errorf("检查项 %d = %q，期望 %q（顺序=loop 表顺序）", i, res.Checks[i].Name, name)
		}
		if !res.Checks[i].Pass {
			t.Errorf("检查项 %s 应 pass: %s", name, res.Checks[i].Detail)
		}
	}
	if res.Job != "job_1" {
		t.Errorf("job = %q", res.Job)
	}
}

// TestRSICheckMissingItems 逐项缺失 → 只该项失败，且 detail 指对文件、给下一步指引。
func TestRSICheckMissingItems(t *testing.T) {
	cases := []struct {
		name      string
		remove    string
		wantCheck string
		wantIn    string
	}{
		{"dev-iterations 缺失", "dev-iterations.md", "dev-iterations", "dev-web"},
		{"gates 缺失", "gates.md", "gates", "gateN.py"},
		{"approval 缺失（最关键项）", "approval.md", "approval", "--dry-run"},
		{"release 缺失", "release.md", "release", "--merge-source"},
		{"resume 缺失", "resume.md", "resume", "挂起清单"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, doing := rsiTestWorkspace(t)
			rsiWriteAll(t, doing)
			if err := os.Remove(filepath.Join(doing, "rsi", tc.remove)); err != nil {
				t.Fatalf("remove: %v", err)
			}
			res := runRSICheck("job_1", doing, "http://127.0.0.1:18413")
			if res.Pass {
				t.Fatalf("缺 %s 时应失败", tc.remove)
			}
			got := rsiFind(res, tc.wantCheck)
			if got.Pass {
				t.Fatalf("%s 应失败", tc.wantCheck)
			}
			if !strings.Contains(got.Evidence, tc.remove) {
				t.Errorf("%s 的 evidence = %q，应指向 %s", tc.wantCheck, got.Evidence, tc.remove)
			}
			if !strings.Contains(got.Detail, tc.wantIn) {
				t.Errorf("%s 的 detail 应含下一步指引 %q，实际 %q", tc.wantCheck, tc.wantIn, got.Detail)
			}
			// 其它项不受影响（MECE：缺一项不该连带失败）。
			// 例外：prod-health 的定义就是「生产 build_id == release.md 里的 version」，
			// 所以缺 release.md 时它必然一起失败——这是**文档化的真实依赖**，不是耦合缺陷。
			for _, c := range res.Checks {
				if c.Name == tc.wantCheck {
					continue
				}
				if c.Name == "prod-health" && tc.remove == "release.md" {
					continue
				}
				if !c.Pass {
					t.Errorf("缺 %s 时 %s 也失败了（应为独立判定）: %s", tc.remove, c.Name, c.Detail)
				}
			}
		})
	}
}

// TestRSICheckContentRules 内容规则：pass=false / 无指纹 / approval 无标记 / resume 缺关键词。
func TestRSICheckContentRules(t *testing.T) {
	cases := []struct {
		name      string
		file      string
		body      string
		wantCheck string
		wantIn    string
	}{
		{"门禁出现 pass=false", "gates.md", "GATE gate7 pass=true\nGATE gate8 pass=false\n", "gates", "pass=false"},
		{"门禁无任何 pass 记录", "gates.md", "门禁都跑了，很绿\n", "gates", "pass="},
		{"dev 记录无构建指纹", "dev-iterations.md", "改了前端，overlay 热更成功\n", "dev-iterations", "构建指纹"},
		{"approval 无 human 标记", "approval.md", "at=2026-09-21T20:40:00+08:00 我自己同意了\n", "approval", "APPROVED by=human"},
		{"approval 无时间戳", "approval.md", "APPROVED by=human\n", "approval", "时间戳"},
		{"release 无回滚点", "release.md", "version=c539c60-260921192330\n", "release", "回滚点"},
		{"release 版本格式错", "release.md", "version=latest\nrollback_point=x\n", "release", "version="},
		{"resume 无挂起记录", "resume.md", "resumed=3 都恢复了\n", "resume", "挂起"},
		{"resume 无恢复结果", "resume.md", "suspended=3 有挂起\n", "resume", "恢复"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, doing := rsiTestWorkspace(t)
			rsiWriteAll(t, doing)
			rsiWrite(t, doing, tc.file, tc.body)
			res := runRSICheck("job_1", doing, "http://127.0.0.1:18413")
			if res.Pass {
				t.Fatalf("内容不合规（%s）时应失败", tc.name)
			}
			got := rsiFind(res, tc.wantCheck)
			if got.Pass {
				t.Fatalf("%s 应失败", tc.wantCheck)
			}
			if !strings.Contains(got.Detail, tc.wantIn) {
				t.Errorf("%s detail 应含 %q，实际 %q", tc.wantCheck, tc.wantIn, got.Detail)
			}
		})
	}
}

// TestRSICheckProdHealth 生产健康项：build_id 一致才过；不一致/探测失败/无地址都要有可操作指引。
func TestRSICheckProdHealth(t *testing.T) {
	t.Run("build_id 一致 → pass", func(t *testing.T) {
		_, doing := rsiTestWorkspace(t)
		rsiWriteAll(t, doing)
		rsiHealthProbe = func(url string) (string, error) {
			if !strings.HasSuffix(url, "18413") {
				t.Errorf("探测 URL = %q，应使用传入的 --prod-url", url)
			}
			return "c539c60-260921192330", nil
		}
		if res := runRSICheck("job_1", doing, "http://127.0.0.1:18413"); !res.Pass {
			t.Fatalf("应 pass: %v", res.Errors)
		}
	})

	t.Run("build_id 不一致 → fail 且指出生产没跑上本次构建", func(t *testing.T) {
		_, doing := rsiTestWorkspace(t)
		rsiWriteAll(t, doing)
		rsiHealthProbe = func(string) (string, error) { return "a60035a-260921195959", nil }
		res := runRSICheck("job_1", doing, "http://127.0.0.1:8413")
		got := rsiFind(res, "prod-health")
		if got.Pass {
			t.Fatal("build_id 不一致应失败")
		}
		if !strings.Contains(got.Detail, "没真的跑上") {
			t.Errorf("detail = %q，应说明生产未跑上新构建", got.Detail)
		}
	})

	t.Run("探测失败 → fail 且给出 curl 指引", func(t *testing.T) {
		_, doing := rsiTestWorkspace(t)
		rsiWriteAll(t, doing)
		rsiHealthProbe = func(string) (string, error) { return "", os.ErrDeadlineExceeded }
		res := runRSICheck("job_1", doing, "http://127.0.0.1:8413")
		got := rsiFind(res, "prod-health")
		if got.Pass {
			t.Fatal("探测失败应失败")
		}
		if !strings.Contains(got.Detail, "/api/health") {
			t.Errorf("detail = %q，应给出可执行的 curl 指引", got.Detail)
		}
	})

	t.Run("未给生产地址 → fail 且提示 --prod-url", func(t *testing.T) {
		_, doing := rsiTestWorkspace(t)
		rsiWriteAll(t, doing)
		res := runRSICheck("job_1", doing, "")
		got := rsiFind(res, "prod-health")
		if got.Pass {
			t.Fatal("无地址应失败")
		}
		if !strings.Contains(got.Detail, "--prod-url") {
			t.Errorf("detail = %q，应提示用 --prod-url", got.Detail)
		}
	})
}

// TestRSIInitSkeletons 骨架幂等 + 骨架本身不能「看起来通过」（防自欺）。
func TestRSIInitSkeletons(t *testing.T) {
	_, doing := rsiTestWorkspace(t)

	created, skipped, err := initRSIEvidence(doing)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if len(created) != 5 || len(skipped) != 0 {
		t.Fatalf("首次 init 应创建 5 个骨架，实际 created=%d skipped=%d", len(created), len(skipped))
	}
	for _, item := range rsiEvidenceFiles {
		if _, err := os.Stat(rsiEvidencePath(doing, item.File)); err != nil {
			t.Errorf("骨架缺失 %s: %v", item.File, err)
		}
	}

	// 骨架必须被判为「未填写」（否则 --init 之后立刻 pass = 契约自欺）
	res := runRSICheck("job_1", doing, "http://127.0.0.1:18413")
	if res.Pass {
		t.Fatal("未填写的骨架绝不能通过校验")
	}
	for _, c := range res.Checks {
		if c.Name == "prod-health" {
			continue // 该项依赖 release 记录，本就不适用
		}
		if c.Pass {
			t.Errorf("骨架状态下 %s 不应 pass: %s", c.Name, c.Detail)
		}
	}

	// 幂等：二次 init 不覆盖已填内容
	rsiWrite(t, doing, "approval.md", "APPROVED by=human at=2026-09-21T20:40:00+08:00\n")
	created2, skipped2, err2 := initRSIEvidence(doing)
	if err2 != nil {
		t.Fatalf("第二次 init: %v", err2)
	}
	if len(created2) != 0 || len(skipped2) != 5 {
		t.Fatalf("二次 init 应全部跳过，实际 created=%d skipped=%d", len(created2), len(skipped2))
	}
	body, _ := os.ReadFile(rsiEvidencePath(doing, "approval.md"))
	if !strings.Contains(string(body), "APPROVED by=human") {
		t.Fatal("二次 init 覆盖了已填写内容（幂等被破坏）")
	}
}

// TestRSIJobResolution job 解析优先级：显式 > 全局 --job > 最近修改的 job。
func TestRSIJobResolution(t *testing.T) {
	ws, _ := rsiTestWorkspace(t)
	// 显式设置 mtime：job_3 最新、job_2 居中、job_1 最旧。
	// 不依赖「创建顺序=时间顺序」——overlayfs 上同秒内创建的两个目录 mtime 可能相同，
	// 会让「取最近修改」的断言随机失败（实测踩到）。
	base := time.Now().Add(-time.Hour)
	for i, name := range []string{"job_1", "job_2", "job_3"} {
		dir := filepath.Join(ws, ".rick", "jobs", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		stamp := base.Add(time.Duration(i) * time.Minute)
		if err := os.Chtimes(dir, stamp, stamp); err != nil {
			t.Fatalf("chtimes %s: %v", name, err)
		}
	}
	got, err := resolveRSIJobID("")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != "job_3" {
		t.Errorf("未指定时应取 mtime 最新的 job_3，实际 %q", got)
	}
	if got2, _ := resolveRSIJobID("job_9"); got2 != "job_9" {
		t.Errorf("显式 --job 应优先，实际 %q", got2)
	}
	jobID = "job_7"
	if got3, _ := resolveRSIJobID(""); got3 != "job_7" {
		t.Errorf("全局 --job 应作为次选，实际 %q", got3)
	}
}

// TestRSICheckJSONShape --json 契约：字段名稳定（脚本/loop 依赖它解析）。
func TestRSICheckJSONShape(t *testing.T) {
	_, doing := rsiTestWorkspace(t)
	rsiWriteAll(t, doing)
	res := runRSICheck("job_1", doing, "http://127.0.0.1:18413")
	body, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var probe map[string]any
	if err := json.Unmarshal(body, &probe); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"pass", "job", "dir", "checks", "errors"} {
		if _, ok := probe[key]; !ok {
			t.Errorf("JSON 缺字段 %q: %s", key, body)
		}
	}
	checks, _ := probe["checks"].([]any)
	if len(checks) != 6 {
		t.Fatalf("checks 长度 = %d，期望 6", len(checks))
	}
	first, _ := checks[0].(map[string]any)
	for _, key := range []string{"name", "pass", "evidence", "detail"} {
		if _, ok := first[key]; !ok {
			t.Errorf("check 项缺字段 %q", key)
		}
	}
	// errors 必须是数组（无错时也要是 []，不能是 null —— 脚本会 .length / len()）
	if probe["errors"] == nil {
		t.Error("errors 不应为 null")
	}
}

// TestRSIParseStartScriptPort 从部署脚本解析端口（不硬编码 8413）。
func TestRSIParseStartScriptPort(t *testing.T) {
	script := "#!/bin/bash\ncd /srv/rick\nexec ./bin/rick web --listen 0.0.0.0 --port 8413\n"
	if got := parseStartScriptPort(script); got != "8413" {
		t.Errorf("port = %q, 期望 8413", got)
	}
	if got := parseStartScriptPort("no port here"); got != "" {
		t.Errorf("无 --port 时应返回空，实际 %q", got)
	}
}
