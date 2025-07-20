package testing

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// BehavioralCoverageAnalyzer analyzes test coverage from a behavioral perspective
type BehavioralCoverageAnalyzer struct {
	ProjectRoot   string
	Packages      []string
	TestPatterns  []string
	coverageData  map[string]*FileCoverage
	complexity    map[string]*ComplexityMetrics
}

// FileCoverage represents behavioral coverage data for a file
type FileCoverage struct {
	File              string                    `json:"file"`
	LineCoverage      float64                   `json:"line_coverage"`
	BranchCoverage    float64                   `json:"branch_coverage"`
	PathCoverage      float64                   `json:"path_coverage"`
	BoundaryTests     []BoundaryTest            `json:"boundary_tests"`
	ErrorPaths        []ErrorPath               `json:"error_paths"`
	StateTransitions  []StateTransition         `json:"state_transitions"`
	ConcurrencyTests  []ConcurrencyTest         `json:"concurrency_tests"`
	ContractTests     []ContractTest            `json:"contract_tests"`
	Behaviors         map[string]BehaviorCoverage `json:"behaviors"`
}

// BehaviorCoverage represents coverage of specific behaviors
type BehaviorCoverage struct {
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Covered        bool      `json:"covered"`
	TestCases      []string  `json:"test_cases"`
	MissingTests   []string  `json:"missing_tests"`
	Priority       Priority  `json:"priority"`
	Complexity     int       `json:"complexity"`
}

// BoundaryTest represents boundary value testing coverage
type BoundaryTest struct {
	Function      string   `json:"function"`
	Parameter     string   `json:"parameter"`
	Boundaries    []string `json:"boundaries"`
	TestedValues  []string `json:"tested_values"`
	MissingTests  []string `json:"missing_tests"`
	Coverage      float64  `json:"coverage"`
}

// ErrorPath represents error handling path coverage
type ErrorPath struct {
	Function     string   `json:"function"`
	ErrorType    string   `json:"error_type"`
	Line         int      `json:"line"`
	Tested       bool     `json:"tested"`
	TestCase     string   `json:"test_case"`
	Suggestions  []string `json:"suggestions"`
}

// StateTransition represents state machine transition coverage
type StateTransition struct {
	From        string   `json:"from"`
	To          string   `json:"to"`
	Trigger     string   `json:"trigger"`
	Tested      bool     `json:"tested"`
	TestCases   []string `json:"test_cases"`
	Priority    Priority `json:"priority"`
}

// ConcurrencyTest represents concurrency testing coverage
type ConcurrencyTest struct {
	Function      string   `json:"function"`
	Pattern       string   `json:"pattern"` // e.g., "race_condition", "deadlock", "resource_leak"
	Tested        bool     `json:"tested"`
	TestCase      string   `json:"test_case"`
	RiskLevel     string   `json:"risk_level"`
	Suggestions   []string `json:"suggestions"`
}

// ContractTest represents design-by-contract testing
type ContractTest struct {
	Function      string   `json:"function"`
	Preconditions []string `json:"preconditions"`
	Postconditions []string `json:"postconditions"`
	Invariants    []string `json:"invariants"`
	TestedContracts []string `json:"tested_contracts"`
	Coverage      float64  `json:"coverage"`
}

// ComplexityMetrics represents complexity analysis for a function
type ComplexityMetrics struct {
	CyclomaticComplexity int     `json:"cyclomatic_complexity"`
	CognitiveComplexity  int     `json:"cognitive_complexity"`
	NestingDepth         int     `json:"nesting_depth"`
	ParameterCount       int     `json:"parameter_count"`
	ReturnCount          int     `json:"return_count"`
	BranchCount          int     `json:"branch_count"`
	RiskScore            float64 `json:"risk_score"`
}

// Priority represents test priority levels
type Priority string

const (
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
)

// NewBehavioralCoverageAnalyzer creates a new behavioral coverage analyzer
func NewBehavioralCoverageAnalyzer(projectRoot string, packages []string) *BehavioralCoverageAnalyzer {
	return &BehavioralCoverageAnalyzer{
		ProjectRoot:  projectRoot,
		Packages:     packages,
		TestPatterns: []string{"*_test.go"},
		coverageData: make(map[string]*FileCoverage),
		complexity:   make(map[string]*ComplexityMetrics),
	}
}

// AnalyzeBehavioralCoverage performs comprehensive behavioral coverage analysis
func (bca *BehavioralCoverageAnalyzer) AnalyzeBehavioralCoverage() (map[string]*FileCoverage, error) {
	// Discover source files
	sourceFiles, err := bca.discoverSourceFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to discover source files: %w", err)
	}

	// Discover test files
	testFiles, err := bca.discoverTestFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to discover test files: %w", err)
	}

	// Analyze each source file
	for _, sourceFile := range sourceFiles {
		coverage, err := bca.analyzeFile(sourceFile, testFiles)
		if err != nil {
			continue // Skip files with analysis errors
		}
		bca.coverageData[sourceFile] = coverage
	}

	// Enhance with cross-file behavioral analysis
	bca.enhanceWithBehavioralPatterns()

	return bca.coverageData, nil
}

// discoverSourceFiles finds all source files to analyze
func (bca *BehavioralCoverageAnalyzer) discoverSourceFiles() ([]string, error) {
	files := make([]string, 0)

	for _, pkg := range bca.Packages {
		pkgPath := filepath.Join(bca.ProjectRoot, pkg)
		
		err := filepath.Walk(pkgPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip vendor and test files
			if strings.Contains(path, "vendor/") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

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

// discoverTestFiles finds all test files
func (bca *BehavioralCoverageAnalyzer) discoverTestFiles() ([]string, error) {
	files := make([]string, 0)

	for _, pkg := range bca.Packages {
		pkgPath := filepath.Join(bca.ProjectRoot, pkg)
		
		err := filepath.Walk(pkgPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if strings.HasSuffix(path, "_test.go") {
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

// analyzeFile performs behavioral coverage analysis for a single file
func (bca *BehavioralCoverageAnalyzer) analyzeFile(sourceFile string, testFiles []string) (*FileCoverage, error) {
	coverage := &FileCoverage{
		File:             sourceFile,
		BoundaryTests:    make([]BoundaryTest, 0),
		ErrorPaths:       make([]ErrorPath, 0),
		StateTransitions: make([]StateTransition, 0),
		ConcurrencyTests: make([]ConcurrencyTest, 0),
		ContractTests:    make([]ContractTest, 0),
		Behaviors:        make(map[string]BehaviorCoverage),
	}

	// Parse source file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, sourceFile, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// Analyze AST for behavioral patterns
	bca.analyzeASTForBehaviors(node, fset, coverage)

	// Analyze corresponding test files
	relatedTestFiles := bca.findRelatedTestFiles(sourceFile, testFiles)
	bca.analyzeTestCoverage(relatedTestFiles, coverage)

	// Calculate coverage metrics
	bca.calculateCoverageMetrics(coverage)

	return coverage, nil
}

// analyzeASTForBehaviors analyzes AST for behavioral patterns
func (bca *BehavioralCoverageAnalyzer) analyzeASTForBehaviors(node *ast.File, fset *token.FileSet, coverage *FileCoverage) {
	ast.Inspect(node, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			bca.analyzeFunctionBehaviors(node, fset, coverage)
		case *ast.IfStmt:
			bca.analyzeConditionalBehaviors(node, fset, coverage)
		case *ast.SwitchStmt:
			bca.analyzeSwitchBehaviors(node, fset, coverage)
		case *ast.TypeSwitchStmt:
			bca.analyzeTypeSwitchBehaviors(node, fset, coverage)
		case *ast.SelectStmt:
			bca.analyzeConcurrencyBehaviors(node, fset, coverage)
		case *ast.ForStmt, *ast.RangeStmt:
			bca.analyzeLoopBehaviors(node, fset, coverage)
		}
		return true
	})
}

// analyzeFunctionBehaviors analyzes function-level behaviors
func (bca *BehavioralCoverageAnalyzer) analyzeFunctionBehaviors(fn *ast.FuncDecl, fset *token.FileSet, coverage *FileCoverage) {
	if fn.Name == nil {
		return
	}

	funcName := fn.Name.Name
	position := fset.Position(fn.Pos())

	// Analyze function signature for boundary tests
	bca.analyzeFunctionSignature(fn, coverage)

	// Analyze error handling patterns
	bca.analyzeErrorHandling(fn, fset, coverage)

	// Analyze concurrency patterns
	bca.analyzeConcurrencyPatterns(fn, fset, coverage)

	// Analyze state machine patterns
	bca.analyzeStateMachinePatterns(fn, fset, coverage)

	// Calculate complexity
	complexity := bca.calculateComplexity(fn)
	bca.complexity[funcName] = complexity

	// Add behavioral expectations based on function patterns
	bca.addBehavioralExpectations(fn, coverage, complexity)
}

// analyzeFunctionSignature analyzes function signature for boundary testing
func (bca *BehavioralCoverageAnalyzer) analyzeFunctionSignature(fn *ast.FuncDecl, coverage *FileCoverage) {
	if fn.Type.Params == nil {
		return
	}

	funcName := fn.Name.Name

	for _, field := range fn.Type.Params.List {
		for _, name := range field.Names {
			paramName := name.Name
			paramType := bca.extractTypeString(field.Type)

			boundaryTest := BoundaryTest{
				Function:     funcName,
				Parameter:    paramName,
				Boundaries:   bca.generateBoundaryValues(paramType),
				TestedValues: make([]string, 0),
				MissingTests: make([]string, 0),
			}

			coverage.BoundaryTests = append(coverage.BoundaryTests, boundaryTest)
		}
	}
}

// analyzeErrorHandling analyzes error handling patterns
func (bca *BehavioralCoverageAnalyzer) analyzeErrorHandling(fn *ast.FuncDecl, fset *token.FileSet, coverage *FileCoverage) {
	ast.Inspect(fn, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt:
			// Look for error checking patterns
			if bca.isErrorCheck(node) {
				position := fset.Position(node.Pos())
				errorPath := ErrorPath{
					Function:    fn.Name.Name,
					ErrorType:   "conditional_error",
					Line:        position.Line,
					Tested:      false,
					Suggestions: []string{"Add test case for error condition"},
				}
				coverage.ErrorPaths = append(coverage.ErrorPaths, errorPath)
			}
		case *ast.CallExpr:
			// Look for function calls that might return errors
			if bca.mightReturnError(node) {
				position := fset.Position(node.Pos())
				errorPath := ErrorPath{
					Function:    fn.Name.Name,
					ErrorType:   "function_call_error",
					Line:        position.Line,
					Tested:      false,
					Suggestions: []string{"Add test case for function call error"},
				}
				coverage.ErrorPaths = append(coverage.ErrorPaths, errorPath)
			}
		}
		return true
	})
}

// analyzeConcurrencyPatterns analyzes concurrency-related behaviors
func (bca *BehavioralCoverageAnalyzer) analyzeConcurrencyPatterns(fn *ast.FuncDecl, fset *token.FileSet, coverage *FileCoverage) {
	ast.Inspect(fn, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GoStmt:
			// Goroutine creation
			concurrencyTest := ConcurrencyTest{
				Function:    fn.Name.Name,
				Pattern:     "goroutine_creation",
				Tested:      false,
				RiskLevel:   "medium",
				Suggestions: []string{"Add test for goroutine behavior", "Test goroutine cleanup"},
			}
			coverage.ConcurrencyTests = append(coverage.ConcurrencyTests, concurrencyTest)
		case *ast.SendStmt:
			// Channel send
			concurrencyTest := ConcurrencyTest{
				Function:    fn.Name.Name,
				Pattern:     "channel_send",
				Tested:      false,
				RiskLevel:   "high",
				Suggestions: []string{"Test channel blocking behavior", "Test channel capacity limits"},
			}
			coverage.ConcurrencyTests = append(coverage.ConcurrencyTests, concurrencyTest)
		case *ast.UnaryExpr:
			if node.Op == token.ARROW {
				// Channel receive
				concurrencyTest := ConcurrencyTest{
					Function:    fn.Name.Name,
					Pattern:     "channel_receive",
					Tested:      false,
					RiskLevel:   "high",
					Suggestions: []string{"Test channel receive timeout", "Test channel close behavior"},
				}
				coverage.ConcurrencyTests = append(coverage.ConcurrencyTests, concurrencyTest)
			}
		}
		return true
	})
}

// analyzeStateMachinePatterns looks for state machine behaviors
func (bca *BehavioralCoverageAnalyzer) analyzeStateMachinePatterns(fn *ast.FuncDecl, fset *token.FileSet, coverage *FileCoverage) {
	// Look for switch statements that might represent state transitions
	ast.Inspect(fn, func(n ast.Node) bool {
		if switchStmt, ok := n.(*ast.SwitchStmt); ok {
			bca.analyzeStateTransitions(switchStmt, fn.Name.Name, coverage)
		}
		return true
	})
}

// analyzeStateTransitions analyzes state transition patterns
func (bca *BehavioralCoverageAnalyzer) analyzeStateTransitions(switchStmt *ast.SwitchStmt, funcName string, coverage *FileCoverage) {
	// This is a simplified state transition analysis
	// In a real implementation, this would be more sophisticated
	
	if switchStmt.Body == nil {
		return
	}

	for _, stmt := range switchStmt.Body.List {
		if caseClause, ok := stmt.(*ast.CaseClause); ok {
			for _, expr := range caseClause.List {
				if ident, ok := expr.(*ast.Ident); ok {
					stateTransition := StateTransition{
						From:      "unknown",
						To:        ident.Name,
						Trigger:   funcName,
						Tested:    false,
						TestCases: make([]string, 0),
						Priority:  PriorityMedium,
					}
					coverage.StateTransitions = append(coverage.StateTransitions, stateTransition)
				}
			}
		}
	}
}

// analyzeConditionalBehaviors analyzes conditional statement behaviors
func (bca *BehavioralCoverageAnalyzer) analyzeConditionalBehaviors(ifStmt *ast.IfStmt, fset *token.FileSet, coverage *FileCoverage) {
	// Analyze branch coverage requirements
	position := fset.Position(ifStmt.Pos())
	
	// Create behavior expectation for both branches
	trueBranch := BehaviorCoverage{
		Name:        fmt.Sprintf("conditional_true_line_%d", position.Line),
		Description: "True branch of conditional statement",
		Covered:     false,
		TestCases:   make([]string, 0),
		MissingTests: []string{"Test case for true condition"},
		Priority:    PriorityHigh,
		Complexity:  1,
	}
	
	falseBranch := BehaviorCoverage{
		Name:        fmt.Sprintf("conditional_false_line_%d", position.Line),
		Description: "False branch of conditional statement",
		Covered:     false,
		TestCases:   make([]string, 0),
		MissingTests: []string{"Test case for false condition"},
		Priority:    PriorityHigh,
		Complexity:  1,
	}

	coverage.Behaviors[trueBranch.Name] = trueBranch
	coverage.Behaviors[falseBranch.Name] = falseBranch
}

// analyzeSwitchBehaviors analyzes switch statement behaviors
func (bca *BehavioralCoverageAnalyzer) analyzeSwitchBehaviors(switchStmt *ast.SwitchStmt, fset *token.FileSet, coverage *FileCoverage) {
	position := fset.Position(switchStmt.Pos())
	
	if switchStmt.Body == nil {
		return
	}

	for i, stmt := range switchStmt.Body.List {
		if caseClause, ok := stmt.(*ast.CaseClause); ok {
			caseBehavior := BehaviorCoverage{
				Name:        fmt.Sprintf("switch_case_%d_line_%d", i, position.Line),
				Description: fmt.Sprintf("Switch case %d", i),
				Covered:     false,
				TestCases:   make([]string, 0),
				MissingTests: []string{fmt.Sprintf("Test case for switch case %d", i)},
				Priority:    PriorityMedium,
				Complexity:  1,
			}
			
			if caseClause.List == nil { // Default case
				caseBehavior.Name = fmt.Sprintf("switch_default_line_%d", position.Line)
				caseBehavior.Description = "Switch default case"
				caseBehavior.Priority = PriorityHigh
			}

			coverage.Behaviors[caseBehavior.Name] = caseBehavior
		}
	}
}

// analyzeTypeSwitchBehaviors analyzes type switch behaviors
func (bca *BehavioralCoverageAnalyzer) analyzeTypeSwitchBehaviors(typeSwitchStmt *ast.TypeSwitchStmt, fset *token.FileSet, coverage *FileCoverage) {
	// Similar to switch analysis but for type switches
	bca.analyzeSwitchBehaviors(&ast.SwitchStmt{Body: typeSwitchStmt.Body}, fset, coverage)
}

// analyzeConcurrencyBehaviors analyzes select statement behaviors
func (bca *BehavioralCoverageAnalyzer) analyzeConcurrencyBehaviors(selectStmt *ast.SelectStmt, fset *token.FileSet, coverage *FileCoverage) {
	position := fset.Position(selectStmt.Pos())
	
	concurrencyTest := ConcurrencyTest{
		Function:    "unknown", // Will be filled by caller
		Pattern:     "select_statement",
		Tested:      false,
		RiskLevel:   "high",
		Suggestions: []string{
			"Test all select cases",
			"Test select timeout behavior",
			"Test select default case",
		},
	}
	coverage.ConcurrencyTests = append(coverage.ConcurrencyTests, concurrencyTest)

	// Create behavior expectations for each select case
	if selectStmt.Body != nil {
		for i, stmt := range selectStmt.Body.List {
			if commClause, ok := stmt.(*ast.CommClause); ok {
				selectBehavior := BehaviorCoverage{
					Name:        fmt.Sprintf("select_case_%d_line_%d", i, position.Line),
					Description: fmt.Sprintf("Select case %d", i),
					Covered:     false,
					TestCases:   make([]string, 0),
					MissingTests: []string{fmt.Sprintf("Test case for select case %d", i)},
					Priority:    PriorityHigh,
					Complexity:  2,
				}
				
				if commClause.Comm == nil { // Default case
					selectBehavior.Name = fmt.Sprintf("select_default_line_%d", position.Line)
					selectBehavior.Description = "Select default case"
				}

				coverage.Behaviors[selectBehavior.Name] = selectBehavior
			}
		}
	}
}

// analyzeLoopBehaviors analyzes loop behaviors
func (bca *BehavioralCoverageAnalyzer) analyzeLoopBehaviors(loopNode ast.Node, fset *token.FileSet, coverage *FileCoverage) {
	position := fset.Position(loopNode.Pos())
	
	// Loop behaviors to test
	loopBehaviors := []string{
		"zero_iterations",
		"one_iteration",
		"multiple_iterations",
		"early_termination",
		"boundary_conditions",
	}

	for _, behavior := range loopBehaviors {
		loopBehavior := BehaviorCoverage{
			Name:        fmt.Sprintf("loop_%s_line_%d", behavior, position.Line),
			Description: fmt.Sprintf("Loop %s behavior", strings.ReplaceAll(behavior, "_", " ")),
			Covered:     false,
			TestCases:   make([]string, 0),
			MissingTests: []string{fmt.Sprintf("Test case for loop %s", behavior)},
			Priority:    PriorityMedium,
			Complexity:  2,
		}
		coverage.Behaviors[loopBehavior.Name] = loopBehavior
	}
}

// calculateComplexity calculates various complexity metrics
func (bca *BehavioralCoverageAnalyzer) calculateComplexity(fn *ast.FuncDecl) *ComplexityMetrics {
	metrics := &ComplexityMetrics{
		CyclomaticComplexity: 1, // Start with 1
		CognitiveComplexity:  0,
		NestingDepth:         0,
		ParameterCount:       0,
		ReturnCount:          0,
		BranchCount:          0,
	}

	// Count parameters
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			metrics.ParameterCount += len(field.Names)
		}
	}

	// Count return values
	if fn.Type.Results != nil {
		for _, field := range fn.Type.Results.List {
			if len(field.Names) == 0 {
				metrics.ReturnCount++
			} else {
				metrics.ReturnCount += len(field.Names)
			}
		}
	}

	// Calculate complexity by walking AST
	currentNesting := 0
	maxNesting := 0

	ast.Inspect(fn, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt:
			metrics.CyclomaticComplexity++
			metrics.BranchCount++
			currentNesting++
			if currentNesting > maxNesting {
				maxNesting = currentNesting
			}
			metrics.CognitiveComplexity += currentNesting
		case *ast.SwitchStmt:
			metrics.CyclomaticComplexity++
			if node.Body != nil {
				metrics.CyclomaticComplexity += len(node.Body.List) - 1
				metrics.BranchCount += len(node.Body.List)
			}
			currentNesting++
			if currentNesting > maxNesting {
				maxNesting = currentNesting
			}
		case *ast.TypeSwitchStmt:
			metrics.CyclomaticComplexity++
			if node.Body != nil {
				metrics.CyclomaticComplexity += len(node.Body.List) - 1
				metrics.BranchCount += len(node.Body.List)
			}
		case *ast.ForStmt, *ast.RangeStmt:
			metrics.CyclomaticComplexity++
			currentNesting++
			if currentNesting > maxNesting {
				maxNesting = currentNesting
			}
			metrics.CognitiveComplexity += currentNesting
		case *ast.SelectStmt:
			metrics.CyclomaticComplexity++
			if node.Body != nil {
				metrics.CyclomaticComplexity += len(node.Body.List) - 1
			}
		}

		// Decrease nesting when leaving certain nodes
		switch n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt:
			// We'll decrease nesting in a post-order traversal
		}

		return true
	})

	metrics.NestingDepth = maxNesting

	// Calculate risk score
	metrics.RiskScore = float64(metrics.CyclomaticComplexity)*0.4 +
		float64(metrics.CognitiveComplexity)*0.3 +
		float64(metrics.NestingDepth)*0.2 +
		float64(metrics.ParameterCount)*0.1

	return metrics
}

// findRelatedTestFiles finds test files related to a source file
func (bca *BehavioralCoverageAnalyzer) findRelatedTestFiles(sourceFile string, testFiles []string) []string {
	related := make([]string, 0)
	
	sourceBase := strings.TrimSuffix(filepath.Base(sourceFile), ".go")
	sourceDir := filepath.Dir(sourceFile)

	for _, testFile := range testFiles {
		testDir := filepath.Dir(testFile)
		testBase := filepath.Base(testFile)

		// Same directory and matching name pattern
		if sourceDir == testDir && strings.Contains(testBase, sourceBase) {
			related = append(related, testFile)
		}
	}

	return related
}

// analyzeTestCoverage analyzes test files to determine behavioral coverage
func (bca *BehavioralCoverageAnalyzer) analyzeTestCoverage(testFiles []string, coverage *FileCoverage) {
	for _, testFile := range testFiles {
		bca.analyzeTestFile(testFile, coverage)
	}
}

// analyzeTestFile analyzes a single test file
func (bca *BehavioralCoverageAnalyzer) analyzeTestFile(testFile string, coverage *FileCoverage) {
	content, err := os.ReadFile(testFile)
	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")
	
	// Look for test function patterns
	testFuncPattern := regexp.MustCompile(`^func\s+(Test\w+|Benchmark\w+|Example\w+)`)
	
	for i, line := range lines {
		if testFuncPattern.MatchString(line) {
			testName := bca.extractTestName(line)
			if testName != "" {
				bca.markBehaviorsCovered(testName, lines[i:], coverage)
			}
		}
	}
}

// extractTestName extracts test function name from line
func (bca *BehavioralCoverageAnalyzer) extractTestName(line string) string {
	re := regexp.MustCompile(`^func\s+(Test\w+|Benchmark\w+|Example\w+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// markBehaviorsCovered marks behaviors as covered based on test analysis
func (bca *BehavioralCoverageAnalyzer) markBehaviorsCovered(testName string, testLines []string, coverage *FileCoverage) {
	// Simple heuristic: look for keywords in test that suggest covered behaviors
	testContent := strings.Join(testLines, " ")
	testContentLower := strings.ToLower(testContent)

	// Mark boundary tests as covered if test suggests boundary testing
	if strings.Contains(testContentLower, "boundary") || strings.Contains(testContentLower, "edge") || strings.Contains(testContentLower, "limit") {
		for i := range coverage.BoundaryTests {
			if !coverage.BoundaryTests[i].Coverage > 0 {
				coverage.BoundaryTests[i].TestedValues = append(coverage.BoundaryTests[i].TestedValues, "detected_in_"+testName)
			}
		}
	}

	// Mark error paths as covered if test suggests error testing
	if strings.Contains(testContentLower, "error") || strings.Contains(testContentLower, "fail") {
		for i := range coverage.ErrorPaths {
			if !coverage.ErrorPaths[i].Tested {
				coverage.ErrorPaths[i].Tested = true
				coverage.ErrorPaths[i].TestCase = testName
			}
		}
	}

	// Mark concurrency tests as covered
	if strings.Contains(testContentLower, "concurrent") || strings.Contains(testContentLower, "goroutine") || strings.Contains(testContentLower, "race") {
		for i := range coverage.ConcurrencyTests {
			if !coverage.ConcurrencyTests[i].Tested {
				coverage.ConcurrencyTests[i].Tested = true
				coverage.ConcurrencyTests[i].TestCase = testName
			}
		}
	}

	// Mark behaviors as covered based on test name patterns
	for behaviorName, behavior := range coverage.Behaviors {
		if bca.testCoversBehavior(testName, testContent, behaviorName, behavior) {
			behavior.Covered = true
			behavior.TestCases = append(behavior.TestCases, testName)
			coverage.Behaviors[behaviorName] = behavior
		}
	}
}

// testCoversBehavior determines if a test covers a specific behavior
func (bca *BehavioralCoverageAnalyzer) testCoversBehavior(testName, testContent, behaviorName string, behavior BehaviorCoverage) bool {
	testNameLower := strings.ToLower(testName)
	testContentLower := strings.ToLower(testContent)
	behaviorNameLower := strings.ToLower(behaviorName)

	// Heuristic matching based on keywords
	keywords := map[string][]string{
		"conditional": {"if", "condition", "branch", "true", "false"},
		"switch":      {"switch", "case", "default"},
		"loop":        {"loop", "iteration", "for", "range", "while"},
		"select":      {"select", "channel", "concurrency"},
		"error":       {"error", "fail", "exception"},
		"boundary":    {"boundary", "edge", "limit", "min", "max"},
	}

	for pattern, words := range keywords {
		if strings.Contains(behaviorNameLower, pattern) {
			for _, word := range words {
				if strings.Contains(testNameLower, word) || strings.Contains(testContentLower, word) {
					return true
				}
			}
		}
	}

	return false
}

// calculateCoverageMetrics calculates final coverage metrics
func (bca *BehavioralCoverageAnalyzer) calculateCoverageMetrics(coverage *FileCoverage) {
	// Calculate boundary test coverage
	for i := range coverage.BoundaryTests {
		boundary := &coverage.BoundaryTests[i]
		if len(boundary.Boundaries) > 0 {
			boundary.Coverage = float64(len(boundary.TestedValues)) / float64(len(boundary.Boundaries)) * 100
		}
		
		// Identify missing tests
		boundary.MissingTests = make([]string, 0)
		for _, boundaryValue := range boundary.Boundaries {
			tested := false
			for _, testedValue := range boundary.TestedValues {
				if boundaryValue == testedValue {
					tested = true
					break
				}
			}
			if !tested {
				boundary.MissingTests = append(boundary.MissingTests, boundaryValue)
			}
		}
	}

	// Calculate contract test coverage
	for i := range coverage.ContractTests {
		contract := &coverage.ContractTests[i]
		totalContracts := len(contract.Preconditions) + len(contract.Postconditions) + len(contract.Invariants)
		if totalContracts > 0 {
			contract.Coverage = float64(len(contract.TestedContracts)) / float64(totalContracts) * 100
		}
	}

	// Calculate overall behavioral coverage
	totalBehaviors := len(coverage.Behaviors)
	coveredBehaviors := 0
	for _, behavior := range coverage.Behaviors {
		if behavior.Covered {
			coveredBehaviors++
		}
	}

	if totalBehaviors > 0 {
		coverage.BranchCoverage = float64(coveredBehaviors) / float64(totalBehaviors) * 100
	}

	// Calculate path coverage (simplified)
	totalPaths := bca.estimatePathCount(coverage)
	coveredPaths := coveredBehaviors // Simplified assumption
	if totalPaths > 0 {
		coverage.PathCoverage = float64(coveredPaths) / float64(totalPaths) * 100
	}
}

// estimatePathCount estimates the number of execution paths
func (bca *BehavioralCoverageAnalyzer) estimatePathCount(coverage *FileCoverage) int {
	// Simplified path count estimation based on branches
	paths := 1
	
	for _, behavior := range coverage.Behaviors {
		if strings.Contains(behavior.Name, "conditional") {
			paths *= 2 // Each conditional doubles paths
		} else if strings.Contains(behavior.Name, "switch") {
			paths += behavior.Complexity // Add switch cases
		} else if strings.Contains(behavior.Name, "loop") {
			paths += 3 // Zero, one, many iterations
		}
	}

	return paths
}

// enhanceWithBehavioralPatterns adds cross-file behavioral analysis
func (bca *BehavioralCoverageAnalyzer) enhanceWithBehavioralPatterns() {
	// This would analyze patterns across files, such as:
	// - Interface implementation consistency
	// - State machine transitions across components
	// - Error propagation patterns
	// - Resource lifecycle management
	
	// For now, this is a placeholder for future enhancements
}

// Helper functions

func (bca *BehavioralCoverageAnalyzer) extractTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return bca.extractTypeString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + bca.extractTypeString(t.Elt)
	case *ast.MapType:
		return "map[" + bca.extractTypeString(t.Key) + "]" + bca.extractTypeString(t.Value)
	case *ast.ChanType:
		return "chan " + bca.extractTypeString(t.Value)
	case *ast.StarExpr:
		return "*" + bca.extractTypeString(t.X)
	default:
		return "unknown"
	}
}

func (bca *BehavioralCoverageAnalyzer) generateBoundaryValues(paramType string) []string {
	switch paramType {
	case "int", "int32", "int64":
		return []string{"0", "1", "-1", "math.MaxInt32", "math.MinInt32"}
	case "uint", "uint32", "uint64":
		return []string{"0", "1", "math.MaxUint32"}
	case "string":
		return []string{`""`, `"single"`, `"very long string with special chars @#$%"`}
	case "bool":
		return []string{"true", "false"}
	case "[]byte", "[]string":
		return []string{"nil", "empty slice", "single element", "large slice"}
	case "map":
		return []string{"nil", "empty map", "single entry", "large map"}
	case "chan":
		return []string{"nil", "unbuffered", "buffered", "closed"}
	default:
		return []string{"nil", "zero value", "valid value"}
	}
}

func (bca *BehavioralCoverageAnalyzer) isErrorCheck(ifStmt *ast.IfStmt) bool {
	// Simple heuristic: check if condition involves error comparison
	if binExpr, ok := ifStmt.Cond.(*ast.BinaryExpr); ok {
		if binExpr.Op == token.NEQ || binExpr.Op == token.EQL {
			// Check if one side is "nil" and other might be error
			return bca.mightBeErrorExpression(binExpr.X) || bca.mightBeErrorExpression(binExpr.Y)
		}
	}
	return false
}

func (bca *BehavioralCoverageAnalyzer) mightBeErrorExpression(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "err" || ident.Name == "error" || ident.Name == "nil"
	}
	return false
}

func (bca *BehavioralCoverageAnalyzer) mightReturnError(callExpr *ast.CallExpr) bool {
	// Heuristic: assume functions with certain names might return errors
	if fun, ok := callExpr.Fun.(*ast.Ident); ok {
		errorProneNames := []string{"Open", "Create", "Parse", "Connect", "Read", "Write", "Execute"}
		for _, name := range errorProneNames {
			if strings.Contains(fun.Name, name) {
				return true
			}
		}
	}
	return false
}

func (bca *BehavioralCoverageAnalyzer) addBehavioralExpectations(fn *ast.FuncDecl, coverage *FileCoverage, complexity *ComplexityMetrics) {
	funcName := fn.Name.Name
	
	// Add expectations based on function name patterns
	if strings.Contains(strings.ToLower(funcName), "validate") {
		coverage.Behaviors["validation_success"] = BehaviorCoverage{
			Name:        "validation_success",
			Description: "Function should validate correct inputs",
			Covered:     false,
			Priority:    PriorityHigh,
			Complexity:  complexity.CyclomaticComplexity,
		}
		coverage.Behaviors["validation_failure"] = BehaviorCoverage{
			Name:        "validation_failure",
			Description: "Function should reject invalid inputs",
			Covered:     false,
			Priority:    PriorityHigh,
			Complexity:  complexity.CyclomaticComplexity,
		}
	}

	if strings.Contains(strings.ToLower(funcName), "process") {
		coverage.Behaviors["process_success"] = BehaviorCoverage{
			Name:        "process_success",
			Description: "Function should process valid inputs successfully",
			Covered:     false,
			Priority:    PriorityHigh,
			Complexity:  complexity.CyclomaticComplexity,
		}
		coverage.Behaviors["process_error"] = BehaviorCoverage{
			Name:        "process_error",
			Description: "Function should handle processing errors gracefully",
			Covered:     false,
			Priority:    PriorityMedium,
			Complexity:  complexity.CyclomaticComplexity,
		}
	}

	// Add expectations based on complexity
	if complexity.CyclomaticComplexity > 10 {
		coverage.Behaviors["high_complexity_paths"] = BehaviorCoverage{
			Name:        "high_complexity_paths",
			Description: "All execution paths in high-complexity function should be tested",
			Covered:     false,
			Priority:    PriorityHigh,
			Complexity:  complexity.CyclomaticComplexity,
		}
	}
}