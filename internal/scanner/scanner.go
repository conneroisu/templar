// Package scanner provides component discovery and analysis for templ templates.
//
// The scanner traverses file systems to find .templ files, parses them using
// Go's AST parser to extract component metadata including parameters, dependencies,
// and documentation. It integrates with the component registry to broadcast
// change events and supports recursive directory scanning with exclude patterns.
// The scanner maintains file hashes for change detection and provides both
// single-file and batch scanning capabilities.
package scanner

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/types"
)

// ScanJob represents a scanning job for the worker pool containing the file
// path to scan and a result channel for asynchronous communication.
type ScanJob struct {
	// filePath is the absolute path to the .templ file to be scanned
	filePath string
	// result channel receives the scan result or error asynchronously
	result chan<- ScanResult
}

// ScanResult represents the result of a scanning operation, containing either
// success status or error information for a specific file.
type ScanResult struct {
	// filePath is the path that was scanned
	filePath string
	// err contains any error that occurred during scanning, nil on success
	err error
}

// WorkerPool manages persistent scanning workers for performance optimization
// using a work-stealing approach to distribute scanning jobs across CPU cores.
type WorkerPool struct {
	// jobQueue buffers scanning jobs for worker distribution
	jobQueue chan ScanJob
	// workers holds references to all active worker goroutines
	workers []*ScanWorker
	// workerCount defines the number of concurrent workers (typically NumCPU)
	workerCount int
	// scanner is the shared component scanner instance
	scanner *ComponentScanner
	// stop signals all workers to terminate gracefully
	stop chan struct{}
	// stopped tracks pool shutdown state
	stopped bool
	// mu protects concurrent access to pool state
	mu sync.RWMutex
}

// ScanWorker represents a persistent worker goroutine that processes scanning
// jobs from the shared job queue. Each worker operates independently and
// can handle multiple file types concurrently.
type ScanWorker struct {
	// id uniquely identifies this worker for debugging and metrics
	id int
	// jobQueue receives scanning jobs from the worker pool
	jobQueue <-chan ScanJob
	// scanner provides the component parsing functionality
	scanner *ComponentScanner
	// stop signals this worker to terminate gracefully
	stop chan struct{}
}

// ComponentScanner discovers and parses templ components using Go's AST parser.
//
// The scanner provides:
// - Recursive directory traversal with exclude patterns
// - AST-based component metadata extraction
// - Concurrent processing via worker pool
// - Integration with component registry for event broadcasting
// - File change detection using CRC32 hashing
type ComponentScanner struct {
	// registry receives discovered components and broadcasts change events
	registry *registry.ComponentRegistry
	// fileSet tracks file positions for AST parsing and error reporting
	fileSet *token.FileSet
	// workerPool manages concurrent scanning operations
	workerPool *WorkerPool
}

// NewComponentScanner creates a new component scanner with optimized worker pool
func NewComponentScanner(registry *registry.ComponentRegistry) *ComponentScanner {
	scanner := &ComponentScanner{
		registry: registry,
		fileSet:  token.NewFileSet(),
	}

	// Initialize worker pool with optimal worker count
	workerCount := runtime.NumCPU()
	if workerCount > 8 {
		workerCount = 8 // Cap at 8 workers for diminishing returns
	}

	scanner.workerPool = NewWorkerPool(workerCount, scanner)
	return scanner
}

// NewWorkerPool creates a new worker pool for scanning operations
func NewWorkerPool(workerCount int, scanner *ComponentScanner) *WorkerPool {
	pool := &WorkerPool{
		jobQueue:    make(chan ScanJob, workerCount*2), // Buffer for work-stealing efficiency
		workerCount: workerCount,
		scanner:     scanner,
		stop:        make(chan struct{}),
	}

	// Start persistent workers
	pool.workers = make([]*ScanWorker, workerCount)
	for i := 0; i < workerCount; i++ {
		worker := &ScanWorker{
			id:       i,
			jobQueue: pool.jobQueue,
			scanner:  scanner,
			stop:     make(chan struct{}),
		}
		pool.workers[i] = worker
		go worker.start()
	}

	return pool
}

// start begins the worker's processing loop
func (w *ScanWorker) start() {
	for {
		select {
		case job := <-w.jobQueue:
			// Process the scanning job
			err := w.scanner.scanFileInternal(job.filePath)
			job.result <- ScanResult{
				filePath: job.filePath,
				err:      err,
			}
		case <-w.stop:
			return
		}
	}
}

// Stop gracefully shuts down the worker pool
func (p *WorkerPool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		return
	}

	p.stopped = true
	close(p.stop)

	// Stop all workers
	for _, worker := range p.workers {
		close(worker.stop)
	}

	// Close job queue
	close(p.jobQueue)
}

// GetRegistry returns the component registry
func (s *ComponentScanner) GetRegistry() *registry.ComponentRegistry {
	return s.registry
}

// Close gracefully shuts down the scanner and its worker pool
func (s *ComponentScanner) Close() error {
	if s.workerPool != nil {
		s.workerPool.Stop()
	}
	return nil
}

// ScanDirectory scans a directory for templ components using optimized worker pool
func (s *ComponentScanner) ScanDirectory(dir string) error {
	// Validate directory path to prevent path traversal
	if _, err := s.validatePath(dir); err != nil {
		return fmt.Errorf("invalid directory path: %w", err)
	}

	// First, collect all .templ files efficiently
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".templ") {
			return nil
		}

		// Validate each file path as we encounter it
		if _, err := s.validatePath(path); err != nil {
			// Skip invalid paths silently for security
			return nil
		}

		files = append(files, path)
		return nil
	})

	if err != nil {
		return err
	}

	// Process files using persistent worker pool (no goroutine creation overhead)
	return s.processBatchWithWorkerPool(files)
}

// processBatchWithWorkerPool processes files using the persistent worker pool
func (s *ComponentScanner) processBatchWithWorkerPool(files []string) error {
	if len(files) == 0 {
		return nil
	}

	// Create result channel for collecting results
	resultChan := make(chan ScanResult, len(files))

	// Submit jobs to persistent worker pool
	for _, file := range files {
		job := ScanJob{
			filePath: file,
			result:   resultChan,
		}

		select {
		case s.workerPool.jobQueue <- job:
			// Job submitted successfully
		default:
			// Worker pool is full, process synchronously as fallback
			err := s.scanFileInternal(file)
			resultChan <- ScanResult{filePath: file, err: err}
		}
	}

	// Collect results
	var errors []error
	for i := 0; i < len(files); i++ {
		result := <-resultChan
		if result.err != nil {
			errors = append(errors, fmt.Errorf("scanning %s: %w", result.filePath, result.err))
		}
	}

	close(resultChan)

	if len(errors) > 0 {
		return fmt.Errorf("scan completed with %d errors: %v", len(errors), errors[0])
	}

	return nil
}

// ScanDirectoryParallel is deprecated in favor of the optimized ScanDirectory
// Kept for backward compatibility
func (s *ComponentScanner) ScanDirectoryParallel(dir string, workers int) error {
	return s.ScanDirectory(dir) // Use optimized version
}

// ScanFile scans a single file for templ components (optimized)
func (s *ComponentScanner) ScanFile(path string) error {
	return s.scanFileInternal(path)
}

// scanFileInternal is the optimized internal scanning method used by workers
func (s *ComponentScanner) scanFileInternal(path string) error {
	// Validate and clean the path to prevent directory traversal
	cleanPath, err := s.validatePath(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Optimized single I/O operation: open file and get both content and info
	file, err := os.Open(cleanPath)
	if err != nil {
		return fmt.Errorf("opening file %s: %w", cleanPath, err)
	}
	defer file.Close()

	// Get file info without separate Stat() call
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("getting file info for %s: %w", cleanPath, err)
	}

	// Read content efficiently based on file size
	var content []byte
	if info.Size() > 64*1024 {
		// Use streaming read for large files to reduce memory pressure
		content, err = s.readFileStreaming(file, info.Size())
	} else {
		// Regular read for small files
		content = make([]byte, info.Size())
		_, err = file.Read(content)
	}

	if err != nil {
		return fmt.Errorf("reading file %s: %w", cleanPath, err)
	}

	// Calculate hash using CRC32 (faster for file change detection)
	hash := fmt.Sprintf("%x", crc32.ChecksumIEEE(content))

	// Parse the file as Go code (templ generates Go)
	astFile, err := parser.ParseFile(s.fileSet, cleanPath, content, parser.ParseComments)
	if err != nil {
		// If it's a .templ file that can't be parsed as Go, try to extract components manually
		return s.parseTemplFile(cleanPath, content, hash, info.ModTime())
	}

	// Extract components from AST
	return s.extractFromAST(cleanPath, astFile, hash, info.ModTime())
}

// readFileStreaming reads large files in chunks to reduce memory pressure
func (s *ComponentScanner) readFileStreaming(file *os.File, size int64) ([]byte, error) {
	const chunkSize = 32 * 1024 // 32KB chunks
	content := make([]byte, 0, size)
	chunk := make([]byte, chunkSize)

	for {
		n, err := file.Read(chunk)
		if n > 0 {
			content = append(content, chunk[:n]...)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if n < chunkSize {
			break
		}
	}

	return content, nil
}

// Backward compatibility method
func (s *ComponentScanner) scanFile(path string) error {
	return s.scanFileInternal(path)
}

func (s *ComponentScanner) parseTemplFile(path string, content []byte, hash string, modTime time.Time) error {
	lines := strings.Split(string(content), "\n")
	packageName := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Extract package name
		if strings.HasPrefix(line, "package ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				packageName = sanitizeIdentifier(parts[1])
			}
		}

		// Extract templ component declarations
		if strings.HasPrefix(line, "templ ") {
			// Extract component name from templ declaration
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[1]
				if idx := strings.Index(name, "("); idx != -1 {
					name = name[:idx]
				}

				// Sanitize component name to prevent injection
				name = sanitizeIdentifier(name)

				component := &types.ComponentInfo{
					Name:         name,
					Package:      packageName,
					FilePath:     path,
					Parameters:   extractParameters(line),
					Imports:      []string{},
					LastMod:      modTime,
					Hash:         hash,
					Dependencies: []string{},
				}

				s.registry.Register(component)
			}
		}
	}

	return nil
}

func (s *ComponentScanner) extractFromAST(path string, astFile *ast.File, hash string, modTime time.Time) error {
	// Walk the AST to find function declarations that might be templ components
	ast.Inspect(astFile, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if node.Name != nil && node.Name.IsExported() {
				// Check if this might be a templ component
				if s.isTemplComponent(node) {
					component := &types.ComponentInfo{
						Name:         node.Name.Name,
						Package:      astFile.Name.Name,
						FilePath:     path,
						Parameters:   s.extractParametersFromFunc(node),
						Imports:      s.extractImports(astFile),
						LastMod:      modTime,
						Hash:         hash,
						Dependencies: []string{},
					}

					s.registry.Register(component)
				}
			}
		}
		return true
	})

	return nil
}

func (s *ComponentScanner) isTemplComponent(fn *ast.FuncDecl) bool {
	// Check if the function returns a templ.Component
	if fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return false
	}

	result := fn.Type.Results.List[0]
	if sel, ok := result.Type.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			return ident.Name == "templ" && sel.Sel.Name == "Component"
		}
	}

	return false
}

func (s *ComponentScanner) extractParametersFromFunc(fn *ast.FuncDecl) []types.ParameterInfo {
	var params []types.ParameterInfo

	if fn.Type.Params == nil {
		return params
	}

	for _, param := range fn.Type.Params.List {
		paramType := ""
		if param.Type != nil {
			paramType = s.typeToString(param.Type)
		}

		for _, name := range param.Names {
			params = append(params, types.ParameterInfo{
				Name:     name.Name,
				Type:     paramType,
				Optional: false,
				Default:  nil,
			})
		}
	}

	return params
}

func (s *ComponentScanner) extractImports(astFile *ast.File) []string {
	var imports []string

	for _, imp := range astFile.Imports {
		if imp.Path != nil {
			imports = append(imports, imp.Path.Value)
		}
	}

	return imports
}

func (s *ComponentScanner) typeToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return s.typeToString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + s.typeToString(e.X)
	case *ast.ArrayType:
		return "[]" + s.typeToString(e.Elt)
	default:
		return "unknown"
	}
}

func extractParameters(line string) []types.ParameterInfo {
	// Simple parameter extraction from templ declaration
	// This is a basic implementation - real parser would be more robust
	if !strings.Contains(line, "(") {
		return []types.ParameterInfo{}
	}

	start := strings.Index(line, "(")
	end := strings.LastIndex(line, ")")
	if start == -1 || end == -1 || start >= end {
		return []types.ParameterInfo{}
	}

	paramStr := line[start+1 : end]
	if strings.TrimSpace(paramStr) == "" {
		return []types.ParameterInfo{}
	}

	// Basic parameter parsing - handle both "name type" and "name, name type" patterns
	parts := strings.Split(paramStr, ",")
	var params []types.ParameterInfo

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split by space to get name and type
		fields := strings.Fields(part)
		if len(fields) >= 2 {
			// Handle "name type" format
			params = append(params, types.ParameterInfo{
				Name:     fields[0],
				Type:     fields[1],
				Optional: false,
				Default:  nil,
			})
		} else if len(fields) == 1 {
			// Handle single parameter name (type might be from previous param)
			params = append(params, types.ParameterInfo{
				Name:     fields[0],
				Type:     "string", // Default type
				Optional: false,
				Default:  nil,
			})
		}
	}

	return params
}

// sanitizeIdentifier removes dangerous characters from identifiers
func sanitizeIdentifier(identifier string) string {
	// Only allow alphanumeric characters and underscores for identifiers
	var cleaned strings.Builder
	for _, r := range identifier {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			cleaned.WriteRune(r)
		}
	}
	return cleaned.String()
}

// validatePath validates and cleans a file path to prevent directory traversal
func (s *ComponentScanner) validatePath(path string) (string, error) {
	// Clean the path to resolve . and .. elements
	cleanPath := filepath.Clean(path)

	// Get absolute path to normalize
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("getting absolute path: %w", err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting current directory: %w", err)
	}

	// Ensure the path is within the current working directory or its subdirectories
	// This prevents directory traversal attacks
	if !strings.HasPrefix(absPath, cwd) {
		return "", fmt.Errorf("path %s is outside current working directory", path)
	}

	// Additional security check: reject paths with suspicious patterns
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("path contains directory traversal: %s", path)
	}

	return cleanPath, nil
}
