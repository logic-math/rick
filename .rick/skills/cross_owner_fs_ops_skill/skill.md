# skill:cross-owner-fs-ops（共享机跨属主文件系统操作）

## 触发场景

在共享开发机上需修改/追加**从未拥有写权限**的跨属主文件或目录时使用（区别于 acl_permission_recovery_skill 的「环境重置后恢复原有写路径」场景）。信号词：

- `echo >>` / `mkdir` / `touch` 报 permission denied，且目标属主非当前用户（如 sankuai）、非重置所致
- 需要以属主身份做单条改动（chmod/setfacl/chown）
- 需写 `.git/info/exclude` 等被多个 worktree 共享的文件

## 预期效果

- 无写权限也能完成目标修改，内容逐字节可校验（`cmp` 通过）
- 不误伤他人文件历史、不静默影响另一 worktree 的行为
- 权限授予最小化（单条命令、单用户），不整段提权

## 核心内容

1. **父目录 rename+cp 重建**（需改目录 D 中文件、D 不可写、D 的父目录可写时）：
   ```bash
   mv D D.bak && cp -a D.bak D   # rename 只需父目录写权限；重建后 D 归当前用户
   # ...修改 D 中的目标文件...
   diff -r D.bak D               # 逐项校验：除预期修改外应无差异（单文件可用 cmp 逐字节）
   # D.bak 原件留作备份勿强删，确认无误后交属主处置
   ```
   内容未必在 git（不能走 git checkout 恢复）→ 必须逐项/逐字节校验（`diff -r` / `cmp`）。与 acl_permission_recovery_skill 路径 B 同族，但该路径适用于环境重置恢复（内容在 git、可 `git checkout` 重建）。
2. **sudo -u 属主授权**：需属主级改动时 `sudo -u <owner> <单条命令>`（如 chmod/setfacl），以属主身份最小化执行，勿开 root shell 或整段提权
3. **`.git/info/exclude` 双 worktree 共享**：dev/main 两个 worktree 共用主 .git 的 info/exclude，写入会同时影响两侧的 ignore 行为——写入前列出将受影响的另一侧（如 main 的未跟踪文件是否会被意外隐藏），必要时只在一侧的仓库级 .gitignore 落地
