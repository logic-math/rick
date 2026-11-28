# skill:avatar-provision（rick 分身机制——新机器安装 rick + 知识/记忆迁移 + 环境适配）

## 触发场景

当需要在**一台新机器上把 rick 作为「分身」跑起来**时使用——即在远程机器上安装 rick、创建工作区、复制知识资产（domain/loops/skills）、迁移记忆（pi 会话历史 + job 历史），并让分身具备独立工作能力。信号词：

- 「在这台机器上装一个 rick 分身」「把 rick 部署到 XX 服务器」「让 XX 机器也能跑 rick」
- 「分身要有我的 domain/loops/skills」「分身失忆了」（= 装了二进制但知识资产没复制）
- 内网服务器部署（job_55 实战：10.99.19.123 super 版 rick）

前置：主身 rick 含 `tools dump/import`（job_46 起）；分身机有 node（pi runtime 需要）。

## 预期效果

- 分身机 `rick web` 可启动、health 通过、token 可用
- 分身的工作区含完整知识资产（domain/loops/skills + workspace 的 .rick/）
- 分身记忆迁移后：pi 会话可 resume 带历史续答、job 历史可读
- 环境适配完成：IP/密码/token/路径/代理全部按分身机重写——不会拿主身的密钥连分身的资源
- 不留明文密码在分身机的 git 可追踪文件里

## 核心内容

### ① 连接方式发现（顺序不可换——job_55 教训：盲扫 5 端口浪费 3 轮）

```text
1. 问 human「分身机的访问方式」（最短路径，一句话）
2. 查 domain/bugs.md、domain/env.md 里有无该机器的历史记录
3. 查 ~/.ssh/config（有无 Host 别名/跳板）
4. 最后才扫端口：ssh -o ConnectTimeout=3 <ip> -p 22/2222/8080/9090/6137
```

沙箱直连模式（内网机常见）：`sshpass -p '<密码>' ssh -o StrictHostKeyChecking=no <user>@<ip>`。

### ② 二进制安装（按分身机平台选路）

| 分身平台 | 方式 | 命令 |
|---|---|---|
| Linux | dump 交叉编译 | 主身 `rick tools dump -o /tmp/rick.tar.gz --goos linux/amd64`（归档含 bin/linux-amd64/rick） |
| macOS（同版本） | scp 现成二进制 | `scp bin/rick <user>@<ip>:~/rick/bin/` + `shasum -a 256` 两端比对 |
| 源码树分发 | git 仓库（内网 git） | 分身机 `git clone <repo>`（job_55 实战：infer-rsi.git 内网 super 版） |

**校验必做**：tar.gz + `.sha256` 边车必须一起传，import 侧校验完整性。

### ③ 工作区创建 + 知识资产复制

```bash
# 分身机：
mkdir -p <workspace>/.rick/{jobs,loops,skills,domain,draft}
# 主身侧复制知识资产（三选一）：
# 全量：dump（--workspace + --sessions 见 avatar-session-replication）
# 定向：rsync -av --exclude='.git' 主身/.rick/{loops,skills,domain} 分身:<workspace>/.rick/
# git 同步（源码树分身）：git push 到分身可拉的内网分支（job_55 实战：dev_rick 分支）
```

**分身失忆根因**（job_55 修复实证）：web `createWorkspaceDirs` 只建**空目录**零知识复制——super 9 件资产（loops/super-loop.md + skills/gen_env_skill 等）不会自动到位，报「找不到 super loop 文件」。新分身装好后**必须显式复制知识资产**，或用已含 DeploySuperAssets 的版本（web 创建会话时自动补 9 件 super 资产）。

### ④ 环境适配（IP/密码/token/路径/代理重写清单）

```bash
# 分身机 ~/.rick/config.json：
# - web_token：换新生成的（openssl rand -hex 16）——绝不用主身 token（主身能连 ≠ 分身该共用身份）
# - pi_path：确认分身机 node + pi runtime 路径（rick tools init-pi）
# - 代理：内网机常见 http_proxy 三件套——检查 mcli.sankuai.com 等网关域名是否需 no_proxy 直连
# 路径重写：主身 /Users/... ≠ 分身 /workdir/...——见 pi_session_portability（pi 会话目录按 cwd slug 转义）
```

**密码纪律**：sshpass 密码只进临时环境变量/命令行历史敏感区，**不写进** domain/git 可追踪文件（记录时写「密码见 1Password/human 口述」）。

### ⑤ 记忆迁移（全量）

```bash
# 主身（评估体积再决策排除——cross-machine-migration 先验：156G 工作树 142G 是 ML 权重）
du -sh <ws>/*/ | sort -rh | head -15
rick tools dump --workspace <ws> -o /tmp/dump.tar.gz --sessions all --goos linux/amd64 \
  --exclude '*.ckpt' --exclude bigdata/work
scp /tmp/dump.tar.gz /tmp/dump.tar.gz.sha256 <user>@<ip>:/tmp/
# 分身机：
rick tools import /tmp/dump.tar.gz          # 自动：解包 + 路径重映射 + pi 会话移植 + marker 合并 + 校验
```

dump 参数表：`-w/--workspace`（可重复）｜`-o/--output`｜`--sessions none|workspaces|all`｜`--slim`（省 runtime，分身机重跑 init-pi）｜`--goos`（交叉编译平台，可重复）｜`--exclude`（相对工作区根，可重复）｜`--state-dir`。

### ⑥ 验证清单

```bash
# 分身机：
rick web --port <port> &          # 或 launchd/systemd 按 avatar-upgrade-debug
curl -s --noproxy '*' http://127.0.0.1:<port>/api/health   # status ok + build_id
curl -s -H "Authorization: Bearer <token>" http://127.0.0.1:<port>/api/workspaces
# 知识资产抽检：ls <ws>/.rick/{loops,skills,domain} 非空 + 关键文件在位（如 loops/super-loop.md）
# 记忆抽检：web 会话列表可见迁移会话 + resume 带历史续答（pi_session_portability 验证法）
```

## 踩坑预案

- **分身失忆**（知识资产空）：见 ③ 根因——显式复制或升级到含 DeploySuperAssets 的 rick 版本
- **pi 会话 resume 找不到**：pi 会话目录按 cwd slug 转义（`--` + cwd 去首 `/`、`/\:` 全替换 `-`、下划线不替换）——分身机工作区路径必须与 jsonl 落位 slug 一致；不匹配用 pi_session_portability 的最小改写集
- **跨进程树 stdio**：分身上任何常驻进程编排（类似主身 daemon 架构）——stdio 一律落文件不共享管道（EPIPE 教训，见 domain/pi-runtime.md）
- **内网 git 权限**：内网仓库（git.sankuai.com）需要分身机的 ssh key/部署密钥——提前配好或走 tar 包分发

## 待实战补全（如实标注）

- **分身认证体系**：token 分发/轮换的正式机制（当前=手工 config.json 换 token）
- **跨平台服务管理**：Linux 分身的 systemd unit（本 skill 只实战过 macOS launchd + 手动 nohup）
- **分身状态上报（推模式）**：分身主动向主身上报心跳/进度（当前只有主身拉取：GET API）
- **多分身编排**：同一主身管理多个分身的拓扑/路由/命名（当前实战仅单分身）
