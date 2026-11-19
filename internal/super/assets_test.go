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
