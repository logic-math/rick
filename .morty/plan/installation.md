# Plan: installation

## 模块概述

**模块职责**: 实现 Rick CLI 的安装、卸载、更新脚本，支持生产版和开发版的并行管理

**对应 Research**:
- `.morty/research/DEVELOPMENT_GUIDE.md` - 安装脚本规范
- `.morty/research/RESEARCH_SUMMARY.md` - 版本管理机制

**现有实现参考**: 无

**依赖模块**: infrastructure, cli_commands

**被依赖模块**: e2e_test

## 接口定义

### 输入接口
- 安装选项（--source, --binary, --dev, --prefix）
- 版本号（可选）

### 输出接口
- 安装的二进制文件
- 命令符号链接
- 配置文件

## 数据模型

### InstallConfig 结构体
```go
type InstallConfig struct {
    Mode      string // source, binary
    IsDevMode bool
    Prefix    string
    Version   string
}
```

## Jobs

---

### Job 1: build.sh 脚本实现

#### 目标

实现构建脚本，支持编译 Rick CLI 二进制文件

#### 前置条件

- cli_commands:job_7 - CLI 命令集成测试完成

#### Tasks

- [x] Task 1: 创建 scripts/build.sh 脚本
- [x] Task 2: 实现 Go 环境检查（版本 >= 1.21）
- [x] Task 3: 实现 `go build` 编译命令
- [x] Task 4: 实现 --output 参数支持
- [x] Task 5: 实现编译后的二进制验证
- [x] Task 6: 实现错误处理和清晰的错误提示
- [x] Task 7: 编写测试脚本，验证 build.sh 功能

#### 验证器

- build.sh 能正确编译二进制文件
- 编译后的二进制可以执行
- --output 参数生效
- Go 版本检查正确
- 错误提示清晰
- 测试脚本通过

#### 调试日志

无

#### 完成状态

✅ 已完成

---

### Job 2: install.sh 脚本实现

#### 目标

实现安装脚本，支持源码安装和二进制安装，支持生产版和开发版

#### 前置条件

- job_1 - build.sh 脚本实现完成

#### Tasks

- [x] Task 1: 创建 scripts/install.sh 脚本
- [x] Task 2: 实现参数解析（--source, --binary, --dev, --prefix）
- [x] Task 3: 实现源码安装流程（调用 build.sh）
- [x] Task 4: 实现二进制安装流程（从 GitHub releases 下载）
- [x] Task 5: 实现生产版本安装（~/.rick）
- [x] Task 6: 实现开发版本安装（~/.rick_dev）
- [x] Task 7: 实现符号链接创建（rick 或 rick_dev）
- [x] Task 8: 实现 PATH 环境变量更新提示
- [x] Task 9: 编写测试脚本，验证 install.sh 功能

#### 验证器

- ✅ install.sh 能正确安装生产版本
- ✅ install.sh 能正确安装开发版本
- ✅ 源码安装工作正确
- ✅ 二进制安装工作正确（GitHub releases 下载支持）
- ✅ 符号链接创建正确
- ✅ 命令 `rick --version` 和 `rick_dev --version` 可执行
- ✅ 测试脚本已创建并验证通过

#### 调试日志

无

#### 完成状态

✅ 已完成

---

### Job 3: uninstall.sh 脚本实现

#### 目标

实现卸载脚本，支持卸载生产版、开发版或所有版本

#### 前置条件

- job_2 - install.sh 脚本实现完成

#### Tasks

- [x] Task 1: 创建 scripts/uninstall.sh 脚本
- [x] Task 2: 实现参数解析（--dev, --all）
- [x] Task 3: 实现卸载生产版本（删除 ~/.rick）
- [x] Task 4: 实现卸载开发版本（删除 ~/.rick_dev）
- [x] Task 5: 实现删除符号链接
- [x] Task 6: 实现卸载确认提示
- [x] Task 7: 编写测试脚本，验证 uninstall.sh 功能

#### 验证器

- ✅ uninstall.sh 能正确卸载生产版本
- ✅ uninstall.sh 能正确卸载开发版本
- ✅ uninstall.sh 能正确卸载所有版本
- ✅ 符号链接被正确删除
- ✅ 测试脚本通过

#### 调试日志

无

#### 完成状态

✅ 已完成

---

### Job 4: update.sh 脚本实现

#### 目标

实现更新脚本，支持更新到最新版本或指定版本

#### 前置条件

- job_3 - uninstall.sh 脚本实现完成

#### Tasks

- [x] Task 1: 创建 scripts/update.sh 脚本
- [x] Task 2: 实现参数解析（--dev, --version）
- [x] Task 3: 实现更新流程（uninstall + install）
- [x] Task 4: 实现版本检查（获取最新版本）
- [x] Task 5: 实现更新确认提示
- [x] Task 6: 实现更新失败的回滚逻辑
- [x] Task 7: 编写测试脚本，验证 update.sh 功能

#### 验证器

- ✅ update.sh 能正确更新生产版本
- ✅ update.sh 能正确更新开发版本
- ✅ update.sh 能更新到指定版本
- ✅ 版本检查正确（支持 latest 和指定版本）
- ✅ 更新失败时能回滚（通过 backup_installation 和 restore_from_backup）
- ✅ 测试脚本通过（10/10 tests passed）

#### 调试日志

无

#### 完成状态

✅ 已完成

---

### Job 5: 版本管理脚本

#### 目标

实现版本管理脚本，支持版本号管理、发布流程

#### 前置条件

- job_4 - update.sh 脚本实现完成

#### Tasks

- [x] Task 1: 创建 scripts/version.sh 脚本
- [x] Task 2: 实现版本号读取（从 VERSION 文件或常量）
- [x] Task 3: 实现版本号更新
- [x] Task 4: 实现版本号格式验证（vMAJOR.MINOR.PATCH）
- [x] Task 5: 实现发布流程（git tag, changelog 生成）
- [x] Task 6: 编写测试脚本，验证版本管理

#### 验证器

- ✅ 版本号能正确读取
- ✅ 版本号能正确更新
- ✅ 版本号格式验证正确
- ✅ git tag 正确创建
- ✅ 测试脚本通过（12/12 tests passed）

#### 调试日志

- explore1: [探索发现] 版本号存储在 cmd/rick/main.go 中的 VERSION 常量，脚本需要解析和更新此常量，已确认
- debug1: macOS grep 不支持 -P 标志, 解决: 使用 sed 替代 Perl 正则, 已修复
- debug2: validate 命令返回值不正确, 原因: 脚本使用了 set -e 导致函数返回 1 时脚本退出, 解决: 移除 set -e 并在 main 函数中正确处理返回值, 已修复

#### 完成状态

✅ 已完成

---

### Job 6: 环境检查和配置

#### 目标

实现环境检查脚本，验证系统环境满足要求

#### 前置条件

- job_5 - 版本管理脚本完成

#### Tasks

- [ ] Task 1: 创建 scripts/check_env.sh 脚本
- [ ] Task 2: 实现 Go 版本检查（>= 1.21）
- [ ] Task 3: 实现 Claude Code CLI 检查
- [ ] Task 4: 实现 Git 检查
- [ ] Task 5: 实现 PATH 环境变量检查
- [ ] Task 6: 实现详细的检查报告
- [ ] Task 7: 编写测试脚本，验证环境检查

#### 验证器

- 环境检查能正确检测 Go 版本
- 环境检查能正确检测 Claude Code CLI
- 环境检查能正确检测 Git
- 检查报告清晰详细
- 测试脚本通过

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 7: 集成测试

#### 目标

验证 installation 模块所有脚本协同工作正确，能正确安装、卸载、更新 Rick CLI

#### 前置条件

- job_1 - build.sh 脚本实现完成
- job_2 - install.sh 脚本实现完成
- job_3 - uninstall.sh 脚本实现完成
- job_4 - update.sh 脚本实现完成
- job_5 - 版本管理脚本完成
- job_6 - 环境检查和配置完成

#### Tasks

- [ ] Task 1: 验证 build.sh 能正确编译
- [ ] Task 2: 验证 install.sh 能正确安装
- [ ] Task 3: 验证 uninstall.sh 能正确卸载
- [ ] Task 4: 验证 update.sh 能正确更新
- [ ] Task 5: 验证生产版本和开发版本能并行运行
- [ ] Task 6: 验证环境检查能正确工作
- [ ] Task 7: 编写集成测试脚本，覆盖完整安装流程

#### 验证器

- build.sh 编译成功
- install.sh 安装成功
- uninstall.sh 卸载成功
- update.sh 更新成功
- 生产版本和开发版本能并行运行
- 环境检查正确
- 集成测试脚本通过

#### 调试日志

无

#### 完成状态

⏳ 待开始

