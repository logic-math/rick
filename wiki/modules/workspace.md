# 工作空间管理模块（internal/workspace）

## 职责

`internal/workspace` 是第四层「基础设施」，负责路径解析。它可被任意层直接使用（不参与功能调用链的「逐级往下」约束）。

## 路径常量

```go
const (
    RickDirName     = ".rick"
    LoopsDirName    = "loops"
    SkillsDirName   = "skills"
    DomainDirName   = "domain"
    JobsDirName     = "jobs"
    PlanDirName     = "plan"
    DoingDirName    = "doing"
    LearningDirName = "learning"
    DreamDirName    = "dream"
    DraftDirName    = "draft"
)
```

## 关键函数

| 函数 | 返回 |
|------|------|
| `GetRickDir()` | cwd 下的 `.rick`（二进制名含 `_dev` 时用 `.rick_dev`） |
| `GetJobDir(jobID)` | `.rick/jobs/<jobID>` |
| `GetJobPlanDir(jobID)` / `GetJobDoingDir(jobID)` / `GetJobLearningDir(jobID)` | 对应阶段目录 |
| `NextJobID()` | 扫描 jobs 目录返回下一个 `job_N` |
| `GetDraftDir()` / `GetRFCDir()` | `.rick/draft` / `.rick/draft/rfc` |
| `NextLoopID(draftDir)` | 扫描 `draft/loops` 返回下一个 `loop_N` |
| `LoadTasksJSON(path)` | 读取 doing/tasks.json |

## .rick/ 目录结构

```
.rick/
├── loops/              # 可复用工作流（{name}-loop.md）
├── skills/             # 原子能力（{name}_skill/skill.md）
├── domain/             # 事实信息（spec.md / rick-spec.md / 命令规范 / 架构）
├── draft/              # 个人判断（rfc/ concepts/ human-learning/ loops/loop_N/）
├── jobs/               # 工作区（job_N/{plan,doing,learning}）
└── dream/              # dream 运行日志
```

> 旧架构的 `.rick/wiki/`、`.rick/SPEC.md`、`.rick/OKR.md` 已于 v2.9.0 迁移删除：wiki → skills/ + domain/，SPEC → domain/（拆分）+ skills/，OKR → job_N/plan/OKR.md。
