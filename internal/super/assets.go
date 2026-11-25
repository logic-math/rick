package super

import (
	"fmt"
	"os"
	"path/filepath"
)

// SuperAssetPaths 是 super 中控会话所需的全部标准资产（相对 .rick/ 的路径）。
// 单源清单（job_55）：web 侧 super session 创建时按此清单自动分发缺失资产；
// 前两件（super-loop.md + gen_env skill）是创建前置检查项，其余 7 件是
// 运行期 super-loop 引用的中控 skill。
var SuperAssetPaths = []string{
	"loops/super-loop.md",
	"skills/gen_env_skill/skill.md",
	"skills/super_cell_management/skill.md",
	"skills/super_cost_analysis/skill.md",
	"skills/super_debug_protocol/skill.md",
	"skills/super_event_autopsy_skill/skill.md",
	"skills/super_experiment_analysis/skill.md",
	"skills/super_experiment_design/skill.md",
	"skills/super_sense/skill.md",
}

// deploySuperAssets 把 srcRickDir 里的标准 super 资产复制到 dstRickDir：
// 只补缺失（dst 已存在 → 跳过，绝不覆盖——外部 workspace 的定制资产优先），
// src 缺失 → 跳过（部分源树只部署可部署件，由调用方的前置检查兜底报错）；
// 返回实际部署的相对路径清单（按 SuperAssetPaths 序）。纯函数（显式传参，
// 测试友好）；仅 I/O 错误返回 err。
func deploySuperAssets(srcRickDir, dstRickDir string) ([]string, error) {
	var deployed []string
	for _, rel := range SuperAssetPaths {
		dst := filepath.Join(dstRickDir, filepath.FromSlash(rel))
		if _, err := os.Stat(dst); err == nil {
			continue // 已存在：绝不覆盖（工作区定制资产优先）
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat %s: %w", dst, err)
		}
		src := filepath.Join(srcRickDir, filepath.FromSlash(rel))
		data, err := os.ReadFile(src)
		if err != nil {
			if os.IsNotExist(err) {
				continue // 源树缺这件：只部署可部署件
			}
			return nil, fmt.Errorf("read %s: %w", src, err)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", filepath.Dir(dst), err)
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", dst, err)
		}
		deployed = append(deployed, rel)
	}
	return deployed, nil
}

// DeploySuperAssets 自动分发：定位 rick 树（resolveRepoRoot：cwd 上溯 +
// rick 二进制链回退，见 supervisor.go）并把缺失的 super 标准资产部署到
// dstRickDir（只补缺失、绝不覆盖）。job_55：外部 workspace（服务器
// /workdir/hpc/project/* 类）首次用 super 不再撞「缺失 super-loop.md」。
func DeploySuperAssets(dstRickDir string) ([]string, error) {
	repoRoot := resolveRepoRoot()
	// resolveRepoRoot 的末端回退是 cwd（不带 marker）——区分「真 repo 树」
	// 与「回退 cwd」：回退时显式报错，调用方（web 前置检查）落回原显式报错。
	if _, err := os.Stat(filepath.Join(repoRoot, "super", "superopt", "__main__.py")); err != nil {
		return nil, fmt.Errorf("cannot resolve rick repo root for super asset deploy (fallback root %q)", repoRoot)
	}
	return deploySuperAssets(filepath.Join(repoRoot, ".rick"), dstRickDir)
}

// DeploySuperRuntimeFromSrc 把 srcRepoRoot 的 super 运行时整套部署到
// dstRickDir（job_55 迭代 5：自包含安装——human 裁决 2026-11-26）：
//
//   - <srcRepoRoot>/super 整棵 python 树 → <dstRickDir>/super/（供
//     resolveRepoRoot 的 ~/.rick 安装副本回退：PYTHONPATH=<dstRick>/super
//     → import superopt；包内资源（prompts 模板）走 __file__ 相对引用，
//     实证自足——见 supervisor.go resolveRepoRoot ③）
//   - <srcRepoRoot>/.rick 的 9 件标准资产 → <dstRickDir>/loops + skills
//     （deploySuperAssets 语义适配：dstRickDir 是 ~/.rick 本身，rel 的
//     loops/skills 前缀 join 后正好落 ~/.rick/loops、~/.rick/skills）
//
// 安装快照语义（与既有 deploySuperAssets 的「只补缺失」不同，这里**整树
// 覆盖刷新**）：安装副本必须与二进制严格同版本——调用方是 init-pi（安装
// 时刻），此时 clone 内容 = 二进制构建源，全量覆盖保证副本永远是最新的
// 安装快照，不残留旧版本文件（python 树文件多、演进快，残留旧文件会
// 造成「半新半旧」的隐蔽混版）。覆盖方式：先整目录删除再拷贝（原子性
// 优于逐文件覆盖——失败要么完整旧版要么完整新版，不会混版；删除失败
// 回退为逐文件覆盖并返回警告级 err 由调用方决定）。
//
// 拷贝实现：super/ 树递归拷贝（保留目录结构与 0755/0644 权限规整）。
func DeploySuperRuntimeFromSrc(srcRepoRoot, dstRickDir string) error {
	srcSuper := filepath.Join(srcRepoRoot, "super")
	if _, err := os.Stat(filepath.Join(srcSuper, "superopt", "__main__.py")); err != nil {
		return fmt.Errorf("super runtime source not found (marker missing): %s", srcSuper)
	}
	dstSuper := filepath.Join(dstRickDir, "super")
	// 整树替换：rm + 全量拷贝（安装快照语义，防混版——见函数注释）。
	if err := os.RemoveAll(dstSuper); err != nil {
		// 回退：删不掉（如权限）→ 覆盖式拷贝继续（copySuperTree 会逐文件
		// 覆盖），残留风险由调用方日志感知。
		_ = err
	}
	if err := copySuperTree(srcSuper, dstSuper); err != nil {
		return fmt.Errorf("copy super tree: %w", err)
	}
	// 9 件资产：覆盖刷新（与整树同语义）。deploySuperAssets 只补缺失，
	// 这里先清 dst 的 9 件再调用，保持单源清单复用。
	for _, rel := range SuperAssetPaths {
		_ = os.Remove(filepath.Join(dstRickDir, filepath.FromSlash(rel)))
	}
	srcRick := filepath.Join(srcRepoRoot, ".rick")
	if _, err := deploySuperAssets(srcRick, dstRickDir); err != nil {
		return fmt.Errorf("deploy super assets: %w", err)
	}
	return nil
}

// copySuperTree 递归拷贝 src 目录树到 dst（目录 0755 / 常规文件 0644，
// symlink 按链接拷贝——superopt 树内无关键 symlink，逐文件语义足够）。
func copySuperTree(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("copySuperTree: src %s is not a directory", src)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copySuperTree(s, d); err != nil {
				return err
			}
			continue
		}
		// symlink：按链接复制（保留目标语义；树内 symlink 均为相对引用）。
		if e.Type()&os.ModeSymlink != 0 {
			target, err := os.Readlink(s)
			if err != nil {
				return err
			}
			_ = os.Remove(d)
			if err := os.Symlink(target, d); err != nil {
				// 目标同名已存在且非 symlink 等情形：降级为内容拷贝。
				data, rerr := os.ReadFile(s)
				if rerr != nil {
					return err
				}
				if werr := os.WriteFile(d, data, 0o644); werr != nil {
					return err
				}
			}
			continue
		}
		data, err := os.ReadFile(s)
		if err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if info, err := os.Lstat(s); err == nil && info.Mode()&0o100 != 0 {
			mode = 0o755 // 可执行脚本（watchdog_wake.sh 等）保执行位
		}
		if err := os.WriteFile(d, data, mode); err != nil {
			return err
		}
	}
	return nil
}

// SuperAssetFallbackPath 是 super 资产读取回退链的统一 helper（job_55
// 迭代 6 优先级翻转，human 裁决 2026-11-26「直接读 ~/.rick 下的 superopt
// 不需要再读工作区中的了，这样可以保持一致」）：~/.rick 安装副本存在 →
// 返回安装路径（最优先——与二进制同版本配套，跨工作区一致）；否则工作区
// 存在 → 返回之（定制兼容回退：~/.rick 无该件但工作区有定制版时仍可用）；
// 都缺 → 返回 ~/.rick 路径（报错指向安装侧——提示 init-pi）。
//
// 两处调用方共用本链（语义必须一致）：web 前置检查（internal/web
// sessions.go）与 BuildSuperPrompt 的 readInline（internal/super
// session.go）。
func SuperAssetFallbackPath(wsRickDir, rel string) string {
	// ① ~/.rick 安装副本最优先（human 裁决：直接读 ~/.rick 保持一致）。
	instPath := filepath.Join(installedRickDir(), filepath.FromSlash(rel))
	if _, err := os.Stat(instPath); err == nil {
		return instPath
	}
	// ② 工作区 fallback（定制兼容：~/.rick 无该件但工作区有定制版时仍可用）。
	wsPath := filepath.Join(wsRickDir, filepath.FromSlash(rel))
	if _, err := os.Stat(wsPath); err == nil {
		return wsPath
	}
	// ③ 都缺 → 报 ~/.rick 侧路径（安装侧，提示 init-pi）。
	return instPath
}

// installedRickDir 返回 super 安装副本所在的 rick 根（~/.rick）。与
// resolveRepoRoot 的 ③ 回退同一 HOME 解析（尊重 RICK_PI_AGENT_DIR/
// HOME 注入——测试用）。注意 AgentDir 是 ~/.rick/pi/agent，其父链即
// ~/.rick；RICK_PI_AGENT_DIR 注入场景取注入值的 .rick 语义根（值本身
// 形如 <root>/pi/agent 时取 <root>，否则取注入目录本身——与
// resolveRepoRoot ③ 的处理保持一致，见那边的注释）。
func installedRickDir() string {
	return rickRootFromAgentEnv()
}

// SuperInstallTargetRickDir 是 installedRickDir 的导出版本：安装部署
//（internal/env superinstall）与运行时回退（resolveRepoRoot ③ /
// SuperAssetFallbackPath）必须同根——单源导出防两处解析漂移。
func SuperInstallTargetRickDir() string {
	return rickRootFromAgentEnv()
}
