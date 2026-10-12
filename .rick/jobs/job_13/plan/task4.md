# 依赖关系
task3

# 任务名称
删除 skills 目录，移除 install.sh 中的 skills 安装/验证逻辑

# 任务目标
完全移除 skills 机制：删除 `skills/` 目录，从 `scripts/install.sh` 中删除 `install_skills()`、`verify_skills()` 函数及其调用，从 `scripts/uninstall.sh` 中删除对应的 `~/.claude/skills` 清理逻辑（如有）。

# 关键结果
1. `skills/` 目录不存在
2. `scripts/install.sh` 不含 `install_skills`、`verify_skills` 函数及调用
3. `scripts/uninstall.sh` 不含 `~/.claude/skills` 相关逻辑（如有则删除）
4. 两个脚本语法检查通过

# 测试方法
1. 检查 skills 目录已删除：`! test -d skills && echo PASS`
2. 检查 install.sh 不含 skills 逻辑：`! grep -q "install_skills\|verify_skills\|claude/skills" scripts/install.sh && echo PASS`
3. install.sh 语法检查：`bash -n scripts/install.sh && echo PASS`
4. uninstall.sh 语法检查：`bash -n scripts/uninstall.sh && echo PASS`
