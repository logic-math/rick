# 依赖关系
task2

# 任务名称
编写安装与部署文档

# 任务目标
创建 `wiki/installation.md`，全面介绍 Rick CLI 的安装和部署方法。包含系统要求、安装方法（源码安装、二进制安装）、脚本详解（build.sh, install.sh, uninstall.sh, update.sh）、配置文件说明、版本管理（生产版 vs 开发版）、自我重构工作流、常见问题排查。

# 关键结果
1. 完成 `wiki/installation.md` 文档创建
2. 说明系统要求（Go 版本, Claude Code CLI, Python3 等）
3. 详细介绍安装方法（源码安装, 二进制安装）
4. 详细讲解脚本功能（build.sh, install.sh, uninstall.sh, update.sh）
5. 说明配置文件格式和所有配置项（config.json）
6. 说明版本管理机制（生产版 ~/.rick vs 开发版 ~/.rick_dev）
7. 提供自我重构工作流示例
8. 提供常见问题排查指南

# 测试方法
1. 验证文件已创建：`test -f wiki/installation.md && echo "PASS" || echo "FAIL"`
2. 检查包含核心章节：`grep -q "## 系统要求\|## 安装方法\|## 脚本详解\|## 配置文件\|## 版本管理\|## 常见问题" wiki/installation.md && echo "PASS" || echo "FAIL"`
3. 验证包含安装脚本说明：`grep -q "build\.sh\|install\.sh\|uninstall\.sh\|update\.sh" wiki/installation.md && echo "PASS" || echo "FAIL"`
4. 验证文档长度（至少 120 行）：`wc -l wiki/installation.md | awk '{if($1>=120) print "PASS"; else print "FAIL"}'`
5. 检查包含配置相关内容：`grep -q "config\.json\|max_retries\|claude_code_path\|\.rick\|\.rick_dev" wiki/installation.md && echo "PASS" || echo "FAIL"`
