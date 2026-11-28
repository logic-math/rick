package super

import (
	"os"
	"path/filepath"
	"testing"
)

// seedAssetFiles 在 dir 下落盘 rels 清单文件（相对 dir，含父目录创建），
// 内容带唯一标记 mark（可断言「来自哪个源树、未被覆盖」）。
func seedAssetFiles(t *testing.T, dir string, rels []string, mark string) {
	t.Helper()
	for _, rel := range rels {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(mark+" "+rel+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestDeploySuperAssetsFillsAllMissing（job_55）：裸 .rick 目标树 → 9 件
// 标准资产全部部署、内容与源一致、返回清单 9 项。
func TestDeploySuperAssetsFillsAllMissing(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src")
	dst := filepath.Join(t.TempDir(), "dst", ".rick")
	seedAssetFiles(t, src, SuperAssetPaths, "FROM_SRC")

	deployed, err := deploySuperAssets(src, dst)
	if err != nil {
		t.Fatalf("deploySuperAssets: %v", err)
	}
	if len(deployed) != len(SuperAssetPaths) {
		t.Fatalf("deployed = %v (len %d), want all %d assets", deployed, len(deployed), len(SuperAssetPaths))
	}
	for i, rel := range SuperAssetPaths {
		if deployed[i] != rel {
			t.Fatalf("deployed[%d] = %q, want %q", i, deployed[i], rel)
		}
		p := filepath.Join(dst, filepath.FromSlash(rel))
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("deployed asset missing %s: %v", p, err)
		}
		if want := "FROM_SRC " + rel + "\n"; string(data) != want {
			t.Fatalf("asset %s content = %q, want %q", rel, data, want)
		}
	}
}

// TestDeploySuperAssetsNeverOverwrites（job_55）：目标树已有定制版
// super-loop.md → 绝不覆盖（内容保持定制版），其余 8 件正常部署。
func TestDeploySuperAssetsNeverOverwrites(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src")
	dst := filepath.Join(t.TempDir(), "dst", ".rick")
	seedAssetFiles(t, src, SuperAssetPaths, "FROM_SRC")
	// 目标树先落定制版 super-loop.md（外部 workspace 用户定制资产优先）
	if err := os.MkdirAll(filepath.Join(dst, "loops"), 0o755); err != nil {
		t.Fatal(err)
	}
	custom := "# Loop: super-loop（用户定制版）\n"
	if err := os.WriteFile(filepath.Join(dst, "loops", "super-loop.md"), []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	deployed, err := deploySuperAssets(src, dst)
	if err != nil {
		t.Fatalf("deploySuperAssets: %v", err)
	}
	if len(deployed) != len(SuperAssetPaths)-1 {
		t.Fatalf("deployed = %v (len %d), want %d（跳过已存在的 super-loop.md）", deployed, len(deployed), len(SuperAssetPaths)-1)
	}
	for _, rel := range deployed {
		if rel == "loops/super-loop.md" {
			t.Fatalf("deployed contains already-existing %q — must never overwrite", rel)
		}
	}
	data, err := os.ReadFile(filepath.Join(dst, "loops", "super-loop.md"))
	if err != nil || string(data) != custom {
		t.Fatalf("custom super-loop.md overwritten: content %q err=%v", data, err)
	}
}

// TestDeploySuperAssetsPartialSourceTree（job_55）：源树只有部分资产 →
// 只部署可部署件、无错误（残缺源树不硬失败，由调用方的前置检查兜底）。
func TestDeploySuperAssetsPartialSourceTree(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src")
	dst := filepath.Join(t.TempDir(), "dst", ".rick")
	partial := SuperAssetPaths[:3]
	seedAssetFiles(t, src, partial, "FROM_SRC")

	deployed, err := deploySuperAssets(src, dst)
	if err != nil {
		t.Fatalf("deploySuperAssets(partial src): %v", err)
	}
	if len(deployed) != len(partial) {
		t.Fatalf("deployed = %v (len %d), want %d", deployed, len(deployed), len(partial))
	}
	for i, rel := range partial {
		if deployed[i] != rel {
			t.Fatalf("deployed[%d] = %q, want %q", i, deployed[i], rel)
		}
	}
	// 未在源树中的资产不应出现在目标树
	for _, rel := range SuperAssetPaths[3:] {
		if _, err := os.Stat(filepath.Join(dst, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Fatalf("asset %q deployed but not in src tree (stat err=%v)", rel, err)
		}
	}
}

// TestDeploySuperAssetsFromResolvedTree（job_55 wrapper）：假 rick 树
// （含 super/superopt/__main__.py marker + .rick/9 件资产）+ cwd 切入假树 →
// DeploySuperAssets 经 resolveRepoRoot cwd 上溯解析到假树并部署。
// ⚠️ 含 chdir，不并行（本包测试全串行）。
func TestDeploySuperAssetsFromResolvedTree(t *testing.T) {
	// 环境隔离（release 门禁实测 2026-11-28）：resolveRepoRoot ① ~/.rick 安装
	// 快照优先于 ② cwd 上溯——HOST HOME 指向带 super 快照的环境（release 门禁
	// 用 dev HOME，其 .rick/super/superopt marker 存在）时 ① 抢先命中，本测试
	// 期望的「② cwd 上溯」路径被环境破坏（deployed=[]）。注入空 RICK_PI_AGENT_DIR
	// 派生链使 ① 确定性落空，任何 HOME 下都走 ②。
	t.Setenv("RICK_PI_AGENT_DIR", filepath.Join(t.TempDir(), "isolated", ".rick", "pi", "agent"))
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "fakerepo")
	// superopt marker（resolveRepoRoot 的 cwd 上溯判据）
	if err := os.MkdirAll(filepath.Join(repo, "super", "superopt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "super", "superopt", "__main__.py"), []byte("#!py\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	seedAssetFiles(t, filepath.Join(repo, ".rick"), SuperAssetPaths, "FROM_FAKE_REPO")

	// cwd 切入假树下的子目录（模拟 web 会话 worker 在工作区内跑）
	inside := filepath.Join(repo, "some", "workspace")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	oldWd, _ := os.Getwd()
	if err := os.Chdir(inside); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	dst := filepath.Join(tmp, "external_ws", ".rick")
	deployed, err := DeploySuperAssets(dst)
	if err != nil {
		t.Fatalf("DeploySuperAssets: %v", err)
	}
	if len(deployed) != len(SuperAssetPaths) {
		t.Fatalf("deployed = %v (len %d), want all %d", deployed, len(deployed), len(SuperAssetPaths))
	}
	// 内容来自假树（带 FROM_FAKE_REPO 标记）而非真实 repo
	data, err := os.ReadFile(filepath.Join(dst, "loops", "super-loop.md"))
	if err != nil {
		t.Fatalf("read deployed super-loop.md: %v", err)
	}
	if want := "FROM_FAKE_REPO loops/super-loop.md\n"; string(data) != want {
		t.Fatalf("super-loop.md content = %q, want %q（应来自假 rick 树）", data, want)
	}
}

// TestSuperAssetFallbackPath_PriorityFlip（job_55 迭代 6）：human 裁决
// 2026-11-26——~/.rick 安装副本最优先（与二进制同版本配套，跨工作区一致），
// 工作区文件降为定制兼容回退；都缺时报 ~/.rick 侧路径。
func TestSuperAssetFallbackPath_PriorityFlip(t *testing.T) {
	rel := "loops/super-loop.md"

	t.Run("installed-first", func(t *testing.T) {
		// ~/.rick 有 + 工作区也有 → ~/.rick 优先（human 裁决「保持一致」）。
		tmp := t.TempDir()
		ws := filepath.Join(tmp, "ws", ".rick")
		inst := filepath.Join(tmp, "inst", ".rick")
		for _, d := range []string{ws, inst} {
			p := filepath.Join(d, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte("v"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		t.Setenv("RICK_PI_AGENT_DIR", filepath.Join(inst, "pi", "agent"))
		got := SuperAssetFallbackPath(ws, rel)
		want := filepath.Join(inst, filepath.FromSlash(rel))
		if got != want {
			t.Errorf("installed-first: got %q, want %q", got, want)
		}
	})

	t.Run("installed-missing-workspace-fallback", func(t *testing.T) {
		// ~/.rick 无 + 工作区有 → 工作区兜底（定制兼容）。
		tmp := t.TempDir()
		ws := filepath.Join(tmp, "ws", ".rick")
		inst := filepath.Join(tmp, "inst", ".rick")
		if err := os.MkdirAll(filepath.Join(ws, "loops"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(inst, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(ws, "loops", "super-loop.md"), []byte("v"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("RICK_PI_AGENT_DIR", filepath.Join(inst, "pi", "agent"))
		got := SuperAssetFallbackPath(ws, rel)
		want := filepath.Join(ws, filepath.FromSlash(rel))
		if got != want {
			t.Errorf("workspace-fallback: got %q, want %q", got, want)
		}
	})

	t.Run("both-missing-report-installed-path", func(t *testing.T) {
		// 都缺 → 报 ~/.rick 侧路径（提示 init-pi）。
		tmp := t.TempDir()
		ws := filepath.Join(tmp, "ws", ".rick")
		inst := filepath.Join(tmp, "inst", ".rick")
		for _, d := range []string{ws, inst} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		t.Setenv("RICK_PI_AGENT_DIR", filepath.Join(inst, "pi", "agent"))
		got := SuperAssetFallbackPath(ws, rel)
		want := filepath.Join(inst, filepath.FromSlash(rel))
		if got != want {
			t.Errorf("both-missing: got %q, want %q", got, want)
		}
	})
}
