package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/types"
)

// EditorRequest represents a request to the interactive editor
type EditorRequest struct {
	ComponentName string `json:"component_name"`
	Content       string `json:"content"`
	Action        string `json:"action"` // "save", "validate", "preview", "format"
	FilePath      string `json:"file_path,omitempty"`
}

// EditorResponse represents a response from the interactive editor
type EditorResponse struct {
	Success           bool                  `json:"success"`
	Content           string                `json:"content,omitempty"`
	PreviewHTML       string                `json:"preview_html,omitempty"`
	Errors            []EditorError         `json:"errors,omitempty"`
	Warnings          []EditorWarning       `json:"warnings,omitempty"`
	Suggestions       []EditorSuggestion    `json:"suggestions,omitempty"`
	ComponentMetadata *ComponentMetadata    `json:"metadata,omitempty"`
	ParsedParameters  []types.ParameterInfo `json:"parsed_parameters,omitempty"`
	Message           string                `json:"message,omitempty"`
}

// EditorError represents an error in the editor
type EditorError struct {
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // "error", "warning", "info"
	Source   string `json:"source"`   // "syntax", "validation", "runtime"
}

// EditorWarning represents a warning in the editor
type EditorWarning struct {
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// EditorSuggestion represents a code suggestion
type EditorSuggestion struct {
	Label       string `json:"label"`
	Kind        string `json:"kind"` // "snippet", "keyword", "function", "variable"
	InsertText  string `json:"insert_text"`
	Detail      string `json:"detail"`
	Description string `json:"description"`
}

// FileRequest represents a file operation request
type FileRequest struct {
	Action   string `json:"action"` // "open", "save", "create", "delete", "list"
	FilePath string `json:"file_path"`
	Content  string `json:"content,omitempty"`
	Name     string `json:"name,omitempty"`
}

// FileResponse represents a file operation response
type FileResponse struct {
	Success bool       `json:"success"`
	Content string     `json:"content,omitempty"`
	Files   []FileInfo `json:"files,omitempty"`
	Error   string     `json:"error,omitempty"`
	Message string     `json:"message,omitempty"`
}

// FileInfo represents file information
type FileInfo struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	IsDirectory  bool      `json:"is_directory"`
	Size         int64     `json:"size"`
	ModifiedTime time.Time `json:"modified_time"`
	IsComponent  bool      `json:"is_component"`
}

// handleEditorAPI handles the main editor API endpoint
func (s *PreviewServer) handleEditorAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req EditorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate component name if provided
	if req.ComponentName != "" {
		if err := validateComponentName(req.ComponentName); err != nil {
			response := EditorResponse{
				Success: false,
				Errors: []EditorError{{
					Line:     1,
					Column:   1,
					Message:  "Invalid component name: " + err.Error(),
					Severity: "error",
					Source:   "validation",
				}},
			}
			s.writeJSONResponse(w, response)
			return
		}
	}

	switch req.Action {
	case "validate":
		s.handleEditorValidate(w, req)
	case "preview":
		s.handleEditorPreview(w, req)
	case "save":
		s.handleEditorSave(w, req)
	case "format":
		s.handleEditorFormat(w, req)
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
	}
}

// handleEditorValidate validates templ content
func (s *PreviewServer) handleEditorValidate(w http.ResponseWriter, req EditorRequest) {
	response := EditorResponse{Success: true}

	// Parse templ content for syntax errors
	errors, warnings := s.validateTemplContent(req.Content)
	response.Errors = errors
	response.Warnings = warnings

	// Extract component parameters if valid
	if len(errors) == 0 {
		params, err := s.parseTemplParameters(req.Content)
		if err == nil {
			response.ParsedParameters = params
		}
	}

	// Generate suggestions
	response.Suggestions = s.generateEditorSuggestions(req.Content)

	s.writeJSONResponse(w, response)
}

// handleEditorPreview generates preview for templ content
func (s *PreviewServer) handleEditorPreview(w http.ResponseWriter, req EditorRequest) {
	response := EditorResponse{Success: true}

	// First validate the content
	errors, _ := s.validateTemplContent(req.Content)
	if len(errors) > 0 {
		response.Success = false
		response.Errors = errors
		s.writeJSONResponse(w, response)
		return
	}

	// Create temporary component for preview
	tempComponent := &types.ComponentInfo{
		Name:     req.ComponentName,
		Package:  "temp",
		FilePath: "temp.templ",
	}

	// Extract parameters from content
	params, err := s.parseTemplParameters(req.Content)
	if err == nil {
		tempComponent.Parameters = params
	}

	// Generate mock props for preview
	mockProps := s.generateIntelligentMockData(tempComponent)

	// Generate preview HTML
	previewHTML, err := s.renderTemplContentWithProps(req.Content, mockProps)
	if err != nil {
		response.Success = false
		response.Errors = []EditorError{{
			Line:     1,
			Column:   1,
			Message:  "Preview generation failed: " + err.Error(),
			Severity: "error",
			Source:   "runtime",
		}}
	} else {
		response.PreviewHTML = previewHTML
		response.ComponentMetadata = s.buildComponentMetadata(tempComponent)
	}

	s.writeJSONResponse(w, response)
}

// handleEditorSave saves templ content to file
func (s *PreviewServer) handleEditorSave(w http.ResponseWriter, req EditorRequest) {
	response := EditorResponse{Success: true}

	// Validate file path
	if req.FilePath == "" {
		response.Success = false
		response.Errors = []EditorError{{
			Message:  "File path is required",
			Severity: "error",
			Source:   "validation",
		}}
		s.writeJSONResponse(w, response)
		return
	}

	// Validate content before saving
	errors, warnings := s.validateTemplContent(req.Content)
	if len(errors) > 0 {
		response.Success = false
		response.Errors = errors
		response.Warnings = warnings
		s.writeJSONResponse(w, response)
		return
	}

	// Ensure directory exists
	dir := filepath.Dir(req.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		response.Success = false
		response.Errors = []EditorError{{
			Message:  "Failed to create directory: " + err.Error(),
			Severity: "error",
			Source:   "filesystem",
		}}
		s.writeJSONResponse(w, response)
		return
	}

	// Write file
	if err := os.WriteFile(req.FilePath, []byte(req.Content), 0644); err != nil {
		response.Success = false
		response.Errors = []EditorError{{
			Message:  "Failed to save file: " + err.Error(),
			Severity: "error",
			Source:   "filesystem",
		}}
	} else {
		response.Message = "File saved successfully"

		// Trigger component scan to update registry
		go func() {
			time.Sleep(100 * time.Millisecond) // Small delay to ensure file is written
			s.scanner.ScanDirectory(dir)
		}()
	}

	s.writeJSONResponse(w, response)
}

// handleEditorFormat formats templ content
func (s *PreviewServer) handleEditorFormat(w http.ResponseWriter, req EditorRequest) {
	response := EditorResponse{Success: true}

	// Format the content (basic formatting for now)
	formatted := s.formatTemplContent(req.Content)
	response.Content = formatted
	response.Message = "Content formatted"

	s.writeJSONResponse(w, response)
}

// handleFileAPI handles file operations
func (s *PreviewServer) handleFileAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req FileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON request: "+err.Error(), http.StatusBadRequest)
		return
	}

	switch req.Action {
	case "open":
		s.handleFileOpen(w, req)
	case "save":
		s.handleFileSave(w, req)
	case "create":
		s.handleFileCreate(w, req)
	case "delete":
		s.handleFileDelete(w, req)
	case "list":
		s.handleFileList(w, req)
	default:
		response := FileResponse{
			Success: false,
			Error:   "Invalid action",
		}
		s.writeJSONResponse(w, response)
	}
}

// handleFileOpen opens a file for editing
func (s *PreviewServer) handleFileOpen(w http.ResponseWriter, req FileRequest) {
	response := FileResponse{Success: true}

	// Validate file path
	if !s.isValidFilePath(req.FilePath) {
		response.Success = false
		response.Error = "Invalid file path"
		s.writeJSONResponse(w, response)
		return
	}

	// Read file content
	content, err := os.ReadFile(req.FilePath)
	if err != nil {
		response.Success = false
		response.Error = "Failed to read file: " + err.Error()
	} else {
		response.Content = string(content)
	}

	s.writeJSONResponse(w, response)
}

// handleFileSave saves a file
func (s *PreviewServer) handleFileSave(w http.ResponseWriter, req FileRequest) {
	response := FileResponse{Success: true}

	// Validate file path
	if !s.isValidFilePath(req.FilePath) {
		response.Success = false
		response.Error = "Invalid file path"
		s.writeJSONResponse(w, response)
		return
	}

	// Write file
	if err := os.WriteFile(req.FilePath, []byte(req.Content), 0644); err != nil {
		response.Success = false
		response.Error = "Failed to save file: " + err.Error()
	} else {
		response.Message = "File saved successfully"
	}

	s.writeJSONResponse(w, response)
}

// handleFileCreate creates a new file
func (s *PreviewServer) handleFileCreate(w http.ResponseWriter, req FileRequest) {
	response := FileResponse{Success: true}

	// Validate file path
	if !s.isValidFilePath(req.FilePath) {
		response.Success = false
		response.Error = "Invalid file path"
		s.writeJSONResponse(w, response)
		return
	}

	// Check if file already exists
	if _, err := os.Stat(req.FilePath); err == nil {
		response.Success = false
		response.Error = "File already exists"
		s.writeJSONResponse(w, response)
		return
	}

	// Create directory if needed
	dir := filepath.Dir(req.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		response.Success = false
		response.Error = "Failed to create directory: " + err.Error()
		s.writeJSONResponse(w, response)
		return
	}

	// Create file with default content or provided content
	content := req.Content
	if content == "" && strings.HasSuffix(req.FilePath, ".templ") {
		content = s.generateDefaultTemplContent(req.Name)
	}

	if err := os.WriteFile(req.FilePath, []byte(content), 0644); err != nil {
		response.Success = false
		response.Error = "Failed to create file: " + err.Error()
	} else {
		response.Message = "File created successfully"
	}

	s.writeJSONResponse(w, response)
}

// handleFileDelete deletes a file
func (s *PreviewServer) handleFileDelete(w http.ResponseWriter, req FileRequest) {
	response := FileResponse{Success: true}

	// Validate file path
	if !s.isValidFilePath(req.FilePath) {
		response.Success = false
		response.Error = "Invalid file path"
		s.writeJSONResponse(w, response)
		return
	}

	// Delete file
	if err := os.Remove(req.FilePath); err != nil {
		response.Success = false
		response.Error = "Failed to delete file: " + err.Error()
	} else {
		response.Message = "File deleted successfully"
	}

	s.writeJSONResponse(w, response)
}

// handleFileList lists files in directory
func (s *PreviewServer) handleFileList(w http.ResponseWriter, req FileRequest) {
	response := FileResponse{Success: true}

	// Default to current directory if no path provided
	dirPath := req.FilePath
	if dirPath == "" {
		dirPath = "."
	}

	// Read directory
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		response.Success = false
		response.Error = "Failed to read directory: " + err.Error()
		s.writeJSONResponse(w, response)
		return
	}

	// Build file list
	var files []FileInfo
	for _, entry := range entries {
		fileInfo := FileInfo{
			Name:         entry.Name(),
			Path:         filepath.Join(dirPath, entry.Name()),
			IsDirectory:  entry.IsDir(),
			Size:         entry.Size(),
			ModifiedTime: entry.ModTime(),
			IsComponent:  strings.HasSuffix(entry.Name(), ".templ"),
		}
		files = append(files, fileInfo)
	}

	response.Files = files
	s.writeJSONResponse(w, response)
}

// isValidFilePath validates file path for security
func (s *PreviewServer) isValidFilePath(filePath string) bool {
	// Basic path validation - prevent directory traversal
	if strings.Contains(filePath, "..") {
		return false
	}
	if strings.HasPrefix(filePath, "/") {
		return false
	}
	// Only allow .templ files for now
	return strings.HasSuffix(filePath, ".templ") || strings.HasSuffix(filePath, ".go")
}

// generateDefaultTemplContent generates default content for new templ files
func (s *PreviewServer) generateDefaultTemplContent(componentName string) string {
	if componentName == "" {
		componentName = "NewComponent"
	}

	return fmt.Sprintf(`package main

templ %s() {
	<div class="component">
		<h1>%s Component</h1>
		<p>This is a new component.</p>
	</div>
}
`, componentName, componentName)
}
