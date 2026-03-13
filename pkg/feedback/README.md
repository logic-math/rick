# Feedback Package - Error Handling and User Feedback

## Overview

The `feedback` package provides a comprehensive error handling and user feedback system for the Rick CLI. It includes:

- **Internationalization (I18n)**: Support for multiple languages (Chinese/English)
- **Error Handling**: Structured error types with stack traces and context information
- **Progress Indicators**: Progress bars, status indicators, and spinners
- **Verbose Logging**: Multi-level logging with verbose mode support

## Components

### 1. I18nMessages (i18n.go)

Provides multilingual message support with fallback mechanisms.

```go
// Create an I18n manager
i18n := feedback.DefaultI18nMessages(feedback.LangEnglish)

// Get a localized message
msg := i18n.Get("ERR_INVALID_JOB_ID", "bad_id")

// Switch language
i18n.SetLanguage(feedback.LangChinese)
msg = i18n.Get("ERR_INVALID_JOB_ID", "bad_id")
```

**Supported Languages:**
- `LangEnglish`: English
- `LangChinese`: Chinese (Simplified)

**Features:**
- Automatic language detection from environment (`LANG` variable)
- Fallback to English if translation not available
- Easy registration of custom messages

### 2. ErrorHandler (error_handler.go)

Provides structured error handling with stack traces and suggestions.

```go
// Create an error handler
i18n := feedback.DefaultI18nMessages(feedback.LangEnglish)
eh := feedback.NewErrorHandler(i18n)
eh.SetIncludeStackTrace(true)

// Handle an error
err := errors.New("something went wrong")
ewc := eh.Handle("ConfigError", err)

// Add context information
ewc.AddContext("file", "config.json")
ewc.AddContext("line", 42)

// Format and display
formatted := ewc.Format(i18n, verbose)
fmt.Println(formatted)

// Get suggestion
suggestion := ewc.GetSuggestion(i18n)
fmt.Println(suggestion)
```

**Features:**
- Stack trace capture with configurable depth
- Context information attachment
- Localized error messages and suggestions
- Supports both verbose and concise formatting

### 3. Progress Indicators (progress.go)

Provides visual feedback for long-running operations.

#### ProgressBar

```go
// Create a progress bar
pb := feedback.NewProgressBar("Processing", 100)

// Update progress
pb.Update(50)
pb.Increment()

// Complete
pb.Complete()
```

Output: `Processing [███████░░░░░░░░░░░░░░░░░░░░░░] 50/100 (50%) - 2m30s`

#### StatusIndicator

```go
si := feedback.NewStatusIndicator()

si.Success("Operation completed")      // ✅
si.Error("Operation failed")           // ❌
si.Warning("Warning message")          // ⚠️
si.Info("Info message")                // ℹ️
si.Debug("Debug message")              // 🐛
si.Running("Running task")             // 🔄
si.Pending("Pending task")             // ⏳
si.Tip("Helpful tip")                  // 💡
```

#### Spinner

```go
spinner := feedback.NewSpinner()

for i := 0; i < 100; i++ {
    spinner.Print("Loading...")
    time.Sleep(100 * time.Millisecond)
}
fmt.Println() // Clear the line
```

### 4. VerboseLogger (verbose_logger.go)

Provides multi-level logging with verbose mode support.

```go
// Create a verbose logger
vl := feedback.NewVerboseLogger(verbose)

// Log at different levels
vl.Info("Info message")
vl.Warn("Warning message")
vl.Error("Error message")
vl.Debug("Debug message")      // Only shown if verbose enabled
vl.Verbose("Verbose message")  // Only shown if verbose enabled
vl.Trace("Trace message")      // Only shown if verbose enabled

// Conditional logging
if vl.IsVerbose() {
    vl.Verbose("Detailed information: %v", complexData)
}

// Command logging
vl.LogCommand("git", []string{"commit", "-m", "test"})
vl.LogCommandResult("git", 0, "Commit successful")

// Context logging
details := map[string]interface{}{
    "file": "config.json",
    "line": 42,
}
vl.WithContext("LoadConfig", details)
```

**Log Levels:**
1. `Info` - General information (always shown)
2. `Warn` - Warning messages (always shown)
3. `Error` - Error messages (always shown)
4. `Debug` - Debug information (verbose mode only)
5. `Verbose` - Verbose details (verbose mode only)
6. `Trace` - Trace information (verbose mode only)

## Integration with Commands

The `FeedbackContext` helper (in `internal/cmd/feedback_helper.go`) integrates all feedback components for easy use in Cobra commands:

```go
// In a command handler
func (cmd *cobra.Command) RunE(args []string) error {
    fc := cmd.NewFeedbackContext(cmd)

    fc.PrintStepStart("Loading configuration")

    // Do work...

    if err != nil {
        fc.HandleError("ConfigError", err)
        return err
    }

    fc.PrintStepComplete("Configuration loaded")

    // Show progress
    pb := fc.CreateProgressBar("Processing", 100)
    for i := 0; i <= 100; i++ {
        pb.Update(i)
        // Do work...
    }
    pb.Complete()

    return nil
}
```

## Error Message Examples

### English
```
❌ [ConfigError] Configuration file not found: ~/.rick/config.json
💡 Suggestion: Please run 'rick init' to initialize the project first

📋 Context:
  file: ~/.rick/config.json
  operation: LoadConfig

📍 Stack Trace:
  1. main() at cmd/rick/main.go:42
  2. runCommand() at internal/cmd/root.go:23
  ...
```

### Chinese
```
❌ [ConfigError] 配置文件未找到: ~/.rick/config.json
💡 建议: 请先运行 'rick init' 初始化项目

📋 Context:
  file: ~/.rick/config.json
  operation: LoadConfig

📍 Stack Trace:
  1. main() at cmd/rick/main.go:42
  2. runCommand() at internal/cmd/root.go:23
  ...
```

## Testing

All components are thoroughly tested with 96.2% coverage:

- **i18n_test.go**: 14 tests for I18nMessages
- **error_handler_test.go**: 16 tests for ErrorHandler
- **progress_test.go**: 28 tests for progress indicators
- **verbose_logger_test.go**: 20 tests for VerboseLogger
- **feedback_helper_test.go**: 22 integration tests

Run tests:
```bash
go test ./pkg/feedback -v
go test ./internal/cmd -v
```

## Usage Patterns

### Pattern 1: Simple Error Handling
```go
if err != nil {
    fc.HandleError("OperationError", err)
    return err
}
```

### Pattern 2: Detailed Error with Context
```go
if err != nil {
    fc.HandleErrorWithMessage("ParseError",
        fmt.Sprintf("Failed to parse line %d", lineNum),
        err)
    return err
}
```

### Pattern 3: Progress Tracking
```go
pb := fc.CreateProgressBar("Processing files", len(files))
for _, file := range files {
    process(file)
    pb.Increment()
}
pb.Complete()
```

### Pattern 4: Step-by-step Operations
```go
fc.PrintStepStart("Initializing workspace")
// ... do work ...
fc.PrintStepComplete("Workspace initialized")

fc.PrintStepStart("Loading configuration")
// ... do work ...
if err != nil {
    fc.PrintStepError("Configuration loading failed")
    return err
}
fc.PrintStepComplete("Configuration loaded")
```

### Pattern 5: Verbose Debugging
```go
fc.LogVerbose("Processing configuration: %v", config)
fc.LogTrace("Detailed trace: %v", details)
fc.WithContext("LoadConfig", map[string]interface{}{
    "file": "config.json",
    "size": fileSize,
})
```

## Customization

### Adding Custom Messages

```go
i18n := feedback.DefaultI18nMessages(feedback.LangEnglish)

// Register custom message
i18n.Register("ERR_CUSTOM", map[feedback.Language]string{
    feedback.LangEnglish: "Custom error: %s",
    feedback.LangChinese: "自定义错误: %s",
})

// Use it
msg := i18n.Get("ERR_CUSTOM", "details")
```

### Custom Error Types

```go
type CustomError struct {
    message string
}

func (e *CustomError) Error() string {
    return e.message
}

// Use with ErrorHandler
eh := feedback.NewErrorHandler(i18n)
ewc := eh.Handle("CustomError", &CustomError{"test"})
```

## Best Practices

1. **Always provide context**: Use `AddContext` to help users understand what went wrong
2. **Use appropriate log levels**: Don't spam with debug messages unless verbose mode is enabled
3. **Provide suggestions**: Use error suggestions to guide users to solutions
4. **Show progress**: Use progress bars for long-running operations
5. **Be consistent**: Use the same error types and messages across commands
6. **Localize messages**: Support multiple languages for better user experience

## Performance Considerations

- Stack trace capture has minimal overhead when disabled (default)
- Progress bar rendering is optimized to avoid excessive output
- Verbose logging is completely skipped when disabled
- I18n message lookup is O(1) with map-based storage
