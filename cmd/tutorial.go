package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var tutorialCmd = &cobra.Command{
	Use:   "tutorial",
	Short: "Interactive tutorial for learning Templar workflows",
	Long: `Interactive tutorial that guides you through common Templar workflows
and helps you discover the most useful commands and flags.

The tutorial covers:
  • Project initialization with templates
  • Component development workflow  
  • Live preview and hot reload
  • Production building and optimization
  • Advanced features and integrations

Examples:
  templar tutorial                    # Start interactive tutorial
  templar tutorial --quick            # Quick 5-minute overview
  templar tutorial --topic=preview    # Focus on specific topic`,
	RunE: runTutorial,
}

var (
	tutorialQuick bool
	tutorialTopicFlag string
)

// tutorialTopic represents a tutorial section
type tutorialTopic struct {
	Key         string
	Title       string
	Description string
}

func init() {
	rootCmd.AddCommand(tutorialCmd)
	
	tutorialCmd.Flags().BoolVar(&tutorialQuick, "quick", false, "Quick 5-minute tutorial overview")
	tutorialCmd.Flags().StringVar(&tutorialTopicFlag, "topic", "", "Focus on specific topic (init, serve, preview, build, deploy)")
}

func runTutorial(cmd *cobra.Command, args []string) error {
	tutorial := NewTutorial()
	
	if tutorialQuick {
		return tutorial.RunQuick()
	}
	
	if tutorialTopicFlag != "" {
		return tutorial.RunTopic(tutorialTopicFlag)
	}
	
	return tutorial.RunFull()
}

// Tutorial manages the interactive tutorial experience
type Tutorial struct {
	reader *bufio.Reader
}

// NewTutorial creates a new tutorial instance
func NewTutorial() *Tutorial {
	return &Tutorial{
		reader: bufio.NewReader(os.Stdin),
	}
}

// RunFull runs the complete interactive tutorial
func (t *Tutorial) RunFull() error {
	fmt.Println("🎓 Welcome to Templar Tutorial!")
	fmt.Println("==============================")
	fmt.Println("This interactive tutorial will guide you through Templar's key features.")
	fmt.Println("You can press Ctrl+C at any time to exit.")
	fmt.Println()

	topics := []*tutorialTopic{
		{"init", "Project Initialization", "Learn how to start new projects with templates"},
		{"serve", "Development Server", "Run live preview with hot reload"},
		{"preview", "Component Preview", "Preview individual components with props"},
		{"build", "Building & Optimization", "Production builds and analysis"},
		{"advanced", "Advanced Features", "Plugins, monitoring, and workflows"},
	}

	for i, topic := range topics {
		fmt.Printf("%d. %s - %s\n", i+1, topic.Title, topic.Description)
	}
	fmt.Println()

	choice := t.askChoice("Which topic would you like to start with?", 
		[]string{"1", "2", "3", "4", "5", "all"}, "all")
	
	switch choice {
	case "1":
		return t.RunTopic("init")
	case "2":
		return t.RunTopic("serve")
	case "3":
		return t.RunTopic("preview")
	case "4":
		return t.RunTopic("build")
	case "5":
		return t.RunTopic("advanced")
	case "all":
		return t.runAllTopics(topics)
	default:
		return nil
	}
}

// RunQuick runs a 5-minute overview
func (t *Tutorial) RunQuick() error {
	fmt.Println("⚡ Quick Templar Tutorial (5 minutes)")
	fmt.Println("====================================")
	fmt.Println()

	sections := []struct {
		title   string
		content string
	}{
		{
			"1. Initialize a Project",
			`templar init my-app --wizard        # Interactive setup with smart defaults
templar init --template=dashboard   # Use dashboard template
templar init --minimal              # Minimal setup`,
		},
		{
			"2. Start Development", 
			`templar serve -p 3000              # Start dev server on port 3000
templar serve --no-open             # Don't auto-open browser
templar watch -v                    # Watch files with verbose output`,
		},
		{
			"3. Preview Components",
			`templar preview Button --props='{"text":"Click me"}'
templar preview Card -f props.json  # Use props from file
templar list --format=json          # List all components`,
		},
		{
			"4. Build for Production",
			`templar build --production         # Optimized production build
templar build --analyze             # Generate build analysis
templar build --clean               # Clean before building`,
		},
	}

	for _, section := range sections {
		fmt.Printf("📝 %s\n", section.title)
		fmt.Printf("%s\n\n", section.content)
		
		if !t.askBool("Continue to next section?", true) {
			break
		}
	}

	fmt.Println("🎉 Tutorial complete! Run 'templar tutorial' for detailed guidance.")
	return nil
}

// RunTopic runs tutorial for a specific topic
func (t *Tutorial) RunTopic(topic string) error {
	switch topic {
	case "init":
		return t.runInitTopic()
	case "serve":
		return t.runServeTopic()
	case "preview":
		return t.runPreviewTopic()
	case "build":
		return t.runBuildTopic()
	case "advanced":
		return t.runAdvancedTopic()
	default:
		return fmt.Errorf("unknown topic: %s. Available: init, serve, preview, build, advanced", topic)
	}
}

func (t *Tutorial) runInitTopic() error {
	fmt.Println("🚀 Project Initialization Tutorial")
	fmt.Println("==================================")
	fmt.Println()

	fmt.Println("Templar provides several ways to initialize projects:")
	fmt.Println()

	examples := []struct {
		command string
		desc    string
	}{
		{"templar init", "Initialize in current directory with examples"},
		{"templar init my-project", "Create new directory 'my-project'"},
		{"templar init --wizard", "Interactive configuration wizard"},
		{"templar init --minimal", "Minimal setup without examples"},
		{"templar init --template=blog", "Use blog template"},
		{"templar init --template=dashboard", "Use dashboard template"},
		{"templar init --template=landing", "Use landing page template"},
	}

	for _, example := range examples {
		fmt.Printf("  %-35s # %s\n", example.command, example.desc)
	}

	fmt.Println()
	fmt.Println("💡 Pro Tips:")
	fmt.Println("  • Use --wizard for smart defaults based on your project structure")
	fmt.Println("  • Templates include pre-built components for common use cases")
	fmt.Println("  • Run 'templar init --help' to see all available templates")
	
	return nil
}

func (t *Tutorial) runServeTopic() error {
	fmt.Println("🌐 Development Server Tutorial")
	fmt.Println("=============================")
	fmt.Println()

	fmt.Println("The development server provides live preview with hot reload:")
	fmt.Println()

	examples := []struct {
		command string
		desc    string
	}{
		{"templar serve", "Start server on localhost:8080"},
		{"templar serve -p 3000", "Use custom port"},
		{"templar serve --host 0.0.0.0", "Bind to all interfaces"},
		{"templar serve --no-open", "Don't auto-open browser"},
		{"templar serve -w '**/*.go'", "Custom watch pattern"},
	}

	for _, example := range examples {
		fmt.Printf("  %-40s # %s\n", example.command, example.desc)
	}

	fmt.Println()
	fmt.Println("💡 Pro Tips:")
	fmt.Println("  • Server automatically rebuilds on file changes")
	fmt.Println("  • WebSocket connection provides instant updates") 
	fmt.Println("  • Use different ports for multiple projects")
	fmt.Println("  • Monitor logs for build errors and warnings")

	return nil
}

func (t *Tutorial) runPreviewTopic() error {
	fmt.Println("👁 Component Preview Tutorial")
	fmt.Println("============================")
	fmt.Println()

	fmt.Println("Preview individual components with different props:")
	fmt.Println()

	examples := []struct {
		command string
		desc    string
	}{
		{`templar preview Button`, "Preview Button component"},
		{`templar preview Button --props='{"text":"Click me"}'`, "With inline props"},
		{`templar preview Card -f card-props.json`, "With props from file"},
		{`templar preview ProductCard --mock auto`, "With auto-generated mock data"},
		{`templar list --with-props`, "List components with their properties"},
		{`templar list --format=json`, "JSON output for tooling"},
	}

	for _, example := range examples {
		fmt.Printf("  %-50s # %s\n", example.command, example.desc)
	}

	fmt.Println()
	fmt.Println("💡 Pro Tips:")
	fmt.Println("  • Props can be JSON strings or file references (@file.json)")
	fmt.Println("  • Auto mock generates realistic test data")
	fmt.Println("  • Preview runs on separate port to avoid conflicts")
	fmt.Println("  • Use --wrapper to customize preview layout")

	return nil
}

func (t *Tutorial) runBuildTopic() error {
	fmt.Println("🔨 Building & Optimization Tutorial")
	fmt.Println("==================================")
	fmt.Println()

	fmt.Println("Build components for production deployment:")
	fmt.Println()

	examples := []struct {
		command string
		desc    string
	}{
		{"templar build", "Standard build"},
		{"templar build --production", "Optimized production build"},
		{"templar build --analyze", "Generate build analysis"},
		{"templar build --clean", "Clean before building"},
		{"templar build -o ./dist", "Custom output directory"},
		{"templar generate --format=types", "Generate TypeScript types"},
	}

	for _, example := range examples {
		fmt.Printf("  %-40s # %s\n", example.command, example.desc)
	}

	fmt.Println()
	fmt.Println("💡 Pro Tips:")
	fmt.Println("  • Production builds enable optimizations and minification")
	fmt.Println("  • Analysis helps identify bundle size issues")  
	fmt.Println("  • Generate types for better IDE integration")
	fmt.Println("  • Clean builds ensure consistent output")

	return nil
}

func (t *Tutorial) runAdvancedTopic() error {
	fmt.Println("⚡ Advanced Features Tutorial")
	fmt.Println("============================")
	fmt.Println()

	fmt.Println("Templar's advanced features for complex workflows:")
	fmt.Println()

	sections := []struct {
		title    string
		commands []string
	}{
		{
			"Configuration Management:",
			[]string{
				"templar config wizard                 # Interactive configuration",
				"templar config validate               # Validate .templar.yml", 
				"templar config show --format=json    # View current config",
			},
		},
		{
			"Component Generation:",
			[]string{
				"templar component create Button --template=interactive",
				"templar component scaffold --project=MyApp",
				"templar generate --format=docs       # Generate documentation",
			},
		},
		{
			"Monitoring & Health:",
			[]string{
				"templar health                        # Check system health",
				"templar version --detailed            # Detailed version info",
			},
		},
	}

	for _, section := range sections {
		fmt.Printf("%s\n", section.title)
		for _, cmd := range section.commands {
			fmt.Printf("  %s\n", cmd)
		}
		fmt.Println()
	}

	fmt.Println("💡 Pro Tips:")
	fmt.Println("  • Use --verbose flag for detailed output on any command")
	fmt.Println("  • Configuration wizard detects your project structure")
	fmt.Println("  • Health checks validate your development environment")
	fmt.Println("  • Component scaffolding creates complete project structures")

	return nil
}

func (t *Tutorial) runAllTopics(topics []*tutorialTopic) error {
	for i, topic := range topics {
		fmt.Printf("\n📚 Topic %d: %s\n", i+1, topic.Title)
		fmt.Println(strings.Repeat("=", len(topic.Title)+15))
		
		if err := t.RunTopic(topic.Key); err != nil {
			return err
		}
		
		if i < len(topics)-1 {
			if !t.askBool("Continue to next topic?", true) {
				break
			}
		}
	}
	
	fmt.Println("\n🎉 Tutorial complete!")
	fmt.Println("For more help, run 'templar <command> --help' or visit the documentation.")
	return nil
}

// Helper methods

func (t *Tutorial) askBool(prompt string, defaultValue bool) bool {
	defaultStr := "n"
	if defaultValue {
		defaultStr = "y"
	}

	fmt.Printf("%s [%s]: ", prompt, defaultStr)

	input, err := t.reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultValue
	}

	return input == "y" || input == "yes" || input == "true"
}

func (t *Tutorial) askChoice(prompt string, choices []string, defaultValue string) string {
	for {
		fmt.Printf("%s [%s] (options: %s): ", prompt, defaultValue, strings.Join(choices, ", "))

		input, err := t.reader.ReadString('\n')
		if err != nil {
			return defaultValue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			return defaultValue
		}

		// Check if input is valid choice
		for _, choice := range choices {
			if strings.ToLower(input) == strings.ToLower(choice) {
				return choice
			}
		}

		fmt.Printf("❌ Invalid choice. Please select from: %s\n", strings.Join(choices, ", "))
	}
}

// askInt removed - unused method (config wizard has identical functionality)