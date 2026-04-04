package config

type Config struct {
	Server Server `toml:"server"`
}

type Server struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}
