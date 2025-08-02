package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/mockdata"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/conneroisu/templar/internal/types"
	"github.com/spf13/cobra"
)

// Error message constants specific to mocks command.
const (
	errorLoadConfigurationMocks = "failed to load configuration: %w"
	errorCreateMockDataDir      = "failed to create mock data directory: %w"
	errorGenerateMockData       = "failed to generate mock data: %w"
	errorWriteMockDataFile      = "failed to write mock data file: %w"
	errorScanComponents         = "failed to scan components: %w"
	errorComponentNotFound      = "component not found: %s"
	errorValidateMockData       = "failed to validate mock data: %w"
	errorLoadTemplate           = "failed to load template: %w"
	errorSaveTemplate           = "failed to save template: %w"
	errorDeleteTemplate         = "failed to delete template: %w"
	errorListTemplates          = "failed to list templates: %w"
)

// Mock data directory constants.
const (
	MockDataDir          = "mocks"
	TemplateDir          = ".templar/templates"
	MockDataExtension    = ".json"
	TemplateExtension    = ".yml"
	DefaultOutputFormat  = "json"
	DefaultMockDataCount = 1
)

var mocksCmd = &cobra.Command{
	Use:   "mocks",
	Short: "Mock data generation and management",
	Long: `Generate and manage mock data for component development and testing.

The mocks command provides comprehensive mock data generation capabilities:
• Generate realistic mock data for components
• Validate mock data against component schemas
• Manage custom mock data templates
• Export mock data to various formats

Supports intelligent mock data generation based on parameter names and types.
Templates enable reusable mock data patterns across components.

Examples:
  templar mocks generate Button                    # Generate mock data for Button component
  templar mocks generate --all                     # Generate mock data for all components
  templar mocks generate Button --output ./data/   # Export to specific directory
  templar mocks generate Button --count 5          # Generate 5 mock data instances
  templar mocks generate Button --template user    # Use specific template
  templar mocks templates list                     # List available templates
  templar mocks templates create user              # Create new template
  templar mocks validate ./mocks/button.json      # Validate mock data

Template Features:
  • Inheritance support for template composition
  • Faker integration for realistic data generation
  • Custom patterns for domain-specific data
  • Schema validation against component parameters

Integration:
  • Works with component preview system
  • Supports hot reload development workflow
  • Validates against component parameter schemas
  • Exports to JSON, YAML, and custom formats

See 'templar mocks generate --help' for generation options.
See 'templar mocks templates --help' for template management.`,
}

var mocksGenerateCmd = &cobra.Command{
	Use:   "generate [component]",
	Short: "Generate mock data for components",
	Long: `Generate realistic mock data for one or more components.

Analyzes component parameters and generates appropriate mock data using:
• Intelligent pattern matching based on parameter names
• Type-aware data generation for Go types
• Faker integration for realistic values (names, emails, addresses)
• Custom templates for reusable mock patterns
• Schema validation against component definitions

Output formats:
  • JSON (default) - Structured data for APIs and storage
  • YAML - Human-readable configuration format
  • TypeScript - Type definitions with mock data
  • Raw - Direct component parameter format

Generation options:
  • --count: Generate multiple instances for testing
  • --template: Use predefined templates for consistency
  • --seed: Deterministic generation for reproducible tests
  • --validate: Validate generated data against schemas

Examples:
  templar mocks generate Button                           # Generate for Button component
  templar mocks generate Button --count 10                # Generate 10 instances
  templar mocks generate Button --template user           # Use user template
  templar mocks generate --all --output ./test-data/      # Generate for all components
  templar mocks generate Card --format yaml               # Export as YAML
  templar mocks generate Form --seed 12345                # Reproducible generation
  templar mocks generate Button --no-validate             # Skip validation

The generated mock data respects component parameter types and uses semantic
pattern matching to create realistic values that enhance development and testing.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMocksGenerate,
}

var mocksTemplatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "Manage mock data templates",
	Long: `Manage custom mock data templates for reusable mock patterns.

Templates provide a way to define consistent mock data structures that can be
reused across multiple components. They support inheritance, allowing specialized
templates to build upon base templates.

Template features:
  • YAML-based definition format for human readability
  • Inheritance system with parent-child relationships
  • Faker integration with {{expression}} syntax
  • Version control support for template evolution
  • Validation against component schemas

Available subcommands:
  • list     - Show all available templates
  • create   - Create a new template interactively
  • edit     - Modify an existing template
  • delete   - Remove a template
  • show     - Display template details
  • validate - Validate template syntax and structure

Template syntax:
  name: user-profile
  description: Template for user profile components
  extends: base-template
  fields:
    firstName: "{{person.firstName}}"
    lastName: "{{person.lastName}}"
    email: "{{internet.email}}"
    avatar: "{{internet.avatar}}"
    active: "{{datatype.boolean}}"

Examples:
  templar mocks templates list                    # List all templates
  templar mocks templates create user             # Create user template
  templar mocks templates show user               # Display user template
  templar mocks templates delete old-template     # Remove template
  templar mocks templates validate user           # Validate template`,
}

var mocksTemplatesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available mock data templates",
	Long: `List all available mock data templates with their descriptions and metadata.

Displays templates from:
  • Built-in templates (default, user, article, product, company, event)
  • Custom project templates (.templar/templates/)
  • Inherited templates with resolution chains

Output includes:
  • Template name and description
  • Template tags for categorization
  • Version information
  • Inheritance relationships
  • Usage statistics

Examples:
  templar mocks templates list                    # List all templates
  templar mocks templates list --format table    # Tabular output
  templar mocks templates list --tags user       # Filter by tags
  templar mocks templates list --verbose         # Show detailed information`,
	RunE: runMocksTemplatesList,
}

var mocksTemplatesCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new mock data template",
	Long: `Create a new mock data template interactively or from specification.

Interactive mode guides you through template creation:
  • Template metadata (name, description, tags)
  • Field definitions with type hints
  • Faker expression suggestions
  • Inheritance configuration
  • Validation and preview

Non-interactive mode accepts template files:
  • YAML template specification
  • JSON template format
  • Template inheritance from existing templates

Template structure:
  name: Required unique identifier
  description: Human-readable description
  extends: Optional parent template name
  fields: Map of field names to faker expressions
  tags: Optional categorization tags
  version: Optional semantic version

Examples:
  templar mocks templates create user                     # Interactive creation
  templar mocks templates create user --from base        # Extend base template
  templar mocks templates create user --file template.yml # From file
  templar mocks templates create user --fields name,email # Quick creation`,
	Args: cobra.ExactArgs(1),
	RunE: runMocksTemplatesCreate,
}

var mocksTemplatesDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a mock data template",
	Long: `Delete a mock data template from the project.

Removes the template file and clears it from the template cache.
Built-in templates cannot be deleted but can be overridden by custom templates.

Safety checks:
  • Confirms deletion before proceeding
  • Checks for template dependencies
  • Prevents deletion of templates in use
  • Backup option for recovery

Examples:
  templar mocks templates delete old-template             # Delete template
  templar mocks templates delete user --force             # Skip confirmation
  templar mocks templates delete user --backup            # Create backup first`,
	Args: cobra.ExactArgs(1),
	RunE: runMocksTemplatesDelete,
}

var mocksValidateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate mock data against component schemas",
	Long: `Validate mock data files against component parameter schemas.

Performs comprehensive validation:
  • Type checking against Go parameter types
  • Required parameter presence validation
  • Semantic validation for emails, URLs, etc.
  • Template expression syntax validation
  • Cross-reference validation for relationships

Validation levels:
  • basic: Type and presence checking
  • strict: Includes semantic validation
  • comprehensive: Full validation with suggestions

Examples:
  templar mocks validate ./mocks/button.json              # Validate file
  templar mocks validate ./mocks/ --recursive             # Validate directory
  templar mocks validate button.json --level strict      # Strict validation
  templar mocks validate data.json --component Button     # Validate against specific component`,
	Args: cobra.ExactArgs(1),
	RunE: runMocksValidate,
}

func init() {
	rootCmd.AddCommand(mocksCmd)

	// Add subcommands
	mocksCmd.AddCommand(mocksGenerateCmd)
	mocksCmd.AddCommand(mocksTemplatesCmd)
	mocksCmd.AddCommand(mocksValidateCmd)

	// Add template subcommands
	mocksTemplatesCmd.AddCommand(mocksTemplatesListCmd)
	mocksTemplatesCmd.AddCommand(mocksTemplatesCreateCmd)
	mocksTemplatesCmd.AddCommand(mocksTemplatesDeleteCmd)

	// Add flags for generate command
	mocksGenerateCmd.Flags().Bool("all", false, "Generate mock data for all components")
	mocksGenerateCmd.Flags().StringP("output", "o", MockDataDir, "Output directory for mock data files")
	mocksGenerateCmd.Flags().StringP("format", "f", DefaultOutputFormat, "Output format (json, yaml, typescript)")
	mocksGenerateCmd.Flags().IntP("count", "c", DefaultMockDataCount, "Number of mock data instances to generate")
	mocksGenerateCmd.Flags().StringP("template", "t", "", "Template to use for mock data generation")
	mocksGenerateCmd.Flags().Int64("seed", 0, "Random seed for deterministic generation")
	mocksGenerateCmd.Flags().Bool("validate", true, "Validate generated mock data against schemas")
	mocksGenerateCmd.Flags().Bool("overwrite", false, "Overwrite existing mock data files")
	mocksGenerateCmd.Flags().StringSlice("tags", []string{}, "Filter components by tags")

	// Add flags for templates list command
	mocksTemplatesListCmd.Flags().StringP("format", "f", "table", "Output format (table, json, yaml)")
	mocksTemplatesListCmd.Flags().StringSlice("tags", []string{}, "Filter templates by tags")
	mocksTemplatesListCmd.Flags().Bool("verbose", false, "Show detailed template information")

	// Add flags for templates create command
	mocksTemplatesCreateCmd.Flags().String("from", "", "Extend from existing template")
	mocksTemplatesCreateCmd.Flags().String("file", "", "Create template from file")
	mocksTemplatesCreateCmd.Flags().StringSlice("fields", []string{}, "Quick field creation (name,email,phone)")
	mocksTemplatesCreateCmd.Flags().StringSlice("tags", []string{}, "Template tags")
	mocksTemplatesCreateCmd.Flags().String("description", "", "Template description")

	// Add flags for templates delete command
	mocksTemplatesDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")
	mocksTemplatesDeleteCmd.Flags().Bool("backup", false, "Create backup before deletion")

	// Add flags for validate command
	mocksValidateCmd.Flags().Bool("recursive", false, "Validate directory recursively")
	mocksValidateCmd.Flags().String("level", "basic", "Validation level (basic, strict, comprehensive)")
	mocksValidateCmd.Flags().String("component", "", "Validate against specific component")
	mocksValidateCmd.Flags().Bool("fix", false, "Attempt to fix validation errors")
}

func runMocksGenerate(cmd *cobra.Command, args []string) error {
	// Parse flags
	generateAll, _ := cmd.Flags().GetBool("all")
	outputDir, _ := cmd.Flags().GetString("output")
	format, _ := cmd.Flags().GetString("format")
	count, _ := cmd.Flags().GetInt("count")
	templateName, _ := cmd.Flags().GetString("template")
	seed, _ := cmd.Flags().GetInt64("seed")
	validateData, _ := cmd.Flags().GetBool("validate")
	overwrite, _ := cmd.Flags().GetBool("overwrite")
	tags, _ := cmd.Flags().GetStringSlice("tags")

	// Validate arguments
	if !generateAll && len(args) == 0 {
		return errors.New("component name required when not using --all flag")
	}

	if generateAll && len(args) > 0 {
		return errors.New("cannot specify component name when using --all flag")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf(errorLoadConfigurationMocks, err)
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf(errorCreateMockDataDir, err)
	}

	// Create component registry and scanner
	componentRegistry := registry.NewComponentRegistry()
	componentScanner := scanner.NewComponentScanner(componentRegistry)

	// Scan all configured paths
	fmt.Println("📁 Scanning for components...")
	for _, scanPath := range cfg.Components.ScanPaths {
		if err := componentScanner.ScanDirectory(scanPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to scan directory %s: %v\n", scanPath, err)
		}
	}

	// Create mock data generator
	mockConfig := mockdata.DefaultMockDataConfig()
	if seed != 0 {
		mockConfig.Seed = seed
	}
	generator := mockdata.NewIntelligentMockGenerator(mockConfig)

	// Generate mock data
	if generateAll {
		return generateMockDataForAll(componentRegistry, generator, outputDir, format, count, templateName, validateData, overwrite, tags)
	}

	componentName := args[0]

	return generateMockDataForComponent(componentRegistry, generator, componentName, outputDir, format, count, templateName, validateData, overwrite)
}

func generateMockDataForAll(
	registry *registry.ComponentRegistry,
	generator *mockdata.IntelligentMockGenerator,
	outputDir string,
	format string,
	count int,
	templateName string,
	validateData bool,
	overwrite bool,
	tags []string,
) error {
	components := registry.GetAll()
	if len(components) == 0 {
		fmt.Println("No components found")

		return nil
	}

	// Filter by tags if specified
	if len(tags) > 0 {
		components = filterComponentsByTags(components, tags)
	}

	fmt.Printf("🎲 Generating mock data for %d components...\n", len(components))

	for _, component := range components {
		if err := generateMockDataForComponent(registry, generator, component.Name, outputDir, format, count, templateName, validateData, overwrite); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to generate mock data for %s: %v\n", component.Name, err)
		}
	}

	fmt.Println("✅ Mock data generation completed")

	return nil
}

func generateMockDataForComponent(
	registry *registry.ComponentRegistry,
	generator *mockdata.IntelligentMockGenerator,
	componentName string,
	outputDir string,
	format string,
	count int,
	templateName string,
	validateData bool,
	overwrite bool,
) error {
	// Find component
	component, exists := registry.Get(componentName)
	if !exists {
		return fmt.Errorf(errorComponentNotFound, componentName)
	}

	fmt.Printf("🎭 Generating mock data for: %s\n", componentName)

	// Generate mock data instances
	var mockInstances []map[string]interface{}
	for range count {
		mockData := generator.GenerateForComponent(component)

		// Validate if requested
		if validateData {
			if err := generator.ValidateGenerated(mockData, component); err != nil {
				return fmt.Errorf(errorValidateMockData, err)
			}
		}

		mockInstances = append(mockInstances, mockData)
	}

	// Prepare output data
	var outputData interface{}
	if count == 1 {
		outputData = mockInstances[0]
	} else {
		outputData = mockInstances
	}

	// Generate output filename
	filename := generateMockDataFilename(componentName, format)
	outputPath := filepath.Join(outputDir, filename)

	// Check if file exists and handle overwrite
	if _, err := os.Stat(outputPath); err == nil && !overwrite {
		fmt.Printf("Warning: %s already exists, use --overwrite to replace\n", outputPath)

		return nil
	}

	// Write mock data to file
	if err := writeMockDataFile(outputPath, outputData, format); err != nil {
		return fmt.Errorf(errorWriteMockDataFile, err)
	}

	fmt.Printf("   ✅ Saved to: %s\n", outputPath)

	return nil
}

func runMocksTemplatesList(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	tags, _ := cmd.Flags().GetStringSlice("tags")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Create template manager
	templateManager := mockdata.NewFileTemplateManager()

	// List templates
	templates, err := templateManager.ListTemplates()
	if err != nil {
		return fmt.Errorf(errorListTemplates, err)
	}

	// Filter by tags if specified
	if len(tags) > 0 {
		templates = filterTemplatesByTags(templates, tags)
	}

	// Display templates
	return displayTemplates(templates, format, verbose)
}

func runMocksTemplatesCreate(cmd *cobra.Command, args []string) error {
	templateName := args[0]
	from, _ := cmd.Flags().GetString("from")
	file, _ := cmd.Flags().GetString("file")
	fields, _ := cmd.Flags().GetStringSlice("fields")
	tags, _ := cmd.Flags().GetStringSlice("tags")
	description, _ := cmd.Flags().GetString("description")

	// Create template manager
	templateManager := mockdata.NewFileTemplateManager()

	// Create template based on options
	if file != "" {
		return createTemplateFromFile(templateManager, templateName, file)
	}

	if from != "" {
		return createTemplateFromParent(templateManager, templateName, from, description, tags)
	}

	if len(fields) > 0 {
		return createQuickTemplate(templateManager, templateName, fields, description, tags)
	}

	// Interactive template creation
	return createTemplateInteractive(templateManager, templateName)
}

func runMocksTemplatesDelete(cmd *cobra.Command, args []string) error {
	templateName := args[0]
	force, _ := cmd.Flags().GetBool("force")
	backup, _ := cmd.Flags().GetBool("backup")

	// Create template manager
	templateManager := mockdata.NewFileTemplateManager()

	// Create backup if requested
	if backup {
		if err := backupTemplate(templateManager, templateName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create backup: %v\n", err)
		}
	}

	// Confirm deletion if not forced
	if !force {
		if !confirmTemplateDelete(templateName) {
			fmt.Println("Template deletion cancelled")

			return nil
		}
	}

	// Delete template
	if err := templateManager.DeleteTemplate(templateName); err != nil {
		return fmt.Errorf(errorDeleteTemplate, err)
	}

	fmt.Printf("✅ Template '%s' deleted successfully\n", templateName)

	return nil
}

func runMocksValidate(cmd *cobra.Command, args []string) error {
	filePath := args[0]
	recursive, _ := cmd.Flags().GetBool("recursive")
	level, _ := cmd.Flags().GetString("level")
	componentName, _ := cmd.Flags().GetString("component")
	fix, _ := cmd.Flags().GetBool("fix")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf(errorLoadConfigurationMocks, err)
	}

	// Create validator
	validator := mockdata.NewMockDataValidator()

	// Validate based on file type
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to access file: %w", err)
	}

	if fileInfo.IsDir() && recursive {
		return validateDirectory(validator, filePath, level, componentName, fix, cfg)
	}

	return validateFile(validator, filePath, level, componentName, fix, cfg)
}

// Helper functions

func filterComponentsByTags(components []*types.ComponentInfo, tags []string) []*types.ComponentInfo {
	if len(tags) == 0 {
		return components
	}

	// Note: Component tag support would need to be added to ComponentInfo struct
	// For now, return all components as tags aren't implemented yet
	filtered := append([]*types.ComponentInfo{}, components...)

	return filtered
}

func filterTemplatesByTags(templates []*mockdata.MockDataTemplate, tags []string) []*mockdata.MockDataTemplate {
	if len(tags) == 0 {
		return templates
	}

	var filtered []*mockdata.MockDataTemplate
	for _, template := range templates {
		if hasMatchingTags(template.Tags, tags) {
			filtered = append(filtered, template)
		}
	}

	return filtered
}

func hasMatchingTags(templateTags, filterTags []string) bool {
	for _, filterTag := range filterTags {
		for _, templateTag := range templateTags {
			if strings.Contains(strings.ToLower(templateTag), strings.ToLower(filterTag)) {
				return true
			}
		}
	}

	return false
}

func generateMockDataFilename(componentName, format string) string {
	baseFilename := strings.ToLower(componentName)
	switch format {
	case OutputFormatYAML, OutputFormatYML:
		return baseFilename + ".yml"
	case "typescript", "ts":
		return baseFilename + ".mock.ts"
	default:
		return baseFilename + MockDataExtension
	}
}

func writeMockDataFile(outputPath string, data interface{}, format string) error {
	switch format {
	case "yaml", "yml":
		return writeMockDataYAML(outputPath, data)
	case "typescript", "ts":
		return writeMockDataTypeScript(outputPath, data)
	default:
		return writeMockDataJSON(outputPath, data)
	}
}

func writeMockDataJSON(outputPath string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, jsonData, 0600)
}

func writeMockDataYAML(outputPath string, data interface{}) error {
	return errors.New("YAML output not yet implemented - use JSON format instead")
}

func writeMockDataTypeScript(outputPath string, data interface{}) error {
	return errors.New("TypeScript output not yet implemented - use JSON format instead")
}

func displayTemplates(templates []*mockdata.MockDataTemplate, format string, verbose bool) error {
	switch format {
	case "json":
		return displayTemplatesJSON(templates)
	case "yaml":
		return displayTemplatesYAML(templates)
	default:
		return displayTemplatesTable(templates, verbose)
	}
}

func displayTemplatesTable(templates []*mockdata.MockDataTemplate, verbose bool) error {
	if len(templates) == 0 {
		fmt.Println("No templates found")

		return nil
	}

	fmt.Printf("Found %d template(s):\n\n", len(templates))
	for _, template := range templates {
		fmt.Printf("📋 %s\n", template.Name)
		fmt.Printf("   %s\n", template.Description)
		if len(template.Tags) > 0 {
			fmt.Printf("   Tags: %s\n", strings.Join(template.Tags, ", "))
		}
		if template.Version != "" {
			fmt.Printf("   Version: %s\n", template.Version)
		}
		if verbose && len(template.Fields) > 0 {
			fmt.Printf("   Fields: %d\n", len(template.Fields))
			for field := range template.Fields {
				fmt.Printf("     - %s\n", field)
			}
		}
		fmt.Println()
	}

	return nil
}

func displayTemplatesJSON(templates []*mockdata.MockDataTemplate) error {
	jsonData, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(jsonData))

	return nil
}

func displayTemplatesYAML(templates []*mockdata.MockDataTemplate) error {
	return errors.New("YAML display not yet implemented - use JSON format instead")
}

func createTemplateFromFile(templateManager mockdata.TemplateManager, templateName, filePath string) error {
	return errors.New("template creation from file not yet implemented")
}

func createTemplateFromParent(templateManager mockdata.TemplateManager, templateName, parentName, description string, tags []string) error {
	return errors.New("template creation from parent not yet implemented")
}

func createQuickTemplate(templateManager mockdata.TemplateManager, templateName string, fields []string, description string, tags []string) error {
	return errors.New("quick template creation not yet implemented")
}

func createTemplateInteractive(templateManager mockdata.TemplateManager, templateName string) error {
	return errors.New("interactive template creation not yet implemented")
}

func backupTemplate(templateManager mockdata.TemplateManager, templateName string) error {
	return errors.New("template backup not yet implemented")
}

func confirmTemplateDelete(templateName string) bool {
	fmt.Printf("Are you sure you want to delete template '%s'? (y/N): ", templateName)
	var response string
	_, _ = fmt.Scanln(&response)

	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}

func validateDirectory(validator mockdata.MockDataValidator, dirPath, level, componentName string, fix bool, cfg *config.Config) error {
	return errors.New("directory validation not yet implemented")
}

func validateFile(validator mockdata.MockDataValidator, filePath, level, componentName string, fix bool, cfg *config.Config) error {
	return errors.New("file validation not yet implemented")
}
