package feedback

import (
	"fmt"
	"strings"
)

// Language represents supported languages
type Language string

const (
	LangChinese Language = "zh"
	LangEnglish Language = "en"
)

// I18nMessages holds localized error messages
type I18nMessages struct {
	lang Language
	msgs map[string]map[Language]string
}

// NewI18nMessages creates a new I18n message manager
func NewI18nMessages(lang Language) *I18nMessages {
	return &I18nMessages{
		lang: lang,
		msgs: make(map[string]map[Language]string),
	}
}

// SetLanguage sets the active language
func (i *I18nMessages) SetLanguage(lang Language) {
	i.lang = lang
}

// Register registers a message key with translations
func (i *I18nMessages) Register(key string, translations map[Language]string) {
	i.msgs[key] = translations
}

// Get retrieves a translated message
func (i *I18nMessages) Get(key string, args ...interface{}) string {
	if msgs, ok := i.msgs[key]; ok {
		if msg, ok := msgs[i.lang]; ok {
			return fmt.Sprintf(msg, args...)
		}
		// Fallback to English
		if msg, ok := msgs[LangEnglish]; ok {
			return fmt.Sprintf(msg, args...)
		}
	}
	return key // Return key if not found
}

// DefaultI18nMessages creates default I18n messages
func DefaultI18nMessages(lang Language) *I18nMessages {
	i18n := NewI18nMessages(lang)

	// Error messages
	errorMsgs := map[string]map[Language]string{
		"ERR_INVALID_JOB_ID": {
			LangChinese: "无效的 Job ID: %s",
			LangEnglish: "Invalid job ID: %s",
		},
		"ERR_JOB_NOT_FOUND": {
			LangChinese: "找不到 Job: %s",
			LangEnglish: "Job not found: %s",
		},
		"ERR_TASK_FAILED": {
			LangChinese: "Task 执行失败: %s",
			LangEnglish: "Task execution failed: %s",
		},
		"ERR_CONFIG_NOT_FOUND": {
			LangChinese: "配置文件未找到: %s",
			LangEnglish: "Configuration file not found: %s",
		},
		"ERR_WORKSPACE_NOT_FOUND": {
			LangChinese: "工作空间未找到: %s",
			LangEnglish: "Workspace not found: %s",
		},
		"ERR_GIT_OPERATION_FAILED": {
			LangChinese: "Git 操作失败: %s",
			LangEnglish: "Git operation failed: %s",
		},
		"ERR_PARSER_FAILED": {
			LangChinese: "解析失败: %s",
			LangEnglish: "Parsing failed: %s",
		},
		"ERR_EXECUTOR_FAILED": {
			LangChinese: "执行器错误: %s",
			LangEnglish: "Executor error: %s",
		},
		"ERR_PERMISSION_DENIED": {
			LangChinese: "权限被拒绝: %s",
			LangEnglish: "Permission denied: %s",
		},
		"ERR_FILE_NOT_FOUND": {
			LangChinese: "文件未找到: %s",
			LangEnglish: "File not found: %s",
		},
	}

	for key, translations := range errorMsgs {
		i18n.Register(key, translations)
	}

	// Suggestion messages
	suggestionMsgs := map[string]map[Language]string{
		"SUG_INVALID_JOB_ID": {
			LangChinese: "💡 建议: Job ID 应该是 'job_1', 'job_2' 等格式",
			LangEnglish: "💡 Suggestion: Job ID should be in format 'job_1', 'job_2', etc.",
		},
		"SUG_JOB_NOT_FOUND": {
			LangChinese: "💡 建议: 请先运行 'rick plan' 创建任务",
			LangEnglish: "💡 Suggestion: Please run 'rick plan' to create tasks first",
		},
		"SUG_CONFIG_NOT_FOUND": {
			LangChinese: "💡 建议: 请先运行 'rick init' 初始化项目",
			LangEnglish: "💡 Suggestion: Please run 'rick init' to initialize the project first",
		},
		"SUG_WORKSPACE_NOT_FOUND": {
			LangChinese: "💡 建议: 确保你在项目根目录中，或检查 .rick 目录是否存在",
			LangEnglish: "💡 Suggestion: Make sure you are in the project root directory or check if .rick directory exists",
		},
		"SUG_GIT_OPERATION_FAILED": {
			LangChinese: "💡 建议: 检查 Git 是否已安装，以及你是否有正确的权限",
			LangEnglish: "💡 Suggestion: Check if Git is installed and you have proper permissions",
		},
		"SUG_PERMISSION_DENIED": {
			LangChinese: "💡 建议: 检查文件权限或尝试使用 sudo（如果适用）",
			LangEnglish: "💡 Suggestion: Check file permissions or try using sudo (if applicable)",
		},
	}

	for key, translations := range suggestionMsgs {
		i18n.Register(key, translations)
	}

	return i18n
}

// ParseLanguageFromEnv parses language from environment variable
func ParseLanguageFromEnv(envLang string) Language {
	envLang = strings.ToLower(envLang)
	if strings.HasPrefix(envLang, "zh") {
		return LangChinese
	}
	return LangEnglish
}
