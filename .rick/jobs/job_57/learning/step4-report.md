# Job_57 Learning Step 4：Loop 模式提取报告

## 结论：**并入既有 loop（不新建）**——更新 `.rick/loops/prompt-template-iteration-loop.md`

## 评估过程

**候选模式**：「prompt 模板多轮实测反馈迭代 loop」（grilling 裁决 → worker 增量 → dev 部署三连 → 建新 loop_N+1 继承产物实测 → human 反馈 → 下一轮——job_57 四轮收敛 v8→v8.3）

**既有 loop 覆盖度对照**（prompt-template-iteration-loop.md 全文核查）：

| 候选模式要素 | 既有 loop 覆盖情况 |
|---|---|
| 反馈→定向模板增量→build/test→dry-run 冒烟 | ✅ worker child A 步 + G1 全覆盖 |
| agent 部署三连（源码同步/重启/init-pi/清单核对） | ✅ B 步全覆盖 |
| 建新 loop_N+1 继承产物实测（先建会话再拷产物/预建目录偏移坑） | ✅ C 步覆盖（含 job_50 的坑） |
| 多轮循环形态（每轮新 loop 承接链） | ✅ 「每轮一个循环」主体覆盖；intro 已补 job_57 四轮实证数据 |
| 停止标准（阶段性满意/连续 2 轮无改善转多方案/形态级转 protocol-redesign-loop） | ✅ 全覆盖 |

**job_57 增量发现（既有 loop 的缺口，已并入修正）**：

1. **「steer 旧会话 ≠ 换模板」防呆规则**（job_57 实际犯过一次：steer loop_20 传达 v8.2 呈现风格，human 纠正「你需要新开 loop 才能触发新的 prompt 逻辑」）→ C 步顶部新增 ⚠️ 硬规则
2. **产物清单过时**：C 步原拷贝清单是 v7 教学产物（mission/zpd/values/syllabus/judgment/lessons/research）——human-loop v8 产物已变为 thought_path.md + orig/ → 更新为「按当前模板的产物清单拷」（v8+/v7 双格式）+ 继承注记补承接链形式（loop_19→…→loop_N）
3. **dev-web 管理命令 go run 踩坑写法**：依赖表与工具表原文用 `go run ./cmd/rick tools dev-web`——job_57 实测 go run 父进程收割 dev web 服务进程（进程即死多次），直跑 `<dev-tree>/bin/rick` 稳定 → 两表改为直跑二进制 + 禁 go run + fingerprint=MISMATCH 竞态以 health 为准
4. **suspended 会话恢复链**（job_57 实测：steer/prompt 均要求 active）→ C 步第 3 步补 resume 前置

## 产物

- 更新：`.rick/loops/prompt-template-iteration-loop.md`（4 处：intro 实证数据 / 依赖表 / 工具表 / C 步重写）
- 不新建 loop 文件（模式已被既有 loop 主体覆盖，避免碎片化）
- 关联：Step 2 改进建议的「dev_session_migration_skill 更新」（loop 迁移双注记坑）与「bugs.md 新条目」在 Step 5 domain 环节处理

## 验证

- `grep -c 'go run ./cmd/rick tools dev-web'` = 0（踩坑写法清零）
- C 步含「禁止 steer 旧会话」硬规则 + v8 产物清单 + 承接链注记
