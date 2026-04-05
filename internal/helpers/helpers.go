package helpers

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/0dev1337/tmpvault-backend/internal/config"
)

func GetStoragePath(cfg *config.Config, expiry string) string {
	sub, ok := config.Configuration.Expiry[expiry]
	if !ok {
		sub = "default"
	}

	return filepath.Join(config.Configuration.Storage.Path, sub)
}

func ParseExpiry(input string) time.Time {
	d, _ := time.ParseDuration(input)
	return time.Now().Add(d)
}

func GenFileName(original string) string {
	ext := strings.ToLower(filepath.Ext(original))
	if ext == "" {
		ext = ".bin" // fallback
	}
	id, _ := gonanoid.New(10)
	return fmt.Sprintf("%s%s", id, ext)
}
