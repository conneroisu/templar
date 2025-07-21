package server

import (
	"fmt"
	"strings"

	"github.com/conneroisu/templar/internal/types"
)

// wrapInPlaygroundLayout wraps component HTML in the interactive playground layout
func (s *PreviewServer) wrapInPlaygroundLayout(componentName, html, theme string, viewport ViewportSize) string {
	if viewport.Width == 0 {
		viewport.Width = 1200
	}
	if viewport.Height == 0 {
		viewport.Height = 800
	}
	if viewport.Name == "" {
		viewport.Name = "Desktop"
	}

	themeClass := "theme-light"
	if theme == "dark" {
		themeClass = "theme-dark"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html class="%s">
<head>
    <title>%s - Interactive Playground</title>
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
        .theme-light {
            --bg-primary: #ffffff;
            --bg-secondary: #f8fafc;
            --text-primary: #1e293b;
            --text-secondary: #64748b;
            --border: #e2e8f0;
            --accent: #3b82f6;
        }
        .theme-dark {
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
            height: 100vh;
            overflow: hidden;
        }
        
        .playground-container {
            display: grid;
            grid-template-columns: 400px 1fr;
            height: 100vh;
        }
        
        .sidebar {
            background-color: var(--bg-primary);
            border-right: 1px solid var(--border);
            overflow-y: auto;
            padding: 20px;
        }
        
        .main-content {
            display: flex;
            flex-direction: column;
            overflow: hidden;
        }
        
        .viewport-controls {
            background-color: var(--bg-primary);
            border-bottom: 1px solid var(--border);
            padding: 15px 20px;
            display: flex;
            align-items: center;
            gap: 15px;
        }
        
        .component-frame {
            flex: 1;
            background: var(--bg-secondary);
            overflow: auto;
            padding: 20px;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        
        .component-container {
            background: var(--bg-primary);
            border-radius: 8px;
            padding: 40px;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
            min-width: %dpx;
            max-width: %dpx;
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
        }
        
        .prop-input:focus {
            outline: none;
            border-color: var(--accent);
            box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
        }
        
        .prop-select {
            width: 100%%;
            padding: 8px 12px;
            border: 1px solid var(--border);
            border-radius: 4px;
            background: var(--bg-primary);
            color: var(--text-primary);
            font-size: 14px;
        }
        
        .prop-checkbox {
            width: 18px;
            height: 18px;
            margin-right: 8px;
            accent-color: var(--accent);
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
        
        .code-output {
            background: #1e293b;
            color: #e2e8f0;
            padding: 16px;
            border-radius: 6px;
            font-family: 'Monaco', 'Menlo', monospace;
            font-size: 13px;
            line-height: 1.5;
            white-space: pre-wrap;
            overflow-x: auto;
        }
        
        .viewport-preset {
            display: flex;
            align-items: center;
            gap: 8px;
            padding: 8px 12px;
            border: 1px solid var(--border);
            border-radius: 4px;
            background: var(--bg-primary);
            cursor: pointer;
            font-size: 14px;
        }
        
        .viewport-preset.active {
            border-color: var(--accent);
            background: rgba(59, 130, 246, 0.1);
        }
        
        .action-button {
            background: var(--accent);
            color: white;
            border: none;
            padding: 10px 16px;
            border-radius: 6px;
            font-weight: 500;
            cursor: pointer;
            transition: opacity 0.2s;
        }
        
        .action-button:hover {
            opacity: 0.9;
        }
        
        .action-button:disabled {
            opacity: 0.5;
            cursor: not-allowed;
        }
        
        .section-title {
            font-size: 18px;
            font-weight: 700;
            margin: 0 0 16px 0;
            color: var(--text-primary);
        }
        
        .subsection-title {
            font-size: 14px;
            font-weight: 600;
            margin: 20px 0 12px 0;
            color: var(--text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }
        
        .metadata-item {
            display: flex;
            justify-content: space-between;
            padding: 8px 0;
            border-bottom: 1px solid var(--border);
        }
        
        .metadata-item:last-child {
            border-bottom: none;
        }
        
        .metadata-label {
            font-weight: 500;
            color: var(--text-secondary);
        }
        
        .metadata-value {
            color: var(--text-primary);
            word-break: break-all;
        }
        
        @media (max-width: 1024px) {
            .playground-container {
                grid-template-columns: 1fr;
            }
            
            .sidebar {
                position: fixed;
                top: 0;
                left: -400px;
                width: 400px;
                height: 100vh;
                z-index: 1000;
                transition: left 0.3s;
            }
            
            .sidebar.open {
                left: 0;
            }
        }
    </style>
</head>
<body>
    <div class="playground-container">
        <div class="sidebar" id="sidebar">
            <h1 class="section-title">%s Playground</h1>
            
            <div class="prop-editor" id="propEditor">
                <!-- Props will be dynamically populated -->
            </div>
            
            <div class="subsection-title">Generated Code</div>
            <div class="code-output" id="generatedCode">
                Loading...
            </div>
            
            <div class="subsection-title">Component Info</div>
            <div id="componentMetadata">
                <!-- Metadata will be populated -->
            </div>
        </div>
        
        <div class="main-content">
            <div class="viewport-controls">
                <select id="themeSelect" class="prop-select" style="width: auto;">
                    <option value="light">Light Theme</option>
                    <option value="dark">Dark Theme</option>
                </select>
                
                <div class="viewport-preset" data-preset="mobile">
                    📱 Mobile (375×667)
                </div>
                <div class="viewport-preset" data-preset="tablet">
                    📟 Tablet (768×1024)
                </div>
                <div class="viewport-preset active" data-preset="desktop">
                    🖥️ Desktop (1200×800)
                </div>
                
                <button class="action-button" onclick="refreshComponent()">
                    🔄 Refresh
                </button>
            </div>
            
            <div class="component-frame" id="componentFrame">
                <div class="component-container" id="componentContainer">
                    %s
                </div>
            </div>
        </div>
    </div>
    
    <script>
        // WebSocket connection for live reload
        const ws = new WebSocket('ws://localhost:' + window.location.port + '/ws');
        
        ws.onmessage = function(event) {
            const message = JSON.parse(event.data);
            if (message.type === 'component_update') {
                refreshComponent();
            }
        };
        
        // Playground state
        let currentProps = {};
        let componentName = '%s';
        let currentTheme = '%s';
        let currentViewport = { width: %d, height: %d, name: '%s' };
        
        // Initialize playground
        document.addEventListener('DOMContentLoaded', function() {
            initializePlayground();
        });
        
        function initializePlayground() {
            loadComponentData();
            setupEventListeners();
        }
        
        function setupEventListeners() {
            // Theme selector
            const themeSelect = document.getElementById('themeSelect');
            themeSelect.value = currentTheme;
            themeSelect.addEventListener('change', function() {
                currentTheme = this.value;
                updateTheme();
                refreshComponent();
            });
            
            // Viewport presets
            document.querySelectorAll('.viewport-preset').forEach(preset => {
                preset.addEventListener('click', function() {
                    document.querySelectorAll('.viewport-preset').forEach(p => p.classList.remove('active'));
                    this.classList.add('active');
                    
                    const presetName = this.dataset.preset;
                    switch(presetName) {
                        case 'mobile':
                            currentViewport = { width: 375, height: 667, name: 'Mobile' };
                            break;
                        case 'tablet':
                            currentViewport = { width: 768, height: 1024, name: 'Tablet' };
                            break;
                        case 'desktop':
                            currentViewport = { width: 1200, height: 800, name: 'Desktop' };
                            break;
                    }
                    
                    updateViewport();
                    refreshComponent();
                });
            });
        }
        
        function updateTheme() {
            document.documentElement.className = 'theme-' + currentTheme;
        }
        
        function updateViewport() {
            const container = document.getElementById('componentContainer');
            if (container) {
                container.style.minWidth = currentViewport.width + 'px';
                container.style.maxWidth = currentViewport.width + 'px';
            }
        }
        
        async function loadComponentData() {
            try {
                const response = await fetch('/api/playground/render', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        component_name: componentName,
                        props: currentProps,
                        theme: currentTheme,
                        viewport_size: currentViewport,
                        mock_data: true,
                        generate_code: true
                    })
                });
                
                const data = await response.json();
                
                if (data.error) {
                    showError(data.error);
                    return;
                }
                
                updatePropEditor(data.available_props, data.current_props);
                updateGeneratedCode(data.generated_code);
                updateMetadata(data.metadata);
                currentProps = data.current_props;
                
            } catch (error) {
                showError('Failed to load component data: ' + error.message);
            }
        }
        
        function updatePropEditor(availableProps, currentProps) {
            const editor = document.getElementById('propEditor');
            editor.innerHTML = '';
            
            availableProps.forEach(prop => {
                const propGroup = createPropEditor(prop, currentProps[prop.name]);
                editor.appendChild(propGroup);
            });
        }
        
        function createPropEditor(prop, currentValue) {
            const group = document.createElement('div');
            group.className = 'prop-group';
            
            const label = document.createElement('div');
            label.className = 'prop-label';
            
            const labelText = document.createElement('span');
            labelText.textContent = prop.name;
            label.appendChild(labelText);
            
            const typeBadge = document.createElement('span');
            typeBadge.className = 'type-badge';
            typeBadge.textContent = prop.type;
            label.appendChild(typeBadge);
            
            if (prop.required) {
                const requiredBadge = document.createElement('span');
                requiredBadge.className = 'required-badge';
                requiredBadge.textContent = 'required';
                label.appendChild(requiredBadge);
            }
            
            group.appendChild(label);
            
            const input = createInputForType(prop, currentValue);
            input.addEventListener('change', function() {
                updateProp(prop.name, this.value, prop.type);
            });
            
            group.appendChild(input);
            
            if (prop.description) {
                const desc = document.createElement('div');
                desc.style.fontSize = '12px';
                desc.style.color = 'var(--text-secondary)';
                desc.style.marginTop = '4px';
                desc.textContent = prop.description;
                group.appendChild(desc);
            }
            
            return group;
        }
        
        function createInputForType(prop, currentValue) {
            switch (prop.type) {
                case 'bool':
                    const checkbox = document.createElement('input');
                    checkbox.type = 'checkbox';
                    checkbox.className = 'prop-checkbox';
                    checkbox.checked = currentValue || false;
                    return checkbox;
                    
                case 'int':
                case 'int64':
                case 'int32':
                    const numberInput = document.createElement('input');
                    numberInput.type = 'number';
                    numberInput.className = 'prop-input';
                    numberInput.value = currentValue || 0;
                    return numberInput;
                    
                case 'string':
                default:
                    const textInput = document.createElement('input');
                    textInput.type = 'text';
                    textInput.className = 'prop-input';
                    textInput.value = currentValue || '';
                    return textInput;
            }
        }
        
        function updateProp(propName, value, type) {
            // Convert value to appropriate type
            switch (type) {
                case 'bool':
                    currentProps[propName] = Boolean(value);
                    break;
                case 'int':
                case 'int64':
                case 'int32':
                    currentProps[propName] = parseInt(value, 10) || 0;
                    break;
                case 'float64':
                case 'float32':
                    currentProps[propName] = parseFloat(value) || 0.0;
                    break;
                default:
                    currentProps[propName] = value;
            }
            
            // Debounce refresh
            clearTimeout(window.refreshTimeout);
            window.refreshTimeout = setTimeout(refreshComponent, 500);
        }
        
        async function refreshComponent() {
            try {
                const response = await fetch('/api/playground/render', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        component_name: componentName,
                        props: currentProps,
                        theme: currentTheme,
                        viewport_size: currentViewport,
                        mock_data: false,
                        generate_code: true
                    })
                });
                
                const data = await response.json();
                
                if (data.error) {
                    showError(data.error);
                    return;
                }
                
                updateComponentHTML(data.html);
                updateGeneratedCode(data.generated_code);
                
            } catch (error) {
                showError('Failed to refresh component: ' + error.message);
            }
        }
        
        function updateComponentHTML(html) {
            const container = document.getElementById('componentContainer');
            // Extract just the component HTML from the full page HTML
            const parser = new DOMParser();
            const doc = parser.parseFromString(html, 'text/html');
            const componentContent = doc.querySelector('.component-container');
            if (componentContent) {
                container.innerHTML = componentContent.innerHTML;
            }
        }
        
        function updateGeneratedCode(code) {
            const codeOutput = document.getElementById('generatedCode');
            codeOutput.textContent = code || 'No code generated';
        }
        
        function updateMetadata(metadata) {
            const metadataContainer = document.getElementById('componentMetadata');
            metadataContainer.innerHTML = '';
            
            if (!metadata) return;
            
            const items = [
                { label: 'Package', value: metadata.package },
                { label: 'File Path', value: metadata.file_path },
                { label: 'Last Modified', value: metadata.last_modified },
                { label: 'Dependencies', value: metadata.dependencies.join(', ') || 'None' }
            ];
            
            items.forEach(item => {
                if (item.value) {
                    const metaItem = document.createElement('div');
                    metaItem.className = 'metadata-item';
                    
                    const label = document.createElement('div');
                    label.className = 'metadata-label';
                    label.textContent = item.label;
                    
                    const value = document.createElement('div');
                    value.className = 'metadata-value';
                    value.textContent = item.value;
                    
                    metaItem.appendChild(label);
                    metaItem.appendChild(value);
                    metadataContainer.appendChild(metaItem);
                }
            });
        }
        
        function showError(message) {
            const container = document.getElementById('componentContainer');
            container.innerHTML = '<div style="color: #ef4444; padding: 20px; text-align: center;">' + 
                                '<h3>Error</h3><p>' + message + '</p></div>';
        }
        
        // Mobile sidebar toggle
        function toggleSidebar() {
            const sidebar = document.getElementById('sidebar');
            sidebar.classList.toggle('open');
        }
        
        // Add mobile menu button for small screens
        if (window.innerWidth <= 1024) {
            const menuButton = document.createElement('button');
            menuButton.textContent = '☰ Props';
            menuButton.className = 'action-button';
            menuButton.onclick = toggleSidebar;
            
            const controls = document.querySelector('.viewport-controls');
            controls.insertBefore(menuButton, controls.firstChild);
        }
    </script>
</body>
</html>`, themeClass, componentName, viewport.Width-100, viewport.Width, componentName, html, componentName, theme, viewport.Width, viewport.Height, viewport.Name)
}

// generatePlaygroundHTML creates the main playground interface for a component
func (s *PreviewServer) generatePlaygroundHTML(component *types.ComponentInfo) string {
	// Generate initial mock data
	mockData := s.generateIntelligentMockData(component)
	
	// Render component with mock data
	html, err := s.renderComponentWithProps(component.Name, mockData)
	if err != nil {
		html = fmt.Sprintf(`<div class="error">Error rendering component: %s</div>`, err.Error())
	}
	
	// Wrap in playground layout
	viewport := ViewportSize{Width: 1200, Height: 800, Name: "Desktop"}
	return s.wrapInPlaygroundLayout(component.Name, html, "light", viewport)
}

// generatePlaygroundIndexHTML creates the index page showing all components
func (s *PreviewServer) generatePlaygroundIndexHTML(components []*types.ComponentInfo) string {
	var componentCards strings.Builder
	
	for _, component := range components {
		componentCards.WriteString(fmt.Sprintf(`
			<div class="component-card" onclick="window.location.href='/playground/%s'">
				<h3 class="component-name">%s</h3>
				<p class="component-package">Package: %s</p>
				<p class="component-params">%d parameters</p>
				<div class="component-preview">
					<span class="preview-badge">Click to Preview</span>
				</div>
			</div>
		`, component.Name, component.Name, component.Package, len(component.Parameters)))
	}
	
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Templar Component Playground</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <script src="https://cdn.tailwindcss.com"></script>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            margin: 0;
            padding: 20px;
            min-height: 100vh;
        }
        
        .header {
            text-align: center;
            color: white;
            margin-bottom: 40px;
        }
        
        .title {
            font-size: 3rem;
            font-weight: 800;
            margin-bottom: 10px;
        }
        
        .subtitle {
            font-size: 1.25rem;
            opacity: 0.9;
        }
        
        .components-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
            gap: 20px;
            max-width: 1200px;
            margin: 0 auto;
        }
        
        .component-card {
            background: white;
            border-radius: 12px;
            padding: 24px;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
            cursor: pointer;
            transition: all 0.2s;
        }
        
        .component-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
        }
        
        .component-name {
            font-size: 1.5rem;
            font-weight: 700;
            color: #1e293b;
            margin: 0 0 8px 0;
        }
        
        .component-package {
            color: #64748b;
            margin: 0 0 4px 0;
            font-size: 0.875rem;
        }
        
        .component-params {
            color: #64748b;
            margin: 0 0 16px 0;
            font-size: 0.875rem;
        }
        
        .component-preview {
            display: flex;
            justify-content: center;
        }
        
        .preview-badge {
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            padding: 8px 16px;
            border-radius: 20px;
            font-size: 0.875rem;
            font-weight: 500;
        }
        
        .stats {
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(10px);
            border-radius: 12px;
            padding: 20px;
            text-align: center;
            color: white;
            margin-bottom: 30px;
            max-width: 400px;
            margin-left: auto;
            margin-right: auto;
        }
        
        .stats-number {
            font-size: 2rem;
            font-weight: 800;
            margin-bottom: 5px;
        }
        
        .stats-label {
            font-size: 0.875rem;
            opacity: 0.9;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1 class="title">🎮 Component Playground</h1>
        <p class="subtitle">Interactive preview and testing for your Templ components</p>
    </div>
    
    <div class="stats">
        <div class="stats-number">%d</div>
        <div class="stats-label">Components Available</div>
    </div>
    
    <div class="components-grid">
        %s
    </div>
    
    <script>
        // Add some interactivity
        document.querySelectorAll('.component-card').forEach(card => {
            card.addEventListener('mouseenter', function() {
                this.style.background = 'linear-gradient(135deg, #f8fafc 0%%, #f1f5f9 100%%)';
            });
            
            card.addEventListener('mouseleave', function() {
                this.style.background = 'white';
            });
        });
    </script>
</body>
</html>`, len(components), componentCards.String())
}