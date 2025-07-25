package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LogLevel represents different log levels
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger interface for structured logging
type Logger interface {
	Debug(ctx context.Context, msg string, fields ...interface{})
	Info(ctx context.Context, msg string, fields ...interface{})
	Warn(ctx context.Context, err error, msg string, fields ...interface{})
	Error(ctx context.Context, err error, msg string, fields ...interface{})
	Fatal(ctx context.Context, err error, msg string, fields ...interface{})

	With(fields ...interface{}) Logger
	WithComponent(component string) Logger
}

// TemplarLogger implements structured logging for Templar
type TemplarLogger struct {
	logger    *slog.Logger
	level     LogLevel
	component string
	fields    map[string]interface{}
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level      LogLevel
	Format     string // "json" or "text"
	Output     io.Writer
	TimeFormat string
	AddSource  bool
	Component  string
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *LoggerConfig {
	return &LoggerConfig{
		Level:      LevelInfo,
		Format:     "text",
		Output:     os.Stdout,
		TimeFormat: time.RFC3339,
		AddSource:  true,
	}
}

// NewLogger creates a new structured logger
func NewLogger(config *LoggerConfig) *TemplarLogger {
	if config == nil {
		config = DefaultConfig()
	}

	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level:     slog.Level(config.Level - 1), // Adjust for slog levels
		AddSource: config.AddSource,
	}

	if config.Format == "json" {
		handler = slog.NewJSONHandler(config.Output, opts)
	} else {
		handler = slog.NewTextHandler(config.Output, opts)
	}

	logger := slog.New(handler)

	return &TemplarLogger{
		logger:    logger,
		level:     config.Level,
		component: config.Component,
		fields:    make(map[string]interface{}),
	}
}

// Debug logs a debug message
func (l *TemplarLogger) Debug(ctx context.Context, msg string, fields ...interface{}) {
	if l.level > LevelDebug {
		return
	}
	l.log(ctx, slog.LevelDebug, nil, msg, fields...)
}

// Info logs an info message
func (l *TemplarLogger) Info(ctx context.Context, msg string, fields ...interface{}) {
	if l.level > LevelInfo {
		return
	}
	l.log(ctx, slog.LevelInfo, nil, msg, fields...)
}

// Warn logs a warning message
func (l *TemplarLogger) Warn(ctx context.Context, err error, msg string, fields ...interface{}) {
	if l.level > LevelWarn {
		return
	}
	l.log(ctx, slog.LevelWarn, err, msg, fields...)
}

// Error logs an error message
func (l *TemplarLogger) Error(ctx context.Context, err error, msg string, fields ...interface{}) {
	if l.level > LevelError {
		return
	}
	l.log(ctx, slog.LevelError, err, msg, fields...)
}

// Fatal logs a fatal message
// Note: This method logs at ERROR level but does not call os.Exit.
// The caller is responsible for handling the fatal condition appropriately.
func (l *TemplarLogger) Fatal(ctx context.Context, err error, msg string, fields ...interface{}) {
	l.log(ctx, slog.LevelError, err, msg, fields...)
}

// With creates a new logger with additional fields
func (l *TemplarLogger) With(fields ...interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}

	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			if key, ok := fields[i].(string); ok {
				newFields[key] = fields[i+1]
			}
		}
	}

	return &TemplarLogger{
		logger:    l.logger,
		level:     l.level,
		component: l.component,
		fields:    newFields,
	}
}

// WithComponent creates a new logger with component context
func (l *TemplarLogger) WithComponent(component string) Logger {
	return &TemplarLogger{
		logger:    l.logger,
		level:     l.level,
		component: component,
		fields:    l.fields,
	}
}

// log is the internal logging method
func (l *TemplarLogger) log(ctx context.Context, level slog.Level, err error, msg string, fields ...interface{}) {
	// Defensive programming - ensure we don't panic on nil logger
	if l.logger == nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Logger is nil - message: %s\n", msg)
		return
	}

	attrs := make([]slog.Attr, 0, len(l.fields)+len(fields)/2+3)

	// Add component if set
	if l.component != "" {
		attrs = append(attrs, slog.String("component", l.component))
	}

	// Add error if provided with enhanced error context
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
		// Add error type for better error categorization
		attrs = append(attrs, slog.String("error_type", fmt.Sprintf("%T", err)))
	}

	// Add persistent fields
	for k, v := range l.fields {
		attrs = append(attrs, slog.Any(k, v))
	}

	// Add provided fields with validation
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			if key, ok := fields[i].(string); ok && key != "" {
				// Sanitize field values for security
				value := fields[i+1]
				if str, isString := value.(string); isString {
					value = SanitizeForLog(str)
				}
				attrs = append(attrs, slog.Any(key, value))
			}
		}
	}

	record := slog.NewRecord(time.Now(), level, msg, 0)
	record.AddAttrs(attrs...)

	// Handle potential errors in logging itself
	if handler := l.logger.Handler(); handler != nil {
		if err := handler.Handle(ctx, record); err != nil {
			// Fallback to stderr if primary logging fails
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to write log: %v - Original message: %s\n", err, msg)
		}
	}
}

// FileLogger creates a logger that writes to files with rotation
type FileLogger struct {
	*TemplarLogger
	file     *os.File
	filePath string
}

// NewFileLogger creates a file-based logger with daily rotation
func NewFileLogger(config *LoggerConfig, logDir string) (*FileLogger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Validate log directory path
	if logDir == "" {
		return nil, fmt.Errorf("log directory cannot be empty")
	}

	// Clean the path to prevent path traversal
	cleanLogDir := filepath.Clean(logDir)
	if strings.Contains(cleanLogDir, "..") {
		return nil, fmt.Errorf("invalid log directory path (contains path traversal): %s", logDir)
	}

	if err := os.MkdirAll(cleanLogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory %s: %w", cleanLogDir, err)
	}

	// Create log file with date
	now := time.Now()
	fileName := fmt.Sprintf("templar-%s.log", now.Format("2006-01-02"))
	filePath := filepath.Join(cleanLogDir, fileName)

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", filePath, err)
	}

	// Update config to use file output
	fileConfig := *config
	fileConfig.Output = file

	logger := NewLogger(&fileConfig)

	return &FileLogger{
		TemplarLogger: logger,
		file:          file,
		filePath:      filePath,
	}, nil
}

// Close closes the file logger
func (f *FileLogger) Close() error {
	if f.file != nil {
		if err := f.file.Close(); err != nil {
			return fmt.Errorf("failed to close log file %s: %w", f.filePath, err)
		}
		f.file = nil // Prevent double-close
	}
	return nil
}

// MultiLogger writes to multiple loggers
type MultiLogger struct {
	loggers []Logger
}

// NewMultiLogger creates a logger that writes to multiple destinations
func NewMultiLogger(loggers ...Logger) *MultiLogger {
	return &MultiLogger{loggers: loggers}
}

// Debug logs to all loggers
func (m *MultiLogger) Debug(ctx context.Context, msg string, fields ...interface{}) {
	for _, logger := range m.loggers {
		logger.Debug(ctx, msg, fields...)
	}
}

// Info logs to all loggers
func (m *MultiLogger) Info(ctx context.Context, msg string, fields ...interface{}) {
	for _, logger := range m.loggers {
		logger.Info(ctx, msg, fields...)
	}
}

// Warn logs to all loggers
func (m *MultiLogger) Warn(ctx context.Context, err error, msg string, fields ...interface{}) {
	for _, logger := range m.loggers {
		logger.Warn(ctx, err, msg, fields...)
	}
}

// Error logs to all loggers
func (m *MultiLogger) Error(ctx context.Context, err error, msg string, fields ...interface{}) {
	for _, logger := range m.loggers {
		logger.Error(ctx, err, msg, fields...)
	}
}

// Fatal logs to all loggers
func (m *MultiLogger) Fatal(ctx context.Context, err error, msg string, fields ...interface{}) {
	for _, logger := range m.loggers {
		logger.Fatal(ctx, err, msg, fields...)
	}
}

// With creates a new multi-logger with additional fields
func (m *MultiLogger) With(fields ...interface{}) Logger {
	newLoggers := make([]Logger, len(m.loggers))
	for i, logger := range m.loggers {
		newLoggers[i] = logger.With(fields...)
	}
	return &MultiLogger{loggers: newLoggers}
}

// WithComponent creates a new multi-logger with component context
func (m *MultiLogger) WithComponent(component string) Logger {
	newLoggers := make([]Logger, len(m.loggers))
	for i, logger := range m.loggers {
		newLoggers[i] = logger.WithComponent(component)
	}
	return &MultiLogger{loggers: newLoggers}
}

// ContextLogger adds request context to logs
type ContextLogger struct {
	Logger
	requestID string
	userID    string
}

// WithRequestID adds request ID to logger context
func (l *TemplarLogger) WithRequestID(requestID string) *ContextLogger {
	return &ContextLogger{
		Logger:    l.With("request_id", requestID),
		requestID: requestID,
	}
}

// WithUserID adds user ID to logger context
func (c *ContextLogger) WithUserID(userID string) *ContextLogger {
	return &ContextLogger{
		Logger: c.Logger.With("user_id", userID),
		userID: userID,
	}
}

// LogFormatter provides custom formatting
type LogFormatter struct {
	TimestampFormat string
	UseColors       bool
}

// FormatLevel formats log level with optional colors
func (f *LogFormatter) FormatLevel(level LogLevel) string {
	if !f.UseColors {
		return level.String()
	}

	switch level {
	case LevelDebug:
		return fmt.Sprintf("\033[36m%s\033[0m", level.String()) // Cyan
	case LevelInfo:
		return fmt.Sprintf("\033[32m%s\033[0m", level.String()) // Green
	case LevelWarn:
		return fmt.Sprintf("\033[33m%s\033[0m", level.String()) // Yellow
	case LevelError:
		return fmt.Sprintf("\033[31m%s\033[0m", level.String()) // Red
	case LevelFatal:
		return fmt.Sprintf("\033[35m%s\033[0m", level.String()) // Magenta
	default:
		return level.String()
	}
}

// Security-focused logging utilities

// SanitizeForLog sanitizes data for safe logging (removes sensitive info)
func SanitizeForLog(data string) string {
	// Remove potential passwords, tokens, etc.
	sensitive := []string{
		"password", "token", "secret", "key", "auth",
	}

	lower := strings.ToLower(data)
	for _, word := range sensitive {
		if strings.Contains(lower, word) {
			return "[REDACTED]"
		}
	}

	// Truncate very long strings
	if len(data) > 1000 {
		return data[:1000] + "...[TRUNCATED]"
	}

	return data
}

// LogSecurityEvent logs security-related events with special handling
func LogSecurityEvent(logger Logger, ctx context.Context, event string, details map[string]interface{}) {
	sanitizedDetails := make(map[string]interface{})
	for k, v := range details {
		if str, ok := v.(string); ok {
			sanitizedDetails[k] = SanitizeForLog(str)
		} else {
			sanitizedDetails[k] = v
		}
	}

	fields := []interface{}{"event_type", "security", "event", event}
	for k, v := range sanitizedDetails {
		fields = append(fields, k, v)
	}

	logger.Error(ctx, nil, "Security event occurred", fields...)
}

// Performance logging utilities

// PerfLogger tracks performance metrics
type PerfLogger struct {
	Logger
	startTime time.Time
	operation string
}

// StartOperation begins performance tracking
func (l *TemplarLogger) StartOperation(operation string) *PerfLogger {
	return &PerfLogger{
		Logger:    l.With("operation", operation),
		startTime: time.Now(),
		operation: operation,
	}
}

// End completes performance tracking and logs the duration
func (p *PerfLogger) End(ctx context.Context) {
	duration := time.Since(p.startTime)
	p.Info(ctx, "Operation completed",
		"duration_ms", duration.Milliseconds(),
		"duration", duration.String(),
	)
}

// EndWithError completes performance tracking and logs an error
func (p *PerfLogger) EndWithError(ctx context.Context, err error) {
	duration := time.Since(p.startTime)
	p.Error(ctx, err, "Operation failed",
		"duration_ms", duration.Milliseconds(),
		"duration", duration.String(),
	)
}

// ErrorCategory represents different types of errors for better categorization
type ErrorCategory string

const (
	ErrorCategorySystem     ErrorCategory = "system"
	ErrorCategoryValidation ErrorCategory = "validation"
	ErrorCategorySecurity   ErrorCategory = "security"
	ErrorCategoryNetwork    ErrorCategory = "network"
	ErrorCategoryFileSystem ErrorCategory = "filesystem"
	ErrorCategoryBuild      ErrorCategory = "build"
	ErrorCategoryComponent  ErrorCategory = "component"
	ErrorCategoryUnknown    ErrorCategory = "unknown"
)

// StructuredError provides enhanced error information for logging
type StructuredError struct {
	Category  ErrorCategory          `json:"category"`
	Operation string                 `json:"operation"`
	Component string                 `json:"component,omitempty"`
	Message   string                 `json:"message"`
	Cause     error                  `json:"cause,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Retryable bool                   `json:"retryable"`
	Severity  string                 `json:"severity"`
}

// Error implements the error interface
func (e *StructuredError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Operation, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Operation, e.Message)
}

// Unwrap returns the underlying error for error unwrapping
func (e *StructuredError) Unwrap() error {
	return e.Cause
}

// NewStructuredError creates a new structured error
func NewStructuredError(category ErrorCategory, operation, message string) *StructuredError {
	return &StructuredError{
		Category:  category,
		Operation: operation,
		Message:   message,
		Timestamp: time.Now(),
		Severity:  "error",
		Context:   make(map[string]interface{}),
	}
}

// WithCause adds a cause error
func (e *StructuredError) WithCause(cause error) *StructuredError {
	e.Cause = cause
	return e
}

// WithComponent adds component context
func (e *StructuredError) WithComponent(component string) *StructuredError {
	e.Component = component
	return e
}

// WithContext adds additional context
func (e *StructuredError) WithContext(key string, value interface{}) *StructuredError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithRetryable marks the error as retryable
func (e *StructuredError) WithRetryable(retryable bool) *StructuredError {
	e.Retryable = retryable
	return e
}

// WithSeverity sets the error severity
func (e *StructuredError) WithSeverity(severity string) *StructuredError {
	e.Severity = severity
	return e
}

// ResilientLogger provides retry capabilities for logging operations
type ResilientLogger struct {
	Logger
	maxRetries int
	retryDelay time.Duration
	mutex      sync.RWMutex
}

// NewResilientLogger creates a logger with retry capabilities
func NewResilientLogger(logger Logger, maxRetries int, retryDelay time.Duration) *ResilientLogger {
	return &ResilientLogger{
		Logger:     logger,
		maxRetries: maxRetries,
		retryDelay: retryDelay,
	}
}

// ErrorWithRetry logs an error with retry mechanism
func (r *ResilientLogger) ErrorWithRetry(ctx context.Context, err error, msg string, fields ...interface{}) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		if attempt > 0 {
			// Add retry context
			fields = append(fields, "retry_attempt", attempt)
			time.Sleep(r.retryDelay)
		}

		// Attempt to log
		success := false
		func() {
			defer func() {
				if recover() != nil {
					// Log attempt failed, will retry if possible
					success = false
				} else {
					success = true
				}
			}()
			r.Logger.Error(ctx, err, msg, fields...)
		}()

		// If logging succeeded, return
		if success {
			return
		}
	}

	// All retry attempts failed, use fallback
	fmt.Fprintf(os.Stderr, "[CRITICAL] Failed to log error after %d retries: %s - %v\n", r.maxRetries, msg, err)
}

// LogStructuredError logs a structured error with enhanced context
func LogStructuredError(logger Logger, ctx context.Context, structErr *StructuredError) {
	fields := []interface{}{
		"error_category", string(structErr.Category),
		"operation", structErr.Operation,
		"severity", structErr.Severity,
		"retryable", structErr.Retryable,
		"timestamp", structErr.Timestamp,
	}

	if structErr.Component != "" {
		fields = append(fields, "component", structErr.Component)
	}

	// Add context fields
	for k, v := range structErr.Context {
		fields = append(fields, k, v)
	}

	logger.Error(ctx, structErr.Cause, structErr.Message, fields...)
}

// NewTestLogger creates a logger for testing purposes
func NewTestLogger() Logger {
	config := &LoggerConfig{
		Level:      LevelDebug,
		Format:     "text",
		Output:     io.Discard, // Discard output in tests to avoid noise
		TimeFormat: time.RFC3339,
		AddSource:  false, // Disable source for cleaner test output
	}
	return NewLogger(config)
}
