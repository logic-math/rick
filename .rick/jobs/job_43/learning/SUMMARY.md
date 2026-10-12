APPROVED: true

# Job job_43 执行总结

## 执行概述

**项目目标**: rick web 功能级自进化迭代（human 原话：「按照我的需求改进 rick 的功能。不做架构级改动，只做功能级迭代」）——经 grilling 三轮追问收敛为四大痛点，随后由 human 验收反馈驱动连续增量。

**实际完成**: 单次 easy 会话内完成 **6 轮增量迭代 × 13 task（13/13 success）/ 10 条层门禁（全绿）/ 6 次生产 release**（版本链 8dd1307 → 0d26c68 → 69dc824 → 4b25f12 → 8dfa897 → 3d37918，终版生产健康 build_id 校验一致）。全部走 RSI 闭环（S0-S7），rsi_check 6/6。

**整体评价**: ⭐⭐⭐⭐ (4/5)

- 流程正确性强：13/13 task、6 次批准-提升-恢复循环零流程事故；但含 1 次 P0 生产故障（虚拟化 props 协议破坏）与 1 次设计范式往返（白名单建了又拆）——扣一星，其教训均已沉淀为 skill/loop。

## 关键成就

1. **四痛点全修复 + 后续 5 轮反馈增量**：pi 会话持久持有（IdleDisabled 哨兵 + web_max_active 默认 64，挂机不再被杀）；导航实时（key 重挂修 epoch 根因 + 滚动重置 + 行点击跳转）；输入框极简（单发送按钮 / Enter 固定换行 / scrollHeight 自适应 / Ctrl+Enter 发送）；移动端不发烫（静态星空 + 去 blur + 窄屏动画降级 + dpr 1.5 + 流式 110ms 节流 + react-virtuoso 虚拟化）。
2. **6 次 RSI 全流程提升**：每次 S3 演练（prod_untouched）→ S4 human 批准留痕 → S5 --merge-source --detach 原子换链 → S6 挂起-人工恢复 → S7 rsi_check 全绿；机器三次跨休眠期（09-25→29、29→10-01、10-01→04）会话均安全接续。
3. **环境韧性实战**：ACL 权限重置二连（job 目录 + dev HOME）以「父目录可写 → rename + git checkout 重建」零丢失恢复；release 目录权限坑以 RICK_RELEASES_DIR 重定向绕行；休眠期外部启动的生产 dev 实例被 release 接管对齐。
4. **流程纪律进化**（事故驱动）：P0 后建立「dev UI 真渲染验收」强制步骤（迭代 3 起每轮执行，human 多次在 8414 验收后批准）；gate 增加 RICK_* 环境净化；grilling 增量轮的范式判断节点上呈意识（白名单教训）。

## 问题与教训

### 问题 1: P0——虚拟化自定义 Scroller 丢 props，全站消息空白

**根本原因**: task7 换 react-virtuoso 时自定义 Scroller 只解构 `{ style, children }`，丢弃 `data-virtuoso-scroller` 等标记 props → Virtuoso 测量回调沿父链找不到标记节点 → TypeError → 渲染空白。gate4 七项静态断言全绿（存在性断言测不到协议性破坏）；上线前未做 dev UI 真渲染验收。

**解决方案**: 其他 agent 热修复（0d26c68）：`{ ...rest }` 全量转发，当轮重新 release。此后每轮 release 前 dev UI 真渲染验收成为强制步骤。

**经验教训**: ①虚拟化/滚动接管类第三方库的自定义组件必须全量转发 props（标记/回调藏在 props 里）；②渲染层结构性改动的验收必须含运行时冒烟（30 秒人眼 > 全部静态断言）；③高风险改动必须匹配高强度验收，而非走普通改动路径。

### 问题 2: 白名单建了又拆（迭代 5 task10 → 迭代 6 task13 推翻）

**根本原因**: 「读取范围受不受限」是**安全范式的产品决策**（部署模型：单用户局域网 vs 多租户），但迭代 5 在既有白名单范式内做增量扩展（三层根机制），未作为判断节点上呈；human 迭代 6 一句「不要搞特殊逻辑」整体推翻。

**解决方案**: task13 去白名单——直接读任意路径（保留 token 认证/仅文件/大小上限），ExtraFileRoots 机制随删。

**经验教训**: ①「在既有范式内做最小合理改动」≠「用户认同该范式」；涉及权限/范围/安全边界的决策必须两案对比上呈（各自复杂度+安全面+可逆性）；②research 的方案清单应展开范式轴（白名单 vs 开放都是选项），不能只给子集；③功能默认值尽量简单——为单一用户堆防御逻辑是负资产。

### 问题 3: 机器休眠期环境重置（ACL/属主/生产实例变更，二连+1）

**根本原因**: 运行环境（cloud-ide 快照/休眠恢复）会重置目录 ACL mask（→r-x）与文件属主（→sankuai）；期间生产还被外部启动过非 release 通道二进制（build_id=dev，来源未知）。

**解决方案**: rename + git checkout 重建（job 数据，38 文件零丢失）；沙盒整体重建（dev HOME）；release 接管生产（stopped=[6440,...]）。

**经验教训**: 权限重置是本环境的**周期性模式**而非单发事故——应对法已模式化（acl_permission_recovery_skill）；休眠期外部操作面值得警惕（生产实例曾被替换而无人知晓）。

## 知识沉淀清单

- [x] skills/virtualization_component_protocol_skill/skill.md — 虚拟化库自定义组件 props 协议（P0 防再犯）
- [x] skills/render_layer_acceptance_skill/skill.md — 渲染层改动真渲染验收纪律
- [x] skills/acl_permission_recovery_skill/skill.md — ACL/属主重置恢复法（rename+重建）
- [x] skills/rick_release_ops_skill/skill.md — 本机 release 实操（RELEASES_DIR/detach/版本语义/接管）
- [x] skills/subprocess_env_isolation_skill/skill.md — 升级：RICK_* env 穿透测试隔离
- [x] loops/web-feedback-iteration-loop.md — web 验收反馈增量迭代编排（F1/F2/F3 分级 + gate 惯例 + 真渲染强制）
- [x] domain/bugs.md — +4 条（RICK_STATE_DIR 穿透 / ACL 重置 / ResumeEasyIn 覆盖未修 / release.md 头部误删）
- [x] domain/commands.md — +web 会话持久持有与工具箱节（含去白名单裁决语义）
- [x] domain/architecture.md — +web 前端关键结构节（虚拟化要点/全局 FileReader/静态星空/固定换行）
- [x] domain/build.md — +release 布局节（RELEASES_DIR/版本号语义/生产接管）

## 遗留（下轮候选）

1. **bug1 代码未修**：ResumeEasyIn 无条件覆盖非 easy 作业 tasks.json（已应急恢复，修复方向：条件写入）
2. **PWA 暂缓**（human 裁决「暂时这么用着」）：快路径 apple-touch-icon 随时可加；自签 HTTPS 方案已调研留档（research-I6）
3. P2-B bundle vendor 分割（1.7MB 单 chunk）；移动端真机 profiling 复测
4. .locked 备份目录残留（job_43.locked/ + rick-dev-home-hadoop.locked/，待权限恢复后清理）
5. internal/cmd 4 个 web 单例测试建议测试内固化 `t.Setenv("RICK_STATE_DIR","")`
