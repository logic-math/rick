package super

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveRepoRootConfigPointerFallback（job_55 方案 2 dev-e2e 现场回归）：
// install.sh 把二进制拷到 ~/.rick/bin/rick（exe 不在 repo 内）+ cwd 也不在
// repo 内时，resolveRepoRoot 必须回退到 config 的 source_tree 指针；指针失效
// （指向不存在/无 marker 的路径）时降级回 cwd 兜底（既有行为不变）。
//
// 复现方式：把测试二进制拷到临时目录（exe 脱离本 repo——进程内无法注入
// os.Executable，走子进程 helper 模式打印 resolveRepoRoot()），cwd 置外部目录，
// HOME 指向含 config.json（source_tree 指针）的临时目录。
func TestResolveRepoRootConfigPointerFallback(t *testing.T) {
	if os.Getenv("RICK_SUPER_HELPER_RESOLVE") != "" {
		// helper 模式：打印解析结果后立即退出（父进程以 -test.run 过滤进来）。
		fmt.Println("RESULT " + resolveRepoRoot())
		os.Exit(0)
	}

	tmp := t.TempDir()

	// 假 repo（指针目标，含 marker）。
	repo := filepath.Join(tmp, "pointedrepo")
	if err := os.MkdirAll(filepath.Join(repo, "super", "superopt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "super", "superopt", "__main__.py"), []byte("#!py\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 外部 cwd（无 marker）。
	ext := filepath.Join(tmp, "external_ws")
	if err := os.MkdirAll(ext, 0o755); err != nil {
		t.Fatal(err)
	}

	// 拷贝测试二进制到临时目录：helper 的 exe 位置脱离本 repo，
	// ①cwd ②exe 两条上溯都找不到 marker，只剩 ③指针/④cwd 兜底可判。
	helper := filepath.Join(tmp, "helper.test")
	testExe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(testExe)
	if err != nil {
		t.Fatalf("read test binary: %v", err)
	}
	if err := os.WriteFile(helper, data, 0o755); err != nil {
		t.Fatal(err)
	}

	// run：以指定 HOME + 外部 cwd 跑 helper，取 RESULT 行。
	run := func(homeDir string) string {
		cmd := exec.Command(helper, "-test.run=TestResolveRepoRootConfigPointerFallback")
		cmd.Dir = ext
		// 剥掉 PWD：强制 os.Getwd 返回物理路径（与 EvalSymlinks 归一可比）。
		var env []string
		for _, kv := range os.Environ() {
			if strings.HasPrefix(kv, "PWD=") || strings.HasPrefix(kv, "HOME=") {
				continue
			}
			env = append(env, kv)
		}
		cmd.Env = append(env, "HOME="+homeDir, "RICK_SUPER_HELPER_RESOLVE=1")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("helper: %v\n%s", err, out)
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "RESULT ") {
				return strings.TrimSpace(strings.TrimPrefix(line, "RESULT "))
			}
		}
		t.Fatalf("helper output missing RESULT line: %q", out)
		return ""
	}

	writePointerConfig := func(homeDir, pointer string) {
		if err := os.MkdirAll(filepath.Join(homeDir, ".rick"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(homeDir, ".rick", "config.json"),
			[]byte(fmt.Sprintf(`{"source_tree":%q}`, pointer)), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// ① 指针有效 → 解析到假 repo（marker 校验通过）。
	home1 := filepath.Join(tmp, "home1")
	writePointerConfig(home1, repo)
	want, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got := run(home1); got != want {
		t.Errorf("pointer fallback: resolveRepoRoot = %q, want %q", got, want)
	}

	// ② 指针失效（指向不存在的路径）→ 降级回 cwd 兜底。
	home2 := filepath.Join(tmp, "home2")
	writePointerConfig(home2, filepath.Join(tmp, "gone"))
	wantCwd, err := filepath.EvalSymlinks(ext)
	if err != nil {
		t.Fatal(err)
	}
	if got := run(home2); got != wantCwd {
		t.Errorf("dangling pointer: resolveRepoRoot = %q, want cwd fallback %q", got, wantCwd)
	}

	// ③ 指针指向存在但无 marker 的目录 → 同样降级回 cwd 兜底。
	home3 := filepath.Join(tmp, "home3")
	writePointerConfig(home3, ext)
	if got := run(home3); got != wantCwd {
		t.Errorf("markerless pointer: resolveRepoRoot = %q, want cwd fallback %q", got, wantCwd)
	}
}

// TestResolveRepoRootInstalledSnapshotFirst（job_55 迭代 6 优先级翻转）：
// ~/.rick/super 安装快照存在时，resolveRepoRoot 必须最优先返回它——即使
// cwd 也在 rick repo 内（human 裁决「直接读 ~/.rick 保持一致」）。
// 复现方式与 ConfigPointerFallback 同：子进程 helper 模式。
func TestResolveRepoRootInstalledSnapshotFirst(t *testing.T) {
	if os.Getenv("RICK_SUPER_HELPER_RESOLVE") != "" {
		fmt.Println("RESULT " + resolveRepoRoot())
		os.Exit(0)
	}

	tmp := t.TempDir()

	// 假安装快照（~/.rick/super/superopt/__main__.py）。
	instHome := filepath.Join(tmp, "insthome")
	instSuper := filepath.Join(instHome, ".rick", "super", "superopt")
	if err := os.MkdirAll(instSuper, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instSuper, "__main__.py"), []byte("#!py\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 假 repo（cwd 侧也有 marker——但优先级低于安装快照）。
	cwdRepo := filepath.Join(tmp, "cwdrepo")
	if err := os.MkdirAll(filepath.Join(cwdRepo, "super", "superopt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwdRepo, "super", "superopt", "__main__.py"), []byte("#!py\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 拷贝测试二进制到 cwdRepo/bin（exe 也在 repo 内——同样应低于安装快照）。
	helper := filepath.Join(cwdRepo, "bin", "helper.test")
	if err := os.MkdirAll(filepath.Dir(helper), 0o755); err != nil {
		t.Fatal(err)
	}
	testExe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(testExe)
	if err != nil {
		t.Fatalf("read test binary: %v", err)
	}
	if err := os.WriteFile(helper, data, 0o755); err != nil {
		t.Fatal(err)
	}

	// 跑 helper：HOME 指向 instHome（安装快照存在）、cwd 在 cwdRepo（marker 也在）。
	// 期望：安装快照最优先（human 裁决——与二进制同版本配套，保持一致）。
	cmd := exec.Command(helper, "-test.run=TestResolveRepoRootInstalledSnapshotFirst")
	cmd.Dir = cwdRepo
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "PWD=") || strings.HasPrefix(kv, "HOME=") || strings.HasPrefix(kv, "RICK_PI_AGENT_DIR=") {
			continue
		}
		env = append(env, kv)
	}
	cmd.Env = append(env, "HOME="+instHome, "RICK_SUPER_HELPER_RESOLVE=1")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("helper: %v\n%s", err, out)
	}
	var got string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "RESULT ") {
			got = strings.TrimSpace(strings.TrimPrefix(line, "RESULT "))
			break
		}
	}
	if got == "" {
		t.Fatalf("helper output missing RESULT line: %q", out)
	}
	want := filepath.Join(instHome, ".rick")
	// macOS /var ↔ /private/var 双形归一。
	wantResolved, _ := filepath.EvalSymlinks(want)
	gotResolved, _ := filepath.EvalSymlinks(got)
	if gotResolved != wantResolved {
		t.Errorf("installed snapshot first: resolveRepoRoot = %q, want %q (got %q vs want %q)", got, want, gotResolved, wantResolved)
	}
}
