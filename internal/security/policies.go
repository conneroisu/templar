// Package server/security provides comprehensive web security features including
// Content Security Policy (CSP), HTTP Strict Transport Security (HSTS),
// XSS protection, CSRF prevention, and request validation.
//
// The security package implements defense-in-depth security measures:
// - CSP with nonce-based script/style protection
// - HSTS with configurable max-age and subdomain inclusion
// - XSS protection and content type validation
// - Origin validation for WebSocket and API requests
// - Rate limiting and malicious user agent blocking
// - Path traversal and injection attack prevention
//
// All security features are configurable and can be enabled/disabled
// based on deployment requirements while maintaining secure defaults.
package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/logging"
	"github.com/conneroisu/templar/internal/validation"
)

// contextKey represents a context key type for type safety
type contextKey string

// nonceContextKey is used to store CSP nonce values in request context
const nonceContextKey contextKey = "csp_nonce"

// SecurityConfig holds comprehensive security configuration for HTTP middleware
// and request processing. All security features can be individually configured
// or disabled based on deployment requirements.
type SecurityConfig struct {
	// CSP configures Content Security Policy headers and nonce generation
	CSP *CSPConfig
	// HSTS enables HTTP Strict Transport Security with configurable options
	HSTS *HSTSConfig
	// XFrameOptions sets X-Frame-Options header (DENY, SAMEORIGIN, ALLOW-FROM)
	XFrameOptions string
	// XContentTypeNoSniff enables X-Content-Type-Options: nosniff header
	XContentTypeNoSniff bool
	// XSSProtection configures X-XSS-Protection header behavior
	XSSProtection *XSSProtectionConfig
	// ReferrerPolicy sets Referrer-Policy header for referrer information control
	ReferrerPolicy string
	// PermissionsPolicy configures Permissions-Policy header for browser feature control
	PermissionsPolicy *PermissionsPolicyConfig
	// EnableNonce controls CSP nonce generation for script and style tags
	EnableNonce bool
	// AllowedOrigins lists origins permitted for CORS and WebSocket connections
	AllowedOrigins []string
	// BlockedUserAgents lists user agent patterns to reject (security scanners, etc.)
	BlockedUserAgents []string
	// RateLimiting configures request rate limiting and DoS protection
	RateLimiting *RateLimitConfig
	// Logger handles security event logging and audit trails
	Logger logging.Logger
}

// CSPConfig holds Content Security Policy configuration
type CSPConfig struct {
	DefaultSrc              []string
	ScriptSrc               []string
	StyleSrc                []string
	ImgSrc                  []string
	ConnectSrc              []string
	FontSrc                 []string
	ObjectSrc               []string
	MediaSrc                []string
	FrameSrc                []string
	ChildSrc                []string
	WorkerSrc               []string
	ManifestSrc             []string
	FrameAncestors          []string
	BaseURI                 []string
	FormAction              []string
	UpgradeInsecureRequests bool
	BlockAllMixedContent    bool
	RequireSRIFor           []string
	ReportURI               string
	ReportTo                string
}

// HSTSConfig holds HTTP Strict Transport Security configuration
type HSTSConfig struct {
	MaxAge            int
	IncludeSubDomains bool
	Preload           bool
}

// XSSProtectionConfig holds X-XSS-Protection configuration
type XSSProtectionConfig struct {
	Enabled   bool
	Mode      string // "block" or "report"
	ReportURI string
}

// PermissionsPolicyConfig holds Permissions Policy configuration
type PermissionsPolicyConfig struct {
	Geolocation       []string
	Camera            []string
	Microphone        []string
	Payment           []string
	USB               []string
	Accelerometer     []string
	Gyroscope         []string
	Magnetometer      []string
	Notifications     []string
	PersistentStorage []string
	Fullscreen        []string
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int
	BurstSize         int
	WindowSize        time.Duration
	Enabled           bool
}

// DefaultSecurityConfig returns a secure default configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		CSP: &CSPConfig{
			DefaultSrc:              []string{"'self'"},
			ScriptSrc:               []string{"'self'", "https://cdn.tailwindcss.com"},
			StyleSrc:                []string{"'self'"},
			ImgSrc:                  []string{"'self'", "data:", "blob:"},
			ConnectSrc:              []string{"'self'", "ws:", "wss:"},
			FontSrc:                 []string{"'self'"},
			ObjectSrc:               []string{"'none'"},
			MediaSrc:                []string{"'self'"},
			FrameSrc:                []string{"'self'"},
			ChildSrc:                []string{"'self'"},
			WorkerSrc:               []string{"'self'"},
			ManifestSrc:             []string{"'self'"},
			FrameAncestors:          []string{"'none'"},
			BaseURI:                 []string{"'self'"},
			FormAction:              []string{"'self'"},
			UpgradeInsecureRequests: false, // Set to true in production with HTTPS
			BlockAllMixedContent:    false, // Set to true in production with HTTPS
			RequireSRIFor:           []string{},
		},
		HSTS: &HSTSConfig{
			MaxAge:            31536000, // 1 year
			IncludeSubDomains: true,
			Preload:           false, // Only enable after testing
		},
		XFrameOptions:       "DENY",
		XContentTypeNoSniff: true,
		XSSProtection: &XSSProtectionConfig{
			Enabled: true,
			Mode:    "block",
		},
		ReferrerPolicy: "strict-origin-when-cross-origin",
		PermissionsPolicy: &PermissionsPolicyConfig{
			Geolocation:       []string{},
			Camera:            []string{},
			Microphone:        []string{},
			Payment:           []string{},
			USB:               []string{},
			Accelerometer:     []string{},
			Gyroscope:         []string{},
			Magnetometer:      []string{},
			Notifications:     []string{},
			PersistentStorage: []string{},
			Fullscreen:        []string{"'self'"},
		},
		EnableNonce:       true,
		AllowedOrigins:    []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		BlockedUserAgents: []string{},
		RateLimiting: &RateLimitConfig{
			RequestsPerMinute: 1000,
			BurstSize:         50,
			WindowSize:        time.Minute,
			Enabled:           true,
		},
	}
}

// DevelopmentSecurityConfig returns a more permissive config for development
func DevelopmentSecurityConfig() *SecurityConfig {
	config := DefaultSecurityConfig()

	// Allow WebSocket connections from any port for development
	config.CSP.ConnectSrc = append(config.CSP.ConnectSrc, "*")

	// Allow iframe embedding for development tools
	config.XFrameOptions = "SAMEORIGIN"
	config.CSP.FrameAncestors = []string{"'self'"}

	// Disable HSTS in development
	config.HSTS = nil

	// More permissive origins
	config.AllowedOrigins = append(config.AllowedOrigins,
		"http://localhost:8080", "http://127.0.0.1:8080",
		"http://localhost:3001", "http://127.0.0.1:3001")

	// Higher rate limits for development
	config.RateLimiting.RequestsPerMinute = 5000
	config.RateLimiting.BurstSize = 200

	// Enable nonce-based CSP for development too
	config.EnableNonce = true

	return config
}

// ProductionSecurityConfig returns a strict config for production
func ProductionSecurityConfig() *SecurityConfig {
	config := DefaultSecurityConfig()

	// Strict CSP for production - remove unsafe directives
	config.CSP.ScriptSrc = []string{"'self'"}
	config.CSP.StyleSrc = []string{"'self'"}
	config.CSP.UpgradeInsecureRequests = true
	config.CSP.BlockAllMixedContent = true
	config.CSP.RequireSRIFor = []string{"script", "style"}

	// Enable nonce for production to allow inline scripts/styles securely
	config.EnableNonce = true

	// Add CSP violation reporting
	config.CSP.ReportURI = "/api/csp-violation-report"

	// Enable HSTS preload for production
	config.HSTS.Preload = true

	// Strict frame options
	config.XFrameOptions = "DENY"
	config.CSP.FrameAncestors = []string{"'none'"}

	// Lower rate limits for production
	config.RateLimiting.RequestsPerMinute = 100
	config.RateLimiting.BurstSize = 20

	// No localhost origins in production
	config.AllowedOrigins = []string{}

	return config
}

// generateNonce generates a cryptographically secure random nonce
func generateNonce() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// GetNonceFromContext retrieves the CSP nonce from the request context
func GetNonceFromContext(ctx context.Context) string {
	if nonce, ok := ctx.Value(nonceContextKey).(string); ok {
		return nonce
	}
	return ""
}

// CSPViolationReport represents a CSP violation report
type CSPViolationReport struct {
	CSPReport struct {
		DocumentURI        string `json:"document-uri"`
		Referrer           string `json:"referrer"`
		ViolatedDirective  string `json:"violated-directive"`
		EffectiveDirective string `json:"effective-directive"`
		OriginalPolicy     string `json:"original-policy"`
		BlockedURI         string `json:"blocked-uri"`
		StatusCode         int    `json:"status-code"`
		LineNumber         int    `json:"line-number"`
		ColumnNumber       int    `json:"column-number"`
		SourceFile         string `json:"source-file"`
	} `json:"csp-report"`
}

// CSPViolationHandler handles CSP violation reports
func CSPViolationHandler(logger logging.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var report CSPViolationReport
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			if logger != nil {
				logger.Warn(r.Context(),
					errors.NewSecurityError("CSP_REPORT_PARSE_ERROR", "Failed to parse CSP violation report"),
					"CSP: Failed to parse violation report",
					"error", err.Error(),
					"ip", getClientIP(r))
			}
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Log the CSP violation
		if logger != nil {
			logger.Warn(r.Context(),
				errors.NewSecurityError("CSP_VIOLATION", "Content Security Policy violation detected"),
				"CSP: Policy violation detected",
				"document_uri", report.CSPReport.DocumentURI,
				"violated_directive", report.CSPReport.ViolatedDirective,
				"blocked_uri", report.CSPReport.BlockedURI,
				"source_file", report.CSPReport.SourceFile,
				"line_number", report.CSPReport.LineNumber,
				"ip", getClientIP(r))
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// SecurityMiddleware creates a security middleware with the given configuration
func SecurityMiddleware(secConfig *SecurityConfig) func(http.Handler) http.Handler {
	if secConfig == nil {
		secConfig = DefaultSecurityConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate nonce for this request if enabled
			var nonce string
			if secConfig.EnableNonce {
				var err error
				nonce, err = generateNonce()
				if err != nil {
					if secConfig.Logger != nil {
						secConfig.Logger.Error(r.Context(), err, "Failed to generate CSP nonce")
					}
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				// Add nonce to request context
				r = r.WithContext(context.WithValue(r.Context(), nonceContextKey, nonce))
			}

			// Apply security headers with nonce
			applySecurityHeaders(w, r, secConfig, nonce)

			// Check blocked user agents
			if err := validation.ValidateUserAgent(r.UserAgent(), secConfig.BlockedUserAgents); err != nil {
				if secConfig.Logger != nil {
					secConfig.Logger.Warn(r.Context(),
						errors.NewSecurityError("BLOCKED_USER_AGENT", "Blocked user agent attempted access"),
						"Security: Blocked user agent",
						"user_agent", r.UserAgent(),
						"ip", getClientIP(r))
				}
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Validate origin for non-GET requests
			if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
				origin := r.Header.Get("Origin")
				if origin == "" {
					// For same-origin requests, browser doesn't send Origin header
					// Check Referer as fallback
					referer := r.Header.Get("Referer")
					if referer != "" {
						if refererURL, err := url.Parse(referer); err == nil {
							origin = fmt.Sprintf("%s://%s", refererURL.Scheme, refererURL.Host)
						}
					}
				}
				if err := validation.ValidateOrigin(origin, secConfig.AllowedOrigins); err != nil {
					if secConfig.Logger != nil {
						secConfig.Logger.Warn(r.Context(),
							errors.NewSecurityError("INVALID_ORIGIN", "Invalid origin in request"),
							"Security: Invalid origin",
							"origin", r.Header.Get("Origin"),
							"referer", r.Header.Get("Referer"),
							"ip", getClientIP(r))
					}
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// applySecurityHeaders applies all configured security headers
func applySecurityHeaders(w http.ResponseWriter, r *http.Request, config *SecurityConfig, nonce string) {
	// Content Security Policy
	if config.CSP != nil {
		cspHeader := buildCSPHeader(config.CSP, nonce)
		w.Header().Set("Content-Security-Policy", cspHeader)
	}

	// HTTP Strict Transport Security
	if config.HSTS != nil && r.TLS != nil {
		hstsHeader := buildHSTSHeader(config.HSTS)
		w.Header().Set("Strict-Transport-Security", hstsHeader)
	}

	// X-Frame-Options
	if config.XFrameOptions != "" {
		w.Header().Set("X-Frame-Options", config.XFrameOptions)
	}

	// X-Content-Type-Options
	if config.XContentTypeNoSniff {
		w.Header().Set("X-Content-Type-Options", "nosniff")
	}

	// X-XSS-Protection
	if config.XSSProtection != nil {
		xssHeader := buildXSSProtectionHeader(config.XSSProtection)
		w.Header().Set("X-XSS-Protection", xssHeader)
	}

	// Referrer-Policy
	if config.ReferrerPolicy != "" {
		w.Header().Set("Referrer-Policy", config.ReferrerPolicy)
	}

	// Permissions-Policy
	if config.PermissionsPolicy != nil {
		permissionsHeader := buildPermissionsPolicyHeader(config.PermissionsPolicy)
		if permissionsHeader != "" {
			w.Header().Set("Permissions-Policy", permissionsHeader)
		}
	}

	// Additional security headers
	w.Header().Set("X-DNS-Prefetch-Control", "off")
	w.Header().Set("X-Download-Options", "noopen")
	w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
	w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
	w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
}

// buildCSPHeader constructs the Content-Security-Policy header value
func buildCSPHeader(csp *CSPConfig, nonce string) string {
	var directives []string

	// Helper function to add directive with optional nonce support
	addDirective := func(name string, values []string) {
		if len(values) > 0 {
			// Filter out unsafe directives if nonce is provided
			if nonce != "" && (name == "script-src" || name == "style-src") {
				filteredValues := []string{}
				for _, value := range values {
					// Remove unsafe directives when using nonce
					if value != "'unsafe-inline'" && value != "'unsafe-eval'" {
						filteredValues = append(filteredValues, value)
					}
				}
				// Add nonce for secure inline scripts/styles
				nonceValue := fmt.Sprintf("'nonce-%s'", nonce)
				filteredValues = append(filteredValues, nonceValue)
				values = filteredValues
			}
			directives = append(directives, fmt.Sprintf("%s %s", name, strings.Join(values, " ")))
		}
	}

	addDirective("default-src", csp.DefaultSrc)
	addDirective("script-src", csp.ScriptSrc)
	addDirective("style-src", csp.StyleSrc)
	addDirective("img-src", csp.ImgSrc)
	addDirective("connect-src", csp.ConnectSrc)
	addDirective("font-src", csp.FontSrc)
	addDirective("object-src", csp.ObjectSrc)
	addDirective("media-src", csp.MediaSrc)
	addDirective("frame-src", csp.FrameSrc)
	addDirective("child-src", csp.ChildSrc)
	addDirective("worker-src", csp.WorkerSrc)
	addDirective("manifest-src", csp.ManifestSrc)
	addDirective("frame-ancestors", csp.FrameAncestors)
	addDirective("base-uri", csp.BaseURI)
	addDirective("form-action", csp.FormAction)

	if csp.UpgradeInsecureRequests {
		directives = append(directives, "upgrade-insecure-requests")
	}

	if csp.BlockAllMixedContent {
		directives = append(directives, "block-all-mixed-content")
	}

	if len(csp.RequireSRIFor) > 0 {
		directives = append(directives, fmt.Sprintf("require-sri-for %s", strings.Join(csp.RequireSRIFor, " ")))
	}

	if csp.ReportURI != "" {
		directives = append(directives, fmt.Sprintf("report-uri %s", csp.ReportURI))
	}

	if csp.ReportTo != "" {
		directives = append(directives, fmt.Sprintf("report-to %s", csp.ReportTo))
	}

	return strings.Join(directives, "; ")
}

// buildHSTSHeader constructs the Strict-Transport-Security header value
func buildHSTSHeader(hsts *HSTSConfig) string {
	header := fmt.Sprintf("max-age=%d", hsts.MaxAge)

	if hsts.IncludeSubDomains {
		header += "; includeSubDomains"
	}

	if hsts.Preload {
		header += "; preload"
	}

	return header
}

// buildXSSProtectionHeader constructs the X-XSS-Protection header value
func buildXSSProtectionHeader(xss *XSSProtectionConfig) string {
	if !xss.Enabled {
		return "0"
	}

	header := "1"

	if xss.Mode == "block" {
		header += "; mode=block"
	} else if xss.Mode == "report" && xss.ReportURI != "" {
		header += fmt.Sprintf("; report=%s", xss.ReportURI)
	}

	return header
}

// buildPermissionsPolicyHeader constructs the Permissions-Policy header value
func buildPermissionsPolicyHeader(pp *PermissionsPolicyConfig) string {
	var policies []string

	// Helper function to add policy
	addPolicy := func(name string, values []string) {
		if len(values) == 0 {
			policies = append(policies, fmt.Sprintf("%s=()", name))
		} else {
			policies = append(policies, fmt.Sprintf("%s=(%s)", name, strings.Join(values, " ")))
		}
	}

	addPolicy("geolocation", pp.Geolocation)
	addPolicy("camera", pp.Camera)
	addPolicy("microphone", pp.Microphone)
	addPolicy("payment", pp.Payment)
	addPolicy("usb", pp.USB)
	addPolicy("accelerometer", pp.Accelerometer)
	addPolicy("gyroscope", pp.Gyroscope)
	addPolicy("magnetometer", pp.Magnetometer)
	addPolicy("notifications", pp.Notifications)
	addPolicy("persistent-storage", pp.PersistentStorage)
	addPolicy("fullscreen", pp.Fullscreen)

	return strings.Join(policies, ", ")
}

// Note: isBlockedUserAgent and isValidOrigin functions have been replaced
// with centralized validation functions in the validation package.

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (proxy/load balancer)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the list
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colonPos := strings.LastIndex(ip, ":"); colonPos != -1 {
		ip = ip[:colonPos]
	}

	return ip
}

// SecurityConfigFromAppConfig creates security config from application config
func SecurityConfigFromAppConfig(cfg *config.Config) *SecurityConfig {
	if cfg.Server.Environment == "production" {
		return ProductionSecurityConfig()
	} else if cfg.Server.Environment == "development" {
		return DevelopmentSecurityConfig()
	}

	return DefaultSecurityConfig()
}

// AuthMiddleware provides authentication for the development server
func AuthMiddleware(authConfig *config.AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication if disabled
			if !authConfig.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Allow localhost bypass if configured
			clientIP := getClientIP(r)
			if authConfig.LocalhostBypass && isLocalhost(clientIP) {
				next.ServeHTTP(w, r)
				return
			}

			// Check IP allowlist
			if len(authConfig.AllowedIPs) > 0 && !isIPAllowed(clientIP, authConfig.AllowedIPs) {
				http.Error(w, "Access denied from this IP", http.StatusForbidden)
				return
			}

			// Require authentication for non-localhost if configured
			if authConfig.RequireAuth && !isLocalhost(clientIP) {
				if !authenticateRequest(r, authConfig) {
					requireAuth(w, authConfig.Mode)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// authenticateRequest validates the request authentication
func authenticateRequest(r *http.Request, authConfig *config.AuthConfig) bool {
	switch authConfig.Mode {
	case "token":
		return authenticateToken(r, authConfig.Token)
	case "basic":
		return authenticateBasic(r, authConfig.Username, authConfig.Password)
	case "none":
		return true
	default:
		return false
	}
}

// authenticateToken validates token-based authentication
func authenticateToken(r *http.Request, expectedToken string) bool {
	if expectedToken == "" {
		return false
	}

	// Check Authorization header: "Bearer <token>"
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		return token == expectedToken
	}

	// Check query parameter: "?token=<token>"
	queryToken := r.URL.Query().Get("token")
	if queryToken != "" {
		return queryToken == expectedToken
	}

	return false
}

// authenticateBasic validates basic authentication
func authenticateBasic(r *http.Request, expectedUsername, expectedPassword string) bool {
	if expectedUsername == "" || expectedPassword == "" {
		return false
	}

	username, password, ok := r.BasicAuth()
	if !ok {
		return false
	}

	return username == expectedUsername && password == expectedPassword
}

// requireAuth sends authentication required response
func requireAuth(w http.ResponseWriter, mode string) {
	switch mode {
	case "basic":
		w.Header().Set("WWW-Authenticate", `Basic realm="Templar Development Server"`)
		http.Error(w, "Authentication required", http.StatusUnauthorized)
	case "token":
		w.Header().Set("WWW-Authenticate", `Bearer realm="Templar Development Server"`)
		http.Error(w, "Authentication required - provide Bearer token", http.StatusUnauthorized)
	default:
		http.Error(w, "Authentication required", http.StatusUnauthorized)
	}
}

// isLocalhost checks if the IP address is localhost
func isLocalhost(ip string) bool {
	// Remove IPv6 brackets if present
	ip = strings.Trim(ip, "[]")

	return ip == "127.0.0.1" || ip == "::1" || ip == "localhost"
}

// isIPAllowed checks if the IP is in the allowed list
func isIPAllowed(clientIP string, allowedIPs []string) bool {
	// Remove IPv6 brackets if present
	clientIP = strings.Trim(clientIP, "[]")

	for _, allowedIP := range allowedIPs {
		// Simple exact match - could be enhanced with CIDR support
		if clientIP == allowedIP {
			return true
		}
	}
	return false
}
