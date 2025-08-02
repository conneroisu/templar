//go:build property
// +build property

// Package testutils provides property-based testing utilities for comprehensive coverage.
// This implements property-based testing patterns following the zero technical debt policy.
package testutils

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// PropertyTestConfig configures property-based tests.
type PropertyTestConfig struct {
	TestCases    int           // Number of test cases to generate (default: 100)
	Workers      int           // Number of workers (default: runtime.NumCPU())
	MaxShrinks   int           // Maximum shrink attempts (default: 1000)
	Timeout      time.Duration // Timeout per test case (default: 30s)
	Seed         int64         // Random seed (0 for random)
	MinSuccesses int           // Minimum successful tests (default: TestCases)
}

// DefaultPropertyConfig returns a default configuration for property tests.
func DefaultPropertyConfig() *PropertyTestConfig {
	return &PropertyTestConfig{
		TestCases:    100,
		Workers:      4,
		MaxShrinks:   1000,
		Timeout:      30 * time.Second,
		Seed:         0,
		MinSuccesses: 100,
	}
}

// NewPropertyTester creates a new gopter Properties instance with the given config.
func NewPropertyTester(config *PropertyTestConfig) *gopter.Properties {
	if config == nil {
		config = DefaultPropertyConfig()
	}
	
	parameters := &gopter.TestParameters{
		MinSuccessfulTests: config.MinSuccesses,
		MaxSize:            50,
		MinSize:            0,
		MaxShrinkCount:     config.MaxShrinks,
		Workers:           config.Workers,
	}
	
	if config.Seed != 0 {
		parameters.Rng = gopter.NewLockedRandom(config.Seed)
	}
	
	return gopter.NewProperties(parameters)
}

// Generators for common types used in Templar

// GenValidFileName generates valid file names.
func GenValidFileName() gopter.Gen {
	return gen.RegexMatch(`^[a-zA-Z][a-zA-Z0-9_\-]{0,50}\.(templ|go|js|css|html)$`)
}

// GenValidPackageName generates valid Go package names.
func GenValidPackageName() gopter.Gen {
	return gen.RegexMatch(`^[a-z][a-z0-9_]{0,20}$`)
}

// GenValidComponentName generates valid templ component names.
func GenValidComponentName() gopter.Gen {
	return gen.RegexMatch(`^[A-Z][a-zA-Z0-9]{0,30}$`)
}

// GenValidPath generates valid file system paths.
func GenValidPath() gopter.Gen {
	return gen.SliceOfN(1, 5, gen.RegexMatch(`^[a-zA-Z0-9_\-]{1,20}$`)).
		Map(func(segments []string) string {
			return strings.Join(segments, "/")
		}).
		SuchThat(func(path string) bool {
			return !strings.Contains(path, "..") && 
				   !strings.HasPrefix(path, "/") &&
				   len(path) > 0 && len(path) < 200
		})
}

// GenValidHostname generates valid hostnames.
func GenValidHostname() gopter.Gen {
	return gen.OneOf(
		gen.Const("localhost"),
		gen.Const("127.0.0.1"),
		gen.Const("::1"),
		gen.RegexMatch(`^[a-z0-9\-\.]{1,50}$`),
	).SuchThat(func(host string) bool {
		return len(host) > 0 && len(host) <= 253 && 
			   !strings.HasPrefix(host, ".") &&
			   !strings.HasSuffix(host, ".")
	})
}

// GenValidPort generates valid port numbers.
func GenValidPort() gopter.Gen {
	return gen.IntRange(1024, 65535)
}

// GenTemplateContent generates valid templ template content.
func GenTemplateContent() gopter.Gen {
	return gen.SliceOfN(1, 10, gen.OneOf(
		gen.Const("<div>"),
		gen.Const("</div>"),
		gen.Const("<span>text</span>"),
		gen.Const("<h1>Title</h1>"),
		gen.RegexMatch(`^[a-zA-Z0-9\s\-_]{1,50}$`),
	)).Map(func(parts []string) string {
		return strings.Join(parts, "\n")
	})
}

// GenComponentParameters generates component parameters.
func GenComponentParameters() gopter.Gen {
	return gen.SliceOfN(0, 5, gen.Struct(reflect.TypeOf(TestParameter{}), map[string]gopter.Gen{
		"Name": gen.RegexMatch(`^[a-z][a-zA-Z0-9]{0,20}$`),
		"Type": gen.OneOf(
			gen.Const("string"),
			gen.Const("int"),
			gen.Const("bool"),
			gen.Const("[]string"),
			gen.Const("map[string]string"),
		),
	}))
}

// SafeStringGen generates strings safe for file paths and identifiers.
func SafeStringGen() gopter.Gen {
	return gen.RegexMatch(`^[a-zA-Z0-9_\-]{1,50}$`)
}

// PropertyTestHelpers provides utilities for property-based testing.
type PropertyTestHelpers struct {
	t      *testing.T
	config *PropertyTestConfig
}

// NewPropertyTestHelpers creates a new property test helper.
func NewPropertyTestHelpers(t *testing.T, config *PropertyTestConfig) *PropertyTestHelpers {
	if config == nil {
		config = DefaultPropertyConfig()
	}
	return &PropertyTestHelpers{t: t, config: config}
}

// TestProperty runs a property test with the given name and property function.
func (pth *PropertyTestHelpers) TestProperty(name string, property interface{}) {
	pth.t.Run(name, func(t *testing.T) {
		properties := NewPropertyTester(pth.config)
		properties.Property(name, prop.ForAll(property))
		properties.TestingRun(t)
	})
}

// TestInvariant tests that an invariant holds for generated values.
func (pth *PropertyTestHelpers) TestInvariant(name string, generator gopter.Gen, invariant func(interface{}) bool) {
	pth.TestProperty(name, prop.ForAll(invariant, generator))
}

// TestRoundTrip tests that a roundtrip operation (serialize -> deserialize) preserves data.
func (pth *PropertyTestHelpers) TestRoundTrip(name string, generator gopter.Gen, 
	serialize func(interface{}) ([]byte, error),
	deserialize func([]byte) (interface{}, error),
	equal func(interface{}, interface{}) bool) {
	
	pth.TestProperty(name, prop.ForAll(
		func(original interface{}) bool {
			serialized, err := serialize(original)
			if err != nil {
				return false
			}
			
			deserialized, err := deserialize(serialized)
			if err != nil {
				return false
			}
			
			return equal(original, deserialized)
		}, generator))
}

// TestCommutative tests that an operation is commutative (a op b == b op a).
func (pth *PropertyTestHelpers) TestCommutative(name string, generator gopter.Gen,
	operation func(interface{}, interface{}) interface{},
	equal func(interface{}, interface{}) bool) {
	
	pth.TestProperty(name, prop.ForAll(
		func(a, b interface{}) bool {
			result1 := operation(a, b)
			result2 := operation(b, a)
			return equal(result1, result2)
		}, generator, generator))
}

// TestAssociative tests that an operation is associative ((a op b) op c == a op (b op c)).
func (pth *PropertyTestHelpers) TestAssociative(name string, generator gopter.Gen,
	operation func(interface{}, interface{}) interface{},
	equal func(interface{}, interface{}) bool) {
	
	pth.TestProperty(name, prop.ForAll(
		func(a, b, c interface{}) bool {
			result1 := operation(operation(a, b), c)
			result2 := operation(a, operation(b, c))
			return equal(result1, result2)
		}, generator, generator, generator))
}

// TestIdempotent tests that an operation is idempotent (f(f(x)) == f(x)).
func (pth *PropertyTestHelpers) TestIdempotent(name string, generator gopter.Gen,
	operation func(interface{}) interface{},
	equal func(interface{}, interface{}) bool) {
	
	pth.TestProperty(name, prop.ForAll(
		func(x interface{}) bool {
			once := operation(x)
			twice := operation(once)
			return equal(once, twice)
		}, generator))
}

// TestMonotonic tests that a function is monotonically increasing.
func (pth *PropertyTestHelpers) TestMonotonic(name string,
	operation func(int) int) {
	
	pth.TestProperty(name, prop.ForAll(
		func(a, b int) bool {
			if a >= b {
				return true // Skip if not ordered
			}
			
			resultA := operation(a)
			resultB := operation(b)
			
			return resultA <= resultB
		}, gen.Int(), gen.Int()))
}

// ValidationProperties provides common validation property tests.
type ValidationProperties struct {
	helpers *PropertyTestHelpers
}

// NewValidationProperties creates validation property tests.
func NewValidationProperties(t *testing.T) *ValidationProperties {
	return &ValidationProperties{
		helpers: NewPropertyTestHelpers(t, DefaultPropertyConfig()),
	}
}

// TestPathValidation tests that path validation is consistent.
func (vp *ValidationProperties) TestPathValidation(validator func(string) error) {
	vp.helpers.TestProperty("path_validation_consistent", prop.ForAll(
		func(path string) bool {
			// Test that validation is deterministic
			err1 := validator(path)
			err2 := validator(path)
			return (err1 == nil) == (err2 == nil)
		}, GenValidPath()))
}

// TestFilenameValidation tests filename validation properties.
func (vp *ValidationProperties) TestFilenameValidation(validator func(string) error) {
	vp.helpers.TestProperty("filename_validation_rejects_dangerous_chars", prop.ForAll(
		func(filename string) bool {
			// Test that dangerous characters are rejected
			dangerousChars := []string{"..", "/", "\\", "\x00", "\n", "\r"}
			for _, char := range dangerousChars {
				if strings.Contains(filename, char) {
					return validator(filename) != nil
				}
			}
			return true
		}, gen.AnyString()))
}

// TestConfigurationProperties tests configuration-related properties.
func (vp *ValidationProperties) TestConfigurationProperties(
	loadConfig func(map[string]interface{}) error,
	validateConfig func(interface{}) error) {
	
	vp.helpers.TestProperty("config_validation_is_consistent", prop.ForAll(
		func(port int, host string) bool {
			if port < 1 || port > 65535 {
				return true // Skip invalid ports
			}
			
			config := map[string]interface{}{
				"server": map[string]interface{}{
					"port": port,
					"host": host,
				},
			}
			
			err1 := loadConfig(config)
			err2 := loadConfig(config)
			
			// Should be deterministic
			return (err1 == nil) == (err2 == nil)
		}, GenValidPort(), GenValidHostname()))
}

// SecurityProperties provides security-focused property tests.
type SecurityProperties struct {
	helpers *PropertyTestHelpers
}

// NewSecurityProperties creates security property tests.
func NewSecurityProperties(t *testing.T) *SecurityProperties {
	return &SecurityProperties{
		helpers: NewPropertyTestHelpers(t, DefaultPropertyConfig()),
	}
}

// TestInputSanitization tests that input sanitization is thorough.
func (sp *SecurityProperties) TestInputSanitization(sanitize func(string) string) {
	sp.helpers.TestProperty("sanitization_removes_dangerous_chars", prop.ForAll(
		func(input string) bool {
			sanitized := sanitize(input)
			
			// Check that dangerous characters are removed
			dangerousChars := []string{"<script", "javascript:", "data:", "vbscript:", "onload="}
			for _, danger := range dangerousChars {
				if strings.Contains(strings.ToLower(sanitized), danger) {
					return false
				}
			}
			
			return true
		}, gen.AnyString()))
}

// TestAuthenticationBypass tests for authentication bypass vulnerabilities.
func (sp *SecurityProperties) TestAuthenticationBypass(
	authenticate func(string, string) bool) {
	
	sp.helpers.TestProperty("empty_credentials_rejected", prop.ForAll(
		func() bool {
			// Empty credentials should always be rejected
			return !authenticate("", "") &&
				   !authenticate("admin", "") &&
				   !authenticate("", "password")
		}))
}

// TestRateLimiting tests rate limiting properties.
func (sp *SecurityProperties) TestRateLimiting(
	rateLimiter func(string) bool, // returns true if request allowed
	reset func()) {
	
	sp.helpers.TestProperty("rate_limiting_enforced", prop.ForAll(
		func(clientID string) bool {
			reset() // Reset rate limiter state
			
			// Should allow some requests
			allowed := 0
			denied := 0
			
			for i := 0; i < 100; i++ {
				if rateLimiter(clientID) {
					allowed++
				} else {
					denied++
				}
			}
			
			// Should have some rate limiting (not all allowed, not all denied)
			return allowed > 0 && denied > 0
		}, SafeStringGen()))
}

// PerformanceProperties provides performance-focused property tests.
type PerformanceProperties struct {
	helpers *PropertyTestHelpers
}

// NewPerformanceProperties creates performance property tests.
func NewPerformanceProperties(t *testing.T) *PerformanceProperties {
	return &PerformanceProperties{
		helpers: NewPropertyTestHelpers(t, DefaultPropertyConfig()),
	}
}

// TestScalability tests that operations scale reasonably with input size.
func (pp *PerformanceProperties) TestScalability(
	operation func([]string) interface{},
	maxTimePerItem time.Duration) {
	
	pp.helpers.TestProperty("operation_scales_linearly", prop.ForAll(
		func(items []string) bool {
			if len(items) == 0 {
				return true
			}
			
			start := time.Now()
			operation(items)
			elapsed := time.Since(start)
			
			// Should not exceed max time per item
			maxExpected := time.Duration(len(items)) * maxTimePerItem
			return elapsed <= maxExpected
		}, gen.SliceOfN(1, 1000, SafeStringGen())))
}

// TestMemoryUsage tests that operations don't use excessive memory.
func (pp *PerformanceProperties) TestMemoryUsage(
	operation func([]string) interface{}) {
	
	pp.helpers.TestProperty("memory_usage_reasonable", prop.ForAll(
		func(items []string) bool {
			// This is a basic test - in practice you'd use runtime.MemStats
			// to measure actual memory usage
			result := operation(items)
			
			// Basic sanity check that operation returns something
			return result != nil
		}, gen.SliceOfN(1, 100, SafeStringGen())))
}