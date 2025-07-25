package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose development environment and tool integration",
	Long: `Diagnose your development environment and check for tool integration issues.

The doctor command analyzes your development setup and provides recommendations
for integrating Templar with your existing workflow. It checks for:

- Tool availability (templ, air, tailwindcss, etc.)
- Port conflicts and suggestions
- Configuration issues
- Integration opportunities
- Workflow optimizations

Examples:
  templar doctor                    # Full environment diagnosis
  templar doctor --verbose          # Detailed diagnostic output
  templar doctor --fix              # Automatically fix common issues
  templar doctor --format json     # Output as JSON for tooling`,
	RunE: runDoctor,
}

var (
	doctorVerbose bool
	doctorFix     bool
	doctorFormat  string
)

// DiagnosticResult represents the result of a diagnostic check
type DiagnosticResult struct {
	Name        string                 `json:"name" yaml:"name"`
	Category    string                 `json:"category" yaml:"category"`
	Status      string                 `json:"status" yaml:"status"` // "ok", "warning", "error", "info"
	Message     string                 `json:"message" yaml:"message"`
	Suggestion  string                 `json:"suggestion,omitempty" yaml:"suggestion,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty" yaml:"details,omitempty"`
	AutoFixable bool                   `json:"auto_fixable" yaml:"auto_fixable"`
}

// DoctorReport represents the complete diagnostic report
type DoctorReport struct {
	Timestamp   time.Time          `json:"timestamp" yaml:"timestamp"`
	Environment map[string]string  `json:"environment" yaml:"environment"`
	Results     []DiagnosticResult `json:"results" yaml:"results"`
	Summary     ReportSummary      `json:"summary" yaml:"summary"`
}

// ReportSummary provides an overview of diagnostic results
type ReportSummary struct {
	Total    int `json:"total" yaml:"total"`
	OK       int `json:"ok" yaml:"ok"`
	Warnings int `json:"warnings" yaml:"warnings"`
	Errors   int `json:"errors" yaml:"errors"`
	Info     int `json:"info" yaml:"info"`
}

func init() {
	rootCmd.AddCommand(doctorCmd)

	doctorCmd.Flags().BoolVarP(&doctorVerbose, "verbose", "v", false, "Show verbose diagnostic information")
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "Automatically fix common issues where possible")
	doctorCmd.Flags().StringVarP(&doctorFormat, "format", "f", "table", "Output format (table|json|yaml)")
}

func runDoctor(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	fmt.Println("🔍 Templar Development Environment Doctor")
	fmt.Println("==========================================")

	if doctorFix {
		fmt.Println("⚡ Auto-fix mode enabled")
	}

	fmt.Println()

	// Create diagnostic report
	report := &DoctorReport{
		Timestamp:   time.Now(),
		Environment: gatherEnvironmentInfo(),
		Results:     []DiagnosticResult{},
	}

	// Run all diagnostic checks
	checks := []func(context.Context, *DoctorReport) DiagnosticResult{
		checkTemplarConfiguration,
		checkTemplTool,
		checkGoEnvironment,
		checkPortAvailability,
		checkAirIntegration,
		checkTailwindIntegration,
		checkVSCodeIntegration,
		checkGitIntegration,
		checkProcessConflicts,
		checkFileSystemPermissions,
		checkNetworkConfiguration,
		checkDevelopmentWorkflow,
	}

	for _, check := range checks {
		result := check(ctx, report)
		report.Results = append(report.Results, result)

		if !doctorVerbose && result.Status == "info" {
			continue
		}

		displayResult(result)
	}

	// Calculate summary
	report.Summary = calculateSummary(report.Results)

	// Display summary
	fmt.Println("\n📊 Summary")
	fmt.Println("==========")
	displaySummary(report.Summary)

	// Output formatted report if requested
	if doctorFormat != "table" {
		fmt.Println("\n📋 Detailed Report")
		fmt.Println("==================")
		if err := outputReport(report, doctorFormat); err != nil {
			return fmt.Errorf("failed to output report: %w", err)
		}
	}

	// Provide final recommendations
	provideFinalRecommendations(report)

	return nil
}

func gatherEnvironmentInfo() map[string]string {
	env := map[string]string{
		"os":          runtime.GOOS,
		"arch":        runtime.GOARCH,
		"go_version":  runtime.Version(),
		"templar_dir": getCurrentDirectory(),
		"user":        os.Getenv("USER"),
		"shell":       os.Getenv("SHELL"),
		"editor":      getPreferredEditor(),
		"path":        os.Getenv("PATH"),
		"gopath":      os.Getenv("GOPATH"),
		"goroot":      os.Getenv("GOROOT"),
	}

	// Add working directory info
	if wd, err := os.Getwd(); err == nil {
		env["working_dir"] = wd
	}

	return env
}

func checkTemplarConfiguration(ctx context.Context, report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:     "Templar Configuration",
		Category: "Configuration",
		Status:   "ok",
	}

	// Check if .templar.yml exists
	configPath := ".templar.yml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		result.Status = "warning"
		result.Message = "No .templar.yml configuration file found"
		result.Suggestion = "Run 'templar init' to create a new project or 'templar config wizard' for interactive setup"
		result.AutoFixable = true
		return result
	}

	// Try to load and validate configuration
	cfg, err := config.Load()
	if err != nil {
		result.Status = "error"
		result.Message = fmt.Sprintf("Configuration file exists but has errors: %v", err)
		result.Suggestion = "Fix configuration errors or run 'templar config wizard' to reconfigure"
		result.AutoFixable = true
		return result
	}

	result.Message = "Configuration file is valid"
	result.Details = map[string]interface{}{
		"scan_paths":    cfg.Components.ScanPaths,
		"server_port":   cfg.Server.Port,
		"build_command": cfg.Build.Command,
		"hot_reload":    cfg.Development.HotReload,
		"monitoring":    cfg.Monitoring.Enabled,
	}

	// Check for common configuration issues
	if len(cfg.Components.ScanPaths) == 0 {
		result.Status = "warning"
		result.Message = "No component scan paths configured"
		result.Suggestion = "Add component directories to scan_paths in .templar.yml"
	}

	return result
}

func checkTemplTool(ctx context.Context, report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:     "Templ Tool",
		Category: "Tools",
		Status:   "ok",
	}

	// Check if templ is installed
	cmd := exec.CommandContext(ctx, "templ", "version")
	output, err := cmd.Output()
	if err != nil {
		result.Status = "error"
		result.Message = "Templ tool not found"
		result.Suggestion = "Install templ with: go install github.com/a-h/templ/cmd/templ@latest"
		result.AutoFixable = true
		return result
	}

	version := strings.TrimSpace(string(output))
	result.Message = fmt.Sprintf("Templ tool installed: %s", version)
	result.Details = map[string]interface{}{
		"version": version,
		"path":    getCommandPath("templ"),
	}

	// Check if it's a recent version
	if strings.Contains(version, "v0.2") {
		result.Status = "warning"
		result.Message = fmt.Sprintf("Templ version may be outdated: %s", version)
		result.Suggestion = "Update templ with: go install github.com/a-h/templ/cmd/templ@latest"
		result.AutoFixable = true
	}

	return result
}

func checkGoEnvironment(ctx context.Context, report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:     "Go Environment",
		Category: "Environment",
		Status:   "ok",
	}

	// Check Go version
	goVersion := runtime.Version()
	result.Message = fmt.Sprintf("Go version: %s", goVersion)

	details := map[string]interface{}{
		"version": goVersion,
		"gopath":  os.Getenv("GOPATH"),
		"goroot":  os.Getenv("GOROOT"),
	}

	// Check for go.mod file
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		result.Status = "warning"
		result.Message = "No go.mod file found in current directory"
		result.Suggestion = "Initialize a Go module with: go mod init <module-name>"
		result.AutoFixable = true
		details["has_go_mod"] = false
	} else {
		details["has_go_mod"] = true
	}

	// Check Go version compatibility
	if strings.Contains(goVersion, "go1.19") || strings.Contains(goVersion, "go1.18") {
		result.Status = "warning"
		result.Message = fmt.Sprintf("Go version may be outdated for optimal templ support: %s", goVersion)
		result.Suggestion = "Consider upgrading to Go 1.20+ for better generics and templ support"
	}

	result.Details = details

	return result
}

func checkPortAvailability(ctx context.Context, report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:     "Port Availability",
		Category: "Network",
		Status:   "ok",
	}

	// Default ports to check
	portsToCheck := []int{8080, 8081, 3000, 3001, 5173, 4000}
	availablePorts := []int{}
	conflictPorts := []int{}

	for _, port := range portsToCheck {
		if isPortAvailable(port) {
			availablePorts = append(availablePorts, port)
		} else {
			conflictPorts = append(conflictPorts, port)
			if port == 8080 { // Default Templar port
				result.Status = "warning"
			}
		}
	}

	if len(conflictPorts) == 0 {
		result.Message = "All common development ports are available"
	} else {
		result.Message = fmt.Sprintf("Port conflicts detected: %v", conflictPorts)
		maxPorts := len(availablePorts)
		if maxPorts > 3 {
			maxPorts = 3
		}
		result.Suggestion = fmt.Sprintf("Use alternative ports: %v, or stop conflicting services", availablePorts[:maxPorts])

		if contains(conflictPorts, 8080) && len(availablePorts) > 0 {
			result.Suggestion += "\nFor Templar, use: templar serve --port " + fmt.Sprintf("%d", availablePorts[0])
		}
	}

	result.Details = map[string]interface{}{
		"available_ports": availablePorts,
		"conflict_ports":  conflictPorts,
	}

	return result
}

func checkAirIntegration(ctx context.Context, report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:     "Air Integration",
		Category: "Tools",
		Status:   "info",
	}

	// Check if air is installed
	cmd := exec.CommandContext(ctx, "air", "-v")
	output, err := cmd.Output()
	if err != nil {
		result.Message = "Air tool not detected"
		result.Suggestion = "Install Air for Go hot reload: go install github.com/air-verse/air@latest"
		result.AutoFixable = true
		return result
	}

	version := strings.TrimSpace(string(output))
	result.Status = "ok"
	result.Message = fmt.Sprintf("Air installed: %s", version)

	// Check for .air.toml configuration
	airConfigExists := false
	if _, err := os.Stat(".air.toml"); err == nil {
		airConfigExists = true
		result.Details = map[string]interface{}{
			"config_file": ".air.toml",
		}
	}

	if !airConfigExists {
		result.Status = "warning"
		result.Message = fmt.Sprintf("Air installed (%s) but no .air.toml configuration found", version)
		result.Suggestion = "Create .air.toml with: air init, or integrate with Templar using our air config template"
		result.AutoFixable = true
	} else {
		result.Message = fmt.Sprintf("Air properly configured: %s", version)
		result.Details["configured"] = true
	}

	return result
}

func checkTailwindIntegration(ctx context.Context, report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:     "Tailwind CSS Integration",
		Category: "Tools",
		Status:   "info",
	}

	// Check for tailwindcss installation
	tailwindPaths := []string{
		"node_modules/.bin/tailwindcss",
		"tailwindcss",
	}

	var tailwindPath string
	for _, path := range tailwindPaths {
		if cmd := exec.CommandContext(ctx, path, "--version"); cmd.Run() == nil {
			tailwindPath = path
			break
		}
	}

	if tailwindPath == "" {
		result.Message = "Tailwind CSS not detected"
		result.Suggestion = "Install Tailwind CSS: npm install -D tailwindcss@latest"
		result.AutoFixable = false
		return result
	}

	// Check for tailwind.config.js
	configFiles := []string{"tailwind.config.js", "tailwind.config.ts", "tailwind.config.cjs"}
	var configFile string
	for _, file := range configFiles {
		if _, err := os.Stat(file); err == nil {
			configFile = file
			break
		}
	}

	result.Status = "ok"
	result.Message = "Tailwind CSS detected"
	result.Details = map[string]interface{}{
		"path":        tailwindPath,
		"config_file": configFile,
	}

	if configFile == "" {
		result.Status = "warning"
		result.Message = "Tailwind CSS found but no configuration file detected"
		result.Suggestion = "Initialize Tailwind config: npx tailwindcss init"
		result.AutoFixable = true
	} else {
		result.Message = "Tailwind CSS properly configured"
	}

	return result
}

func checkVSCodeIntegration(ctx context.Context, report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:     "VS Code Integration",
		Category: "Editor",
		Status:   "info",
	}

	// Check if VS Code is available
	vscodeCommands := []string{"code", "code-insiders"}
	var vscodeCmd string
	for _, cmd := range vscodeCommands {
		if exec.CommandContext(ctx, cmd, "--version").Run() == nil {
			vscodeCmd = cmd
			break
		}
	}

	if vscodeCmd == "" {
		result.Message = "VS Code not detected"
		result.Suggestion = "Install VS Code for better templ development experience"
		return result
	}

	result.Status = "ok"
	result.Message = "VS Code detected"
	result.Details = map[string]interface{}{
		"command": vscodeCmd,
	}

	// Check for .vscode directory and settings
	vscodeDir := ".vscode"
	if _, err := os.Stat(vscodeDir); err == nil {
		result.Details["workspace_config"] = true

		// Check for recommended extensions
		if _, err := os.Stat(filepath.Join(vscodeDir, "extensions.json")); err == nil {
			result.Details["recommended_extensions"] = true
			result.Message = "VS Code workspace properly configured"
		} else {
			result.Status = "warning"
			result.Message = "VS Code detected but no recommended extensions configured"
			result.Suggestion = "Add templ extension recommendations to .vscode/extensions.json"
			result.AutoFixable = true
		}
	} else {
		result.Status = "warning"
		result.Message = "VS Code detected but no workspace configuration"
		result.Suggestion = "Create .vscode/settings.json and extensions.json for better development experience"
		result.AutoFixable = true
	}

	return result
}

func checkGitIntegration(ctx context.Context, report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:     "Git Integration",
		Category: "Version Control",
		Status:   "info",
	}

	// Check if we're in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		result.Message = "Not a Git repository"
		result.Suggestion = "Initialize Git repository: git init"
		result.AutoFixable = true
		return result
	}

	result.Status = "ok"
	result.Message = "Git repository detected"

	// Check for .gitignore
	gitignoreExists := false
	if _, err := os.Stat(".gitignore"); err == nil {
		gitignoreExists = true
	}

	if !gitignoreExists {
		result.Status = "warning"
		result.Message = "Git repository found but no .gitignore file"
		result.Suggestion = "Create .gitignore to exclude build artifacts and cache files"
		result.AutoFixable = true
	} else {
		// Check if common patterns are ignored
		content, err := os.ReadFile(".gitignore")
		if err == nil {
			gitignoreContent := string(content)
			requiredPatterns := []string{"*_templ.go", ".templar/", "node_modules/"}
			missingPatterns := []string{}

			for _, pattern := range requiredPatterns {
				if !strings.Contains(gitignoreContent, pattern) {
					missingPatterns = append(missingPatterns, pattern)
				}
			}

			if len(missingPatterns) > 0 {
				result.Status = "warning"
				result.Message = "Git configured but .gitignore may be missing templ-related patterns"
				result.Suggestion = fmt.Sprintf("Add these patterns to .gitignore: %v", missingPatterns)
				result.AutoFixable = true
			}
		}
	}

	result.Details = map[string]interface{}{
		"has_gitignore": gitignoreExists,
	}

	return result
}

func checkProcessConflicts(ctx context.Context, report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:     "Process Conflicts",
		Category: "System",
		Status:   "ok",
	}

	conflictingProcesses := []string{}

	// Check for common development server processes
	processesToCheck := []string{
		"air",
		"nodemon",
		"webpack-dev-server",
		"vite",
	}

	for _, process := range processesToCheck {
		if isProcessRunning(process) {
			conflictingProcesses = append(conflictingProcesses, process)
		}
	}

	if len(conflictingProcesses) == 0 {
		result.Message = "No conflicting development processes detected"
	} else {
		result.Status = "warning"
		result.Message = fmt.Sprintf("Development processes running: %v", conflictingProcesses)
		result.Suggestion = "These processes might conflict with Templar. Consider coordinating or using different ports."
	}

	result.Details = map[string]interface{}{
		"running_processes": conflictingProcesses,
	}

	return result
}

func checkFileSystemPermissions(ctx context.Context, report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:     "File System Permissions",
		Category: "System",
		Status:   "ok",
	}

	// Check write permissions in current directory
	testFile := ".templar-permission-test"
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		result.Status = "error"
		result.Message = "Cannot write to current directory"
		result.Suggestion = "Check directory permissions or change to a writable directory"
		return result
	}
	os.Remove(testFile) // Clean up

	// Check cache directory permissions
	cacheDir := ".templar"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		result.Status = "warning"
		result.Message = "Cannot create .templar cache directory"
		result.Suggestion = "Check permissions for creating directories in current location"
		return result
	}

	result.Message = "File system permissions are adequate"
	return result
}

func checkNetworkConfiguration(ctx context.Context, report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:     "Network Configuration",
		Category: "Network",
		Status:   "ok",
	}

	// Check if we can bind to localhost
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		result.Status = "error"
		result.Message = "Cannot bind to localhost"
		result.Suggestion = "Check network configuration and firewall settings"
		return result
	}

	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	result.Message = "Network configuration is working"
	result.Details = map[string]interface{}{
		"test_port":            port,
		"localhost_accessible": true,
	}

	return result
}

func checkDevelopmentWorkflow(ctx context.Context, report *DoctorReport) DiagnosticResult {
	result := DiagnosticResult{
		Name:     "Development Workflow",
		Category: "Workflow",
		Status:   "info",
	}

	recommendations := []string{}
	workflowScore := 0

	// Analyze detected tools and provide workflow recommendations
	for _, prevResult := range report.Results {
		switch prevResult.Name {
		case "Air Integration":
			if prevResult.Status == "ok" {
				recommendations = append(recommendations, "✅ Air + Templar: Use 'air' for Go hot reload and 'templar serve' for component preview")
				workflowScore++
			}
		case "Tailwind CSS Integration":
			if prevResult.Status == "ok" {
				recommendations = append(recommendations, "✅ Tailwind + Templar: Run 'tailwindcss --watch' alongside 'templar serve'")
				workflowScore++
			}
		case "VS Code Integration":
			if prevResult.Status == "ok" {
				recommendations = append(recommendations, "✅ VS Code: Install templ extension for syntax highlighting")
				workflowScore++
			}
		case "Git Integration":
			if prevResult.Status == "ok" {
				recommendations = append(recommendations, "✅ Git: Exclude *_templ.go files from version control")
				workflowScore++
			}
		}
	}

	// Provide workflow quality assessment
	if workflowScore >= 3 {
		result.Status = "ok"
		result.Message = "Well-integrated development workflow detected"
	} else if workflowScore >= 1 {
		result.Status = "warning"
		result.Message = "Partial development workflow integration"
		result.Suggestion = "Consider integrating more development tools for optimal experience"
	} else {
		result.Status = "warning"
		result.Message = "Basic development setup detected"
		result.Suggestion = "Integrate development tools like Air, Tailwind, and VS Code for enhanced productivity"
	}

	result.Details = map[string]interface{}{
		"workflow_score":    workflowScore,
		"recommendations":   recommendations,
		"integration_level": getIntegrationLevel(workflowScore),
	}

	return result
}

// Helper functions

func getCurrentDirectory() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "unknown"
}

func getPreferredEditor() string {
	editors := []string{"VISUAL", "EDITOR"}
	for _, env := range editors {
		if editor := os.Getenv(env); editor != "" {
			return editor
		}
	}
	return "unknown"
}

func getCommandPath(command string) string {
	if path, err := exec.LookPath(command); err == nil {
		return path
	}
	return "not found"
}

func isPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

func isProcessRunning(processName string) bool {
	cmd := exec.Command("pgrep", "-f", processName)
	return cmd.Run() == nil
}

func contains(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func getIntegrationLevel(score int) string {
	if score >= 4 {
		return "excellent"
	} else if score >= 2 {
		return "good"
	} else if score >= 1 {
		return "basic"
	}
	return "minimal"
}

func displayResult(result DiagnosticResult) {
	var icon string
	switch result.Status {
	case "ok":
		icon = "✅"
	case "warning":
		icon = "⚠️"
	case "error":
		icon = "❌"
	case "info":
		icon = "ℹ️"
	default:
		icon = "•"
	}

	fmt.Printf("%s [%s] %s: %s\n", icon, strings.ToUpper(result.Category), result.Name, result.Message)

	if result.Suggestion != "" {
		fmt.Printf("   💡 %s\n", result.Suggestion)
	}

	if doctorVerbose && result.Details != nil && len(result.Details) > 0 {
		fmt.Printf("   📋 Details: %+v\n", result.Details)
	}

	fmt.Println()
}

func calculateSummary(results []DiagnosticResult) ReportSummary {
	summary := ReportSummary{
		Total: len(results),
	}

	for _, result := range results {
		switch result.Status {
		case "ok":
			summary.OK++
		case "warning":
			summary.Warnings++
		case "error":
			summary.Errors++
		case "info":
			summary.Info++
		}
	}

	return summary
}

func displaySummary(summary ReportSummary) {
	fmt.Printf("Total Checks: %d\n", summary.Total)
	fmt.Printf("✅ OK: %d\n", summary.OK)
	fmt.Printf("⚠️  Warnings: %d\n", summary.Warnings)
	fmt.Printf("❌ Errors: %d\n", summary.Errors)
	fmt.Printf("ℹ️  Info: %d\n", summary.Info)

	// Calculate health score
	healthScore := float64(summary.OK) / float64(summary.Total) * 100
	fmt.Printf("\n🎯 Environment Health Score: %.0f%%\n", healthScore)
}

func outputReport(report *DoctorReport, format string) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	case "yaml":
		encoder := yaml.NewEncoder(os.Stdout)
		return encoder.Encode(report)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func provideFinalRecommendations(report *DoctorReport) {
	fmt.Println("\n🚀 Final Recommendations")
	fmt.Println("========================")

	hasErrors := report.Summary.Errors > 0
	hasWarnings := report.Summary.Warnings > 0

	if hasErrors {
		fmt.Println("❌ Critical Issues Detected:")
		fmt.Println("   Address the errors above before starting development")
		fmt.Println()
	}

	if hasWarnings {
		fmt.Println("⚠️  Optimization Opportunities:")
		fmt.Println("   Review warnings above to improve your development experience")
		fmt.Println()
	}

	if !hasErrors && !hasWarnings {
		fmt.Println("🎉 Your development environment looks great!")
		fmt.Println("   You're ready to start using Templar effectively")
		fmt.Println()
	}

	// Provide specific next steps based on findings
	fmt.Println("📝 Next Steps:")

	if !hasTemplarConfig(report) {
		fmt.Println("   1. Run 'templar init' to set up a new project")
	} else {
		fmt.Println("   1. Run 'templar serve' to start the development server")
	}

	if hasIntegrationOpportunities(report) {
		fmt.Println("   2. Consider integrating detected tools for better workflow")
	}

	fmt.Println("   3. Visit https://templar.dev/docs for comprehensive guides")
	fmt.Println()
}

func hasTemplarConfig(report *DoctorReport) bool {
	for _, result := range report.Results {
		if result.Name == "Templar Configuration" && result.Status == "ok" {
			return true
		}
	}
	return false
}

func hasIntegrationOpportunities(report *DoctorReport) bool {
	for _, result := range report.Results {
		if result.AutoFixable && (result.Status == "warning" || result.Status == "error") {
			return true
		}
	}
	return false
}
