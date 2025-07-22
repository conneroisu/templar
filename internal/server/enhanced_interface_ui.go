package server

import (
	"fmt"
	"strings"

	"github.com/conneroisu/templar/internal/types"
)

// generateEnhancedEditorHTML creates the enhanced component editor interface
func (s *PreviewServer) generateEnhancedEditorHTML(component *types.ComponentInfo) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s - Enhanced Editor</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <script src="https://cdn.tailwindcss.com"></script>
    <script>
        tailwind.config = {
            darkMode: 'class',
            theme: {
                extend: {
                    colors: {
                        primary: { 50: '#eff6ff', 100: '#dbeafe', 500: '#3b82f6', 600: '#2563eb', 700: '#1d4ed8', 900: '#1e3a8a' },
                        secondary: { 50: '#f8fafc', 100: '#f1f5f9', 500: '#64748b', 600: '#475569', 700: '#334155', 900: '#0f172a' }
                    }
                }
            }
        }
    </script>
    <style>
        :root {
            --bg-primary: #ffffff;
            --bg-secondary: #f8fafc;
            --text-primary: #1e293b;
            --text-secondary: #64748b;
            --border: #e2e8f0;
            --accent: #3b82f6;
        }
        
        .dark {
            --bg-primary: #0f172a;
            --bg-secondary: #1e293b;
            --text-primary: #f1f5f9;
            --text-secondary: #94a3b8;
            --border: #334155;
            --accent: #60a5fa;
        }
        
        body {
            background-color: var(--bg-secondary);
            color: var(--text-primary);
            font-family: system-ui, -apple-system, sans-serif;
            margin: 0;
        }
        
        .editor-container {
            display: grid;
            grid-template-columns: 400px 1fr 350px;
            height: 100vh;
            gap: 1px;
            background: var(--border);
        }
        
        .sidebar {
            background: var(--bg-primary);
            overflow-y: auto;
            padding: 20px;
        }
        
        .main-panel {
            background: var(--bg-primary);
            display: flex;
            flex-direction: column;
            overflow: hidden;
        }
        
        .toolbar {
            background: var(--bg-secondary);
            padding: 12px 20px;
            border-bottom: 1px solid var(--border);
            display: flex;
            align-items: center;
            justify-content: space-between;
        }
        
        .preview-area {
            flex: 1;
            padding: 20px;
            overflow: auto;
            background: var(--bg-secondary);
        }
        
        .component-preview {
            background: var(--bg-primary);
            border-radius: 8px;
            padding: 40px;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
            min-height: 300px;
        }
        
        .prop-editor {
            display: flex;
            flex-direction: column;
            gap: 16px;
        }
        
        .prop-group {
            background: var(--bg-secondary);
            border-radius: 6px;
            padding: 16px;
            border: 1px solid var(--border);
        }
        
        .prop-label {
            font-weight: 600;
            margin-bottom: 8px;
            color: var(--text-primary);
            display: flex;
            align-items: center;
            gap: 8px;
        }
        
        .prop-input {
            width: 100%%;
            padding: 8px 12px;
            border: 1px solid var(--border);
            border-radius: 4px;
            background: var(--bg-primary);
            color: var(--text-primary);
            font-size: 14px;
            transition: border-color 0.2s, box-shadow 0.2s;
        }
        
        .prop-input:focus {
            outline: none;
            border-color: var(--accent);
            box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
        }
        
        .prop-input.error {
            border-color: #ef4444;
            box-shadow: 0 0 0 2px rgba(239, 68, 68, 0.1);
        }
        
        .type-badge {
            background: var(--accent);
            color: white;
            padding: 2px 6px;
            border-radius: 3px;
            font-size: 11px;
            font-weight: 500;
            text-transform: uppercase;
        }
        
        .required-badge {
            background: #ef4444;
            color: white;
            padding: 2px 6px;
            border-radius: 3px;
            font-size: 11px;
            font-weight: 500;
        }
        
        .error-message {
            color: #ef4444;
            font-size: 12px;
            margin-top: 4px;
        }
        
        .warning-message {
            color: #f59e0b;
            font-size: 12px;
            margin-top: 4px;
        }
        
        .prop-suggestion {
            background: #f3f4f6;
            border: 1px dashed #d1d5db;
            border-radius: 4px;
            padding: 8px;
            margin-top: 4px;
            cursor: pointer;
            font-size: 12px;
            color: #6b7280;
        }
        
        .prop-suggestion:hover {
            background: #e5e7eb;
        }
        
        .action-button {
            background: var(--accent);
            color: white;
            border: none;
            padding: 8px 16px;
            border-radius: 4px;
            font-weight: 500;
            cursor: pointer;
            font-size: 14px;
            transition: opacity 0.2s;
        }
        
        .action-button:hover {
            opacity: 0.9;
        }
        
        .action-button.secondary {
            background: var(--border);
            color: var(--text-primary);
        }
        
        .section-title {
            font-size: 18px;
            font-weight: 700;
            margin: 0 0 16px 0;
            color: var(--text-primary);
        }
        
        .code-display {
            background: #1e293b;
            color: #e2e8f0;
            padding: 16px;
            border-radius: 6px;
            font-family: 'Monaco', 'Menlo', monospace;
            font-size: 13px;
            line-height: 1.5;
            white-space: pre-wrap;
            overflow-x: auto;
            max-height: 200px;
        }
        
        .validation-status {
            padding: 8px 12px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: 500;
            margin-bottom: 16px;
        }
        
        .validation-status.valid {
            background: #dcfce7;
            color: #166534;
            border: 1px solid #bbf7d0;
        }
        
        .validation-status.invalid {
            background: #fef2f2;
            color: #dc2626;
            border: 1px solid #fecaca;
        }
        
        .loading {
            opacity: 0.5;
            pointer-events: none;
        }
        
        @media (max-width: 1200px) {
            .editor-container {
                grid-template-columns: 1fr;
                grid-template-rows: auto 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="editor-container">
        <!-- Property Editor Sidebar -->
        <div class="sidebar">
            <h1 class="section-title">%s Editor</h1>
            
            <div id="validationStatus" class="validation-status valid">
                ✓ All props valid
            </div>
            
            <div class="prop-editor" id="propEditor">
                <!-- Props will be dynamically populated -->
            </div>
            
            <div style="margin-top: 24px;">
                <button class="action-button" onclick="resetToDefaults()">
                    🔄 Reset to Defaults
                </button>
                <button class="action-button secondary" onclick="copyToClipboard()">
                    📋 Copy Code
                </button>
            </div>
        </div>
        
        <!-- Main Preview Area -->
        <div class="main-panel">
            <div class="toolbar">
                <div>
                    <span class="text-sm font-medium">Preview: %s</span>
                </div>
                <div class="flex gap-2">
                    <button class="action-button secondary" onclick="toggleTheme()">
                        🌓 Theme
                    </button>
                    <button class="action-button" onclick="refreshPreview()">
                        🔄 Refresh
                    </button>
                    <button class="action-button secondary" onclick="openPlayground()">
                        🎮 Playground
                    </button>
                </div>
            </div>
            
            <div class="preview-area">
                <div class="component-preview" id="componentPreview">
                    <div class="text-center text-gray-500">
                        Loading component preview...
                    </div>
                </div>
            </div>
        </div>
        
        <!-- Code and Info Panel -->
        <div class="sidebar">
            <h2 class="section-title">Generated Code</h2>
            <div class="code-display" id="generatedCode">
                Loading...
            </div>
            
            <h2 class="section-title" style="margin-top: 24px;">Component Info</h2>
            <div class="text-sm">
                <div class="mb-2">
                    <strong>Package:</strong> %s
                </div>
                <div class="mb-2">
                    <strong>File:</strong> %s
                </div>
                <div class="mb-2">
                    <strong>Parameters:</strong> %d
                </div>
                <div class="mb-2">
                    <strong>Dependencies:</strong> %d
                </div>
            </div>
            
            <button class="action-button" style="width: 100%%; margin-top: 16px;" onclick="window.history.back()">
                ← Back to List
            </button>
        </div>
    </div>
    
    <script>
        // Component state
        let componentName = '%s';
        let currentProps = {};
        let validationTimer = null;
        let renderTimer = null;
        
        // Initialize editor
        document.addEventListener('DOMContentLoaded', function() {
            initializeEditor();
        });
        
        function initializeEditor() {
            loadComponentMetadata();
            setupPropEditor();
            renderPreview();
        }
        
        function loadComponentMetadata() {
            // Component metadata is already rendered in the template
            // This function can be extended for dynamic loading
        }
        
        function setupPropEditor() {
            const editor = document.getElementById('propEditor');
            const parameters = %s; // Component parameters as JSON
            
            editor.innerHTML = '';
            currentProps = {};
            
            parameters.forEach(param => {
                const propGroup = createPropEditor(param);
                editor.appendChild(propGroup);
                
                // Initialize with default/mock value
                currentProps[param.name] = generateDefaultValue(param);
            });
        }
        
        function createPropEditor(param) {
            const group = document.createElement('div');
            group.className = 'prop-group';
            group.id = 'prop-' + param.name;
            
            // Label with badges
            const label = document.createElement('div');
            label.className = 'prop-label';
            
            const labelText = document.createElement('span');
            labelText.textContent = param.name;
            label.appendChild(labelText);
            
            const typeBadge = document.createElement('span');
            typeBadge.className = 'type-badge';
            typeBadge.textContent = param.type;
            label.appendChild(typeBadge);
            
            if (!param.optional) {
                const requiredBadge = document.createElement('span');
                requiredBadge.className = 'required-badge';
                requiredBadge.textContent = 'required';
                label.appendChild(requiredBadge);
            }
            
            group.appendChild(label);
            
            // Input field
            const input = createInputForParam(param);
            input.addEventListener('input', function() {
                updateProp(param.name, this.value, param.type);
            });
            
            group.appendChild(input);
            
            // Validation message container
            const messageDiv = document.createElement('div');
            messageDiv.id = 'message-' + param.name;
            group.appendChild(messageDiv);
            
            return group;
        }
        
        function createInputForParam(param) {
            const input = document.createElement('input');
            input.type = getInputType(param.type);
            input.className = 'prop-input';
            input.id = 'input-' + param.name;
            input.value = generateDefaultValue(param);
            
            if (param.type === 'bool') {
                input.type = 'checkbox';
                input.checked = generateDefaultValue(param);
                input.addEventListener('change', function() {
                    updateProp(param.name, this.checked, param.type);
                });
            }
            
            return input;
        }
        
        function getInputType(type) {
            switch (type) {
                case 'int':
                case 'int32':
                case 'int64':
                case 'float64':
                case 'float32':
                    return 'number';
                case 'bool':
                    return 'checkbox';
                default:
                    return 'text';
            }
        }
        
        function generateDefaultValue(param) {
            // Generate contextual default values
            const contextualValues = {
                'title': 'Sample Title',
                'text': 'Sample Text',
                'name': 'John Doe',
                'email': 'user@example.com',
                'url': 'https://example.com',
                'count': 5,
                'width': 300,
                'height': 200,
                'enabled': true,
                'visible': true,
                'active': true
            };
            
            if (contextualValues[param.name]) {
                return contextualValues[param.name];
            }
            
            switch (param.type) {
                case 'string':
                    return 'Sample ' + param.name;
                case 'int':
                case 'int32':
                case 'int64':
                    return 42;
                case 'float64':
                case 'float32':
                    return 3.14;
                case 'bool':
                    return true;
                default:
                    return '';
            }
        }
        
        function updateProp(propName, value, type) {
            // Convert value to appropriate type
            switch (type) {
                case 'bool':
                    currentProps[propName] = Boolean(value);
                    break;
                case 'int':
                case 'int32':
                case 'int64':
                    currentProps[propName] = parseInt(value, 10) || 0;
                    break;
                case 'float64':
                case 'float32':
                    currentProps[propName] = parseFloat(value) || 0.0;
                    break;
                default:
                    currentProps[propName] = value;
            }
            
            // Debounced validation and rendering
            clearTimeout(validationTimer);
            clearTimeout(renderTimer);
            
            validationTimer = setTimeout(validateProps, 300);
            renderTimer = setTimeout(renderPreview, 800);
        }
        
        async function validateProps() {
            try {
                const response = await fetch('/api/inline-editor', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        component_name: componentName,
                        props: currentProps,
                        action: 'validate'
                    })
                });
                
                const data = await response.json();
                updateValidationUI(data);
                
            } catch (error) {
                console.error('Validation failed:', error);
            }
        }
        
        function updateValidationUI(validationData) {
            const statusDiv = document.getElementById('validationStatus');
            
            if (validationData.valid) {
                statusDiv.className = 'validation-status valid';
                statusDiv.textContent = '✓ All props valid';
            } else {
                statusDiv.className = 'validation-status invalid';
                statusDiv.textContent = '⚠ ' + validationData.errors.length + ' validation errors';
            }
            
            // Clear previous error messages
            document.querySelectorAll('[id^="message-"]').forEach(msg => msg.innerHTML = '');
            document.querySelectorAll('.prop-input').forEach(input => input.classList.remove('error'));
            
            // Show validation errors
            if (validationData.errors) {
                validationData.errors.forEach(error => {
                    const messageDiv = document.getElementById('message-' + error.property);
                    const input = document.getElementById('input-' + error.property);
                    
                    if (messageDiv && input) {
                        const messageClass = error.severity === 'error' ? 'error-message' : 'warning-message';
                        messageDiv.innerHTML = '<div class="' + messageClass + '">' + error.message + '</div>';
                        
                        if (error.severity === 'error') {
                            input.classList.add('error');
                        }
                    }
                });
            }
        }
        
        async function renderPreview() {
            const previewDiv = document.getElementById('componentPreview');
            const codeDiv = document.getElementById('generatedCode');
            
            previewDiv.classList.add('loading');
            
            try {
                const response = await fetch('/api/inline-editor', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        component_name: componentName,
                        props: currentProps,
                        action: 'render'
                    })
                });
                
                const data = await response.json();
                
                if (data.error) {
                    previewDiv.innerHTML = '<div class="text-red-500 text-center p-8">Error: ' + data.error + '</div>';
                } else {
                    // Extract just the component HTML
                    const parser = new DOMParser();
                    const doc = parser.parseFromString(data.html, 'text/html');
                    const componentContent = doc.querySelector('.component');
                    
                    if (componentContent) {
                        previewDiv.innerHTML = componentContent.outerHTML;
                    } else {
                        previewDiv.innerHTML = data.html;
                    }
                    
                    codeDiv.textContent = data.generated_code || 'No code generated';
                }
                
            } catch (error) {
                previewDiv.innerHTML = '<div class="text-red-500 text-center p-8">Failed to render: ' + error.message + '</div>';
                console.error('Render failed:', error);
            } finally {
                previewDiv.classList.remove('loading');
            }
        }
        
        function resetToDefaults() {
            setupPropEditor();
            renderPreview();
        }
        
        function copyToClipboard() {
            const code = document.getElementById('generatedCode').textContent;
            navigator.clipboard.writeText(code).then(() => {
                // Show temporary feedback
                const button = event.target;
                const originalText = button.textContent;
                button.textContent = '✓ Copied!';
                setTimeout(() => button.textContent = originalText, 2000);
            });
        }
        
        function toggleTheme() {
            document.documentElement.classList.toggle('dark');
        }
        
        function refreshPreview() {
            renderPreview();
            validateProps();
        }
        
        function openPlayground() {
            window.open('/playground/' + componentName, '_blank');
        }
    </script>
</body>
</html>`, 
		component.Name, 
		component.Name, 
		component.Name,
		component.Package, 
		component.FilePath, 
		len(component.Parameters), 
		len(component.Dependencies),
		component.Name,
		s.parametersToJSON(component.Parameters))
}

// parametersToJSON converts component parameters to JSON for JavaScript
func (s *PreviewServer) parametersToJSON(parameters []types.ParameterInfo) string {
	var paramJSON []string
	
	for _, param := range parameters {
		paramJSON = append(paramJSON, fmt.Sprintf(`{
			"name": "%s",
			"type": "%s",
			"optional": %t
		}`, param.Name, param.Type, param.Optional))
	}
	
	return "[" + strings.Join(paramJSON, ",") + "]"
}

// generateEnhancedIndexHTML creates an enhanced index page with inline prop editing
func (s *PreviewServer) generateEnhancedIndexHTML() string {
	return `<!DOCTYPE html>
<html>
<head>
    <title>Templar - Enhanced Component Interface</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <script src="https://cdn.tailwindcss.com"></script>
    <script>
        tailwind.config = {
            darkMode: 'class',
            theme: {
                extend: {
                    colors: {
                        primary: { 50: '#eff6ff', 100: '#dbeafe', 500: '#3b82f6', 600: '#2563eb', 700: '#1d4ed8', 900: '#1e3a8a' },
                        secondary: { 50: '#f8fafc', 100: '#f1f5f9', 500: '#64748b', 600: '#475569', 700: '#334155', 900: '#0f172a' }
                    }
                }
            }
        }
    </script>
    <style>
        :root {
            --bg-primary: #ffffff;
            --bg-secondary: #f8fafc;
            --text-primary: #1e293b;
            --text-secondary: #64748b;
            --border: #e2e8f0;
            --accent: #3b82f6;
        }
        
        .dark {
            --bg-primary: #0f172a;
            --bg-secondary: #1e293b;
            --text-primary: #f1f5f9;
            --text-secondary: #94a3b8;
            --border: #334155;
            --accent: #60a5fa;
        }
        
        body {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: var(--text-primary);
            font-family: system-ui, -apple-system, sans-serif;
            margin: 0;
            min-height: 100vh;
        }
        
        .main-container {
            background: var(--bg-primary);
            min-height: 100vh;
        }
        
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 2rem;
            text-align: center;
        }
        
        .title {
            font-size: 2.5rem;
            font-weight: 800;
            margin-bottom: 0.5rem;
        }
        
        .subtitle {
            font-size: 1.125rem;
            opacity: 0.9;
        }
        
        .toolbar {
            background: var(--bg-secondary);
            border-bottom: 1px solid var(--border);
            padding: 1rem 2rem;
            display: flex;
            align-items: center;
            justify-content: space-between;
        }
        
        .view-toggle {
            display: flex;
            gap: 0.5rem;
        }
        
        .toggle-btn {
            padding: 0.5rem 1rem;
            border-radius: 0.375rem;
            border: 1px solid var(--border);
            background: var(--bg-primary);
            color: var(--text-primary);
            cursor: pointer;
            font-size: 0.875rem;
            transition: all 0.2s;
        }
        
        .toggle-btn.active {
            background: var(--accent);
            color: white;
            border-color: var(--accent);
        }
        
        .toggle-btn:hover {
            opacity: 0.8;
        }
        
        .content {
            padding: 2rem;
        }
        
        .component-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
            gap: 1.5rem;
        }
        
        .component-card {
            background: var(--bg-primary);
            border: 1px solid var(--border);
            border-radius: 0.75rem;
            overflow: hidden;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
            transition: all 0.2s;
        }
        
        .component-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
        }
        
        .card-header {
            padding: 1.5rem 1.5rem 1rem;
            border-bottom: 1px solid var(--border);
        }
        
        .card-title {
            font-size: 1.25rem;
            font-weight: 700;
            color: var(--accent);
            margin-bottom: 0.5rem;
        }
        
        .card-meta {
            font-size: 0.875rem;
            color: var(--text-secondary);
            margin-bottom: 0.5rem;
        }
        
        .card-params {
            font-size: 0.75rem;
            color: var(--text-secondary);
        }
        
        .card-body {
            padding: 1rem 1.5rem;
            background: var(--bg-secondary);
        }
        
        .quick-props {
            display: flex;
            flex-direction: column;
            gap: 0.75rem;
        }
        
        .prop-row {
            display: flex;
            align-items: center;
            gap: 0.75rem;
        }
        
        .prop-label {
            font-size: 0.75rem;
            font-weight: 500;
            color: var(--text-primary);
            min-width: 4rem;
        }
        
        .prop-input-inline {
            flex: 1;
            padding: 0.375rem 0.75rem;
            border: 1px solid var(--border);
            border-radius: 0.25rem;
            background: var(--bg-primary);
            color: var(--text-primary);
            font-size: 0.75rem;
        }
        
        .card-actions {
            padding: 1rem 1.5rem;
            background: var(--bg-primary);
            border-top: 1px solid var(--border);
            display: flex;
            gap: 0.5rem;
        }
        
        .action-btn {
            flex: 1;
            padding: 0.5rem;
            border-radius: 0.375rem;
            border: none;
            cursor: pointer;
            font-size: 0.75rem;
            font-weight: 500;
            transition: all 0.2s;
        }
        
        .action-btn.primary {
            background: var(--accent);
            color: white;
        }
        
        .action-btn.secondary {
            background: var(--border);
            color: var(--text-primary);
        }
        
        .action-btn:hover {
            opacity: 0.8;
        }
        
        .inline-preview {
            margin-top: 1rem;
            padding: 1rem;
            background: var(--bg-primary);
            border: 2px dashed var(--border);
            border-radius: 0.5rem;
            min-height: 100px;
            display: none;
        }
        
        .inline-preview.show {
            display: block;
        }
        
        .status-indicator {
            position: fixed;
            top: 1rem;
            right: 1rem;
            padding: 0.75rem 1rem;
            border-radius: 0.5rem;
            color: white;
            font-weight: 600;
            font-size: 0.875rem;
            z-index: 1000;
        }
        
        .status-indicator.connected {
            background: #10b981;
        }
        
        .status-indicator.disconnected {
            background: #ef4444;
        }
        
        .loading {
            opacity: 0.6;
            pointer-events: none;
        }
        
        .category-badge {
            display: inline-block;
            padding: 2px 6px;
            border-radius: 3px;
            font-size: 10px;
            font-weight: 600;
            text-transform: uppercase;
            color: white;
        }
        
        .category-badge.ui { background: #3b82f6; }
        .category-badge.layout { background: #8b5cf6; }
        .category-badge.form { background: #10b981; }
        .category-badge.data { background: #f59e0b; }
        .category-badge.navigation { background: #06b6d4; }
        .category-badge.feedback { background: #ef4444; }
        .category-badge.other { background: #6b7280; }
        
        .prop-controls {
            display: flex;
            gap: 0.5rem;
            margin-bottom: 1rem;
            flex-wrap: wrap;
        }
        
        .prop-combination-select,
        .state-select {
            flex: 1;
            min-width: 120px;
            padding: 0.25rem 0.5rem;
            border: 1px solid var(--border);
            border-radius: 0.25rem;
            background: var(--bg-primary);
            color: var(--text-primary);
            font-size: 0.75rem;
        }
        
        .mini-btn {
            padding: 0.25rem 0.5rem;
            border-radius: 0.25rem;
            border: 1px solid var(--border);
            background: var(--bg-primary);
            color: var(--text-primary);
            cursor: pointer;
            font-size: 0.75rem;
            font-weight: 500;
            transition: all 0.2s;
        }
        
        .mini-btn:hover {
            background: var(--accent);
            color: white;
        }
        
        .state-indicator {
            position: absolute;
            top: 0.5rem;
            right: 0.5rem;
            padding: 0.25rem 0.5rem;
            border-radius: 0.25rem;
            font-size: 0.6rem;
            font-weight: 600;
            text-transform: uppercase;
        }
        
        .state-indicator.loading { background: #fbbf24; color: #92400e; }
        .state-indicator.error { background: #fca5a5; color: #dc2626; }
        .state-indicator.disabled { background: #d1d5db; color: #6b7280; }
        .state-indicator.success { background: #86efac; color: #166534; }
    </style>
</head>
<body>
    <div class="main-container">
        <div class="header">
            <h1 class="title">🛠️ Enhanced Component Interface</h1>
            <p class="subtitle">Interactive component development with real-time prop editing</p>
        </div>
        
        <div class="toolbar">
            <div class="view-toggle">
                <button class="toggle-btn active" id="cardView" onclick="switchView('card')">
                    📋 Card View
                </button>
                <button class="toggle-btn" id="listView" onclick="switchView('list')">
                    📄 List View
                </button>
                <button class="toggle-btn" id="playgroundView" onclick="window.open('/playground', '_blank')">
                    🎮 Playground
                </button>
            </div>
            
            <div class="flex items-center gap-4">
                <input type="text" id="searchInput" placeholder="Search components..." 
                       class="px-3 py-2 border border-gray-300 rounded-md text-sm"
                       onkeyup="filterComponents()">
                <select id="categoryFilter" class="px-3 py-2 border border-gray-300 rounded-md text-sm" 
                        onchange="filterComponents()">
                    <option value="">All Categories</option>
                    <option value="ui">UI Components</option>
                    <option value="layout">Layout</option>
                    <option value="form">Forms</option>
                    <option value="data">Data Display</option>
                    <option value="navigation">Navigation</option>
                    <option value="feedback">Feedback</option>
                    <option value="other">Other</option>
                </select>
                <button class="toggle-btn" onclick="toggleTheme()">
                    🌓 Theme
                </button>
                <button class="toggle-btn" onclick="refreshComponents()">
                    🔄 Refresh
                </button>
            </div>
        </div>
        
        <div class="content">
            <div id="components" class="component-grid">
                <div class="component-card">
                    <div class="card-header">
                        <div class="card-title">Loading...</div>
                        <div class="card-meta">Discovering components...</div>
                    </div>
                </div>
            </div>
        </div>
    </div>
    
    <div id="status" class="status-indicator disconnected">
        Disconnected
    </div>
    
    <script>
        let ws;
        let reconnectInterval;
        let components = {};
        let componentProps = {}; // Store props for each component
        let propCombinations = {}; // Store saved prop combinations
        let filteredComponents = {}; // Currently filtered components
        let componentStates = {}; // Component states (loading, error, etc.)
        
        // Initialize
        document.addEventListener('DOMContentLoaded', function() {
            connect();
            loadComponents();
            loadSavedPropCombinations();
        });
        
        function connect() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            ws = new WebSocket(protocol + '//' + window.location.host + '/ws');
            
            ws.onopen = function() {
                document.getElementById('status').className = 'status-indicator connected';
                document.getElementById('status').textContent = 'Connected';
                clearInterval(reconnectInterval);
            };
            
            ws.onmessage = function(event) {
                const message = JSON.parse(event.data);
                handleMessage(message);
            };
            
            ws.onclose = function() {
                document.getElementById('status').className = 'status-indicator disconnected';
                document.getElementById('status').textContent = 'Disconnected';
                reconnectInterval = setInterval(connect, 2000);
            };
        }
        
        function handleMessage(message) {
            switch(message.type) {
                case 'full_reload':
                    window.location.reload();
                    break;
                case 'component_update':
                    loadComponents();
                    break;
            }
        }
        
        function loadComponents() {
            fetch('/components')
                .then(response => response.json())
                .then(data => {
                    components = data;
                    filteredComponents = components; // Initially show all components
                    renderComponents();
                })
                .catch(error => {
                    console.error('Failed to load components:', error);
                });
        }
        
        // Component categorization based on name patterns
        function categorizeComponent(component) {
            const name = component.name.toLowerCase();
            const params = component.parameters || [];
            
            // UI Components
            if (['button', 'link', 'icon', 'badge', 'avatar', 'chip'].some(term => name.includes(term))) {
                return 'ui';
            }
            
            // Layout Components
            if (['container', 'grid', 'flex', 'layout', 'column', 'row', 'section'].some(term => name.includes(term))) {
                return 'layout';
            }
            
            // Form Components
            if (['input', 'form', 'field', 'select', 'checkbox', 'radio', 'textarea'].some(term => name.includes(term))) {
                return 'form';
            }
            
            // Data Display
            if (['table', 'list', 'card', 'panel', 'display', 'chart', 'graph'].some(term => name.includes(term))) {
                return 'data';
            }
            
            // Navigation
            if (['nav', 'menu', 'breadcrumb', 'tab', 'stepper', 'pagination'].some(term => name.includes(term))) {
                return 'navigation';
            }
            
            // Feedback
            if (['alert', 'toast', 'modal', 'dialog', 'notification', 'progress', 'loader', 'spinner'].some(term => name.includes(term))) {
                return 'feedback';
            }
            
            // Check parameters for additional hints
            const hasErrorProps = params.some(p => p.name.toLowerCase().includes('error'));
            const hasLoadingProps = params.some(p => p.name.toLowerCase().includes('loading'));
            if (hasErrorProps || hasLoadingProps) {
                return 'feedback';
            }
            
            return 'other';
        }
        
        // Filter components based on search and category
        function filterComponents() {
            const searchTerm = document.getElementById('searchInput').value.toLowerCase();
            const selectedCategory = document.getElementById('categoryFilter').value;
            
            filteredComponents = {};
            
            Object.values(components).forEach(component => {
                const matchesSearch = !searchTerm || 
                    component.name.toLowerCase().includes(searchTerm) ||
                    (component.package && component.package.toLowerCase().includes(searchTerm)) ||
                    component.parameters.some(p => p.name.toLowerCase().includes(searchTerm));
                
                const componentCategory = categorizeComponent(component);
                const matchesCategory = !selectedCategory || componentCategory === selectedCategory;
                
                if (matchesSearch && matchesCategory) {
                    filteredComponents[component.name] = component;
                }
            });
            
            renderComponents();
        }
        
        function renderComponents() {
            const container = document.getElementById('components');
            
            if (Object.keys(filteredComponents).length === 0) {
                const hasComponents = Object.keys(components).length > 0;
                const message = hasComponents ? 'No components match your filters' : 'No components found';
                const submessage = hasComponents ? 'Try adjusting your search or category filter' : 'Create a .templ file to get started';
                
                container.innerHTML = 
                    '<div class="component-card">' +
                    '<div class="card-header">' +
                    '<div class="card-title">' + message + '</div>' +
                    '<div class="card-meta">' + submessage + '</div>' +
                    '</div>' +
                    '</div>';
                return;
            }
            
            container.innerHTML = '';
            
            Object.values(filteredComponents).forEach(component => {
                const card = createComponentCard(component);
                container.appendChild(card);
                
                // Initialize props for this component
                if (!componentProps[component.name]) {
                    componentProps[component.name] = generateDefaultProps(component);
                }
                
                // Initialize prop combinations storage
                if (!propCombinations[component.name]) {
                    propCombinations[component.name] = {
                        default: componentProps[component.name],
                        saved: {}
                    };
                }
                
                // Initialize component state
                if (!componentStates[component.name]) {
                    componentStates[component.name] = 'normal';
                }
                
                // Set up prop combination dropdown after DOM is ready
                setTimeout(() => updatePropCombinationOptions(component.name), 0);
            });
        }
        
        function createComponentCard(component) {
            const card = document.createElement('div');
            card.className = 'component-card';
            
            const params = component.parameters || [];
            const quickParams = params.slice(0, 3); // Show first 3 params for quick editing
            const category = categorizeComponent(component);
            const categoryLabel = category.charAt(0).toUpperCase() + category.slice(1);
            
            card.innerHTML = 
                '<div class="card-header">' +
                '<div class="card-title">' + component.name + '</div>' +
                '<div class="card-meta">' + component.filePath + '</div>' +
                '<div class="card-params">' +
                    '<span class="category-badge ' + category + '">' + categoryLabel + '</span> • ' +
                    'Package: ' + (component.package || 'unknown') + ' • ' + params.length + ' parameters' +
                '</div>' +
                '</div>' +
                '<div class="card-body">' +
                '<div class="prop-controls">' +
                    '<select class="prop-combination-select" id="combo-' + component.name + '" ' +
                    'onchange="loadPropCombination(\\\'' + component.name + '\\\', this.value)">' +
                        '<option value="default">Default Props</option>' +
                    '</select>' +
                    '<button class="mini-btn" onclick="savePropCombination(\\\'' + component.name + '\\\')">💾 Save</button>' +
                    '<select class="state-select" id="state-' + component.name + '" ' +
                    'onchange="changeComponentState(\\\'' + component.name + '\\\', this.value)">' +
                        '<option value="normal">Normal</option>' +
                        '<option value="loading">Loading</option>' +
                        '<option value="error">Error</option>' +
                        '<option value="disabled">Disabled</option>' +
                        '<option value="success">Success</option>' +
                    '</select>' +
                '</div>' +
                '<div class="quick-props">' + createQuickProps(component, quickParams) + '</div>' +
                '</div>' +
                '<div class="card-actions">' +
                '<button class="action-btn primary" onclick="openEditor(\\\'' + component.name + '\\\')">🛠️ Editor</button>' +
                '<button class="action-btn secondary" onclick="togglePreview(\\\'' + component.name + '\\\')">👁️ Preview</button>' +
                '<button class="action-btn secondary" onclick="openPlayground(\\\'' + component.name + '\\\')">🎮 Play</button>' +
                '</div>' +
                '<div class="inline-preview" id="preview-' + component.name + '">' +
                '<div class="text-center text-gray-500">Click "Preview" to render component</div>' +
                '</div>';
            
            return card;
        }
        
        function createQuickProps(component, params) {
            if (params.length === 0) {
                return '<div class="text-center text-gray-400 text-sm">No parameters</div>';
            }
            
            return params.map(param => {
                const inputType = getInputType(param.type);
                const value = componentProps[component.name]?.[param.name] || generateDefaultValue(param);
                
                return 
                    '<div class="prop-row">' +
                    '<label class="prop-label">' + param.name + ':</label>' +
                    '<input class="prop-input-inline" type="' + inputType + '" ' +
                    'value="' + value + '" ' +
                    'onchange="updateComponentProp(\\\'' + component.name + '\\\', \\\'' + param.name + '\\\', this.value, \\\'' + param.type + '\\\')">' +
                    '</div>';
            }).join('');
        }
        
        function generateDefaultProps(component) {
            const props = {};
            component.parameters.forEach(param => {
                props[param.name] = generateDefaultValue(param);
            });
            return props;
        }
        
        function generateDefaultValue(param) {
            const contextualValues = {
                'title': 'Sample Title',
                'text': 'Sample Text',
                'name': 'John Doe',
                'email': 'user@example.com',
                'count': 5
            };
            
            if (contextualValues[param.name]) {
                return contextualValues[param.name];
            }
            
            switch (param.type) {
                case 'string': return 'Sample ' + param.name;
                case 'int': case 'int32': case 'int64': return 42;
                case 'float64': case 'float32': return 3.14;
                case 'bool': return true;
                default: return '';
            }
        }
        
        function getInputType(type) {
            switch (type) {
                case 'int': case 'int32': case 'int64': 
                case 'float64': case 'float32':
                    return 'number';
                case 'bool':
                    return 'checkbox';
                default:
                    return 'text';
            }
        }
        
        function updateComponentProp(componentName, propName, value, type) {
            if (!componentProps[componentName]) {
                componentProps[componentName] = {};
            }
            
            // Convert value to appropriate type
            switch (type) {
                case 'bool':
                    componentProps[componentName][propName] = Boolean(value);
                    break;
                case 'int': case 'int32': case 'int64':
                    componentProps[componentName][propName] = parseInt(value, 10) || 0;
                    break;
                case 'float64': case 'float32':
                    componentProps[componentName][propName] = parseFloat(value) || 0.0;
                    break;
                default:
                    componentProps[componentName][propName] = value;
            }
            
            // If preview is open, update it
            const preview = document.getElementById('preview-' + componentName);
            if (preview && preview.classList.contains('show')) {
                renderInlinePreview(componentName);
            }
        }
        
        function togglePreview(componentName) {
            const preview = document.getElementById('preview-' + componentName);
            
            if (preview.classList.contains('show')) {
                preview.classList.remove('show');
            } else {
                preview.classList.add('show');
                renderInlinePreview(componentName);
            }
        }
        
        async function renderInlinePreview(componentName) {
            const preview = document.getElementById('preview-' + componentName);
            preview.classList.add('loading');
            
            try {
                const props = componentProps[componentName] || {};
                const response = await fetch('/api/inline-editor', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        component_name: componentName,
                        props: props,
                        action: 'render'
                    })
                });
                
                const data = await response.json();
                
                if (data.error) {
                    preview.innerHTML = '<div class="text-red-500 text-sm">Error: ' + data.error + '</div>';
                } else {
                    const parser = new DOMParser();
                    const doc = parser.parseFromString(data.html, 'text/html');
                    const componentContent = doc.querySelector('.component');
                    
                    if (componentContent) {
                        preview.innerHTML = componentContent.outerHTML;
                    } else {
                        preview.innerHTML = data.html;
                    }
                }
                
            } catch (error) {
                preview.innerHTML = '<div class="text-red-500 text-sm">Failed to render: ' + error.message + '</div>';
            } finally {
                preview.classList.remove('loading');
            }
        }
        
        function openEditor(componentName) {
            window.location.href = '/editor/' + componentName;
        }
        
        function openPlayground(componentName) {
            window.open('/playground/' + componentName, '_blank');
        }
        
        function switchView(view) {
            document.querySelectorAll('.toggle-btn').forEach(btn => btn.classList.remove('active'));
            document.getElementById(view + 'View').classList.add('active');
            
            // Implement different view layouts
            if (view === 'list') {
                // Switch to list view
                document.getElementById('components').className = 'flex flex-col gap-4';
            } else {
                // Switch to card view
                document.getElementById('components').className = 'component-grid';
            }
        }
        
        function toggleTheme() {
            document.documentElement.classList.toggle('dark');
        }
        
        function refreshComponents() {
            loadComponents();
        }
        
        // Prop combination management
        function savePropCombination(componentName) {
            const combinationName = prompt('Enter a name for this prop combination:');
            if (!combinationName) return;
            
            const currentProps = Object.assign({}, componentProps[componentName]);
            
            if (!propCombinations[componentName]) {
                propCombinations[componentName] = { default: {}, saved: {} };
            }
            
            propCombinations[componentName].saved[combinationName] = currentProps;
            
            // Update the select dropdown
            updatePropCombinationOptions(componentName);
            
            // Save to localStorage for persistence
            localStorage.setItem('templar_prop_combinations', JSON.stringify(propCombinations));
        }
        
        function loadPropCombination(componentName, combinationName) {
            let propsToLoad;
            
            if (combinationName === 'default') {
                propsToLoad = propCombinations[componentName].default;
            } else {
                propsToLoad = propCombinations[componentName].saved[combinationName];
            }
            
            if (propsToLoad) {
                componentProps[componentName] = Object.assign({}, propsToLoad);
                
                // Update the input fields in the quick props section
                updateQuickPropsInputs(componentName);
                
                // If preview is open, update it
                const preview = document.getElementById('preview-' + componentName);
                if (preview && preview.classList.contains('show')) {
                    renderInlinePreview(componentName);
                }
            }
        }
        
        function updatePropCombinationOptions(componentName) {
            const select = document.getElementById('combo-' + componentName);
            if (!select) return;
            
            const combinations = propCombinations[componentName];
            if (!combinations) return;
            
            // Clear existing options except default
            select.innerHTML = '<option value="default">Default Props</option>';
            
            // Add saved combinations
            Object.keys(combinations.saved).forEach(name => {
                const option = document.createElement('option');
                option.value = name;
                option.textContent = name;
                select.appendChild(option);
            });
        }
        
        function updateQuickPropsInputs(componentName) {
            const component = components[componentName];
            if (!component) return;
            
            const quickParams = component.parameters.slice(0, 3);
            
            quickParams.forEach(param => {
                const input = document.querySelector('input[onchange*="' + componentName + '"][onchange*="' + param.name + '"]');
                if (input && componentProps[componentName][param.name] !== undefined) {
                    const value = componentProps[componentName][param.name];
                    
                    if (param.type === 'bool') {
                        input.checked = Boolean(value);
                    } else {
                        input.value = value;
                    }
                }
            });
        }
        
        // Component state management
        function changeComponentState(componentName, newState) {
            componentStates[componentName] = newState;
            
            // Apply state-specific props
            const stateProps = getStateSpecificProps(componentName, newState);
            Object.assign(componentProps[componentName], stateProps);
            
            // Update component visually
            const card = document.querySelector('[onclick*="' + componentName + '"]').closest('.component-card');
            updateComponentStateIndicator(card, componentName, newState);
            
            // Update quick props inputs
            updateQuickPropsInputs(componentName);
            
            // If preview is open, update it
            const preview = document.getElementById('preview-' + componentName);
            if (preview && preview.classList.contains('show')) {
                renderInlinePreview(componentName);
            }
        }
        
        function getStateSpecificProps(componentName, state) {
            const component = components[componentName];
            const stateProps = {};
            
            switch (state) {
                case 'loading':
                    // Set loading-related props
                    component.parameters.forEach(param => {
                        if (param.name.toLowerCase().includes('loading')) {
                            stateProps[param.name] = true;
                        }
                        if (param.name.toLowerCase().includes('disabled')) {
                            stateProps[param.name] = true;
                        }
                    });
                    break;
                    
                case 'error':
                    component.parameters.forEach(param => {
                        if (param.name.toLowerCase().includes('error')) {
                            stateProps[param.name] = 'Something went wrong';
                        }
                        if (param.name.toLowerCase().includes('variant')) {
                            stateProps[param.name] = 'error';
                        }
                    });
                    break;
                    
                case 'disabled':
                    component.parameters.forEach(param => {
                        if (param.name.toLowerCase().includes('disabled')) {
                            stateProps[param.name] = true;
                        }
                    });
                    break;
                    
                case 'success':
                    component.parameters.forEach(param => {
                        if (param.name.toLowerCase().includes('success')) {
                            stateProps[param.name] = true;
                        }
                        if (param.name.toLowerCase().includes('variant')) {
                            stateProps[param.name] = 'success';
                        }
                    });
                    break;
            }
            
            return stateProps;
        }
        
        function updateComponentStateIndicator(card, componentName, state) {
            // Remove existing state indicator
            const existingIndicator = card.querySelector('.state-indicator');
            if (existingIndicator) {
                existingIndicator.remove();
            }
            
            // Add new state indicator if not normal
            if (state !== 'normal') {
                const indicator = document.createElement('div');
                indicator.className = 'state-indicator ' + state;
                indicator.textContent = state;
                
                const cardHeader = card.querySelector('.card-header');
                cardHeader.style.position = 'relative';
                cardHeader.appendChild(indicator);
            }
        }
        
        // Load saved prop combinations from localStorage
        function loadSavedPropCombinations() {
            try {
                const saved = localStorage.getItem('templar_prop_combinations');
                if (saved) {
                    propCombinations = JSON.parse(saved);
                    
                    // Update all dropdowns
                    Object.keys(components).forEach(componentName => {
                        updatePropCombinationOptions(componentName);
                    });
                }
            } catch (error) {
                console.warn('Failed to load saved prop combinations:', error);
            }
        }
    </script>
</body>
</html>`;
}