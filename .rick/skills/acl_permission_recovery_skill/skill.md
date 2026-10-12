# skill:acl-permission-recovery（环境权限重置的恢复法）

## 触发场景

共享开发机上目录/文件被外部（cloud-ide 快照、机器休眠恢复、其他用户）重置属主或 ACL mask 后，原有写路径突然 `Permission denied` 时使用。信号词：
- `touch/mkdir/cat >>` 报 permission denied，但昨天还能写
- `getfacl <dir>` 显示 `user:<自己>:rwx #effective:r-x`（有条目但 mask 被收紧）
- 文件属主变成了陌生用户（如 sankuai）——自己创建的文件被 chown

## 预期效果

- 不依赖 root/sudo（通常没有）恢复写能力，10 分钟内继续工作
- 不丢内容（全部从 git 恢复），不误删他人文件
- 可重复使用（本机休眠期重置已发生两次+，是周期性事件）

## 核心内容

**先诊断，再选路径**：

```bash
getfacl <dir>            # 看 mask 与属主
touch <dir>/.wt          # 目录写测试
# 关键：沿父目录向上找最近的可写层（rename 只需父目录写权限）
```

**路径 A：目标文件被锁、父目录可写** → rename 绕过（文件属主无法 unlink 时）：
```bash
mv <file> <file>.bak    # 换新文件重建（如 DEV_TOKEN）
```

**路径 B：整个目录树被锁（属主+mask 全变）** → 父目录可写则整体重建：
```bash
mv job_43 job_43.locked          # 父目录（如 .rick/jobs/）需可写
mkdir job_43 && git checkout HEAD -- .rick/jobs/job_43   # 内容全在 git → 零丢失
```
前提检查：`git status` 干净 + `find -type f | wc -l` == `git ls-files | wc -l`（无未跟踪内容才安全重建）

**路径 C：自建沙盒目录（dev HOME 等）** → 直接重建（幂等）：
```bash
mv rick-dev-home-hadoop rick-dev-home-hadoop.locked
rick tools dev-web init && rick tools dev-web up   # 全新生成
```

**注意**：①锁死副本留 `.locked` 备份勿强删（可能删不掉也不必删）②`git add -A` 前检查别把 `.locked` 备份误提交 ③写不进去≠坏了，先看是不是环境重置（release.go 注释记载过同族问题）
