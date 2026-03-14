# Tutorial 5: 集成到 CI/CD 流程

> 学习如何将 Rick CLI 集成到持续集成和持续部署流程中

## 📋 目标

在本教程中，你将学习：
- 如何在 CI/CD 环境中使用 Rick
- 如何自动化任务规划和执行
- 如何集成到 GitHub Actions/GitLab CI
- 如何实现自动化代码审查和测试

## 🎯 CI/CD 集成架构

```
┌──────────────────────────────────────────────────────────┐
│                   CI/CD Pipeline                          │
└──────────────────────────────────────────────────────────┘

1. 代码提交 (git push)
   └─> 触发 CI/CD

2. 环境准备
   ├─> 安装 Rick CLI
   ├─> 安装 Claude Code CLI
   └─> 配置环境变量

3. 自动化任务
   ├─> rick plan "自动化任务描述"
   ├─> rick doing job_n
   └─> rick learning job_n

4. 质量检查
   ├─> 运行测试
   ├─> 代码审查
   └─> 安全扫描

5. 部署
   └─> 自动部署到生产环境
```

---

## Step 1: GitHub Actions 集成

### 创建 GitHub Actions 工作流

```bash
# 创建工作流目录
mkdir -p .github/workflows

# 创建 Rick CI 工作流
cat > .github/workflows/rick-ci.yml << 'EOF'
name: Rick CLI CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

env:
  GO_VERSION: '1.21'
  RICK_VERSION: 'latest'

jobs:
  rick-automated-tasks:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: Install Rick CLI
        run: |
          # 从 GitHub Releases 安装
          curl -fsSL https://raw.githubusercontent.com/anthropics/rick/main/scripts/install.sh | bash
          echo "$HOME/.local/bin" >> $GITHUB_PATH

      - name: Install Claude Code CLI
        run: |
          # 安装 Claude Code CLI
          npm install -g @anthropic-ai/claude-code
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}

      - name: Run Rick automated tasks
        run: |
          # 规划任务
          rick plan "自动化代码审查和测试"

          # 执行任务
          rick doing job_0

          # 知识积累
          rick learning job_0
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}

      - name: Run tests
        run: |
          go test ./... -v -cover

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out

      - name: Archive Rick logs
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: rick-logs
          path: .rick/jobs/*/doing/logs/
EOF
```

### 配置 Secrets

```bash
# 在 GitHub 仓库设置中添加 Secrets
# Settings > Secrets and variables > Actions > New repository secret

# 添加以下 Secrets:
# - ANTHROPIC_API_KEY: Claude API 密钥
```

---

## Step 2: GitLab CI 集成

### 创建 GitLab CI 配置

```bash
cat > .gitlab-ci.yml << 'EOF'
image: golang:1.21

variables:
  RICK_VERSION: "latest"

stages:
  - setup
  - plan
  - execute
  - test
  - deploy

before_script:
  - apt-get update && apt-get install -y curl git
  - curl -fsSL https://raw.githubusercontent.com/anthropics/rick/main/scripts/install.sh | bash
  - export PATH="$HOME/.local/bin:$PATH"

setup:
  stage: setup
  script:
    - go version
    - rick --version
  only:
    - main
    - develop

rick_plan:
  stage: plan
  script:
    - rick plan "CI/CD 自动化任务"
  artifacts:
    paths:
      - .rick/jobs/*/plan/
    expire_in: 1 week
  only:
    - main
    - develop

rick_execute:
  stage: execute
  script:
    - rick doing job_0
  artifacts:
    paths:
      - .rick/jobs/*/doing/
    expire_in: 1 week
  dependencies:
    - rick_plan
  only:
    - main
    - develop

test:
  stage: test
  script:
    - go test ./... -v -cover -coverprofile=coverage.out
    - go tool cover -html=coverage.out -o coverage.html
  artifacts:
    paths:
      - coverage.out
      - coverage.html
    expire_in: 1 month
  dependencies:
    - rick_execute
  only:
    - main
    - develop

rick_learning:
  stage: test
  script:
    - rick learning job_0
  artifacts:
    paths:
      - .rick/jobs/*/learning/
    expire_in: 1 month
  dependencies:
    - rick_execute
  only:
    - main
    - develop

deploy:
  stage: deploy
  script:
    - echo "Deploying to production..."
    # 部署脚本
  only:
    - main
  when: manual
EOF
```

---

## Step 3: 自动化代码审查

### 创建代码审查工作流

```bash
cat > .github/workflows/rick-code-review.yml << 'EOF'
name: Rick Code Review

on:
  pull_request:
    types: [opened, synchronize]

jobs:
  automated-review:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3
        with:
          fetch-depth: 0  # 获取完整历史

      - name: Install Rick CLI
        run: |
          curl -fsSL https://raw.githubusercontent.com/anthropics/rick/main/scripts/install.sh | bash
          echo "$HOME/.local/bin" >> $GITHUB_PATH

      - name: Get changed files
        id: changed-files
        run: |
          echo "files=$(git diff --name-only ${{ github.event.pull_request.base.sha }} ${{ github.sha }} | tr '\n' ' ')" >> $GITHUB_OUTPUT

      - name: Rick automated code review
        run: |
          rick plan "审查以下文件的代码质量、安全性、性能: ${{ steps.changed-files.outputs.files }}"
          rick doing job_0
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}

      - name: Post review comments
        uses: actions/github-script@v6
        with:
          script: |
            const fs = require('fs');
            const reviewContent = fs.readFileSync('.rick/jobs/job_0/doing/review.md', 'utf8');

            github.rest.issues.createComment({
              owner: context.repo.owner,
              repo: context.repo.repo,
              issue_number: context.issue.number,
              body: `## 🤖 Rick 自动化代码审查\n\n${reviewContent}`
            });
EOF
```

---

## Step 4: 自动化测试生成

### 创建测试生成工作流

```bash
cat > .github/workflows/rick-test-gen.yml << 'EOF'
name: Rick Test Generation

on:
  push:
    branches: [ develop ]
    paths:
      - '**.go'
      - '!**_test.go'

jobs:
  generate-tests:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Install Rick CLI
        run: |
          curl -fsSL https://raw.githubusercontent.com/anthropics/rick/main/scripts/install.sh | bash
          echo "$HOME/.local/bin" >> $GITHUB_PATH

      - name: Find files without tests
        id: find-files
        run: |
          FILES=""
          for file in $(find . -name "*.go" ! -name "*_test.go"); do
            test_file="${file%.go}_test.go"
            if [ ! -f "$test_file" ]; then
              FILES="$FILES $file"
            fi
          done
          echo "files=$FILES" >> $GITHUB_OUTPUT

      - name: Generate tests with Rick
        if: steps.find-files.outputs.files != ''
        run: |
          rick plan "为以下文件生成单元测试: ${{ steps.find-files.outputs.files }}"
          rick doing job_0
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}

      - name: Create Pull Request
        if: steps.find-files.outputs.files != ''
        uses: peter-evans/create-pull-request@v5
        with:
          commit-message: "test: 自动生成单元测试"
          title: "🤖 Rick: 自动生成单元测试"
          body: |
            ## 自动生成的测试

            Rick CLI 自动为以下文件生成了单元测试：
            ${{ steps.find-files.outputs.files }}

            请审查测试代码并合并。
          branch: rick/auto-tests
EOF
```

---

## Step 5: 性能监控和优化

### 创建性能监控工作流

```bash
cat > .github/workflows/rick-performance.yml << 'EOF'
name: Rick Performance Monitoring

on:
  schedule:
    - cron: '0 0 * * 0'  # 每周日运行
  workflow_dispatch:

jobs:
  performance-analysis:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install Rick CLI
        run: |
          curl -fsSL https://raw.githubusercontent.com/anthropics/rick/main/scripts/install.sh | bash
          echo "$HOME/.local/bin" >> $GITHUB_PATH

      - name: Run benchmarks
        run: |
          go test -bench=. -benchmem ./... > benchmark.txt

      - name: Analyze performance with Rick
        run: |
          rick plan "分析性能基准测试结果，识别性能瓶颈，提出优化建议"
          rick doing job_0
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}

      - name: Create performance issue
        uses: actions/github-script@v6
        with:
          script: |
            const fs = require('fs');
            const analysis = fs.readFileSync('.rick/jobs/job_0/doing/analysis.md', 'utf8');

            github.rest.issues.create({
              owner: context.repo.owner,
              repo: context.repo.repo,
              title: '📊 性能分析报告 - ' + new Date().toISOString().split('T')[0],
              body: analysis,
              labels: ['performance', 'automated']
            });
EOF
```

---

## 🎓 学习要点

### 1. CI/CD 集成原则

- **自动化**: 尽可能自动化任务
- **快速反馈**: 快速发现问题
- **可重复性**: 确保流程可重复
- **安全性**: 保护敏感信息（使用 Secrets）

### 2. Rick CLI 在 CI/CD 中的优势

| 优势 | 说明 |
|------|------|
| 自动化任务分解 | 自动将复杂任务分解为小任务 |
| 智能代码生成 | 使用 AI 生成高质量代码 |
| 自动化测试 | 自动生成和运行测试 |
| 知识积累 | 持续学习和改进 |
| 失败重试 | 自动处理失败并重试 |

### 3. 最佳实践

- 使用 Secrets 管理敏感信息
- 缓存依赖以加速构建
- 并行运行独立任务
- 保存构建产物（artifacts）
- 监控 CI/CD 性能

---

## 💡 高级技巧

### 技巧 1: 条件执行

```yaml
# 只在特定条件下运行 Rick
- name: Run Rick
  if: contains(github.event.head_commit.message, '[rick]')
  run: |
    rick plan "${{ github.event.head_commit.message }}"
    rick doing job_0
```

### 技巧 2: 矩阵构建

```yaml
strategy:
  matrix:
    go-version: [1.21, 1.22]
    os: [ubuntu-latest, macos-latest]
runs-on: ${{ matrix.os }}
steps:
  - uses: actions/setup-go@v4
    with:
      go-version: ${{ matrix.go-version }}
```

### 技巧 3: 缓存优化

```yaml
- name: Cache Rick installation
  uses: actions/cache@v3
  with:
    path: ~/.rick
    key: ${{ runner.os }}-rick-${{ hashFiles('**/go.sum') }}
    restore-keys: |
      ${{ runner.os }}-rick-
```

---

## 🚀 下一步

恭喜完成 CI/CD 集成！接下来你可以：

1. **[最佳实践](../best-practices.md)** - 学习 CI/CD 集成的最佳实践
2. **[核心概念](../core-concepts.md)** - 深入理解 Rick 的核心概念
3. 实践：在你的项目中集成 Rick CLI

---

*最后更新: 2026-03-14*
