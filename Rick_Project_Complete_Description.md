# Rick 项目完整描述（优化版）

## 核心理论

**AICoding 最优实践公式**：`AICoding = Humans + Agents`

其中：
- **Agents = Models + Harness**
- **Harness** 由两个反馈循环组成：
  1. **Agent Loop**（局部最优）：`UI → Model → Tools`
     - 这是Claude Code等AI CLI已实现的能力，可直接复用
     - 随着工程师控制能力增长，后续可重构底层以解决上层特殊问题

  2. **Context Loop**（全局最优）：`Plan → Doing → Learning`
     - 这是Rick需要完成的核心部分
     - 通过组织**环境、验证、反馈**三个维度的信息，确保Agent大规模协同工作

- **Humans Loop**（人类决策层）：`Learning → Thinking → Express`
  - 基于SENSE方法，无需额外开发

**Agent协同的三维信息组织**：
在Context Loop中，人类的所有决策可归纳为三个维度：
- **环境**：项目全局目标(OKR)、开发规范(SPEC)、项目知识库(Wiki)
- **验证**：任务测试方法(task.py)、任务依赖关系(tasks.json)
- **反馈**：执行过程中的问题记录(debug.md)、执行日志(doing.log)

只要有效组织这三个维度的信息，就能确保Agents的协同工作质量。

---

## 工作空间结构

```
.rick/                          # Rick工作空间根目录（项目级持久化上下文）
│
├── OKR.md                      # 项目全局目标与关键结果
│                               # 作用：所有Agent必须加载，了解全局工作目标与约束条件
│
├── SPEC.md                     # 项目开发规范与行为准则
│                               # 作用：所有Agent在执行过程中必须遵循的标准
│
├── wiki/                       # 项目级知识库（系统运行原理与控制方法）
│   ├── index.md               # Wiki索引
│   ├── architecture.md        # 架构设计文档
│   └── ...
│
├── skills/                     # 项目级可复用技能库（Context Loop学习沉淀）
│   ├── skill_1/
│   │   ├── description.md     # 技能描述与使用场景
│   │   ├── implementation.md  # 实现细节
│   │   └── examples/
│   ├── skill_2/
│   └── skill_N/
│
└── jobs/                       # 任务执行计划集合（每个Plan生成一个Job）
    ├── job_1/
    │   ├── plan/              # 规划阶段产物
    │   │   ├── draft/         # 调研草稿（形式灵活）
    │   │   │   ├── draft_1.md
    │   │   │   └── ...
    │   │   │
    │   │   └── tasks/         # 最终确定的任务定义
    │   │       ├── task1.md   # 单个Agent Loop应执行的任务
    │   │       ├── task2.md
    │   │       └── taskN.md
    │   │
    │   ├── doing/             # 执行阶段产物
    │   │   ├── doing.log      # 多个Agent Loop的执行日志
    │   │   ├── debug.md       # 每次迭代遇到的问题记录（下次Loop加载）
    │   │   ├── tasks.json     # 任务DAG的拓扑排序结果
    │   │   └── tests/         # 任务验证脚本（taskN.py与taskN.md一一对应）
    │   │       ├── task1.py
    │   │       └── task2.py
    │   │
    │   └── learning/          # 学习阶段产物（最终合并到.rick全局目录）
    │       ├── OKR.md         # 本Job提取的OKR更新
    │       ├── SPEC.md        # 本Job提取的规范更新
    │       ├── wiki/          # 本Job的知识沉淀
    │       └── skills/        # 本Job沉淀的可复用技能
    │
    ├── job_2/
    │   ├── plan/
    │   ├── doing/
    │   └── learning/
    │
    └── job_N/
        ├── plan/
        ├── doing/
        └── learning/
```

**关键设计**：
- 分级上下文：全局信息(.rick/)由人类把控，Job级信息由Agent生成后人类审核合并
- 强制规范：task.md、tasks.json、debug.md 是驱动Context Loop的关键

---

## 核心文件格式规范

### 1. task.md 格式（由 `rick plan job_n` 自动生成）

```markdown
# 依赖关系
task1, task2

# 任务名称
创建 server.go 源文件

# 任务目标
基于gRPC框架，完成gRPC Server的搭建工作，创建server.go源文件。

# 关键结果
1. 检查是否已安装gRPC最新版本，若未安装则从GitHub下载安装
2. 检查Go版本是否满足最新gRPC要求，若不满足则升级
3. 查阅gRPC最新版本Server构建文档，学习官方最佳实践
4. 编写Golang代码：实现一个gRPC Hello World处理函数
5. 确保代码通过lint检查，遵循项目SPEC规范

# 测试方法
1. 学习gRPC服务端单元测试规范
2. 编写单元测试：启动Server，发起RPC请求，验证返回值为"hello world"
3. 运行单元测试：`go test -v ./...`，确保所有测试通过
4. 验证Server可正常启动且无内存泄漏
```

**task.md特点**：
- 由Agent生成而非人类编写
- 包含依赖关系（用于构建DAG）
- 关键结果必须可验证（对应task.py的测试方法）

### 2. tasks.json 格式（由 `rick doing job_n` 自动生成）

```json
[
    {
        "task_id": "task1",
        "task_name": "环境检查与依赖安装",
        "dep": [],
        "state_info": {
            "status": "pending"
        }
    },
    {
        "task_id": "task2",
        "task_name": "学习gRPC最佳实践",
        "dep": ["task1"],
        "state_info": {
            "status": "pending"
        }
    },
    {
        "task_id": "task4",
        "task_name": "创建 server.go 源文件",
        "dep": ["task1", "task2"],
        "state_info": {
            "status": "pending"
        }
    }
]
```

**自动生成流程**：
1. 扫描plan/tasks/目录下所有task.md文件
2. 解析文件名提取task_id
3. 通过Markdown解析器提取任务名称、依赖关系
4. 构建有向无环图(DAG)
5. 执行拓扑排序生成执行序列
6. 序列化为tasks.json

**任务状态定义**：
- `pending`：未执行
- `running`：执行中
- `done`：已完成

### 3. debug.md 格式（每次Agent Loop完成后追加）

```markdown
# debug1: 域名解析失败

**问题描述**
无法科学上网，无法解析github.com域名

**解决状态**
已解决

**解决方法**
- step1: 启动proxy.sh配置代理
- step2: ping github.com验证连通性
- step3: 更新系统DNS配置

---

# debug2: Server启动失败 - OOM

**问题描述**
直接运行server.go启动失败

**现象记录**
1. 首次启动报错：空指针异常
2. 对client赋值后，空指针消失，但Server无法启动，无报错信息
3. 通过dmesg查看系统日志：进程被OS直接kill，启动时OOM

**解决状态**
未解决（待继续调查）

**初步分析**
- 内存泄漏：Server初始化时申请过多内存
- 建议：使用pprof进行内存profile分析

**下次Loop建议**
- 添加内存限制参数启动Server
- 使用Go的race detector检测并发问题
```

**debug.md的作用**：
- 记录每次迭代遇到的问题与解决方案
- 在下次Agent Loop时加载，用于构建提示词避免重复踩坑
- 是Job执行过程中最有价值的工作日志
- Learning阶段从中提取知识更新OKR/SPEC/Skills

---

## 核心工作流程

### 阶段1：规划（Plan）

**用户命令**：`rick plan "开发一个plato的IM系统"`

**Rick执行流程**：
1. 创建.rick目录结构
2. 根据job序号自增创建jobN目录
3. 在jobN下创建plan/目录
4. 启动交互式AI Agent对话窗口
5. Agent通过调用工具进行深度调研：
   - 搜索相关开源项目
   - 查阅技术文档
   - 分析架构设计
   - 调研依赖库
6. 调研结果保存到plan/draft/目录
7. Agent调用 `/rick_plan` skill将draft转换为task.md格式
8. 生成plan/tasks/目录下的一组task*.md文件（含依赖关系）

**输出**：
- plan/draft/：调研报告（形式灵活，供人类审查）
- plan/tasks/task*.md：标准化任务定义（驱动Doing阶段）

### 阶段2：执行（Doing）

**用户命令**：`rick doing job_n`

**Rick执行流程**：

**初始化**：
1. 创建doing/目录
2. 如果tasks.json不存在，根据plan/tasks/中的task.md生成tasks.json（包含DAG拓扑排序）
3. 如果debug.md不存在，创建空debug.md
4. 如果doing.log不存在，创建空doing.log

**循环执行**（Schedule Loop）：
```
while 存在pending状态的task:
    1. 从tasks.json中取第一个pending状态的task
    2. 加载task.md、debug.md、全局OKR.md、SPEC.md构建Agent提示词

    3. [测试生成阶段]
       - 启动Agent，输入test_prompt（约束Agent只根据task.md的测试方法生成测试）
       - Agent生成doing/tests/taskN.py（与taskN.md一一对应）
       - taskN.py返回JSON格式的测试报告（包含pass/fail与错误信息）

    4. 将tasks.json中当前task状态改为running

    5. [执行循环]（Execution Loop）
       while taskN.py返回fail:
           - 构建完整的doing_prompt，包含：
             * task.md（任务定义）
             * debug.md（历史问题与解决方案）
             * OKR.md、SPEC.md（全局约束）
             * 上一轮的测试失败信息

           - 交给Agent后台执行（子进程，不受控）
           - 执行完毕，捕获异常信息（如有）

           - 执行taskN.py进行硬性验证
           - 如果失败，收集错误信息作为下一轮提示词的一部分

       end while

    6. [提交变更]
       - 执行git commit，固定本次变更便于CR
       - 将debug.md中新发现的问题追加记录

    7. 将tasks.json中当前task状态改为done

end while
```

**特点**：
- 测试先行：在执行前生成测试脚本，用测试结果判断任务完成
- 反馈循环：失败信息自动成为下一轮提示词的一部分
- 原子化提交：每个task完成后立即git commit，便于追踪
- 问题记录：每次迭代的问题都记录到debug.md，形成知识积累

**输出**：
- doing/doing.log：完整执行日志
- doing/debug.md：问题与解决方案的积累
- doing/tests/taskN.py：所有任务的测试脚本
- git历史：每个task对应一个commit

### 阶段3：学习（Learning）

**用户命令**：`rick learning job_n`

**Rick执行流程**：
1. 读取doing/debug.md（本Job的问题记录）
2. 读取git历史（本Job的所有变更）
3. 分析这两个信息源，理解项目发生了什么变化
4. 基于理解，Agent与人类进行对话，逐步确定：
   - **OKR更新**：这个Job对项目长期目标的影响
   - **SPEC更新**：是否需要调整开发规范
   - **Wiki更新**：是否有新的系统运行原理需要记录
   - **Skills沉淀**：是否有可复用的技能需要提取
5. 生成learning/目录下的OKR.md、SPEC.md、wiki/、skills/
6. 人类审核并调整这些内容
7. 最后调用merge skill，将learning/中的内容与.rick/全局目录合并：
   - OKR.md与.rick/OKR.md合并（去重、冲突解决）
   - SPEC.md与.rick/SPEC.md合并
   - wiki/内容与.rick/wiki/合并
   - skills/与.rick/skills/合并

**特点**：
- 分级管理：全局信息由人类把控，Job级信息由Agent生成后审核
- 知识沉淀：将执行过程中的经验教训转化为持久化的项目资产
- 质量把关：通过对话与审核确保学习内容的准确性

**输出**：
- .rick/OKR.md、SPEC.md、wiki/、skills/：更新的全局上下文

---

## 新项目 vs 旧项目工作流

### 新项目流程
```
rick plan "项目描述"
  ↓
rick doing job_1
  ↓
rick learning job_1
  ↓
rick plan "下一阶段需求"  （循环）
```

### 旧项目流程
```
rick init
  ├─ 创建.rick目录
  ├─ 自动生成源码探索task.md（跨越上下文窗口深度探索）
  ├─ 生成.rick/wiki/（深度调研报告，含index.md索引）
  ├─ 自动执行learning逻辑
  │  ├─ 读取debug.md + wiki
  │  └─ 生成.rick/OKR.md、SPEC.md、skills/
  └─ 人类审核并调整OKR.md和SPEC.md

rick doing job_1  （基于已有的OKR/SPEC继续开发）
```

**init命令的特殊性**：
- 自动化源码探索：使用特殊prompt生成task.md用于深度探索项目
- 跨越上下文限制：通过多个Agent Loop逐步探索，积累到wiki/
- 快速上手：一次init命令即可建立项目的完整知识库

---

## 核心设计优势

1. **强制规范驱动**：task.md、tasks.json、debug.md的严格格式确保了Context Loop的自动化
2. **分级上下文管理**：全局信息与Job级信息分离，人类把控质量
3. **知识积累**：每个Job都会沉淀OKR、SPEC、Skills、Wiki，形成项目资产
4. **问题反馈**：debug.md自动加载到下一轮提示词，避免重复踩坑
5. **原子化追踪**：每个task一个commit，便于问题回溯与CR
6. **可验证的完成**：task.py作为硬性验证，确保任务真正完成
