package helpers

import (
	"crypto/rand"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

const defaultGeneratedIDLength = 10

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
	name, err := GenFileNameWithLength(original, 0)
	if err == nil {
		return name
	}

	ext := strings.ToLower(filepath.Ext(original))
	if ext == "" {
		ext = ".bin"
	}
	id, err := gonanoid.New(defaultGeneratedIDLength)
	if err != nil || id == "" {
		id = strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return id + ext
}

// GenFileNameWithLength returns a unique filename preserving extension and
// matching the requested total file ID length (including extension).
func GenFileNameWithLength(original string, fileIDLength int) (string, error) {
	ext := strings.ToLower(filepath.Ext(original))
	if ext == "" {
		ext = ".bin"
	}

	idLength := defaultGeneratedIDLength
	if fileIDLength > 0 {
		idLength = fileIDLength - len(ext)
		if idLength < 1 {
			return "", fmt.Errorf("file_name_length must be greater than extension length")
		}
	}

	id, err := gonanoid.New(idLength)
	if err != nil || id == "" {
		id, err = randomString(idLength)
		if err != nil {
			return "", err
		}
	}

	return id + ext, nil
}

func randomString(length int) (string, error) {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(buf), nil
}
