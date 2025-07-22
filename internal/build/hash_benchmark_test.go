package build

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"hash/crc32"
	"hash/fnv"
	"strconv"
	"testing"
)

// BenchmarkHashFunctions compares performance of different hash functions
// on various data sizes to determine optimal choice for file change detection

func BenchmarkHashFunctions(b *testing.B) {
	// Test data of different sizes
	sizes := []int{
		1024,     // 1KB - small component
		10240,    // 10KB - medium component  
		102400,   // 100KB - large component
		1048576,  // 1MB - very large component
	}

	for _, size := range sizes {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i % 256)
		}

		b.Run(formatSize(size), func(b *testing.B) {
			// Benchmark CRC32 (current implementation)
			b.Run("CRC32", func(b *testing.B) {
				b.SetBytes(int64(size))
				for i := 0; i < b.N; i++ {
					_ = crc32.ChecksumIEEE(data)
				}
			})

			// Benchmark MD5 (for comparison - should be slower)
			b.Run("MD5", func(b *testing.B) {
				b.SetBytes(int64(size))
				for i := 0; i < b.N; i++ {
					_ = md5.Sum(data)
				}
			})

			// Benchmark SHA256 (for comparison - should be slowest)
			b.Run("SHA256", func(b *testing.B) {
				b.SetBytes(int64(size))
				for i := 0; i < b.N; i++ {
					_ = sha256.Sum256(data)
				}
			})

			// Benchmark FNV-1a (non-cryptographic alternative)
			b.Run("FNV1a", func(b *testing.B) {
				b.SetBytes(int64(size))
				for i := 0; i < b.N; i++ {
					h := fnv.New64a()
					h.Write(data)
					_ = h.Sum64()
				}
			})

			// Benchmark simple sum (baseline)
			b.Run("SimpleSum", func(b *testing.B) {
				b.SetBytes(int64(size))
				for i := 0; i < b.N; i++ {
					var sum uint64
					for _, b := range data {
						sum += uint64(b)
					}
					_ = sum
				}
			})
		})
	}
}

// BenchmarkCRC32Variants compares different CRC32 variants
func BenchmarkCRC32Variants(b *testing.B) {
	data := make([]byte, 64*1024) // 64KB test data
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.Run("CRC32_IEEE", func(b *testing.B) {
		b.SetBytes(int64(len(data)))
		for i := 0; i < b.N; i++ {
			_ = crc32.ChecksumIEEE(data)
		}
	})

	b.Run("CRC32_Castagnoli", func(b *testing.B) {
		crcTable := crc32.MakeTable(crc32.Castagnoli)
		b.SetBytes(int64(len(data)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = crc32.Checksum(data, crcTable)
		}
	})

	b.Run("CRC32_Koopman", func(b *testing.B) {
		crcTable := crc32.MakeTable(crc32.Koopman)
		b.SetBytes(int64(len(data)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = crc32.Checksum(data, crcTable)
		}
	})
}

// BenchmarkHashStringConversion tests the cost of converting hash to string
func BenchmarkHashStringConversion(b *testing.B) {
	data := make([]byte, 64*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.Run("CRC32_Printf", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			hash := crc32.ChecksumIEEE(data)
			_ = formatHash(hash)
		}
	})

	b.Run("CRC32_FormatUint", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			hash := crc32.ChecksumIEEE(data)
			_ = formatHashOptimized(hash)
		}
	})
}

// formatHash formats hash using current method (fmt.Sprintf)
func formatHash(hash uint32) string {
	return formatHashInline(hash)
}

// formatHashOptimized formats hash using optimized method
func formatHashOptimized(hash uint32) string {
	return formatHashInlineOptimized(hash)
}

// Inline functions to avoid call overhead in benchmarks
func formatHashInline(hash uint32) string {
	return fmt.Sprintf("%x", hash)
}

func formatHashInlineOptimized(hash uint32) string {
	return strconv.FormatUint(uint64(hash), 16)
}

// formatSize converts size to human readable string
func formatSize(size int) string {
	if size >= 1024*1024 {
		return "1MB"
	} else if size >= 1024*100 {
		return "100KB"
	} else if size >= 1024*10 {
		return "10KB"
	} else {
		return "1KB"
	}
}

// BenchmarkMemoryPooledHashing tests if using a memory pool for hash results improves performance
func BenchmarkMemoryPooledHashing(b *testing.B) {
	data := make([]byte, 64*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	// Test current approach
	b.Run("DirectAllocation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			hash := crc32.ChecksumIEEE(data)
			result := make([]byte, 8) // Simulate hash result allocation
			copy(result, []byte{byte(hash), byte(hash >> 8), byte(hash >> 16), byte(hash >> 24)})
			_ = result
		}
	})

	// Test with hypothetical pool
	b.Run("PooledAllocation", func(b *testing.B) {
		pool := make(chan []byte, 100)
		// Pre-populate pool
		for i := 0; i < 100; i++ {
			pool <- make([]byte, 8)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			hash := crc32.ChecksumIEEE(data)
			var result []byte
			select {
			case result = <-pool:
			default:
				result = make([]byte, 8)
			}
			copy(result, []byte{byte(hash), byte(hash >> 8), byte(hash >> 16), byte(hash >> 24)})
			
			// Return to pool
			select {
			case pool <- result:
			default:
				// Pool full, discard
			}
		}
	})
}