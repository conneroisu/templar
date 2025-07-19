package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/logging"
)

// SecurityConfig holds security configuration
type SecurityConfig struct {
	CSP                *CSPConfig
	HSTS               *HSTSConfig
	XFrameOptions      string
	XContentTypeNoSniff bool
	XSSProtection      *XSSProtectionConfig
	ReferrerPolicy     string
	PermissionsPolicy  *PermissionsPolicyConfig
	EnableNonce        bool
	AllowedOrigins     []string
	BlockedUserAgents  []string
	RateLimiting       *RateLimitConfig
	Logger             logging.Logger
}

// CSPConfig holds Content Security Policy configuration
type CSPConfig struct {
	DefaultSrc    []string
	ScriptSrc     []string
	StyleSrc      []string
	ImgSrc        []string
	ConnectSrc    []string
	FontSrc       []string
	ObjectSrc     []string
	MediaSrc      []string
	FrameSrc      []string
	ChildSrc      []string
	WorkerSrc     []string
	ManifestSrc   []string
	FrameAncestors []string
	BaseURI       []string
	FormAction    []string
	UpgradeInsecureRequests bool
	BlockAllMixedContent    bool
	RequireSRIFor          []string
	ReportURI              string
	ReportTo               string
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
	Geolocation      []string
	Camera           []string
	Microphone       []string
	Payment          []string
	USB              []string
	Accelerometer    []string
	Gyroscope        []string
	Magnetometer     []string
	Notifications    []string
	PersistentStorage []string
	Fullscreen       []string
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
			DefaultSrc:  []string{"'self'"},
			ScriptSrc:   []string{"'self'", "'unsafe-inline'", "'unsafe-eval'"},
			StyleSrc:    []string{"'self'", "'unsafe-inline'"},
			ImgSrc:      []string{"'self'", "data:", "blob:"},
			ConnectSrc:  []string{"'self'", "ws:", "wss:"},
			FontSrc:     []string{"'self'"},
			ObjectSrc:   []string{"'none'"},
			MediaSrc:    []string{"'self'"},
			FrameSrc:    []string{"'self'"},
			ChildSrc:    []string{"'self'"},
			WorkerSrc:   []string{"'self'"},
			ManifestSrc: []string{"'self'"},
			FrameAncestors: []string{"'none'"},
			BaseURI:     []string{"'self'"},
			FormAction:  []string{"'self'"},
			UpgradeInsecureRequests: false, // Set to true in production with HTTPS
			BlockAllMixedContent:    false, // Set to true in production with HTTPS
			RequireSRIFor:          []string{},
		},
		HSTS: &HSTSConfig{
			MaxAge:            31536000, // 1 year
			IncludeSubDomains: true,
			Preload:           false, // Only enable after testing
		},
		XFrameOptions:      "DENY",
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
	
	// Allow more permissive CSP for development
	config.CSP.ScriptSrc = append(config.CSP.ScriptSrc, "'unsafe-eval'", "'unsafe-inline'")
	config.CSP.StyleSrc = append(config.CSP.StyleSrc, "'unsafe-inline'")
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
	
	return config
}

// ProductionSecurityConfig returns a strict config for production
func ProductionSecurityConfig() *SecurityConfig {
	config := DefaultSecurityConfig()
	
	// Strict CSP for production
	config.CSP.ScriptSrc = []string{"'self'"}
	config.CSP.StyleSrc = []string{"'self'"}
	config.CSP.UpgradeInsecureRequests = true
	config.CSP.BlockAllMixedContent = true
	config.CSP.RequireSRIFor = []string{"script", "style"}
	
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

// SecurityMiddleware creates a security middleware with the given configuration
func SecurityMiddleware(secConfig *SecurityConfig) func(http.Handler) http.Handler {
	if secConfig == nil {
		secConfig = DefaultSecurityConfig()
	}
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Apply security headers
			applySecurityHeaders(w, r, secConfig)
			
			// Check blocked user agents
			if isBlockedUserAgent(r.UserAgent(), secConfig.BlockedUserAgents) {
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
				if !isValidOrigin(r, secConfig.AllowedOrigins) {
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
func applySecurityHeaders(w http.ResponseWriter, r *http.Request, config *SecurityConfig) {
	// Content Security Policy
	if config.CSP != nil {
		cspHeader := buildCSPHeader(config.CSP, config.EnableNonce)
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
func buildCSPHeader(csp *CSPConfig, enableNonce bool) string {
	var directives []string
	
	// Helper function to add directive
	addDirective := func(name string, values []string) {
		if len(values) > 0 {
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

// isBlockedUserAgent checks if the user agent is in the blocked list
func isBlockedUserAgent(userAgent string, blockedAgents []string) bool {
	if userAgent == "" {
		return false
	}
	
	userAgentLower := strings.ToLower(userAgent)
	for _, blocked := range blockedAgents {
		if strings.Contains(userAgentLower, strings.ToLower(blocked)) {
			return true
		}
	}
	
	return false
}

// isValidOrigin validates the request origin against allowed origins
func isValidOrigin(r *http.Request, allowedOrigins []string) bool {
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
	
	if origin == "" {
		return false
	}
	
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	
	return false
}

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