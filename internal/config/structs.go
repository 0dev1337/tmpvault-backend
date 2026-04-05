package config

type Config struct {
	Server  Server      `toml:"server"`
	Expiry  ExpiryPaths `toml:"expiry_paths"`
	Storage Storage     `toml:"storage"`
}

type Server struct {
	Host          string `toml:"host"`
	Port          int    `toml:"port"`
	MaxUploadSize int    `toml:"max_upload_limit"`
}

type ExpiryPaths struct {
	Paths map[string]string `toml:"expiry_paths"`
}

type Storage struct {
	Path string `toml:"base_path"`
}
