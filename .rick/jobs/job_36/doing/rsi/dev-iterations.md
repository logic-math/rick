# dev 迭代记录（job_36）

## 迭代 2026-09-22：rick-rsi-loop 自身修订（11 项，递归 RSI）

- 改动：`.rick/loops/rick-rsi-loop.md` 6533 → 7502 字节（纯文档，无源码改动）
- 内容：P0×3（S5 强制 --detach / 去硬编码 job_36 gate 路径 / 冲突修复流程与实际操作对齐）+ P1×4（依赖准备加引导 / prod-url 参数化 / S7 指明运行位置 / trigger 递归自指）+ P2×4（--init 先建骨架 / 无进展计数操作化 / 版本链卫生 / 重启时长表述）
- 验证：`rick tools loops_check --dir .rick` pass（loops 6）；`gate11` pass（loop 载体 + prod 零触碰）；job_36 残留引用 = 0
- build_id：907aeaf-260922172523（loop 修订后 dev 实例重建指纹）；本次纯文档改动，随 59ac918-260922143616 一并提升

## 迭代 2026-09-22b：/compact 功能补全 + 三个连带缺陷修复

- 用户反馈「web 版 /compact 无法使用」→ 根因：**整条链路未实现**（pi 协议有 compact 命令，rick 的 RpcClient 未封装、无 API 路由、前端无命令分支）
- 修复链路：RpcClient.Compact/SetAutoCompaction → POST /api/sessions/{id}/compact（custom_instructions 可选）→ 前端 /compact 命令
- 连带缺陷①（rpc 超时）：rpcRequest 硬编码 5s——compact 触发真实 LLM 摘要（实测 40.8s）→ 新增 rpcRequestTimeout，compact 走 180s 慢通道
- 连带缺陷②（dev 沙盒不完整 → dev pi 全崩）：种子只拷文件不拷目录 → pi 联网装 pi-web-access 失败（离线 EAI_AGAIN）exit 1 → seededDirs（npm/ + extensions/）+ seedHomeConfigs（~/.config/mcopilot-cli 模型认证）
- 连带缺陷③（**生产级**：proxy 未透传）：release 的 prodEnv/dev-web 的 ServerEnv 白名单丢 http_proxy → 本机无直连外网，deepseek 全部 Connection error（实测直连 000/走代理 401）→ 白名单透传 6 个 proxy 变量 + start-web.sh 显式 export
- 验证（dev 实例实测）：模型链路通（回复「好的」）→ 灌 200KB 上下文 → compact **成功**：202 + 真实摘要（Goal/Constraints/Progress）+ 40.8s
- build_id：907aeaf-260922174111 → 907aeaf-260922193039（compact 迭代期间 dev 实例多轮重启）；提升版本 1c3f085-260922193340
build_id=1c3f085-260922193340（对应 release commit 1c3f085，生产 build_id 实测一致）
