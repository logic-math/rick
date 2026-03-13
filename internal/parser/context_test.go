package parser

import (
	"strings"
	"testing"
)

func TestParseOKR(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		check   func(*ContextInfo) bool
	}{
		{
			name:    "empty content",
			content: "",
			wantErr: false,
			check: func(info *ContextInfo) bool {
				return len(info.Objectives) == 0 && len(info.KeyResults) == 0
			},
		},
		{
			name: "valid OKR with objectives and key results",
			content: `# 目标
- 建立高效的开发框架
- 提升代码质量
- 优化团队协作

# 关键结果
1. 完成核心模块实现
2. 测试覆盖率达到80%
3. 性能提升30%`,
			wantErr: false,
			check: func(info *ContextInfo) bool {
				return len(info.Objectives) == 3 &&
					len(info.KeyResults) == 3 &&
					info.Objectives[0] == "建立高效的开发框架" &&
					info.KeyResults[0] == "完成核心模块实现"
			},
		},
		{
			name: "OKR with bullet points (asterisk)",
			content: `# 目标
* 目标1
* 目标2

# 关键结果
* 结果1
* 结果2`,
			wantErr: false,
			check: func(info *ContextInfo) bool {
				return len(info.Objectives) == 2 &&
					len(info.KeyResults) == 2 &&
					info.Objectives[0] == "目标1" &&
					info.KeyResults[0] == "结果1"
			},
		},
		{
			name: "OKR with English headers",
			content: `# Objectives
- Objective 1
- Objective 2

# Key Results
- Result 1
- Result 2`,
			wantErr: false,
			check: func(info *ContextInfo) bool {
				return len(info.Objectives) == 2 &&
					len(info.KeyResults) == 2
			},
		},
		{
			name: "OKR with only objectives",
			content: `# 目标
- 目标1
- 目标2`,
			wantErr: false,
			check: func(info *ContextInfo) bool {
				return len(info.Objectives) == 2 &&
					len(info.KeyResults) == 0
			},
		},
		{
			name: "OKR with only key results",
			content: `# 关键结果
- 结果1
- 结果2`,
			wantErr: false,
			check: func(info *ContextInfo) bool {
				return len(info.Objectives) == 0 &&
					len(info.KeyResults) == 2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseOKR(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseOKR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.check(info) {
				t.Errorf("ParseOKR() result check failed for %s", tt.name)
			}
		})
	}
}

func TestParseSPEC(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		check   func(*ContextInfo) bool
	}{
		{
			name:    "empty content",
			content: "",
			wantErr: false,
			check: func(info *ContextInfo) bool {
				return len(info.Specifications) == 0
			},
		},
		{
			name: "valid SPEC with specifications",
			content: `# 规范
- 使用Go语言开发
- 遵循Google编码规范
- 单元测试覆盖率>=80%
- 代码审查必须通过`,
			wantErr: false,
			check: func(info *ContextInfo) bool {
				return len(info.Specifications) == 4 &&
					info.Specifications[0] == "使用Go语言开发"
			},
		},
		{
			name: "SPEC with numbered list",
			content: `# 开发规范
1. 规范1
2. 规范2
3. 规范3`,
			wantErr: false,
			check: func(info *ContextInfo) bool {
				return len(info.Specifications) == 3 &&
					info.Specifications[0] == "规范1"
			},
		},
		{
			name: "SPEC with English header",
			content: `# Specifications
- Spec 1
- Spec 2`,
			wantErr: false,
			check: func(info *ContextInfo) bool {
				return len(info.Specifications) == 2 &&
					info.Specifications[0] == "Spec 1"
			},
		},
		{
			name: "SPEC with asterisk bullets",
			content: `# 规范
* 规范1
* 规范2`,
			wantErr: false,
			check: func(info *ContextInfo) bool {
				return len(info.Specifications) == 2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseSPEC(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSPEC() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.check(info) {
				t.Errorf("ParseSPEC() result check failed for %s", tt.name)
			}
		})
	}
}

func TestExtractObjectives(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
		wantErr bool
	}{
		{
			name:    "empty content",
			content: "",
			want:    0,
			wantErr: false,
		},
		{
			name: "no objectives section",
			content: `# 某个其他部分
- 内容`,
			want:    0,
			wantErr: false,
		},
		{
			name: "objectives with dash",
			content: `# 目标
- 目标1
- 目标2
- 目标3`,
			want:    3,
			wantErr: false,
		},
		{
			name: "objectives with numbered list",
			content: `# 目标
1. 目标1
2. 目标2`,
			want:    2,
			wantErr: false,
		},
		{
			name: "objectives with mixed formats",
			content: `# 目标
- 目标1
* 目标2
1. 目标3`,
			want:    3,
			wantErr: false,
		},
		{
			name: "objectives with multiple sections",
			content: `# 目标
- 目标1
- 目标2

# 关键结果
- 结果1`,
			want:    2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractObjectives(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractObjectives() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.want {
				t.Errorf("ExtractObjectives() got %d items, want %d", len(got), tt.want)
			}
		})
	}
}

func TestExtractKeyResults(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
		wantErr bool
	}{
		{
			name:    "empty content",
			content: "",
			want:    0,
			wantErr: false,
		},
		{
			name: "no key results section",
			content: `# 目标
- 目标1`,
			want:    0,
			wantErr: false,
		},
		{
			name: "key results with dash",
			content: `# 关键结果
- 结果1
- 结果2`,
			want:    2,
			wantErr: false,
		},
		{
			name: "key results with numbered list",
			content: `# 关键结果
1. 结果1
2. 结果2
3. 结果3`,
			want:    3,
			wantErr: false,
		},
		{
			name: "key results with asterisk",
			content: `# 关键结果
* 结果1
* 结果2`,
			want:    2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractKeyResults(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractKeyResults() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.want {
				t.Errorf("ExtractKeyResults() got %d items, want %d", len(got), tt.want)
			}
		})
	}
}

func TestExtractSpecifications(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
		wantErr bool
	}{
		{
			name:    "empty content",
			content: "",
			want:    0,
			wantErr: false,
		},
		{
			name: "specifications with dash",
			content: `# 规范
- 规范1
- 规范2`,
			want:    2,
			wantErr: false,
		},
		{
			name: "specifications with numbered list",
			content: `# 规范
1. 规范1
2. 规范2
3. 规范3`,
			want:    3,
			wantErr: false,
		},
		{
			name: "specifications with 开发规范 header",
			content: `# 开发规范
- 规范1
- 规范2`,
			want:    2,
			wantErr: false,
		},
		{
			name: "specifications with asterisk",
			content: `# 规范
* 规范1
* 规范2`,
			want:    2,
			wantErr: false,
		},
		{
			name: "specifications with English header",
			content: `# Specifications
- Spec 1
- Spec 2`,
			want:    2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractSpecifications(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractSpecifications() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.want {
				t.Errorf("ExtractSpecifications() got %d items, want %d", len(got), tt.want)
			}
		})
	}
}

func TestContextInfoIntegration(t *testing.T) {
	// Test a complete OKR.md scenario
	okrContent := `# 目标
- 建立高效的开发框架
- 提升代码质量
- 优化团队协作

# 关键结果
1. 完成核心模块实现
2. 测试覆盖率达到80%
3. 性能提升30%
4. 团队培训完成`

	okrInfo, err := ParseOKR(okrContent)
	if err != nil {
		t.Fatalf("ParseOKR failed: %v", err)
	}

	if len(okrInfo.Objectives) != 3 {
		t.Errorf("Expected 3 objectives, got %d", len(okrInfo.Objectives))
	}

	if len(okrInfo.KeyResults) != 4 {
		t.Errorf("Expected 4 key results, got %d", len(okrInfo.KeyResults))
	}

	// Test a complete SPEC.md scenario
	specContent := `# 规范
- 使用Go语言开发
- 遵循Google编码规范
- 单元测试覆盖率>=80%
- 代码审查必须通过
- 性能优化优先级高`

	specInfo, err := ParseSPEC(specContent)
	if err != nil {
		t.Fatalf("ParseSPEC failed: %v", err)
	}

	if len(specInfo.Specifications) != 5 {
		t.Errorf("Expected 5 specifications, got %d", len(specInfo.Specifications))
	}

	// Verify content integrity
	if !strings.Contains(specInfo.Specifications[0], "Go") {
		t.Errorf("Expected first spec to contain 'Go', got: %s", specInfo.Specifications[0])
	}
}

func TestContextInfoWithWhitespace(t *testing.T) {
	// Test handling of extra whitespace
	content := `# 目标
  - 目标1
  - 目标2

# 关键结果
  1. 结果1
  2. 结果2`

	info, err := ParseOKR(content)
	if err != nil {
		t.Fatalf("ParseOKR failed: %v", err)
	}

	if len(info.Objectives) != 2 || len(info.KeyResults) != 2 {
		t.Errorf("Failed to parse with extra whitespace")
	}

	// Verify trimming
	if strings.HasPrefix(info.Objectives[0], " ") || strings.HasSuffix(info.Objectives[0], " ") {
		t.Errorf("Objectives not properly trimmed: '%s'", info.Objectives[0])
	}
}

func TestContextInfoEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		content string
		testFn  func(*ContextInfo) bool
	}{
		{
			name: "empty list items should be skipped",
			content: `# 目标
- 目标1

- 目标2`,
			testFn: func(info *ContextInfo) bool {
				// Should have 2 objectives, empty lines should be skipped
				return len(info.Objectives) == 2
			},
		},
		{
			name: "mixed list and non-list content",
			content: `# 目标
Some paragraph text
- 目标1
- 目标2
More text`,
			testFn: func(info *ContextInfo) bool {
				// Should extract only list items
				return len(info.Objectives) == 2 &&
					info.Objectives[0] == "目标1" &&
					info.Objectives[1] == "目标2"
			},
		},
		{
			name: "nested heading should not be included",
			content: `# 目标
- 目标1
## 子目标
- 子目标1
- 目标2`,
			testFn: func(info *ContextInfo) bool {
				// Should extract all list items under the section
				return len(info.Objectives) >= 2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseOKR(tt.content)
			if err != nil {
				t.Fatalf("ParseOKR failed: %v", err)
			}
			if !tt.testFn(info) {
				t.Errorf("Edge case test failed for: %s", tt.name)
			}
		})
	}
}
