# RSI 证据：dev-iterations

通过标准：至少记录一条构建指纹（`rick.dev.<sha7>-<ts>` 或 `build_id=<sha7>-<ts>`）。
示例：`DEV_UP bin=... build_id=c539c60-260921192330 port=8414`

<!-- TODO: 填写本文件后再跑 rick tools rsi_check（残留本标记即视为未填写） -->


## 迭代 1（2026-10-17）
- 环境：跨机搬迁（linux→macOS arm64）后 dev 树重建——`dev-web init` 重锚定（tree=/Users/logic/workdir/rick-dev home=/Users/logic/workdir/rick-dev-home）；web/node_modules 平台不符修复（rm -rf + npm i）；dev/self-evolve ff→main HEAD 506fa35
- DEV_UP build_id=506fa35-261017095018 port=8414 health ok
- plan 落盘：job_47（14 task / Kahn 4 层 / gate1-4.py），pipeline_gate 通过；commit 见上
- 关键裁决执行：human 复查计划后裁决 ①执行树走 B（dev 工作区）②INFER_RSI 参照=/Users/logic/workdir/INFER_RSI ③realpi 冒烟可跑
- job 编号说明：main 树 job_45=super RFC 会话锚（grilling 产物）；dev 树 job_45=human-loop v5 已发布（rsi 证据陈旧，防假通过）→ 实现取新号 job_47
- 下一步：L1 七 worker 并行派发（task1/2/3/4/10/12/13）
