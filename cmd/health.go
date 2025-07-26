package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

// HealthStatus represents the health check response.
type HealthStatus struct {
	Status    string           `json:"status"`
	Timestamp time.Time        `json:"timestamp"`
	Checks    map[string]Check `json:"checks"`
	Overall   bool             `json:"overall"`
}

// Check represents an individual health check result.
type Check struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Healthy bool   `json:"healthy"`
}

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check the health status of Templar server",
	Long: `Performs comprehensive health checks on the Templar server including:
- HTTP server responsiveness
- File system access
- Build tool availability
- Component directory access

This command is used by Docker health checks and deployment readiness probes.`,
	RunE: runHealthCheck,
}

var (
	healthPort    int
	healthHost    string
	healthTimeout time.Duration
	healthVerbose bool
)

func init() {
	rootCmd.AddCommand(healthCmd)

	healthCmd.Flags().IntVarP(&healthPort, "port", "p", 8080, "Port to check for HTTP server")
	healthCmd.Flags().
		StringVarP(&healthHost, "host", "H", "localhost", "Host to check for HTTP server")
	healthCmd.Flags().
		DurationVarP(&healthTimeout, "timeout", "t", 3*time.Second, "Timeout for health checks")
	healthCmd.Flags().BoolVarP(&healthVerbose, "verbose", "v", false, "Verbose health check output")
}

func runHealthCheck(cmd *cobra.Command, args []string) error {
	status := &HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Checks:    make(map[string]Check),
		Overall:   true,
	}

	// Perform all health checks
	checkHTTPServer(status)
	checkFileSystemAccess(status)
	checkBuildTools(status)
	checkComponentDirectories(status)

	// Output results
	if healthVerbose {
		output, _ := json.MarshalIndent(status, "", "  ")
		fmt.Println(string(output))
	} else {
		if status.Overall {
			fmt.Println("✅ All health checks passed")
		} else {
			fmt.Println("❌ Health checks failed")
			for name, check := range status.Checks {
				if !check.Healthy {
					fmt.Printf("  - %s: %s\n", name, check.Message)
				}
			}
		}
	}

	if !status.Overall {
		return errors.New("health checks failed")
	}

	return nil
}

// checkHTTPServer verifies the HTTP server is responding.
func checkHTTPServer(status *HealthStatus) {
	client := &http.Client{
		Timeout: healthTimeout,
	}

	url := fmt.Sprintf("http://%s:%d/health", healthHost, healthPort)
	resp, err := client.Get(url)

	if err != nil {
		status.Checks["http_server"] = Check{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Failed to connect to server: %v", err),
			Healthy: false,
		}
		status.Overall = false

		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		status.Checks["http_server"] = Check{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Server returned status %d", resp.StatusCode),
			Healthy: false,
		}
		status.Overall = false

		return
	}

	status.Checks["http_server"] = Check{
		Status:  "healthy",
		Message: "HTTP server responding",
		Healthy: true,
	}
}

// checkFileSystemAccess verifies basic file system access.
func checkFileSystemAccess(status *HealthStatus) {
	// Check current directory access
	_, err := os.Getwd()
	if err != nil {
		status.Checks["filesystem"] = Check{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Cannot access current directory: %v", err),
			Healthy: false,
		}
		status.Overall = false

		return
	}

	// Check if we can create temporary files
	tmpFile, err := os.CreateTemp("", "templar-health-*")
	if err != nil {
		status.Checks["filesystem"] = Check{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Cannot create temporary files: %v", err),
			Healthy: false,
		}
		status.Overall = false

		return
	}
	if err := tmpFile.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to close temp file: %v\n", err)
	}
	if err := os.Remove(tmpFile.Name()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to remove temp file: %v\n", err)
	}

	status.Checks["filesystem"] = Check{
		Status:  "healthy",
		Message: "File system access working",
		Healthy: true,
	}
}

// checkBuildTools verifies required build tools are available.
func checkBuildTools(status *HealthStatus) {
	// Check for templ binary
	_, err := exec.LookPath("templ")
	if err != nil {
		status.Checks["build_tools"] = Check{
			Status:  "warning",
			Message: "templ binary not found in PATH (optional for runtime)",
			Healthy: true, // Not critical for health check to pass
		}
	} else {
		status.Checks["build_tools"] = Check{
			Status:  "healthy",
			Message: "Build tools available",
			Healthy: true,
		}
	}
}

// checkComponentDirectories verifies component directories are accessible.
func checkComponentDirectories(status *HealthStatus) {
	// Check common component directories
	commonDirs := []string{"./components", "./views", "./examples"}
	accessibleDirs := 0

	for _, dir := range commonDirs {
		if _, err := os.Stat(dir); err == nil {
			accessibleDirs++
		}
	}

	if accessibleDirs == 0 {
		status.Checks["component_dirs"] = Check{
			Status:  "warning",
			Message: "No standard component directories found (components/, views/, examples/)",
			Healthy: true, // Not critical - directories might not exist yet
		}
	} else {
		status.Checks["component_dirs"] = Check{
			Status:  "healthy",
			Message: fmt.Sprintf("Found %d component directories", accessibleDirs),
			Healthy: true,
		}
	}
}
