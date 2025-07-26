package preview

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPreviewConfig(t *testing.T) {
	config := DefaultPreviewConfig()

	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 8080, config.Port)
	assert.Equal(t, "/preview", config.BasePath)
	assert.True(t, config.EnableLiveReload)
	assert.True(t, config.HotReloadEnabled)
	assert.True(t, config.EnableSandboxing)
	assert.True(t, config.EnableCaching)
	assert.Equal(t, 5*time.Minute, config.CacheTimeout)
	assert.Equal(t, 10, config.MaxConcurrentRenders)
}

func TestNewEnhancedPreviewSystem(t *testing.T) {
	eps := NewEnhancedPreviewSystem(nil, nil, nil)

	require.NotNil(t, eps)
	require.NotNil(t, eps.config)
	require.NotNil(t, eps.templateManager)
	require.NotNil(t, eps.assetManager)
	require.NotNil(t, eps.liveReload)
	require.NotNil(t, eps.sandboxManager)
	require.NotNil(t, eps.sessionManager)
	require.NotNil(t, eps.performanceMonitor)
}

func TestTemplateManager_Creation(t *testing.T) {
	config := DefaultPreviewConfig()
	tm := NewTemplateManager(config)

	require.NotNil(t, tm)
	require.NotNil(t, tm.templates)
	require.NotNil(t, tm.layoutTemplates)
	require.NotNil(t, tm.partialTemplates)
}

func TestAssetManager_Creation(t *testing.T) {
	config := DefaultPreviewConfig()
	am := NewAssetManager(config)

	require.NotNil(t, am)
	require.NotNil(t, am.assets)
}

func TestLiveReloadManager_Creation(t *testing.T) {
	config := DefaultPreviewConfig()
	lrm := NewLiveReloadManager(config)

	require.NotNil(t, lrm)
	require.NotNil(t, lrm.connections)
	require.NotNil(t, lrm.broadcastCh)

	// Test broadcast
	event := LiveReloadEvent{
		Type:      "test",
		Timestamp: time.Now(),
	}

	lrm.Broadcast(event)

	// Should be able to receive the event
	select {
	case receivedEvent := <-lrm.broadcastCh:
		assert.Equal(t, "test", receivedEvent.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected to receive broadcast event")
	}
}

func TestSandboxManager_CreateDestroySandbox(t *testing.T) {
	config := DefaultPreviewConfig()
	sm := NewSandboxManager(config)

	require.NotNil(t, sm)
	require.NotNil(t, sm.resourceLimits)

	// Test sandbox creation
	sandbox, err := sm.CreateSandbox("TestComponent", 1)
	require.NoError(t, err)
	require.NotNil(t, sandbox)

	assert.Equal(t, "TestComponent", sandbox.ComponentName)
	assert.Equal(t, 1, sandbox.IsolationLevel)
	assert.NotEmpty(t, sandbox.ID)

	// Verify sandbox is tracked
	sm.sandboxMutex.RLock()
	_, exists := sm.sandboxes[sandbox.ID]
	sm.sandboxMutex.RUnlock()
	assert.True(t, exists)

	// Test sandbox destruction
	sm.DestroySandbox(sandbox.ID)

	// Verify sandbox is removed
	sm.sandboxMutex.RLock()
	_, exists = sm.sandboxes[sandbox.ID]
	sm.sandboxMutex.RUnlock()
	assert.False(t, exists)
}

func TestSessionManager_GetOrCreateSession(t *testing.T) {
	config := DefaultPreviewConfig()
	sesm := NewSessionManager(config)

	ctx := context.Background()

	// Test creating new session with empty ID
	session1 := sesm.GetOrCreateSession(ctx, "")
	require.NotNil(t, session1)
	assert.NotEmpty(t, session1.ID)
	assert.Equal(t, 1200, session1.ViewportSize.Width)
	assert.Equal(t, 800, session1.ViewportSize.Height)

	// Test getting existing session
	session2 := sesm.GetOrCreateSession(ctx, session1.ID)
	assert.Equal(t, session1.ID, session2.ID)
	assert.Equal(t, session1, session2)

	// Test creating session with specific ID
	customID := "custom-session-123"
	session3 := sesm.GetOrCreateSession(ctx, customID)
	assert.Equal(t, customID, session3.ID)
}

func TestPreviewPerformanceMonitor_RecordRender(t *testing.T) {
	monitor := NewPreviewPerformanceMonitor()

	// Test successful render
	monitor.RecordRender(100*time.Millisecond, nil)

	monitor.metricsMutex.RLock()
	assert.Equal(t, int64(1), monitor.metrics.TotalRenders)
	assert.Equal(t, int64(1), monitor.metrics.SuccessfulRenders)
	assert.Equal(t, int64(0), monitor.metrics.FailedRenders)
	assert.Equal(t, 100*time.Millisecond, monitor.metrics.AverageRenderTime)
	monitor.metricsMutex.RUnlock()

	// Test failed render
	monitor.RecordRender(200*time.Millisecond, assert.AnError)

	monitor.metricsMutex.RLock()
	assert.Equal(t, int64(2), monitor.metrics.TotalRenders)
	assert.Equal(t, int64(1), monitor.metrics.SuccessfulRenders)
	assert.Equal(t, int64(1), monitor.metrics.FailedRenders)
	assert.Equal(t, 150*time.Millisecond, monitor.metrics.AverageRenderTime)
	monitor.metricsMutex.RUnlock()
}

func TestPreviewOptions_Validation(t *testing.T) {
	options := &PreviewOptions{
		SessionID:      "test-session",
		Theme:          "dark",
		ViewportSize:   &ViewportSize{Width: 1024, Height: 768, Scale: 1.0},
		DeviceMode:     "desktop",
		IsolationLevel: 2,
		MockData:       true,
		ShowDebugInfo:  true,
		Layout:         "default",
	}

	assert.Equal(t, "test-session", options.SessionID)
	assert.Equal(t, "dark", options.Theme)
	assert.Equal(t, 1024, options.ViewportSize.Width)
	assert.Equal(t, 768, options.ViewportSize.Height)
	assert.Equal(t, 1.0, options.ViewportSize.Scale)
	assert.True(t, options.MockData)
	assert.True(t, options.ShowDebugInfo)
}

func TestEnhancedPreviewSystem_PreviewComponent(t *testing.T) {
	eps := NewEnhancedPreviewSystem(nil, nil, nil)

	ctx := context.Background()
	componentName := "TestComponent"
	props := map[string]interface{}{
		"title": "Test Title",
		"count": 42,
	}

	options := &PreviewOptions{
		SessionID:     "test-session",
		Theme:         "light",
		ViewportSize:  &ViewportSize{Width: 1200, Height: 800, Scale: 1.0},
		DeviceMode:    "desktop",
		MockData:      true,
		ShowDebugInfo: true,
	}

	// Mock the managers to avoid actual initialization
	eps.templateManager = NewTemplateManager(eps.config)
	eps.assetManager = NewAssetManager(eps.config)
	eps.liveReload = NewLiveReloadManager(eps.config)
	eps.sandboxManager = NewSandboxManager(eps.config)
	eps.sessionManager = NewSessionManager(eps.config)

	result, err := eps.PreviewComponent(ctx, componentName, props, options)

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.NotEmpty(t, result.HTML)
	assert.NotNil(t, result.Metadata)
	assert.Equal(t, componentName, result.Metadata.ComponentName)
	assert.Equal(t, props, result.Metadata.Props)
	assert.NotNil(t, result.Performance)

	// Verify session was created and updated
	session := eps.sessionManager.GetOrCreateSession(ctx, options.SessionID)
	assert.Equal(t, componentName, session.CurrentComponent)
	assert.Equal(t, props, session.ComponentProps)
	assert.Len(t, session.History, 1)
	assert.Equal(t, componentName, session.History[0].ComponentName)
}

func TestEnhancedPreviewSystem_GetPreviewMetrics(t *testing.T) {
	eps := NewEnhancedPreviewSystem(nil, nil, nil)

	// Record some test metrics
	eps.performanceMonitor.RecordRender(100*time.Millisecond, nil)
	eps.performanceMonitor.RecordRender(200*time.Millisecond, assert.AnError)

	// Create some sessions to test active session count
	ctx := context.Background()
	eps.sessionManager.GetOrCreateSession(ctx, "session1")
	eps.sessionManager.GetOrCreateSession(ctx, "session2")

	metrics := eps.GetPreviewMetrics()

	require.NotNil(t, metrics)
	assert.Equal(t, int64(2), metrics.TotalRenders)
	assert.Equal(t, int64(1), metrics.SuccessfulRenders)
	assert.Equal(t, int64(1), metrics.FailedRenders)
	assert.Equal(t, 2, metrics.ActiveSessions)
	assert.Equal(t, 0.5, metrics.ErrorRate)
	assert.Equal(t, 150*time.Millisecond, metrics.AverageRenderTime)
}

func TestViewportSize_Validation(t *testing.T) {
	viewport := ViewportSize{
		Width:  1920,
		Height: 1080,
		Scale:  1.5,
	}

	assert.Equal(t, 1920, viewport.Width)
	assert.Equal(t, 1080, viewport.Height)
	assert.Equal(t, 1.5, viewport.Scale)
}

func TestPreviewHistoryEntry_Creation(t *testing.T) {
	entry := PreviewHistoryEntry{
		ComponentName: "Button",
		Props:         map[string]interface{}{"text": "Click me"},
		Timestamp:     time.Now(),
		Title:         "Button Preview",
	}

	assert.Equal(t, "Button", entry.ComponentName)
	assert.Equal(t, "Click me", entry.Props["text"])
	assert.Equal(t, "Button Preview", entry.Title)
	assert.False(t, entry.Timestamp.IsZero())
}

func TestComponentBookmark_Creation(t *testing.T) {
	bookmark := ComponentBookmark{
		ID:            "bookmark-123",
		Name:          "My Favorite Button",
		ComponentName: "Button",
		Props:         map[string]interface{}{"variant": "primary"},
		Description:   "A nice primary button",
		CreatedAt:     time.Now(),
	}

	assert.Equal(t, "bookmark-123", bookmark.ID)
	assert.Equal(t, "My Favorite Button", bookmark.Name)
	assert.Equal(t, "Button", bookmark.ComponentName)
	assert.Equal(t, "primary", bookmark.Props["variant"])
	assert.Equal(t, "A nice primary button", bookmark.Description)
	assert.False(t, bookmark.CreatedAt.IsZero())
}

func TestResourceLimits_DefaultValues(t *testing.T) {
	config := DefaultPreviewConfig()
	sm := NewSandboxManager(config)

	limits := sm.resourceLimits

	assert.Equal(t, 100, limits.MaxMemoryMB)
	assert.Equal(t, 50.0, limits.MaxCPUPercent)
	assert.Equal(t, 30*time.Second, limits.MaxExecutionTime)
	assert.Equal(t, int64(10*1024*1024), limits.MaxFileSize)
	assert.Equal(t, 10, limits.MaxNetworkCalls)
}

func BenchmarkSessionManager_GetOrCreateSession(b *testing.B) {
	config := DefaultPreviewConfig()
	sesm := NewSessionManager(config)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		sessionID := "benchmark-session"
		sesm.GetOrCreateSession(ctx, sessionID)
	}
}

func BenchmarkPreviewPerformanceMonitor_RecordRender(b *testing.B) {
	monitor := NewPreviewPerformanceMonitor()

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		monitor.RecordRender(100*time.Millisecond, nil)
	}
}

func BenchmarkSandboxManager_CreateDestroySandbox(b *testing.B) {
	config := DefaultPreviewConfig()
	sm := NewSandboxManager(config)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		sandbox, _ := sm.CreateSandbox("BenchmarkComponent", 1)
		sm.DestroySandbox(sandbox.ID)
	}
}
