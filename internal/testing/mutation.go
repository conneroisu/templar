package testing

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MutationTestResult represents the result of a mutation test
type MutationTestResult struct {
	MutationID   string        `json:"mutation_id"`
	OriginalCode string        `json:"original_code"`
	MutatedCode  string        `json:"mutated_code"`
	File         string        `json:"file"`
	Line         int           `json:"line"`
	Column       int           `json:"column"`
	TestsPassed  bool          `json:"tests_passed"`
	TestOutput   string        `json:"test_output"`
	Duration     time.Duration `json:"duration"`
	Operator     string        `json:"operator"`
	Killed       bool          `json:"killed"` // true if tests caught the mutation
}

// MutationTestSummary provides aggregate results of mutation testing
type MutationTestSummary struct {
	TotalMutations    int                  `json:"total_mutations"`
	KilledMutations   int                  `json:"killed_mutations"`
	SurvivedMutations int                  `json:"survived_mutations"`
	MutationScore     float64              `json:"mutation_score"`
	Duration          time.Duration        `json:"duration"`
	Results           []MutationTestResult `json:"results"`
	WeakSpots         []WeakSpot           `json:"weak_spots"`
}

// WeakSpot represents code that survived mutations (indicating weak tests)
type WeakSpot struct {
	File        string   `json:"file"`
	Line        int      `json:"line"`
	Function    string   `json:"function"`
	Mutations   int      `json:"mutations"`
	Survivors   int      `json:"survivors"`
	WeakScore   float64  `json:"weak_score"`
	Suggestions []string `json:"suggestions"`
}

// MutationTester performs mutation testing to assess test quality
type MutationTester struct {
	ProjectRoot  string
	Packages     []string
	Timeout      time.Duration
	MaxMutations int
	SkipVendor   bool
	Verbose      bool
}

// NewMutationTester creates a new mutation tester
func NewMutationTester(projectRoot string, packages []string) *MutationTester {
	return &MutationTester{
		ProjectRoot:  projectRoot,
		Packages:     packages,
		Timeout:      30 * time.Second,
		MaxMutations: 100,
		SkipVendor:   true,
		Verbose:      false,
	}
}

// RunMutationTests performs mutation testing on the specified packages
func (mt *MutationTester) RunMutationTests() (*MutationTestSummary, error) {
	start := time.Now()

	summary := &MutationTestSummary{
		Results:   make([]MutationTestResult, 0),
		WeakSpots: make([]WeakSpot, 0),
	}

	// Discover Go files to mutate
	files, err := mt.discoverGoFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to discover Go files: %w", err)
	}

	// Generate mutations for each file
	mutations := make([]Mutation, 0)
	for _, file := range files {
		fileMutations, err := mt.generateMutations(file)
		if err != nil {
			if mt.Verbose {
				fmt.Printf("Warning: failed to generate mutations for %s: %v\n", file, err)
			}
			continue
		}
		mutations = append(mutations, fileMutations...)
	}

	// Limit mutations if too many
	if len(mutations) > mt.MaxMutations {
		mutations = mutations[:mt.MaxMutations]
	}

	summary.TotalMutations = len(mutations)

	// Run baseline tests to ensure they pass
	if err := mt.runBaselineTests(); err != nil {
		return nil, fmt.Errorf("baseline tests failed: %w", err)
	}

	// Apply each mutation and run tests
	for i, mutation := range mutations {
		if mt.Verbose {
			fmt.Printf("Testing mutation %d/%d: %s\n", i+1, len(mutations), mutation.Description)
		}

		result := mt.testMutation(mutation)
		summary.Results = append(summary.Results, result)

		if result.Killed {
			summary.KilledMutations++
		} else {
			summary.SurvivedMutations++
		}
	}

	// Calculate mutation score
	if summary.TotalMutations > 0 {
		summary.MutationScore = float64(summary.KilledMutations) / float64(summary.TotalMutations) * 100
	}

	summary.Duration = time.Since(start)

	// Analyze weak spots
	summary.WeakSpots = mt.analyzeWeakSpots(summary.Results)

	return summary, nil
}

// Mutation represents a code mutation
type Mutation struct {
	ID           string
	File         string
	Line         int
	Column       int
	OriginalCode string
	MutatedCode  string
	Operator     string
	Description  string
}

// discoverGoFiles finds all Go files in the specified packages
func (mt *MutationTester) discoverGoFiles() ([]string, error) {
	files := make([]string, 0)

	for _, pkg := range mt.Packages {
		pkgPath := filepath.Join(mt.ProjectRoot, pkg)

		err := filepath.Walk(pkgPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip vendor directories
			if mt.SkipVendor && strings.Contains(path, "vendor/") {
				return filepath.SkipDir
			}

			// Skip test files (we want to mutate source, not tests)
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}

			// Only include Go files
			if strings.HasSuffix(path, ".go") {
				files = append(files, path)
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

// generateMutations creates mutations for a Go file
func (mt *MutationTester) generateMutations(filename string) ([]Mutation, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, content, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	mutations := make([]Mutation, 0)
	mutationID := 0

	// AST visitor to find mutation opportunities
	ast.Inspect(node, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.BinaryExpr:
			// Mutate comparison operators
			mutations = append(mutations, mt.mutateBinaryExpr(filename, fset, node, &mutationID)...)
		case *ast.IfStmt:
			// Mutate conditional statements
			mutations = append(mutations, mt.mutateIfStmt(filename, fset, node, &mutationID)...)
		case *ast.BasicLit:
			// Mutate literals
			mutations = append(mutations, mt.mutateBasicLit(filename, fset, node, &mutationID)...)
		}
		return true
	})

	return mutations, nil
}

// mutateBinaryExpr creates mutations for binary expressions
func (mt *MutationTester) mutateBinaryExpr(filename string, fset *token.FileSet, expr *ast.BinaryExpr, mutationID *int) []Mutation {
	mutations := make([]Mutation, 0)
	position := fset.Position(expr.Pos())

	// Get original code
	var buf bytes.Buffer
	format.Node(&buf, fset, expr)
	originalCode := buf.String()

	// Mutation mappings for comparison operators
	mutations = append(mutations, mt.createOperatorMutations(filename, position, expr, originalCode, mutationID)...)

	return mutations
}

// mutateIfStmt creates mutations for if statements
func (mt *MutationTester) mutateIfStmt(filename string, fset *token.FileSet, stmt *ast.IfStmt, mutationID *int) []Mutation {
	mutations := make([]Mutation, 0)
	position := fset.Position(stmt.Pos())

	// Get original condition
	var buf bytes.Buffer
	format.Node(&buf, fset, stmt.Cond)
	originalCode := buf.String()

	// Create negation mutation
	*mutationID++
	mutation := Mutation{
		ID:           fmt.Sprintf("MUT_%d", *mutationID),
		File:         filename,
		Line:         position.Line,
		Column:       position.Column,
		OriginalCode: originalCode,
		MutatedCode:  "!(" + originalCode + ")",
		Operator:     "NEGATE_CONDITION",
		Description:  fmt.Sprintf("Negate condition at %s:%d", filepath.Base(filename), position.Line),
	}
	mutations = append(mutations, mutation)

	return mutations
}

// mutateBasicLit creates mutations for basic literals
func (mt *MutationTester) mutateBasicLit(filename string, fset *token.FileSet, lit *ast.BasicLit, mutationID *int) []Mutation {
	mutations := make([]Mutation, 0)
	position := fset.Position(lit.Pos())

	switch lit.Kind {
	case token.INT:
		// Mutate integer literals
		if val, err := strconv.Atoi(lit.Value); err == nil {
			mutations = append(mutations, mt.createIntegerMutations(filename, position, lit.Value, val, mutationID)...)
		}
	case token.STRING:
		// Mutate string literals
		mutations = append(mutations, mt.createStringMutations(filename, position, lit.Value, mutationID)...)
	}

	return mutations
}

// createOperatorMutations creates mutations for operators
func (mt *MutationTester) createOperatorMutations(filename string, position token.Position, expr *ast.BinaryExpr, originalCode string, mutationID *int) []Mutation {
	mutations := make([]Mutation, 0)

	operatorMutations := map[token.Token][]token.Token{
		token.EQL:  {token.NEQ, token.LSS, token.GTR},
		token.NEQ:  {token.EQL, token.LSS, token.GTR},
		token.LSS:  {token.LEQ, token.GTR, token.GEQ, token.EQL},
		token.LEQ:  {token.LSS, token.GTR, token.GEQ},
		token.GTR:  {token.GEQ, token.LSS, token.LEQ, token.EQL},
		token.GEQ:  {token.GTR, token.LSS, token.LEQ},
		token.ADD:  {token.SUB, token.MUL, token.QUO},
		token.SUB:  {token.ADD, token.MUL, token.QUO},
		token.MUL:  {token.QUO, token.ADD, token.SUB},
		token.QUO:  {token.MUL, token.REM},
		token.LAND: {token.LOR},
		token.LOR:  {token.LAND},
	}

	if replacements, exists := operatorMutations[expr.Op]; exists {
		for _, replacement := range replacements {
			*mutationID++

			// Create mutated expression
			var leftBuf, rightBuf bytes.Buffer
			format.Node(&leftBuf, token.NewFileSet(), expr.X)
			format.Node(&rightBuf, token.NewFileSet(), expr.Y)

			mutatedCode := leftBuf.String() + " " + replacement.String() + " " + rightBuf.String()

			mutation := Mutation{
				ID:           fmt.Sprintf("MUT_%d", *mutationID),
				File:         filename,
				Line:         position.Line,
				Column:       position.Column,
				OriginalCode: originalCode,
				MutatedCode:  mutatedCode,
				Operator:     fmt.Sprintf("REPLACE_%s_WITH_%s", expr.Op.String(), replacement.String()),
				Description:  fmt.Sprintf("Replace %s with %s at %s:%d", expr.Op.String(), replacement.String(), filepath.Base(filename), position.Line),
			}
			mutations = append(mutations, mutation)
		}
	}

	return mutations
}

// createIntegerMutations creates mutations for integer literals
func (mt *MutationTester) createIntegerMutations(filename string, position token.Position, original string, value int, mutationID *int) []Mutation {
	mutations := make([]Mutation, 0)

	// Common integer mutations
	intMutations := []int{value + 1, value - 1, 0, 1, -1}

	for _, mutated := range intMutations {
		if mutated == value {
			continue // Skip if same as original
		}

		*mutationID++
		mutation := Mutation{
			ID:           fmt.Sprintf("MUT_%d", *mutationID),
			File:         filename,
			Line:         position.Line,
			Column:       position.Column,
			OriginalCode: original,
			MutatedCode:  strconv.Itoa(mutated),
			Operator:     "REPLACE_INTEGER",
			Description:  fmt.Sprintf("Replace %s with %d at %s:%d", original, mutated, filepath.Base(filename), position.Line),
		}
		mutations = append(mutations, mutation)
	}

	return mutations
}

// createStringMutations creates mutations for string literals
func (mt *MutationTester) createStringMutations(filename string, position token.Position, original string, mutationID *int) []Mutation {
	mutations := make([]Mutation, 0)

	// String mutations
	stringMutations := []string{`""`, `"X"`, `"mutated"`}

	for _, mutated := range stringMutations {
		if mutated == original {
			continue
		}

		*mutationID++
		mutation := Mutation{
			ID:           fmt.Sprintf("MUT_%d", *mutationID),
			File:         filename,
			Line:         position.Line,
			Column:       position.Column,
			OriginalCode: original,
			MutatedCode:  mutated,
			Operator:     "REPLACE_STRING",
			Description:  fmt.Sprintf("Replace %s with %s at %s:%d", original, mutated, filepath.Base(filename), position.Line),
		}
		mutations = append(mutations, mutation)
	}

	return mutations
}

// runBaselineTests runs tests to ensure they pass before mutation testing
func (mt *MutationTester) runBaselineTests() error {
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = mt.ProjectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("baseline tests failed: %s", string(output))
	}

	return nil
}

// testMutation applies a mutation and runs tests
func (mt *MutationTester) testMutation(mutation Mutation) MutationTestResult {
	start := time.Now()

	result := MutationTestResult{
		MutationID:   mutation.ID,
		OriginalCode: mutation.OriginalCode,
		MutatedCode:  mutation.MutatedCode,
		File:         mutation.File,
		Line:         mutation.Line,
		Column:       mutation.Column,
		Operator:     mutation.Operator,
	}

	// Apply mutation
	if err := mt.applyMutation(mutation); err != nil {
		result.TestOutput = fmt.Sprintf("Failed to apply mutation: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	// Run tests
	testsPassed, output := mt.runTests()
	result.TestsPassed = testsPassed
	result.TestOutput = output
	result.Killed = !testsPassed // Mutation is "killed" if tests fail

	// Restore original code
	mt.restoreMutation(mutation)

	result.Duration = time.Since(start)
	return result
}

// applyMutation modifies the source file with the mutation
func (mt *MutationTester) applyMutation(mutation Mutation) error {
	content, err := os.ReadFile(mutation.File)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	if mutation.Line <= 0 || mutation.Line > len(lines) {
		return fmt.Errorf("invalid line number: %d", mutation.Line)
	}

	// Simple string replacement for the mutation
	// This is a simplified approach - a more robust implementation would use AST manipulation
	line := lines[mutation.Line-1]
	mutatedLine := strings.Replace(line, mutation.OriginalCode, mutation.MutatedCode, 1)
	lines[mutation.Line-1] = mutatedLine

	mutatedContent := strings.Join(lines, "\n")
	return os.WriteFile(mutation.File, []byte(mutatedContent), 0644)
}

// restoreMutation restores the original code after testing
func (mt *MutationTester) restoreMutation(mutation Mutation) error {
	content, err := os.ReadFile(mutation.File)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	if mutation.Line <= 0 || mutation.Line > len(lines) {
		return fmt.Errorf("invalid line number: %d", mutation.Line)
	}

	// Restore original code
	line := lines[mutation.Line-1]
	restoredLine := strings.Replace(line, mutation.MutatedCode, mutation.OriginalCode, 1)
	lines[mutation.Line-1] = restoredLine

	restoredContent := strings.Join(lines, "\n")
	return os.WriteFile(mutation.File, []byte(restoredContent), 0644)
}

// runTests executes the test suite and returns pass/fail status and output
func (mt *MutationTester) runTests() (bool, string) {
	cmd := exec.Command("go", "test", "-short", "./...")
	cmd.Dir = mt.ProjectRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String() + stderr.String()
	passed := err == nil

	return passed, output
}

// analyzeWeakSpots identifies code areas with poor test coverage based on surviving mutations
func (mt *MutationTester) analyzeWeakSpots(results []MutationTestResult) []WeakSpot {
	// Group mutations by file and line
	locationMap := make(map[string]map[int][]MutationTestResult)

	for _, result := range results {
		if locationMap[result.File] == nil {
			locationMap[result.File] = make(map[int][]MutationTestResult)
		}
		locationMap[result.File][result.Line] = append(locationMap[result.File][result.Line], result)
	}

	weakSpots := make([]WeakSpot, 0)

	// Analyze each location
	for file, lines := range locationMap {
		for line, mutations := range lines {
			if len(mutations) == 0 {
				continue
			}

			survivors := 0
			for _, mutation := range mutations {
				if !mutation.Killed {
					survivors++
				}
			}

			// Consider it a weak spot if more than 50% of mutations survived
			weakScore := float64(survivors) / float64(len(mutations)) * 100
			if weakScore > 50 {
				weakSpot := WeakSpot{
					File:        file,
					Line:        line,
					Function:    mt.extractFunctionName(file, line),
					Mutations:   len(mutations),
					Survivors:   survivors,
					WeakScore:   weakScore,
					Suggestions: mt.generateTestSuggestions(mutations),
				}
				weakSpots = append(weakSpots, weakSpot)
			}
		}
	}

	return weakSpots
}

// extractFunctionName attempts to determine the function containing the given line
func (mt *MutationTester) extractFunctionName(filename string, line int) string {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "unknown"
	}

	lines := strings.Split(string(content), "\n")

	// Look backwards from the target line to find function declaration
	funcPattern := regexp.MustCompile(`^func\s+(\w+)`)
	for i := line - 1; i >= 0 && i < len(lines); i-- {
		if matches := funcPattern.FindStringSubmatch(lines[i]); len(matches) > 1 {
			return matches[1]
		}
	}

	return "unknown"
}

// generateTestSuggestions provides suggestions for improving test coverage
func (mt *MutationTester) generateTestSuggestions(mutations []MutationTestResult) []string {
	suggestions := make([]string, 0)

	operatorCounts := make(map[string]int)
	for _, mutation := range mutations {
		if !mutation.Killed {
			operatorCounts[mutation.Operator]++
		}
	}

	// Generate suggestions based on surviving mutation types
	for operator, _ := range operatorCounts {
		switch {
		case strings.Contains(operator, "NEGATE_CONDITION"):
			suggestions = append(suggestions, "Add tests for both true and false branches of conditional statements")
		case strings.Contains(operator, "REPLACE_EQL_WITH_NEQ"):
			suggestions = append(suggestions, "Add tests that verify exact equality conditions")
		case strings.Contains(operator, "REPLACE_INTEGER"):
			suggestions = append(suggestions, "Add boundary value tests for integer comparisons")
		case strings.Contains(operator, "REPLACE_STRING"):
			suggestions = append(suggestions, "Add tests that verify specific string values")
		default:
			suggestions = append(suggestions, fmt.Sprintf("Add tests for %s operations", operator))
		}
	}

	return suggestions
}

// GenerateReport creates a detailed mutation testing report
func (mt *MutationTester) GenerateReport(summary *MutationTestSummary, outputPath string) error {
	var report strings.Builder

	// Header
	report.WriteString("# Mutation Testing Report\n\n")
	report.WriteString(fmt.Sprintf("**Generated:** %s\n", time.Now().Format("2006-01-02 15:04:05")))
	report.WriteString(fmt.Sprintf("**Duration:** %v\n", summary.Duration))
	report.WriteString("\n")

	// Summary
	report.WriteString("## Summary\n\n")
	report.WriteString(fmt.Sprintf("- **Total Mutations:** %d\n", summary.TotalMutations))
	report.WriteString(fmt.Sprintf("- **Killed Mutations:** %d\n", summary.KilledMutations))
	report.WriteString(fmt.Sprintf("- **Survived Mutations:** %d\n", summary.SurvivedMutations))
	report.WriteString(fmt.Sprintf("- **Mutation Score:** %.2f%%\n", summary.MutationScore))
	report.WriteString("\n")

	// Quality assessment
	report.WriteString("## Test Quality Assessment\n\n")
	switch {
	case summary.MutationScore >= 90:
		report.WriteString("🟢 **Excellent** - Your tests are very effective at catching defects.\n")
	case summary.MutationScore >= 80:
		report.WriteString("🟡 **Good** - Your tests catch most defects, but there's room for improvement.\n")
	case summary.MutationScore >= 70:
		report.WriteString("🟠 **Fair** - Your tests catch some defects, but significant improvements are needed.\n")
	default:
		report.WriteString("🔴 **Poor** - Your tests are not effective at catching defects. Major improvements needed.\n")
	}
	report.WriteString("\n")

	// Weak spots
	if len(summary.WeakSpots) > 0 {
		report.WriteString("## Weak Spots\n\n")
		report.WriteString("These areas have poor test coverage based on surviving mutations:\n\n")

		for _, spot := range summary.WeakSpots {
			report.WriteString(fmt.Sprintf("### %s:%d (%s)\n", filepath.Base(spot.File), spot.Line, spot.Function))
			report.WriteString(fmt.Sprintf("- **Mutations:** %d\n", spot.Mutations))
			report.WriteString(fmt.Sprintf("- **Survivors:** %d\n", spot.Survivors))
			report.WriteString(fmt.Sprintf("- **Weak Score:** %.1f%%\n", spot.WeakScore))

			if len(spot.Suggestions) > 0 {
				report.WriteString("- **Suggestions:**\n")
				for _, suggestion := range spot.Suggestions {
					report.WriteString(fmt.Sprintf("  - %s\n", suggestion))
				}
			}
			report.WriteString("\n")
		}
	}

	// Detailed results
	report.WriteString("## Detailed Results\n\n")
	survivedCount := 0
	for _, result := range summary.Results {
		if !result.Killed {
			survivedCount++
			if survivedCount <= 10 { // Limit to first 10 survivors
				report.WriteString(fmt.Sprintf("### %s\n", result.MutationID))
				report.WriteString(fmt.Sprintf("- **File:** %s:%d\n", filepath.Base(result.File), result.Line))
				report.WriteString(fmt.Sprintf("- **Operator:** %s\n", result.Operator))
				report.WriteString(fmt.Sprintf("- **Original:** `%s`\n", result.OriginalCode))
				report.WriteString(fmt.Sprintf("- **Mutated:** `%s`\n", result.MutatedCode))
				report.WriteString(fmt.Sprintf("- **Status:** %s\n", func() string {
					if result.Killed {
						return "KILLED ✅"
					}
					return "SURVIVED ❌"
				}()))
				report.WriteString("\n")
			}
		}
	}

	if survivedCount > 10 {
		report.WriteString(fmt.Sprintf("*... and %d more surviving mutations*\n\n", survivedCount-10))
	}

	// Write report to file
	return os.WriteFile(outputPath, []byte(report.String()), 0644)
}
