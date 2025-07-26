package interfaces

import (
	"context"
	"io"
	"time"
)

// Logger defines the interface for application logging
//
// All logging implementations should provide structured logging with
// multiple severity levels and contextual information.
//
// Thread Safety:
//
//	All methods must be thread-safe for concurrent access.
//
// Performance:
//
//	Log methods should be non-blocking and high-performance.
//	Consider async logging for high-throughput scenarios.
type Logger interface {
	// Debug logs debug-level messages (development only)
	Debug(msg string, fields ...LogField)

	// Info logs informational messages
	Info(msg string, fields ...LogField)

	// Warn logs warning messages
	Warn(msg string, fields ...LogField)

	// Error logs error messages
	Error(msg string, err error, fields ...LogField)

	// Fatal logs fatal messages and terminates the application
	Fatal(msg string, err error, fields ...LogField)

	// WithContext returns a logger with context for request tracing
	WithContext(ctx context.Context) Logger

	// WithFields returns a logger with pre-configured fields
	WithFields(fields ...LogField) Logger

	// SetLevel sets the minimum logging level
	SetLevel(level LogLevel)

	// GetLevel returns the current logging level
	GetLevel() LogLevel
}

// MetricsCollector defines the interface for application metrics
//
// Metrics provide insight into application performance, health, and usage.
// All metrics should be efficiently collected and queryable.
//
// Performance:
//
//	Metric recording should have minimal overhead (< 1µs per metric)
//	Consider using atomic operations for high-frequency counters.
type MetricsCollector interface {
	// Counter metrics (monotonically increasing)
	IncrementCounter(name string, tags ...MetricTag)
	AddToCounter(name string, value float64, tags ...MetricTag)

	// Gauge metrics (current value)
	SetGauge(name string, value float64, tags ...MetricTag)

	// Histogram metrics (distribution of values)
	RecordHistogram(name string, value float64, tags ...MetricTag)

	// Timing metrics (duration measurements)
	RecordTiming(name string, duration time.Duration, tags ...MetricTag)
	StartTimer(name string, tags ...MetricTag) Timer

	// Metric queries
	GetCounter(name string, tags ...MetricTag) float64
	GetGauge(name string, tags ...MetricTag) float64
	GetHistogramStats(name string, tags ...MetricTag) HistogramStats

	// Metric management
	ListMetrics() []MetricInfo
	ResetMetrics()
	ExportMetrics(format MetricFormat) ([]byte, error)
}

// Timer represents an active timing measurement
type Timer interface {
	// Stop stops the timer and records the duration
	Stop()

	// Elapsed returns the elapsed time without stopping
	Elapsed() time.Duration
}

// HealthChecker defines the interface for application health monitoring
//
// Health checks provide insight into the application's operational status
// and dependencies. All checks should complete quickly and reliably.
//
// Performance:
//
//	Health checks should complete within 5 seconds
//	Critical checks should complete within 1 second
type HealthChecker interface {
	// Check performs a health check and returns the result
	Check(ctx context.Context) HealthResult

	// CheckWithName performs a named health check
	CheckWithName(ctx context.Context, name string) HealthResult

	// RegisterCheck registers a custom health check
	RegisterCheck(name string, check HealthCheckFunc)

	// UnregisterCheck removes a health check
	UnregisterCheck(name string)

	// GetChecks returns all registered health checks
	GetChecks() []string

	// IsHealthy returns true if all critical checks pass
	IsHealthy(ctx context.Context) bool
}

// NotificationService defines the interface for sending notifications
//
// Notifications can be sent through various channels (email, SMS, webhook)
// with appropriate prioritization and delivery guarantees.
type NotificationService interface {
	// SendEmail sends an email notification
	SendEmail(ctx context.Context, msg EmailMessage) error

	// SendSMS sends an SMS notification
	SendSMS(ctx context.Context, msg SMSMessage) error

	// SendWebhook sends a webhook notification
	SendWebhook(ctx context.Context, msg WebhookMessage) error

	// SendBatch sends multiple notifications in batch
	SendBatch(ctx context.Context, notifications []Notification) error

	// GetDeliveryStatus returns delivery status for a notification
	GetDeliveryStatus(notificationID string) (DeliveryStatus, error)

	// RegisterTemplate registers a notification template
	RegisterTemplate(name string, template NotificationTemplate) error

	// SendWithTemplate sends a notification using a template
	SendWithTemplate(
		ctx context.Context,
		templateName string,
		data map[string]interface{},
		recipient string,
	) error
}

// CacheService defines the interface for application caching
//
// The cache provides high-performance storage for frequently accessed data
// with configurable expiration and eviction policies.
//
// Performance:
//
//	Get operations should complete in < 1ms for in-memory caches
//	Set operations should be async when possible
type CacheService interface {
	// Get retrieves a value from the cache
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value in the cache with expiration
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error

	// Delete removes a value from the cache
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists in the cache
	Exists(ctx context.Context, key string) (bool, error)

	// Clear removes all values from the cache
	Clear(ctx context.Context) error

	// GetStats returns cache performance statistics
	GetStats(ctx context.Context) CacheServiceStats

	// GetMulti retrieves multiple values in a single operation
	GetMulti(ctx context.Context, keys []string) (map[string][]byte, error)

	// SetMulti stores multiple values in a single operation
	SetMulti(ctx context.Context, items map[string][]byte, expiration time.Duration) error
}

// FileService defines the interface for file operations
//
// File operations provide safe, atomic file handling with proper
// error handling and cleanup.
type FileService interface {
	// Read reads the entire file contents
	Read(ctx context.Context, path string) ([]byte, error)

	// Write writes data to a file atomically
	Write(ctx context.Context, path string, data []byte) error

	// Append appends data to a file
	Append(ctx context.Context, path string, data []byte) error

	// Copy copies a file from source to destination
	Copy(ctx context.Context, src, dst string) error

	// Move moves a file from source to destination
	Move(ctx context.Context, src, dst string) error

	// Delete removes a file
	Delete(ctx context.Context, path string) error

	// Exists checks if a file exists
	Exists(ctx context.Context, path string) (bool, error)

	// Stat returns file information
	Stat(ctx context.Context, path string) (FileInfo, error)

	// List returns files in a directory
	List(ctx context.Context, dir string) ([]FileInfo, error)

	// Watch monitors file changes
	Watch(ctx context.Context, path string) (<-chan FileEvent, error)

	// CreateTemp creates a temporary file
	CreateTemp(ctx context.Context, pattern string) (TempFile, error)
}

// SecurityService defines the interface for security operations
//
// Security operations include authentication, authorization, encryption,
// and audit logging with strong security guarantees.
type SecurityService interface {
	// Authenticate verifies user credentials
	Authenticate(ctx context.Context, credentials Credentials) (AuthResult, error)

	// Authorize checks if a user has permission for an action
	Authorize(ctx context.Context, user User, resource string, action string) (bool, error)

	// Encrypt encrypts data with the default key
	Encrypt(ctx context.Context, data []byte) ([]byte, error)

	// Decrypt decrypts data with the default key
	Decrypt(ctx context.Context, data []byte) ([]byte, error)

	// Hash generates a secure hash of data
	Hash(data []byte) string

	// VerifyHash verifies data against a hash
	VerifyHash(data []byte, hash string) bool

	// GenerateToken generates a secure token
	GenerateToken(ctx context.Context, claims map[string]interface{}) (string, error)

	// VerifyToken verifies and parses a token
	VerifyToken(ctx context.Context, token string) (map[string]interface{}, error)

	// AuditLog records a security event
	AuditLog(ctx context.Context, event SecurityEvent)
}

// Type definitions for service interfaces

// LogField represents a structured log field
type LogField struct {
	Key   string
	Value interface{}
}

// LogLevel represents logging severity levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// MetricTag represents a metric tag for labeling
type MetricTag struct {
	Key   string
	Value string
}

// MetricInfo represents metadata about a metric
type MetricInfo struct {
	Name        string
	Type        string
	Description string
	Tags        []MetricTag
}

// MetricFormat represents metric export formats
type MetricFormat int

const (
	MetricFormatPrometheus MetricFormat = iota
	MetricFormatJSON
	MetricFormatInfluxDB
)

// HistogramStats represents statistics from a histogram metric
type HistogramStats struct {
	Count      int64
	Sum        float64
	Min        float64
	Max        float64
	Mean       float64
	Percentile map[int]float64 // e.g., 50, 95, 99
}

// HealthResult represents the result of a health check
type HealthResult struct {
	Status    HealthStatus
	Details   map[string]interface{}
	Error     error
	Duration  time.Duration
	Timestamp time.Time
}

// HealthStatus represents health check status
type HealthStatus int

const (
	HealthStatusHealthy HealthStatus = iota
	HealthStatusUnhealthy
	HealthStatusDegraded
	HealthStatusUnknown
)

// HealthCheckFunc is a function that performs a health check
type HealthCheckFunc func(ctx context.Context) HealthResult

// EmailMessage represents an email notification
type EmailMessage struct {
	To      []string
	CC      []string
	BCC     []string
	Subject string
	Body    string
	IsHTML  bool
	Headers map[string]string
}

// SMSMessage represents an SMS notification
type SMSMessage struct {
	To   string
	Body string
}

// WebhookMessage represents a webhook notification
type WebhookMessage struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    []byte
	Timeout time.Duration
}

// Notification represents a generic notification
type Notification struct {
	ID       string
	Type     NotificationType
	Priority NotificationPriority
	Message  interface{}
}

// NotificationType represents the type of notification
type NotificationType int

const (
	NotificationTypeEmail NotificationType = iota
	NotificationTypeSMS
	NotificationTypeWebhook
)

// NotificationPriority represents notification priority
type NotificationPriority int

const (
	NotificationPriorityLow NotificationPriority = iota
	NotificationPriorityNormal
	NotificationPriorityHigh
	NotificationPriorityCritical
)

// DeliveryStatus represents notification delivery status
type DeliveryStatus struct {
	Status    DeliveryState
	Timestamp time.Time
	Error     error
	Attempts  int
}

// DeliveryState represents the state of notification delivery
type DeliveryState int

const (
	DeliveryStatePending DeliveryState = iota
	DeliveryStateDelivered
	DeliveryStateFailed
	DeliveryStateRetrying
)

// NotificationTemplate represents a notification template
type NotificationTemplate struct {
	Name      string
	Type      NotificationType
	Subject   string
	Body      string
	Variables []string
	IsHTML    bool
}

// CacheServiceStats represents cache performance statistics
type CacheServiceStats struct {
	Hits      int64
	Misses    int64
	Sets      int64
	Deletes   int64
	Evictions int64
	Size      int64
	MaxSize   int64
	HitRate   float64
	Memory    int64
	MaxMemory int64
}

// FileInfo represents file metadata
type FileInfo struct {
	Name    string
	Size    int64
	Mode    int32
	ModTime time.Time
	IsDir   bool
}

// FileEvent represents a file system event
type FileEvent struct {
	Path      string
	Operation FileOperation
	Timestamp time.Time
}

// TempFile represents a temporary file
type TempFile interface {
	io.ReadWriteCloser
	Name() string
	Remove() error
}

// Credentials represents authentication credentials
type Credentials struct {
	Username string
	Password string
	Token    string
	Type     CredentialType
}

// CredentialType represents the type of credentials
type CredentialType int

const (
	CredentialTypePassword CredentialType = iota
	CredentialTypeToken
	CredentialTypeAPIKey
	CredentialTypeCertificate
)

// AuthResult represents authentication result
type AuthResult struct {
	Success   bool
	User      User
	Token     string
	ExpiresAt time.Time
	Error     error
}

// User represents an authenticated user
type User struct {
	ID       string
	Username string
	Email    string
	Roles    []string
	Groups   []string
	Metadata map[string]interface{}
}

// SecurityEvent represents a security audit event
type SecurityEvent struct {
	Type      SecurityEventType
	User      string
	Resource  string
	Action    string
	Success   bool
	IP        string
	UserAgent string
	Timestamp time.Time
	Details   map[string]interface{}
}

// SecurityEventType represents the type of security event
type SecurityEventType int

const (
	SecurityEventTypeLogin SecurityEventType = iota
	SecurityEventTypeLogout
	SecurityEventTypeAccess
	SecurityEventTypePermissionDenied
	SecurityEventTypePasswordChange
	SecurityEventTypeTokenGenerated
	SecurityEventTypeTokenRevoked
)

// String methods for enum types

func (ll LogLevel) String() string {
	switch ll {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

func (hs HealthStatus) String() string {
	switch hs {
	case HealthStatusHealthy:
		return "HEALTHY"
	case HealthStatusUnhealthy:
		return "UNHEALTHY"
	case HealthStatusDegraded:
		return "DEGRADED"
	case HealthStatusUnknown:
		return "UNKNOWN"
	default:
		return "UNKNOWN"
	}
}

func (mf MetricFormat) String() string {
	switch mf {
	case MetricFormatPrometheus:
		return "prometheus"
	case MetricFormatJSON:
		return "json"
	case MetricFormatInfluxDB:
		return "influxdb"
	default:
		return "unknown"
	}
}

func (nt NotificationType) String() string {
	switch nt {
	case NotificationTypeEmail:
		return "email"
	case NotificationTypeSMS:
		return "sms"
	case NotificationTypeWebhook:
		return "webhook"
	default:
		return "unknown"
	}
}

func (np NotificationPriority) String() string {
	switch np {
	case NotificationPriorityLow:
		return "low"
	case NotificationPriorityNormal:
		return "normal"
	case NotificationPriorityHigh:
		return "high"
	case NotificationPriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

func (ds DeliveryState) String() string {
	switch ds {
	case DeliveryStatePending:
		return "pending"
	case DeliveryStateDelivered:
		return "delivered"
	case DeliveryStateFailed:
		return "failed"
	case DeliveryStateRetrying:
		return "retrying"
	default:
		return "unknown"
	}
}

func (ct CredentialType) String() string {
	switch ct {
	case CredentialTypePassword:
		return "password"
	case CredentialTypeToken:
		return "token"
	case CredentialTypeAPIKey:
		return "api_key"
	case CredentialTypeCertificate:
		return "certificate"
	default:
		return "unknown"
	}
}

func (set SecurityEventType) String() string {
	switch set {
	case SecurityEventTypeLogin:
		return "login"
	case SecurityEventTypeLogout:
		return "logout"
	case SecurityEventTypeAccess:
		return "access"
	case SecurityEventTypePermissionDenied:
		return "permission_denied"
	case SecurityEventTypePasswordChange:
		return "password_change"
	case SecurityEventTypeTokenGenerated:
		return "token_generated"
	case SecurityEventTypeTokenRevoked:
		return "token_revoked"
	default:
		return "unknown"
	}
}
