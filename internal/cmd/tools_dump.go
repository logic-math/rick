package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sunquan/rick/internal/env/relocate"
)

// NewDumpCmd creates the dump subcommand: pack one or more workspaces plus
// rick's state (config, web registries, pi agent dir incl. session history)
// into a single portable archive. Thin Cobra entry — the work lives in
// internal/env/relocate (job_46 design tree P1/P3).
func NewDumpCmd(rickVersion string) *cobra.Command {
	var (
		workspaces   []string
		output       string
		sessions     string
		slim         bool
		goos         []string
		noExclude    bool
		extraExclude []string
		stateDir     string
	)
	cmd := &cobra.Command{
		Use:   "dump",
		Short: "Pack workspaces + rick state into one portable archive",
		Long: `Pack the current directory (or --workspace paths) together with rick's
state (~/.rick: config, web registries, pi agent dir incl. session history)
into a single tar.gz, ready for ` + "`rick tools import`" + ` on another machine
(platform-independent: Linux ↔ macOS).

What travels along:
  workspaces/<name>/   the packed workspace trees (all files except the
                       exclusion rules below)
  state/               config.json, web.json, web/sessions.json ... ,
                       start-web.sh (paths get remapped at import time)
  state/pi-agent/      auth.json / models-store.json / settings.json /
                       extensions / themes / runtime (unless --slim) and the
                       pi session history (--sessions)
  bin/<goos>-<goarch>/ cross-compiled rick binaries (default linux/amd64,
                       darwin/arm64, darwin/amd64)

Preprocessing during packing (each hit recorded in the archive's manifest):
  platform-elf  machine-bound binaries (e.g. pi's bin/rg — re-downloaded on
                the target for its platform)
  regenerable   web/node_modules (contains linux-x64 native modules),
                workspace bin/, web/dist.prev
  cache         .pi/, .morty/, web-search-cache/, config.json.bak.*
  runtime       web.pid / web.lock / *.log
  external      catpaw-proxy extension (company-internal provider; ruled out
                of scope by human decision)

Source-side effects (idempotent): each workspace gets a
.rick-workspace.json identity marker when missing, and the source state's
registries are upgraded from legacy path-hash ids to marker ids — moving a
workspace no longer orphans its session history.

Exit codes:
  0  dump written (see output path + .sha256 sidecar)
  1  invalid options, unreadable state, or native binary build failure`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("dump 不接受位置参数（用 --workspace 追加工作区）")
			}
			if sessions != "" && sessions != relocate.SessionsPolicyNone &&
				sessions != relocate.SessionsPolicyWorkspaces && sessions != relocate.SessionsPolicyAll {
				return fmt.Errorf("--sessions 取值无效：%q（none|workspaces|all）", sessions)
			}
			opts := relocate.DumpOptions{
				Workspaces:   workspaces,
				Output:       output,
				Sessions:     sessions,
				Slim:         slim,
				Goos:         goos,
				NoExclude:    noExclude,
				ExtraExclude: extraExclude,
				StateDir:     stateDir,
				RickVersion:  rickVersion,
			}
			res, err := relocate.Dump(context.Background(), opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ dump failed: %v\n", err)
				os.Exit(1)
			}
			for _, w := range res.Warnings {
				fmt.Printf("⚠️  %s\n", w)
			}
			fmt.Printf("📦 dump 完成：%s (%s)\n", res.ArchivePath, humanBytes(res.ArchiveSize))
			fmt.Printf("   sha256=%s（边车 %s.sha256）\n", res.SHA256[:16]+"…", filepath.Base(res.ArchivePath))
			fmt.Printf("   工作区 %d 个：%s\n", len(res.Workspaces), strings.Join(basenames(res.Workspaces), ", "))
			fmt.Printf("   归档 %d 个文件 / payload %s；排除 %d 项；内嵌二进制 %s\n",
				res.Files, humanBytes(res.Bytes), res.Excluded, platformList(res.Binaries))
			if !res.AgentPacked {
				fmt.Printf("⚠️  未找到 pi agent 目录（归档不含 pi 配置/会话）\n")
			}
			if res.MarkersNew > 0 {
				fmt.Printf("   新建 marker %d 个；源侧回填 %d 个状态文件\n", res.MarkersNew, len(res.Backfilled))
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVarP(&workspaces, "workspace", "w", nil,
		"workspace path to pack (repeatable; default: current directory)")
	cmd.Flags().StringVarP(&output, "output", "o", "",
		"archive output path (default: ./rick-dump-<timestamp>.tar.gz)")
	cmd.Flags().StringVar(&sessions, "sessions", relocate.SessionsPolicyWorkspaces,
		"pi session history to pack: none | workspaces (only the packed workspaces') | all")
	cmd.Flags().BoolVar(&slim, "slim", false,
		"omit pi-agent runtime/ and npm/ (target re-runs rick tools init-pi)")
	cmd.Flags().StringSliceVar(&goos, "goos", relocate.DefaultDumpPlatforms,
		"platforms to cross-compile into the archive (goos/goarch each, repeatable)")
	cmd.Flags().BoolVar(&noExclude, "no-exclude", false,
		"disable the default exclusion rules (pack everything)")
	cmd.Flags().StringSliceVar(&extraExclude, "exclude", nil,
		"extra exclusion pattern relative to a workspace root, repeatable (e.g. --exclude bigdata/work --exclude '*.ckpt')")
	cmd.Flags().StringVar(&stateDir, "state-dir", "",
		"rick state dir to pack (default: $RICK_STATE_DIR or ~/.rick)")
	return cmd
}

// NewImportCmd creates the import subcommand: unpack a dump archive, merge
// it into this machine's directories, remap path-bound state, port the pi
// session history, and verify. Thin Cobra entry — the work lives in
// internal/env/relocate (job_46 design tree P4/P5/P6).
func NewImportCmd() *cobra.Command {
	var (
		target   string
		stateDir string
		force    bool
	)
	cmd := &cobra.Command{
		Use:   "import <archive>",
		Short: "Unpack a dump archive and make it work on this machine",
		Long: `Unpack a ` + "`rick tools dump`" + ` archive onto this machine: workspaces land
under --target (a single workspace lands there directly when the target is
empty or already holds its marker; otherwise under target/<name>), rick
state merges into --state-dir (default $RICK_STATE_DIR or ~/.rick), and the
pi session history moves into the pi agent dir with every session's header
cwd rewritten — so resume (web session continue / pi --session) keeps
working across machines and paths.

Merge semantics (default — never destroys what is already there):
  files        existing files win; archive files are placed only when absent
  web.json     entries dedupe by path (target wins), archive entries append
  sessions.json / archived.json / job-names.json
               dedupe by id / key (target wins)
  config.json / start-web.sh
               the target's own versions win when present
  pi sessions  dedupe by session file name
  --force      conflicting destinations are backed up first
               (<path>.bak-import-<ts>), then replaced by the archive's
               version

Path remapping (automatic): web.json workspace paths + ids (marker ids are
stable across machines — that is the point of .rick-workspace.json),
sessions.json params._prompt_file/_method_file, config.json
default_workspace, start-web.sh cd lines, and pi session directories
(re-escaped to the new workspace path).

Exit codes:
  0  imported and verified (see the checklist in the output)
  1  archive missing/corrupt (sha256 mismatch vs the .sha256 sidecar), or
     post-import verification failed`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := relocate.Import(relocate.ImportOptions{
				Archive:  args[0],
				Target:   target,
				StateDir: stateDir,
				Force:    force,
			})
			if res != nil {
				printImportResult(res)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ import failed: %v\n", err)
				os.Exit(1)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&target, "target", "",
		"workspace landing directory (default: current directory)")
	cmd.Flags().StringVar(&stateDir, "state-dir", "",
		"rick state dir (default: $RICK_STATE_DIR or ~/.rick)")
	cmd.Flags().BoolVar(&force, "force", false,
		"back up and replace conflicting destinations instead of merging")
	return cmd
}

func printImportResult(res *relocate.ImportResult) {
	if res.SHA256Verified {
		fmt.Printf("✅ 归档完整性校验通过（sha256 边车）\n")
	} else if res.SHA256SkippedReason != "" {
		fmt.Printf("⚠️  %s\n", res.SHA256SkippedReason)
	}
	for _, w := range res.Workspaces {
		fmt.Printf("📥 工作区 %s：%s\n", w.Name, w.DestPath)
		fmt.Printf("   放置 %d 文件 / 跳过已有 %d\n", w.FilesPlaced, w.FilesSkipped)
	}
	fmt.Printf("   状态文件：放置 %d / 跳过已有 %d；web.json 合并 %d 条 / sessions 合并 %d 条\n",
		res.StateFilesPlaced, res.StateFilesSkipped, res.MergedWebEntries, res.MergedSessions)
	if res.SessionsPlaced > 0 || res.SessionsSkipped > 0 {
		fmt.Printf("   pi 会话：放置 %d / 跳过已有 %d（header.cwd 已重写）\n", res.SessionsPlaced, res.SessionsSkipped)
	}
	for _, b := range res.ForceBackups {
		fmt.Printf("🔒 --force 备份：%s\n", b)
	}
	fmt.Printf("验证清单：\n")
	for _, c := range res.Checks {
		mark := "✅"
		if !c.OK {
			mark = "❌"
		}
		detail := c.Detail
		if detail != "" {
			detail = " — " + detail
		}
		fmt.Printf("  %s %s%s\n", mark, c.Name, detail)
	}
	for _, s := range res.NextSteps {
		fmt.Printf("👉 %s\n", s)
	}
}

func basenames(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, filepath.Base(p))
	}
	return out
}

func platformList(plats []string) string {
	if len(plats) == 0 {
		return "无"
	}
	return strings.Join(plats, ", ")
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(n)/float64(div), "KMGTPE"[exp])
}
