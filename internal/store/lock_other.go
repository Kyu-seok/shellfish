//go:build !unix && !windows

package store

import "os"

// lockFile is a no-op where the OS offers no file lock shellfish uses.
func lockFile(*os.File) error { return nil }
