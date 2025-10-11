package downloads

import (
	"sync"
)

var (
	cacheOnce sync.Once
	cachedDir string
	cacheErr  error
)

// GetUserDownloadsDir returns the user's Downloads directory, cached across calls.
// It uses a platform-specific implementation in getDownloadsDir().
func GetUserDownloadsDir() (string, error) {
	cacheOnce.Do(func() {
		cachedDir, cacheErr = getDownloadsDir()
	})
	return cachedDir, cacheErr
}
