// Package paths resolves OS-specific paths for config, data, logs, cache, and backups.
// See AI.md PART 4 for complete path specifications.
package paths

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// Hardcoded project identifiers (never changes, even if binary is renamed)
	projectOrg  = "casapps"
	projectName = "ytdl"
)

// PathConfig holds all resolved paths for the application
type PathConfig struct {
	ConfigDir  string
	ConfigFile string
	DataDir    string
	CacheDir   string
	LogDir     string
	LogFile    string
	BackupDir  string
	PIDFile    string
	SSLDir     string
	SecurityDir string
	DBDir      string
}

// ResolvePaths determines all paths based on OS and privilege level.
// CLI overrides take precedence over defaults.
func ResolvePaths(isRoot bool, configDir, dataDir, cacheDir, logDir, backupDir, pidFile string) PathConfig {
	pc := resolveDefaultPaths(isRoot)

	// Apply CLI overrides
	if configDir != "" {
		pc.ConfigDir = configDir
		pc.ConfigFile = filepath.Join(configDir, "server.yml")
		pc.SSLDir = filepath.Join(configDir, "ssl")
		pc.SecurityDir = filepath.Join(configDir, "security")
	}
	if dataDir != "" {
		pc.DataDir = dataDir
		pc.DBDir = filepath.Join(dataDir, "db")
	}
	if cacheDir != "" {
		pc.CacheDir = cacheDir
	}
	if logDir != "" {
		pc.LogDir = logDir
		pc.LogFile = filepath.Join(logDir, "server.log")
	}
	if backupDir != "" {
		pc.BackupDir = backupDir
	}
	if pidFile != "" {
		pc.PIDFile = pidFile
	}

	return pc
}

func resolveDefaultPaths(isRoot bool) PathConfig {
	// Docker/container check
	if IsRunningInContainer() {
		return containerPaths()
	}

	switch runtime.GOOS {
	case "darwin":
		return darwinPaths(isRoot)
	case "windows":
		return windowsPaths(isRoot)
	case "freebsd", "openbsd", "netbsd":
		return bsdPaths(isRoot)
	default:
		// Linux and other Unix-like
		return linuxPaths(isRoot)
	}
}

func linuxPaths(isRoot bool) PathConfig {
	if isRoot {
		return PathConfig{
			ConfigDir:   fmt.Sprintf("/etc/%s/%s", projectOrg, projectName),
			ConfigFile:  fmt.Sprintf("/etc/%s/%s/server.yml", projectOrg, projectName),
			DataDir:     fmt.Sprintf("/var/lib/%s/%s", projectOrg, projectName),
			CacheDir:    fmt.Sprintf("/var/cache/%s/%s", projectOrg, projectName),
			LogDir:      fmt.Sprintf("/var/log/%s/%s", projectOrg, projectName),
			LogFile:     fmt.Sprintf("/var/log/%s/%s/server.log", projectOrg, projectName),
			BackupDir:   fmt.Sprintf("/mnt/Backups/%s/%s", projectOrg, projectName),
			PIDFile:     fmt.Sprintf("/var/run/%s/%s.pid", projectOrg, projectName),
			SSLDir:      fmt.Sprintf("/etc/%s/%s/ssl", projectOrg, projectName),
			SecurityDir: fmt.Sprintf("/etc/%s/%s/security", projectOrg, projectName),
			DBDir:       fmt.Sprintf("/var/lib/%s/%s/db", projectOrg, projectName),
		}
	}

	homeDir := userHomeDir()
	return PathConfig{
		ConfigDir:   filepath.Join(homeDir, ".config", projectOrg, projectName),
		ConfigFile:  filepath.Join(homeDir, ".config", projectOrg, projectName, "server.yml"),
		DataDir:     filepath.Join(homeDir, ".local", "share", projectOrg, projectName),
		CacheDir:    filepath.Join(homeDir, ".cache", projectOrg, projectName),
		LogDir:      filepath.Join(homeDir, ".local", "log", projectOrg, projectName),
		LogFile:     filepath.Join(homeDir, ".local", "log", projectOrg, projectName, "server.log"),
		BackupDir:   filepath.Join(homeDir, ".local", "share", "Backups", projectOrg, projectName),
		PIDFile:     filepath.Join(homeDir, ".local", "share", projectOrg, projectName, projectName+".pid"),
		SSLDir:      filepath.Join(homeDir, ".config", projectOrg, projectName, "ssl"),
		SecurityDir: filepath.Join(homeDir, ".config", projectOrg, projectName, "security"),
		DBDir:       filepath.Join(homeDir, ".local", "share", projectOrg, projectName, "db"),
	}
}

func darwinPaths(isRoot bool) PathConfig {
	if isRoot {
		appSupport := fmt.Sprintf("/Library/Application Support/%s/%s", projectOrg, projectName)
		return PathConfig{
			ConfigDir:   appSupport,
			ConfigFile:  filepath.Join(appSupport, "server.yml"),
			DataDir:     filepath.Join(appSupport, "data"),
			CacheDir:    fmt.Sprintf("/Library/Caches/%s/%s", projectOrg, projectName),
			LogDir:      fmt.Sprintf("/Library/Logs/%s/%s", projectOrg, projectName),
			LogFile:     fmt.Sprintf("/Library/Logs/%s/%s/server.log", projectOrg, projectName),
			BackupDir:   fmt.Sprintf("/Library/Backups/%s/%s", projectOrg, projectName),
			PIDFile:     fmt.Sprintf("/var/run/%s/%s.pid", projectOrg, projectName),
			SSLDir:      filepath.Join(appSupport, "ssl"),
			SecurityDir: filepath.Join(appSupport, "security"),
			DBDir:       filepath.Join(appSupport, "db"),
		}
	}

	homeDir := userHomeDir()
	appSupport := filepath.Join(homeDir, "Library", "Application Support", projectOrg, projectName)
	return PathConfig{
		ConfigDir:   appSupport,
		ConfigFile:  filepath.Join(appSupport, "server.yml"),
		DataDir:     appSupport,
		CacheDir:    filepath.Join(homeDir, "Library", "Caches", projectOrg, projectName),
		LogDir:      filepath.Join(homeDir, "Library", "Logs", projectOrg, projectName),
		LogFile:     filepath.Join(homeDir, "Library", "Logs", projectOrg, projectName, "server.log"),
		BackupDir:   filepath.Join(homeDir, "Library", "Backups", projectOrg, projectName),
		PIDFile:     filepath.Join(appSupport, projectName+".pid"),
		SSLDir:      filepath.Join(appSupport, "ssl"),
		SecurityDir: filepath.Join(appSupport, "security"),
		DBDir:       filepath.Join(appSupport, "db"),
	}
}

func bsdPaths(isRoot bool) PathConfig {
	if isRoot {
		return PathConfig{
			ConfigDir:   fmt.Sprintf("/usr/local/etc/%s/%s", projectOrg, projectName),
			ConfigFile:  fmt.Sprintf("/usr/local/etc/%s/%s/server.yml", projectOrg, projectName),
			DataDir:     fmt.Sprintf("/var/db/%s/%s", projectOrg, projectName),
			CacheDir:    fmt.Sprintf("/var/cache/%s/%s", projectOrg, projectName),
			LogDir:      fmt.Sprintf("/var/log/%s/%s", projectOrg, projectName),
			LogFile:     fmt.Sprintf("/var/log/%s/%s/server.log", projectOrg, projectName),
			BackupDir:   fmt.Sprintf("/var/backups/%s/%s", projectOrg, projectName),
			PIDFile:     fmt.Sprintf("/var/run/%s/%s.pid", projectOrg, projectName),
			SSLDir:      fmt.Sprintf("/usr/local/etc/%s/%s/ssl", projectOrg, projectName),
			SecurityDir: fmt.Sprintf("/usr/local/etc/%s/%s/security", projectOrg, projectName),
			DBDir:       fmt.Sprintf("/var/db/%s/%s/db", projectOrg, projectName),
		}
	}

	// BSD user paths same as Linux
	homeDir := userHomeDir()
	return PathConfig{
		ConfigDir:   filepath.Join(homeDir, ".config", projectOrg, projectName),
		ConfigFile:  filepath.Join(homeDir, ".config", projectOrg, projectName, "server.yml"),
		DataDir:     filepath.Join(homeDir, ".local", "share", projectOrg, projectName),
		CacheDir:    filepath.Join(homeDir, ".cache", projectOrg, projectName),
		LogDir:      filepath.Join(homeDir, ".local", "log", projectOrg, projectName),
		LogFile:     filepath.Join(homeDir, ".local", "log", projectOrg, projectName, "server.log"),
		BackupDir:   filepath.Join(homeDir, ".local", "share", "Backups", projectOrg, projectName),
		PIDFile:     filepath.Join(homeDir, ".local", "share", projectOrg, projectName, projectName+".pid"),
		SSLDir:      filepath.Join(homeDir, ".config", projectOrg, projectName, "ssl"),
		SecurityDir: filepath.Join(homeDir, ".config", projectOrg, projectName, "security"),
		DBDir:       filepath.Join(homeDir, ".local", "share", projectOrg, projectName, "db"),
	}
}

func windowsPaths(isRoot bool) PathConfig {
	if isRoot {
		programData := os.Getenv("ProgramData")
		if programData == "" {
			programData = `C:\ProgramData`
		}
		base := filepath.Join(programData, projectOrg, projectName)
		return PathConfig{
			ConfigDir:   base,
			ConfigFile:  filepath.Join(base, "server.yml"),
			DataDir:     filepath.Join(base, "data"),
			CacheDir:    filepath.Join(base, "cache"),
			LogDir:      filepath.Join(base, "logs"),
			LogFile:     filepath.Join(base, "logs", "server.log"),
			BackupDir:   filepath.Join(programData, "Backups", projectOrg, projectName),
			PIDFile:     "",
			SSLDir:      filepath.Join(base, "ssl"),
			SecurityDir: filepath.Join(base, "security"),
			DBDir:       filepath.Join(base, "db"),
		}
	}

	appData := os.Getenv("AppData")
	localAppData := os.Getenv("LocalAppData")
	if appData == "" {
		homeDir := userHomeDir()
		appData = filepath.Join(homeDir, "AppData", "Roaming")
	}
	if localAppData == "" {
		homeDir := userHomeDir()
		localAppData = filepath.Join(homeDir, "AppData", "Local")
	}

	return PathConfig{
		ConfigDir:   filepath.Join(appData, projectOrg, projectName),
		ConfigFile:  filepath.Join(appData, projectOrg, projectName, "server.yml"),
		DataDir:     filepath.Join(localAppData, projectOrg, projectName),
		CacheDir:    filepath.Join(localAppData, projectOrg, projectName, "cache"),
		LogDir:      filepath.Join(localAppData, projectOrg, projectName, "logs"),
		LogFile:     filepath.Join(localAppData, projectOrg, projectName, "logs", "server.log"),
		BackupDir:   filepath.Join(localAppData, "Backups", projectOrg, projectName),
		PIDFile:     "",
		SSLDir:      filepath.Join(appData, projectOrg, projectName, "ssl"),
		SecurityDir: filepath.Join(appData, projectOrg, projectName, "security"),
		DBDir:       filepath.Join(localAppData, projectOrg, projectName, "db"),
	}
}

func containerPaths() PathConfig {
	return PathConfig{
		ConfigDir:   fmt.Sprintf("/config/%s", projectName),
		ConfigFile:  fmt.Sprintf("/config/%s/server.yml", projectName),
		DataDir:     fmt.Sprintf("/data/%s", projectName),
		CacheDir:    fmt.Sprintf("/data/%s/cache", projectName),
		LogDir:      fmt.Sprintf("/data/log/%s", projectName),
		LogFile:     fmt.Sprintf("/data/log/%s/server.log", projectName),
		BackupDir:   fmt.Sprintf("/data/backups/%s", projectName),
		PIDFile:     fmt.Sprintf("/var/run/%s/%s.pid", projectOrg, projectName),
		SSLDir:      fmt.Sprintf("/config/%s/ssl", projectName),
		SecurityDir: fmt.Sprintf("/config/%s/security", projectName),
		DBDir:       "/data/db/sqlite",
	}
}

// IsRunningAsRoot returns true if the process is running with root/admin privileges
func IsRunningAsRoot() bool {
	if runtime.GOOS == "windows" {
		// On Windows, check if running as administrator
		// Simplified check - full implementation would use Windows APIs
		return os.Getenv("USERNAME") == "SYSTEM" || os.Getenv("USERNAME") == "Administrator"
	}
	return os.Getuid() == 0
}

// IsRunningInContainer returns true if running inside a Docker/container environment
func IsRunningInContainer() bool {
	// Check for /.dockerenv
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check cgroup for docker/lxc/containerd
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "docker") ||
			strings.Contains(content, "lxc") ||
			strings.Contains(content, "containerd") {
			return true
		}
	}

	// Check for container env var
	if os.Getenv("container") != "" {
		return true
	}

	return false
}

// EnsureAllDirs creates all required directories with proper permissions
func EnsureAllDirs(pc PathConfig, isRoot bool) error {
	dirs := []string{
		pc.ConfigDir,
		pc.DataDir,
		pc.CacheDir,
		pc.LogDir,
		pc.BackupDir,
		pc.SSLDir,
		pc.SecurityDir,
		pc.DBDir,
	}

	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if err := EnsureDir(dir, isRoot); err != nil {
			return err
		}
	}

	// Create PID file directory if needed
	if pc.PIDFile != "" {
		pidDir := filepath.Dir(pc.PIDFile)
		if err := EnsureDir(pidDir, isRoot); err != nil {
			return err
		}
	}

	return nil
}

// EnsureDir creates directory with proper permissions if it doesn't exist
func EnsureDir(path string, isRoot bool) error {
	perm := os.FileMode(0700)
	if isRoot {
		perm = 0755
	}

	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	return nil
}

func userHomeDir() string {
	if homeDir, err := os.UserHomeDir(); err == nil {
		return homeDir
	}
	if u, err := user.Current(); err == nil {
		return u.HomeDir
	}
	return os.Getenv("HOME")
}
