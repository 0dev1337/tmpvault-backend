package upload

import (
	"context"
	"os"
	"sync"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

const (
	sessionIDLength = 21
	sessionTTL      = 2 * time.Hour
)

// uploadSession tracks one in-progress chunked upload. Chunks are written
// directly into a pre-sized temp file by offset, so they may be retried or
// (in principle) sent out of order; completeness is tracked per chunk index.
type uploadSession struct {
	mu            sync.Mutex
	TempPath      string
	TotalSize     int64
	ChunkSize     int64
	TotalChunks   int
	received      []bool
	OriginalName  string
	GeneratedName string
	Expiry        string
	ExpirySubdir  string
	ExpiryUnix    int64
	CreatedAt     time.Time
}

var (
	sessionsMu sync.Mutex
	sessions   = map[string]*uploadSession{}
)

func newSessionID() (string, error) {
	return gonanoid.New(sessionIDLength)
}

func putSession(id string, s *uploadSession) {
	sessionsMu.Lock()
	sessions[id] = s
	sessionsMu.Unlock()
}

func getSession(id string) (*uploadSession, bool) {
	sessionsMu.Lock()
	s, ok := sessions[id]
	sessionsMu.Unlock()
	return s, ok
}

func deleteSession(id string) {
	sessionsMu.Lock()
	delete(sessions, id)
	sessionsMu.Unlock()
}

// cleanupStaleSessions removes sessions (and their temp files) that have sat
// unfinished for longer than sessionTTL, so abandoned uploads don't leak disk
// or memory. Sessions live only in memory, so a server restart also drops
// any in-progress chunked upload; clients must restart it from init.
func cleanupStaleSessions() {
	cutoff := time.Now().Add(-sessionTTL)

	type stale struct{ id, path string }
	var toRemove []stale

	sessionsMu.Lock()
	for id, s := range sessions {
		if s.CreatedAt.Before(cutoff) {
			toRemove = append(toRemove, stale{id, s.TempPath})
			delete(sessions, id)
		}
	}
	sessionsMu.Unlock()

	for _, s := range toRemove {
		_ = os.Remove(s.path)
	}
}

// StartSessionJanitor periodically clears out abandoned chunked-upload
// sessions until ctx is cancelled.
func StartSessionJanitor(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanupStaleSessions()
		}
	}
}
