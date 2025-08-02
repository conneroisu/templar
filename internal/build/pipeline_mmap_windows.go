//go:build windows

package build

import (
	"io"
	"os"
)

// readFileWithMmap reads file content using standard I/O on Windows (mmap not easily available).
func (bp *BuildPipeline) readFileWithMmap(file *os.File, size int64) ([]byte, error) {
	// On Windows, fall back to standard file I/O
	// This is less optimal but ensures compatibility
	return io.ReadAll(file)
}
