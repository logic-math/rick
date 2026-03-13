package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/pkg/feedback"
)

// FeedbackContext holds feedback utilities for commands
type FeedbackContext struct {
	I18n         *feedback.I18nMessages
	ErrorHandler *feedback.ErrorHandler
	Logger       *feedback.VerboseLogger
	StatusInd    *feedback.StatusIndicator
	ProgressBar  *feedback.ProgressBar
}

// NewFeedbackContext creates a new feedback context
func NewFeedbackContext(cmd *cobra.Command) *FeedbackContext {
	// Determine language from environment or default to English
	langEnv := os.Getenv("LANG")
	lang := feedback.ParseLanguageFromEnv(langEnv)

	// Create I18n messages
	i18n := feedback.DefaultI18nMessages(lang)

	// Get verbose flag
	verboseFlag, _ := cmd.Flags().GetBool("verbose")

	// Create logger
	logger := feedback.NewVerboseLogger(verboseFlag)

	// Create error handler
	errorHandler := feedback.NewErrorHandler(i18n)
	if verboseFlag {
		errorHandler.SetIncludeStackTrace(true)
	}

	// Create status indicator
	statusInd := feedback.NewStatusIndicator()

	return &FeedbackContext{
		I18n:         i18n,
		ErrorHandler: errorHandler,
		Logger:       logger,
		StatusInd:    statusInd,
		ProgressBar:  nil, // Will be created as needed
	}
}

// HandleError formats and prints an error with suggestions
func (fc *FeedbackContext) HandleError(errorType string, err error) {
	ewc := fc.ErrorHandler.Handle(errorType, err)
	formatted := ewc.Format(fc.I18n, fc.Logger.IsVerbose())
	fc.Logger.Error("%s", formatted)

	// Print suggestion if available
	suggestion := ewc.GetSuggestion(fc.I18n)
	if suggestion != "" {
		fc.StatusInd.Tip(suggestion)
	}
}

// HandleErrorWithMessage formats and prints an error with custom message and suggestions
func (fc *FeedbackContext) HandleErrorWithMessage(errorType, message string, err error) {
	ewc := fc.ErrorHandler.HandleWithMessage(errorType, message, err)
	formatted := ewc.Format(fc.I18n, fc.Logger.IsVerbose())
	fc.Logger.Error("%s", formatted)

	// Print suggestion if available
	suggestion := ewc.GetSuggestion(fc.I18n)
	if suggestion != "" {
		fc.StatusInd.Tip(suggestion)
	}
}

// SetLanguage changes the active language
func (fc *FeedbackContext) SetLanguage(lang feedback.Language) {
	fc.I18n.SetLanguage(lang)
}

// SetVerbose enables or disables verbose mode
func (fc *FeedbackContext) SetVerbose(verbose bool) {
	fc.Logger.SetVerbose(verbose)
	if verbose {
		fc.ErrorHandler.SetIncludeStackTrace(true)
	}
}

// CreateProgressBar creates a new progress bar
func (fc *FeedbackContext) CreateProgressBar(title string, total int) *feedback.ProgressBar {
	fc.ProgressBar = feedback.NewProgressBar(title, total)
	return fc.ProgressBar
}

// GetLocalizedError returns a localized error message
func (fc *FeedbackContext) GetLocalizedError(errorKey string, args ...interface{}) string {
	return fc.I18n.Get(errorKey, args...)
}

// GetLocalizedSuggestion returns a localized suggestion message
func (fc *FeedbackContext) GetLocalizedSuggestion(suggestionKey string, args ...interface{}) string {
	return fc.I18n.Get(suggestionKey, args...)
}

// PrintStepStart prints the start of a step
func (fc *FeedbackContext) PrintStepStart(step string) {
	fc.StatusInd.Running(step)
}

// PrintStepComplete prints the completion of a step
func (fc *FeedbackContext) PrintStepComplete(step string) {
	fc.StatusInd.Success(step)
}

// PrintStepWarning prints a warning for a step
func (fc *FeedbackContext) PrintStepWarning(step string) {
	fc.StatusInd.Warning(step)
}

// PrintStepError prints an error for a step
func (fc *FeedbackContext) PrintStepError(step string) {
	fc.StatusInd.Error(step)
}

// GetMsgWithContext returns a message with context information
func (fc *FeedbackContext) GetMsgWithContext(operation string, details map[string]interface{}) {
	fc.Logger.WithContext(operation, details)
}

// LogVerbose logs a verbose message
func (fc *FeedbackContext) LogVerbose(format string, args ...interface{}) {
	fc.Logger.Verbose(format, args...)
}

// LogDebug logs a debug message
func (fc *FeedbackContext) LogDebug(format string, args ...interface{}) {
	fc.Logger.Debug(format, args...)
}

// LogTrace logs a trace message
func (fc *FeedbackContext) LogTrace(format string, args ...interface{}) {
	fc.Logger.Trace(format, args...)
}

// LogCommand logs a command execution
func (fc *FeedbackContext) LogCommand(cmd string, args []string) {
	fc.Logger.LogCommand(cmd, args)
}

// LogCommandResult logs a command result
func (fc *FeedbackContext) LogCommandResult(cmd string, exitCode int, output string) {
	fc.Logger.LogCommandResult(cmd, exitCode, output)
}

// IsVerboseMode returns whether verbose mode is enabled
func (fc *FeedbackContext) IsVerboseMode() bool {
	return fc.Logger.IsVerbose()
}

// PrintInfo prints an info message
func (fc *FeedbackContext) PrintInfo(message string) {
	fc.StatusInd.Info(message)
}

// PrintWarning prints a warning message
func (fc *FeedbackContext) PrintWarning(message string) {
	fc.StatusInd.Warning(message)
}

// PrintDebug prints a debug message
func (fc *FeedbackContext) PrintDebug(message string) {
	fc.StatusInd.Debug(message)
}

// PrintPending prints a pending message
func (fc *FeedbackContext) PrintPending(message string) {
	fc.StatusInd.Pending(message)
}

// FormatErrorMessage returns a formatted error message string
func (fc *FeedbackContext) FormatErrorMessage(errorType, message string) string {
	return strings.Join([]string{"[" + errorType + "]", message}, " ")
}
