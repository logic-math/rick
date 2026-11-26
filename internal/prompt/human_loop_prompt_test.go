package prompt

import (
	"os"
	"strings"
	"testing"
)

// forbiddenV7Teaching 列出 v8 已删除的 v7 教学机制关键词（human 裁决：
// 全部教学机制/状态机/历史兼容删除，完全重新设计）。
var forbiddenV7Teaching = []string{
	"ZPD", "费曼", "探索流", "教学大纲", "课程", "价值裁决", "学习使命",
	"mission.md", "values.md", "syllabus.md", "judgment.md", "anchors.md",
	"lesson-writer", "recall-review", "syllabus-builder",
	"文件状态机", "每课教学回合", "认知强制", "详略", "可重访锚",
}

func TestGenerateHumanLoopPromptFileWritesV8FileSet(t *testing.T) {
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

	// v8.1 skill 副本集（8 个）：SENSE canon + 五个阶段 skill + orig agent + 教学润色。
	skills := []string{
		"skill_sense.md",
		"skill_sense_reconstruct.md",
		"skill_sense_perspective.md",
		"skill_sense_contradiction.md",
		"skill_sense_reversal.md",
		"skill_sense_critique.md",
		"skill_orig.md",
		"skill_teacher_polish.md",
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

	// v7 教学 skills 不再渲染。
	for _, gone := range []string{
		"skill_zpd.md", "skill_feynman.md", "skill_exploration_stream.md",
		"skill_syllabus_builder.md", "skill_lesson_writer.md", "skill_recall_review.md",
		"skill_research.md",
	} {
		if _, err := os.Stat(promptsDir + "/" + gone); !os.IsNotExist(err) {
			t.Errorf("removed v7 skill %s should no longer be rendered", gone)
		}
	}

	// v8 产物结构：orig/ 中间结构目录预建 + thought_path.md 初始化。
	if _, err := os.Stat(loopDir + "/orig"); os.IsNotExist(err) {
		t.Errorf("orig intermediate dir not created: %s/orig", loopDir)
	}
	tp, err := os.ReadFile(loopDir + "/thought_path.md")
	if err != nil {
		t.Fatalf("thought_path.md not initialized: %v", err)
	}
	if len(tp) == 0 {
		t.Error("thought_path.md should be initialized with a header line")
	}

	// v7 学习材料目录不再预建。
	for _, gone := range []string{loopDir + "/research", loopDir + "/lessons"} {
		if _, err := os.Stat(gone); !os.IsNotExist(err) {
			t.Errorf("v7 learning material dir should no longer be created: %s", gone)
		}
	}

	// 主 prompt 引用 v8.1 路径约定：thought_path + orig stage 落盘 + 五阶段 skill + orig skill + 教学润色 skill。
	for _, want := range []string{
		"thought_path.md", "orig/stage", "skill_sense.md", "skill_sense_reconstruct.md",
		"skill_sense_perspective.md", "skill_sense_contradiction.md", "skill_sense_reversal.md",
		"skill_sense_critique.md", "skill_orig.md", "skill_teacher_polish.md",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("main prompt missing v8 path reference %q", want)
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
	// v8 产物路径约定注入。
	for _, want := range []string{"thought_path.md", "orig"} {
		if !strings.Contains(content, want) {
			t.Errorf("dry-run output missing v8 artifact path %q", want)
		}
	}
	// v8.1 呈现层关键词：教学润色（科普视角呈现）。
	for _, want := range []string{"教学润色"} {
		if !strings.Contains(content, want) {
			t.Errorf("dry-run output missing v8.1 polish keyword %q", want)
		}
	}
	// v8 五阶段名（human 裁决表述）。
	for _, want := range []string{"问题还原", "视角生产", "判断矛盾", "辩证逆转", "良质批判"} {
		if !strings.Contains(content, want) {
			t.Errorf("dry-run output missing SENSE stage name %q", want)
		}
	}
	// v8 核心机制关键词：思考门禁 + orig 启发 + 推进权。
	for _, want := range []string{"门禁", "orig", "启发", "推进", "thought_path"} {
		if !strings.Contains(content, want) {
			t.Errorf("dry-run output missing v8 core keyword %q", want)
		}
	}
	// v7 教学机制不得回流。
	for _, gone := range forbiddenV7Teaching {
		if strings.Contains(content, gone) {
			t.Errorf("dry-run output should not contain v7 teaching keyword %q", gone)
		}
	}
}

func TestSenseLoopTemplateIsV8GateLoop(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_loop")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_loop) error = %v", err)
	}

	// v8 协议关键词：五阶段 + 思考门禁 + orig 启发 + thought_path 唯一产出 + 推进权。
	for _, kw := range []string{
		"问题还原", "视角生产", "判断矛盾", "辩证逆转", "良质批判",
		"思考门禁", "门禁", "orig", "启发",
		"thought_path.md", "唯一产出",
		"推进权", "human",
		"{{sense_reconstruct_skill_path}}", "{{sense_perspective_skill_path}}", "{{sense_contradiction_skill_path}}",
		"{{sense_reversal_skill_path}}", "{{sense_critique_skill_path}}", "{{orig_skill_path}}", "{{sense_skill_path}}",
		"{{teacher_polish_skill_path}}", "教学润色",
		"stage", "workflowScript", "runs.run",
	} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_loop.md missing v8 keyword %q", kw)
		}
	}

	// v7 教学机制全清（human 裁决：全部删除）。
	for _, gone := range forbiddenV7Teaching {
		if strings.Contains(tpl.Content, gone) {
			t.Errorf("sense_loop.md should not contain v7 teaching mechanism %q", gone)
		}
	}
	// 旧四 agent 名不得作为派发对象回流（v8 唯一 agent = orig）。
	for _, gone := range []string{"'research'", "'syllabus-builder'", "'lesson-writer'", "'recall-review'"} {
		if strings.Contains(tpl.Content, gone) {
			t.Errorf("sense_loop.md should not dispatch removed agent %q", gone)
		}
	}
}

func TestSenseLoopThoughtPathIsHumanTrajectory(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_loop")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_loop) error = %v", err)
	}
	// thought_path 写入契约：阶段标记 + human 原话 + AI 启发引用，工具层写入不播报。
	for _, kw := range []string{"human 原话", "阶段标记", "工具层"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_loop.md missing thought_path contract keyword %q", kw)
		}
	}
}

// TestV8SkillsCoreContracts 验证 v8 五阶段 skill + orig skill 的核心契约关键词
//（每阶段：思考目标 + orig 派发契约（唯一提示词）+ 启发呈现纪律 + 门禁标准 + 未过门禁行为；
// orig：唯一提示词 + 调研方法 + 启发返回 + stage 落盘）。
func TestV8SkillsCoreContracts(t *testing.T) {
	// 唯一提示词（human 裁决的 orig agent 核心原则）。
	const origPrompt = "给用户返回怎样的内容才能引导他完成本阶段的思考目标"
	cases := []struct {
		skill    string
		required []string
		forbid   []string
	}{
		{
			skill: "sense_reconstruct",
			required: []string{
				"问题还原", origPrompt, "门禁", "未过门禁", "推进权",
				"期望", "现状", "差距", "事实", "价值", "stage",
			},
			forbid: forbiddenV7Teaching,
		},
		{
			skill: "sense_perspective",
			required: []string{
				"视角生产", origPrompt, "门禁", "未过门禁", "推进权",
				"强相关", "弱相关", "收敛", "发散", "应用价值", "因果运动", "stage",
			},
			forbid: forbiddenV7Teaching,
		},
		{
			skill: "sense_contradiction",
			required: []string{
				"判断矛盾", origPrompt, "门禁", "未过门禁", "推进权",
				"全局性", "根本性", "决定性", "多个备选", "因果运动", "stage",
			},
			forbid: forbiddenV7Teaching,
		},
		{
			skill: "sense_reversal",
			required: []string{
				"辩证逆转", origPrompt, "门禁", "未过门禁", "推进权",
				"X 是 Y 必然成立的前提", "多个可行的解决方案", "必要条件", "stage",
			},
			forbid: forbiddenV7Teaching,
		},
		{
			skill: "sense_critique",
			required: []string{
				"良质批判", origPrompt, "门禁", "未过门禁", "推进权",
				"假设", "可证伪", "逻辑性", "stage",
			},
			forbid: forbiddenV7Teaching,
		},
		{
			skill: "orig",
			required: []string{
				origPrompt, "web_search", "OpenAlex", "stage",
				"启发", "不是答案", "缺口", "概念跳跃", "差异性", "同一性",
			},
			forbid: nil,
		},
		{
			skill: "teacher_polish",
			required: []string{
				"科普视角", "短句", "一句一义", "主动语态", "主语明确", "少嵌套",
				"准确", "通俗", "有逻辑", "有趣味",
				"已知结论", "争议", "知识边界",
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

// TestRemovedV8SkillsAreGone 验证 v8 删除的 v7 教学 skill 文件不复存在
//（human 裁决：全部教学机制删除；sense.md 保留作 SENSE canon）。
func TestRemovedV8SkillsAreGone(t *testing.T) {
	for _, gone := range []string{
		"zpd", "feynman", "exploration_stream", "syllabus_builder",
		"lesson_writer", "recall_review", "research",
	} {
		if _, err := ReadEmbeddedSkill(gone); err == nil {
			t.Errorf("v7 teaching skill %q should be removed in v8", gone)
		}
	}
}

// TestSenseSkillIsParadigmCanon 验证 sense canon：五步权威文本（第一步=问题还原）+
// 范式声明 + 可操作定义（sense.md 保留——dream 共用 + 五阶段 skill 提炼来源）。
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

	// v8 canon 注释：human-loop v8 单 orig agent。
	for _, kw := range []string{"v8", "orig"} {
		if !strings.Contains(sense, kw) {
			t.Errorf("sense canon header should describe v8 human-loop agent %q", kw)
		}
	}

	// 旧方法论全清。
	for _, gone := range []string{"系统论描述符", "三维打分", "反向回流", "良知批判", "第一步 事实还原"} {
		if strings.Contains(sense, gone) {
			t.Errorf("sense canon should not contain removed methodology %q", gone)
		}
	}
}
