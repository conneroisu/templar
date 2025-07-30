package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebSocketMessageValidation validates WebSocket message structure and flow
func TestWebSocketMessageValidation(t *testing.T) {
	t.Run("update_message_structure", func(t *testing.T) {
		// Test UpdateMessage structure and JSON serialization
		testMessage := UpdateMessage{
			Type:      "build_success",
			Target:    "Button",
			Content:   "Component built successfully",
			Timestamp: GetCurrentTime(),
		}

		// Verify message fields
		assert.Equal(t, "build_success", testMessage.Type)
		assert.Equal(t, "Button", testMessage.Target)
		assert.Equal(t, "Component built successfully", testMessage.Content)
		assert.NotEmpty(t, testMessage.Timestamp)

		// Test JSON marshaling
		jsonData, err := json.Marshal(testMessage)
		require.NoError(t, err, "Should marshal to JSON successfully")

		jsonStr := string(jsonData)
		assert.Contains(t, jsonStr, "build_success")
		assert.Contains(t, jsonStr, "Button")
		assert.Contains(t, jsonStr, "Component built successfully")

		// Test JSON unmarshaling
		var unmarshaled UpdateMessage
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err, "Should unmarshal from JSON successfully")

		assert.Equal(t, testMessage.Type, unmarshaled.Type)
		assert.Equal(t, testMessage.Target, unmarshaled.Target)
		assert.Equal(t, testMessage.Content, unmarshaled.Content)
		assert.Equal(t, testMessage.Timestamp, unmarshaled.Timestamp)

		t.Log("✅ UpdateMessage structure validation successful")
	})

	t.Run("message_types_validation", func(t *testing.T) {
		// Test all expected message types
		messageTypes := []struct {
			messageType string
			target      string
			content     string
			description string
		}{
			{
				messageType: "file_change",
				target:      "Button.templ",
				content:     "File modified",
				description: "File system change notification",
			},
			{
				messageType: "component_updated",
				target:      "Button",
				content:     "Component parameters changed",
				description: "Component registry update",
			},
			{
				messageType: "build_start",
				target:      "Button",
				content:     "Build process started",
				description: "Build pipeline start notification",
			},
			{
				messageType: "build_success",
				target:      "Button",
				content:     "Build completed successfully",
				description: "Successful build completion",
			},
			{
				messageType: "build_error",
				target:      "Card",
				content:     "Syntax error on line 42: missing closing tag",
				description: "Build failure notification",
			},
			{
				messageType: "scan_start",
				target:      "components/",
				content:     "Component scanning started",
				description: "Scanner process start",
			},
			{
				messageType: "scan_complete",
				target:      "components/",
				content:     "Found 15 components",
				description: "Scanner process completion",
			},
			{
				messageType: "server_ready",
				target:      "templar",
				content:     "Development server ready",
				description: "Server initialization complete",
			},
		}

		for _, msgType := range messageTypes {
			t.Run(msgType.messageType, func(t *testing.T) {
				message := UpdateMessage{
					Type:      msgType.messageType,
					Target:    msgType.target,
					Content:   msgType.content,
					Timestamp: GetCurrentTime(),
				}

				// Validate message can be serialized
				jsonData, err := json.Marshal(message)
				require.NoError(t, err, "Message type %s should serialize", msgType.messageType)

				// Ensure JSON is valid and contains expected data
				var parsed UpdateMessage
				err = json.Unmarshal(jsonData, &parsed)
				require.NoError(t, err, "Message type %s should deserialize", msgType.messageType)

				assert.Equal(t, msgType.messageType, parsed.Type)
				assert.Equal(t, msgType.target, parsed.Target)
				assert.Equal(t, msgType.content, parsed.Content)

				t.Logf("✅ %s: %s", msgType.messageType, msgType.description)
			})
		}

		t.Log("✅ All message types validated successfully")
	})

	t.Run("timestamp_validation", func(t *testing.T) {
		// Test timestamp generation and format
		timestamp1 := GetCurrentTime()
		time.Sleep(1 * time.Millisecond)
		timestamp2 := GetCurrentTime()

		// Timestamps should be different
		assert.NotEqual(t, timestamp1, timestamp2, "Timestamps should be unique")

		// Timestamps should be valid time values
		assert.False(t, timestamp1.IsZero(), "Timestamp1 should not be zero")
		assert.False(t, timestamp2.IsZero(), "Timestamp2 should not be zero")
		assert.True(t, timestamp2.After(timestamp1), "Timestamp2 should be after timestamp1")

		// Test in message context
		msg := UpdateMessage{
			Type:      "test",
			Target:    "test",
			Content:   "test",
			Timestamp: timestamp1,
		}

		jsonData, err := json.Marshal(msg)
		require.NoError(t, err)

		var parsed UpdateMessage
		err = json.Unmarshal(jsonData, &parsed)
		require.NoError(t, err)

		assert.Equal(t, timestamp1, parsed.Timestamp, "Timestamp should survive serialization")

		t.Log("✅ Timestamp validation successful")
	})

	t.Run("message_content_validation", func(t *testing.T) {
		// Test various content types and sizes
		testCases := []struct {
			name    string
			content string
		}{
			{
				name:    "short_content",
				content: "OK",
			},
			{
				name:    "empty_content",
				content: "",
			},
			{
				name:    "html_content",
				content: "<div class=\"error\">Syntax error in template</div>",
			},
			{
				name:    "json_content",
				content: `{"component": "Button", "parameters": ["text", "variant"]}`,
			},
			{
				name:    "multiline_content",
				content: "Line 1\nLine 2\nLine 3",
			},
			{
				name:    "unicode_content",
				content: "Component built successfully ✅ 用户界面组件",
			},
			{
				name:    "large_content",
				content: generateLargeContent(1000),
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				message := UpdateMessage{
					Type:      "test",
					Target:    "TestComponent",
					Content:   tc.content,
					Timestamp: GetCurrentTime(),
				}

				// Test serialization
				jsonData, err := json.Marshal(message)
				require.NoError(t, err, "Should serialize %s", tc.name)

				// Test deserialization
				var parsed UpdateMessage
				err = json.Unmarshal(jsonData, &parsed)
				require.NoError(t, err, "Should deserialize %s", tc.name)

				assert.Equal(t, tc.content, parsed.Content, 
					"Content should match for %s", tc.name)

				t.Logf("✅ %s: content length %d", tc.name, len(tc.content))
			})
		}

		t.Log("✅ Message content validation successful")
	})

	t.Run("message_flow_sequences", func(t *testing.T) {
		// Test typical message flow sequences
		sequences := []struct {
			name     string
			messages []UpdateMessage
		}{
			{
				name: "successful_hot_reload",
				messages: []UpdateMessage{
					{Type: "file_change", Target: "Button.templ", Content: "File modified", Timestamp: GetCurrentTime()},
					{Type: "scan_start", Target: "Button.templ", Content: "Scanning component", Timestamp: GetCurrentTime()},
					{Type: "component_updated", Target: "Button", Content: "Component updated", Timestamp: GetCurrentTime()},
					{Type: "build_start", Target: "Button", Content: "Building component", Timestamp: GetCurrentTime()},
					{Type: "build_success", Target: "Button", Content: "Build successful", Timestamp: GetCurrentTime()},
				},
			},
			{
				name: "build_failure_recovery",
				messages: []UpdateMessage{
					{Type: "file_change", Target: "Card.templ", Content: "File modified", Timestamp: GetCurrentTime()},
					{Type: "build_start", Target: "Card", Content: "Building component", Timestamp: GetCurrentTime()},
					{Type: "build_error", Target: "Card", Content: "Syntax error", Timestamp: GetCurrentTime()},
					{Type: "file_change", Target: "Card.templ", Content: "File fixed", Timestamp: GetCurrentTime()},
					{Type: "build_start", Target: "Card", Content: "Rebuilding component", Timestamp: GetCurrentTime()},
					{Type: "build_success", Target: "Card", Content: "Build successful", Timestamp: GetCurrentTime()},
				},
			},
			{
				name: "multiple_component_update",
				messages: []UpdateMessage{
					{Type: "file_change", Target: "utils.go", Content: "Helper file modified", Timestamp: GetCurrentTime()},
					{Type: "build_start", Target: "Button", Content: "Rebuilding affected components", Timestamp: GetCurrentTime()},
					{Type: "build_start", Target: "Card", Content: "Rebuilding affected components", Timestamp: GetCurrentTime()},
					{Type: "build_start", Target: "Modal", Content: "Rebuilding affected components", Timestamp: GetCurrentTime()},
					{Type: "build_success", Target: "Button", Content: "Build successful", Timestamp: GetCurrentTime()},
					{Type: "build_success", Target: "Card", Content: "Build successful", Timestamp: GetCurrentTime()},
					{Type: "build_success", Target: "Modal", Content: "Build successful", Timestamp: GetCurrentTime()},
				},
			},
		}

		for _, sequence := range sequences {
			t.Run(sequence.name, func(t *testing.T) {
				// Validate each message in the sequence
				for i, msg := range sequence.messages {
					// Ensure message is valid
					assert.NotEmpty(t, msg.Type, "Message %d should have type", i+1)
					assert.NotEmpty(t, msg.Target, "Message %d should have target", i+1)
					assert.NotEmpty(t, msg.Timestamp, "Message %d should have timestamp", i+1)

					// Ensure JSON serialization works
					jsonData, err := json.Marshal(msg)
					require.NoError(t, err, "Message %d should serialize", i+1)

					var parsed UpdateMessage
					err = json.Unmarshal(jsonData, &parsed)
					require.NoError(t, err, "Message %d should deserialize", i+1)

					assert.Equal(t, msg.Type, parsed.Type, "Message %d type should match", i+1)
				}

				t.Logf("✅ %s: validated %d messages", sequence.name, len(sequence.messages))
			})
		}

		t.Log("✅ Message flow sequences validated successfully")
	})

	t.Run("websocket_manager_configuration", func(t *testing.T) {
		// Test WebSocket manager configuration and setup
		validator := &simpleOriginValidator{}
		
		// Test creation without panicking
		wsManager := NewWebSocketManager(validator, nil)
		require.NotNil(t, wsManager, "WebSocket manager should be created")

		// Test initial state
		clientCount := wsManager.GetConnectedClients()
		assert.Equal(t, 0, clientCount, "Should start with no clients")

		isShutdown := wsManager.IsShutdown()
		assert.False(t, isShutdown, "Should not be shutdown initially")

		// Test graceful shutdown
		err := wsManager.Shutdown(context.TODO())
		assert.NoError(t, err, "Shutdown should succeed")

		isShutdown = wsManager.IsShutdown()
		assert.True(t, isShutdown, "Should be shutdown after calling Shutdown()")

		t.Log("✅ WebSocket manager configuration validated successfully")
	})
}

// Helper functions

func generateLargeContent(size int) string {
	content := make([]byte, size)
	for i := range content {
		content[i] = byte('A' + (i % 26))
	}
	return string(content)
}

type simpleOriginValidator struct{}

func (s *simpleOriginValidator) IsAllowedOrigin(origin string) bool {
	return true // Allow all origins for testing
}