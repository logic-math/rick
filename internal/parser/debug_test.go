package parser

import (
	"strings"
	"testing"
)

func TestParseDebug(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "empty content",
			content:   "",
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "single debug entry",
			content: `**调试日志**:
- debug1: JSON 序列化失败, 复杂对象循环引用, 猜想: 1)缺少循环引用处理 2)未使用 JSON.stringify 的 replacer, 验证: 添加 replacer 函数测试, 修复: 使用 WeakSet 检测循环引用, 待修复`,
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "multiple debug entries",
			content: `**调试日志**:
- debug1: 问题1, 复现1, 猜想1, 验证1, 修复1, 进展1
- debug2: 问题2, 复现2, 猜想2, 验证2, 修复2, 进展2
- debug3: 问题3, 复现3, 猜想3, 验证3, 修复3, 进展3`,
			wantCount: 3,
			wantErr:   false,
		},
		{
			name: "debug entry with spaces",
			content: `**调试日志**:
- debug1:  问题描述  ,  如何复现  ,  可能原因  ,  验证方法  ,  修复方案  ,  当前进展  `,
			wantCount: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDebug(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDebug() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil {
				t.Errorf("ParseDebug() returned nil")
				return
			}
			if len(got.Entries) != tt.wantCount {
				t.Errorf("ParseDebug() got %d entries, want %d", len(got.Entries), tt.wantCount)
			}
		})
	}
}

func TestParseDebugContent(t *testing.T) {
	content := `**调试日志**:
- debug1: 日志轮转时丢失消息, 高频写入时触发轮转, 猜想: 1)文件句柄未同步 2)并发竞争, 验证: 添加文件锁测试, 修复: 使用 flock 同步, 待修复`

	debugInfo, err := ParseDebug(content)
	if err != nil {
		t.Fatalf("ParseDebug() error: %v", err)
	}

	if len(debugInfo.Entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(debugInfo.Entries))
	}

	entry := debugInfo.Entries[0]
	if entry.ID != 1 {
		t.Errorf("Expected ID=1, got %d", entry.ID)
	}
	if entry.Phenomenon != "日志轮转时丢失消息" {
		t.Errorf("Expected phenomenon='日志轮转时丢失消息', got '%s'", entry.Phenomenon)
	}
	if entry.Reproduce != "高频写入时触发轮转" {
		t.Errorf("Expected reproduce='高频写入时触发轮转', got '%s'", entry.Reproduce)
	}
	if !strings.Contains(entry.Hypothesis, "文件句柄未同步") {
		t.Errorf("Expected hypothesis to contain '文件句柄未同步', got '%s'", entry.Hypothesis)
	}
	if entry.Progress != "待修复" {
		t.Errorf("Expected progress='待修复', got '%s'", entry.Progress)
	}
}

func TestAppendDebug(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		entry       DebugEntry
		wantContain string
	}{
		{
			name:    "append to empty content",
			content: "",
			entry: DebugEntry{
				ID:         1,
				Phenomenon: "问题1",
				Reproduce:  "复现1",
				Hypothesis: "猜想1",
				Verify:     "验证1",
				Fix:        "修复1",
				Progress:   "进展1",
			},
			wantContain: "- debug1: 问题1, 复现1, 猜想1, 验证1, 修复1, 进展1",
		},
		{
			name: "append to existing debug log",
			content: `**调试日志**:
- debug1: 问题1, 复现1, 猜想1, 验证1, 修复1, 进展1`,
			entry: DebugEntry{
				ID:         2,
				Phenomenon: "问题2",
				Reproduce:  "复现2",
				Hypothesis: "猜想2",
				Verify:     "验证2",
				Fix:        "修复2",
				Progress:   "进展2",
			},
			wantContain: "- debug2: 问题2, 复现2, 猜想2, 验证2, 修复2, 进展2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AppendDebug(tt.content, tt.entry)
			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("AppendDebug() result doesn't contain '%s'\nGot: %s", tt.wantContain, got)
			}
		})
	}
}

func TestGetDebugCount(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "empty content",
			content: "",
			want:    0,
		},
		{
			name: "single entry",
			content: `**调试日志**:
- debug1: 问题1, 复现1, 猜想1, 验证1, 修复1, 进展1`,
			want: 1,
		},
		{
			name: "multiple entries",
			content: `**调试日志**:
- debug1: 问题1, 复现1, 猜想1, 验证1, 修复1, 进展1
- debug2: 问题2, 复现2, 猜想2, 验证2, 修复2, 进展2
- debug3: 问题3, 复现3, 猜想3, 验证3, 修复3, 进展3`,
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDebugCount(tt.content)
			if got != tt.want {
				t.Errorf("GetDebugCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGenerateDebugEntry(t *testing.T) {
	tests := []struct {
		name       string
		id         int
		phenomenon string
		reproduce  string
		hypothesis string
		verify     string
		fix        string
		progress   string
		want       string
	}{
		{
			name:       "generate entry 1",
			id:         1,
			phenomenon: "问题1",
			reproduce:  "复现1",
			hypothesis: "猜想1",
			verify:     "验证1",
			fix:        "修复1",
			progress:   "进展1",
			want:       "- debug1: 问题1, 复现1, 猜想1, 验证1, 修复1, 进展1",
		},
		{
			name:       "generate entry 2",
			id:         2,
			phenomenon: "问题2",
			reproduce:  "复现2",
			hypothesis: "猜想2",
			verify:     "验证2",
			fix:        "修复2",
			progress:   "进展2",
			want:       "- debug2: 问题2, 复现2, 猜想2, 验证2, 修复2, 进展2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateDebugEntry(tt.id, tt.phenomenon, tt.reproduce,
				tt.hypothesis, tt.verify, tt.fix, tt.progress)
			if got != tt.want {
				t.Errorf("GenerateDebugEntry() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGetNextDebugID(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "empty content",
			content: "",
			want:    1,
		},
		{
			name: "single entry",
			content: `**调试日志**:
- debug1: 问题1, 复现1, 猜想1, 验证1, 修复1, 进展1`,
			want: 2,
		},
		{
			name: "multiple entries",
			content: `**调试日志**:
- debug1: 问题1, 复现1, 猜想1, 验证1, 修复1, 进展1
- debug2: 问题2, 复现2, 猜想2, 验证2, 修复2, 进展2
- debug3: 问题3, 复现3, 猜想3, 验证3, 修复3, 进展3`,
			want: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetNextDebugID(tt.content)
			if got != tt.want {
				t.Errorf("GetNextDebugID() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGetDebugByID(t *testing.T) {
	content := `**调试日志**:
- debug1: 问题1, 复现1, 猜想1, 验证1, 修复1, 进展1
- debug2: 问题2, 复现2, 猜想2, 验证2, 修复2, 进展2`

	tests := []struct {
		name    string
		id      int
		wantErr bool
	}{
		{
			name:    "get existing entry",
			id:      1,
			wantErr: false,
		},
		{
			name:    "get another existing entry",
			id:      2,
			wantErr: false,
		},
		{
			name:    "get non-existing entry",
			id:      3,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetDebugByID(content, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDebugByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("GetDebugByID() returned nil for existing entry")
			}
			if !tt.wantErr && got.ID != tt.id {
				t.Errorf("GetDebugByID() got ID %d, want %d", got.ID, tt.id)
			}
		})
	}
}

func TestUpdateDebugEntry(t *testing.T) {
	content := `**调试日志**:
- debug1: 问题1, 复现1, 猜想1, 验证1, 修复1, 进展1`

	newContent, err := UpdateDebugEntry(content, 1, "新问题1", "新复现1", "新猜想1", "新验证1", "新修复1", "已修复")
	if err != nil {
		t.Fatalf("UpdateDebugEntry() error: %v", err)
	}

	if !strings.Contains(newContent, "新问题1") {
		t.Errorf("UpdateDebugEntry() didn't update phenomenon")
	}
	if !strings.Contains(newContent, "已修复") {
		t.Errorf("UpdateDebugEntry() didn't update progress")
	}

	// Verify the updated entry can be parsed correctly
	debugInfo, err := ParseDebug(newContent)
	if err != nil {
		t.Fatalf("ParseDebug() error: %v", err)
	}
	if len(debugInfo.Entries) != 1 {
		t.Fatalf("Expected 1 entry after update, got %d", len(debugInfo.Entries))
	}
	if debugInfo.Entries[0].Phenomenon != "新问题1" {
		t.Errorf("Expected phenomenon='新问题1', got '%s'", debugInfo.Entries[0].Phenomenon)
	}
}

func TestIntegrationParseAndAppend(t *testing.T) {
	// Start with empty content
	content := ""

	// Append first entry
	entry1 := DebugEntry{
		ID:         1,
		Phenomenon: "问题1",
		Reproduce:  "复现1",
		Hypothesis: "猜想1",
		Verify:     "验证1",
		Fix:        "修复1",
		Progress:   "进展1",
	}
	content = AppendDebug(content, entry1)

	// Append second entry
	entry2 := DebugEntry{
		ID:         2,
		Phenomenon: "问题2",
		Reproduce:  "复现2",
		Hypothesis: "猜想2",
		Verify:     "验证2",
		Fix:        "修复2",
		Progress:   "进展2",
	}
	content = AppendDebug(content, entry2)

	// Parse and verify
	debugInfo, err := ParseDebug(content)
	if err != nil {
		t.Fatalf("ParseDebug() error: %v", err)
	}

	if len(debugInfo.Entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(debugInfo.Entries))
	}

	if debugInfo.Entries[0].Phenomenon != "问题1" {
		t.Errorf("Expected first entry phenomenon='问题1', got '%s'", debugInfo.Entries[0].Phenomenon)
	}
	if debugInfo.Entries[1].Phenomenon != "问题2" {
		t.Errorf("Expected second entry phenomenon='问题2', got '%s'", debugInfo.Entries[1].Phenomenon)
	}

	// Test GetDebugCount
	count := GetDebugCount(content)
	if count != 2 {
		t.Errorf("GetDebugCount() = %d, want 2", count)
	}

	// Test GetNextDebugID
	nextID := GetNextDebugID(content)
	if nextID != 3 {
		t.Errorf("GetNextDebugID() = %d, want 3", nextID)
	}
}

func TestComplexDebugEntry(t *testing.T) {
	// Test with complex content that includes commas in fields
	content := `**调试日志**:
- debug1: JSON 序列化失败, 复杂对象循环引用, 猜想: 1)缺少循环引用处理 2)未使用 JSON.stringify 的 replacer, 验证: 添加 replacer 函数测试, 修复: 使用 WeakSet 检测循环引用, 待修复`

	debugInfo, err := ParseDebug(content)
	if err != nil {
		t.Fatalf("ParseDebug() error: %v", err)
	}

	if len(debugInfo.Entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(debugInfo.Entries))
	}

	entry := debugInfo.Entries[0]
	if entry.Phenomenon != "JSON 序列化失败" {
		t.Errorf("Expected phenomenon='JSON 序列化失败', got '%s'", entry.Phenomenon)
	}
	if !strings.Contains(entry.Hypothesis, "缺少循环引用处理") {
		t.Errorf("Expected hypothesis to contain '缺少循环引用处理'")
	}
}
