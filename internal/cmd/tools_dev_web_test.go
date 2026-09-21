package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/sunquan/rick/internal/env/devweb"
)

// TestExitCodeForStage 锁定退出码契约（AI 会话据此判定失败阶段）：
// 0 ok / 2 build / 3 start（含 init）/ 4 health + fingerprint。
func TestExitCodeForStage(t *testing.T) {
	cases := map[string]int{
		"build":       devExitBuild,
		"start":       devExitStart,
		"init":        devExitStart,
		"health":      devExitHealth,
		"fingerprint": devExitHealth,
		"unknown":     devExitStart,
	}
	for stage, want := range cases {
		if got := exitCodeFor(stage); got != want {
			t.Errorf("exitCodeFor(%q) = %d, want %d", stage, got, want)
		}
	}
	if devExitBuild != 2 || devExitStart != 3 || devExitHealth != 4 {
		t.Fatalf("退出码常量被改动: build=%d start=%d health=%d", devExitBuild, devExitStart, devExitHealth)
	}
}

// TestStageOf 验证从 StageError 提取阶段；非 StageError 归到 start。
func TestStageOf(t *testing.T) {
	if got := stageOf(&devweb.StageError{Stage: "build", Err: errors.New("x")}); got != "build" {
		t.Fatalf("stageOf(StageError) = %q", got)
	}
	wrapped := errors.New("boom")
	if got := stageOf(wrapped); got != "start" {
		t.Fatalf("stageOf(generic) = %q, want start", got)
	}
}

// TestDevFailPrintsReceiptAndExits 验证失败回执格式与退出码（不真退出：替换 devExit）。
func TestDevFailPrintsReceiptAndExits(t *testing.T) {
	recorded := -1
	restore := devExit
	devExit = func(code int) { recorded = code }
	defer func() { devExit = restore }()

	var out bytes.Buffer
	cmd := NewDevWebCmd()
	cmd.SetOut(&out)
	// 多行错误必须被压成单行（AI 解析一行回执）
	_ = devFail(cmd, "build", errors.New("line1\nline2"))
	if recorded != devExitBuild {
		t.Fatalf("退出码 = %d, want %d", recorded, devExitBuild)
	}
	line := strings.TrimSpace(out.String())
	if !strings.HasPrefix(line, "DEV_FAIL stage=build detail=") {
		t.Fatalf("回执格式不符: %q", line)
	}
	if strings.Contains(line, "\n") || !strings.Contains(line, "line1 | line2") {
		t.Fatalf("多行错误未压成单行: %q", line)
	}
}

// TestDevWebCommandTree 验证子命令与关键 flag 齐备（AI 靠 --help 发现能力）。
func TestDevWebCommandTree(t *testing.T) {
	devWeb := NewDevWebCmd()
	want := map[string]bool{"init": false, "build": false, "up": false, "restart": false, "status": false, "down": false}
	for _, sub := range devWeb.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("dev-web 缺少子命令 %s", name)
		}
	}
	// 每个子命令都要有 --prod-repo（真实仓库可注入）；status 额外支持 --json
	for _, sub := range devWeb.Commands() {
		if sub.Flags().Lookup("prod-repo") == nil {
			t.Errorf("%s 缺少 --prod-repo", sub.Name())
		}
	}
	if status := findSub(devWeb, "status"); status == nil || status.Flags().Lookup("json") == nil {
		t.Error("status 应支持 --json")
	}
	if up := findSub(devWeb, "up"); up == nil || up.Flags().Lookup("no-build") == nil {
		t.Error("up 应支持 --no-build")
	}
	// tools 父命令必须挂载 dev-web（AI 通过 `rick tools --help` 发现）
	found := false
	for _, sub := range NewToolsCmd().Commands() {
		if sub.Name() == "dev-web" {
			found = true
		}
	}
	if !found {
		t.Fatal("`rick tools` 未挂载 dev-web")
	}
}

// TestStatusReportsDownForFreshLayout 是 CLI 级的只读走查：给一套空布局（env
// 覆盖到临时目录）→ status 必须报 down 且不触碰任何真实进程/状态。
func TestStatusReportsDownForFreshLayout(t *testing.T) {
	root := t.TempDir()
	prodRepo := filepath.Join(root, "prod", "rick")
	// dev 树夹具需具备「rick 源码树」形态（cmd/rick + web/package.json）——F4 起
	// 需要已有树的子命令会在解析阶段校验，空目录会被明确拒绝。
	devTree := filepath.Join(root, "dev-tree")
	for _, d := range []string{prodRepo, devTree, filepath.Join(devTree, "cmd", "rick"), filepath.Join(devTree, "web")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(devTree, "web", "package.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(devweb.EnvTree, devTree)
	t.Setenv(devweb.EnvHome, filepath.Join(root, "dev-home"))
	t.Setenv(devweb.EnvPort, "18499")

	var out bytes.Buffer
	cmd := NewDevWebCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"status", "--prod-repo", prodRepo, "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("status 执行失败: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, `"running": false`) {
		t.Fatalf("空布局应报 running=false: %s", body)
	}
	if !strings.Contains(body, "state_path") {
		t.Fatalf("status --json 应包含 state_path: %s", body)
	}
	// 只读走查：不该在临时布局里写到状态文件
	if _, err := os.Stat(filepath.Join(root, "dev-home", "dev-state.json")); !os.IsNotExist(err) {
		t.Fatal("status 不该写状态文件")
	}
}

// TestInitFailsFastWithoutGitRepo 验证 init 在没有 git 工作树时快速失败，
// 且回执 stage=init（并被映射为退出码 3）。
func TestInitFailsFastWithoutGitRepo(t *testing.T) {
	recorded := -1
	restore := devExit
	devExit = func(code int) { recorded = code }
	defer func() { devExit = restore }()

	root := t.TempDir()
	prodRepo := filepath.Join(root, "not-a-repo") // 有意不带 .git
	if err := os.MkdirAll(prodRepo, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(devweb.EnvTree, filepath.Join(root, "dev-tree"))
	t.Setenv(devweb.EnvHome, filepath.Join(root, "dev-home"))
	t.Setenv(devweb.EnvPort, "18498")

	var out bytes.Buffer
	cmd := NewDevWebCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"init", "--prod-repo", prodRepo})
	_ = cmd.Execute() // devFail 已经接管退出码
	if recorded != devExitStart {
		t.Fatalf("init 失败应退出码 %d，实际 %d", devExitStart, recorded)
	}
	if !strings.Contains(out.String(), "DEV_FAIL stage=init") {
		t.Fatalf("回执应标明 stage=init: %s", out.String())
	}
}

// TestDefaultProdRepoIsUsable 验证生产仓库推导至少返回一个存在的目录
// （真实运行时是 `<repo>/bin/rick` 的祖父目录）。
func TestDefaultProdRepoIsUsable(t *testing.T) {
	got := defaultProdRepo()
	if got == "" {
		t.Fatal("defaultProdRepo 返回空")
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("defaultProdRepo 返回不存在的路径 %q: %v", got, err)
	}
}

// findSub 在子命令集合里按名字查找（测试用）。
func findSub(parent *cobra.Command, name string) *cobra.Command {
	for _, sub := range parent.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	return nil
}
