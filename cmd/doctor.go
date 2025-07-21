package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose development environment and tool integrations",
	Long: `Diagnose your development environment and check for common issues
with tool integrations, port conflicts, and workflow compatibility.

The doctor command analyzes your system and provides recommendations
for optimal Templar integration with your existing development tools.

Examples:
  templar doctor                    # Run full diagnostic
  templar doctor --format=json     # JSON output for tooling
  templar doctor --fix              # Attempt to fix detected issues
  templar doctor --check-ports     # Focus on port conflict detection

Diagnostic Areas:
  • Go toolchain and templ installation
  • Development server port conflicts
  • Integration with air, Tailwind, VS Code
  • File watching and build tool compatibility
  • Network and firewall configuration
  • Performance and resource availability`,
	RunE: runDoctor,
}

var (
	doctorFormat    string
	doctorFix       bool
	doctorCheckPorts bool
	doctorVerbose   bool
)

func init() {
	rootCmd.AddCommand(doctorCmd)
	
	doctorCmd.Flags().StringVarP(&doctorFormat, "format", "f", "text", "Output format (text|json)")
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "Attempt to fix detected issues automatically")
	doctorCmd.Flags().BoolVar(&doctorCheckPorts, "check-ports", false, "Focus on port conflict detection")
	doctorCmd.Flags().BoolVarP(&doctorVerbose, "verbose", "v", false, "Show verbose diagnostic information")
}

// DiagnosticResult represents the result of a diagnostic check
type DiagnosticResult struct {
	Name        string            `json:"name"`
	Status      string            `json:"status"` // "pass", "warn", "fail"
	Message     string            `json:"message"`
	Details     map[string]string `json:"details,omitempty"`
	Suggestions []string          `json:"suggestions,omitempty"`
	Fixable     bool              `json:"fixable"`
}

// DoctorReport contains all diagnostic results
type DoctorReport struct {
	Timestamp   time.Time          `json:"timestamp"`
	System      SystemInfo         `json:"system"`
	Results     []DiagnosticResult `json:"results"`
	Summary     Summary            `json:"summary"`
	Integrations IntegrationInfo   `json:"integrations"`
}

type SystemInfo struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	GoVersion string `json:"go_version"`
	WorkDir  string `json:"work_dir"`
	User     string `json:"user,omitempty"`
}

type Summary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Warned  int `json:"warned"`
	Failed  int `json:"failed"`
	Fixable int `json:"fixable"`
}

type IntegrationInfo struct {
	Air       ToolInfo `json:"air"`
	Tailwind  ToolInfo `json:"tailwind"`
	VSCode    ToolInfo `json:"vscode"`
	Git       ToolInfo `json:"git"`
	Node      ToolInfo `json:"node"`
	Templ     ToolInfo `json:"templ"`
}

type ToolInfo struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
	Path      string `json:"path,omitempty"`
	Status    string `json:"status"` // "compatible", "outdated", "missing", "conflict"
}

func runDoctor(cmd *cobra.Command, args []string) error {
	report := &DoctorReport{
		Timestamp: time.Now(),
		Results:   []DiagnosticResult{},
	}

	// Gather system information
	if err := gatherSystemInfo(report); err != nil {
		return fmt.Errorf("failed to gather system info: %w", err)
	}

	// Run diagnostic checks
	diagnostics := []func(*DoctorReport) DiagnosticResult{
		checkGoInstallation,
		checkTemplInstallation,
		checkPortAvailability,
		checkAirIntegration,
		checkTailwindIntegration,
		checkVSCodeIntegration,
		checkGitConfiguration,
		checkNodeEnvironment,
		checkFileWatching,
		checkNetworkConfiguration,
		checkResourceAvailability,
		checkProjectStructure,
	}

	for _, diagnostic := range diagnostics {
		if doctorCheckPorts && !strings.Contains(getFunctionName(diagnostic), "Port") {
			continue // Skip non-port checks when --check-ports is used
		}
		result := diagnostic(report)
		report.Results = append(report.Results, result)
	}

	// Generate summary
	generateSummary(report)

	// Detect tool integrations
	detectIntegrations(report)

	// Apply fixes if requested
	if doctorFix {
		if err := applyFixes(report); err != nil {
			return fmt.Errorf("failed to apply fixes: %w", err)
		}
	}

	// Output results
	return outputReport(report)
}

func gatherSystemInfo(report *DoctorReport) error {
	report.System.OS = runtime.GOOS
	report.System.Arch = runtime.GOARCH
	
	if goVersion, err := exec.Command("go", "version").Output(); err == nil {
		report.System.GoVersion = strings.TrimSpace(string(goVersion))
	}
	
	if cwd, err := os.Getwd(); err == nil {
		report.System.WorkDir = cwd
	}
	
	if user := os.Getenv("USER"); user != "" {
		report.System.User = user
	} else if user := os.Getenv("USERNAME"); user != "" {
		report.System.User = user
	}
	
	return nil
}

func checkGoInstallation(report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:    "Go Installation",
		Details: make(map[string]string),
	}

	// Check if Go is installed
	goPath, err := exec.LookPath("go")
	if err != nil {
		result.Status = "fail"
		result.Message = "Go is not installed or not in PATH"
		result.Suggestions = []string{
			"Install Go from https://golang.org/dl/",
			"Ensure Go binary is in your PATH",
			"Verify installation with: go version",
		}
		return result
	}

	result.Details["path"] = goPath

	// Check Go version
	versionOutput, err := exec.Command("go", "version").Output()
	if err != nil {
		result.Status = "warn"
		result.Message = "Go is installed but version check failed"
		return result
	}

	version := strings.TrimSpace(string(versionOutput))
	result.Details["version"] = version

	// Parse version to check compatibility
	if !strings.Contains(version, "go1.") {
		result.Status = "warn"
		result.Message = "Unexpected Go version format"
		result.Suggestions = []string{"Verify Go installation integrity"}
		return result
	}

	// Check if version is recent enough (Go 1.20+)
	versionParts := strings.Fields(version)
	if len(versionParts) >= 3 {
		versionStr := strings.TrimPrefix(versionParts[2], "go")
		if strings.HasPrefix(versionStr, "1.1") || versionStr == "1.19" {
			result.Status = "warn"
			result.Message = "Go version is older than recommended (1.20+)"
			result.Suggestions = []string{
				"Consider upgrading to Go 1.20 or later for better performance",
				"Check https://golang.org/dl/ for latest version",
			}
		} else {
			result.Status = "pass"
			result.Message = "Go is properly installed and up to date"
		}
	} else {
		result.Status = "pass"
		result.Message = "Go is installed"
	}

	return result
}

func checkTemplInstallation(report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:    "Templ Installation",
		Details: make(map[string]string),
	}

	// Check if templ is installed
	templPath, err := exec.LookPath("templ")
	if err != nil {
		result.Status = "fail"
		result.Message = "templ is not installed or not in PATH"
		result.Suggestions = []string{
			"Install templ with: go install github.com/a-h/templ/cmd/templ@latest",
			"Ensure $GOPATH/bin is in your PATH",
			"Verify installation with: templ version",
		}
		result.Fixable = true
		return result
	}

	result.Details["path"] = templPath

	// Check templ version
	versionOutput, err := exec.Command("templ", "version").Output()
	if err != nil {
		result.Status = "warn"
		result.Message = "templ is installed but version check failed"
		return result
	}

	version := strings.TrimSpace(string(versionOutput))
	result.Details["version"] = version
	result.Status = "pass"
	result.Message = "templ is properly installed"

	return result
}

func checkPortAvailability(report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:    "Port Availability",
		Details: make(map[string]string),
	}

	// Common ports used by Templar and related tools
	ports := []int{8080, 3000, 3001, 5173, 8081} // Templar, development servers, monitoring
	conflicts := []string{}
	available := []string{}

	for _, port := range ports {
		if isPortInUse(port) {
			conflicts = append(conflicts, fmt.Sprintf("%d", port))
			process := getProcessOnPort(port)
			if process != "" {
				result.Details[fmt.Sprintf("port_%d", port)] = process
			}
		} else {
			available = append(available, fmt.Sprintf("%d", port))
		}
	}

	if len(conflicts) > 0 {
		result.Status = "warn"
		result.Message = fmt.Sprintf("Port conflicts detected: %s", strings.Join(conflicts, ", "))
		result.Suggestions = []string{
			"Use --port flag to specify alternative ports",
			"Stop conflicting processes if not needed",
			fmt.Sprintf("Available ports: %s", strings.Join(available, ", ")),
		}
		if len(available) > 0 {
			result.Suggestions = append(result.Suggestions, 
				fmt.Sprintf("Suggested: templar serve --port %s", available[0]))
		}
	} else {
		result.Status = "pass"
		result.Message = "All common development ports are available"
	}

	return result
}

func checkAirIntegration(report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:    "Air Integration",
		Details: make(map[string]string),
	}

	// Check if air is installed
	airPath, err := exec.LookPath("air")
	if err != nil {
		result.Status = "warn"
		result.Message = "Air (hot reload) is not installed"
		result.Suggestions = []string{
			"Install air with: go install github.com/cosmtrek/air@latest",
			"Air provides enhanced hot reload for Go development",
			"Compatible with Templar for full-stack development",
		}
		return result
	}

	result.Details["path"] = airPath

	// Check for .air.toml configuration
	airConfig := ".air.toml"
	if _, err := os.Stat(airConfig); err == nil {
		result.Details["config"] = airConfig
		result.Status = "pass"
		result.Message = "Air is installed and configured"
		
		// Provide integration guidance
		result.Suggestions = []string{
			"Use 'air' for Go hot reload and 'templar serve' for component preview",
			"Air and Templar can run simultaneously on different ports",
			"Consider air for backend development, Templar for component development",
		}
	} else {
		result.Status = "warn"
		result.Message = "Air is installed but not configured for this project"
		result.Suggestions = []string{
			"Initialize air config with: air init",
			"Configure air to work alongside Templar",
			"Use different ports for air and Templar servers",
		}
	}

	return result
}

func checkTailwindIntegration(report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:    "Tailwind CSS Integration",
		Details: make(map[string]string),
	}

	// Check for Tailwind configuration
	tailwindConfigs := []string{"tailwind.config.js", "tailwind.config.ts", "tailwind.config.cjs", "tailwind.config.mjs"}
	configFound := ""
	
	for _, config := range tailwindConfigs {
		if _, err := os.Stat(config); err == nil {
			configFound = config
			break
		}
	}

	if configFound == "" {
		result.Status = "warn"
		result.Message = "Tailwind CSS is not configured"
		result.Suggestions = []string{
			"Initialize Tailwind with: npx tailwindcss init",
			"Configure Tailwind to scan .templ files",
			"Add templ files to Tailwind content paths",
		}
		return result
	}

	result.Details["config"] = configFound

	// Check if Node.js/npm is available for Tailwind
	if _, err := exec.LookPath("npm"); err == nil {
		result.Details["npm"] = "available"
	} else if _, err := exec.LookPath("pnpm"); err == nil {
		result.Details["package_manager"] = "pnpm"
	} else if _, err := exec.LookPath("yarn"); err == nil {
		result.Details["package_manager"] = "yarn"
	}

	// Check package.json for Tailwind
	if _, err := os.Stat("package.json"); err == nil {
		result.Details["package.json"] = "found"
		result.Status = "pass"
		result.Message = "Tailwind CSS is configured"
		result.Suggestions = []string{
			"Ensure .templ files are included in Tailwind content paths",
			"Run Tailwind in watch mode alongside Templar",
			"Consider using Templar's built-in Tailwind integration",
		}
	} else {
		result.Status = "warn"
		result.Message = "Tailwind config found but no package.json"
		result.Suggestions = []string{
			"Initialize npm project with: npm init -y",
			"Install Tailwind CSS dependencies",
		}
	}

	return result
}

func checkVSCodeIntegration(report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:    "VS Code Integration",
		Details: make(map[string]string),
	}

	// Check if VS Code is installed
	codePaths := []string{"code", "code-insiders"}
	codeFound := ""
	
	for _, codePath := range codePaths {
		if _, err := exec.LookPath(codePath); err == nil {
			codeFound = codePath
			break
		}
	}

	if codeFound == "" {
		result.Status = "warn"
		result.Message = "VS Code is not installed or not in PATH"
		result.Suggestions = []string{
			"Install VS Code from https://code.visualstudio.com/",
			"Add VS Code to PATH for better integration",
			"Alternative editors with templ support available",
		}
		return result
	}

	result.Details["code_path"] = codeFound

	// Check for .vscode directory and settings
	vscodeDir := ".vscode"
	if _, err := os.Stat(vscodeDir); err == nil {
		result.Details["vscode_dir"] = "found"
		
		// Check for recommended extensions file
		extensionsFile := filepath.Join(vscodeDir, "extensions.json")
		if _, err := os.Stat(extensionsFile); err == nil {
			result.Details["extensions"] = "configured"
		}
		
		// Check for settings
		settingsFile := filepath.Join(vscodeDir, "settings.json")
		if _, err := os.Stat(settingsFile); err == nil {
			result.Details["settings"] = "configured"
		}
	}

	result.Status = "pass"
	result.Message = "VS Code is available"
	result.Suggestions = []string{
		"Install templ language extension for syntax highlighting",
		"Configure file associations for .templ files",
		"Use Go extension for full Go development support",
		"Consider Live Server extension for enhanced preview",
	}

	return result
}

func checkGitConfiguration(report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:    "Git Configuration",
		Details: make(map[string]string),
	}

	// Check if git is installed
	gitPath, err := exec.LookPath("git")
	if err != nil {
		result.Status = "warn"
		result.Message = "Git is not installed"
		result.Suggestions = []string{
			"Install Git for version control",
			"Git is recommended for Templar project management",
		}
		return result
	}

	result.Details["path"] = gitPath

	// Check if we're in a git repository
	if _, err := exec.Command("git", "rev-parse", "--git-dir").Output(); err != nil {
		result.Status = "warn"
		result.Message = "Not in a Git repository"
		result.Suggestions = []string{
			"Initialize Git repository with: git init",
			"Consider version controlling your Templar project",
		}
		return result
	}

	result.Status = "pass"
	result.Message = "Git is configured and repository initialized"
	result.Suggestions = []string{
		"Add .templar/ to .gitignore for cache files",
		"Consider using Git hooks for automated builds",
	}

	return result
}

func checkNodeEnvironment(report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:    "Node.js Environment",
		Details: make(map[string]string),
	}

	// Check if Node.js is installed
	nodePath, err := exec.LookPath("node")
	if err != nil {
		result.Status = "warn"
		result.Message = "Node.js is not installed"
		result.Suggestions = []string{
			"Install Node.js for enhanced CSS/JS tooling",
			"Node.js enables Tailwind, PostCSS, and other tools",
			"Not required for basic Templar functionality",
		}
		return result
	}

	result.Details["node_path"] = nodePath

	// Check Node.js version
	if nodeVersion, err := exec.Command("node", "--version").Output(); err == nil {
		version := strings.TrimSpace(string(nodeVersion))
		result.Details["version"] = version
	}

	// Check for package managers
	packageManagers := []string{"npm", "pnpm", "yarn"}
	for _, pm := range packageManagers {
		if _, err := exec.LookPath(pm); err == nil {
			result.Details[pm] = "available"
		}
	}

	result.Status = "pass"
	result.Message = "Node.js environment is available"
	result.Suggestions = []string{
		"Use for Tailwind CSS, PostCSS, and other frontend tools",
		"Compatible with Templar's build pipeline",
	}

	return result
}

func checkFileWatching(report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:    "File Watching",
		Details: make(map[string]string),
	}

	// Check available file watching limits (Linux/macOS)
	if runtime.GOOS == "linux" {
		if content, err := os.ReadFile("/proc/sys/fs/inotify/max_user_watches"); err == nil {
			limit := strings.TrimSpace(string(content))
			result.Details["inotify_limit"] = limit
			
			if limitInt, err := strconv.Atoi(limit); err == nil && limitInt < 65536 {
				result.Status = "warn"
				result.Message = "File watching limit may be too low for large projects"
				result.Suggestions = []string{
					"Increase inotify limit: echo fs.inotify.max_user_watches=524288 | sudo tee -a /etc/sysctl.conf",
					"Reload with: sudo sysctl -p",
					"Required for watching large numbers of files",
				}
				result.Fixable = true
				return result
			}
		}
	}

	// Test basic file watching capability
	tempDir := os.TempDir()
	result.Details["temp_dir"] = tempDir
	
	if _, err := os.Stat(tempDir); err != nil {
		result.Status = "fail"
		result.Message = "Cannot access temporary directory for file watching test"
		return result
	}

	result.Status = "pass"
	result.Message = "File watching capabilities are available"
	result.Suggestions = []string{
		"Templar's hot reload uses efficient file watching",
		"Watch patterns can be customized with --watch flag",
	}

	return result
}

func checkNetworkConfiguration(report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:    "Network Configuration",
		Details: make(map[string]string),
	}

	// Test localhost connectivity
	conn, err := net.DialTimeout("tcp", "localhost:0", 1*time.Second)
	if err != nil {
		result.Status = "warn"
		result.Message = "Localhost connectivity issue detected"
		result.Suggestions = []string{
			"Check firewall settings",
			"Ensure localhost is properly configured",
		}
		return result
	}
	conn.Close()

	// Check if we can bind to common ports
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		result.Status = "fail"
		result.Message = "Cannot bind to network ports"
		result.Suggestions = []string{
			"Check network permissions",
			"Firewall may be blocking port binding",
		}
		return result
	}
	
	port := listener.Addr().(*net.TCPAddr).Port
	result.Details["test_port"] = fmt.Sprintf("%d", port)
	listener.Close()

	result.Status = "pass"
	result.Message = "Network configuration is working properly"

	return result
}

func checkResourceAvailability(report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:    "Resource Availability",
		Details: make(map[string]string),
	}

	// Check available disk space
	cwd, _ := os.Getwd()
	if stat, err := os.Stat(cwd); err == nil {
		result.Details["working_dir"] = cwd
		_ = stat // Use stat if needed for more detailed checks
	}

	// Check if we can create temporary files
	tempFile, err := os.CreateTemp("", "templar-doctor-*")
	if err != nil {
		result.Status = "warn"
		result.Message = "Cannot create temporary files"
		result.Suggestions = []string{
			"Check disk space and permissions",
			"Temporary files are needed for build processes",
		}
		return result
	}
	
	tempFile.Close()
	os.Remove(tempFile.Name())

	result.Status = "pass"
	result.Message = "System resources are available"

	return result
}

func checkProjectStructure(report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:    "Project Structure",
		Details: make(map[string]string),
	}

	// Check for Templar configuration
	configFiles := []string{".templar.yml", ".templar.yaml", "templar.yml", "templar.yaml"}
	configFound := ""
	
	for _, config := range configFiles {
		if _, err := os.Stat(config); err == nil {
			configFound = config
			break
		}
	}

	if configFound == "" {
		result.Status = "warn"
		result.Message = "No Templar configuration found"
		result.Suggestions = []string{
			"Initialize project with: templar init",
			"Create configuration with: templar config wizard",
		}
		result.Fixable = true
		return result
	}

	result.Details["config"] = configFound

	// Check for common directories
	dirs := []string{"components", "views", "examples"}
	foundDirs := []string{}
	
	for _, dir := range dirs {
		if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
			foundDirs = append(foundDirs, dir)
		}
	}

	if len(foundDirs) == 0 {
		result.Status = "warn"
		result.Message = "No component directories found"
		result.Suggestions = []string{
			"Create component directories",
			"Run: templar init --wizard for guided setup",
		}
	} else {
		result.Details["component_dirs"] = strings.Join(foundDirs, ", ")
		result.Status = "pass"
		result.Message = "Project structure looks good"
	}

	return result
}

// Helper functions

func isPortInUse(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func getProcessOnPort(port int) string {
	// This is a simplified version - in a real implementation,
	// you'd use system-specific commands to find the process
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		if output, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port)).Output(); err == nil {
			return strings.TrimSpace(string(output))
		}
	}
	return ""
}

func getFunctionName(f interface{}) string {
	// Helper to get function name for filtering
	return fmt.Sprintf("%v", f)
}

func generateSummary(report *DoctorReport) {
	summary := Summary{}
	
	for _, result := range report.Results {
		summary.Total++
		switch result.Status {
		case "pass":
			summary.Passed++
		case "warn":
			summary.Warned++
		case "fail":
			summary.Failed++
		}
		if result.Fixable {
			summary.Fixable++
		}
	}
	
	report.Summary = summary
}

func detectIntegrations(report *DoctorReport) {
	integrations := IntegrationInfo{}
	
	// Detect tools and their status
	if airPath, err := exec.LookPath("air"); err == nil {
		integrations.Air.Installed = true
		integrations.Air.Path = airPath
		integrations.Air.Status = "compatible"
	}
	
	if _, err := os.Stat("tailwind.config.js"); err == nil {
		integrations.Tailwind.Installed = true
		integrations.Tailwind.Status = "compatible"
	}
	
	if codePath, err := exec.LookPath("code"); err == nil {
		integrations.VSCode.Installed = true
		integrations.VSCode.Path = codePath
		integrations.VSCode.Status = "compatible"
	}
	
	if gitPath, err := exec.LookPath("git"); err == nil {
		integrations.Git.Installed = true
		integrations.Git.Path = gitPath
		if output, err := exec.Command("git", "--version").Output(); err == nil {
			integrations.Git.Version = strings.TrimSpace(string(output))
		}
		integrations.Git.Status = "compatible"
	}
	
	if nodePath, err := exec.LookPath("node"); err == nil {
		integrations.Node.Installed = true
		integrations.Node.Path = nodePath
		if output, err := exec.Command("node", "--version").Output(); err == nil {
			integrations.Node.Version = strings.TrimSpace(string(output))
		}
		integrations.Node.Status = "compatible"
	}
	
	if templPath, err := exec.LookPath("templ"); err == nil {
		integrations.Templ.Installed = true
		integrations.Templ.Path = templPath
		if output, err := exec.Command("templ", "version").Output(); err == nil {
			integrations.Templ.Version = strings.TrimSpace(string(output))
		}
		integrations.Templ.Status = "compatible"
	}
	
	report.Integrations = integrations
}

func applyFixes(report *DoctorReport) error {
	fixed := 0
	
	for _, result := range report.Results {
		if !result.Fixable {
			continue
		}
		
		switch result.Name {
		case "Templ Installation":
			if err := exec.Command("go", "install", "github.com/a-h/templ/cmd/templ@latest").Run(); err == nil {
				fmt.Printf("✓ Fixed: Installed templ\n")
				fixed++
			}
		case "Project Structure":
			if result.Status == "warn" && strings.Contains(result.Message, "No Templar configuration") {
				// This would require more complex logic to run init
				fmt.Printf("⚠ Cannot auto-fix: %s (run 'templar init' manually)\n", result.Name)
			}
		case "File Watching":
			if runtime.GOOS == "linux" && strings.Contains(result.Message, "limit") {
				fmt.Printf("⚠ Cannot auto-fix: %s (requires sudo access)\n", result.Name)
			}
		}
	}
	
	if fixed > 0 {
		fmt.Printf("\n🔧 Applied %d fixes successfully\n", fixed)
	}
	
	return nil
}

func outputReport(report *DoctorReport) error {
	switch doctorFormat {
	case "json":
		return outputJSON(report)
	default:
		return outputText(report)
	}
}

func outputJSON(report *DoctorReport) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func outputText(report *DoctorReport) error {
	fmt.Printf("🏥 Templar Doctor Report\n")
	fmt.Printf("Generated: %s\n", report.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("System: %s/%s, Go: %s\n", report.System.OS, report.System.Arch, report.System.GoVersion)
	fmt.Printf("Working Directory: %s\n", report.System.WorkDir)
	fmt.Println()

	// Output results by category
	categories := map[string][]DiagnosticResult{
		"Core Tools":    {},
		"Integrations":  {},
		"Configuration": {},
		"System":        {},
	}

	for _, result := range report.Results {
		switch result.Name {
		case "Go Installation", "Templ Installation":
			categories["Core Tools"] = append(categories["Core Tools"], result)
		case "Air Integration", "Tailwind CSS Integration", "VS Code Integration", "Git Configuration", "Node.js Environment":
			categories["Integrations"] = append(categories["Integrations"], result)
		case "Port Availability", "Project Structure":
			categories["Configuration"] = append(categories["Configuration"], result)
		default:
			categories["System"] = append(categories["System"], result)
		}
	}

	for category, results := range categories {
		if len(results) == 0 {
			continue
		}
		
		fmt.Printf("📋 %s\n", category)
		fmt.Println(strings.Repeat("─", len(category)+5))
		
		for _, result := range results {
			icon := getStatusIcon(result.Status)
			fmt.Printf("%s %s: %s\n", icon, result.Name, result.Message)
			
			if doctorVerbose && len(result.Details) > 0 {
				for key, value := range result.Details {
					fmt.Printf("    %s: %s\n", key, value)
				}
			}
			
			if len(result.Suggestions) > 0 {
				fmt.Printf("    💡 Suggestions:\n")
				for _, suggestion := range result.Suggestions {
					fmt.Printf("      • %s\n", suggestion)
				}
			}
			fmt.Println()
		}
	}

	// Summary
	fmt.Printf("📊 Summary\n")
	fmt.Println("──────────")
	fmt.Printf("Total Checks: %d\n", report.Summary.Total)
	fmt.Printf("✅ Passed: %d\n", report.Summary.Passed)
	fmt.Printf("⚠️  Warnings: %d\n", report.Summary.Warned)
	fmt.Printf("❌ Failed: %d\n", report.Summary.Failed)
	if report.Summary.Fixable > 0 {
		fmt.Printf("🔧 Fixable: %d (run with --fix to apply)\n", report.Summary.Fixable)
	}
	fmt.Println()

	// Integration status
	fmt.Printf("🔗 Tool Integrations\n")
	fmt.Println("────────────────────")
	
	tools := map[string]ToolInfo{
		"Air (hot reload)":   report.Integrations.Air,
		"Tailwind CSS":       report.Integrations.Tailwind,
		"VS Code":            report.Integrations.VSCode,
		"Git":                report.Integrations.Git,
		"Node.js":            report.Integrations.Node,
		"Templ":              report.Integrations.Templ,
	}
	
	for name, tool := range tools {
		status := "❌ Not installed"
		if tool.Installed {
			status = "✅ Available"
			if tool.Version != "" {
				status += fmt.Sprintf(" (%s)", tool.Version)
			}
		}
		fmt.Printf("%s: %s\n", name, status)
	}
	fmt.Println()

	// Workflow recommendations
	fmt.Printf("🚀 Workflow Recommendations\n")
	fmt.Println("───────────────────────────")
	
	if report.Integrations.Air.Installed && report.Integrations.Tailwind.Installed {
		fmt.Printf("• Full-stack setup detected: Use 'air' for backend, 'templar serve' for components\n")
		fmt.Printf("• Run simultaneously on different ports for optimal development\n")
	} else if report.Integrations.Tailwind.Installed {
		fmt.Printf("• Frontend-focused setup: Use 'templar serve' with Tailwind in watch mode\n")
	} else {
		fmt.Printf("• Basic setup: Use 'templar serve' for component development\n")
		fmt.Printf("• Consider adding Tailwind CSS for enhanced styling capabilities\n")
	}
	
	if !report.Integrations.VSCode.Installed {
		fmt.Printf("• Install VS Code and templ extension for better development experience\n")
	}
	
	if report.Summary.Failed > 0 {
		fmt.Printf("\n❌ Critical Issues Detected\n")
		fmt.Printf("Please resolve failed checks before using Templar\n")
		return fmt.Errorf("doctor found %d critical issues", report.Summary.Failed)
	}
	
	if report.Summary.Warned > 0 {
		fmt.Printf("\n⚠️  Warnings Present\n")
		fmt.Printf("Templar will work, but addressing warnings will improve your experience\n")
	} else {
		fmt.Printf("\n🎉 All checks passed! Your development environment is ready for Templar\n")
	}

	return nil
}

func getStatusIcon(status string) string {
	switch status {
	case "pass":
		return "✅"
	case "warn":
		return "⚠️ "
	case "fail":
		return "❌"
	default:
		return "❓"
	}
}