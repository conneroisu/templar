package build

import (
	"fmt"
	"hash/crc32"
	"strconv"
	"testing"
)

// BenchmarkHashOptimizations compares the old vs new hash implementation
func BenchmarkHashOptimizations(b *testing.B) {
	data := make([]byte, 64*1024) // 64KB test data
	for i := range data {
		data[i] = byte(i % 256)
	}

	// Old implementation (CRC32 IEEE + fmt.Sprintf)
	b.Run("Old_CRC32_IEEE_Printf", func(b *testing.B) {
		b.SetBytes(int64(len(data)))
		for i := 0; i < b.N; i++ {
			hash := crc32.ChecksumIEEE(data)
			_ = fmt.Sprintf("%x", hash)
		}
	})

	// New implementation (CRC32 Castagnoli + strconv.FormatUint)
	b.Run("New_CRC32_Castagnoli_FormatUint", func(b *testing.B) {
		crcTable := crc32.MakeTable(crc32.Castagnoli)
		b.SetBytes(int64(len(data)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			hash := crc32.Checksum(data, crcTable)
			_ = strconv.FormatUint(uint64(hash), 16)
		}
	})

	// Hybrid test: CRC32 IEEE + optimized string conversion
	b.Run("Hybrid_CRC32_IEEE_FormatUint", func(b *testing.B) {
		b.SetBytes(int64(len(data)))
		for i := 0; i < b.N; i++ {
			hash := crc32.ChecksumIEEE(data)
			_ = strconv.FormatUint(uint64(hash), 16)
		}
	})

	// Hybrid test: CRC32 Castagnoli + old string conversion
	b.Run("Hybrid_CRC32_Castagnoli_Printf", func(b *testing.B) {
		crcTable := crc32.MakeTable(crc32.Castagnoli)
		b.SetBytes(int64(len(data)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			hash := crc32.Checksum(data, crcTable)
			_ = fmt.Sprintf("%x", hash)
		}
	})
}

// BenchmarkHashOnlyPerformance benchmarks just the hash calculation without string conversion
func BenchmarkHashOnlyPerformance(b *testing.B) {
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
}
