package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// interactiveCmd provides an interactive command selection menu.
var interactiveCmd = &cobra.Command{
	Use:     "interactive",
	Aliases: []string{"menu", "m"},
	Short:   "Interactive command selection menu",
	Long: `Launch an interactive menu to select and run Templar commands.
This provides a user-friendly way to discover and execute commands without
remembering exact command names and flags.

The menu displays all available commands with descriptions and allows you
to select one to run with guided parameter input.`,
	RunE: runInteractive,
}

func init() {
	rootCmd.AddCommand(interactiveCmd)
}

func runInteractive(cmd *cobra.Command, args []string) error {
	for {
		if err := showInteractiveMenu(); err != nil {
			return err
		}
	}
}

func showInteractiveMenu() error {
	fmt.Println()
	fmt.Println("╭─────────────────────────────────────────────────────────╮")
	fmt.Println("│                   Templar CLI Menu                      │")
	fmt.Println("├─────────────────────────────────────────────────────────┤")
	fmt.Println("│  Select a command to run:                              │")
	fmt.Println("├─────────────────────────────────────────────────────────┤")
	fmt.Println("│  1. init (i)     - Initialize new project              │")
	fmt.Println("│  2. serve (s)    - Start development server            │")
	fmt.Println("│  3. preview (p)  - Preview specific component          │")
	fmt.Println("│  4. build (b)    - Build all components                │")
	fmt.Println("│  5. list (l)     - List discovered components          │")
	fmt.Println("│  6. watch (w)    - Watch files and rebuild             │")
	fmt.Println("│  7. help         - Show help information               │")
	fmt.Println("│  8. version      - Show version information            │")
	fmt.Println("│  0. exit         - Exit interactive mode               │")
	fmt.Println("╰─────────────────────────────────────────────────────────╯")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your choice (0-8): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)
	if err != nil {
		fmt.Printf("Invalid input '%s'. Please enter a number between 0-8.\n", input)

		return nil
	}

	switch choice {
	case 0:
		fmt.Println("Goodbye!")
		os.Exit(0)

		return nil
	case 1:
		return runInteractiveInit()
	case 2:
		return runInteractiveServe()
	case 3:
		return runInteractivePreview()
	case 4:
		return runInteractiveBuild()
	case 5:
		return runInteractiveList()
	case 6:
		return runInteractiveWatch()
	case 7:
		return rootCmd.Help()
	case 8:
		versionCmd, _, err := rootCmd.Find([]string{"version"})
		if err == nil && versionCmd != nil {
			return versionCmd.RunE(versionCmd, []string{})
		}
		fmt.Println("Version command not available")

		return nil
	default:
		fmt.Printf("Invalid choice '%d'. Please enter a number between 0-8.\n", choice)

		return nil
	}
}

func runInteractiveInit() error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\n=== Initialize New Project ===")

	// Get project name
	fmt.Print("Project name (press Enter for current directory): ")
	projectName, _ := reader.ReadString('\n')
	projectName = strings.TrimSpace(projectName)

	// Get template choice
	fmt.Println("\nAvailable templates:")
	fmt.Println("1. minimal        - Basic component setup")
	fmt.Println("2. blog          - Blog posts, layouts, and content management")
	fmt.Println("3. dashboard     - Admin dashboard with sidebar navigation")
	fmt.Println("4. landing       - Marketing landing page")
	fmt.Println("5. ecommerce     - Product listings and shopping cart")
	fmt.Println("6. documentation - Technical documentation")
	fmt.Print("Choose template (1-6, default: minimal): ")

	templateChoice, _ := reader.ReadString('\n')
	templateChoice = strings.TrimSpace(templateChoice)

	var template string
	switch templateChoice {
	case "2":
		template = "blog"
	case "3":
		template = "dashboard"
	case "4":
		template = "landing"
	case "5":
		template = "ecommerce"
	case "6":
		template = "documentation"
	default:
		template = "minimal"
	}

	// Ask for minimal setup
	fmt.Print("Use minimal setup? (y/N): ")
	minimalChoice, _ := reader.ReadString('\n')
	minimalChoice = strings.TrimSpace(strings.ToLower(minimalChoice))

	// Build command args
	args := []string{}
	if projectName != "" {
		args = append(args, projectName)
	}
	if template != "minimal" {
		args = append(args, "--template="+template)
	}
	if minimalChoice == "y" || minimalChoice == "yes" {
		args = append(args, "--minimal")
	}

	fmt.Printf("\nRunning: templar init %s\n", strings.Join(args, " "))

	return initCmd.RunE(initCmd, args)
}

func runInteractiveServe() error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\n=== Start Development Server ===")

	// Get port
	fmt.Print("Port (default: 8080): ")
	portInput, _ := reader.ReadString('\n')
	portInput = strings.TrimSpace(portInput)

	// Get host
	fmt.Print("Host (default: localhost): ")
	hostInput, _ := reader.ReadString('\n')
	hostInput = strings.TrimSpace(hostInput)

	// Ask about opening browser
	fmt.Print("Open browser automatically? (Y/n): ")
	openChoice, _ := reader.ReadString('\n')
	openChoice = strings.TrimSpace(strings.ToLower(openChoice))

	// Build command args
	args := []string{}
	if portInput != "" {
		args = append(args, "--port="+portInput)
	}
	if hostInput != "" {
		args = append(args, "--host="+hostInput)
	}
	if openChoice == "n" || openChoice == "no" {
		args = append(args, "--no-open")
	}

	fmt.Printf("\nRunning: templar serve %s\n", strings.Join(args, " "))

	return serveCmd.RunE(serveCmd, args)
}

func runInteractivePreview() error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\n=== Preview Component ===")

	// Get component name
	fmt.Print("Component name: ")
	componentName, _ := reader.ReadString('\n')
	componentName = strings.TrimSpace(componentName)

	if componentName == "" {
		fmt.Println("Error: Component name is required")

		return nil
	}

	// Get port
	fmt.Print("Port (default: 8080): ")
	portInput, _ := reader.ReadString('\n')
	portInput = strings.TrimSpace(portInput)

	// Ask about props
	fmt.Print("Props JSON (optional): ")
	propsInput, _ := reader.ReadString('\n')
	propsInput = strings.TrimSpace(propsInput)

	// Build command args
	args := []string{componentName}
	if portInput != "" {
		args = append(args, "--port="+portInput)
	}
	if propsInput != "" {
		args = append(args, "--props="+propsInput)
	}

	fmt.Printf("\nRunning: templar preview %s\n", strings.Join(args, " "))

	return previewCmd.RunE(previewCmd, args)
}

func runInteractiveBuild() error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\n=== Build Components ===")

	// Ask for production build
	fmt.Print("Production build? (y/N): ")
	prodChoice, _ := reader.ReadString('\n')
	prodChoice = strings.TrimSpace(strings.ToLower(prodChoice))

	// Ask for analysis
	fmt.Print("Generate build analysis? (y/N): ")
	analyzeChoice, _ := reader.ReadString('\n')
	analyzeChoice = strings.TrimSpace(strings.ToLower(analyzeChoice))

	// Build command args
	args := []string{}
	if prodChoice == "y" || prodChoice == "yes" {
		args = append(args, "--production")
	}
	if analyzeChoice == "y" || analyzeChoice == "yes" {
		args = append(args, "--analyze")
	}

	fmt.Printf("\nRunning: templar build %s\n", strings.Join(args, " "))

	return buildCmd.RunE(buildCmd, args)
}

func runInteractiveList() error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\n=== List Components ===")

	// Get format
	fmt.Println("Output format:")
	fmt.Println("1. table (default)")
	fmt.Println("2. json")
	fmt.Println("3. yaml")
	fmt.Print("Choose format (1-3): ")

	formatChoice, _ := reader.ReadString('\n')
	formatChoice = strings.TrimSpace(formatChoice)

	var format string
	switch formatChoice {
	case "2":
		format = "json"
	case "3":
		format = "yaml"
	default:
		format = "table"
	}

	// Ask for additional info
	fmt.Print("Include properties? (y/N): ")
	propsChoice, _ := reader.ReadString('\n')
	propsChoice = strings.TrimSpace(strings.ToLower(propsChoice))

	fmt.Print("Include dependencies? (y/N): ")
	depsChoice, _ := reader.ReadString('\n')
	depsChoice = strings.TrimSpace(strings.ToLower(depsChoice))

	// Build command args
	args := []string{}
	if format != "table" {
		args = append(args, "--format="+format)
	}
	if propsChoice == "y" || propsChoice == "yes" {
		args = append(args, "--with-props")
	}
	if depsChoice == "y" || depsChoice == "yes" {
		args = append(args, "--with-deps")
	}

	fmt.Printf("\nRunning: templar list %s\n", strings.Join(args, " "))

	return listCmd.RunE(listCmd, args)
}

func runInteractiveWatch() error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\n=== Watch Files ===")

	// Ask for verbose output
	fmt.Print("Verbose output? (y/N): ")
	verboseChoice, _ := reader.ReadString('\n')
	verboseChoice = strings.TrimSpace(strings.ToLower(verboseChoice))

	// Ask for custom command
	fmt.Print("Custom command to run on changes (optional): ")
	commandInput, _ := reader.ReadString('\n')
	commandInput = strings.TrimSpace(commandInput)

	// Build command args
	args := []string{}
	if verboseChoice == "y" || verboseChoice == "yes" {
		args = append(args, "--verbose")
	}
	if commandInput != "" {
		args = append(args, "--command="+commandInput)
	}

	fmt.Printf("\nRunning: templar watch %s\n", strings.Join(args, " "))

	return watchCmd.RunE(watchCmd, args)
}
