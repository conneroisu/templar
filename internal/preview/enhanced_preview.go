package preview

import (
	"context"
	"fmt"
	"html/template"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/logging"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/renderer"
)

// EnhancedPreviewSystem provides advanced component preview capabilities
type EnhancedPreviewSystem struct {
	registry *registry.ComponentRegistry
	renderer *renderer.ComponentRenderer
	logger   logging.Logger

	// Preview configuration
	config *PreviewConfig

	// Template and asset management
	templateManager *TemplateManager
	assetManager    *AssetManager

	// Live reload and hot reloading
	liveReload *LiveReloadManager

	// Component isolation and sandboxing
	sandboxManager *SandboxManager

	// Preview sessions and state management
	sessionManager *SessionManager

	// Performance monitoring
	performanceMonitor *PreviewPerformanceMonitor
}

// PreviewConfig holds configuration for the preview system
type PreviewConfig struct {
	// Preview server settings
	Host     string
	Port     int
	BasePath string

	// Template settings
	TemplateDir     string
	AssetsDir       string
	StaticAssetsDir string

	// Live reload settings
	EnableLiveReload bool
	LiveReloadPort   int
	HotReloadEnabled bool

	// Sandbox settings
	EnableSandboxing bool
	AllowedOrigins   []string
	CSPPolicy        string

	// Performance settings
	EnableCaching        bool
	CacheTimeout         time.Duration
	MaxConcurrentRenders int

	// Development features
	ShowPerformanceMetrics bool
	EnableDebugMode        bool
	ShowComponentTree      bool
	EnableMockData         bool
}

// TemplateManager handles preview templates and layouts
type TemplateManager struct {
	templates        map[string]*template.Template
	layoutTemplates  map[string]string
	partialTemplates map[string]string
}

// AssetManager handles static assets and bundling
type AssetManager struct {
	assets map[string]*Asset
}

// Asset represents a static asset
type Asset struct {
	Path         string
	Content      []byte
	ContentType  string
	Hash         string
	Size         int64
	LastModified time.Time
	Compressed   bool
}

// LiveReloadManager handles live reload functionality
type LiveReloadManager struct {
	connections map[string]*LiveReloadConnection
	broadcastCh chan LiveReloadEvent
}

// LiveReloadConnection represents a live reload WebSocket connection
type LiveReloadConnection struct {
	ID            string
	SessionID     string
	Connection    interface{} // WebSocket connection
	LastPing      time.Time
	Subscriptions []string
}

// LiveReloadEvent represents a live reload event
type LiveReloadEvent struct {
	Type      string                 `json:"type"`
	Target    string                 `json:"target,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// SandboxManager provides component isolation and security
type SandboxManager struct {
	sandboxes    map[string]*ComponentSandbox
	sandboxMutex sync.RWMutex

	// Security policies
	cspPolicies map[string]string

	// Resource limits
	resourceLimits *ResourceLimits
}

// ComponentSandbox isolates component execution
type ComponentSandbox struct {
	ID             string
	ComponentName  string
	IsolationLevel int
	AllowedAPIs    []string
	ResourceLimits *ResourceLimits

	// Execution context
	context context.Context
	cancel  context.CancelFunc

	// Security restrictions
	allowedDomains []string
	blockedURLs    []string
}

// ResourceLimits defines resource constraints for sandboxed components
type ResourceLimits struct {
	MaxMemoryMB      int
	MaxCPUPercent    float64
	MaxExecutionTime time.Duration
	MaxFileSize      int64
	MaxNetworkCalls  int
}

// SessionManager manages preview sessions and state
type SessionManager struct {
	sessions     map[string]*PreviewSession
	sessionMutex sync.RWMutex

	// Session configuration
	sessionTimeout time.Duration
	maxSessions    int
}

// PreviewSession represents a user's preview session
type PreviewSession struct {
	ID           string
	UserID       string
	CreatedAt    time.Time
	LastActivity time.Time

	// Session state
	CurrentComponent string
	ComponentProps   map[string]interface{}
	CustomCSS        string
	CustomJS         string

	// User preferences
	Theme        string
	ViewportSize ViewportSize
	DeviceMode   string

	// History and navigation
	History   []PreviewHistoryEntry
	Bookmarks []ComponentBookmark
}

// ViewportSize represents viewport dimensions
type ViewportSize struct {
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Scale  float64 `json:"scale"`
}

// PreviewHistoryEntry tracks component preview history
type PreviewHistoryEntry struct {
	ComponentName string                 `json:"component_name"`
	Props         map[string]interface{} `json:"props"`
	Timestamp     time.Time              `json:"timestamp"`
	Title         string                 `json:"title"`
}

// ComponentBookmark allows users to save component configurations
type ComponentBookmark struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	ComponentName string                 `json:"component_name"`
	Props         map[string]interface{} `json:"props"`
	Description   string                 `json:"description"`
	CreatedAt     time.Time              `json:"created_at"`
}

// PreviewPerformanceMonitor tracks preview system performance
type PreviewPerformanceMonitor struct {
	metrics      *PreviewMetrics
	metricsMutex sync.RWMutex

	// Performance tracking
	renderTimes []time.Duration
	errorCounts map[string]int

	// Resource utilization
	requestCounts map[string]int64
}

// PreviewMetrics contains performance metrics
type PreviewMetrics struct {
	TotalRenders      int64         `json:"total_renders"`
	SuccessfulRenders int64         `json:"successful_renders"`
	FailedRenders     int64         `json:"failed_renders"`
	AverageRenderTime time.Duration `json:"average_render_time"`
	ActiveSessions    int           `json:"active_sessions"`
	ActiveConnections int           `json:"active_connections"`
	CacheHitRate      float64       `json:"cache_hit_rate"`
	ErrorRate         float64       `json:"error_rate"`
	LastUpdated       time.Time     `json:"last_updated"`
}

// NewEnhancedPreviewSystem creates a new enhanced preview system
func NewEnhancedPreviewSystem(registry *registry.ComponentRegistry, renderer *renderer.ComponentRenderer, logger logging.Logger) *EnhancedPreviewSystem {
	config := DefaultPreviewConfig()

	templateManager := NewTemplateManager(config)
	assetManager := NewAssetManager(config)
	liveReload := NewLiveReloadManager(config)
	sandboxManager := NewSandboxManager(config)
	sessionManager := NewSessionManager(config)
	performanceMonitor := NewPreviewPerformanceMonitor()

	return &EnhancedPreviewSystem{
		registry:           registry,
		renderer:           renderer,
		logger:             logger,
		config:             config,
		templateManager:    templateManager,
		assetManager:       assetManager,
		liveReload:         liveReload,
		sandboxManager:     sandboxManager,
		sessionManager:     sessionManager,
		performanceMonitor: performanceMonitor,
	}
}

// DefaultPreviewConfig returns default configuration
func DefaultPreviewConfig() *PreviewConfig {
	return &PreviewConfig{
		Host:                   "localhost",
		Port:                   8080,
		BasePath:               "/preview",
		TemplateDir:            "./templates",
		AssetsDir:              "./assets",
		StaticAssetsDir:        "./static",
		EnableLiveReload:       true,
		LiveReloadPort:         8081,
		HotReloadEnabled:       true,
		EnableSandboxing:       true,
		AllowedOrigins:         []string{"http://localhost:8080"},
		CSPPolicy:              "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'",
		EnableCaching:          true,
		CacheTimeout:           5 * time.Minute,
		MaxConcurrentRenders:   10,
		ShowPerformanceMetrics: true,
		EnableDebugMode:        true,
		ShowComponentTree:      true,
		EnableMockData:         true,
	}
}

// Start starts the enhanced preview system
func (eps *EnhancedPreviewSystem) Start(ctx context.Context) error {
	// Start all subsystems
	if err := eps.templateManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start template manager: %w", err)
	}

	if err := eps.assetManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start asset manager: %w", err)
	}

	if eps.config.EnableLiveReload {
		if err := eps.liveReload.Start(ctx); err != nil {
			return fmt.Errorf("failed to start live reload: %w", err)
		}
	}

	if err := eps.sandboxManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start sandbox manager: %w", err)
	}

	if err := eps.sessionManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start session manager: %w", err)
	}

	// Start performance monitoring
	go eps.performanceMonitor.Start(ctx)

	if eps.logger != nil {
		eps.logger.Info(ctx, "Enhanced preview system started",
			"host", eps.config.Host,
			"port", eps.config.Port,
			"live_reload", eps.config.EnableLiveReload,
			"sandboxing", eps.config.EnableSandboxing)
	}

	return nil
}

// PreviewComponent renders a component preview with enhanced features
func (eps *EnhancedPreviewSystem) PreviewComponent(ctx context.Context, componentName string, props map[string]interface{}, options *PreviewOptions) (*PreviewResult, error) {
	startTime := time.Now()
	defer func() {
		eps.performanceMonitor.RecordRender(time.Since(startTime), nil)
	}()

	// Get or create session
	session := eps.sessionManager.GetOrCreateSession(ctx, options.SessionID)

	// Create component sandbox if enabled
	var sandbox *ComponentSandbox
	if eps.config.EnableSandboxing {
		var err error
		sandbox, err = eps.sandboxManager.CreateSandbox(componentName, options.IsolationLevel)
		if err != nil {
			return nil, fmt.Errorf("failed to create sandbox: %w", err)
		}
		defer eps.sandboxManager.DestroySandbox(sandbox.ID)
	}

	// Render component with enhanced features
	result, err := eps.renderComponentEnhanced(ctx, componentName, props, options, session, sandbox)
	if err != nil {
		eps.performanceMonitor.RecordRender(time.Since(startTime), err)
		return nil, err
	}

	// Update session state
	session.CurrentComponent = componentName
	session.ComponentProps = props
	session.LastActivity = time.Now()

	// Add to history
	historyEntry := PreviewHistoryEntry{
		ComponentName: componentName,
		Props:         props,
		Timestamp:     time.Now(),
		Title:         fmt.Sprintf("%s Preview", componentName),
	}
	session.History = append(session.History, historyEntry)

	// Broadcast live reload event if enabled
	if eps.config.EnableLiveReload {
		event := LiveReloadEvent{
			Type:      "component_rendered",
			Target:    componentName,
			Data:      map[string]interface{}{"props": props},
			Timestamp: time.Now(),
		}
		eps.liveReload.Broadcast(event)
	}

	return result, nil
}

// PreviewOptions contains options for component preview
type PreviewOptions struct {
	SessionID      string
	Theme          string
	ViewportSize   *ViewportSize
	DeviceMode     string
	IsolationLevel int
	MockData       bool
	ShowDebugInfo  bool
	CustomCSS      string
	CustomJS       string
	Layout         string
}

// PreviewResult contains the result of a component preview
type PreviewResult struct {
	HTML             string             `json:"html"`
	CSS              string             `json:"css"`
	JavaScript       string             `json:"javascript"`
	Metadata         *PreviewMetadata   `json:"metadata"`
	Performance      *RenderPerformance `json:"performance,omitempty"`
	DebugInfo        *DebugInfo         `json:"debug_info,omitempty"`
	LiveReloadScript string             `json:"live_reload_script,omitempty"`
}

// PreviewMetadata contains metadata about the preview
type PreviewMetadata struct {
	ComponentName string                 `json:"component_name"`
	Props         map[string]interface{} `json:"props"`
	Dependencies  []string               `json:"dependencies"`
	Theme         string                 `json:"theme"`
	ViewportSize  *ViewportSize          `json:"viewport_size"`
	GeneratedAt   time.Time              `json:"generated_at"`
	CacheKey      string                 `json:"cache_key"`
	Version       string                 `json:"version"`
}

// RenderPerformance contains performance metrics for the render
type RenderPerformance struct {
	RenderTime    time.Duration `json:"render_time"`
	TemplateTime  time.Duration `json:"template_time"`
	AssetLoadTime time.Duration `json:"asset_load_time"`
	CacheHit      bool          `json:"cache_hit"`
	MemoryUsed    int64         `json:"memory_used"`
}

// DebugInfo contains debugging information
type DebugInfo struct {
	ComponentTree  *ComponentTreeNode     `json:"component_tree"`
	PropValidation []ValidationError      `json:"prop_validation"`
	RenderSteps    []RenderStep           `json:"render_steps"`
	AssetManifest  map[string]interface{} `json:"asset_manifest"`
}

// ComponentTreeNode represents a node in the component tree
type ComponentTreeNode struct {
	Name       string                 `json:"name"`
	Props      map[string]interface{} `json:"props"`
	Children   []*ComponentTreeNode   `json:"children"`
	RenderTime time.Duration          `json:"render_time"`
	MemoryUsed int64                  `json:"memory_used"`
}

// ValidationError represents a prop validation error
type ValidationError struct {
	Property string `json:"property"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// RenderStep represents a step in the rendering process
type RenderStep struct {
	Step        string        `json:"step"`
	Duration    time.Duration `json:"duration"`
	Description string        `json:"description"`
	Data        interface{}   `json:"data,omitempty"`
}

// Placeholder implementations for managers
func NewTemplateManager(config *PreviewConfig) *TemplateManager {
	return &TemplateManager{
		templates:        make(map[string]*template.Template),
		layoutTemplates:  make(map[string]string),
		partialTemplates: make(map[string]string),
	}
}

func NewAssetManager(config *PreviewConfig) *AssetManager {
	return &AssetManager{
		assets: make(map[string]*Asset),
	}
}

func NewLiveReloadManager(config *PreviewConfig) *LiveReloadManager {
	return &LiveReloadManager{
		connections: make(map[string]*LiveReloadConnection),
		broadcastCh: make(chan LiveReloadEvent, 100),
	}
}

func NewSandboxManager(config *PreviewConfig) *SandboxManager {
	return &SandboxManager{
		sandboxes:   make(map[string]*ComponentSandbox),
		cspPolicies: make(map[string]string),
		resourceLimits: &ResourceLimits{
			MaxMemoryMB:      100,
			MaxCPUPercent:    50.0,
			MaxExecutionTime: 30 * time.Second,
			MaxFileSize:      10 * 1024 * 1024, // 10MB
			MaxNetworkCalls:  10,
		},
	}
}

func NewSessionManager(config *PreviewConfig) *SessionManager {
	return &SessionManager{
		sessions:       make(map[string]*PreviewSession),
		sessionTimeout: 1 * time.Hour,
		maxSessions:    1000,
	}
}

func NewPreviewPerformanceMonitor() *PreviewPerformanceMonitor {
	return &PreviewPerformanceMonitor{
		metrics:       &PreviewMetrics{},
		renderTimes:   make([]time.Duration, 0, 1000),
		errorCounts:   make(map[string]int),
		requestCounts: make(map[string]int64),
	}
}

// Manager start methods (placeholder implementations)
func (tm *TemplateManager) Start(ctx context.Context) error {
	// Load and compile templates
	return nil
}

func (am *AssetManager) Start(ctx context.Context) error {
	// Initialize asset bundling and optimization
	return nil
}

func (lrm *LiveReloadManager) Start(ctx context.Context) error {
	// Start WebSocket server and file watcher
	return nil
}

func (sm *SandboxManager) Start(ctx context.Context) error {
	// Initialize sandbox environment
	return nil
}

func (sesm *SessionManager) Start(ctx context.Context) error {
	// Start session cleanup and management
	return nil
}

func (ppm *PreviewPerformanceMonitor) Start(ctx context.Context) {
	// Start performance monitoring
}

// Additional methods for core functionality
func (eps *EnhancedPreviewSystem) renderComponentEnhanced(ctx context.Context, componentName string, props map[string]interface{}, options *PreviewOptions, session *PreviewSession, sandbox *ComponentSandbox) (*PreviewResult, error) {
	// Enhanced rendering implementation
	return &PreviewResult{
		HTML:        "<div>Enhanced preview placeholder</div>",
		CSS:         "",
		JavaScript:  "",
		Metadata:    &PreviewMetadata{ComponentName: componentName, Props: props, GeneratedAt: time.Now()},
		Performance: &RenderPerformance{RenderTime: time.Millisecond * 10},
	}, nil
}

func (sesm *SessionManager) GetOrCreateSession(ctx context.Context, sessionID string) *PreviewSession {
	sesm.sessionMutex.Lock()
	defer sesm.sessionMutex.Unlock()

	if sessionID == "" {
		sessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
	}

	if session, exists := sesm.sessions[sessionID]; exists {
		return session
	}

	session := &PreviewSession{
		ID:             sessionID,
		CreatedAt:      time.Now(),
		LastActivity:   time.Now(),
		ComponentProps: make(map[string]interface{}),
		ViewportSize:   ViewportSize{Width: 1200, Height: 800, Scale: 1.0},
		History:        make([]PreviewHistoryEntry, 0),
		Bookmarks:      make([]ComponentBookmark, 0),
	}

	sesm.sessions[sessionID] = session
	return session
}

func (sm *SandboxManager) CreateSandbox(componentName string, isolationLevel int) (*ComponentSandbox, error) {
	sm.sandboxMutex.Lock()
	defer sm.sandboxMutex.Unlock()

	sandboxID := fmt.Sprintf("sandbox_%s_%d", componentName, time.Now().UnixNano())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	sandbox := &ComponentSandbox{
		ID:             sandboxID,
		ComponentName:  componentName,
		IsolationLevel: isolationLevel,
		ResourceLimits: sm.resourceLimits,
		context:        ctx,
		cancel:         cancel,
		allowedDomains: []string{"localhost"},
		blockedURLs:    []string{},
	}

	sm.sandboxes[sandboxID] = sandbox
	return sandbox, nil
}

func (sm *SandboxManager) DestroySandbox(sandboxID string) {
	sm.sandboxMutex.Lock()
	defer sm.sandboxMutex.Unlock()

	if sandbox, exists := sm.sandboxes[sandboxID]; exists {
		sandbox.cancel()
		delete(sm.sandboxes, sandboxID)
	}
}

func (lrm *LiveReloadManager) Broadcast(event LiveReloadEvent) {
	select {
	case lrm.broadcastCh <- event:
	default:
		// Channel full, drop event
	}
}

func (ppm *PreviewPerformanceMonitor) RecordRender(duration time.Duration, err error) {
	ppm.metricsMutex.Lock()
	defer ppm.metricsMutex.Unlock()

	ppm.metrics.TotalRenders++
	if err != nil {
		ppm.metrics.FailedRenders++
	} else {
		ppm.metrics.SuccessfulRenders++
	}

	ppm.renderTimes = append(ppm.renderTimes, duration)
	if len(ppm.renderTimes) > 1000 {
		ppm.renderTimes = ppm.renderTimes[1:] // Keep last 1000 entries
	}

	// Calculate average
	var total time.Duration
	for _, t := range ppm.renderTimes {
		total += t
	}
	ppm.metrics.AverageRenderTime = total / time.Duration(len(ppm.renderTimes))
	ppm.metrics.LastUpdated = time.Now()
}

// GetPreviewMetrics returns current preview system metrics
func (eps *EnhancedPreviewSystem) GetPreviewMetrics() *PreviewMetrics {
	eps.performanceMonitor.metricsMutex.RLock()
	defer eps.performanceMonitor.metricsMutex.RUnlock()

	// Copy metrics to avoid race conditions
	metrics := *eps.performanceMonitor.metrics
	metrics.ActiveSessions = len(eps.sessionManager.sessions)
	metrics.ActiveConnections = len(eps.liveReload.connections)

	if metrics.TotalRenders > 0 {
		metrics.ErrorRate = float64(metrics.FailedRenders) / float64(metrics.TotalRenders)
	}

	return &metrics
}

// Interface definitions for external dependencies
type SessionStorage interface {
	Store(sessionID string, session *PreviewSession) error
	Load(sessionID string) (*PreviewSession, error)
	Delete(sessionID string) error
}

type TemplateWatcher interface {
	Watch(templatePath string) error
	Stop() error
}

type AssetBundler interface {
	Bundle(assets []string) (*Asset, error)
}

type AssetOptimizer interface {
	Optimize(asset *Asset) (*Asset, error)
}

type CDNConfig struct {
	Enabled bool
	BaseURL string
	APIKey  string
}

type AssetCacheManager interface {
	Get(key string) (*Asset, bool)
	Set(key string, asset *Asset) error
	Clear() error
}

type FileWatcher interface {
	Watch(path string) error
	Stop() error
}

type WebSocketServer interface {
	Start(port int) error
	Stop() error
	Broadcast(data []byte) error
}
