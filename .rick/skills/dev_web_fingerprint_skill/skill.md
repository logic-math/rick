# skill:dev-web-fingerprint（dev-web 部署指纹校验——测的必须是建的那个）

## 触发场景

在 dev 8414 上验证任何修复/功能**之前**使用。信号词：
- 刚 `go build` + 重启了 dev web，准备跑无头浏览器或提请 human 验收
- human 验收说「还是不行」但代码确实改了
- 用了 `dev-web up --no-build` 或手动 `./bin/rick web` 启动

## 预期效果

- 消灭「测了半天旧代码」类整轮浪费（job_53 实测：多轮无头验证跑的全是 task49 之前的旧二进制，human 误报一次往返，合计 30-60 分钟 + 信任损耗）
- health 的 build_id 指纹纳入固定动作——旧二进制启动立刻 fail-fast

## 核心内容

### 静默陷阱（机制实证）

`dev-web up --no-build` 复用的是 **dev-home/bin 里最近的 `rick.dev.*` 构建产物**；手动 `go build -o bin/rick` 的产物落在 **dev 树**——两个位置静默分叉，`--no-build` 对「bin 里的比 dev 树 HEAD 旧」零告警，health 报 ok 完全掩盖版本不一致。手动 `./bin/rick web` 启动则报 `build_id:"dev"`，指纹校验彻底失效。

### 指纹前置检查（每次验证前必做）

```bash
SHA=$(git -C /Users/logic/workdir/rick-dev rev-parse --short HEAD)
BID=$(curl -s --noproxy '*' http://127.0.0.1:8414/api/health | python3 -c "import json,sys;print(json.load(sys.stdin)['build_id'])")
[ "${BID:0:7}" = "$SHA" ] && echo "✓ 指纹一致" || echo "✗ 旧二进制！禁止跑测试"
```

### 部署收敛（替代手动 build+kill+start 四连）

```bash
go build -o bin/rick ./cmd/rick/ && \
cp bin/rick "$RICK_DEV_HOME/bin/rick.dev.$(git rev-parse --short HEAD)-$(date +%y%m%d%H%M%S)" && \
RICK_DEV_HOME=$RICK_DEV_HOME ./bin/rick tools dev-web down && \
RICK_DEV_HOME=$RICK_DEV_HOME ./bin/rick tools dev-web up --no-build
```

产物自动入 dev-home/bin（保住 `dev-web status` 的指纹一致性校验）。

### 事故模式（供对照）

human 报「修复无效还是不行」+ health build_id 前缀 ≠ 本轮 commit sha7 = **第一嫌疑就是旧二进制**，先做指纹检查再排查代码。

代码级根治（`up --no-build` 解析 bin 文件名 sha7 与 `git rev-parse --short HEAD` 比对，不一致警告/拒绝启动）记 backlog 未做。
