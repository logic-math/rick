# RSI 证据：dev-iterations（S1 隔离开发迭代记录）

dev 工作区：`/workdir/sunquan20/rick-dev`（分支 `dev/self-evolve`，HEAD `57f599c`）
dev HOME：`/workdir/sunquan20/rick-dev-home`（dev 实例端口 8414，overlay dist 软链指向 dev 树 `web/dist`）
生产树（只读比对，未触碰）：`/workdir/sunquan20/AI_CODING/rick`

## 构建指纹（本次构建）

```
DEV_UP bin=/workdir/sunquan20/rick-dev-home/bin/rick.dev.57f599c-261010152340 pid=411622 health_ms=201 build_id=57f599c-261010152340 port=8414
DEV_STATUS state=up pid=411622 port=8414 build_id=57f599c-261010152340 fingerprint=ok
```

健康探测（`curl -s http://127.0.0.1:8414/api/health`）：

```
{"status":"ok","build_id":"57f599c-261010152340","started_at":"2026-10-10T15:23:45+08:00"}
```

`build_id=57f599c-261010152340` 为本次构建（sha7=57f599c，时间戳 261010152340）。
换构建前旧指纹为 `907aeaf-260922174111` —— 本次已换成新构建，dev 实例跑的不是旧指纹。

## 变更摘要（本轮 job_44 自进化）

- L1（task1，commit `5c533078`）：human-loop v4 教师会话协议重构——主体提示词重写（文件状态机 + 认知强制 + 退出准则）、teach 新建（SENSE 五关注面仅讲概念本质）、research 调整、think/exporter 全链删除（模板 + 技能 + agent + builder 链 + 版本号）。
- L2（task2+task3，commit `57f599c7`）：humans 学习页面（后端 draft tree/file API 镜像 knowledge 契约 + 前端 `Humans.tsx`/`HumansBrowser` 左树右内容 + 侧栏导航）与书写体验修复（`ChatInput` 草稿按会话 id 持久化 localStorage，刷新/重载/中断不丢）。
- L3（task4，本层）：dev 实例换构建验证 + RSI 证据 S1-S3 留痕 + 试用指引 + `internal/cmd/tools_update_pi.go` 帮助文本清理（`think/research/exporter` → `teach/research`，一行文档修复）。

## 验证结论

- `rick tools dev-web init`（幂等）→ `restart`（build + up）→ 8414 `/api/health` = `status=ok` 且 `build_id=57f599c-261010152340`（本次构建，指纹复核 `fingerprint=ok`）。
- 后端改动（L1/L2）经 `dev-web restart` 生效；前端 `npm run build`（dev 树 `web/`）产出最新 `web/dist`。
- overlay 软链刷新即生效：`/workdir/sunquan20/rick-dev-home/.rick/web/dist -> /workdir/sunquan20/rick-dev/web/dist`；8414 首页引用的前端产物为本次构建。
- 前端产物含 humans 页面：bundle `dist/assets/index-*.js` 命中 `humans`；dev 后端 `GET /api/workspaces/{ws}/humans/tree` 返回学习材料文件树。

## rsi_check 现状（S1-S3 完成时的留档）

命令：`go run ./cmd/rick tools rsi_check --job job_44 --json` → 3/6 通过：

- ✓ dev-iterations：已记录 dev 实例构建指纹
- ✓ gates：3 条门禁记录全绿
- ✓ release：已记录演练版本与回滚点
- ✗ approval：缺人类批准行 —— **S4 未到，属预期挂起**
- ✗ resume：仍是占位骨架 —— **S6 回填，属预期挂起**
- ✗ prod-health：生产 build_id="dev" ≠ 本次 version —— **S5 提升后才可能匹配，属预期挂起**

即：S1-S3 应产出的三项证据（dev-iterations / gates / release）全部通过；
其余三项分别由 S4/S5/S6 完成后转绿。

