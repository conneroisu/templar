package build

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test fixtures and helper functions

func createTempDir(t *testing.T) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "templar-bundler-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	return tmpDir
}

func createTestFile(t *testing.T, path string, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file %s: %v", path, err)
	}
}

func createTestConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{} // Basic config for testing
}

// TestNewAssetBundler validates bundler initialization
func TestNewAssetBundler(t *testing.T) {
	tmpDir := createTempDir(t)
	cfg := createTestConfig(t)
	outputDir := filepath.Join(tmpDir, "output")
	
	bundler := NewAssetBundler(cfg, outputDir)
	
	require.NotNil(t, bundler)
	assert.Equal(t, cfg, bundler.config)
	assert.Equal(t, outputDir, bundler.outputDir)
	assert.NotNil(t, bundler.bundleMap)
	assert.NotNil(t, bundler.chunkMap)
	assert.True(t, bundler.sourceMapEnabled)
	assert.Equal(t, 0, len(bundler.bundleMap))
	assert.Equal(t, 0, len(bundler.chunkMap))
}

// TestDiscoverAssets_EmptyDirectory tests asset discovery in empty directories
func TestDiscoverAssets_EmptyDirectory(t *testing.T) {
	tmpDir := createTempDir(t)
	cfg := createTestConfig(t)
	bundler := NewAssetBundler(cfg, tmpDir)
	
	// Change to temp directory for relative path discovery
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)
	
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	
	ctx := context.Background()
	manifest, err := bundler.DiscoverAssets(ctx)
	
	require.NoError(t, err)
	require.NotNil(t, manifest)
	assert.Empty(t, manifest.JavaScript)
	assert.Empty(t, manifest.CSS)
	assert.Empty(t, manifest.Images)
	assert.Empty(t, manifest.Fonts)
	assert.Empty(t, manifest.Other)
	assert.NotNil(t, manifest.Dependencies)
	assert.Equal(t, 0, len(manifest.Dependencies))
}

// TestDiscoverAssets_WithAssets tests asset discovery with various file types
func TestDiscoverAssets_WithAssets(t *testing.T) {
	tmpDir := createTempDir(t)
	cfg := createTestConfig(t)
	bundler := NewAssetBundler(cfg, tmpDir)
	
	// Change to temp directory for relative path discovery
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)
	
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	
	// Create test assets
	createTestFile(t, "static/main.js", "console.log('Hello from main.js');")
	createTestFile(t, "static/app.css", "body { margin: 0; }")
	createTestFile(t, "static/logo.png", "\x89PNG\r\n\x1a\n") // PNG header
	createTestFile(t, "static/font.woff2", "wOFF2") // Simple font content
	createTestFile(t, "components/index.ts", "export * from './Button';")
	createTestFile(t, "components/Button.tsx", "import React from 'react';")
	
	ctx := context.Background()
	manifest, err := bundler.DiscoverAssets(ctx)
	
	require.NoError(t, err)
	require.NotNil(t, manifest)
	
	// Validate JavaScript files
	assert.Len(t, manifest.JavaScript, 3) // main.js, index.ts, Button.tsx
	jsFiles := make(map[string]AssetFile)
	for _, file := range manifest.JavaScript {
		jsFiles[file.Name] = file
		assert.Equal(t, "javascript", file.Type)
		assert.Greater(t, file.Size, int64(0))
		assert.NotEmpty(t, file.Hash)
		assert.False(t, file.ModTime.IsZero())
	}
	
	// Check specific JS files
	assert.Contains(t, jsFiles, "main.js")
	assert.Contains(t, jsFiles, "index.ts")
	assert.Contains(t, jsFiles, "Button.tsx")
	
	// Validate CSS files
	assert.Len(t, manifest.CSS, 1)
	cssFile := manifest.CSS[0]
	assert.Equal(t, "app.css", cssFile.Name)
	assert.Equal(t, "css", cssFile.Type)
	assert.Greater(t, cssFile.Size, int64(0))
	
	// Validate Images
	assert.Len(t, manifest.Images, 1)
	imageFile := manifest.Images[0]
	assert.Equal(t, "logo.png", imageFile.Name)
	assert.Equal(t, "image", imageFile.Type)
	
	// Validate Fonts
	assert.Len(t, manifest.Fonts, 1)
	fontFile := manifest.Fonts[0]
	assert.Equal(t, "font.woff2", fontFile.Name)
	assert.Equal(t, "font", fontFile.Type)
	
	// Check dependencies were analyzed
	assert.NotNil(t, manifest.Dependencies)
}

// TestDiscoverAssets_PathTraversalSecurity tests security against path traversal attacks
func TestDiscoverAssets_PathTraversalSecurity(t *testing.T) {
	tmpDir := createTempDir(t)
	cfg := createTestConfig(t)
	bundler := NewAssetBundler(cfg, tmpDir)
	
	// Change to temp directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)
	
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	
	// Create directory structure with potential security issues
	// Note: These paths will be normalized by the OS, but we test the discovery logic
	createTestFile(t, "static/safe.js", "console.log('safe');")
	createTestFile(t, ".git/config", "[core]")
	createTestFile(t, "node_modules/package/index.js", "module.exports = {};")
	createTestFile(t, "static/.hidden.js", "console.log('hidden');")
	
	// Create files that simulate traversal attempts (but within safe boundaries)
	createTestFile(t, "static/passwd", "simulated system file")
	createTestFile(t, "static/system32", "simulated windows file")
	
	ctx := context.Background()
	manifest, err := bundler.DiscoverAssets(ctx)
	
	require.NoError(t, err)
	require.NotNil(t, manifest)
	
	// Should only find the safe.js file, not hidden files or files in ignored directories
	jsFiles := make([]string, 0)
	for _, file := range manifest.JavaScript {
		jsFiles = append(jsFiles, file.Name)
	}
	
	// Should find safe.js but not hidden files
	assert.Contains(t, jsFiles, "safe.js")
	assert.NotContains(t, jsFiles, ".hidden.js") // Hidden files should be ignored
	assert.NotContains(t, jsFiles, "index.js") // node_modules should be ignored
	
	// Verify no ignored paths were processed
	allPaths := make([]string, 0)
	for _, files := range [][]AssetFile{
		manifest.JavaScript, manifest.CSS, manifest.Images, 
		manifest.Fonts, manifest.Other,
	} {
		for _, file := range files {
			allPaths = append(allPaths, file.Path)
		}
	}
	
	for _, path := range allPaths {
		assert.False(t, strings.Contains(path, ".git"), "Path should not include .git files: %s", path)
		assert.False(t, strings.Contains(path, "node_modules"), "Path should not include node_modules: %s", path)
		assert.False(t, strings.HasPrefix(filepath.Base(path), "."), "Hidden files should be ignored: %s", path)
	}
}

// TestGetAssetType validates asset type classification
func TestGetAssetType(t *testing.T) {
	bundler := NewAssetBundler(createTestConfig(t), createTempDir(t))
	
	testCases := []struct {
		ext      string
		expected string
	}{
		{".js", "javascript"},
		{".mjs", "javascript"},
		{".ts", "javascript"},
		{".jsx", "javascript"},
		{".tsx", "javascript"},
		{".css", "css"},
		{".scss", "css"},
		{".sass", "css"},
		{".less", "css"},
		{".png", "image"},
		{".jpg", "image"},
		{".jpeg", "image"},
		{".gif", "image"},
		{".svg", "image"},
		{".webp", "image"},
		{".avif", "image"},
		{".woff", "font"},
		{".woff2", "font"},
		{".ttf", "font"},
		{".otf", "font"},
		{".eot", "font"},
		{".html", "html"},
		{".htm", "html"},
		{".json", "data"},
		{".xml", "data"},
		{".txt", "data"},
		{".md", "data"},
		{".unknown", ""},
		{"", ""},
	}
	
	for _, tc := range testCases {
		t.Run(tc.ext, func(t *testing.T) {
			result := bundler.getAssetType(tc.ext)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestIsEntryPoint validates entry point detection
func TestIsEntryPoint(t *testing.T) {
	bundler := NewAssetBundler(createTestConfig(t), createTempDir(t))
	
	testCases := []struct {
		path      string
		assetType string
		expected  bool
	}{
		{"main.js", "javascript", true},
		{"index.js", "javascript", true},
		{"app.js", "javascript", true},
		{"entry.js", "javascript", true},
		{"main.css", "css", true},
		{"index.css", "css", true},
		{"app.css", "css", true},
		{"entry.css", "css", true},
		{"component.js", "javascript", false},
		{"utils.js", "javascript", false},
		{"style.css", "css", false},
		{"main.png", "image", false},
		{"index.woff", "font", false},
		{"", "javascript", false},
		{"main", "", false},
	}
	
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_%s", tc.path, tc.assetType), func(t *testing.T) {
			result := bundler.isEntryPoint(tc.path, tc.assetType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestCalculateFileHash validates file hash generation
func TestCalculateFileHash(t *testing.T) {
	tmpDir := createTempDir(t)
	bundler := NewAssetBundler(createTestConfig(t), tmpDir)
	
	// Test with known content
	testContent := "console.log('test');"
	testFile := filepath.Join(tmpDir, "test.js")
	createTestFile(t, testFile, testContent)
	
	hash1, err := bundler.calculateFileHash(testFile)
	require.NoError(t, err)
	assert.NotEmpty(t, hash1)
	assert.Equal(t, 12, len(hash1)) // Should truncate to 12 chars
	
	// Calculate expected hash manually
	hasher := sha256.New()
	hasher.Write([]byte(testContent))
	expectedHash := fmt.Sprintf("%x", hasher.Sum(nil))[:12]
	assert.Equal(t, expectedHash, hash1)
	
	// Test that same content produces same hash
	hash2, err := bundler.calculateFileHash(testFile)
	require.NoError(t, err)
	assert.Equal(t, hash1, hash2)
	
	// Test that different content produces different hash
	createTestFile(t, testFile, "different content")
	hash3, err := bundler.calculateFileHash(testFile)
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash3)
	
	// Test with non-existent file
	_, err = bundler.calculateFileHash("non-existent.js")
	assert.Error(t, err)
}

// TestBundle_JavaScript validates JavaScript bundling
func TestBundle_JavaScript(t *testing.T) {
	tmpDir := createTempDir(t)
	cfg := createTestConfig(t)
	outputDir := filepath.Join(tmpDir, "output")
	bundler := NewAssetBundler(cfg, outputDir)
	
	// Create test assets
	createTestFile(t, filepath.Join(tmpDir, "main.js"), "console.log('Hello from main.js');")
	createTestFile(t, filepath.Join(tmpDir, "utils.js"), "function util() { return 'utility'; }")
	
	// Create manifest
	mainFile := AssetFile{
		Path:    filepath.Join(tmpDir, "main.js"),
		Name:    "main.js",
		Type:    "javascript",
		Size:    30,
		Hash:    "abcd1234",
		ModTime: time.Now(),
		IsEntry: true,
	}
	
	utilsFile := AssetFile{
		Path:    filepath.Join(tmpDir, "utils.js"),
		Name:    "utils.js",
		Type:    "javascript",
		Size:    25,
		Hash:    "efgh5678",
		ModTime: time.Now(),
		IsEntry: false,
	}
	
	manifest := &AssetManifest{
		JavaScript:   []AssetFile{mainFile, utilsFile},
		Dependencies: make(map[string][]string),
	}
	
	options := BundlerOptions{
		Minify:      false,
		SourceMaps:  false,
		Environment: "development",
	}
	
	// Ensure output directories exist
	err := os.MkdirAll(filepath.Join(outputDir, "js"), 0755)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(outputDir, "css"), 0755)
	require.NoError(t, err)
	
	ctx := context.Background()
	bundledFiles, err := bundler.Bundle(ctx, manifest, options)
	
	require.NoError(t, err)
	require.NotEmpty(t, bundledFiles)
	
	// Check that bundle file was created
	bundleExists := false
	for _, bundledFile := range bundledFiles {
		if strings.Contains(bundledFile, "/js/") && strings.HasSuffix(bundledFile, ".js") {
			bundleExists = true
			
			// Verify bundle content
			content, err := os.ReadFile(bundledFile)
			require.NoError(t, err)
			
			bundleContent := string(content)
			assert.Contains(t, bundleContent, "Generated by Templar Build System")
			assert.Contains(t, bundleContent, "Hello from main.js")
			assert.Contains(t, bundleContent, "use strict")
			assert.Contains(t, bundleContent, "(function() {")
			assert.Contains(t, bundleContent, "})();")
		}
	}
	
	assert.True(t, bundleExists, "JavaScript bundle should have been created")
}

// TestBundle_CSS validates CSS bundling
func TestBundle_CSS(t *testing.T) {
	tmpDir := createTempDir(t)
	cfg := createTestConfig(t)
	outputDir := filepath.Join(tmpDir, "output")
	bundler := NewAssetBundler(cfg, outputDir)
	
	// Create test CSS files
	createTestFile(t, filepath.Join(tmpDir, "main.css"), "body { margin: 0; padding: 0; }")
	createTestFile(t, filepath.Join(tmpDir, "theme.css"), ".theme { color: blue; }")
	
	// Create manifest
	mainCSS := AssetFile{
		Path:    filepath.Join(tmpDir, "main.css"),
		Name:    "main.css",
		Type:    "css",
		Size:    30,
		Hash:    "css1234",
		ModTime: time.Now(),
		IsEntry: true,
	}
	
	themeCSS := AssetFile{
		Path:    filepath.Join(tmpDir, "theme.css"),
		Name:    "theme.css",
		Type:    "css",
		Size:    25,
		Hash:    "css5678",
		ModTime: time.Now(),
		IsEntry: false,
	}
	
	manifest := &AssetManifest{
		CSS:          []AssetFile{mainCSS, themeCSS},
		Dependencies: make(map[string][]string),
	}
	
	options := BundlerOptions{
		Minify:      false,
		SourceMaps:  false,
		Environment: "development",
	}
	
	// Ensure output directories exist
	err := os.MkdirAll(filepath.Join(outputDir, "js"), 0755)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(outputDir, "css"), 0755)
	require.NoError(t, err)
	
	ctx := context.Background()
	bundledFiles, err := bundler.Bundle(ctx, manifest, options)
	
	require.NoError(t, err)
	require.NotEmpty(t, bundledFiles)
	
	// Check that CSS bundle was created
	cssBundleExists := false
	for _, bundledFile := range bundledFiles {
		if strings.Contains(bundledFile, "css/main-") && strings.HasSuffix(bundledFile, ".css") {
			cssBundleExists = true
			
			// Verify bundle content
			content, err := os.ReadFile(bundledFile)
			require.NoError(t, err)
			
			bundleContent := string(content)
			assert.Contains(t, bundleContent, "Source: "+mainCSS.Path)
			assert.Contains(t, bundleContent, "Source: "+themeCSS.Path)
			assert.Contains(t, bundleContent, "body { margin: 0; padding: 0; }")
			assert.Contains(t, bundleContent, ".theme { color: blue; }")
		}
	}
	
	assert.True(t, cssBundleExists, "CSS bundle should have been created")
}

// TestBundle_WithMinification validates minification
func TestBundle_WithMinification(t *testing.T) {
	tmpDir := createTempDir(t)
	cfg := createTestConfig(t)
	outputDir := filepath.Join(tmpDir, "output")
	bundler := NewAssetBundler(cfg, outputDir)
	
	// Create test JavaScript with comments and extra whitespace
	jsContent := `
// This is a comment
console.log('Hello World');

/* Block comment */
function test() {
    return 'test';
}
`
	createTestFile(t, filepath.Join(tmpDir, "main.js"), jsContent)
	
	// Create test CSS with extra whitespace
	cssContent := `
/* CSS Comment */
body {
    margin: 0;
    
    padding: 10px;
}

.container {
    width: 100%;
}
`
	createTestFile(t, filepath.Join(tmpDir, "main.css"), cssContent)
	
	// Create manifest
	jsFile := AssetFile{
		Path:    filepath.Join(tmpDir, "main.js"),
		Name:    "main.js",
		Type:    "javascript",
		IsEntry: true,
	}
	
	cssFile := AssetFile{
		Path:    filepath.Join(tmpDir, "main.css"),
		Name:    "main.css",
		Type:    "css",
		IsEntry: true,
	}
	
	manifest := &AssetManifest{
		JavaScript:   []AssetFile{jsFile},
		CSS:          []AssetFile{cssFile},
		Dependencies: make(map[string][]string),
	}
	
	options := BundlerOptions{
		Minify:      true,
		SourceMaps:  false,
		Environment: "production",
	}
	
	// Ensure output directories exist
	err := os.MkdirAll(filepath.Join(outputDir, "js"), 0755)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(outputDir, "css"), 0755)
	require.NoError(t, err)
	
	ctx := context.Background()
	bundledFiles, err := bundler.Bundle(ctx, manifest, options)
	
	require.NoError(t, err)
	require.NotEmpty(t, bundledFiles)
	
	// Verify JS minification
	for _, bundledFile := range bundledFiles {
		if strings.Contains(bundledFile, ".js") {
			content, err := os.ReadFile(bundledFile)
			require.NoError(t, err)
			
			bundleContent := string(content)
			// Should not contain comments or extra whitespace
			assert.NotContains(t, bundleContent, "// This is a comment")
			assert.NotContains(t, bundleContent, "/* Block comment */")
		}
		
		if strings.Contains(bundledFile, ".css") {
			content, err := os.ReadFile(bundledFile)
			require.NoError(t, err)
			
			bundleContent := string(content)
			// Should be minified (no newlines, reduced whitespace)
			assert.NotContains(t, bundleContent, "\n")
			assert.NotContains(t, bundleContent, "\t")
		}
	}
}

// TestBundle_WithSourceMaps validates source map generation
func TestBundle_WithSourceMaps(t *testing.T) {
	tmpDir := createTempDir(t)
	cfg := createTestConfig(t)
	outputDir := filepath.Join(tmpDir, "output")
	bundler := NewAssetBundler(cfg, outputDir)
	
	// Create test JavaScript file
	createTestFile(t, filepath.Join(tmpDir, "main.js"), "console.log('Hello with source maps');")
	
	jsFile := AssetFile{
		Path:    filepath.Join(tmpDir, "main.js"),
		Name:    "main.js",
		Type:    "javascript",
		IsEntry: true,
	}
	
	manifest := &AssetManifest{
		JavaScript:   []AssetFile{jsFile},
		Dependencies: make(map[string][]string),
	}
	
	options := BundlerOptions{
		Minify:      false,
		SourceMaps:  true,
		Environment: "development",
	}
	
	// Ensure output directories exist
	err := os.MkdirAll(filepath.Join(outputDir, "js"), 0755)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(outputDir, "css"), 0755)
	require.NoError(t, err)
	
	ctx := context.Background()
	bundledFiles, err := bundler.Bundle(ctx, manifest, options)
	
	require.NoError(t, err)
	require.NotEmpty(t, bundledFiles)
	
	// Should have both JS bundle and source map
	jsBundle := ""
	sourceMap := ""
	
	for _, bundledFile := range bundledFiles {
		if strings.HasSuffix(bundledFile, ".js") && !strings.HasSuffix(bundledFile, ".js.map") {
			jsBundle = bundledFile
		} else if strings.HasSuffix(bundledFile, ".js.map") {
			sourceMap = bundledFile
		}
	}
	
	assert.NotEmpty(t, jsBundle, "JavaScript bundle should exist")
	assert.NotEmpty(t, sourceMap, "Source map should exist")
	
	// Verify source map content
	if sourceMap != "" {
		content, err := os.ReadFile(sourceMap)
		require.NoError(t, err)
		
		sourceMapContent := string(content)
		assert.Contains(t, sourceMapContent, `"version":3`)
		assert.Contains(t, sourceMapContent, `"sources"`)
		assert.Contains(t, sourceMapContent, jsFile.Path)
	}
}

// TestCopyWithFingerprint validates fingerprinted asset copying
func TestCopyWithFingerprint(t *testing.T) {
	tmpDir := createTempDir(t)
	outputDir := filepath.Join(tmpDir, "output")
	bundler := NewAssetBundler(createTestConfig(t), outputDir)
	
	// Create test image file
	testContent := "\x89PNG\r\n\x1a\nFake PNG content"
	testFile := filepath.Join(tmpDir, "logo.png")
	createTestFile(t, testFile, testContent)
	
	asset := AssetFile{
		Path: testFile,
		Name: "logo.png",
		Type: "image",
		Hash: "abcd1234",
	}
	
	processedPath, err := bundler.copyWithFingerprint(asset, "images")
	
	require.NoError(t, err)
	assert.Contains(t, processedPath, "output/images/logo-abcd1234.png")
	
	// Verify file was copied correctly
	assert.FileExists(t, processedPath)
	
	copiedContent, err := os.ReadFile(processedPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(copiedContent))
}

// TestAnalyzeDependencies validates dependency analysis
func TestAnalyzeDependencies(t *testing.T) {
	tmpDir := createTempDir(t)
	bundler := NewAssetBundler(createTestConfig(t), tmpDir)
	
	// Create JavaScript file with ES6 imports
	jsContent := `
import React from 'react';
import { Button } from './components/Button';
import utils from '../utils/helpers';
const lodash = require('lodash');
const path = require("path");
`
	jsFile := filepath.Join(tmpDir, "main.js")
	createTestFile(t, jsFile, jsContent)
	
	// Create CSS file with @import statements
	cssContent := `
@import "normalize.css";
@import './theme.css';
@import url("fonts.css");
body { margin: 0; }
`
	cssFile := filepath.Join(tmpDir, "main.css")
	createTestFile(t, cssFile, cssContent)
	
	manifest := &AssetManifest{
		JavaScript: []AssetFile{
			{Path: jsFile, Name: "main.js", Type: "javascript"},
		},
		CSS: []AssetFile{
			{Path: cssFile, Name: "main.css", Type: "css"},
		},
		Dependencies: make(map[string][]string),
	}
	
	err := bundler.analyzeDependencies(manifest)
	require.NoError(t, err)
	
	// Check JavaScript dependencies
	jsDeps := manifest.Dependencies[jsFile]
	require.NotEmpty(t, jsDeps)
	assert.Contains(t, jsDeps, "react")
	assert.Contains(t, jsDeps, "./components/Button")
	assert.Contains(t, jsDeps, "../utils/helpers")
	assert.Contains(t, jsDeps, "lodash")
	assert.Contains(t, jsDeps, "path")
	
	// Check CSS dependencies
	cssDeps := manifest.Dependencies[cssFile]
	require.NotEmpty(t, cssDeps)
	assert.Contains(t, cssDeps, "normalize.css")
	assert.Contains(t, cssDeps, "./theme.css")
	assert.Contains(t, cssDeps, "fonts.css")
}

// TestBundle_SecurityValidation validates security measures in bundling
func TestBundle_SecurityValidation(t *testing.T) {
	tmpDir := createTempDir(t)
	cfg := createTestConfig(t)
	outputDir := filepath.Join(tmpDir, "output")
	bundler := NewAssetBundler(cfg, outputDir)
	
	// Create malicious JavaScript content
	maliciousJS := `
// Attempt to access system
const fs = require('fs');
eval('malicious code here');
document.write('<script>alert("XSS")</script>');
`
	createTestFile(t, filepath.Join(tmpDir, "malicious.js"), maliciousJS)
	
	jsFile := AssetFile{
		Path:    filepath.Join(tmpDir, "malicious.js"),
		Name:    "malicious.js",
		Type:    "javascript",
		IsEntry: true,
	}
	
	manifest := &AssetManifest{
		JavaScript:   []AssetFile{jsFile},
		Dependencies: make(map[string][]string),
	}
	
	options := BundlerOptions{
		Minify:      false,
		SourceMaps:  false,
		Environment: "production",
	}
	
	// Ensure output directories exist
	err := os.MkdirAll(filepath.Join(outputDir, "js"), 0755)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(outputDir, "css"), 0755)
	require.NoError(t, err)
	
	ctx := context.Background()
	bundledFiles, err := bundler.Bundle(ctx, manifest, options)
	
	// Bundle should complete (bundler doesn't sanitize content, that's the responsibility of the original code)
	require.NoError(t, err)
	require.NotEmpty(t, bundledFiles)
	
	// However, the bundled files should be contained within the output directory
	for _, bundledFile := range bundledFiles {
		assert.True(t, strings.HasPrefix(bundledFile, outputDir), 
			"Bundled file should be within output directory: %s", bundledFile)
		
		// Verify no path traversal in output
		assert.False(t, strings.Contains(bundledFile, "../"), 
			"Bundled file path should not contain traversal: %s", bundledFile)
	}
}

// TestGenerateBundleName validates bundle name generation
func TestGenerateBundleName(t *testing.T) {
	bundler := NewAssetBundler(createTestConfig(t), createTempDir(t))
	
	name1 := bundler.generateBundleName("main", "js")
	time.Sleep(2 * time.Second) // Ensure different timestamps (Unix is in seconds)
	name2 := bundler.generateBundleName("main", "js")
	
	// Both should follow the pattern
	assert.Contains(t, name1, "main-")
	assert.Contains(t, name2, "main-")
	assert.True(t, strings.HasSuffix(name1, ".js"))
	assert.True(t, strings.HasSuffix(name2, ".js"))
	
	// Names should be different due to timestamp (after sufficient delay)
	assert.NotEqual(t, name1, name2)
}

// TestFindEntryPoints validates entry point detection
func TestFindEntryPoints(t *testing.T) {
	bundler := NewAssetBundler(createTestConfig(t), createTempDir(t))
	
	files := []AssetFile{
		{Name: "main.js", IsEntry: true},
		{Name: "utils.js", IsEntry: false},
		{Name: "app.js", IsEntry: true},
		{Name: "helper.js", IsEntry: false},
	}
	
	entryPoints := bundler.findEntryPoints(files)
	
	require.Len(t, entryPoints, 2)
	assert.Equal(t, "main.js", entryPoints[0].Name)
	assert.Equal(t, "app.js", entryPoints[1].Name)
}

// TestFindEntryPoints_NoExplicitEntries tests fallback behavior when no entry points are marked
func TestFindEntryPoints_NoExplicitEntries(t *testing.T) {
	bundler := NewAssetBundler(createTestConfig(t), createTempDir(t))
	
	files := []AssetFile{
		{Name: "utils.js", IsEntry: false},
		{Name: "helper.js", IsEntry: false},
	}
	
	entryPoints := bundler.findEntryPoints(files)
	
	// Should use first file as entry point
	require.Len(t, entryPoints, 1)
	assert.Equal(t, "utils.js", entryPoints[0].Name)
}

// TestFindEntryPoints_EmptyFiles tests behavior with no files
func TestFindEntryPoints_EmptyFiles(t *testing.T) {
	bundler := NewAssetBundler(createTestConfig(t), createTempDir(t))
	
	files := []AssetFile{}
	entryPoints := bundler.findEntryPoints(files)
	
	assert.Len(t, entryPoints, 0)
}

// TestBundle_ErrorHandling validates error handling in various scenarios
func TestBundle_ErrorHandling(t *testing.T) {
	tmpDir := createTempDir(t)
	cfg := createTestConfig(t)
	outputDir := filepath.Join(tmpDir, "output")
	bundler := NewAssetBundler(cfg, outputDir)
	
	t.Run("non-existent source file", func(t *testing.T) {
		jsFile := AssetFile{
			Path:    "/non/existent/file.js",
			Name:    "file.js",
			Type:    "javascript",
			IsEntry: true,
		}
		
		manifest := &AssetManifest{
			JavaScript:   []AssetFile{jsFile},
			Dependencies: make(map[string][]string),
		}
		
		options := BundlerOptions{}
		ctx := context.Background()
		
		_, err := bundler.Bundle(ctx, manifest, options)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "JavaScript bundling failed")
	})
	
	t.Run("invalid output directory", func(t *testing.T) {
		// Create a file where we expect a directory
		invalidOutputDir := filepath.Join(tmpDir, "invalid-output")
		createTestFile(t, invalidOutputDir, "this is a file, not a directory")
		
		invalidBundler := NewAssetBundler(cfg, invalidOutputDir)
		
		jsFile := AssetFile{
			Path:    filepath.Join(tmpDir, "valid.js"),
			Name:    "valid.js",
			Type:    "javascript",
			IsEntry: true,
		}
		createTestFile(t, jsFile.Path, "console.log('valid');")
		
		manifest := &AssetManifest{
			JavaScript:   []AssetFile{jsFile},
			Dependencies: make(map[string][]string),
		}
		
		options := BundlerOptions{}
		ctx := context.Background()
		
		_, err := invalidBundler.Bundle(ctx, manifest, options)
		assert.Error(t, err)
	})
}

// TestProcessOtherAssets validates non-JS/CSS asset processing
func TestProcessOtherAssets(t *testing.T) {
	tmpDir := createTempDir(t)
	cfg := createTestConfig(t)
	outputDir := filepath.Join(tmpDir, "output")
	bundler := NewAssetBundler(cfg, outputDir)
	
	// Create test assets
	createTestFile(t, filepath.Join(tmpDir, "logo.png"), "\x89PNG\r\n\x1a\nFake PNG")
	createTestFile(t, filepath.Join(tmpDir, "font.woff2"), "wOFF2 font data")
	createTestFile(t, filepath.Join(tmpDir, "data.json"), `{"test": true}`)
	
	manifest := &AssetManifest{
		Images: []AssetFile{
			{
				Path: filepath.Join(tmpDir, "logo.png"),
				Name: "logo.png",
				Type: "image",
				Hash: "img123",
			},
		},
		Fonts: []AssetFile{
			{
				Path: filepath.Join(tmpDir, "font.woff2"),
				Name: "font.woff2",
				Type: "font",
				Hash: "font456",
			},
		},
		Other: []AssetFile{
			{
				Path: filepath.Join(tmpDir, "data.json"),
				Name: "data.json",
				Type: "data",
				Hash: "data789",
			},
		},
	}
	
	options := BundlerOptions{}
	ctx := context.Background()
	
	processedFiles, err := bundler.processOtherAssets(ctx, manifest, options)
	
	require.NoError(t, err)
	require.Len(t, processedFiles, 3)
	
	// Verify files were copied with fingerprints
	for _, processedFile := range processedFiles {
		assert.FileExists(t, processedFile)
		
		if strings.Contains(processedFile, "logo-img123.png") {
			assert.Contains(t, processedFile, "images/logo-img123.png")
		} else if strings.Contains(processedFile, "font-font456.woff2") {
			assert.Contains(t, processedFile, "fonts/font-font456.woff2")
		} else if strings.Contains(processedFile, "data-data789.json") {
			assert.Contains(t, processedFile, "other/data-data789.json")
		}
	}
}

// TestMinifyJS validates JavaScript minification
func TestMinifyJS(t *testing.T) {
	bundler := NewAssetBundler(createTestConfig(t), createTempDir(t))
	
	input := `
// Single line comment
console.log('Hello World');

/* Block comment */
function test() {
    return 'test';
}

// Another comment
var x = 1;
`
	
	result := bundler.minifyJS(input)
	
	// Should remove comments and extra whitespace
	assert.NotContains(t, result, "//")
	assert.NotContains(t, result, "/*")
	assert.NotContains(t, result, "*/")
	assert.Contains(t, result, "console.log('Hello World');")
	assert.Contains(t, result, "function test() {")
	assert.Contains(t, result, "var x = 1;")
	
	// Should be more compact than original
	assert.Less(t, len(result), len(input))
}

// TestMinifyCSS validates CSS minification
func TestMinifyCSS(t *testing.T) {
	bundler := NewAssetBundler(createTestConfig(t), createTempDir(t))
	
	input := `
/* CSS comment */
body {
    margin: 0;
    padding: 10px;
}

.container {
    width: 100%;
}
`
	
	result := bundler.minifyCSS(input)
	
	// Should remove comments and reduce whitespace
	assert.NotContains(t, result, "/*")
	assert.NotContains(t, result, "*/")
	assert.NotContains(t, result, "\n")
	assert.NotContains(t, result, "\t")
	assert.Contains(t, result, "body {")
	assert.Contains(t, result, "margin: 0;")
	assert.Contains(t, result, ".container {")
	
	// Should be more compact than original
	assert.Less(t, len(result), len(input))
}

// TestCreateAssetFile validates asset file creation
func TestCreateAssetFile(t *testing.T) {
	tmpDir := createTempDir(t)
	bundler := NewAssetBundler(createTestConfig(t), tmpDir)
	
	// Create test file
	testContent := "console.log('test');"
	testFile := filepath.Join(tmpDir, "test.js")
	createTestFile(t, testFile, testContent)
	
	info, err := os.Stat(testFile)
	require.NoError(t, err)
	
	assetFile, err := bundler.createAssetFile(testFile, info)
	
	require.NoError(t, err)
	require.NotNil(t, assetFile)
	
	assert.Equal(t, testFile, assetFile.Path)
	assert.Equal(t, "test.js", assetFile.Name)
	assert.Equal(t, "javascript", assetFile.Type)
	assert.Equal(t, info.Size(), assetFile.Size)
	assert.NotEmpty(t, assetFile.Hash)
	assert.Equal(t, info.ModTime(), assetFile.ModTime)
	assert.False(t, assetFile.IsEntry) // test.js is not an entry point pattern
	
	// Test with entry point file
	mainFile := filepath.Join(tmpDir, "main.js")
	createTestFile(t, mainFile, testContent)
	mainInfo, err := os.Stat(mainFile)
	require.NoError(t, err)
	
	mainAsset, err := bundler.createAssetFile(mainFile, mainInfo)
	require.NoError(t, err)
	assert.True(t, mainAsset.IsEntry) // main.js should be detected as entry point
}

// TestCreateAssetFile_UnsupportedType validates handling of unsupported file types
func TestCreateAssetFile_UnsupportedType(t *testing.T) {
	tmpDir := createTempDir(t)
	bundler := NewAssetBundler(createTestConfig(t), tmpDir)
	
	// Create unsupported file type
	testFile := filepath.Join(tmpDir, "test.xyz")
	createTestFile(t, testFile, "unsupported content")
	
	info, err := os.Stat(testFile)
	require.NoError(t, err)
	
	assetFile, err := bundler.createAssetFile(testFile, info)
	
	require.NoError(t, err)
	assert.Nil(t, assetFile) // Should return nil for unsupported types
}