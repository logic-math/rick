package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// loopsCheckResult is the --json contract of `rick tools loops_check`:
// {"pass":bool,"dir":string,"loops":int,"skills":int,"errors":[]}
type loopsCheckResult struct {
	Pass   bool     `json:"pass"`
	Dir    string   `json:"dir"`
	Loops  int      `json:"loops"`
	Skills int      `json:"skills"`
	Errors []string `json:"errors"`
}

// resolveRickDirForCheck accepts either the .rick directory itself or a
// workspace root that contains .rick/, and returns the .rick dir to validate.
//
// 为什么两种都接受：`--dir .rick`（job 脚本习惯）与 `--dir <工作区>`（人习惯）
// 都很自然；只接受一种会让人踩「明明文件在，却报目录不存在」的坑。
func resolveRickDirForCheck(dir string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve cwd: %w", err)
		}
		dir = cwd
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", dir, err)
	}
	// ① 本身就是 .rick（含 loops/ 或 skills/）
	if isDir(filepath.Join(abs, "loops")) || isDir(filepath.Join(abs, "skills")) {
		return abs, nil
	}
	// ② 工作区根：其下有 .rick/
	nested := filepath.Join(abs, ".rick")
	if isDir(filepath.Join(nested, "loops")) || isDir(filepath.Join(nested, "skills")) {
		return nested, nil
	}
	// ③ 两者皆无：若 .rick 目录存在就按它校验（可能只是还没建 loops/skills，
	//    校验器会把「目录不存在」当通过）；否则报错给出可操作提示。
	if isDir(nested) {
		return nested, nil
	}
	return "", fmt.Errorf("找不到可校验的 loops/skills 目录：%s（也不是含 .rick/ 的工作区根）", abs)
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// countMarkdownFiles counts *.md files in <rickDir>/<sub> excluding README.md —
// the same selection rule the validator applies (so the numbers match what was
// actually checked).
func countMarkdownFiles(rickDir, sub string) int {
	entries, err := os.ReadDir(filepath.Join(rickDir, sub))
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || filepath.Base(e.Name()) == "README.md" {
			continue
		}
		n++
	}
	return n
}

// runLoopsCheck validates a .rick dir and returns the JSON-shaped verdict.
func runLoopsCheck(rickDir string) loopsCheckResult {
	errs := runLoopsAndSkillsCheck(rickDir)
	if errs == nil {
		errs = []string{}
	}
	return loopsCheckResult{
		Pass:   len(errs) == 0,
		Dir:    rickDir,
		Loops:  countMarkdownFiles(rickDir, "loops"),
		Skills: countMarkdownFiles(rickDir, "skills"),
		Errors: errs,
	}
}

// NewLoopsCheckCmd validates .rick/loops/*.md and .rick/skills/*.md format.
//
// 为什么需要它：`runLoopsAndSkillsCheck` 之前只被 learning_check/dream_check
// 间接调用，没有独立入口 —— 于是 loop 目录可以长期「没人校验」（实测：loops/
// README 的目录清单就漏了 go-refactor-migration-loop）。把校验挂成命令后，
// loop 的格式（含 rick-rsi-loop 这个制度载体）可随时被机器检查。
func NewLoopsCheckCmd() *cobra.Command {
	var (
		dir     string
		jsonOut bool
	)

	cmd := &cobra.Command{
		Use:   "loops_check",
		Short: "Validate .rick/loops/*.md and .rick/skills/*.md format",
		Long: `Validate the format of project loops and skills.

Checks performed (per file, README.md skipped, deprecated/ not scanned):
  - .rick/loops/*.md:  frontmatter (name, trigger) + sections 目标/上下文管理/可调用工具/产出评估/停止标准
  - .rick/skills/*.md: frontmatter (name, description) + sections When to Use/Procedure/Pitfalls/Verification

Arguments:
  --dir   要校验的位置：.rick 目录本身，或含 .rick/ 的工作区根（默认：<cwd>/.rick）
  --json  输出一行 JSON 结论 {"pass":bool,"dir":...,"loops":N,"skills":M,"errors":[]}

Output:
  ✅ loops_check passed: <dir> (loops N / skills M)
  ❌ loops_check failed: <dir>
     - <file>: <problem>

Exit codes:
  0  all checks passed
  1  one or more checks failed (or the directory could not be resolved)`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rickDir, err := resolveRickDirForCheck(dir)
			if err != nil {
				if jsonOut {
					out, _ := json.Marshal(loopsCheckResult{Pass: false, Errors: []string{err.Error()}})
					fmt.Fprintln(cmd.OutOrStdout(), string(out))
				} else {
					fmt.Fprintf(os.Stderr, "❌ loops_check failed: %v\n", err)
				}
				os.Exit(1)
			}

			res := runLoopsCheck(rickDir)
			if jsonOut {
				out, mErr := json.Marshal(res)
				if mErr != nil {
					return fmt.Errorf("marshal result: %w", mErr)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
			} else if res.Pass {
				fmt.Fprintf(cmd.OutOrStdout(), "✅ loops_check passed: %s (loops %d / skills %d)\n",
					res.Dir, res.Loops, res.Skills)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "❌ loops_check failed: %s\n", res.Dir)
				for _, e := range res.Errors {
					fmt.Fprintf(cmd.OutOrStdout(), "   - %s\n", e)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "   （loop 格式规范见 .rick/loops/README.md：frontmatter name/trigger + 五个必需小节）\n")
			}
			if !res.Pass {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "要校验的位置：.rick 目录或含 .rick/ 的工作区根（默认 <cwd>/.rick）")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "输出一行 JSON 结论（脚本/AI 解析用）")
	return cmd
}
