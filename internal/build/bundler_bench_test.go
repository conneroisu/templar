package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/conneroisu/templar/internal/config"
)

// createBenchConfig creates a config for benchmarking
func createBenchConfig(b *testing.B) *config.Config {
	return &config.Config{
		Build: config.BuildConfig{
			Command:  "templ generate",
			Watch:    []string{"**/*.templ"},
			Ignore:   []string{"node_modules", ".git"},
			CacheDir: ".templar/cache",
		},
	}
}

// BenchmarkDiscoverAssets benchmarks asset discovery operations
func BenchmarkDiscoverAssets(b *testing.B) {
	// Create test directory with different numbers of files
	testDirs := []struct {
		name      string
		numFiles  int
		structure string // flat, nested, mixed
	}{
		{"small_flat", 10, "flat"},
		{"medium_flat", 100, "flat"},
		{"large_flat", 1000, "flat"},
		{"small_nested", 10, "nested"},
		{"medium_nested", 100, "nested"},
		{"large_nested", 1000, "nested"},
		{"mixed_structure", 500, "mixed"},
	}

	for _, testDir := range testDirs {
		b.Run(testDir.name, func(b *testing.B) {
			// Create temporary directory structure
			tempDir := createTestAssetStructure(b, testDir.numFiles, testDir.structure)
			defer os.RemoveAll(tempDir)

			cfg := createBenchConfig(b)
			bundler := NewAssetBundler(cfg, tempDir)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				manifest, err := bundler.DiscoverAssets(ctx)
				if err != nil {
					b.Fatal(err)
				}
				_ = manifest
			}
		})
	}
}

// BenchmarkBundle benchmarks different bundling operations
func BenchmarkBundle(b *testing.B) {
	// Create test assets
	tempDir := createTestAssetStructure(b, 50, "mixed")
	defer os.RemoveAll(tempDir)

	cfg := createBenchConfig(b)
	bundler := NewAssetBundler(cfg, tempDir)
	ctx := context.Background()

	// Discover assets first
	manifest, err := bundler.DiscoverAssets(ctx)
	if err != nil {
		b.Fatal(err)
	}

	outputDir, err := os.MkdirTemp("", "bench_output_*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(outputDir)

	bundleTypes := []struct {
		name   string
		minify bool
	}{
		{"standard", false},
		{"minified", true},
	}

	for _, bundleType := range bundleTypes {
		b.Run(bundleType.name, func(b *testing.B) {
			options := BundlerOptions{
				Minify:      bundleType.minify,
				Environment: "production",
				Target:      "es2020",
				Format:      "esm",
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err := bundler.Bundle(ctx, manifest, options)
				if err != nil {
					b.Fatal(err)
				}
				_ = result
			}
		})
	}
}

// BenchmarkAssetProcessing benchmarks asset processing operations
func BenchmarkAssetProcessing(b *testing.B) {
	tempDir := createTestAssetStructure(b, 100, "mixed")
	defer os.RemoveAll(tempDir)

	cfg := createBenchConfig(b)
	bundler := NewAssetBundler(cfg, tempDir)
	ctx := context.Background()

	b.Run("asset_discovery", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			manifest, err := bundler.DiscoverAssets(ctx)
			if err != nil {
				b.Fatal(err)
			}
			_ = manifest
		}
	})
}

// BenchmarkConcurrentBundling benchmarks concurrent bundling operations
func BenchmarkConcurrentBundling(b *testing.B) {
	// Create test assets
	tempDir := createTestAssetStructure(b, 200, "mixed")
	defer os.RemoveAll(tempDir)

	cfg := createBenchConfig(b)

	b.Run("sequential_bundling", func(b *testing.B) {
		bundler := NewAssetBundler(cfg, tempDir)
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			manifest, err := bundler.DiscoverAssets(ctx)
			if err != nil {
				b.Fatal(err)
			}

			options := BundlerOptions{
				Minify:      false,
				Environment: "development",
				Target:      "es2020",
				Format:      "esm",
			}

			result, err := bundler.Bundle(ctx, manifest, options)
			if err != nil {
				b.Fatal(err)
			}
			_ = result
		}
	})

	b.Run("concurrent_bundling", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			bundler := NewAssetBundler(cfg, tempDir)
			ctx := context.Background()

			for pb.Next() {
				manifest, err := bundler.DiscoverAssets(ctx)
				if err != nil {
					b.Fatal(err)
				}

				options := BundlerOptions{
					Minify:      false,
					Environment: "development",
					Target:      "es2020",
					Format:      "esm",
				}

				result, err := bundler.Bundle(ctx, manifest, options)
				if err != nil {
					b.Fatal(err)
				}
				_ = result
			}
		})
	})
}

// BenchmarkMemoryUsage benchmarks memory usage patterns
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("large_asset_processing", func(b *testing.B) {
		// Create large assets
		tempDir, err := os.MkdirTemp("", "memory_bench_*")
		if err != nil {
			b.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		// Create a large JavaScript file (1MB)
		largeContent := fmt.Sprintf("// Large content\n%s",
			strings.Repeat("console.log('test');\n", 50000))
		largeFile := filepath.Join(tempDir, "large.js")
		err = os.WriteFile(largeFile, []byte(largeContent), 0644)
		if err != nil {
			b.Fatal(err)
		}

		cfg := createBenchConfig(b)
		bundler := NewAssetBundler(cfg, tempDir)
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			manifest, err := bundler.DiscoverAssets(ctx)
			if err != nil {
				b.Fatal(err)
			}
			_ = manifest
		}
	})

	b.Run("many_small_assets", func(b *testing.B) {
		tempDir, err := os.MkdirTemp("", "many_assets_*")
		if err != nil {
			b.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		// Create many small assets
		for i := 0; i < 1000; i++ {
			content := fmt.Sprintf("var x%d = %d;\n", i, i)
			fileName := fmt.Sprintf("asset_%d.js", i)
			filePath := filepath.Join(tempDir, fileName)

			err := os.WriteFile(filePath, []byte(content), 0644)
			if err != nil {
				b.Fatal(err)
			}
		}

		cfg := createBenchConfig(b)
		bundler := NewAssetBundler(cfg, tempDir)
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			manifest, err := bundler.DiscoverAssets(ctx)
			if err != nil {
				b.Fatal(err)
			}
			_ = manifest
		}
	})
}

// Helper functions for benchmark setup

func createTestAssetStructure(b *testing.B, numFiles int, structure string) string {
	tempDir, err := os.MkdirTemp("", "asset_bench_*")
	if err != nil {
		b.Fatal(err)
	}

	switch structure {
	case "flat":
		createFlatStructure(b, tempDir, numFiles)
	case "nested":
		createNestedStructure(b, tempDir, numFiles)
	case "mixed":
		createMixedStructure(b, tempDir, numFiles)
	}

	return tempDir
}

func createFlatStructure(b *testing.B, baseDir string, numFiles int) {
	extensions := []string{"js", "css", "ts", "scss"}

	for i := 0; i < numFiles; i++ {
		ext := extensions[i%len(extensions)]
		fileName := fmt.Sprintf("file_%d.%s", i, ext)
		filePath := filepath.Join(baseDir, fileName)

		content := generateTestContent(ext, i)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func createNestedStructure(b *testing.B, baseDir string, numFiles int) {
	extensions := []string{"js", "css", "ts", "scss"}
	dirsPerLevel := 5
	maxDepth := 4

	for i := 0; i < numFiles; i++ {
		ext := extensions[i%len(extensions)]

		// Create nested directory path
		var pathParts []string
		for depth := 0; depth < maxDepth && i > 0; depth++ {
			dirNum := (i / (depth + 1)) % dirsPerLevel
			pathParts = append(pathParts, fmt.Sprintf("dir_%d_%d", depth, dirNum))
		}

		dirPath := filepath.Join(append([]string{baseDir}, pathParts...)...)
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			b.Fatal(err)
		}

		fileName := fmt.Sprintf("file_%d.%s", i, ext)
		filePath := filepath.Join(dirPath, fileName)

		content := generateTestContent(ext, i)
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func createMixedStructure(b *testing.B, baseDir string, numFiles int) {
	// Mix of flat and nested files
	flatFiles := numFiles / 2
	nestedFiles := numFiles - flatFiles

	createFlatStructure(b, baseDir, flatFiles)
	createNestedStructure(b, baseDir, nestedFiles)
}

func generateTestContent(ext string, index int) string {
	switch ext {
	case "js":
		return fmt.Sprintf(`
			function func_%d() {
				var x = %d;
				var y = "value_%d";
				console.log("Function %d called");
				return x * %d;
			}
			
			var global_%d = func_%d();
		`, index, index, index, index, index+1, index, index)

	case "css":
		return fmt.Sprintf(`
			.class_%d {
				width: %dpx;
				height: %dpx;
				margin: %dpx;
				padding: %dpx;
				background-color: #%06x;
			}
			
			#id_%d {
				position: relative;
				z-index: %d;
			}
		`, index, index*10, index*8, index%20, index%10,
			(index*12345)&0xFFFFFF, index, index%100)

	case "ts":
		return fmt.Sprintf(`
			interface Interface_%d {
				prop%d: number;
				method%d(): string;
			}
			
			class Class_%d implements Interface_%d {
				private value: number = %d;
				
				method%d(): string {
					return "class_%d_method";
				}
			}
		`, index, index, index, index, index, index, index, index)

	case "scss":
		return fmt.Sprintf(`
			$color_%d: #%06x;
			$size_%d: %dpx;
			
			.component_%d {
				color: $color_%d;
				font-size: $size_%d;
				
				&:hover {
					opacity: 0.%d;
				}
				
				.nested_%d {
					margin: %dpx;
				}
			}
		`, index, (index*54321)&0xFFFFFF, index, index%50+10,
			index, index, index, index%10, index, index%30+5)

	default:
		return fmt.Sprintf("/* Content for file %d */\n", index)
	}
}

// Additional I/O intensive benchmarks
func BenchmarkFileIOOperations(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "io_bench_*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files of different sizes
	fileSizes := map[string]int{
		"small":  1024,   // 1KB
		"medium": 10240,  // 10KB
		"large":  102400, // 100KB
	}

	cfg := createBenchConfig(b)
	bundler := NewAssetBundler(cfg, tempDir)

	for sizeName, size := range fileSizes {
		fileName := fmt.Sprintf("test_%s.js", sizeName)
		filePath := filepath.Join(tempDir, fileName)
		content := strings.Repeat("a", size)

		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			b.Fatal(err)
		}

		b.Run(fmt.Sprintf("process_%s", sizeName), func(b *testing.B) {
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				manifest, err := bundler.DiscoverAssets(ctx)
				if err != nil {
					b.Fatal(err)
				}
				_ = manifest
			}
		})
	}
}

// BenchmarkErrorHandling benchmarks error handling paths
func BenchmarkErrorHandling(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "error_bench_*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	cfg := createBenchConfig(b)
	bundler := NewAssetBundler(cfg, tempDir)

	b.Run("discovery_with_invalid_files", func(b *testing.B) {
		// Create some invalid/empty files
		for i := 0; i < 10; i++ {
			fileName := fmt.Sprintf("invalid_%d.js", i)
			filePath := filepath.Join(tempDir, fileName)
			os.WriteFile(filePath, []byte(""), 0000) // No permissions
		}

		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := bundler.DiscoverAssets(ctx)
			// Don't fail on errors - we're benchmarking error handling
			_ = err
		}
	})
}
