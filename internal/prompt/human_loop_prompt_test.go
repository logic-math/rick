package prompt

import (
	"os"
	"strings"
	"testing"
)

// forbiddenTeachMethodology 列出 human 明确要求删除的旧方法论/机制关键词。
// SENSE 已重构为思考范式 canon（五步：问题还原/构建视角/洞察矛盾/辩证逆转/良质批判），
// 旧阶段机器/打分/描述符不得回流。
var forbiddenTeachMethodology = []string{
	"三维打分",
	"系统论描述符",
	"4 维打分",
	"think_top_n",
	"反向回流",
	"max_backflows",
	"node/input/output/inner/edge",
}

func TestGenerateHumanLoopPromptFileWritesV7FileSet(t *testing.T) {
	tmpDir := t.TempDir()
	draftDir := tmpDir + "/draft"
	rfcDir := draftDir + "/rfc"
	loopDir := draftDir + "/loops/loop_1"
	promptsDir := loopDir + "/prompts"

	pm := NewPromptManager()

	mainFile, err := GenerateHumanLoopPromptFile("test topic", rfcDir, draftDir, loopDir, pm)
	if err != nil {
		t.Fatalf("GenerateHumanLoopPromptFile() error = %v", err)
	}

	if !strings.HasPrefix(mainFile, promptsDir) {
		t.Errorf("mainFile %s is not under promptsDir %s", mainFile, promptsDir)
	}

	content, err := os.ReadFile(mainFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", mainFile, err)
	}
	s := string(content)
	if strings.Contains(s, "{{") {
		t.Errorf("main prompt still contains an unreplaced template variable")
	}
	if !strings.Contains(s, draftDir) {
		t.Errorf("main prompt does not contain draftDir %q", draftDir)
	}

	// v7.2 skill 副本集（8 个）：sense + 三个 parent 内联方法论（zpd/feynman/exploration_stream）
	// + 四个重生型 agent 契约（syllabus_builder/lesson_writer/recall_review/research）。
	skills := []string{
		"skill_sense.md",
		"skill_zpd.md",
		"skill_feynman.md",
		"skill_exploration_stream.md",
		"skill_syllabus_builder.md",
		"skill_lesson_writer.md",
		"skill_recall_review.md",
		"skill_research.md",
	}
	for _, skill := range skills {
		path := promptsDir + "/" + skill
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("skill file not written to prompts dir: %s (%v)", skill, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("skill file %s is empty", skill)
		}
	}

	// v7.1 的 zpd_assessor/originality_sparker 副本不再渲染（agent 已移除，方法论改为内联 skill_zpd/exploration_stream）。
	for _, gone := range []string{"skill_zpd_assessor.md", "skill_originality_sparker.md"} {
		if _, err := os.Stat(promptsDir + "/" + gone); !os.IsNotExist(err) {
			t.Errorf("removed skill %s should no longer be rendered", gone)
		}
	}

	// 学习材料目录预建：research/ 与 lessons/。
	for _, d := range []string{loopDir + "/research", loopDir + "/lessons"} {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			t.Errorf("learning material dir not created: %s", d)
		}
	}

	// 主 prompt 引用 v7.2 路径约定：三个内联方法论 skill 路径 + 四个 agent 名。
	for _, want := range []string{
		"zpd.md", "anchors.md", "teach_{N}.md", "skill_sense.md", "skill_zpd.md", "skill_feynman.md", "skill_exploration_stream.md",
		"research", "syllabus-builder", "lesson-writer", "recall-review",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("main prompt missing v7.2 path/agent reference %q", want)
		}
	}
}

func TestGenerateHumanLoopPromptDryRunNoUnreplacedVars(t *testing.T) {
	draftDir := "/tmp/test-draft"
	rfcDir := draftDir + "/rfc"
	pm := NewPromptManager()

	content, err := GenerateHumanLoopPrompt("test topic", rfcDir, draftDir, pm)
	if err != nil {
		t.Fatalf("GenerateHumanLoopPrompt() error = %v", err)
	}

	if strings.Contains(content, "{{") {
		t.Errorf("dry-run output contains an unreplaced template variable")
	}
	// 学习材料路径约定注入（v7.2：含 zpd/anchors + 四个方法论 skill 路径）。
	for _, want := range []string{"mission.md", "values.md", "syllabus.md", "zpd.md", "anchors.md", "lessons", "research", "judgment.md",
		"skill_sense.md", "skill_zpd.md", "skill_feynman.md", "skill_exploration_stream.md"} {
		if !strings.Contains(content, want) {
			t.Errorf("dry-run output missing learning-material path %q", want)
		}
	}
	// SENSE 五步名（v7：第一步=问题还原）。
	for _, want := range []string{"问题还原", "构建视角", "洞察矛盾", "辩证逆转", "良质批判"} {
		if !strings.Contains(content, want) {
			t.Errorf("dry-run output missing SENSE step name %q", want)
		}
	}
	// v7.2 核心循环关键词 + v7.4（原创捕获 + ⑤ 统一探索流）。
	for _, want := range []string{
		"费曼", "原创", "ZPD", "详略",
		"结构化提问", "ZPD 生成", "ZPD 档案", "费曼演绎",
		"探索流", "原创性信息点", "原创性表达", "推进下一课", "相似异质性概念", "详细描述",
		"会什么", "薄弱点", "parent 独写",
		"原创捕获", "原创观点", "统一探索流", "总结与统一探索流", "原创信息点清单",
		"思考还原链", "明确鼓励",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("dry-run output missing v7.2 core-loop keyword %q", want)
		}
	}
	// v7.3 输出纪律（沉默的编排者）：text 只出教学内容，编排动作全部隐藏。
	for _, want := range []string{
		"输出纪律", "沉默的编排者", "只允许七类", "禁止翻译", "Translation:",
		"验收元数据", "run id", "内部失败内部修", "动作不打字",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("dry-run output missing v7.3 output-discipline keyword %q", want)
		}
	}
	if strings.Contains(content, "exporter") {
		t.Error("dry-run output still contains exporter")
	}
}

func TestSenseLoopTemplateIsV7TeachingSession(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_loop")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_loop) error = %v", err)
	}

	// v7.2 协议关键词：四 agent + 三个内联方法论 skill + 学习材料路径 + 文件状态机 + 核心循环 + 交付门禁。
	for _, kw := range []string{
		"research", "syllabus-builder", "lesson-writer", "recall-review",
		"mission.md", "syllabus", "教学大纲", "lessons", "judgment.md", "zpd.md", "anchors.md",
		"文件状态机", "学习目标", "考核标准", "退出准则",
		"values_path", "价值裁决", "changelog", "大纲动态变更",
		"sense_skill_path", "zpd_skill_path", "feynman_skill_path", "exploration_skill_path",
		"问题还原", "构建视角", "洞察矛盾", "辩证逆转", "良质批判",
		"费曼", "原创思考", "详略", "核心循环", "ZPD 生成", "认知强制",
		"三次同类",
		// v7.2：结构化提问 + parent 内联（费曼演绎 3 步/探索流/档案）。
		"结构化提问", "ZPD 档案", "费曼演绎", "探索流", "原创性信息点", "原创性表达",
		"相似异质性概念", "推进下一课", "会什么", "薄弱点", "parent 独写", "不派 subagent",
		// v7.4：④ 原创捕获 + ⑤ 总结与统一探索流 + 思考还原链。
		"原创捕获", "原创观点", "总结与统一探索流", "统一探索流", "原创信息点清单",
		"结束询问", "明确鼓励", "不谄媚", "思考还原链", "human 决定何时结束",
	} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_loop.md missing v7.2 keyword %q", kw)
		}
	}

	// v6/v7.1 遗留机制不得回流：teach agent、旧 SENSE 第一步名、旧 ZPD 机制、
	// v7.1 的三个交互型 agent（v7.2 已移除，改 parent 内联）。
	for _, gone := range []string{
		"agent:'teach'", "teach_agent_path", "事实还原", "exporter", "阶段 1:S", "N1:",
		"反向回流", "三维打分", "系统论描述符", "max_retries", "max_backflows",
		"think_agent_path", "domain-survey", "module-facts",
		"ZPD 初始化", "任务模式 `probe`", "任务模式 `assess`",
		"agent:'zpd-assessor'", "agent:'feynman'", "agent:'originality-sparker'",
		"feynman_{N}", "spark_{N}",
		// v7.4：④ 每课探索流（移除）——触发句式与探索流作为第 5 步不得回流。
		"要进入探索流吗", "**5. 探索流",
	} {
		if strings.Contains(tpl.Content, gone) {
			t.Errorf("sense_loop.md should not contain removed mechanism %q", gone)
		}
	}

	// 派发显式语法 + 交付门禁（只覆盖四个 agent）。
	for _, kw := range []string{"workflowScript", "timeoutMs: 3600000", "runs.run", "交付门禁", "resume", "只覆盖四个"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_loop.md missing dispatch keyword %q", kw)
		}
	}

	// 单写者按文件隔离：四 agent + parent 独写 mission/judgment/zpd。
	for _, kw := range []string{"独写", "teach_{N}.md", "zpd.md", "judgment.md"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_loop.md missing single-writer keyword %q", kw)
		}
	}
}

func TestSenseLoopJudgmentIsHumanOnly(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_loop")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_loop) error = %v", err)
	}
	if !strings.Contains(tpl.Content, "human 原话") && !strings.Contains(tpl.Content, "human 判断") {
		t.Error("sense_loop.md missing human-only constraint for judgment.md")
	}
	if !strings.Contains(tpl.Content, "AI 推理") {
		t.Error("sense_loop.md missing prohibition of AI reasoning in judgment.md")
	}
}

// TestV7SkillsCoreContracts 验证 v7.2 教学 skill 的核心契约关键词
//（费曼演绎 3 步、ZPD 档案格式、探索流循环协议、四 agent 契约、独写文件）。
func TestV7SkillsCoreContracts(t *testing.T) {
	cases := []struct {
		skill    string
		required []string
		forbid   []string
	}{
		{
			skill: "feynman",
			required: []string{
				"费曼", "3 步协议", "parent 会话内联执行", "内联方法论",
				"引导开始", "用自己的话讲出来", "现在你当老师",
				"清晰化引导", "事实模糊", "价值模糊", "逐处追问", "对话内连续追问",
				"补充答案", "重新教学", "只补讲该缺口",
				"记录 ZPD", "模糊点", "薄弱知识领域", "skill_zpd",
				"不替 human 讲解", "需补事实", "不编造", "≤3 处模糊点",
				"输出纪律", "动作不打字", "不在 text 播报",
			},
			forbid: []string{"challenge", "verdict", "逼问清单", "独写 lessons"},
		},
		{
			skill: "zpd",
			required: []string{
				"ZPD 档案", "parent 会话内联执行", "内联方法论", "parent 独写",
				"概念掌握表", "状态（会/边缘/不会）", "证据（模糊点与对错摘要）", "时间",
				"当前结论", "可加速", "重点教", "补基础",
				"薄弱点", "薄弱知识领域标记",
				"详略建议", "费曼演绎步③", "建档", "lesson-writer",
				"动作不打字", "不在 text 播报", "text 不播报写入",
			},
			forbid: []string{"select", "consolidate", "zpd-assessor"},
		},
		{
			skill: "exploration_stream",
			required: []string{
				"探索流", "⑤ 收口阶段统一进行", "parent 会话内联执行",
				"SENSE 状态机", "状态判定", "问题还原", "构建视角", "洞察矛盾", "辩证逆转", "良质批判",
				"全部课程学习完", "human 同意收口", "结束权在 human", "不催促",
				"原创信息点", "judgment.md", "一次完整 SENSE 思考",
				"research", "相似异质性概念", "详细描述", "机制级", "不替他想",
				"达标判据", "推进", "原创性想法", "后续内容",
				"价值清晰、符合事实", "识别为假设",
				"输出纪律", "动作不打字", "不在 text 播报",
			},
			forbid: []string{"originality-sparker", "spark_{N}", "混沌学园", "三动作", "关键词级点火", "话轮平等", "不戴帽子", "零建议零反馈", "只做呼应", "Y 轴"},
		},
		{
			skill: "syllabus_builder",
			required: []string{
				"syllabus-builder", "syllabus", "syllabus-update", "changelog",
				"问题还原", "构建视角", "洞察矛盾", "辩证逆转", "良质批判",
				"教学序列", "多个备选", "多个可行方案", "强相关", "弱相关",
				"独写", "期望", "现状", "判据", "一课一事", "不替 human 收敛",
				"纯正文", "验收元数据",
			},
			forbid: nil,
		},
		{
			skill: "lesson_writer",
			required: []string{
				"lesson-writer", "lesson", "teach_{N}", "zpd.md", "档位",
				"还原作者的 SENSE 思考过程", "思考还原链", "期望 vs 现状", "结构化呈现",
				"事实块", "证据表", "禁止散文化堆砌", "流水账散文",
				"会", "点名跳过", "边缘", "重点展开", "不会", "完整铺开", "更低的起点焊入",
				"ZPD 详略", "先判图/文", "code-fence",
				"box-drawing", "推导层留白", "留白", "一课一事",
				"引注", "关键概念清单", "费曼", "独写", "正文全文",
				"纯正文", "不含交付自检", "验收元数据",
			},
			forbid: nil,
		},
		{
			skill: "recall_review",
			required: []string{
				"recall-review", "recall", "anchors.md", "独写",
				"关键词", "卡点", "未解矛盾", "≤5", "复习关键词",
				"允许遗忘", "再创造", "mission",
				"纯清单", "验收元数据",
			},
			forbid: nil,
		},
	}
	for _, c := range cases {
		skill, err := ReadEmbeddedSkill(c.skill)
		if err != nil {
			t.Fatalf("ReadEmbeddedSkill(%s) error = %v", c.skill, err)
		}
		for _, kw := range c.required {
			if !strings.Contains(skill, kw) {
				t.Errorf("skill %s missing contract keyword %q", c.skill, kw)
			}
		}
		for _, gone := range c.forbid {
			if strings.Contains(skill, gone) {
				t.Errorf("skill %s should not contain %q", c.skill, gone)
			}
		}
	}
}

// TestRemovedV72SkillsAreGone 验证 v7.2 删除的 skill 文件不复存在
//（zpd_assessor/originality_sparker 已被 skill_zpd/skill_exploration_stream 替代）。
func TestRemovedV72SkillsAreGone(t *testing.T) {
	for _, gone := range []string{"zpd_assessor", "originality_sparker"} {
		if _, err := ReadEmbeddedSkill(gone); err == nil {
			t.Errorf("skill %q should be removed in v7.2 (replaced by inline methodology)", gone)
		}
	}
}

func TestResearchSkillKeepsFactsContract(t *testing.T) {
	skill, err := ReadEmbeddedSkill("research")
	if err != nil {
		t.Fatalf("ReadEmbeddedSkill(research) error = %v", err)
	}
	for _, kw := range []string{"facts", "递归", "常识", "价值性信息", "values.md", "待裁决", "尽调树", "MECE", "R7 上报", "Gaps", "问题还原"} {
		if !strings.Contains(skill, kw) {
			t.Errorf("research skill missing keyword %q", kw)
		}
	}
	// v7 消费者已换：teach 协作段不再出现。
	if strings.Contains(skill, "- **teach**：消费你落盘的事实材料") {
		t.Error("research skill still references teach as consumer (v7 consumers are syllabus-builder/lesson-writer)")
	}
	for _, obsolete := range []string{"domain-survey", "module-facts"} {
		if strings.Contains(skill, obsolete) {
			t.Errorf("research skill should not contain obsolete task mode %q", obsolete)
		}
	}
}

// TestSenseSkillIsParadigmCanon 验证 sense canon：五步权威文本（v7 第一步=问题还原）+
// 范式声明 + 四个可操作定义；旧方法论（系统论描述符/三维打分/阶段/回流）全清。
func TestSenseSkillIsParadigmCanon(t *testing.T) {
	sense, err := ReadEmbeddedSkill("sense")
	if err != nil {
		t.Fatalf("ReadEmbeddedSkill(sense) error = %v", err)
	}

	// 五步权威文本关键语义（问题还原：事实+价值+期望/现状分离）。
	for _, kw := range []string{
		"第一步 问题还原", "第二步 构建视角", "第三步 洞察矛盾", "第四步 辩证逆转", "第五步 良质批判",
		"还原问题的全部事实信息与价值判断", "分清楚期望与现状",
		"递归地调研所有涉及到的事实信息", "常识性的置信信息源", "识别所有的价值性信息", "上报给 human",
		"第一性原理", "强相关", "弱相关", "收敛性思维", "发散性思维",
		"全局性、根本性、决定性", "多个备选", "X 是 Y 必然成立的前提", "将阻碍变成达成目标的手段",
		"多个可行的解决方案", "是否足够置信", "必然得出目标可达成的结论", "被忽略的关键假设",
	} {
		if !strings.Contains(sense, kw) {
			t.Errorf("sense canon missing authoritative-text keyword %q", kw)
		}
	}

	// 范式声明 + 四个可操作定义（新增期望/现状分离）。
	for _, kw := range []string{"思考范式", "不是流程清单", "形式自由", "递归终止三层", "回溯不到常识层", "收益递减", "批量呈报", "裁决完成前不进入第 2 步", "期望与现状的分离", "最近发展区"} {
		if !strings.Contains(sense, kw) {
			t.Errorf("sense canon missing paradigm definition %q", kw)
		}
	}

	// 旧方法论全清（事实还原作为第一步名不得回流）。
	for _, gone := range []string{"系统论描述符", "三维打分", "反向回流", "node", "input", "output", "inner", "edge", "N1", "N2", "S-R", "良知批判", "跃迁", "问题确认", "第一步 事实还原"} {
		if strings.Contains(sense, gone) {
			t.Errorf("sense canon should not contain removed methodology %q", gone)
		}
	}
}
