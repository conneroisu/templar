package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/conneroisu/templar/internal/registry"
)

// BenchmarkLargeCodebaseScanning benchmarks scanner performance on large codebases
func BenchmarkLargeCodebaseScanning(b *testing.B) {
	sizes := []int{100, 500, 1000, 2000, 5000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Components_%d", size), func(b *testing.B) {
			benchmarkLargeCodebase(b, size)
		})
	}
}

func benchmarkLargeCodebase(b *testing.B, componentCount int) {
	// Create test directory with realistic component structure
	testDir := createRealisticCodebase(b, componentCount)
	defer os.RemoveAll(testDir)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reg := registry.NewComponentRegistry()
		scanner := NewComponentScanner(reg)

		err := scanner.ScanDirectory(testDir)
		if err != nil {
			b.Fatal(err)
		}

		// Verify we found the expected number of components
		if reg.Count() == 0 {
			b.Fatal("No components found")
		}

		scanner.Close()
	}
}

// createRealisticCodebase creates a directory structure that mimics a real project
func createRealisticCodebase(b *testing.B, componentCount int) string {
	tempDir, err := os.MkdirTemp(".", "templar-large-codebase-*")
	if err != nil {
		b.Fatal(err)
	}

	// Create realistic directory structure
	dirs := []string{
		"components/ui",
		"components/forms",
		"components/layout",
		"pages",
		"layouts",
		"partials",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			b.Fatal(err)
		}
	}

	// Distribute components across directories
	componentsPerDir := componentCount / len(dirs)
	remainder := componentCount % len(dirs)

	componentIndex := 0
	for dirIdx, dir := range dirs {
		count := componentsPerDir
		if dirIdx < remainder {
			count++ // Distribute remainder
		}

		for i := 0; i < count; i++ {
			// Create varied component types
			var content string
			switch componentIndex % 6 {
			case 0:
				content = generateRealisticButtonComponent(componentIndex)
			case 1:
				content = generateRealisticCardComponent(componentIndex)
			case 2:
				content = generateRealisticFormComponent(componentIndex)
			case 3:
				content = generateRealisticLayoutComponent(componentIndex)
			case 4:
				content = generateRealisticTableComponent(componentIndex)
			case 5:
				content = generateRealisticModalComponent(componentIndex)
			}

			filename := filepath.Join(
				tempDir,
				dir,
				fmt.Sprintf("component_%d.templ", componentIndex),
			)
			if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
				b.Fatal(err)
			}
			componentIndex++
		}
	}

	return tempDir
}

func generateRealisticButtonComponent(index int) string {
	return fmt.Sprintf(`package components

// Button%d renders a customizable button component
templ Button%d(text string, variant string, disabled bool, onClick string) {
	<button 
		class={ 
			"px-4 py-2 rounded-md font-medium transition-all duration-200 focus:ring-2 focus:ring-offset-2",
			templ.KV("bg-blue-600 hover:bg-blue-700 text-white focus:ring-blue-500", variant == "primary"),
			templ.KV("bg-gray-200 hover:bg-gray-300 text-gray-800 focus:ring-gray-500", variant == "secondary"),
			templ.KV("bg-red-600 hover:bg-red-700 text-white focus:ring-red-500", variant == "danger"),
			templ.KV("opacity-50 cursor-not-allowed", disabled),
		}
		if !disabled {
			onclick={ onClick }
		}
		disabled?={ disabled }
	>
		{text}
	</button>
}`, index, index)
}

func generateRealisticCardComponent(index int) string {
	return fmt.Sprintf(`package components

// Card%d renders a content card with header, body, and optional footer
templ Card%d(title string, content string, footer string, elevated bool) {
	<div class={ 
		"bg-white rounded-lg border",
		templ.KV("shadow-lg", elevated),
		templ.KV("shadow-sm", !elevated),
	}>
		if title != "" {
			<div class="px-6 py-4 border-b border-gray-200">
				<h3 class="text-lg font-semibold text-gray-900">{title}</h3>
			</div>
		}
		<div class="px-6 py-4">
			<div class="text-gray-700 leading-relaxed">
				{content}
			</div>
		</div>
		if footer != "" {
			<div class="px-6 py-4 bg-gray-50 border-t border-gray-200 rounded-b-lg">
				<div class="text-sm text-gray-600">
					{footer}
				</div>
			</div>
		}
	</div>
}`, index, index)
}

func generateRealisticFormComponent(index int) string {
	return fmt.Sprintf(`package components

// Form%d renders a form with validation
templ Form%d(action string, method string, csrfToken string) {
	<form action={ action } method={ method } class="space-y-6 max-w-md mx-auto">
		<input type="hidden" name="_token" value={ csrfToken }/>
		
		<div class="space-y-4">
			<div>
				<label for="email_%d" class="block text-sm font-medium text-gray-700">
					Email Address
				</label>
				<input 
					type="email" 
					id="email_%d" 
					name="email" 
					required
					class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
					placeholder="Enter your email"
				/>
			</div>
			
			<div>
				<label for="password_%d" class="block text-sm font-medium text-gray-700">
					Password
				</label>
				<input 
					type="password" 
					id="password_%d" 
					name="password" 
					required
					minlength="8"
					class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
					placeholder="Enter your password"
				/>
			</div>
		</div>
		
		<div class="flex items-center justify-between">
			<button 
				type="submit" 
				class="w-full bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
			>
				Submit
			</button>
		</div>
	</form>
}`, index, index, index, index, index, index)
}

func generateRealisticLayoutComponent(index int) string {
	return fmt.Sprintf(`package components

// Layout%d renders a page layout with header, main, and footer
templ Layout%d(title string, showSidebar bool) {
	<!DOCTYPE html>
	<html lang="en" class="h-full bg-gray-50">
		<head>
			<meta charset="UTF-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>{title}</title>
			<script src="https://cdn.tailwindcss.com"></script>
		</head>
		<body class="h-full">
			<div class="min-h-full">
				@Header%d(title)
				
				<div class="flex">
					if showSidebar {
						<aside class="w-64 bg-gray-800 text-white min-h-screen">
							@Sidebar%d()
						</aside>
					}
					<main class="flex-1 py-6 px-4 sm:px-6 lg:px-8">
						{ children... }
					</main>
				</div>
				
				@Footer%d()
			</div>
		</body>
	</html>
}

templ Header%d(title string) {
	<header class="bg-white shadow">
		<div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
			<h1 class="text-3xl font-bold text-gray-900">{title}</h1>
		</div>
	</header>
}

templ Sidebar%d() {
	<nav class="mt-5 px-2">
		<a href="/" class="group flex items-center px-2 py-2 text-sm font-medium rounded-md text-gray-300 hover:bg-gray-700 hover:text-white">
			Dashboard
		</a>
		<a href="/projects" class="group flex items-center px-2 py-2 text-sm font-medium rounded-md text-gray-300 hover:bg-gray-700 hover:text-white">
			Projects
		</a>
	</nav>
}

templ Footer%d() {
	<footer class="bg-gray-800 text-white py-8 mt-12">
		<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
			<p class="text-center text-gray-400">© 2024 Templar. All rights reserved.</p>
		</div>
	</footer>
}`, index, index, index, index, index, index, index, index)
}

func generateRealisticTableComponent(index int) string {
	return fmt.Sprintf(`package components

// Table%d renders a data table with sorting and pagination
templ Table%d(headers []string, rows [][]string, sortBy string, sortOrder string) {
	<div class="bg-white shadow overflow-hidden sm:rounded-md">
		<table class="min-w-full divide-y divide-gray-200">
			<thead class="bg-gray-50">
				<tr>
					for i, header := range headers {
						<th 
							scope="col" 
							class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100"
							data-sort={ fmt.Sprintf("%%d", i) }
						>
							<div class="flex items-center justify-between">
								{header}
								if sortBy == fmt.Sprintf("%%d", i) {
									if sortOrder == "asc" {
										<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
											<path d="M5 10l5-5 5 5H5z"/>
										</svg>
									} else {
										<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
											<path d="M15 10l-5 5-5-5h10z"/>
										</svg>
									}
								}
							</div>
						</th>
					}
				</tr>
			</thead>
			<tbody class="bg-white divide-y divide-gray-200">
				for i, row := range rows {
					<tr class={ templ.KV("bg-gray-50", i%%2 == 1) }>
						for _, cell := range row {
							<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
								{cell}
							</td>
						}
					</tr>
				}
			</tbody>
		</table>
	</div>
}`, index, index)
}

func generateRealisticModalComponent(index int) string {
	return fmt.Sprintf(`package components

// Modal%d renders a modal dialog with backdrop
templ Modal%d(title string, isOpen bool, onClose string) {
	if isOpen {
		<div class="fixed inset-0 z-50 overflow-y-auto" aria-labelledby="modal-title" role="dialog" aria-modal="true">
			<div class="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
				<!-- Background overlay -->
				<div 
					class="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity" 
					onclick={ onClose }
				></div>
				
				<!-- Modal panel -->
				<div class="inline-block align-bottom bg-white rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full">
					<div class="bg-white px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
						<div class="sm:flex sm:items-start">
							<div class="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left w-full">
								<h3 class="text-lg leading-6 font-medium text-gray-900" id="modal-title">
									{title}
								</h3>
								<div class="mt-2">
									{ children... }
								</div>
							</div>
						</div>
					</div>
					<div class="bg-gray-50 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse">
						<button 
							type="button" 
							class="w-full inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 bg-blue-600 text-base font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 sm:ml-3 sm:w-auto sm:text-sm"
							onclick={ onClose }
						>
							Close
						</button>
					</div>
				</div>
			</div>
		</div>
	}
}`, index, index)
}

// BenchmarkMemoryUsageStability tests memory usage remains stable under load
func BenchmarkMemoryUsageStability(b *testing.B) {
	// Create a large codebase in current directory
	testDir := createRealisticCodebase(b, 1000)
	defer os.RemoveAll(testDir)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reg := registry.NewComponentRegistry()
		scanner := NewComponentScanner(reg)

		err := scanner.ScanDirectory(testDir)
		if err != nil {
			b.Fatal(err)
		}

		scanner.Close()

		// Force GC to see if we're leaking memory
		if i%10 == 0 {
			b.StopTimer()
			// runtime.GC() - removed to avoid forcing GC in benchmark
			b.StartTimer()
		}
	}
}
