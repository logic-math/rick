# S6 挂起-恢复记录（job_43）

## 重启对账（release 02:00:53）

挂起清单（suspended=1）：
- e7dbebd0-5a39-444c-ad96-9d67f87dd223（easy job_43）——**本 RSI 迭代的承载会话**（web 实例重启，原 pi 进程 142585 终止）

## 人工恢复（02:08:52）

- human 在 web UI 点击「恢复继续」→ 本会话 resume（新 pi 进程 192787，加载原会话历史）
- recovery-report.json：recovered=1 reason="resumed by human"，failed=0
- 恢复后继续执行 S7 留痕校验
- 其余在跑会话：无（重启时仅本会话 active）
