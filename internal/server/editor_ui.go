package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/conneroisu/templar/internal/types"
)

// handleEditorIndex serves the main editor interface
func (s *PreviewServer) handleEditorIndex(w http.ResponseWriter, r *http.Request) {
	// Check if a specific component is requested
	path := strings.TrimPrefix(r.URL.Path, "/editor")
	if path != "" && path != "/" {
		componentName := strings.TrimPrefix(path, "/")
		s.handleComponentEditorView(w, r, componentName)
		return
	}

	// Serve the main editor interface
	html := s.generateEditorHTML()
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// handleComponentEditorView serves the editor for a specific component
func (s *PreviewServer) handleComponentEditorView(w http.ResponseWriter, r *http.Request, componentName string) {
	// Validate component name
	if err := validateComponentName(componentName); err != nil {
		http.Error(w, "Invalid component name: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get component from registry
	component, exists := s.registry.Get(componentName)
	if !exists {
		http.NotFound(w, r)
		return
	}

	// Serve the component-specific editor interface
	html := s.generateComponentEditorHTML(component)
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// generateEditorHTML generates the main editor interface HTML
func (s *PreviewServer) generateEditorHTML() string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Templar Interactive Editor</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://unpkg.com/monaco-editor@0.44.0/min/vs/loader.js"></script>
    <style>
        .editor-container {
            height: 100vh;
            display: flex;
            flex-direction: column;
        }
        .editor-toolbar {
            height: 60px;
            background: #1e1e1e;
            color: white;
            display: flex;
            align-items: center;
            padding: 0 20px;
            gap: 15px;
        }
        .editor-content {
            flex: 1;
            display: flex;
        }
        .file-explorer {
            width: 300px;
            background: #252526;
            color: white;
            overflow-y: auto;
        }
        .editor-pane {
            flex: 1;
            display: flex;
            flex-direction: column;
        }
        .editor-tabs {
            height: 40px;
            background: #2d2d30;
            display: flex;
            align-items: center;
        }
        .editor-main {
            flex: 1;
            display: flex;
        }
        .code-editor {
            flex: 1;
        }
        .preview-pane {
            width: 400px;
            background: white;
            display: flex;
            flex-direction: column;
        }
        .preview-header {
            height: 40px;
            background: #f3f3f3;
            display: flex;
            align-items: center;
            padding: 0 15px;
            border-bottom: 1px solid #ddd;
        }
        .preview-content {
            flex: 1;
            padding: 15px;
            overflow-y: auto;
        }
        .props-panel {
            height: 200px;
            border-top: 1px solid #ddd;
            background: #fafafa;
            padding: 15px;
            overflow-y: auto;
        }
        .file-item {
            padding: 8px 15px;
            cursor: pointer;
            transition: background-color 0.2s;
        }
        .file-item:hover {
            background: rgba(255, 255, 255, 0.1);
        }
        .file-item.selected {
            background: #094771;
        }
        .file-icon {
            display: inline-block;
            width: 16px;
            margin-right: 8px;
        }
        .editor-tab {
            padding: 8px 15px;
            background: #2d2d30;
            color: #cccccc;
            cursor: pointer;
            border-right: 1px solid #3e3e42;
            position: relative;
        }
        .editor-tab.active {
            background: #1e1e1e;
            color: white;
        }
        .editor-tab .close-btn {
            margin-left: 8px;
            opacity: 0.7;
        }
        .editor-tab .close-btn:hover {
            opacity: 1;
        }
        .prop-input {
            width: 100%%;
            padding: 5px;
            margin: 5px 0;
            border: 1px solid #ddd;
            border-radius: 3px;
        }
        .error-line {
            background: rgba(255, 0, 0, 0.1);
            border-left: 3px solid #ff0000;
        }
        .warning-line {
            background: rgba(255, 165, 0, 0.1);
            border-left: 3px solid #ffa500;
        }
        .status-bar {
            height: 25px;
            background: #007acc;
            color: white;
            display: flex;
            align-items: center;
            padding: 0 15px;
            font-size: 12px;
        }
    </style>
</head>
<body class="editor-container">
    <!-- Toolbar -->
    <div class="editor-toolbar">
        <div class="flex items-center gap-4">
            <h1 class="text-lg font-semibold">Templar Editor</h1>
            <button id="newFile" class="px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700">
                New File
            </button>
            <button id="saveFile" class="px-3 py-1 bg-green-600 text-white rounded hover:bg-green-700">
                Save
            </button>
            <button id="formatCode" class="px-3 py-1 bg-purple-600 text-white rounded hover:bg-purple-700">
                Format
            </button>
            <button id="previewToggle" class="px-3 py-1 bg-gray-600 text-white rounded hover:bg-gray-700">
                Toggle Preview
            </button>
        </div>
        <div class="flex-1"></div>
        <div id="connectionStatus" class="text-sm">
            <span class="inline-block w-2 h-2 bg-green-500 rounded-full mr-2"></span>
            Connected
        </div>
    </div>

    <!-- Main Content -->
    <div class="editor-content">
        <!-- File Explorer -->
        <div class="file-explorer">
            <div class="p-3 border-b border-gray-600">
                <h3 class="text-sm font-semibold text-gray-300">EXPLORER</h3>
            </div>
            <div id="fileTree" class="p-2">
                <!-- File tree will be populated here -->
            </div>
        </div>

        <!-- Editor Pane -->
        <div class="editor-pane">
            <!-- Editor Tabs -->
            <div class="editor-tabs" id="editorTabs">
                <!-- Tabs will be added here -->
            </div>

            <!-- Editor and Preview -->
            <div class="editor-main">
                <!-- Code Editor -->
                <div class="code-editor">
                    <div id="monacoEditor" style="width: 100%%; height: 100%%;"></div>
                </div>

                <!-- Preview Pane -->
                <div class="preview-pane" id="previewPane">
                    <div class="preview-header">
                        <h4 class="text-sm font-semibold">Live Preview</h4>
                        <div class="flex-1"></div>
                        <button id="refreshPreview" class="text-xs text-gray-600 hover:text-gray-800">
                            Refresh
                        </button>
                    </div>
                    <div class="preview-content" id="previewContent">
                        <div class="text-gray-500 text-center py-8">
                            Select a component to see preview
                        </div>
                    </div>
                    <div class="props-panel">
                        <h5 class="text-sm font-semibold mb-2">Component Props</h5>
                        <div id="propsPanel">
                            <!-- Props inputs will be added here -->
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- Status Bar -->
    <div class="status-bar">
        <span id="statusText">Ready</span>
        <div class="flex-1"></div>
        <span id="cursorPosition">Ln 1, Col 1</span>
    </div>

    <script>
        %s
    </script>
</body>
</html>`, s.generateEditorJavaScript())
}

// generateComponentEditorHTML generates the editor HTML for a specific component
func (s *PreviewServer) generateComponentEditorHTML(component *types.ComponentInfo) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Editing: %s - Templar Editor</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://unpkg.com/monaco-editor@0.44.0/min/vs/loader.js"></script>
    <style>
        /* Same styles as main editor */
        .editor-container { height: 100vh; display: flex; flex-direction: column; }
        .editor-toolbar { height: 60px; background: #1e1e1e; color: white; display: flex; align-items: center; padding: 0 20px; gap: 15px; }
        .editor-content { flex: 1; display: flex; }
        .code-editor { flex: 1; }
        .preview-pane { width: 400px; background: white; display: flex; flex-direction: column; }
        .preview-header { height: 40px; background: #f3f3f3; display: flex; align-items: center; padding: 0 15px; border-bottom: 1px solid #ddd; }
        .preview-content { flex: 1; padding: 15px; overflow-y: auto; }
        .props-panel { height: 200px; border-top: 1px solid #ddd; background: #fafafa; padding: 15px; overflow-y: auto; }
        .status-bar { height: 25px; background: #007acc; color: white; display: flex; align-items: center; padding: 0 15px; font-size: 12px; }
    </style>
</head>
<body class="editor-container">
    <!-- Toolbar -->
    <div class="editor-toolbar">
        <div class="flex items-center gap-4">
            <a href="/editor" class="text-blue-300 hover:text-blue-100">← Back to Editor</a>
            <h1 class="text-lg font-semibold">Editing: %s</h1>
            <button id="saveFile" class="px-3 py-1 bg-green-600 text-white rounded hover:bg-green-700">
                Save
            </button>
            <button id="formatCode" class="px-3 py-1 bg-purple-600 text-white rounded hover:bg-purple-700">
                Format
            </button>
        </div>
        <div class="flex-1"></div>
        <div class="text-sm text-gray-300">%s</div>
    </div>

    <!-- Main Content -->
    <div class="editor-content">
        <!-- Code Editor -->
        <div class="code-editor">
            <div id="monacoEditor" style="width: 100%%; height: 100%%;"></div>
        </div>

        <!-- Preview Pane -->
        <div class="preview-pane">
            <div class="preview-header">
                <h4 class="text-sm font-semibold">Live Preview</h4>
                <div class="flex-1"></div>
                <button id="refreshPreview" class="text-xs text-gray-600 hover:text-gray-800">
                    Refresh
                </button>
            </div>
            <div class="preview-content" id="previewContent">
                <div class="text-gray-500 text-center py-8">
                    Loading preview...
                </div>
            </div>
            <div class="props-panel">
                <h5 class="text-sm font-semibold mb-2">Component Props</h5>
                <div id="propsPanel">
                    <!-- Props inputs will be populated -->
                </div>
            </div>
        </div>
    </div>

    <!-- Status Bar -->
    <div class="status-bar">
        <span id="statusText">Ready</span>
        <div class="flex-1"></div>
        <span id="cursorPosition">Ln 1, Col 1</span>
    </div>

    <script>
        window.componentName = '%s';
        window.componentFilePath = '%s';
        %s
    </script>
</body>
</html>`, component.Name, component.Name, component.FilePath, component.Name, component.FilePath, s.generateComponentEditorJavaScript(component))
}

// generateEditorJavaScript generates JavaScript for the main editor
func (s *PreviewServer) generateEditorJavaScript() string {
	return `
        // Global state
        let editor = null;
        let currentFile = null;
        let openTabs = new Map();
        let ws = null;
        let previewVisible = true;

        // Initialize Monaco Editor
        require.config({ paths: { vs: 'https://unpkg.com/monaco-editor@0.44.0/min/vs' } });
        require(['vs/editor/editor.main'], function() {
            // Register templ language
            monaco.languages.register({ id: 'templ' });
            
            // Define templ language configuration
            monaco.languages.setLanguageConfiguration('templ', {
                comments: {
                    lineComment: '//',
                    blockComment: ['/*', '*/']
                },
                brackets: [
                    ['{', '}'],
                    ['[', ']'],
                    ['(', ')'],
                    ['<', '>']
                ],
                autoClosingPairs: [
                    { open: '{', close: '}' },
                    { open: '[', close: ']' },
                    { open: '(', close: ')' },
                    { open: '<', close: '>', notIn: ['string'] },
                    { open: '"', close: '"' },
                    { open: "'", close: "'" }
                ]
            });

            // Define templ syntax highlighting
            monaco.languages.setMonarchTokensProvider('templ', {
                tokenizer: {
                    root: [
                        [/templ\s+\w+/, 'keyword.templ'],
                        [/package\s+\w+/, 'keyword.package'],
                        [/import/, 'keyword.import'],
                        [/\{[^}]*\}/, 'expression.templ'],
                        [/<[^>]*>/, 'tag.html'],
                        [/\/\/.*$/, 'comment'],
                        [/\/\*[\s\S]*?\*\//, 'comment'],
                        [/"[^"]*"/, 'string'],
                        [/'[^']*'/, 'string'],
                        [/\d+/, 'number']
                    ]
                }
            });

            // Define theme for templ
            monaco.editor.defineTheme('templ-dark', {
                base: 'vs-dark',
                inherit: true,
                rules: [
                    { token: 'keyword.templ', foreground: '569cd6', fontStyle: 'bold' },
                    { token: 'keyword.package', foreground: 'c586c0' },
                    { token: 'keyword.import', foreground: 'c586c0' },
                    { token: 'expression.templ', foreground: 'dcdcaa' },
                    { token: 'tag.html', foreground: '4ec9b0' },
                    { token: 'comment', foreground: '6a9955' },
                    { token: 'string', foreground: 'ce9178' },
                    { token: 'number', foreground: 'b5cea8' }
                ],
                colors: {}
            });

            // Create editor
            editor = monaco.editor.create(document.getElementById('monacoEditor'), {
                value: '// Select a file to start editing',
                language: 'templ',
                theme: 'templ-dark',
                automaticLayout: true,
                minimap: { enabled: true },
                scrollBeyondLastLine: false,
                lineNumbers: 'on',
                renderWhitespace: 'selection',
                folding: true,
                wordWrap: 'on'
            });

            // Listen for content changes
            editor.onDidChangeModelContent(() => {
                if (currentFile) {
                    debounceValidation();
                    debouncePreview();
                }
            });

            // Listen for cursor position changes
            editor.onDidChangeCursorPosition((e) => {
                document.getElementById('cursorPosition').textContent = 
                    'Ln ' + e.position.lineNumber + ', Col ' + e.position.column;
            });

            // Initialize WebSocket connection
            initWebSocket();
            
            // Load file tree
            loadFileTree();
        });

        // WebSocket connection for live updates
        function initWebSocket() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            ws = new WebSocket(protocol + '//' + window.location.host + '/ws');
            
            ws.onopen = function() {
                document.getElementById('connectionStatus').innerHTML = 
                    '<span class="inline-block w-2 h-2 bg-green-500 rounded-full mr-2"></span>Connected';
            };
            
            ws.onclose = function() {
                document.getElementById('connectionStatus').innerHTML = 
                    '<span class="inline-block w-2 h-2 bg-red-500 rounded-full mr-2"></span>Disconnected';
                // Attempt to reconnect
                setTimeout(initWebSocket, 3000);
            };
            
            ws.onmessage = function(event) {
                const message = JSON.parse(event.data);
                handleWebSocketMessage(message);
            };
        }

        function handleWebSocketMessage(message) {
            if (message.type === 'file_changed' && message.target === currentFile) {
                // File was changed externally, ask user if they want to reload
                if (confirm('File has been changed externally. Reload?')) {
                    loadFile(currentFile);
                }
            }
        }

        // Debounced validation
        let validationTimeout;
        function debounceValidation() {
            clearTimeout(validationTimeout);
            validationTimeout = setTimeout(validateContent, 500);
        }

        // Debounced preview update
        let previewTimeout;
        function debouncePreview() {
            clearTimeout(previewTimeout);
            previewTimeout = setTimeout(updatePreview, 1000);
        }

        // Validate editor content
        function validateContent() {
            if (!editor || !currentFile) return;
            
            const content = editor.getValue();
            
            fetch('/api/editor', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    action: 'validate',
                    content: content,
                    component_name: getComponentNameFromFile(currentFile)
                })
            })
            .then(response => response.json())
            .then(data => {
                // Clear existing markers
                monaco.editor.setModelMarkers(editor.getModel(), 'templ', []);
                
                // Add error markers
                if (data.errors && data.errors.length > 0) {
                    const markers = data.errors.map(error => ({
                        startLineNumber: error.line,
                        startColumn: error.column || 1,
                        endLineNumber: error.line,
                        endColumn: (error.column || 1) + 10,
                        message: error.message,
                        severity: error.severity === 'error' ? 
                            monaco.MarkerSeverity.Error : monaco.MarkerSeverity.Warning
                    }));
                    monaco.editor.setModelMarkers(editor.getModel(), 'templ', markers);
                }
                
                // Update status
                const errorCount = data.errors ? data.errors.length : 0;
                const warningCount = data.warnings ? data.warnings.length : 0;
                document.getElementById('statusText').textContent = 
                    errorCount > 0 ? errorCount + ' error(s)' :
                    warningCount > 0 ? warningCount + ' warning(s)' : 'Ready';
            })
            .catch(error => {
                console.error('Validation error:', error);
            });
        }

        // Update preview
        function updatePreview() {
            if (!editor || !currentFile || !previewVisible) return;
            
            const content = editor.getValue();
            
            fetch('/api/editor', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    action: 'preview',
                    content: content,
                    component_name: getComponentNameFromFile(currentFile)
                })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success && data.preview_html) {
                    document.getElementById('previewContent').innerHTML = data.preview_html;
                } else {
                    document.getElementById('previewContent').innerHTML = 
                        '<div class="text-red-500">Preview error: ' + (data.errors?.[0]?.message || 'Unknown error') + '</div>';
                }
            })
            .catch(error => {
                console.error('Preview error:', error);
                document.getElementById('previewContent').innerHTML = 
                    '<div class="text-red-500">Preview failed</div>';
            });
        }

        // Load file tree
        function loadFileTree() {
            fetch('/api/files', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ action: 'list', file_path: '.' })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success && data.files) {
                    renderFileTree(data.files);
                }
            })
            .catch(error => {
                console.error('Failed to load file tree:', error);
            });
        }

        // Render file tree
        function renderFileTree(files) {
            const fileTree = document.getElementById('fileTree');
            fileTree.innerHTML = '';
            
            files.forEach(file => {
                if (file.is_component) {
                    const item = document.createElement('div');
                    item.className = 'file-item';
                    item.innerHTML = 
                        '<span class="file-icon">📄</span>' +
                        file.name;
                    item.onclick = () => loadFile(file.path);
                    fileTree.appendChild(item);
                }
            });
        }

        // Load file content
        function loadFile(filePath) {
            fetch('/api/files', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ action: 'open', file_path: filePath })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    currentFile = filePath;
                    editor.setValue(data.content);
                    
                    // Update UI
                    document.querySelectorAll('.file-item').forEach(item => 
                        item.classList.remove('selected'));
                    event.target.closest('.file-item')?.classList.add('selected');
                    
                    // Validate and preview
                    setTimeout(() => {
                        validateContent();
                        updatePreview();
                    }, 100);
                }
            })
            .catch(error => {
                console.error('Failed to load file:', error);
            });
        }

        // Save current file
        function saveFile() {
            if (!editor || !currentFile) return;
            
            const content = editor.getValue();
            
            fetch('/api/editor', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    action: 'save',
                    content: content,
                    file_path: currentFile
                })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    document.getElementById('statusText').textContent = 'Saved';
                    setTimeout(() => {
                        document.getElementById('statusText').textContent = 'Ready';
                    }, 2000);
                } else {
                    alert('Save failed: ' + (data.errors?.[0]?.message || 'Unknown error'));
                }
            })
            .catch(error => {
                console.error('Save error:', error);
                alert('Save failed');
            });
        }

        // Format code
        function formatCode() {
            if (!editor || !currentFile) return;
            
            const content = editor.getValue();
            
            fetch('/api/editor', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    action: 'format',
                    content: content
                })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success && data.content) {
                    editor.setValue(data.content);
                    document.getElementById('statusText').textContent = 'Formatted';
                    setTimeout(() => {
                        document.getElementById('statusText').textContent = 'Ready';
                    }, 2000);
                }
            })
            .catch(error => {
                console.error('Format error:', error);
            });
        }

        // Toggle preview pane
        function togglePreview() {
            const previewPane = document.getElementById('previewPane');
            previewVisible = !previewVisible;
            previewPane.style.display = previewVisible ? 'flex' : 'none';
            editor.layout(); // Resize editor
        }

        // Utility function to extract component name from file path
        function getComponentNameFromFile(filePath) {
            const fileName = filePath.split('/').pop();
            return fileName.replace('.templ', '');
        }

        // Event listeners
        document.getElementById('saveFile').onclick = saveFile;
        document.getElementById('formatCode').onclick = formatCode;
        document.getElementById('previewToggle').onclick = togglePreview;
        document.getElementById('refreshPreview').onclick = updatePreview;

        // Keyboard shortcuts
        window.addEventListener('keydown', (e) => {
            if (e.ctrlKey || e.metaKey) {
                switch (e.key) {
                    case 's':
                        e.preventDefault();
                        saveFile();
                        break;
                    case 'k':
                        if (e.shiftKey) {
                            e.preventDefault();
                            formatCode();
                        }
                        break;
                }
            }
        });
    `;
}

// generateComponentEditorJavaScript generates JavaScript for component-specific editor
func (s *PreviewServer) generateComponentEditorJavaScript(component *types.ComponentInfo) string {
	return `
        // Component-specific editor JavaScript
        let editor = null;
        
        require.config({ paths: { vs: 'https://unpkg.com/monaco-editor@0.44.0/min/vs' } });
        require(['vs/editor/editor.main'], function() {
            // Same language setup as main editor
            monaco.languages.register({ id: 'templ' });
            monaco.languages.setLanguageConfiguration('templ', {
                comments: { lineComment: '//', blockComment: ['/*', '*/'] },
                brackets: [['{', '}'], ['[', ']'], ['(', ')'], ['<', '>']],
                autoClosingPairs: [
                    { open: '{', close: '}' },
                    { open: '[', close: ']' },
                    { open: '(', close: ')' },
                    { open: '<', close: '>', notIn: ['string'] },
                    { open: '"', close: '"' },
                    { open: "'", close: "'" }
                ]
            });

            // Load component file content
            fetch('/api/files', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ 
                    action: 'open', 
                    file_path: window.componentFilePath 
                })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    editor = monaco.editor.create(document.getElementById('monacoEditor'), {
                        value: data.content,
                        language: 'templ',
                        theme: 'vs-dark',
                        automaticLayout: true,
                        minimap: { enabled: true }
                    });
                    
                    // Auto-validate and preview
                    setTimeout(() => {
                        validateContent();
                        updatePreview();
                    }, 500);
                }
            });
        });

        function validateContent() {
            // Same validation logic as main editor
        }

        function updatePreview() {
            // Same preview logic as main editor
        }

        function saveFile() {
            // Same save logic as main editor
        }

        // Event listeners
        document.getElementById('saveFile').onclick = saveFile;
        document.getElementById('formatCode').onclick = () => {
            // Format code implementation
        };
        document.getElementById('refreshPreview').onclick = updatePreview;
    `
}