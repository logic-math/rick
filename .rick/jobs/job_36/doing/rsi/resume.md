# 挂起-恢复记录（job_36 自进化增量）

（按 rick-rsi-loop S6 语义补录：平台自动回来，会话挂起待人工一键恢复，绝不自动续跑）

## 41726a8 提升后（13:43）
挂起清单（3）：eaf6b422（easy job_3 imported）/ 1ce87458（easy job_5 imported）/ 6f1889ff（easy job_15）
恢复结果：未逐条人工恢复（当时用户优先做 /compact 验证；后续迭代中 eaf6b422 被代恢复）

## 1c3f085 提升后（19:34）
挂起清单（承接）：eaf6b422 / 1ce87458 / 6f1889ff
恢复结果：
- eaf6b422：close→continue 重启 worker（加载 catpaw compact 修复）→ 发送测试消息 →
  assistant 回复「通了」→ 随后 /compact 实测成功（202 + 真实结构化摘要，51.3s）✓
- 1ce87458 / 6f1889ff：保留挂起，等用户按需在 UI 点「恢复继续」（挂起可恢复，无数据损伤）

教训：S5 提升会同时停掉「承载 RSI 会话自己的 worker」——web 承载的 RSI 会话在 S5 后
必然挂起，S6/S7 的留痕需要等该会话被人工恢复后才能继续（已在 loop 中补充说明）。
