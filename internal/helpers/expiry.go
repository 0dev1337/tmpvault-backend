package helpers

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

// ParseExpiryDuration parses values like "1h", "30m", "24h", or "1d"/"3d" (days).
func ParseExpiryDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty expiry")
	}
	if len(s) >= 2 && s[len(s)-1] == 'd' {
		nStr := s[:len(s)-1]
		n, err := strconv.Atoi(nStr)
		if err != nil || n < 0 {
			return 0, fmt.Errorf("invalid expiry %q", s)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

// ExpiryToUnix returns Unix seconds for now + expiry duration.
func ExpiryToUnix(expiry string) (int64, error) {
	d, err := ParseExpiryDuration(expiry)
	if err != nil {
		return 0, err
	}
	return time.Now().Add(d).Unix(), nil
}

// GenFileName returns a unique filename preserving extension.
func GenFileName(original string) string {
	ext := strings.ToLower(filepath.Ext(original))
	if ext == "" {
		ext = ".bin"
	}
	id, err := gonanoid.New(10)
	if err != nil || id == "" {
		id = strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return id + ext
}
