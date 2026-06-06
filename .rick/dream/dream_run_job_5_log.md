# Dream Run: job_5

## 处理概述

- **处理时间**: 2026-06-04
- **Job 状态**: 已完成反思
- **数据来源**: debug.md（3 条目）+ tasks.json（7 tasks, all success）

## 反思发现

1. **dirname 次数错误**（task5）：测试脚本从 `.rick/jobs/job_N/doing/tests/` 出发，需要 6 次 dirname 到达项目根；原代码只有 5 次，缺少跨越 `.rick/` 层级的那一次
2. **autoFix 干扰测试设计**（task5）：`--auto-fix` 默认开启时，删除 debug.md 后 Claude 自动修复导致测试期望的"失败态"变为"成功态"；修复方案是将 `--auto-fix` 改为 opt-in
3. **字符串否定引用误报**（task2）：测试检查"不含某段文字"时，文件中含对该文字的否定引用，导致 substring 匹配误报；修复：改写源文件措辞
4. **并行 task 接口对齐**（task3）：新增 `KeyResults` 校验后，现有测试用例未包含该字段，需补充

## 变更记录

### Skills 变更
- 新增: `test_script_best_practices.md`（dirname 规范、字符串匹配精确性见陷阱 2/5）
- 修改: 无
- 删除: 无

### SPEC.md 变更
- `路径规范` 条目已存在（6 次 dirname），本 job 作为来源 evidence
- 新增「测试脚本 binary 规范」条目
- 修复2处 `workspace/tools.go` → `internal/workspace/tools.go` 断链

### Wiki 文档
- 无变更（check_mechanism.md 已覆盖 job_5 实现的 check 工具）

## 下次建议关注
1. `autoFix` opt-in 模式已稳定，关注后续 job 中是否有测试设计绕过 check 自动修复的情况
2. 并行 task 的接口一致性问题值得关注（SPEC.md 已有`接口签名协商`条目）
