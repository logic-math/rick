# RSI 证据：release

通过标准：含 `version=<sha7>-<ts>` 与回滚点（`rollback_point=` 或 `released=yes`）。
示例：
```
version=c539c60-260921192330
rollback_point=bin/releases/a60035a-260921195959
released=yes
```

<!-- TODO: 填写本文件后再跑 rick tools rsi_check（残留本标记即视为未填写） -->


## S3 演练（2026-10-19，job_47+48 合并发布）
RELEASE_DRYRUN version=207e234-261019181758 prod_untouched=true cleanup=ok gate_pass=true(tests+frontend 45s)
首次 dry-run 的 gate 在 internal/web 偶发失败（负载敏感 flaky——env/release 5 个真构建打满 CPU 时 web 的 spawn 全链测试超时）；单包复跑×2 绿后整体重跑通过。遗留观察：web 全链测试对系统负载敏感，可在后续 job 加负载容错（重试或超时放宽）。
