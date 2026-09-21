package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFileT 写入测试文件（父目录自动创建）。
func writeFileT(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fakeRickTree 造一个「像 rick 源码树」的临时工作区（含 cmd/rick/main.go 与
// internal/web/），可选写入 loop 文件。绝不接触真实仓库。
func fakeRickTree(t *testing.T, withLoop bool) string {
	t.Helper()
	ws := t.TempDir()
	writeFileT(t, filepath.Join(ws, "cmd", "rick", "main.go"), "package main\nfunc main() {}\n")
	writeFileT(t, filepath.Join(ws, "internal", "web", "web.go"), "package web\n")
	if withLoop {
		writeFileT(t, RSILoopPath(ws), testLoopContent)
	}
	return ws
}

const testLoopContent = `---
name: rick-rsi-loop
trigger: "当需要修改 rick 自身时触发"
scope: "全局"
---

# Loop: rick 自进化

## 目标（Goal）

一次自进化迭代安全送达生产。

## 上下文管理（Context Management）

保留改动清单与门禁结论。

## 可调用工具（Tool Access）

- ` + "`rick tools dev-web restart`" + ` —— 隔离 dev 实例
- ` + "`rick tools release --merge-source`" + ` —— 人类确认后提升
- ` + "`rick tools rsi_check --job <job>`" + ` —— 产出评估

## 产出评估（Output Evaluation）

由 rsi_check 机器校验。

## 停止标准（Termination Condition）

成功 = rsi_check pass=true。
`

func TestValidateRSIWorkspace(t *testing.T) {
	// 每个用例都在临时目录上跑；生产仓库根用显式 env 声明，避免依赖真实机器状态。
	t.Setenv(RSIProdRepoEnv, filepath.Join(t.TempDir(), "prod-repo"))
	t.Setenv(RSIAllowProdTreeEnv, "")

	t.Run("compliant rick source tree with loop", func(t *testing.T) {
		ws := fakeRickTree(t, true)
		if err := ValidateRSIWorkspace(ws); err != nil {
			t.Fatalf("expected valid, got %v", err)
		}
	})

	t.Run("not a rick source tree (missing cmd/rick/main.go)", func(t *testing.T) {
		ws := t.TempDir()
		writeFileT(t, filepath.Join(ws, "internal", "web", "web.go"), "package web\n")
		writeFileT(t, RSILoopPath(ws), testLoopContent)
		err := ValidateRSIWorkspace(ws)
		if err == nil || !strings.Contains(err.Error(), "cmd/rick/main.go") {
			t.Fatalf("want cmd/rick/main.go error, got %v", err)
		}
	})

	t.Run("not a rick source tree (missing internal/web)", func(t *testing.T) {
		ws := t.TempDir()
		writeFileT(t, filepath.Join(ws, "cmd", "rick", "main.go"), "package main\n")
		writeFileT(t, RSILoopPath(ws), testLoopContent)
		err := ValidateRSIWorkspace(ws)
		if err == nil || !strings.Contains(err.Error(), "internal/web") {
			t.Fatalf("want internal/web error, got %v", err)
		}
	})

	t.Run("missing loop file", func(t *testing.T) {
		ws := fakeRickTree(t, false)
		err := ValidateRSIWorkspace(ws)
		if err == nil || !strings.Contains(err.Error(), RSILoopFileName) {
			t.Fatalf("want missing-loop error mentioning %s, got %v", RSILoopFileName, err)
		}
		if err != nil && !strings.Contains(err.Error(), "dev-web init") {
			t.Fatalf("missing-loop error should point at the next step (dev-web init): %v", err)
		}
	})

	t.Run("production repo root is rejected", func(t *testing.T) {
		ws := fakeRickTree(t, true)
		t.Setenv(RSIProdRepoEnv, ws) // 该工作区即“生产仓库根”
		err := ValidateRSIWorkspace(ws)
		if err == nil || !strings.Contains(err.Error(), "dev 工作区") {
			t.Fatalf("want dev-workspace rejection, got %v", err)
		}
		// 逃生开关：仅模拟环境使用
		t.Setenv(RSIAllowProdTreeEnv, "1")
		if err := ValidateRSIWorkspace(ws); err != nil {
			t.Fatalf("escape hatch should allow prod tree, got %v", err)
		}
	})

	t.Run("empty and non-existent paths", func(t *testing.T) {
		if err := ValidateRSIWorkspace("  "); err == nil {
			t.Fatal("empty workspace must be rejected")
		}
		if err := ValidateRSIWorkspace(filepath.Join(t.TempDir(), "nope")); err == nil {
			t.Fatal("non-existent workspace must be rejected")
		}
	})
}

func TestBuildRSIPrompt(t *testing.T) {
	t.Setenv(RSIProdRepoEnv, filepath.Join(t.TempDir(), "prod-repo"))
	t.Setenv(RSIAllowProdTreeEnv, "")

	ws := fakeRickTree(t, true)
	text, method, err := BuildRSIPrompt(ws)
	if err != nil {
		t.Fatalf("BuildRSIPrompt: %v", err)
	}

	// methodFile = loop 自身的绝对路径（resume 时由 --append-system-prompt 复用）
	wantLoop := RSILoopPath(ws)
	if filepath.Clean(method) != filepath.Clean(wantLoop) {
		t.Fatalf("methodFile = %q, want %q", method, wantLoop)
	}
	if _, err := os.Stat(method); err != nil {
		t.Fatalf("methodFile must exist: %v", err)
	}

	// bootstrap 必须内嵌 loop 全文 + 声明强制遵循 +
	// 给出机制命令（loop 的执行面）
	for _, want := range []string{
		"rick-rsi-loop",                // loop 标识（frontmatter name）
		"## 停止标准",                      // loop 五要素之一（全文内嵌的证据）
		"必须完整遵循",                       // 强制遵循的声明
		"dev-web",                      // 机制：隔离 dev 实例
		"release --merge-source",       // 机制：提升（人类确认后）
		"rsi_check",                    // 机制：产出机器校验
		"不得 kill/重启生产实例",               // 硬约束
		"不自动续跑",                        // 硬约束（human 裁决）
		filepath.Join(".rick", "jobs"), // 落盘位置提示
	} {
		if !strings.Contains(text, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
	if !strings.Contains(text, wantLoop) {
		t.Errorf("prompt should name the authoritative loop path %q", wantLoop)
	}

	// 不满足校验的工作区不得产生提示词
	if ws2 := fakeRickTree(t, false); true {
		if _, _, err := BuildRSIPrompt(ws2); err == nil {
			t.Fatal("BuildRSIPrompt must fail when the loop is missing")
		}
	}
}

func TestEnsureRSIDirsAllocatesIncrementingIDs(t *testing.T) {
	rickDir := filepath.Join(t.TempDir(), ".rick")
	d1, err := EnsureRSIDirs(rickDir)
	if err != nil {
		t.Fatalf("EnsureRSIDirs: %v", err)
	}
	if filepath.Base(d1) != "rsi_1" {
		t.Fatalf("first dir = %q, want rsi_1", filepath.Base(d1))
	}
	d2, err := EnsureRSIDirs(rickDir)
	if err != nil {
		t.Fatalf("EnsureRSIDirs(2): %v", err)
	}
	if filepath.Base(d2) != "rsi_2" {
		t.Fatalf("second dir = %q, want rsi_2", filepath.Base(d2))
	}
	if filepath.Dir(d2) != filepath.Join(rickDir, "draft", "rsi") {
		t.Fatalf("dirs must live under <rickDir>/draft/rsi, got %s", d2)
	}
	// 非 rsi_N 目录不参与计数
	if err := os.MkdirAll(filepath.Join(rickDir, "draft", "rsi", "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	d3, err := EnsureRSIDirs(rickDir)
	if err != nil {
		t.Fatalf("EnsureRSIDirs(3): %v", err)
	}
	if filepath.Base(d3) != "rsi_3" {
		t.Fatalf("third dir = %q, want rsi_3", filepath.Base(d3))
	}
}

func TestEnsureRSISessionIDPersistsAndReuses(t *testing.T) {
	dir := t.TempDir()
	id1, err := EnsureRSISessionID(dir)
	if err != nil {
		t.Fatalf("EnsureRSISessionID: %v", err)
	}
	if len(id1) != 36 || strings.Count(id1, "-") != 4 {
		t.Fatalf("session id %q is not uuid-v4 shaped", id1)
	}
	data, err := os.ReadFile(filepath.Join(dir, "session_id"))
	if err != nil || strings.TrimSpace(string(data)) != id1 {
		t.Fatalf("session_id not persisted: %q err=%v", string(data), err)
	}
	id2, err := EnsureRSISessionID(dir)
	if err != nil || id2 != id1 {
		t.Fatalf("second call must reuse the persisted id (got %q, want %q, err=%v)", id2, id1, err)
	}
	// 空文件 → 重新生成（而不是返回空 id）
	if err := os.WriteFile(filepath.Join(dir, "session_id"), []byte("  \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	id3, err := EnsureRSISessionID(dir)
	if err != nil || id3 == "" || id3 == id1 {
		t.Fatalf("blank session_id must be regenerated (got %q err=%v)", id3, err)
	}
}

func TestGitMainRepoRootFrom(t *testing.T) {
	// 主仓库（.git 是目录）→ 返回该目录
	main := t.TempDir()
	if err := os.MkdirAll(filepath.Join(main, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(main, "internal", "web")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := gitMainRepoRootFrom(nested); filepath.Clean(got) != filepath.Clean(main) {
		t.Fatalf("main repo root = %q, want %q", got, main)
	}

	// worktree（.git 是文件，指向主仓库的 .git/worktrees/<name>）→ 返回主仓库根
	worktree := t.TempDir()
	writeFileT(t, filepath.Join(worktree, ".git"),
		"gitdir: "+filepath.Join(main, ".git", "worktrees", "rick-dev")+"\n")
	if got := gitMainRepoRootFrom(worktree); filepath.Clean(got) != filepath.Clean(main) {
		t.Fatalf("worktree → main repo root = %q, want %q", got, main)
	}

	// 无仓库 → ""
	if got := gitMainRepoRootFrom(t.TempDir()); got != "" {
		t.Fatalf("non-repo dir should yield empty, got %q", got)
	}
}

func TestRepoFromDeployScript(t *testing.T) {
	script := "#!/bin/bash\n# comment\ncd /workdir/sunquan20/AI_CODING/rick\nexec ./bin/rick web\n"
	if got := repoFromDeployScript(script); got != "/workdir/sunquan20/AI_CODING/rick" {
		t.Fatalf("deploy script repo = %q", got)
	}
	if got := repoFromDeployScript("#!/bin/bash\nrelative/path\n"); got != "" {
		t.Fatalf("non-absolute cd should be ignored, got %q", got)
	}
}
