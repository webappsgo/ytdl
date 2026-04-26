// Package config handles configuration loading and validation.
// See AI.md PART 5 for complete configuration specifications.
// Configuration hierarchy: CLI flags > env vars > file > defaults
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// Environment variable prefix for all config overrides
	envPrefix = "YTDL_"
)

// ServerConfig holds the complete server configuration
type ServerConfig struct {
	Server   ServerSection   `yaml:"server"`
	Download DownloadSection `yaml:"download"`
}

// ServerSection holds core server settings
type ServerSection struct {
	// Application mode: production or development
	Mode string `yaml:"mode"`
	// Listen address
	Address string `yaml:"address"`
	// Listen port
	Port int `yaml:"port"`
	// URL path prefix
	BaseURL string `yaml:"baseurl"`
	// Admin panel path
	AdminPath string `yaml:"admin_path"`
	// Debug mode
	Debug bool `yaml:"debug"`
	// FQDN for the server
	FQDN string `yaml:"fqdn"`
	// Database configuration
	Database DatabaseConfig `yaml:"database"`
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	// Database driver: sqlite, libsql, postgres, mysql, mssql, mongodb
	Driver string `yaml:"driver"`
	// Database URL or path
	URL string `yaml:"url"`
	// Authentication token (for libsql/turso)
	Token string `yaml:"token"`
}

// DownloadSection holds download-specific settings
type DownloadSection struct {
	// Number of concurrent download workers
	Workers int `yaml:"workers"`
	// Maximum download queue size
	MaxQueueSize int `yaml:"max_queue_size"`
	// File retention period in hours (0 = keep forever)
	RetentionHours int `yaml:"retention_hours"`
	// Maximum file size in bytes (0 = unlimited)
	MaxFileSize int64 `yaml:"max_file_size"`
	// Global download speed limit in bytes/sec (0 = unlimited)
	SpeedLimit int64 `yaml:"speed_limit"`
	// Path to yt-dlp binary
	YTDLPPath string `yaml:"ytdlp_path"`
	// Path to ffmpeg binary
	FFmpegPath string `yaml:"ffmpeg_path"`
	// Output path template
	OutputTemplate string `yaml:"output_template"`
	// Default audio bitrate
	DefaultAudioBitrate string `yaml:"default_audio_bitrate"`
	// Default video quality
	DefaultVideoQuality string `yaml:"default_video_quality"`
	// Default subtitle languages
	DefaultSubtitleLanguages []string `yaml:"default_subtitle_languages"`
}

// CLIOverrides holds values explicitly set via CLI flags.
// Defined here so both main package and config package can share the type.
type CLIOverrides struct {
	Mode      string
	ConfigDir string
	DataDir   string
	CacheDir  string
	LogDir    string
	BackupDir string
	PIDFile   string
	Address   string
	Port      int
	BaseURL   string
	Debug     bool
	Daemon    bool
	Color     string
}

// LoadConfig loads configuration from file with CLI overrides applied.
// Priority: CLI flags > env vars > file > defaults
func LoadConfig(configFile string, cliOverrides CLIOverrides) (*ServerConfig, error) {
	// Start with defaults
	cfg := defaultConfig()

	// Load from file if exists
	if err := loadFromFile(configFile, cfg); err != nil {
		// File not found is OK (first run) - use defaults
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	}

	// Apply environment variable overrides
	if err := applyEnvOverrides(cfg); err != nil {
		return nil, fmt.Errorf("applying environment overrides: %w", err)
	}

	// Apply CLI overrides
	applyCLIOverrides(cfg, cliOverrides)

	// Validate
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return cfg, nil
}

func defaultConfig() *ServerConfig {
	return &ServerConfig{
		Server: ServerSection{
			Mode:      "production",
			Address:   "0.0.0.0",
			Port:      0,
			BaseURL:   "/",
			AdminPath: "admin",
			Debug:     false,
			Database: DatabaseConfig{
				Driver: "sqlite",
			},
		},
		Download: DownloadSection{
			Workers:                  2,
			MaxQueueSize:             100,
			RetentionHours:           72,
			MaxFileSize:              0,
			SpeedLimit:               0,
			YTDLPPath:                "yt-dlp",
			FFmpegPath:               "ffmpeg",
			OutputTemplate:           "{site}/{channel}/{title}.{ext}",
			DefaultAudioBitrate:      "320k",
			DefaultVideoQuality:      "1080",
			DefaultSubtitleLanguages: []string{"en", "es"},
		},
	}
}

func loadFromFile(path string, cfg *ServerConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parsing YAML: %w", err)
	}

	return nil
}

func applyEnvOverrides(cfg *ServerConfig) error {
	if v := os.Getenv(envPrefix + "MODE"); v != "" {
		cfg.Server.Mode = v
	}
	if v := os.Getenv(envPrefix + "ADDRESS"); v != "" {
		cfg.Server.Address = v
	}
	if v := os.Getenv(envPrefix + "PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv(envPrefix + "BASEURL"); v != "" {
		cfg.Server.BaseURL = v
	}
	if v := os.Getenv(envPrefix + "DEBUG"); v != "" {
		parsedValue, err := ParseBool(v, cfg.Server.Debug)
		if err != nil {
			return fmt.Errorf("%sDEBUG: %w", envPrefix, err)
		}
		cfg.Server.Debug = parsedValue
	}
	if v := os.Getenv(envPrefix + "FQDN"); v != "" {
		cfg.Server.FQDN = v
	}
	if v := os.Getenv(envPrefix + "DB_DRIVER"); v != "" {
		cfg.Server.Database.Driver = v
	}
	if v := os.Getenv(envPrefix + "DB_URL"); v != "" {
		cfg.Server.Database.URL = v
	}
	if v := os.Getenv(envPrefix + "WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Download.Workers = n
		}
	}
	if v := os.Getenv(envPrefix + "SPEED_LIMIT"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.Download.SpeedLimit = n
		}
	}

	return nil
}

func applyCLIOverrides(cfg *ServerConfig, o CLIOverrides) {
	if o.Mode != "" {
		cfg.Server.Mode = o.Mode
	}
	if o.Address != "" {
		cfg.Server.Address = o.Address
	}
	if o.Port != 0 {
		cfg.Server.Port = o.Port
	}
	if o.BaseURL != "" {
		cfg.Server.BaseURL = o.BaseURL
	}
	if o.Debug {
		cfg.Server.Debug = true
	}
}

func validateConfig(cfg *ServerConfig) error {
	// Validate mode
	appMode := strings.ToLower(cfg.Server.Mode)
	if appMode != "production" && appMode != "development" && appMode != "dev" {
		return fmt.Errorf("invalid mode %q: must be 'production' or 'development'", cfg.Server.Mode)
	}

	// Validate port range
	if cfg.Server.Port < 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid port %d: must be 0-65535", cfg.Server.Port)
	}

	// Validate download workers
	if cfg.Download.Workers < 1 {
		cfg.Download.Workers = 1
	}
	if cfg.Download.Workers > 32 {
		cfg.Download.Workers = 32
	}

	// Validate queue size
	if cfg.Download.MaxQueueSize < 1 {
		cfg.Download.MaxQueueSize = 100
	}

	// Normalize database driver
	cfg.Server.Database.Driver = NormalizeDBDriver(cfg.Server.Database.Driver)

	return nil
}

// NormalizeDBDriver maps user-friendly config values to actual Go driver names
func NormalizeDBDriver(driver string) string {
	switch strings.ToLower(driver) {
	case "sqlite", "sqlite2", "sqlite3":
		return "sqlite"
	case "libsql", "turso":
		return "libsql"
	case "postgres", "pgsql", "postgresql":
		return "pgx"
	case "mysql", "mariadb":
		return "mysql"
	case "mssql":
		return "sqlserver"
	case "mongodb", "mongo":
		return "mongodb"
	default:
		return driver
	}
}

// SaveConfig writes the current configuration to the specified file
func SaveConfig(path string, cfg *ServerConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
