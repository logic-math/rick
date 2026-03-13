package feedback

import (
	"strings"
	"testing"
)

func TestNewI18nMessages(t *testing.T) {
	i18n := NewI18nMessages(LangEnglish)

	if i18n == nil {
		t.Fatal("NewI18nMessages returned nil")
	}
	if i18n.lang != LangEnglish {
		t.Errorf("expected language %v, got %v", LangEnglish, i18n.lang)
	}
}

func TestSetLanguage(t *testing.T) {
	i18n := NewI18nMessages(LangEnglish)
	i18n.SetLanguage(LangChinese)

	if i18n.lang != LangChinese {
		t.Errorf("expected language %v, got %v", LangChinese, i18n.lang)
	}
}

func TestRegisterAndGet(t *testing.T) {
	i18n := NewI18nMessages(LangEnglish)

	translations := map[Language]string{
		LangEnglish: "Hello %s",
		LangChinese: "你好 %s",
	}

	i18n.Register("greeting", translations)

	engMsg := i18n.Get("greeting", "World")
	if engMsg != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", engMsg)
	}

	i18n.SetLanguage(LangChinese)
	zhMsg := i18n.Get("greeting", "世界")
	if zhMsg != "你好 世界" {
		t.Errorf("expected '你好 世界', got '%s'", zhMsg)
	}
}

func TestGetFallback(t *testing.T) {
	i18n := NewI18nMessages(LangChinese)

	translations := map[Language]string{
		LangEnglish: "Error: %s",
	}

	i18n.Register("error", translations)

	// Should fallback to English when Chinese not available
	msg := i18n.Get("error", "test")
	if msg != "Error: test" {
		t.Errorf("expected 'Error: test', got '%s'", msg)
	}
}

func TestGetNotFound(t *testing.T) {
	i18n := NewI18nMessages(LangEnglish)

	msg := i18n.Get("nonexistent")
	if msg != "nonexistent" {
		t.Errorf("expected 'nonexistent', got '%s'", msg)
	}
}

func TestDefaultI18nMessages(t *testing.T) {
	i18n := DefaultI18nMessages(LangEnglish)

	if i18n == nil {
		t.Fatal("DefaultI18nMessages returned nil")
	}

	// Test a known message
	msg := i18n.Get("ERR_INVALID_JOB_ID", "bad_id")
	if !strings.Contains(msg, "bad_id") {
		t.Errorf("expected message to contain 'bad_id', got '%s'", msg)
	}

	// Test Chinese
	i18n.SetLanguage(LangChinese)
	msg = i18n.Get("ERR_INVALID_JOB_ID", "bad_id")
	if !strings.Contains(msg, "bad_id") {
		t.Errorf("expected message to contain 'bad_id', got '%s'", msg)
	}
}

func TestParseLanguageFromEnv(t *testing.T) {
	testCases := []struct {
		input    string
		expected Language
	}{
		{"zh", LangChinese},
		{"zh_CN", LangChinese},
		{"zh_TW", LangChinese},
		{"en", LangEnglish},
		{"en_US", LangEnglish},
		{"en_GB", LangEnglish},
		{"ja", LangEnglish}, // Default to English
		{"", LangEnglish},   // Default to English
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			lang := ParseLanguageFromEnv(tc.input)
			if lang != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, lang)
			}
		})
	}
}

func TestI18nMessagesWithMultipleLanguages(t *testing.T) {
	i18n := NewI18nMessages(LangEnglish)

	translations := map[Language]string{
		LangEnglish: "File not found: %s",
		LangChinese: "文件未找到: %s",
	}

	i18n.Register("file_error", translations)

	// Test English
	msg := i18n.Get("file_error", "config.json")
	if msg != "File not found: config.json" {
		t.Errorf("expected 'File not found: config.json', got '%s'", msg)
	}

	// Switch to Chinese
	i18n.SetLanguage(LangChinese)
	msg = i18n.Get("file_error", "config.json")
	if msg != "文件未找到: config.json" {
		t.Errorf("expected '文件未找到: config.json', got '%s'", msg)
	}
}

func TestDefaultI18nMessagesChinese(t *testing.T) {
	i18n := DefaultI18nMessages(LangChinese)

	msg := i18n.Get("ERR_JOB_NOT_FOUND", "job_1")
	if !strings.Contains(msg, "job_1") {
		t.Errorf("expected message to contain 'job_1', got '%s'", msg)
	}

	// Should contain Chinese characters
	if !strings.Contains(msg, "找不到") {
		t.Errorf("expected Chinese message, got '%s'", msg)
	}
}

func TestDefaultI18nMessagesWithSuggestions(t *testing.T) {
	i18n := DefaultI18nMessages(LangEnglish)

	// Test error message
	errMsg := i18n.Get("ERR_CONFIG_NOT_FOUND", "config.json")
	if !strings.Contains(errMsg, "config.json") {
		t.Errorf("expected error message to contain 'config.json', got '%s'", errMsg)
	}

	// Test suggestion message
	sugMsg := i18n.Get("SUG_CONFIG_NOT_FOUND")
	if !strings.Contains(sugMsg, "rick init") {
		t.Errorf("expected suggestion to contain 'rick init', got '%s'", sugMsg)
	}
}
