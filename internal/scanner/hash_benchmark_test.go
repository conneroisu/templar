package scanner

import (
	"hash/crc32"
	"os"
	"strconv"
	"testing"
	"time"
)

// BenchmarkHashGeneration compares original vs optimized hash generation
func BenchmarkHashGeneration(b *testing.B) {
	// Create test data of different sizes
	testSizes := []struct {
		name string
		size int
	}{
		{"Small_1KB", 1024},
		{"Medium_32KB", 32 * 1024},
		{"Large_256KB", 256 * 1024},
		{"XLarge_1MB", 1024 * 1024},
		{"XXLarge_4MB", 4 * 1024 * 1024},
	}
	
	for _, testSize := range testSizes {
		content := make([]byte, testSize.size)
		// Fill with realistic data (simulate templ file content)
		for i := range content {
			content[i] = byte(i % 256)
		}
		
		// Create mock file info
		fileInfo := &mockFileInfo{
			size:    int64(testSize.size),
			modTime: time.Now(),
		}
		
		// Test original hash generation
		b.Run("Original_"+testSize.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Original CRC32 hash on full content
				_ = strconv.FormatUint(uint64(crc32.Checksum(content, crcTable)), 16)
			}
		})
		
		// Test optimized hash generation
		scanner := &ComponentScanner{}
		b.Run("Optimized_"+testSize.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = scanner.generateOptimizedHash(content, fileInfo)
			}
		})
	}
}

// BenchmarkHashStrategies tests different hashing strategies individually
func BenchmarkHashStrategies(b *testing.B) {
	// Large file content (1MB)
	content := make([]byte, 1024*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}
	
	scanner := &ComponentScanner{}
	fileInfo := &mockFileInfo{size: int64(len(content)), modTime: time.Now()}
	
	b.Run("FullContent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = crc32.Checksum(content, crcTable)
		}
	})
	
	b.Run("SampledHash", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = scanner.generateSampledHash(content)
		}
	})
	
	b.Run("HierarchicalHash", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = scanner.generateHierarchicalHash(content, fileInfo)
		}
	})
}

// mockFileInfo implements os.FileInfo for testing
type mockFileInfo struct {
	size    int64
	modTime time.Time
}

func (m *mockFileInfo) Name() string       { return "test.templ" }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return m.modTime }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() interface{}   { return nil }