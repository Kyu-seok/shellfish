package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// LockFileName is the single-instance lock file's name inside the data folder.
const LockFileName = "fish.lock"

// ErrLocked means another open Tank already holds the data folder's lock.
var ErrLocked = errors.New("data folder is locked by another shellfish")

// Lock is an OS-level exclusive lock on the data folder. The OS drops it when
// the process ends, however it ends, so a crash never leaves a stale lock.
type Lock struct{ f *os.File }

// Acquire takes the data folder's lock without waiting, creating dir if
// needed. It returns ErrLocked if another process (or another Acquire in this
// one) holds it.
func Acquire(dir string) (*Lock, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("locking data folder: %w", err)
	}
	f, err := os.OpenFile(filepath.Join(dir, LockFileName), os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("locking data folder: %w", err)
	}
	if err := lockFile(f); err != nil {
		f.Close()
		if errors.Is(err, ErrLocked) {
			return nil, ErrLocked
		}
		return nil, fmt.Errorf("locking data folder: %w", err)
	}
	return &Lock{f: f}, nil
}

// Release drops the lock. It is safe to call more than once, and on nil.
// The lock file stays behind: removing it could let two processes lock
// different files.
func (l *Lock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	f := l.f
	l.f = nil
	return f.Close() // closing the file drops the OS lock
}
