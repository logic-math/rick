package release

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ---- 夹具：真实的小 git 仓库 + worktree（全部落在 t.TempDir()，绝不碰真实仓库）----

type mergeFixture struct {
	root     string
	prodRepo string
	devTree  string
	mainPath string // prod 仓库里的受控文件（冲突实验用）
}

// gitT 在 dir 里跑 git，失败即 Fail（测试夹具用；被测代码自己的 git 调用不走这里）。
func gitT(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_CONFIG_NOSYSTEM=1",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s（dir=%s）失败: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

// gitSoft 与 gitT 相同，但**允许非零退出**（例如 rev-parse --verify MERGE_HEAD 在
// 「没有合并在进行」时就是 exit 1，那是被断言的正常态，不该 Fail）。
func gitSoft(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_CONFIG_NOSYSTEM=1",
	)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// newMergeFixture 造：prod 仓库（分支 main，含 f.txt）+ 它的 worktree dev（分支 dev/self-evolve）。
// 生产分支里预置一次提交，供冲突实验使用。
func newMergeFixture(t *testing.T) mergeFixture {
	t.Helper()
	root := t.TempDir()
	prod := filepath.Join(root, "prod")
	dev := filepath.Join(root, "dev")
	mustMkdir(t, prod)
	gitT(t, prod, "init", "-q", "-b", "main")
	gitT(t, prod, "config", "user.email", "test@example.com")
	gitT(t, prod, "config", "user.name", "test")
	mustWrite(t, filepath.Join(prod, "go.mod"), "module github.com/sunquan/rick\n")
	mustWrite(t, filepath.Join(prod, "f.txt"), "base\n")
	gitT(t, prod, "add", ".")
	gitT(t, prod, "commit", "-qm", "base")
	gitT(t, prod, "worktree", "add", "-q", "-b", "dev/self-evolve", dev)
	return mergeFixture{root: root, prodRepo: prod, devTree: dev, mainPath: filepath.Join(prod, "f.txt")}
}

// devCommit 在 dev 工作树提交一个文件改动。
func (f mergeFixture) devCommit(t *testing.T, name, body string) {
	t.Helper()
	mustWrite(t, filepath.Join(f.devTree, name), body)
	gitT(t, f.devTree, "add", "--", name)
	gitT(t, f.devTree, "commit", "-qm", "dev: "+name)
}

// mainCommit 在生产分支提交一个文件改动。
func (f mergeFixture) mainCommit(t *testing.T, name, body string) {
	t.Helper()
	mustWrite(t, filepath.Join(f.prodRepo, name), body)
	gitT(t, f.prodRepo, "add", "--", name)
	gitT(t, f.prodRepo, "commit", "-qm", "main: "+name)
}

func (f mergeFixture) plan() Plan {
	return Plan{ProdRepo: f.prodRepo, DevTree: f.devTree, ProdHome: f.root, Port: 18999}
}

func (f mergeFixture) porcelain(t *testing.T) string {
	t.Helper()
	return gitT(t, f.prodRepo, "status", "--porcelain")
}

func (f mergeFixture) log1(t *testing.T) string {
	t.Helper()
	return gitT(t, f.prodRepo, "log", "-1", "--pretty=%s")
}

// TestMergeSourceCleanMerge 无冲突：合并成功、main 产生 merge commit、引入的文件被计入、工作树干净。
func TestMergeSourceCleanMerge(t *testing.T) {
	f := newMergeFixture(t)
	f.devCommit(t, "g.txt", "dev only\n")

	res, err := f.plan().MergeSource()
	if err != nil {
		t.Fatalf("MergeSource: %v", err)
	}
	if !res.Merged {
		t.Fatalf("Merged = false，期望 true（res=%+v）", res)
	}
	if res.Branch != "dev/self-evolve" || res.Main != "main" {
		t.Fatalf("分支记录错误: branch=%q main=%q", res.Branch, res.Main)
	}
	if res.Commit == "" || len(res.Commit) < 7 {
		t.Fatalf("commit 短 sha 缺失: %q", res.Commit)
	}
	if res.Files < 1 {
		t.Fatalf("Files = %d，期望 ≥1（引入 g.txt）", res.Files)
	}
	// merge commit 真的落到 main 上，且信息可审计
	head := f.log1(t)
	if !strings.Contains(head, "merge dev/self-evolve") {
		t.Fatalf("HEAD 提交信息 = %q，期望含 merge 摘要", head)
	}
	if got := f.porcelain(t); got != "" {
		t.Fatalf("合并后生产工作树应干净，实际: %q", got)
	}
	// merge 的 ParentCount = 2（--no-ff）
	parents := gitT(t, f.prodRepo, "rev-list", "--parents", "-n", "1", "HEAD")
	if len(strings.Fields(parents)) != 3 {
		t.Fatalf("期望 merge commit（2 个父提交），实际 rev-list --parents: %q", parents)
	}
}

// TestMergeSourceConflictAbortsAndRestores 冲突：报错含冲突文件与修复指引，
// 且**生产仓库恢复到合并前**（MERGE_HEAD 消失、工作树干净、文件内容为 main 版本）。
func TestMergeSourceConflictAbortsAndRestores(t *testing.T) {
	f := newMergeFixture(t)
	f.devCommit(t, "f.txt", "dev version\n")
	f.mainCommit(t, "f.txt", "main version\n")

	before := gitT(t, f.prodRepo, "rev-parse", "HEAD")

	_, err := f.plan().MergeSource()
	if err == nil {
		t.Fatal("冲突合并应返回错误，实际 nil")
	}
	msg := err.Error()
	for _, want := range []string{"f.txt", "冲突", "git merge --abort", "--no-merge-source", "release --merge-source"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("冲突错误信息缺少 %q:\n%s", want, msg)
		}
	}
	if stage := stageOf(t, err); stage != "merge" {
		t.Fatalf("失败阶段 = %q，期望 merge", stage)
	}
	// 现场必须恢复：无 MERGE_HEAD、工作树干净、HEAD 未动、文件内容 = main 版本
	if sha, err := gitSoft(t, f.prodRepo, "rev-parse", "-q", "--verify", "MERGE_HEAD"); err == nil || sha != "" {
		t.Fatalf("MERGE_HEAD 残留（err=%v sha=%q）", err, sha)
	}
	if got := f.porcelain(t); got != "" {
		t.Fatalf("abort 后工作树应干净，实际: %q", got)
	}
	if after := gitT(t, f.prodRepo, "rev-parse", "HEAD"); after != before {
		t.Fatalf("HEAD 被改动: %s → %s（合并失败不得产生提交）", before, after)
	}
	body, err := os.ReadFile(f.mainPath)
	if err != nil {
		t.Fatalf("read f.txt: %v", err)
	}
	if string(body) != "main version\n" {
		t.Fatalf("f.txt 内容 = %q，期望回到合并前（main version）", string(body))
	}
}

// TestMergeSourceDirtyWorktreeAborts 脏工作树 → 中止且零改动，错误信息点明「先提交或暂存」。
func TestMergeSourceDirtyWorktreeAborts(t *testing.T) {
	cases := []struct {
		name  string
		dirty func(t *testing.T, f mergeFixture)
	}{
		{"modified tracked", func(t *testing.T, f mergeFixture) {
			mustWrite(t, f.mainPath, "uncommitted edit\n")
		}},
		{"untracked file", func(t *testing.T, f mergeFixture) {
			mustWrite(t, filepath.Join(f.prodRepo, "untracked.txt"), "x\n")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newMergeFixture(t)
			f.devCommit(t, "g.txt", "dev only\n")
			tc.dirty(t, f)
			before := gitT(t, f.prodRepo, "rev-parse", "HEAD")

			_, err := f.plan().MergeSource()
			if err == nil {
				t.Fatal("脏工作树应中止，实际返回 nil")
			}
			if stage := stageOf(t, err); stage != "merge" {
				t.Fatalf("失败阶段 = %q，期望 merge", stage)
			}
			if !strings.Contains(err.Error(), "提交或暂存") {
				t.Fatalf("错误未提示先提交/暂存:\n%s", err.Error())
			}
			if after := gitT(t, f.prodRepo, "rev-parse", "HEAD"); after != before {
				t.Fatalf("脏工作树也不得产生提交: %s → %s", before, after)
			}
			// dev 侧零改动
			if got := gitT(t, f.devTree, "status", "--porcelain"); got != "" {
				t.Fatalf("dev 工作树被改动: %q", got)
			}
		})
	}
}

// TestMergeSourceAlreadyUpToDate dev 无新提交 → merged=false（幂等成功，不是失败）。
func TestMergeSourceAlreadyUpToDate(t *testing.T) {
	f := newMergeFixture(t)
	res, err := f.plan().MergeSource()
	if err != nil {
		t.Fatalf("already up to date 不应报错: %v", err)
	}
	if res.Merged {
		t.Fatal("Merged = true，期望 false（dev 未领先）")
	}
	if !strings.Contains(res.Reason, "up to date") {
		t.Fatalf("Reason = %q，期望包含 up to date", res.Reason)
	}
	if got := f.porcelain(t); got != "" {
		t.Fatalf("幂等路径不得改动工作树: %q", got)
	}
}

// TestMergeSourceRejectsUnusableBranches 无歧义前提：detached HEAD / 同分支 / 缺目录都要拒绝。
func TestMergeSourceRejectsUnusableBranches(t *testing.T) {
	t.Run("dev detached HEAD", func(t *testing.T) {
		f := newMergeFixture(t)
		gitT(t, f.devTree, "checkout", "-q", "--detach")
		_, err := f.plan().MergeSource()
		if err == nil || !strings.Contains(err.Error(), "detached HEAD") {
			t.Fatalf("期望 detached HEAD 错误，实际: %v", err)
		}
	})
	t.Run("same branch", func(t *testing.T) {
		// git 不允许两个 worktree 同时 checkout 同一分支，所以这里用一个**独立仓库**
		// 当 dev 树：它的分支名也叫 main → 与生产分支同名，合并无意义。
		root := t.TempDir()
		dev := filepath.Join(root, "dev-main")
		mustMkdir(t, dev)
		gitT(t, dev, "init", "-q", "-b", "main")
		gitT(t, dev, "config", "user.email", "test@example.com")
		gitT(t, dev, "config", "user.name", "test")
		mustWrite(t, filepath.Join(dev, "go.mod"), "module github.com/sunquan/rick\n")
		gitT(t, dev, "add", ".")
		gitT(t, dev, "commit", "-qm", "dev main")
		prod := filepath.Join(root, "prod")
		mustMkdir(t, prod)
		gitT(t, prod, "init", "-q", "-b", "main")
		gitT(t, prod, "config", "user.email", "test@example.com")
		gitT(t, prod, "config", "user.name", "test")
		mustWrite(t, filepath.Join(prod, "go.mod"), "module github.com/sunquan/rick\n")
		gitT(t, prod, "add", ".")
		gitT(t, prod, "commit", "-qm", "prod main")
		p := Plan{ProdRepo: prod, DevTree: dev, ProdHome: root, Port: 18999}
		_, err := p.MergeSource()
		if err == nil || !strings.Contains(err.Error(), "同一分支") {
			t.Fatalf("期望「同一分支」错误，实际: %v", err)
		}
	})
	t.Run("missing dev tree", func(t *testing.T) {
		f := newMergeFixture(t)
		p := f.plan()
		p.DevTree = filepath.Join(f.root, "nope")
		_, err := p.MergeSource()
		if err == nil || !strings.Contains(err.Error(), "dev 工作树不存在") {
			t.Fatalf("期望「dev 工作树不存在」，实际: %v", err)
		}
	})
	t.Run("merge already in progress", func(t *testing.T) {
		f := newMergeFixture(t)
		f.devCommit(t, "f.txt", "dev version\n")
		f.mainCommit(t, "f.txt", "main version\n")
		// 手工制造一个未完成的合并（不 abort），再调 MergeSource 应被拒绝
		if _, err := gitCombined(f.prodRepo, "merge", "--no-ff", "--no-commit", "dev/self-evolve"); err == nil {
			t.Fatal("夹具期望制造出冲突")
		}
		_, err := f.plan().MergeSource()
		if err == nil || !strings.Contains(err.Error(), "未完成的合并") {
			t.Fatalf("期望「未完成的合并」错误，实际: %v", err)
		}
		// 清理夹具现场（避免 temp 目录残留影响后续）
		_ = gitAbortMerge(f.prodRepo)
	})
}

// TestMergeSourceCommitMessageCarriesVersion 合并提交信息带 MergeVersion（与提升版本对应）。
func TestMergeSourceCommitMessageCarriesVersion(t *testing.T) {
	f := newMergeFixture(t)
	f.devCommit(t, "g.txt", "dev only\n")
	p := f.plan()
	p.MergeVersion = "abcdef1-260101000000"
	if _, err := p.MergeSource(); err != nil {
		t.Fatalf("MergeSource: %v", err)
	}
	if head := f.log1(t); !strings.Contains(head, "version=abcdef1-260101000000") {
		t.Fatalf("merge commit 信息 = %q，期望含 version=", head)
	}
}

// TestMergeSourceWithoutGitIdentity仍能提交：生产仓库没配 user.email 时补中性身份，
// 不让 release 卡在 "Please tell me who you are"。
func TestMergeSourceWorksWithoutGitIdentity(t *testing.T) {
	f := newMergeFixture(t)
	f.devCommit(t, "g.txt", "dev only\n")
	// 清掉 prod 仓库的 user.email（worktree 与主仓库共享 config）
	gitT(t, f.prodRepo, "config", "--unset", "user.email")
	gitT(t, f.prodRepo, "config", "--unset", "user.name")
	if _, err := f.plan().MergeSource(); err != nil {
		t.Fatalf("缺 git 身份时应补默认身份完成提交，实际: %v", err)
	}
	if head := f.log1(t); !strings.Contains(head, "merge dev/self-evolve") {
		t.Fatalf("HEAD = %q，期望 merge commit", head)
	}
}
