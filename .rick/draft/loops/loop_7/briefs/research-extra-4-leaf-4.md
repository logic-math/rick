# pi 扩展故障真实案例(badlogic/pi-mono→earendil-works/pi)

URL:github.com/earendil-works/pi/issues/<编号>(旧名同号重定向);日期均 2026 年。

## ① 案例(编号|日期|现象|严重度|解决)
崩溃级:
- #2431|03-19|扩展 registerProvider 缺 api,启动即崩|崩溃|当日修复关闭
- #3556|04-22|emitToolCall 是 runner.ts 唯一未包 try/catch 的 emit,扩展 tool_call handler 抛错即崩 agent 循环|崩溃|关为 not_planned;pi-lens#1655 证实至今抛 "Extension failed, blocking execution"
- #3578/#3595|04-23|footer 扩展退出时访问 stale ctx→关机崩溃,终端残留 raw mode|崩溃|PR#3597(ff220fa)当日合并
- #3606|04-23|官方示例扩展 newSession 后 stale 报错|降级|commit f0cf8a5
- #3510|04-21|扩展工具返回超大文本→渲染 RangeError(Node25)|崩溃|已关闭
- #4909|05-22|v0.75.4 所有扩展工具 "content is not iterable" 崩;根因=无类型 JS 扩展注入 null content|崩溃|PR#6343
- #4333|05-09|pi-web-providers 下 Ctrl+O "Theme not initialized"|崩溃|已关闭
- #7187|日期未取|第三方包 manifest 字段类型错→启动 TypeError 崩,screenpipe 生产全会话被杀;pi -ne 无效|崩溃|状态未查证
- #7731|08-06|0.84.0 起 widget 工厂 Proxy→包装 tui.render 无限递归|崩溃|当日关闭
- PR#4426|约 05 月|官方自述 uncaughtException 最常见触发=扩展异步回调抛错(stale ctx);修复=进程级兜底恢复终端
- #3084|≤04-16|afterToolCall 抛错中止整批工具|批中止|commit b9cd557

跳过/降级级(pi 继续):
- #681(Bun 二进制全扩展加载失败);#1820/#1831(v0.56.0 全局安装模块解析失败);#4528/#6455(依赖解析/原生插件);#6222|07-01(扩展突挂且 extension list 不可用);#7985|08-11(报错误导);#8092(pnpm);#8237(SEA);#1896(本地扩展被静默丢弃);#7154(compaction 致运行时永久失效)。
- CHANGELOG 官方设 pi -ne 无扩展逃生,并修 #8424 失败工厂残留。

## ② 扩展搞挂主进程+官方修复:确有
#2431/#3578/#3595/#4909/#7187 均为实例;官方以 PR#3597、#6343、#4426、f0cf8a5 修复;但 #3556 缺口关 not_planned、第三方证实仍在——tool_call 路径隔离例外。

## ③ 频率量级
issue 编号至 #8600+(08 月);检索定位扩展故障约 25–30 条(崩溃级 10–12),占比<0.5%,量级=数十条,集中 03~05 月后系统修复。DNS 不通无精确计数,属量级估计。

## Gaps
#3556 关闭理由、#4426 合并日、#7187 状态未核验;频率为抽样估计。

## 结论
1. 已记录约 10–12 起扩展致主进程崩溃 issue,多有官方修复。
2. 加载失败多为跳过降级;官方设 pi -ne 逃生并多次修 loader。
3. 扩展故障量级数十条(<0.5%),2026 上半年集中后回落。
