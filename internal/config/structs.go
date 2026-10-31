package config

type Config struct {
	Server   Server            `toml:"server"`
	Expiry   map[string]string `toml:"expiry_paths"`
	Storage  Storage           `toml:"storage"`
	Database Database          `toml:"database"`
}

type Server struct {
	Host          string `toml:"host"`
	Port          int    `toml:"port"`
	MaxUploadSize int    `toml:"max_upload_limit"`
	ReadTimeout   int    `toml:"read_timeout_seconds"`
	WriteTimeout  int    `toml:"write_timeout_seconds"`
	IdleTimeout   int    `toml:"idle_timeout_seconds"`
}

type Storage struct {
	Path string `toml:"base_path"`
}

type Database struct {
	MongoURI string `toml:"mongodb_url"`
	DBName   string `toml:"db_name"`
}
