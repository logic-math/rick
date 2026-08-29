# research-extra-3-leaf-3 — dsh 插件规模化风险（本地源码考察）

> 叶子任务：dsh 插件声明/依赖/加载机制、可逆 effect 边界、增删插件耦合 | 方法：只读本地源码 /workdir/sunquan20/AI_CODING/deepseek-harness | 基准 2026-08-25 | 全部事实附本地路径

## ① 插件声明/依赖/加载机制

1. **无 cordis.yml 主配置文件**：插件树由四层有序组合（在空 entry 列表上依序应用）：① profile 的 `dsh.profile.bundles` 列表内每个 bundle 的 patch（按列表顺序）；② profile 自己的 `cordis.patch.yml`；③ home 级 `$DSH_HOME/cordis.patch.yml`；④ `--patch` 覆盖层（argv 顺序）。后层按 row id 整体替换该行 config（不深合并）。[docs/architecture.md:15-27；docs/user/develop/basic/publish.md:115-125]
2. **声明方式**：bundle 在自身 `package.json` 的 `dsh.bundle.patch` 字段指向 patch 文件；profile 的 `dsh.profile.bundles` 列出有序 bundle 层；profile 目录 `package.json` 的 dependencies 由 pnpm 管理第三方插件。[docs/architecture.md:23；publish.md:86-105]
3. **加载顺序=服务需求而非手动序列**：插件声明 `inject`（所需服务名数组），等待服务出现才激活；fiber 状态机 PENDING→LOADING→ACTIVE/FAILED/UNLOADING→DISPOSED。[docs/cordis-primer.md「Declare service dependency via inject」；vendor/cordis/src/fiber.ts:140-155]
4. **`!!js` 表达式在挂载时插值**：entry 的 config 字段在声明注入激活后对该插件上下文求值（`ctx.serviceName`），disabled 字段在每次挂载决策时求值。[docs/cordis-primer.md「Loader Configuration」]
5. **`dsh plugin` = pnpm 转发**：`dsh plugin --profile <name> <args>` 直接转发 pnpm；add 时 pnpm 装包+自动追加 bundle 到 dsh.profile.bundles（仅当包声明 dsh.bundle）；remove 同时移除依赖与层。profile 的 node_modules 由 `healProfilesModuleFallback` 维护为平铺 symlink 目录（BFS 遍历 dsh app 依赖闭包+peerDependencies）。[publish.md:77-110；packages/boot/app-boot/src/profile.ts:206-242]
6. **无 bundle 声明的包只装为普通依赖**：`dsh plugin` 打印警告且不激活任何层。[publish.md:64]

## ② 可逆 effect 边界（回滚覆盖什么/不覆盖什么）

1. **回滚机制**：所有注册（工具 schema/prompt section/适配器/listener/服务注册）经 `ctx.effect()` 或 `ctx.on()` 返回 disposer；插件卸载时 disposer 按注册逆序运行（异步 disposer 并发跑，顺序敏感须合入单个 disposer）；`fiber.dispose()` 递归卸载子插件并等待全部异步清理完成。[docs/cordis-primer.md:13,44；docs/cordis-tutorial/02-lifecycle-and-effects.md:65-94]
2. **服务变化触发卸载-重载**：依赖服务的 fiber uid 变化（服务重注册）→ 依赖插件先 UNLOAD（全部 disposer 运行）再 RELOAD（重新执行 apply）；服务消失 → 卸载后停 PENDING。[vendor/cordis/src/fiber.ts:_setEpoch/_refresh/_reload/_unload（568-680 行区间）]
3. **不可回滚项（已验证源码/文档）**：① session 事件日志——append-only 持久事实，插件卸载不回滚已追加事件（LLM 历史由 log 派生，replay=重派生）[docs/subsystems/session.md:5]；② 插件 apply 里在 Cordis API 之外做的副作用（写文件/起进程/外部系统调用）必须自己包 ctx.effect 否则泄漏——primer 明示「resources managed outside those APIs must be wrapped in ctx.effect()」[docs/cordis-tutorial/02-lifecycle-and-effects.md:5]；③ 动态包（agent 自写插件）「remain active across later turns and may affect other sessions in that process, but disappear after…DSH restart」——进程内存态，非持久 [packages/extensions/tool-cordis/README.md]；④ profile/package.json 磁盘状态变更（add/remove）是普通文件操作，与运行时回滚无关。

## ③ 增删插件的耦合与其他逻辑

1. **boot 是全树事务（无单插件故障隔离）**：任一 entry 激活失败 → 整树 dispose 后抛 `plugin tree failed to load`；晚到的未处理插件 rejection 也经 installFailLoud 转为 exit(1)。故一个坏插件可阻止整树挂载。[packages/boot/app-boot/src/index.ts:757-800（boot 函数 catch 分支 dispose partial context）,609(installFailLoud)]
2. **设置 UI 自身是插件**：插件管理界面 `ui-settings-plugins` inject=['slots','locale','connection','remote','settingsScope']——其依赖（连接/远程服务）不可用时插件管理入口随之消失，与 Discussion #4175 模式同构。[packages/client/ui-settings-plugins/src/client/index.ts:51-53]
3. **移除被依赖服务=级联卸载**：inject 了该服务的全部插件自动 UNLOAD→PENDING（服务回来才 RELOAD），非局部失败隔离而是依赖链传导。[vendor/cordis/src/fiber.ts:_setEpoch；docs/cordis-api/registry.md:26]
4. **已知组合缺陷类别**：postmortem 0001——同一插件多出口形态（`export default apply`）使 Loader 丢弃 inject 声明，178 个绿灯单测+100% 行覆盖未拦截，真实 Zed 连接即崩；修复靠补「真实 Loader 路径」测试。[docs/postmortem/0001-acp-default-export-drops-inject.md]
5. **HMR 用户补丁层**：cordis.patch.yml 变更经 Cordis HMR 事务性重组合整棵补丁列表；坏的读/解析/Loader 候选保留最后好树并广播 hmr/config-update-failed——此为用户补丁层的容错边界（非插件代码层）。[packages/boot/app-boot/README.md:45]

## 结论

1. dsh 增删插件走 profile/pnpm+patch 层，声明面薄，但加载是全树事务：一坏全崩，无单插件隔离。
2. 可逆 effect 只覆盖注册类操作；日志/磁盘/外部副作用与动态包均不在回滚边界内。
3. 服务依赖变化自动级联卸载-重载依赖方；设置 UI 也是插件，坏依赖可带走卸载入口本身。
