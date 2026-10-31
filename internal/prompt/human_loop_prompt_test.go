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

	// v7 skill 副本集：sense + research + 五个教学 agent（teach 不复存在）。
	skills := []string{
		"skill_sense.md",
		"skill_research.md",
		"skill_zpd_assessor.md",
		"skill_syllabus_builder.md",
		"skill_lesson_writer.md",
		"skill_originality_sparker.md",
		"skill_recall_review.md",
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

	// v6 的 teach/research task prompt 不再渲染。
	for _, gone := range []string{"teach.md", "research.md"} {
		if _, err := os.Stat(promptsDir + "/" + gone); !os.IsNotExist(err) {
			t.Errorf("v6 task prompt %s should no longer be rendered", gone)
		}
	}

	// 学习材料目录预建：research/ 与 lessons/。
	for _, d := range []string{loopDir + "/research", loopDir + "/lessons"} {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			t.Errorf("learning material dir not created: %s", d)
		}
	}

	// 主 prompt 引用 v7 学习材料路径约定（zpd/anchors/spark）。
	for _, want := range []string{
		"zpd.md", "anchors.md", "spark_{N}.md", "skill_sense.md",
		"zpd-assessor", "syllabus-builder", "lesson-writer", "originality-sparker", "recall-review",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("main prompt missing v7 path/agent reference %q", want)
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
	// 学习材料路径约定注入（v7：含 zpd/anchors + sense canon 路径）。
	for _, want := range []string{"mission.md", "values.md", "syllabus.md", "zpd.md", "anchors.md", "lessons", "research", "judgment.md", "skill_sense.md"} {
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
	// v7 核心循环关键词。
	for _, want := range []string{"费曼", "原创", "ZPD", "详略"} {
		if !strings.Contains(content, want) {
			t.Errorf("dry-run output missing v7 core-loop keyword %q", want)
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

	// v7 协议关键词：六 agent + 学习材料路径 + 文件状态机 + 核心循环 + 交付门禁。
	for _, kw := range []string{
		"research", "zpd-assessor", "syllabus-builder", "lesson-writer", "originality-sparker", "recall-review",
		"mission.md", "syllabus", "教学大纲", "lessons", "judgment.md", "zpd.md", "anchors.md",
		"文件状态机", "学习目标", "考核标准", "退出准则",
		"values_path", "价值裁决", "changelog", "大纲动态变更",
		"sense_skill_path",
		"问题还原", "构建视角", "洞察矛盾", "辩证逆转", "良质批判",
		"费曼", "原创思考", "详略", "核心循环", "ZPD 初始化", "认知强制", "节奏规则",
		"三次同类", "无能力", "二次探针",
	} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_loop.md missing v7 keyword %q", kw)
		}
	}

	// v6 遗留机制不得回流：teach agent、旧 SENSE 第一步名、exporter。
	for _, gone := range []string{"agent:'teach'", "teach_agent_path", "事实还原", "exporter", "阶段 1:S", "N1:", "反向回流", "三维打分", "系统论描述符", "max_retries", "max_backflows", "think_agent_path", "domain-survey", "module-facts"} {
		if strings.Contains(tpl.Content, gone) {
			t.Errorf("sense_loop.md should not contain removed mechanism %q", gone)
		}
	}

	// 派发显式语法 + 交付门禁。
	for _, kw := range []string{"workflowScript", "timeoutMs: 3600000", "runs.run", "交付门禁", "resume"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_loop.md missing dispatch keyword %q", kw)
		}
	}

	// 单写者按文件隔离：六 agent 各自的独写文件。
	for _, kw := range []string{"独写", "spark_{N}.md", "teach_{N}.md"} {
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

// TestV7SkillsCoreContracts 验证五个 v7 教学 skill 的核心契约关键词
//（判级 rubric / 误差吸收 / 独写文件 / 交付契约）。
func TestV7SkillsCoreContracts(t *testing.T) {
	cases := []struct {
		skill    string
		required []string
		forbid   []string
	}{
		{
			skill: "zpd_assessor",
			required: []string{
				"zpd-assessor", "probe", "assess", "强", "中", "弱",
				"无错误断言", "错误断言", "二次探针", "部分线索", "换措辞",
				"两轮不过", "无能力", "仅本模式", "zpd.md", "独写", "学习知识记录",
				"费曼", "关键概念清单", "期望", "现状", "rubric", "禁止百分比",
			},
			forbid: nil,
		},
		{
			skill: "syllabus_builder",
			required: []string{
				"syllabus-builder", "syllabus", "syllabus-update", "changelog",
				"问题还原", "构建视角", "洞察矛盾", "辩证逆转", "良质批判",
				"教学序列", "多个备选", "多个可行方案", "强相关", "弱相关",
				"独写", "期望", "现状", "判据", "一课一事", "不替 human 收敛",
			},
			forbid: nil,
		},
		{
			skill: "lesson_writer",
			required: []string{
				"lesson-writer", "lesson", "teach_{N}", "zpd.md", "档位",
				"ZPD 详略", "少而精", "加速", "先判图/文", "code-fence",
				"box-drawing", "事实层", "推导层", "留白", "一课一事",
				"引注", "关键概念清单", "费曼", "独写", "正文全文",
			},
			forbid: nil,
		},
		{
			skill: "originality_sparker",
			required: []string{
				"originality-sparker", "spark", "spark_{N}", "费曼",
				"真实异质性", "部分违背", "非完全否定", "更优替代", "差异维度",
				"跨领域", "web", "信源", "魔鬼代言人式角色扮演", "合成异议", "独写",
			},
			forbid: nil,
		},
		{
			skill: "recall_review",
			required: []string{
				"recall-review", "recall", "anchors.md", "独写",
				"关键词", "卡点", "未解矛盾", "≤5", "复习关键词",
				"允许遗忘", "再创造", "mission",
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
