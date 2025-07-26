package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/types"
)

// createTestComponents creates a directory with test component files.
func createTestComponents(count int) string {
	tempDir := fmt.Sprintf("scanner_bench_%d_%d", count, time.Now().UnixNano())
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		panic(err)
	}

	for i := range count {
		// Create different types of components for variety
		var content string
		switch i % 4 {
		case 0:
			content = generateSimpleComponent(i)
		case 1:
			content = generateMediumComponent(i)
		case 2:
			content = generateComplexComponent(i)
		case 3:
			content = generateNestedComponent(i)
		}

		filename := filepath.Join(tempDir, fmt.Sprintf("component_%d.templ", i))
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			panic(err)
		}
	}

	return tempDir
}

func generateSimpleComponent(index int) string {
	return fmt.Sprintf(`package components

templ Button%d(text string) {
	<button class="btn">{text}</button>
}
`, index)
}

func generateMediumComponent(index int) string {
	return fmt.Sprintf(`package components

import "fmt"

templ Card%d(title string, content string, actions []string) {
	<div class="card">
		<div class="card-header">
			<h3>{title}</h3>
		</div>
		<div class="card-body">
			<p>{content}</p>
		</div>
		<div class="card-footer">
			for _, action := range actions {
				<button class="btn">{action}</button>
			}
		</div>
	</div>
}

templ CardSimple%d(title string) {
	<div class="simple-card">{title}</div>
}
`, index, index)
}

func generateComplexComponent(index int) string {
	return fmt.Sprintf(`package components

import (
	"fmt"
	"strings"
	"time"
)

type User%d struct {
	ID   int
	Name string
	Role string
}

templ DataTable%d(users []User%d, sortBy string, ascending bool, pageSize int, currentPage int) {
	<div class="data-table">
		<div class="table-header">
			<div class="table-controls">
				<select name="pageSize">
					<option value="10">10</option>
					<option value="25">25</option>
					<option value="50">50</option>
				</select>
				<input type="search" placeholder="Search users..." />
			</div>
		</div>
		<table class="table">
			<thead>
				<tr>
					<th 
						class={ "sortable", templ.KV("ascending", sortBy == "id" && ascending), templ.KV("descending", sortBy == "id" && !ascending) }
						data-sort="id"
					>
						ID
					</th>
					<th 
						class={ "sortable", templ.KV("ascending", sortBy == "name" && ascending), templ.KV("descending", sortBy == "name" && !ascending) }
						data-sort="name"
					>
						Name
					</th>
					<th 
						class={ "sortable", templ.KV("ascending", sortBy == "role" && ascending), templ.KV("descending", sortBy == "role" && !ascending) }
						data-sort="role"
					>
						Role
					</th>
					<th>Actions</th>
				</tr>
			</thead>
			<tbody>
				for i, user := range users {
					if i >= currentPage * pageSize && i < (currentPage + 1) * pageSize {
						<tr class={ templ.KV("even", i%%2 == 0), templ.KV("odd", i%%2 == 1) }>
							<td>{fmt.Sprintf("%%d", user.ID)}</td>
							<td>
								<div class="user-info">
									<span class="user-name">{user.Name}</span>
									if strings.Contains(user.Role, "admin") {
										<span class="badge badge-admin">Admin</span>
									}
								</div>
							</td>
							<td>
								<span class={ "role", fmt.Sprintf("role-%%s", strings.ToLower(user.Role)) }>
									{user.Role}
								</span>
							</td>
							<td>
								<div class="action-buttons">
									<button class="btn btn-sm btn-primary" data-action="edit" data-id={fmt.Sprintf("%%d", user.ID)}>
										Edit
									</button>
									<button class="btn btn-sm btn-danger" data-action="delete" data-id={fmt.Sprintf("%%d", user.ID)}>
										Delete
									</button>
								</div>
							</td>
						</tr>
					}
				}
			</tbody>
		</table>
		<div class="table-footer">
			<div class="pagination">
				if currentPage > 0 {
					<button class="btn btn-outline" data-page={fmt.Sprintf("%%d", currentPage-1)}>
						Previous
					</button>
				}
				<span class="page-info">
					Page {fmt.Sprintf("%%d", currentPage+1)} of {fmt.Sprintf("%%d", (len(users)+pageSize-1)/pageSize)}
				</span>
				if (currentPage+1)*pageSize < len(users) {
					<button class="btn btn-outline" data-page={fmt.Sprintf("%%d", currentPage+1)}>
						Next
					</button>
				}
			</div>
		</div>
	</div>
}
`, index, index, index)
}

func generateNestedComponent(index int) string {
	return fmt.Sprintf(`package components

templ Layout%d(title string, sidebar bool) {
	<!DOCTYPE html>
	<html>
		<head>
			<title>{title}</title>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1"/>
		</head>
		<body>
			@Header%d(title)
			<main class="main-content">
				if sidebar {
					<div class="layout-with-sidebar">
						<aside class="sidebar">
							@Sidebar%d()
						</aside>
						<div class="content">
							{ children... }
						</div>
					</div>
				} else {
					<div class="content-full">
						{ children... }
					</div>
				}
			</main>
			@Footer%d()
		</body>
	</html>
}

templ Header%d(title string) {
	<header class="header">
		<nav class="navbar">
			<div class="navbar-brand">
				<a href="/">{title}</a>
			</div>
			<div class="navbar-nav">
				<a href="/dashboard">Dashboard</a>
				<a href="/projects">Projects</a>
				<a href="/settings">Settings</a>
			</div>
		</nav>
	</header>
}

templ Sidebar%d() {
	<nav class="sidebar-nav">
		<ul class="nav-list">
			<li><a href="/dashboard" class="nav-link">Dashboard</a></li>
			<li><a href="/projects" class="nav-link">Projects</a></li>
			<li class="nav-group">
				<span class="nav-group-title">Components</span>
				<ul class="nav-sublist">
					<li><a href="/components/buttons" class="nav-link">Buttons</a></li>
					<li><a href="/components/forms" class="nav-link">Forms</a></li>
					<li><a href="/components/cards" class="nav-link">Cards</a></li>
				</ul>
			</li>
		</ul>
	</nav>
}

templ Footer%d() {
	<footer class="footer">
		<div class="footer-content">
			<p>&copy; 2024 Templar Framework. All rights reserved.</p>
		</div>
	</footer>
}
`, index, index, index, index, index, index, index)
}

// BenchmarkComponentScanner_ScanDirectory benchmarks directory scanning performance.
func BenchmarkComponentScanner_ScanDirectory(b *testing.B) {
	componentCounts := []int{10, 50, 100, 500, 1000}

	for _, count := range componentCounts {
		b.Run(fmt.Sprintf("components-%d", count), func(b *testing.B) {
			testDir := createTestComponents(count)
			defer os.RemoveAll(testDir)

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				reg := registry.NewComponentRegistry()
				scanner := NewComponentScanner(reg)
				err := scanner.ScanDirectory(testDir)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkComponentScanner_ScanFile benchmarks single file scanning performance.
func BenchmarkComponentScanner_ScanFile(b *testing.B) {
	complexities := []struct {
		name      string
		generator func(int) string
	}{
		{"simple", generateSimpleComponent},
		{"medium", generateMediumComponent},
		{"complex", generateComplexComponent},
		{"nested", generateNestedComponent},
	}

	for _, complexity := range complexities {
		b.Run(complexity.name, func(b *testing.B) {
			// Create a test file
			content := complexity.generator(0)
			tempFile, err := os.CreateTemp("", "component_*.templ")
			if err != nil {
				b.Fatal(err)
			}
			defer os.Remove(tempFile.Name())

			if _, err := tempFile.WriteString(content); err != nil {
				b.Fatal(err)
			}
			tempFile.Close()

			reg := registry.NewComponentRegistry()
			scanner := NewComponentScanner(reg)

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				err := scanner.ScanFile(tempFile.Name())
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkExtractParameters benchmarks parameter extraction performance.
func BenchmarkExtractParameters(b *testing.B) {
	testLines := []string{
		"templ Button(text string) {",
		"templ Card(title string, content string, active bool) {",
		"templ DataTable(users []User, sortBy string, ascending bool, pageSize int, currentPage int) {",
		"templ ComplexComponent(id int, name string, tags []string, meta map[string]interface{}, opts ...Option) {",
		"templ VeryComplexComponent(a string, b int, c bool, d []string, e map[string]int, f func(string) bool, g chan int, h interface{}) {",
	}

	for i, line := range testLines {
		b.Run(fmt.Sprintf("params-%d", i+1), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_ = extractParameters(line)
			}
		})
	}
}

// BenchmarkComponentScanner_MemoryUsage benchmarks memory usage patterns.
func BenchmarkComponentScanner_MemoryUsage(b *testing.B) {
	b.Run("SmallCodebase", func(b *testing.B) {
		benchmarkScannerMemoryUsage(b, 50)
	})

	b.Run("MediumCodebase", func(b *testing.B) {
		benchmarkScannerMemoryUsage(b, 200)
	})

	b.Run("LargeCodebase", func(b *testing.B) {
		benchmarkScannerMemoryUsage(b, 1000)
	})
}

func benchmarkScannerMemoryUsage(b *testing.B, componentCount int) {
	testDir := createTestComponents(componentCount)
	defer os.RemoveAll(testDir)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		reg := registry.NewComponentRegistry()
		scanner := NewComponentScanner(reg)

		err := scanner.ScanDirectory(testDir)
		if err != nil {
			b.Fatal(err)
		}

		// Verify we found components
		if reg.Count() == 0 {
			b.Fatal("No components found")
		}
	}
}

// BenchmarkComponentScanner_ConcurrentScanning benchmarks concurrent scanning performance.
func BenchmarkComponentScanner_ConcurrentScanning(b *testing.B) {
	testDir := createTestComponents(100)
	defer os.RemoveAll(testDir)

	// Get list of component files
	files, err := filepath.Glob(filepath.Join(testDir, "*.templ"))
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		reg := registry.NewComponentRegistry()
		scanner := NewComponentScanner(reg)
		fileIndex := 0

		for pb.Next() {
			file := files[fileIndex%len(files)]
			err := scanner.ScanFile(file)
			if err != nil {
				b.Fatal(err)
			}
			fileIndex++
		}
	})
}

// BenchmarkComponentRegistry_Operations benchmarks registry operations.
func BenchmarkComponentRegistry_Operations(b *testing.B) {
	b.Run("Register", func(b *testing.B) {
		reg := registry.NewComponentRegistry()

		b.ResetTimer()
		b.ReportAllocs()

		for i := range b.N {
			component := &types.ComponentInfo{
				Name:     fmt.Sprintf("Component%d", i),
				Package:  "components",
				FilePath: fmt.Sprintf("component_%d.templ", i),
				Parameters: []types.ParameterInfo{
					{Name: "title", Type: "string"},
					{Name: "active", Type: "bool"},
				},
			}
			reg.Register(component)
		}
	})

	b.Run("Get", func(b *testing.B) {
		reg := registry.NewComponentRegistry()

		// Pre-populate registry
		for i := range 1000 {
			component := &types.ComponentInfo{
				Name:     fmt.Sprintf("Component%d", i),
				Package:  "components",
				FilePath: fmt.Sprintf("component_%d.templ", i),
			}
			reg.Register(component)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := range b.N {
			componentName := fmt.Sprintf("Component%d", i%1000)
			_, _ = reg.Get(componentName)
		}
	})

	b.Run("GetAll", func(b *testing.B) {
		reg := registry.NewComponentRegistry()

		// Pre-populate registry
		for i := range 1000 {
			component := &types.ComponentInfo{
				Name:     fmt.Sprintf("Component%d", i),
				Package:  "components",
				FilePath: fmt.Sprintf("component_%d.templ", i),
			}
			reg.Register(component)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			_ = reg.GetAll()
		}
	})
}

// BenchmarkParallelVsSequential compares parallel vs sequential scanning performance.
func BenchmarkParallelVsSequential(b *testing.B) {
	componentCounts := []int{100, 500, 1000}

	for _, count := range componentCounts {
		testDir := createTestComponents(count)
		defer os.RemoveAll(testDir)

		b.Run(fmt.Sprintf("Sequential-%d", count), func(b *testing.B) {
			reg := registry.NewComponentRegistry()
			scanner := NewComponentScanner(reg)

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				// Force sequential scanning by using 1 worker
				err := scanner.ScanDirectoryParallel(testDir, 1)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("Parallel-%d", count), func(b *testing.B) {
			reg := registry.NewComponentRegistry()
			scanner := NewComponentScanner(reg)

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				// Use parallel scanning with default worker count (runtime.NumCPU())
				err := scanner.ScanDirectory(testDir)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkWorkerCount benchmarks different worker counts for parallel scanning.
func BenchmarkWorkerCount(b *testing.B) {
	testDir := createTestComponents(500)
	defer os.RemoveAll(testDir)

	workerCounts := []int{1, 2, 4, 8, 16}

	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("Workers-%d", workers), func(b *testing.B) {
			reg := registry.NewComponentRegistry()
			scanner := NewComponentScanner(reg)

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				err := scanner.ScanDirectoryParallel(testDir, workers)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkPathValidation benchmarks path validation performance.
func BenchmarkPathValidation(b *testing.B) {
	reg := registry.NewComponentRegistry()
	scanner := NewComponentScanner(reg)

	testPaths := []string{
		"component.templ",
		"./components/button.templ",
		"../components/card.templ",
		"./nested/deep/component.templ",
		"../../../etc/passwd",
		"/absolute/path/component.templ",
		"components/../other/component.templ",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := range b.N {
		path := testPaths[i%len(testPaths)]
		_, _ = scanner.validatePath(path)
	}
}
