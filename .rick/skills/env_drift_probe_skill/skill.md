# skill:env-drift-probe（环境反转探针）

## 触发场景

当环境迁移/跨机部署后出现「以前好用的配置现在报连接错误或行为反转」时使用：
- 同一套配置旧机器正常、新机器 `Connection error` / 行为反转
- 代码/脚本注释里写着「必须 X 否则 Y」，但环境已变更（**历史经验注释本身成了 bug 源**）
- 不同入口行为分裂：CLI 会话正常、web/子进程挂

信号词：「Connection error 但 CLI 正常」「历史经验失效」「环境反转」「丢了 proxy 会全部 Connection error」

## 预期效果

- **10 分钟内**钉死「环境假设是否反转」，给出定性结论（环境反转 / 代码 bug / 网络故障）
- 避免在错误层面（代码/网络/权限）盲目排查数小时
- 修复落在**最小改动**（部署配置级一行），并沉淀到部署脚本防复发

## 核心内容

### Step 1：找对照载体（差集 = 嫌疑面）

找一个**工作正常的同类进程**（job_49 = 我的 CLI 会话），对比它的 env 与故障进程的 env：

```bash
# 故障链路：web 起 worker → AgentEnv 继承 web 进程 env；对照：本 CLI 会话 env
env | grep -iE 'proxy|home|state'   # 两边各跑一次取差集
```

差集里每个变量都是嫌疑项（本案：`http_proxy/https_proxy/no_proxy` 三件套——web 有、CLI 无）。

### Step 2：逐变量对照实验（铁证）

同环境**只改一个变量**，其余全等：

```bash
# 复现：旧 no_proxy + proxy
no_proxy=localhost,127.0.0.1 https_proxy=http://10.229.18.27:8412 pi --print hi
# → Connection error.
# 恢复：no_proxy 追加目标域名，proxy 不动
no_proxy=localhost,127.0.0.1,mcli.sankuai.com https_proxy=http://10.229.18.27:8412 pi --print hi
# → 通
```

只有「同 shell 只改一个变量」的对照才构成铁证（同时改两个变量无法归因）。

### Step 3：差异排除法（钉死单点）

测 proxy 对**其他目标**的转发能力，排除「proxy 整体故障」假设：

```bash
curl -x http://10.229.18.27:8412 -sI https://github.com -o /dev/null -w '%{http_code}\n'  # 200 → proxy 活着
curl -x http://10.229.18.27:8412 -s  https://mcli.sankuai.com -o /dev/null -w '%{http_code}\n'  # 000 → 只这个目标不走
```

github 通 + mcli 不通 → 根因钉死在「内网域名路由」单点，**不是 proxy 挂了**。

### Step 4：结论定性

**环境反转 = 老经验在新环境为反**（不是代码 bug、不是网络故障、不是权限）。判据：Step 2 复现/恢复 + Step 3 差异排除后，「环境假设过期」成为唯一解释。

### Step 5：修复落点（最小改动 + 沉淀 + 遗留标记）

- **最小改动**：部署配置级一行（start-web.sh `no_proxy` 追加 `mcli.sankuai.com`）+ 重启 + 实测「通」
- **沉淀**：结论写进部署脚本（新机器/新实例自动带，非逐会话手修）
- **遗留标记**：记录非源码级的残留风险（job_49：release 的 prodEnv 透传调用方 proxy——旧 no_proxy shell 跑 release 会复发；源码级候选修复单列：catpaw-proxy.ts 用 no-proxy dispatcher）

### Step 6：预防（迁移后校准清单）

环境迁移后，对「带历史注释的配置项」做**实测校准**：

1. 列出所有「必须 X 否则 Y」注释涉及的 env / 路径 / 端口
2. 逐项在新环境跑最小探针（通/不通二值，成本极低）
3. 失效项**立即改注释 + 改配置**——旧注释会误导下一个迁移的人

### 案例出处

job_49 P5（`.rick/jobs/job_49/doing/debug/bug1-web-worker-model-connection.md`）：跨机迁移后「模型网关必须走 proxy」经验反转，10 分钟定位 + start-web.sh 一行修复。镜像技能：`.rick/skills/subprocess_env_isolation_skill/skill.md` Step 7。
