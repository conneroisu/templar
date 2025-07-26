package server

import "time"

// GetCurrentTime returns the current time
// This utility function provides a consistent way to get timestamps across the server package.
func GetCurrentTime() time.Time {
	return time.Now()
}
