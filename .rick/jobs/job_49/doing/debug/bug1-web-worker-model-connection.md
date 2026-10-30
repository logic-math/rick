# bug1：web worker「无法连接 pi runtime / 模型不能访问」（2026-10-30 23:50 排查）

## 症状
human 在 web UI 对导入会话（ed8419ef "easy job_49 (imported)"）执行 learning，前端报「网页版似乎无法连接 pi runtime，模型不能正常访问」。

## 证据链（S→E）
1. 会话 ed8419ef 状态 active（"imported & resumed"）但 **无 pi worker 进程存活**（ps 无 --mode rpc）
2. worker spawn 用的 env = AgentEnv() = 继承 web 进程 env（web 由 start-web.sh 起，export 了 http/https_proxy）
3. pi 的模型链路：catpaw-proxy provider → `https://mcli.sankuai.com`（内网域名，key 从 ~/.config/mcopilot-cli/.config.yaml）
4. **对照实验（铁证）**：
   - `no_proxy=localhost,127.0.0.1` + proxy → pi --print → `Connection error.`（复现）
   - `no_proxy=...,mcli.sankuai.com` + proxy → pi --print → `通`（恢复）
   - proxy 10.229.18.27:8412 活着且能转发 github（200），但**转发 mcli.sankuai.com HTTP 000**（内网域名不走 proxy 的路由）
   - 我的 CLI 会话 env 无 proxy 变量 → 直连 mcli 正常（这就是 CLI 一直正常而 web 挂的原因）

## 根因
**环境反转**：老机器时代「模型网关必须走 proxy」（start-web.sh 注释所写）；跨机迁移后 mcli.sankuai.com 成内网直达域名，proxy 10.229.18.27:8412 反而转发不了它。带 proxy 的 web worker 全部模型调用 Connection error。

## 修复
- start-web.sh：no_proxy 追加 `mcli.sankuai.com`（已落地+重启 web pid=95589+pi --print 验证「通」）
- ⚠️ 遗留防线（learning 沉淀项）：release.go prodEnv 透传调用方 proxy 变量——若 release 从旧 no_proxy 的 shell 跑，生产实例会复发（本次 S5 起的实例 89800 也中招过，被我的 start-web.sh 重启修复覆盖）。候选源码级修复：catpaw-proxy.ts 扩展用 no-proxy dispatcher（pi 侧定制层，非 rick 源码树）。

## 关联
- 本排查由 human 报错触发（web 执行 learning 报「无法连接 pi runtime」）
- 排查过程全程零触碰生产源码；start-web.sh 是部署配置文件级修改（合法运维面）
