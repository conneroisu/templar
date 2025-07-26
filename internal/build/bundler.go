package build

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/config"
)

// AssetBundler handles bundling and optimization of JavaScript, CSS, and other assets.
type AssetBundler struct {
	config           *config.Config
	outputDir        string
	bundleMap        map[string]string
	chunkMap         map[string][]string
	sourceMapEnabled bool
}

// AssetManifest represents discovered assets and their dependencies.
type AssetManifest struct {
	JavaScript   []AssetFile         `json:"javascript"`
	CSS          []AssetFile         `json:"css"`
	Images       []AssetFile         `json:"images"`
	Fonts        []AssetFile         `json:"fonts"`
	Other        []AssetFile         `json:"other"`
	Dependencies map[string][]string `json:"dependencies"`
}

// AssetFile represents an individual asset file.
type AssetFile struct {
	Path         string    `json:"path"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Size         int64     `json:"size"`
	Hash         string    `json:"hash"`
	ModTime      time.Time `json:"mod_time"`
	Dependencies []string  `json:"dependencies,omitempty"`
	IsEntry      bool      `json:"is_entry"`
}

// BundlerOptions configures the bundling process.
type BundlerOptions struct {
	Minify        bool              `json:"minify"`
	TreeShaking   bool              `json:"tree_shaking"`
	CodeSplitting bool              `json:"code_splitting"`
	SourceMaps    bool              `json:"source_maps"`
	Environment   string            `json:"environment"`
	Target        string            `json:"target"`   // "es2020", "es2015", etc.
	Format        string            `json:"format"`   // "esm", "cjs", "iife"
	External      []string          `json:"external"` // External dependencies
	Define        map[string]string `json:"define"`   // Define constants
	Plugins       []string          `json:"plugins"`  // Custom plugins
}

// BundleResult represents the result of a bundling operation.
type BundleResult struct {
	Bundles    map[string]BundleInfo `json:"bundles"`
	Chunks     map[string]ChunkInfo  `json:"chunks"`
	SourceMaps map[string]string     `json:"source_maps,omitempty"`
	Statistics BundleStatistics      `json:"statistics"`
	Errors     []string              `json:"errors,omitempty"`
	Warnings   []string              `json:"warnings,omitempty"`
}

// BundleInfo contains information about a generated bundle.
type BundleInfo struct {
	Name      string   `json:"name"`
	Path      string   `json:"path"`
	Size      int64    `json:"size"`
	GzipSize  int64    `json:"gzip_size,omitempty"`
	Hash      string   `json:"hash"`
	Imports   []string `json:"imports"`
	Exports   []string `json:"exports,omitempty"`
	IsEntry   bool     `json:"is_entry"`
	SourceMap string   `json:"source_map,omitempty"`
}

// ChunkInfo contains information about code-split chunks.
type ChunkInfo struct {
	Name          string   `json:"name"`
	Path          string   `json:"path"`
	Size          int64    `json:"size"`
	Hash          string   `json:"hash"`
	Modules       []string `json:"modules"`
	DynamicImport bool     `json:"dynamic_import"`
}

// BundleStatistics provides metrics about the bundling process.
type BundleStatistics struct {
	TotalSize       int64         `json:"total_size"`
	CompressedSize  int64         `json:"compressed_size"`
	BundleCount     int           `json:"bundle_count"`
	ChunkCount      int           `json:"chunk_count"`
	ModuleCount     int           `json:"module_count"`
	DuplicationRate float64       `json:"duplication_rate"`
	BuildTime       time.Duration `json:"build_time"`
	TreeShakeRatio  float64       `json:"tree_shake_ratio,omitempty"`
}

// AssetManifestFile represents the final asset manifest for deployment.
type AssetManifestFile struct {
	Version       string            `json:"version"`
	BuildTime     time.Time         `json:"build_time"`
	StaticFiles   []string          `json:"static_files"`
	BundledAssets []string          `json:"bundled_assets"`
	SourceMaps    []string          `json:"source_maps,omitempty"`
	AssetMap      map[string]string `json:"asset_map"`
	Integrity     map[string]string `json:"integrity"`
}

// NewAssetBundler creates a new asset bundler instance.
func NewAssetBundler(cfg *config.Config, outputDir string) *AssetBundler {
	return &AssetBundler{
		config:           cfg,
		outputDir:        outputDir,
		bundleMap:        make(map[string]string),
		chunkMap:         make(map[string][]string),
		sourceMapEnabled: true,
	}
}

// DiscoverAssets scans the project for assets that need bundling.
func (b *AssetBundler) DiscoverAssets(ctx context.Context) (*AssetManifest, error) {
	manifest := &AssetManifest{
		JavaScript:   make([]AssetFile, 0),
		CSS:          make([]AssetFile, 0),
		Images:       make([]AssetFile, 0),
		Fonts:        make([]AssetFile, 0),
		Other:        make([]AssetFile, 0),
		Dependencies: make(map[string][]string),
	}

	// Asset discovery paths
	discoveryPaths := []string{
		"static",
		"assets",
		"public",
		"src",
		"components",
	}

	for _, basePath := range discoveryPaths {
		if _, err := os.Stat(basePath); os.IsNotExist(err) {
			continue // Skip if directory doesn't exist
		}

		err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			// Skip hidden files and common ignore patterns
			if strings.HasPrefix(info.Name(), ".") ||
				strings.Contains(path, "node_modules") ||
				strings.Contains(path, ".git") {
				return nil
			}

			assetFile, err := b.createAssetFile(path, info)
			if err != nil {
				return fmt.Errorf("failed to process asset %s: %w", path, err)
			}

			if assetFile == nil {
				return nil // Not a recognized asset type
			}

			// Categorize asset by type
			switch assetFile.Type {
			case "javascript":
				manifest.JavaScript = append(manifest.JavaScript, *assetFile)
			case "css":
				manifest.CSS = append(manifest.CSS, *assetFile)
			case "image":
				manifest.Images = append(manifest.Images, *assetFile)
			case "font":
				manifest.Fonts = append(manifest.Fonts, *assetFile)
			default:
				manifest.Other = append(manifest.Other, *assetFile)
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to walk directory %s: %w", basePath, err)
		}
	}

	// Analyze dependencies
	if err := b.analyzeDependencies(manifest); err != nil {
		return nil, fmt.Errorf("failed to analyze dependencies: %w", err)
	}

	return manifest, nil
}

// Bundle performs the actual bundling of assets.
func (b *AssetBundler) Bundle(
	ctx context.Context,
	manifest *AssetManifest,
	options BundlerOptions,
) ([]string, error) {
	startTime := time.Now()
	bundledFiles := make([]string, 0)

	// Bundle JavaScript files
	if len(manifest.JavaScript) > 0 {
		jsBundle, err := b.bundleJavaScript(ctx, manifest.JavaScript, options)
		if err != nil {
			return nil, fmt.Errorf("JavaScript bundling failed: %w", err)
		}
		bundledFiles = append(bundledFiles, jsBundle...)
	}

	// Bundle CSS files
	if len(manifest.CSS) > 0 {
		cssBundle, err := b.bundleCSS(ctx, manifest.CSS, options)
		if err != nil {
			return nil, fmt.Errorf("CSS bundling failed: %w", err)
		}
		bundledFiles = append(bundledFiles, cssBundle...)
	}

	// Process other assets (copy with fingerprinting)
	otherAssets, err := b.processOtherAssets(ctx, manifest, options)
	if err != nil {
		return nil, fmt.Errorf("other asset processing failed: %w", err)
	}
	bundledFiles = append(bundledFiles, otherAssets...)

	fmt.Printf("Asset bundling completed in %v\n", time.Since(startTime))

	return bundledFiles, nil
}

// bundleJavaScript handles JavaScript bundling with esbuild-like functionality.
func (b *AssetBundler) bundleJavaScript(
	ctx context.Context,
	jsFiles []AssetFile,
	options BundlerOptions,
) ([]string, error) {
	if len(jsFiles) == 0 {
		return nil, nil
	}

	// For this implementation, we'll create a simple bundler
	// In a real implementation, you'd integrate with esbuild, webpack, or rollup

	bundledFiles := make([]string, 0)
	entryPoints := b.findEntryPoints(jsFiles)

	for _, entry := range entryPoints {
		bundleName := b.generateBundleName(entry.Name, "js")
		bundlePath := filepath.Join(b.outputDir, "js", bundleName)

		// Simple concatenation bundler (replace with real bundler)
		content, err := b.simpleJSBundle(entry, jsFiles, options)
		if err != nil {
			return nil, fmt.Errorf("failed to bundle %s: %w", entry.Name, err)
		}

		// Write bundle
		if err := os.WriteFile(bundlePath, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write bundle %s: %w", bundlePath, err)
		}

		bundledFiles = append(bundledFiles, bundlePath)

		// Generate source map if enabled
		if options.SourceMaps {
			sourceMapPath := bundlePath + ".map"
			sourceMap := b.generateSimpleSourceMap(entry, bundleName)
			if err := os.WriteFile(sourceMapPath, []byte(sourceMap), 0644); err != nil {
				return nil, fmt.Errorf("failed to write source map %s: %w", sourceMapPath, err)
			}
			bundledFiles = append(bundledFiles, sourceMapPath)
		}
	}

	return bundledFiles, nil
}

// bundleCSS handles CSS bundling and optimization.
func (b *AssetBundler) bundleCSS(
	ctx context.Context,
	cssFiles []AssetFile,
	options BundlerOptions,
) ([]string, error) {
	if len(cssFiles) == 0 {
		return nil, nil
	}

	bundledFiles := make([]string, 0)

	// Combine all CSS files into a single bundle
	bundleName := b.generateBundleName("main", "css")
	bundlePath := filepath.Join(b.outputDir, "css", bundleName)

	var cssContent strings.Builder

	for _, cssFile := range cssFiles {
		content, err := os.ReadFile(cssFile.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to read CSS file %s: %w", cssFile.Path, err)
		}

		// Add file header comment
		cssContent.WriteString(fmt.Sprintf("/* Source: %s */\n", cssFile.Path))
		cssContent.Write(content)
		cssContent.WriteString("\n\n")
	}

	finalCSS := cssContent.String()

	// Apply CSS optimization if requested
	if options.Minify {
		finalCSS = b.minifyCSS(finalCSS)
	}

	// Write bundle
	if err := os.WriteFile(bundlePath, []byte(finalCSS), 0644); err != nil {
		return nil, fmt.Errorf("failed to write CSS bundle %s: %w", bundlePath, err)
	}

	bundledFiles = append(bundledFiles, bundlePath)

	return bundledFiles, nil
}

// processOtherAssets handles images, fonts, and other static assets.
func (b *AssetBundler) processOtherAssets(
	ctx context.Context,
	manifest *AssetManifest,
	options BundlerOptions,
) ([]string, error) {
	processedFiles := make([]string, 0)

	// Process images
	for _, imageFile := range manifest.Images {
		processedPath, err := b.copyWithFingerprint(imageFile, "images")
		if err != nil {
			return nil, fmt.Errorf("failed to process image %s: %w", imageFile.Path, err)
		}
		processedFiles = append(processedFiles, processedPath)
	}

	// Process fonts
	for _, fontFile := range manifest.Fonts {
		processedPath, err := b.copyWithFingerprint(fontFile, "fonts")
		if err != nil {
			return nil, fmt.Errorf("failed to process font %s: %w", fontFile.Path, err)
		}
		processedFiles = append(processedFiles, processedPath)
	}

	// Process other assets
	for _, otherFile := range manifest.Other {
		processedPath, err := b.copyWithFingerprint(otherFile, "other")
		if err != nil {
			return nil, fmt.Errorf("failed to process asset %s: %w", otherFile.Path, err)
		}
		processedFiles = append(processedFiles, processedPath)
	}

	return processedFiles, nil
}

// Helper methods

// createAssetFile creates an AssetFile from a file path and info.
func (b *AssetBundler) createAssetFile(path string, info os.FileInfo) (*AssetFile, error) {
	ext := strings.ToLower(filepath.Ext(path))

	assetType := b.getAssetType(ext)
	if assetType == "" {
		return nil, nil // Not a recognized asset type
	}

	// Calculate file hash
	hash, err := b.calculateFileHash(path)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate hash: %w", err)
	}

	return &AssetFile{
		Path:    path,
		Name:    info.Name(),
		Type:    assetType,
		Size:    info.Size(),
		Hash:    hash,
		ModTime: info.ModTime(),
		IsEntry: b.isEntryPoint(path, assetType),
	}, nil
}

// getAssetType determines the asset type from file extension.
func (b *AssetBundler) getAssetType(ext string) string {
	switch ext {
	case ".js", ".mjs", ".ts", ".jsx", ".tsx":
		return "javascript"
	case ".css", ".scss", ".sass", ".less":
		return "css"
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".avif":
		return "image"
	case ".woff", ".woff2", ".ttf", ".otf", ".eot":
		return "font"
	case ".html", ".htm":
		return "html"
	case ".json", ".xml", ".txt", ".md":
		return "data"
	default:
		return ""
	}
}

// isEntryPoint determines if a file should be treated as an entry point.
func (b *AssetBundler) isEntryPoint(path, assetType string) bool {
	if assetType != "javascript" && assetType != "css" {
		return false
	}

	// Common entry point patterns
	entryPatterns := []string{
		"main.",
		"index.",
		"app.",
		"entry.",
	}

	fileName := filepath.Base(path)
	for _, pattern := range entryPatterns {
		if strings.HasPrefix(fileName, pattern) {
			return true
		}
	}

	return false
}

// calculateFileHash generates a SHA256 hash for a file.
func (b *AssetBundler) calculateFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil))[:12], nil // Use first 12 chars
}

// generateBundleName creates a fingerprinted bundle name.
func (b *AssetBundler) generateBundleName(baseName, ext string) string {
	timestamp := time.Now().Unix()

	return fmt.Sprintf("%s-%d.%s", baseName, timestamp, ext)
}

// findEntryPoints identifies entry point files for bundling.
func (b *AssetBundler) findEntryPoints(files []AssetFile) []AssetFile {
	entryPoints := make([]AssetFile, 0)

	for _, file := range files {
		if file.IsEntry {
			entryPoints = append(entryPoints, file)
		}
	}

	// If no explicit entry points, use first file as entry
	if len(entryPoints) == 0 && len(files) > 0 {
		entryPoints = append(entryPoints, files[0])
	}

	return entryPoints
}

// simpleJSBundle creates a simple JavaScript bundle (replace with real bundler).
func (b *AssetBundler) simpleJSBundle(
	entry AssetFile,
	allFiles []AssetFile,
	options BundlerOptions,
) (string, error) {
	var bundle strings.Builder

	// Add bundle header
	bundle.WriteString("/* Generated by Templar Build System */\n")
	bundle.WriteString("(function() {\n")
	bundle.WriteString("'use strict';\n\n")

	// Read and include the entry file
	content, err := os.ReadFile(entry.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read entry file: %w", err)
	}

	bundle.WriteString(fmt.Sprintf("/* Entry: %s */\n", entry.Path))
	bundle.Write(content)
	bundle.WriteString("\n\n")

	// Add bundle footer
	bundle.WriteString("})();\n")

	bundleContent := bundle.String()

	// Apply minification if requested
	if options.Minify {
		bundleContent = b.minifyJS(bundleContent)
	}

	return bundleContent, nil
}

// minifyJS applies basic JavaScript minification.
func (b *AssetBundler) minifyJS(content string) string {
	// Basic minification - remove comments and extra whitespace
	// In production, use a real minifier like esbuild or terser

	lines := strings.Split(content, "\n")
	var minified strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Remove block comments (basic implementation)
		if strings.Contains(trimmed, "/*") && strings.Contains(trimmed, "*/") {
			// Skip lines with block comments
			continue
		}

		minified.WriteString(trimmed)
		minified.WriteString(" ")
	}

	return strings.TrimSpace(minified.String())
}

// minifyCSS applies basic CSS minification.
func (b *AssetBundler) minifyCSS(content string) string {
	// Basic CSS minification
	// Remove comments
	content = strings.ReplaceAll(content, "/*", "")
	content = strings.ReplaceAll(content, "*/", "")

	// Remove extra whitespace
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "\t", " ")

	// Remove multiple spaces
	for strings.Contains(content, "  ") {
		content = strings.ReplaceAll(content, "  ", " ")
	}

	return strings.TrimSpace(content)
}

// copyWithFingerprint copies a file with a fingerprinted name.
func (b *AssetBundler) copyWithFingerprint(asset AssetFile, subdir string) (string, error) {
	ext := filepath.Ext(asset.Name)
	name := strings.TrimSuffix(asset.Name, ext)
	fingerprintedName := fmt.Sprintf("%s-%s%s", name, asset.Hash, ext)

	outputPath := filepath.Join(b.outputDir, subdir, fingerprintedName)

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Copy file
	source, err := os.Open(asset.Path)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	dest, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dest.Close()

	if _, err := io.Copy(dest, source); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	return outputPath, nil
}

// generateSimpleSourceMap creates a basic source map.
func (b *AssetBundler) generateSimpleSourceMap(entry AssetFile, bundleName string) string {
	sourceMap := map[string]interface{}{
		"version":  3,
		"file":     bundleName,
		"sources":  []string{entry.Path},
		"names":    []string{},
		"mappings": "", // Would need proper source map generation
	}

	data, _ := json.Marshal(sourceMap)

	return string(data)
}

// analyzeDependencies analyzes asset dependencies (simplified implementation).
func (b *AssetBundler) analyzeDependencies(manifest *AssetManifest) error {
	// For each JavaScript file, analyze imports/requires
	for _, jsFile := range manifest.JavaScript {
		deps, err := b.analyzeJSDependencies(jsFile.Path)
		if err != nil {
			return fmt.Errorf("failed to analyze dependencies for %s: %w", jsFile.Path, err)
		}
		manifest.Dependencies[jsFile.Path] = deps
	}

	// For CSS files, analyze @import statements
	for _, cssFile := range manifest.CSS {
		deps, err := b.analyzeCSSImports(cssFile.Path)
		if err != nil {
			return fmt.Errorf("failed to analyze CSS imports for %s: %w", cssFile.Path, err)
		}
		manifest.Dependencies[cssFile.Path] = deps
	}

	return nil
}

// analyzeJSDependencies finds import/require statements (basic implementation).
func (b *AssetBundler) analyzeJSDependencies(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	deps := make([]string, 0)
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for ES6 imports
		if strings.HasPrefix(line, "import ") && strings.Contains(line, "from ") {
			start := strings.Index(line, "from ") + 5
			if start < len(line) {
				dep := strings.Trim(line[start:], " '\";")
				if dep != "" {
					deps = append(deps, dep)
				}
			}
		}

		// Look for CommonJS requires
		if strings.Contains(line, "require(") {
			start := strings.Index(line, "require(") + 8
			end := strings.Index(line[start:], ")")
			if end > 0 {
				dep := strings.Trim(line[start:start+end], " '\"")
				if dep != "" {
					deps = append(deps, dep)
				}
			}
		}
	}

	return deps, nil
}

// analyzeCSSImports finds @import statements in CSS files.
func (b *AssetBundler) analyzeCSSImports(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	deps := make([]string, 0)
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "@import ") {
			// Extract the imported file
			start := strings.Index(line, "\"")
			if start == -1 {
				start = strings.Index(line, "'")
			}
			if start != -1 {
				start++
				end := strings.Index(line[start:], line[start-1:start])
				if end > 0 {
					dep := line[start : start+end]
					if dep != "" {
						deps = append(deps, dep)
					}
				}
			}
		}
	}

	return deps, nil
}

// AnalyzeBundles returns bundle analysis for reporting.
func (b *AssetBundler) AnalyzeBundles() BundleStatistics {
	// Return basic statistics
	// In a real implementation, this would be populated during bundling
	return BundleStatistics{
		TotalSize:      0,
		CompressedSize: 0,
		BundleCount:    len(b.bundleMap),
		ChunkCount:     0,
		ModuleCount:    0,
		BuildTime:      0,
	}
}

// writeJSONFile writes data as JSON to a file.
func writeJSONFile(path string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
