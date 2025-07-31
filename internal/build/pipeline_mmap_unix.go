//go:build !windows

package build

import (
	"os"
	"syscall"
)

// readFileWithMmap reads file content using memory mapping for better performance on large files.
func (bp *BuildPipeline) readFileWithMmap(file *os.File, size int64) ([]byte, error) {
	// Memory map the file for efficient reading
	mmap, err := syscall.Mmap(int(file.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}

	// Copy the mapped data to avoid keeping the mapping open
	content := make([]byte, size)
	copy(content, mmap)

	// Unmap the memory - ignore errors as we have the content
	_ = syscall.Munmap(mmap)

	return content, nil
}