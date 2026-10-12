APPROVED: true

# Job job_13 执行总结

## 执行概述

**项目目标**: 将 human-loop 的三个 sub agent 从 skills 机制迁移到 rick prompt 模板系统，直接嵌入主 prompt，完全解耦 Claude Code CLI
**实际完成**: 4/4 任务全部成功，零重试
**整体评价**: ⭐⭐⭐⭐⭐

## 关键成就

1. **Sub Agent 模板系统**: 新建三个 RFC 规范提示词模板（think/learn/express），通过 Go embed 注入主 prompt，完全替代 skills 斜杠命令机制
2. **渐进式加载设计**: human_loop.md 主控模板只注入 sub agent 文件路径，AI 按需读取，降低初始 prompt 长度，符合"按需加载"设计原则
3. **Skills 机制完全退出**: 删除 skills/ 目录，移除 install.sh/uninstall.sh 中所有 skills 相关逻辑，架构更简洁
4. **check_prompt_variables.py 扩展**: 支持 `--phase human-loop` 参数，human-loop dry-run 现在可用统一工具验证

## 问题与教训

### 问题1: task.md 测试方法使用了不存在的工具接口

**根本原因**: task3.md 中测试方法引用了理想化的 `--command`/`--variables` 接口，但 check_prompt_variables.py 从未实现这些参数；plan 阶段生成测试脚本时直接套用 task.md 描述的接口，没有验证工具实际支持的参数。

**解决方案**: 
1. 扩展 `check_prompt_variables.py`，新增 `check_human_loop_prompt` 函数，支持 `--phase human-loop --topic` 接口
2. 修复 task3.py 中 `build_and_get_rick_bin.py` 的 JSON 输出解析（提取 `bin_path` 字段）
3. 移除对 `/tmp/` 真实路径的检查（dry-run 使用占位路径 `<tmp>/...`，不写真实文件）

**经验教训**: task.md 的"测试方法"必须基于**现有工具实际接口**来描述，而非假想接口。plan 阶段应验证 tools/ 下的工具是否支持 task.md 描述的调用方式。

### 问题2: dry-run 模式与真实运行的文件路径不一致

**根本原因**: dry-run 使用占位路径 `<tmp>/human_loop_think_*.md`，真实运行生成实际 tmp 文件；task3.md 测试方法 5 试图在 dry-run 输出中查找真实 `/tmp/` 路径，导致失败。

**解决方案**: 移除对 dry-run 输出中真实 tmp 路径的检查。dry-run 的正确验证目标是"占位路径关键词存在"，而非"真实路径存在"。

**经验教训**: dry-run 和真实运行的验证策略需要区分。dry-run 验证占位词，集成测试验证真实路径。

## 技术总结

### 关键技术决策

- **渐进式加载 vs 内联**: 选择路径注入而非内联 sub agent 内容，主控 prompt 更简洁，sub agent 按需加载。代价是需要 AI 执行时自行读取文件，增加一步文件 I/O。
- **Go embed + tmp 文件模式**: 模板内容通过 embed 编译进二进制，运行时写出到 tmp 文件，AI 通过文件路径访问。这套模式统一了所有 prompt 模板的分发方式。
- **Tmp 文件清理**: 使用 `defer` 在 human-loop 会话结束后清理所有 tmp 文件（主 prompt + 三个 sub agent），避免文件泄漏。

### 知识沉淀清单

- [x] wiki/human_loop_subagent_pattern.md - human-loop sub agent 路径注入模式
- [x] tools/check_prompt_variables.py - 更新：支持 human-loop 阶段验证
- [ ] OKR.md - 无需更新（本次 job 未改变 OKR 目标）
- [x] SPEC.md - 更新：human-loop 命令规范补充 sub agent 路径注入说明
