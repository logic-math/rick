package prompt

import (
	"os"
	"strings"
	"testing"
)

// forbiddenTeachMethodology 列出 human 明确要求删除的旧方法论/机制关键词。
// SENSE 已重构为思考范式 canon（五步：事实还原/构建视角/洞察矛盾/辩证逆转/良质批判），
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

func TestGenerateHumanLoopPromptFileWritesThreeFiles(t *testing.T) {
	tmpDir := t.TempDir()
	draftDir := tmpDir + "/draft"
	rfcDir := draftDir + "/rfc"
	loopDir := draftDir + "/loops/loop_1"
	promptsDir := loopDir + "/prompts"

	pm := NewPromptManager()

	mainFile, teachFile, researchFile, err := GenerateHumanLoopPromptFile("test topic", rfcDir, draftDir, loopDir, pm)
	if err != nil {
		t.Fatalf("GenerateHumanLoopPromptFile() error: %v", err)
	}

	for _, f := range []string{mainFile, teachFile, researchFile} {
		if !strings.HasPrefix(f, promptsDir) {
			t.Errorf("file %s is not under promptsDir %s", f, promptsDir)
		}
	}

	for _, f := range []string{mainFile, teachFile, researchFile} {
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("failed to read %s: %v", f, err)
		}
		s := string(content)
		if strings.Contains(s, "{{") {
			t.Errorf("file %s still contains an unreplaced template variable", f)
		}
		if !strings.Contains(s, draftDir) {
			t.Errorf("file %s does not contain draftDir %q", f, draftDir)
		}
	}

	// Skills: sense + teach + research（不再有 skill_think/skill_exporter）。
	for _, skill := range []string{"skill_sense.md", "skill_teach.md", "skill_research.md"} {
		if _, err := os.Stat(promptsDir + "/" + skill); os.IsNotExist(err) {
			t.Errorf("skill file not written to prompts dir: %s", skill)
		}
	}

	// 学习材料目录预建：research/ 与 lessons/。
	for _, d := range []string{loopDir + "/research", loopDir + "/lessons"} {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			t.Errorf("learning material dir not created: %s", d)
		}
	}

	// v6：teach/research prompt 引用 SENSE canon（sense_skill_path）与价值清单（values_path）。
	for _, f := range []string{teachFile, researchFile} {
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("failed to read %s: %v", f, err)
		}
		s := string(content)
		if !strings.Contains(s, promptsDir+"/skill_sense.md") {
			t.Errorf("file %s does not reference the sense canon (skill_sense.md)", f)
		}
		if !strings.Contains(s, loopDir+"/values.md") {
			t.Errorf("file %s does not reference values_path", f)
		}
	}
}

func TestGenerateHumanLoopPromptDryRunNoUnreplacedVars(t *testing.T) {
	draftDir := "/tmp/test-draft"
	rfcDir := draftDir + "/rfc"
	pm := NewPromptManager()

	content, err := GenerateHumanLoopPrompt("test topic", rfcDir, draftDir, pm)
	if err != nil {
		t.Fatalf("GenerateHumanLoopPrompt() error: %v", err)
	}

	if strings.Contains(content, "{{") {
		t.Errorf("dry-run output contains an unreplaced template variable")
	}
	// 学习材料路径约定注入（v6：含 values.md 价值清单 + sense canon 路径）。
	for _, want := range []string{"mission.md", "values.md", "syllabus.md", "lessons", "research", "judgment.md", "skill_sense.md"} {
		if !strings.Contains(content, want) {
			t.Errorf("dry-run output missing learning-material path %q", want)
		}
	}
	// SENSE 五步名（渲染冒烟：事实还原…良质批判）。
	for _, want := range []string{"事实还原", "构建视角", "洞察矛盾", "辩证逆转", "良质批判"} {
		if !strings.Contains(content, want) {
			t.Errorf("dry-run output missing SENSE step name %q", want)
		}
	}
	if strings.Contains(content, "exporter") {
		t.Error("dry-run output still contains exporter")
	}
}

func TestSenseLoopTemplateIsTeacherSession(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_loop")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_loop) error: %v", err)
	}

	// v6 协议关键词：teach + research + 学习材料路径约定 + 文件状态机 + 四类回合 + SENSE canon。
	for _, kw := range []string{"teach", "research", "mission.md", "syllabus", "教学大纲", "lessons", "judgment.md", "文件状态机", "学习目标", "考核标准", "退出准则",
		"values_path", "价值裁决", "changelog", "大纲动态变更", "assess", "考核演练", "sense_skill_path",
		"事实还原", "构建视角", "洞察矛盾", "辩证逆转", "良质批判"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_loop.md missing keyword %q", kw)
		}
	}

	// 已删除的机械化机制不得回流。
	for _, gone := range []string{"exporter", "阶段 1:S", "N1:", "反向回流", "三维打分", "系统论描述符", "max_retries", "max_backflows", "think_agent_path", "domain-survey", "module-facts"} {
		if strings.Contains(tpl.Content, gone) {
			t.Errorf("sense_loop.md should not contain removed mechanism %q", gone)
		}
	}

	// 派发显式语法 + 交付门禁。
	for _, kw := range []string{"workflowScript", "timeoutMs: 3600000", "runs.run"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_loop.md missing dispatch keyword %q", kw)
		}
	}
}

func TestSenseLoopJudgmentIsHumanOnly(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_loop")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_loop) error: %v", err)
	}
	if !strings.Contains(tpl.Content, "human 原话") && !strings.Contains(tpl.Content, "human 判断") {
		t.Error("sense_loop.md missing human-only constraint for judgment.md")
	}
	if !strings.Contains(tpl.Content, "AI 推理") {
		t.Error("sense_loop.md missing prohibition of AI reasoning in judgment.md")
	}
}

func TestTeachTemplateHasSenseParadigmAndTaskModes(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("teach")
	if err != nil {
		t.Fatalf("LoadTemplate(teach) error: %v", err)
	}
	skill, err := ReadEmbeddedSkill("teach")
	if err != nil {
		t.Fatalf("ReadEmbeddedSkill(teach) error: %v", err)
	}
	all := tpl.Content + "\n" + skill

	// 四种任务模式（facts 由 research 承担）。
	for _, kw := range []string{"syllabus", "lesson", "syllabus-update", "assess"} {
		if !strings.Contains(all, kw) {
			t.Errorf("teach template/skill missing task mode %q", kw)
		}
	}

	// SENSE 五步（范式新名）。
	for _, kw := range []string{"事实还原", "构建视角", "洞察矛盾", "辩证逆转", "良质批判"} {
		if !strings.Contains(all, kw) {
			t.Errorf("teach template/skill missing SENSE step %q", kw)
		}
	}

	// lesson = 写作要求（非模板）：还原作者 SENSE 过程七问 + 良质自检 + 追问回到学生。
	for _, kw := range []string{"原始问题", "事实调研", "价值判断", "知识领域", "主要矛盾", "教学良质自检", "学生自身的知识体系", "无法理解", "凭空假设", "追问回到学生自己", "写作要求", "形式自由"} {
		if !strings.Contains(all, kw) {
			t.Errorf("teach template/skill missing lesson writing requirement %q", kw)
		}
	}

	// syllabus：一次完整 SENSE 思考留痕 + 教学序列 + 矛盾与逆转多备选。
	for _, kw := range []string{"教学序列", "多个备选", "多个可行方案", "强相关", "弱相关"} {
		if !strings.Contains(all, kw) {
			t.Errorf("teach template/skill missing syllabus keyword %q", kw)
		}
	}

	// assess：human 当老师、teach 当学生+教练。
	for _, kw := range []string{"教练", "学生", "老师", "缺证据", "价值未裁决", "矛盾未定", "逆转是否成立"} {
		if !strings.Contains(all, kw) {
			t.Errorf("teach template/skill missing assess keyword %q", kw)
		}
	}

	// v5 已有交付契约保留：正文入会话 + 落盘 + 引注 + primary source。
	for _, kw := range []string{"正文全文", "lessons/", "teach_", "primary source", "引注"} {
		if !strings.Contains(all, kw) {
			t.Errorf("teach template/skill missing delivery keyword %q", kw)
		}
	}

	// 已删方法论不得回流；v5 缺陷行（禁止正文入会话）必须清除。
	for _, gone := range append(append([]string{}, forbiddenTeachMethodology...), "把课程正文全文塞进最终回复") {
		if strings.Contains(all, gone) {
			t.Errorf("teach template/skill should not contain removed content %q", gone)
		}
	}
}

func TestResearchTemplateHasRecursiveFactsAndValues(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("research")
	if err != nil {
		t.Fatalf("LoadTemplate(research) error: %v", err)
	}
	for _, kw := range []string{"递归", "常识性的置信信息源", "facts", "价值性信息", "values_path", "尽调树", "MECE", "下钻", "信源", "等级", "置信度", "subagent", "research_", "Gaps", "回溯不到常识层", "收益递减"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("research.md missing keyword %q", kw)
		}
	}
	for _, obsolete := range []string{"BFS", "全量收集", "QUEUE", "exporter", "domain-survey", "module-facts"} {
		if strings.Contains(tpl.Content, obsolete) {
			t.Errorf("research.md should not contain obsolete concept %q", obsolete)
		}
	}
	skill, err := ReadEmbeddedSkill("research")
	if err != nil {
		t.Fatalf("ReadEmbeddedSkill(research) error: %v", err)
	}
	for _, kw := range []string{"facts", "递归", "常识", "价值性信息", "values.md", "待裁决", "尽调树", "MECE", "R7 上报", "Gaps"} {
		if !strings.Contains(skill, kw) {
			t.Errorf("research skill missing keyword %q", kw)
		}
	}
	for _, obsolete := range []string{"domain-survey", "module-facts"} {
		if strings.Contains(skill, obsolete) {
			t.Errorf("research skill should not contain obsolete task mode %q", obsolete)
		}
	}
}

// TestSenseSkillIsParadigmCanon 验证 sense canon：五步权威文本逐字语义 + 范式声明 +
// 三个可操作定义；旧方法论（系统论描述符/三维打分/阶段/回流）全清。
func TestSenseSkillIsParadigmCanon(t *testing.T) {
	sense, err := ReadEmbeddedSkill("sense")
	if err != nil {
		t.Fatalf("ReadEmbeddedSkill(sense) error: %v", err)
	}

	// 五步权威文本关键语义（递归至常识性置信信息源 / 价值性信息上报 / 强弱相关 /
	// 收敛发散 / 全局根本决定 / 阻碍变手段 / 假设置信与必然性）。
	for _, kw := range []string{
		"第一步 事实还原", "第二步 构建视角", "第三步 洞察矛盾", "第四步 辩证逆转", "第五步 良质批判",
		"递归的调研所有涉及到的事实信息", "常识性的置信信息源", "识别所有的价值性信息", "上报给 human",
		"第一性原理", "强相关", "弱相关", "收敛性思维", "发散性思维",
		"全局性、根本性、决定性", "多个备选", "X 是 Y 必然成立的前提", "将阻碍变成达成目标的手段",
		"多个可行的解决方案", "是否足够置信", "必然得出目标可达成的结论", "被忽略的关键假设",
	} {
		if !strings.Contains(sense, kw) {
			t.Errorf("sense canon missing authoritative-text keyword %q", kw)
		}
	}

	// 范式声明 + 三个可操作定义。
	for _, kw := range []string{"思考范式", "不是流程清单", "形式自由", "递归终止三层", "回溯不到常识层", "收益递减", "批量呈报", "裁决完成前不进入第 2 步"} {
		if !strings.Contains(sense, kw) {
			t.Errorf("sense canon missing paradigm definition %q", kw)
		}
	}

	// 旧方法论全清。
	for _, gone := range []string{"系统论描述符", "三维打分", "反向回流", "node", "input", "output", "inner", "edge", "N1", "N2", "S-R", "良知批判", "跃迁", "问题确认"} {
		if strings.Contains(sense, gone) {
			t.Errorf("sense canon should not contain removed methodology %q", gone)
		}
	}
}
