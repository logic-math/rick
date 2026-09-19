## [job_35] Domain 事实同步 - 2026-08-17

### 新增已知问题与解法
- **`git add bin/rick` 静默失败（bin/ 在 .gitignore）**：必须用 `git add -f bin/rick` 强制暂存，否则 feat commit 漏二进制（来源：domain/bugs.md）
- **tasks.json `updated_at` 缺时区导致 Go time.Parse 失败**：每条 `updated_at` 必须带 RFC3339 时区（`+08:00`），mark_task_success.py 已用 `timezone(timedelta(hours=8))` 自动修复（来源：domain/bugs.md）

### 新增架构/构建事实
- **四层架构 + 5 模块 + env 四职责契约**：cli → handler → builder → runtime/env/workspace/prompt 四层，pi 调用逻辑收口到 runtime 层（来源：domain/rick-spec.md）
- **构建/门禁命令链**：`go build -o bin/rick ./cmd/rick` + `git add -f bin/rick` + `go test ./... -timeout 120s` + `python3 .rick/skills/rick-gates/helper.py .rick/jobs/job_N/doing`（来源：domain/build.md）

### 其他新增事实
- rick 自定义 agent（think/research/exporter）经 env 职责 3 `deployRickAgents` 幂等落盘到 `~/.rick/pi/agent/agents/`（来源：domain/env.md）

## [job_17] Domain 事实同步 - 2026-09-19

### 新增已知问题与解法
- **bash 工具后台进程 pid 跨调用不可靠（nohup $! / kill %1 双形态 + 假阳性回执）**：`pkill -f <程序名>` pattern 匹配清理 + `ss -tlnp | grep <端口>` 复核；自终止探活配方 `timeout 15 python3 <prog>`；「ps -p $PID 干净退出」回执为假阳性（验证的是包装层 pid）——起服 worker 2/2 系统性残留实证（来源：domain/bugs.md）
- **派发前闸门信号量纲混用（预计产出 >300 行被响应为写入层缓解而非派发层拆轮）**：修正为单次派发全部段合计口径——合计 >700 行或三段结构且任一段 >500 行 → 首派即拆轮（校准 n=3：634 过 / 860 截断）；窄边界重派模板（三条件：已验证资产 + 机器可判验收 + 极窄边界）实证一次通过（来源：domain/bugs.md + skills/subagent_truncation_recovery_skill）

### 新增架构/构建事实
- **web 版交付基线**：全量 not-slow 383 passed + 9 deselected（job_9 基线 327 零回退 + 56 新增）；gomoku_web.py 546 行纯标准库 http.server + web/ 三件套 vanilla JS；层门禁 `python3 .rick/jobs/job_17/plan/gates/gate{1,2,3}.py`（来源：domain/build.md + domain/web.md）
- **e2e HTTP 测试纪律**：build_server(host, port, store) 测试缝（注入 store + 端口 0）+ 后台线程 serve_forever + teardown 三连；HTTP 客户端必须 ProxyHandler({}) 绕代理；4xx 断言走 HTTPError 分支（来源：domain/web.md）

### 其他新增事实
- **公司 docker 环境网络分段**：18080 端口对 human 访问网不通，8000 网段可达——对外交付 web 服务优先 8000 左右网段（来源：domain/env.md，human 冒烟反馈实证）
- **GOMOKU_WEB_HOST/GOMOKU_WEB_PORT/GOMOKU_WEB_LOG 环境变量**（默认 0.0.0.0/18080/安静）；本机 http_proxy 已导出致 urllib 127.0.0.1 请求假 503（来源：domain/env.md）
