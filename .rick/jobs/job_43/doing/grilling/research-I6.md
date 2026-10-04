# research-I6 简报：rick web PWA 跨平台安装可行性矩阵与达成路径（job_43 增量 6 前置）

> 阶段：增量 6 前置调研 | 主题：http://LAN-IP:8413 下 Win/Android/pad/iOS 的「安装为桌面应用」 | 基准：2026-10-01 · dist 8dfa897 | 方法：联网查证（web.dev/Chrome 官方/OpenPWA/firt.dev/SO）+ 本地构建产物核对

## 事实性结论（12 条）

1. **Chrome 正式安装（Install Page as App）硬要求安全上下文**：HTTPS（或 localhost）+ manifest（name/icon 192/512/start_url/display）+ 互动启发；`http://LAN-IP` 一律报「not served from a secure origin」不可装。[web.dev install-criteria + SO，高]
2. **SW 已非安装必要条件**：Chrome 108(Android)/112(桌面) 起菜单安装不再要求 SW fetch handler；但 http 下 SW 本就不注册（安全上下文限制）——rick 的 SW/离线缓存在 http 访问下整体缺席，安装门槛本身不受影响。[OpenPWA installability + MDN，高]
3. **Win Chrome 在 http 下可「另存为快捷方式→作为窗口打开」**：Chrome 72+ 恢复「Open as window」，对任意 http 站可用——产生无地址栏独立窗口 + 桌面图标（favicon），体验接近装 PWA（菜单 2024 更名 Cast,save,share→Create shortcut；Chrome 128 起对非可装页生成快捷方式）。[howtogeek/SuperUser/Chrome 官方帮助，高]
4. **Win「安装为应用」在 http 不可用**：Install page as app 仅出现在可安装（安全上下文）页面；http 页菜单只有快捷方式项。[Chrome 帮助 + Chromium 安装管线文档，高]
5. **Android Chrome http 下只能加普通快捷方式**：菜单「安装并创建快捷方式→创建快捷方式」对任意站可用，但产物是打开 Chrome 标签页（带地址栏）的主屏图标；「安装应用」（WebAPK，独立窗口/无地址栏）仅 HTTPS 站出现。[Chrome Android 帮助 + 9to5google，高]
6. **iOS Safari/Chrome 均可对 http 站「添加到主屏幕」**：纯用户手势（Share→Add to Home Screen），iOS 16.4 起第三方浏览器（含 iOS Chrome，WebKit 内核）也有该入口；iOS 26 增加「Open as Web App」开关。[OpenPWA + WWDC23，高]
7. **iOS standalone 由 manifest display 驱动（11.3+）**：rick dist 已注入 `<link rel="manifest">` 且 display:standalone——iOS 添加主屏后无地址栏独立运行**在 http 下也生效**（A2HS 不要求安全上下文）。[firt.dev iOS PWA 兼容表 + OpenPWA，中高]
8. **iOS 图标缺失会退化截图**：index.html 无 `apple-touch-icon`，iOS 添加主屏用页面截图当图标；需补 `<link rel="apple-touch-icon" href="/icon-192.png">`（public 已有 PNG）。apple-mobile-web-app-capable/status-bar-style 为 11.3 前兼容，现代 iOS 可不加（manifest 已覆盖），加了无害。[Apple Safari Web Content Guide（存档）+ 本地 dist 核对，高]
9. **Safari 18.2（2024-12）起 HTTPS 优先**：iOS 输入/打开 http URL 先尝试 HTTPS、失败回退并可能弹「不安全连接」确认——http 访问仍可用但多一步确认，体验打折。[lapcatsoftware + fstab 实测，高]
10. **自签证书全平台可行但需每设备一次性信任**：mkcert 生成含 LAN IP SAN 的证书（`mkcert <IP>`），rootCA.pem 分发到手机（二维码/HTTP/AirDrop）；Android 设置→安装 CA 证书（Chrome 信任用户 CA）；iOS 下载描述文件→设置信任。完成后 https://IP:8413 无警告、SW 注册、全平台完整 PWA 安装。[mkcert README + SO 实战 + web.dev local-https，高]
11. **Go 加 TLS 成本极小**：`srv.ListenAndServeTLS(certFile, keyFile)`（或 --tls flag 切换），无新依赖；证书放 ~/.rick/ 下由 flag 指定即可（与 start-web.sh 集成一行改动）。[Go 标准库常识，高]
12. **rick 现状**：vite-plugin-pwa 产物齐备（manifest standalone/图标 192+512+maskable/SW autoUpdate），**唯一缺口是 iOS 的 apple-touch-icon 链接**（dist 核对：只有 manifest link）。[本地 dist 核对，高]

## R7 上报（无法自行澄清）

1. 用户是否接受「每台设备一次性安装/信任 mkcert 根证书」的操作成本（Android ~5 步 / iOS ~6 步）——决定走完整 HTTPS 还是先零成本快捷方式
2. 服务器 LAN IP 是否固定（DHCP 变动会使证书 SAN 失效，需重签或用内部域名）
3. iOS 用户实际用的是 iOS Safari 还是 iOS Chrome（两者 A2HS 入口路径不同，引导文案需按平台区分）

## 体验矩阵（平台 × 访问方式）

| 平台/浏览器 | http://LAN-IP（现状，零改动） | 自签 https://LAN-IP（一次信任） |
|---|---|---|
| Windows Chrome | △ 菜单「另存为快捷方式」勾选**作为窗口打开**→ 无地址栏独立窗口 + 桌面图标（favicon）；无离线缓存；地址栏区域可能显示「不安全」细条 | ✅ 完整 PWA「安装页面为应用」→ 独立窗口 + 正式图标 + SW 自动更新 |
| Android Chrome | △ 主屏**普通快捷方式**（打开 Chrome 标签页带地址栏）；无独立窗口 | ✅ 「安装应用」WebAPK → 独立窗口/无地址栏/任务切换器独立条目 |
| Android pad Chrome | △ 同上（快捷方式，标签页打开） | ✅ 同上 |
| iOS Safari（及 16.4+ iOS Chrome） | ✅ **standalone 可用**：分享→添加到主屏幕 → 无地址栏独立运行（manifest display 生效，不要求 https）；图标退化截图（补 apple-touch-icon 后正常）；Safari 18.2 起 http 打开可能多一步「不安全连接」确认 | ✅ 同左且无警告 + SW 离线缓存 |

注：iOS 是四平台中唯一「http 下也能拿到接近完整体验」的；Win 次之（窗口模式）；Android http 最弱（仅快捷方式带地址栏）。

## 达成路径（三选，可叠加：先 A 后 B）

- **A 零成本快路径（前端 1 行）**：index.html 补 `<link rel="apple-touch-icon" href="/icon-192.png">`（可顺带 apple-mobile-web-app-status-bar-style=black-translucent）→ iOS 全量体验修复；Win 用「快捷方式→作为窗口打开」；Android http 下接受标签页模式。工作量：极小（1-3 行 + release）
- **B 自签 HTTPS 完整 PWA（推荐叠加）**：`rick web` 加 `--tls-cert/--tls-key` flag（Go 标准库，~30 行 + 测试）；mkcert 生成含 LAN IP 的证书；文档化每设备信任流程（Android 设置装 CA / iOS 描述文件）；start-web.sh 改 https。工作量：小（后端 flag + 脚本 + README 指引）
- **C 不推荐**：chrome://flags 逐设备白名单（不可分发的 hack）；公网域名+真证书（引入公网暴露面，违背局域网工具定位）

## 推荐组合

**A + B**：先落 A（iOS 立即完整），再做 B 一次性解决 Win/Android 的正式安装 + 全平台无警告。B 的判断点（R7-1/R7-2）建议 human 裁决后进 task。

## 关联现状

- rick web 监听 0.0.0.0:8413（http）；vite.config 注释已记载「LAN IP over http 下 SW 自动降级」——与结论 2 一致
- PWA 插件配置在 web/vite.config.ts:36-59；图标 web/public/icon-{192,512,maskable-512}.png 齐备
