检查当前.rick的上下文是否符合预期

---

## Grilling 澄清结论（2026-07-02）

**检查范围**：结构 + 文件内容（不含与 Go 代码实现的一致性校验）
**基准**：v2.9.0 架构规范（loops/skills/domain 三层，无 wiki/tools/SPEC）
**交付形式**：终端输出 check report（符合/不符合，列出问题项）

**重点关注**：
1. Skills / Loops 的触发概率分析 — 当前 trigger 描述是否足够精准？如何强化触发概率？
2. Domain 读写信息流 — domain/*.md 是否被正确加载到 prompt？dream/learning 阶段是否有写回路径？