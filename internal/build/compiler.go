// Package build provides templ compilation functionality with security validation.
package build

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/conneroisu/templar/internal/types"
	"github.com/conneroisu/templar/internal/validation"
)

// TemplCompiler handles templ compilation
type TemplCompiler struct {
	command string
	args    []string
}

// NewTemplCompiler creates a new templ compiler
func NewTemplCompiler() *TemplCompiler {
	return &TemplCompiler{
		command: "templ",
		args:    []string{"generate"},
	}
}

// Compile compiles a component using templ generate with context-based timeout
func (tc *TemplCompiler) Compile(ctx context.Context, component *types.ComponentInfo) ([]byte, error) {
	// Validate command and arguments to prevent command injection
	if err := tc.validateCommand(); err != nil {
		return nil, fmt.Errorf("command validation failed: %w", err)
	}

	// Run templ generate command with context for timeout handling
	cmd := exec.CommandContext(ctx, tc.command, tc.args...)
	cmd.Dir = "." // Run in current directory

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if error is due to context cancellation (timeout)
		if ctx.Err() != nil {
			return nil, fmt.Errorf("templ generate timed out: %w", ctx.Err())
		}
		return nil, fmt.Errorf("templ generate failed: %w\nOutput: %s", err, output)
	}

	return output, nil
}

// CompileWithPools performs compilation using object pools for memory efficiency with context-based timeout
func (tc *TemplCompiler) CompileWithPools(ctx context.Context, component *types.ComponentInfo, pools *ObjectPools) ([]byte, error) {
	// Validate command and arguments to prevent command injection
	if err := tc.validateCommand(); err != nil {
		return nil, fmt.Errorf("command validation failed: %w", err)
	}

	// Get pooled buffer for output
	outputBuffer := pools.GetOutputBuffer()
	defer pools.PutOutputBuffer(outputBuffer)

	// Run templ generate command with context for timeout handling
	cmd := exec.CommandContext(ctx, tc.command, tc.args...)
	cmd.Dir = "." // Run in current directory

	// Use pooled buffers for command output
	var err error

	if output, cmdErr := cmd.CombinedOutput(); cmdErr != nil {
		// Check if error is due to context cancellation (timeout)
		if ctx.Err() != nil {
			return nil, fmt.Errorf("templ generate timed out: %w", ctx.Err())
		}
		// Copy output to our buffer to avoid keeping the original allocation
		outputBuffer = append(outputBuffer, output...)
		err = fmt.Errorf("templ generate failed: %w\nOutput: %s", cmdErr, outputBuffer)
		return nil, err
	} else {
		// Copy successful output to our buffer
		outputBuffer = append(outputBuffer, output...)
	}

	// Return a copy of the buffer content (caller owns this memory)
	result := make([]byte, len(outputBuffer))
	copy(result, outputBuffer)
	return result, nil
}

// validateCommand validates the command and arguments to prevent command injection
func (tc *TemplCompiler) validateCommand() error {
	// Allowlist of permitted commands
	allowedCommands := map[string]bool{
		"templ": true,
		"go":    true,
	}

	// Use centralized validation for command
	if err := validation.ValidateCommand(tc.command, allowedCommands); err != nil {
		return err
	}

	// Validate arguments using centralized validation
	for _, arg := range tc.args {
		if err := validation.ValidateArgument(arg); err != nil {
			return fmt.Errorf("invalid argument '%s': %w", arg, err)
		}
	}

	return nil
}
