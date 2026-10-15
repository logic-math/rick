# skill:cross-machine-migration（rick 工作区跨机搬迁操作）

## 触发场景

当需要把 rick 工作区（含 job 历史、pi 会话、配置状态）从一台机器搬到另一台机器时使用：Linux↔Mac 搬迁、A 机分发运行中工作区到 B 机（分布式场景）。信号词：

- 「把工作区迁到 mac / 新机器」「在 B 机上把 rick 跑起来」「搬迁后所有会话正常」
- 目标机是异构平台（源 Linux 目标 darwin 或反之）

前置：源机 rick 需含 `tools dump/import`（job_46 起交付；main 未 release 时从 dev 树构建安装）。

## 预期效果

- 目标机 `rick web` 可启动、全部工作区 job 历史可读、pi 会话可 resume 并带历史续答（job_46 实测：8.6GB / 28.6 万文件 / 3778 会话，端到端模型应答验证）
- 无孤儿化：workspace id（marker）跨机稳定，sessions 引用不断链
- 按本技能走无需发明步骤，踩坑点全部有预案

## 核心内容

**① 源机 dump**（在能读全树的环境执行；先评估体积再决策排除）：

```bash
du -sh <ws>/*/ | sort -rh | head -15    # 体积分布（实测 156G 工作树中 142G 是 ML 权重 → 必须排除）
rick tools dump --workspace <ws> -o /tmp/dump.tar.gz \
  --sessions all \                      # 注册了子工作区时必须 all（workspaces 策略只匹配打包根的转义目录）
  --goos linux/amd64 \                  # 按目标机平台；默认内嵌三平台
  --exclude CAIXI/work --exclude '*.ckpt'   # 相对工作区根，可重复
```

产出 `dump.tar.gz` + `.sha256` 边车（**两者必须一起传**，import 校验完整性）。树上他属主 700 目录/ACL 文件会被跳过并记入 manifest（kind=unreadable）——这是留痕不是失败。

**② 传输 + 目标机 import**：

```bash
scp dump.tar.gz* user@target:~/
# 目标机没有 rick 时用归档内嵌二进制执行 import：
mkdir boot && tar xzf dump.tar.gz -C boot bin/<plat>/rick && chmod +x boot/bin/<plat>/rick
boot/bin/<plat>/rick tools import ~/dump.tar.gz --target ~/workdir
# 布局惯例：工作区 → ~/workdir（单工作区+空 target 直接落位）；状态 → ~/.rick（默认，勿自作聪明挪）
# 默认合并不覆盖已有文件；重布局/要覆盖时 --force（先自动备份 <path>.bak-import-<ts>）
```

**③ 目标机 web 启动**（三项环境缺一不可）：

```bash
# node ≥22.19（pi 必需；缺失时代理下载官方 tar.xz 解压即用，export PATH）
export RICK_STATE_DIR=<非默认时>     # ⚠️ 用非默认 state-dir 必须同时 export RICK_PI_AGENT_DIR=<state>/pi/agent
                                     #    否则启动被安全守卫拒绝（防 pi 沙盒指向别家生产，正确行为）
rick web --listen 0.0.0.0 --port 8413 --daemon    # --daemon 存活不占会话；8413 是对外开放约定端口
```

**④ 验收断言**（逐条 curl；TOKEN=`~/.rick/config.json` 的 `web_token`——config 固定读 `$HOME/.rick`，**不跟随 --state-dir**，非默认 state-dir 部署时 token 是 $HOME 里新生成的那个）：

```bash
curl -s http://127.0.0.1:8413/api/health                                        # status=ok
curl -s -H "Authorization: Bearer $T" http://127.0.0.1:8413/api/workspaces      # 数量=源机、路径已重映射、id 未变
curl -s -H "Authorization: Bearer $T" http://127.0.0.1:8413/api/sessions        # 会话数 > 0
curl -s -X POST -H "Authorization: Bearer $T" .../api/sessions/<id>/resume      # resumed=true → status=active
curl -s -X POST -H "Authorization: Bearer $T" -H "Content-Type: application/json" \
     -d '{"message":"只回复两个字：成功"}' .../api/sessions/<id>/prompt          # 模型带历史应答
```

**踩坑速查**（job_46 全部实测）：

- 远程重启 web：先 `pgrep` 拿精确 pid 再 `kill $PID`；**禁 `pkill -f "rick web"`**——-f 会匹配承载命令自身的 bash -c 字符串，自杀 ssh 会话（exit 255）
- import 重跑幂等（合并语义）；重布局（换 target/state-dir）时先 kill web 再 import，完成后清理旧残留
- git worktree（如 rick-dev）跨机后 `.git` gitfile 断链属预期：手工改指新主仓 `.git/worktrees/<name>` + 主仓 `git worktree repair`
- `--sessions=all` 会带上非随迁 cwd 的会话（import 输出会列孤儿目录清单，按 id 全局检索仍可用，`--continue` 不可用）
