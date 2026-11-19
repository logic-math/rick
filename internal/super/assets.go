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
