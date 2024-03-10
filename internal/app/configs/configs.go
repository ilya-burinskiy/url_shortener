package configs

import (
	"flag"
	"os"
)

// Application configs
type Config struct {
	ServerAddress        string
	ShortenedURLBaseAddr string
	FileStoragePath      string
	DatabaseDSN          string
}

// Parse configs from flag variables. Environment variables override flag variables
func Parse() Config {
	config := Config{}
	flag.StringVar(&config.ServerAddress, "a", "localhost:8080", "server's address")
	flag.StringVar(
		&config.ShortenedURLBaseAddr,
		"b", "http://localhost:8080",
		"base address of the resulting shortened URL")
	flag.StringVar(&config.FileStoragePath, "f", "", "file storage path")
	flag.StringVar(&config.DatabaseDSN, "d", "", "database URL")
	flag.Parse()

	if envServerAddress := os.Getenv("SERVER_ADDRESS"); envServerAddress != "" {
		config.ServerAddress = envServerAddress
	}
	if envShortenedURLBaseAddr := os.Getenv("BASE_URL"); envShortenedURLBaseAddr != "" {
		config.ShortenedURLBaseAddr = envShortenedURLBaseAddr
	}
	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		config.FileStoragePath = envFileStoragePath
	}
	if envDatabaseDSN := os.Getenv("DATABASE_DSN"); envDatabaseDSN != "" {
		config.DatabaseDSN = envDatabaseDSN
	}

	return config
}

// Use database storage
func (c Config) UseDBStorage() bool {
	return c.DatabaseDSN != ""
}

// Use file storage
func (c Config) UseFileStorage() bool {
	return c.FileStoragePath != ""
}
