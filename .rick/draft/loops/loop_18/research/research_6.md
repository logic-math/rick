# research_6 — 「自举的最小信任锚」点火候选 ×3 — 2026-11-15

> 时间基准 2026-11-15。信源等级：A = 一手原文（RFC 全文 / 官方博客原文 / 作者主页题录与论文 PDF，全部经 fetch 核实；查不到可靠出处的已删）。映射目标：他在 rick super（固定 task 四元组驱动自改进）上追问的「rsi 系统的可验证性在哪」「判定器怎么验证自己的改进而不漂移」。禁重复清单已核对（Willard 自验证公理系统、PCC、信号检测论、LOSA 等均已给过，不重讲，仅做分界）。据本地抓取缓存判断，另有并行调研在处理 seL4/CompCert/Thompson 方向，本文三概念不与之重叠（此为推断，供 parent 排重参考）。

## 概念 1｜Milawa 自验证定理证明器 × Jitawa 验证栈（形式验证 / 证明器工程）

**①名称与领域**：Milawa——ACL2 谱系、可运行时自我升级的定理证明器；Jitawa——验证到 x86 机器码的 Lisp 运行时。领域：交互式定理证明、程序验证。

**②真实来源**：Jared C. Davis, *A Self-Verifying Theorem Prover*, 博士论文, University of Texas at Austin, 2009-12；Magnus O. Myreen & Jared Davis, "A verified runtime for a verified theorem prover", ITP 2011, Springer LNCS；Myreen & Davis, "The reflective Milawa theorem prover is sound (down to the machine code that runs it)"（Myreen 剑桥主页 jitawa 页公开手稿，标注 Submitted to ITP 2014）。三处题录与机制细节均在该手稿 PDF 及 jitawa 页核实。（A）

**③是什么**：Milawa 把声音性关键代码收缩到约 2000 行 Lisp 的小内核（LCF 式），初始 proof checker 只接受逻辑的原始推理。核心机制是 reflection 的 switch 事件：运行时可用用户提供的更强 checker 替换现任 checker，但替换被接受的前提是——先证明「新 checker 能证明的命题，初始 checker 也都能证明」（保守性证明，由现任 checker 检查）。反复执行替换（bootstrapping sequence），最终把条件重写、case splitting、泛化等 ACL2 式大步推理安装为单一推理步——prover 在运行中长大，每次升级却被锁死在初始锚上。Myreen–Davis 再把整条栈向下验证：HOL4 证明 Milawa 逻辑可靠 + 2000 行内核忠实于逻辑 + 内核跑在 Jitawa 上（动态编译、验证到 x86 机器码的 Lisp 运行时）；顶层 HOL4 定理：Milawa 在 Jitawa 上只会证明语义为真的命题，一路落到具体 x86 机器码。作者自评这是迄今对证明器声音性最完整的机器形式证据；过程中「在 Milawa 实现里发现了错误，但没有发现声音性 bug」。

**④与映射目标的结构关系**：这是「判定器怎么验证自己的改进而不漂移」的工程字面答案——判定器可运行时升级，但 gate 不是自评而是继任者验证：现任 checker 证明新 checker 严格保守于**初始**锚（锚不随代际移动），漂移被构造性排除；task 四元组对应那个不变的初始锚，一切升级只能做保守扩展。「验证者被验证」的递归不在系统内部闭合（Milawa 从不证明自己可靠），而是换向落地：向上是时间链（每代验证下一代），向下落到外部更强元逻辑 HOL4（LCF 家族，定理只能由小内核的推理规则产生）+ x86 模型 + 2000 行可审计内核——递归终点不是又一层证明，而是几个小到可审计的信任件。分界：与已给 Willard 相反（Willard 在系统内部削弱算术以自证一致性、绕哥德尔 II；Milawa 从不自证）；与已给 PCC 相异（PCC 是不可信代码携带证明交消费端检查；Milawa 升级的是验证器本身，gate 是对验证器的保守性证明）。

## 概念 2｜Guix 全源自举：357 字节 hex0 种子（系统构建 / bootstrappable builds 运动）

**①名称与领域**：Full-Source Bootstrap（全源自举）/ bootstrappable builds 运动。领域：软件供应链安全、编译器工具链自举。

**②真实来源**：Janneke Nieuwenhuizen & Ludovic Courtès, "The Full-Source Bootstrap: Building from source all the way down", Guix 官方博客, 2023-04-26（全文已读）；stage0-posix 由 Jeremiah Orians 创建，bootstrap-seeds 位于其 GitHub（oriansj/，官方博客引用）；GNU Mes、Bootstrappable Builds 项目（bootstrappable.org）。官方 2019/2020 两篇种子缩减里程碑博文被 2023 文引用。（A）

**③是什么**：2023-04 合入 Guix master 后，x86 上超过 22,000 个节点的包图根在一个 357 字节的 hex0 二进制种子。hex0 的「源码」就是十六进制数字对（如 "58" 即字节 0x58，加两种行注释），种子是手写机器码，hex0 自举（能重编译自身）。stage0-posix 从它逐级自举：hex1→catm→hex2→M0→cc_x86→M1→M2→get_machine（=MesCC-Tools）→M2-Planet→GNU Mes v0.24（约 5000 行简单 C 的 Scheme 解释器 + 用 Scheme 写的 C 编译器 MesCC）→tcc-boot→…→gcc-core 2.95.3/binutils 2.20.1a/glibc 2.2.5→…→gcc 4.9.4。动机直指 Thompson 1984 trusting-trust 攻击，官方博客原话：该攻击「只是自举故事不完整的症状」——每级由上一级构建，恶意逻辑必须活在可读源码里而非不可审计的二进制 blob 里。边界诚实：仍剩一个约 25 MiB 的静态 Guile 驱动无法从源头构建——最小信任基在逼近零，从未到零。

**④与映射目标的结构关系**：「要多小的种子」的生产答案：种子大小由「人类能否逐字节手审」定标，不由功能强弱定标——种子的职责不是去验证整个系统，而是小到可信，让上层全部成为推导物。自举链是派生链而非证明链：没有任何一级的正确性被证明；被消灭的是不可审计的二进制信任，换来的是全源码可重建的出处链——映射到 rick super：task 四元组扮演 hex0（固定小锚），「不漂移」不靠每级证明，而靠任何漂移都必然在可审计层留下源码痕迹。附带工程教训：老版本工具（gcc 2.95.3 一档）是自举的必经台阶——自改进系统要保留历史阶梯当 rung，不能只留最新一代。

## 概念 3｜Certificate Transparency 证书透明度日志（Web PKI / 分布式可审计性）

**①名称与领域**：Certificate Transparency（CT，证书透明度）。领域：Web PKI、密码学可审计基础设施。

**②真实来源**：RFC 6962 "Certificate Transparency"（Experimental），Ben Laurie、Adam Langley、Emilia Kasper（Google），2013-06（RFC 全文已读）；RFC 9162（CT v2.0，2021，取代 RFC 6962，摘要与 §1 已读）；Chrome CT 政策与时间线：chromium/ct-policy 仓库 ct_policy.md / changes.md（已 clone 核读）。（A）

**③是什么**：CA 签发的每张 TLS 证书须提交到由独立运营方运行的公开、只追加、不信任的日志，日志返回 SCT（签名时间戳），并承诺在 Maximum Merge Delay（MMD）内并入 Merkle 树。浏览器拒绝无足够 SCT 的证书；Chrome 政策要求 SCT 来自至少 2 个不同运营方的日志，2018-04-30 之后签发的证书无有效披露即整页拦截（net::ERR_CERTIFICATE_TRANSPARENCY_REQUIRED；2015-01 起先对 EV 证书施行）。验证角色被拆开：monitor 定期拉取新条目盯误签发，auditor 拿 SCT 索要 O(log n) 的 Merkle 包含证明，consistency proof 用 O(log n) 证明「新树头是旧树头的超集」（只追加性）。日志给不同人看不同视图（分叉）时，比对树根+一致性证明即可高效发现——误签发与日志作恶都变成公开可复验的证据，而不是需要更高裁判的争端。诚实边界（RFC 9162 §1.6）：观察者之间不互相 gossip 交叉比对时，单个日志仍须被当作可信第三方——分叉检测依赖观察者网络，协议本身不解决最后一步。

**④与映射目标的结构关系**：当判定器无法被最小化、也找不到向上的更强元验证器时的第三条路：让每次裁定成为公开、只追加、O(log n) 可复验的记录——「验证者被验证」不向上找更大的验证者，而是横向摊给任何愿意跑 monitor/auditor 的独立第三方。漂移的失败模式从「静默改判」变成「分叉视图」：单点漂移对内不可见，但两个独立观察者手里的树根必然分叉，分歧本身即证据；MMD 把验证挪出关键路径但给出硬截止——与推理服务里异步校验加超时回退的工程形态同构。「≥2 个不同运营方」是制度化的多见证冗余，防单验证器漂移/合谋。

## 排序（最可能点燃）

**概念 1（Milawa×Jitawa）**：把他的两个问题压进同一个工程物——判定器可运行时升级，但每次升级须过现任判定的保守性证明（锚不随代际移动，漂移被构造性排除），整栈再验证到 x86；机制与已给过的 Willard/PCC 正交，恰好是他「哥德尔编译器类比」的可执行对照物。次之概念 2（357 字节的数字化答案 + 他的编译器主场）；再次概念 3（生产系统同构性最强，但离 rsi 核心最远）。

## Gaps
- Myreen–Davis soundness 手稿仅核实到 Myreen 主页标注「Submitted to ITP 2014」，正式发表版未核实到（按手稿引用）。
- Davis 2009 博士论文未逐字读（题录经该手稿参考文献核实）；Milawa bootstrapping sequence 的具体步数未载。
- Guix 自举链各级未逐级复算（以官方博客一手叙述为准）；ARM/RISC-V 路线未展开。
- CT 的 gossip/观察者互检协议在 RFC 6962/9162 范围外（仅有 RFC 9162 §1.6 的边界声明），draft 谱系未展开。
