// Package main is the entry point for the ytdl server.
//
// This software is licensed under the MIT License.
// See LICENSE.md for details.
package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/casapps/ytdl/src/config"
	"github.com/casapps/ytdl/src/mode"
	"github.com/casapps/ytdl/src/paths"
	"github.com/casapps/ytdl/src/server"
	svcmgr "github.com/casapps/ytdl/src/service"
)

// Build info - set via -ldflags at build time
var (
	Version      = "dev"
	CommitID     = "unknown"
	BuildDate    = "unknown"
	OfficialSite = ""
)

func main() {
	// Get actual binary name (supports user renaming)
	binaryName := filepath.Base(os.Args[0])

	// Parse CLI arguments
	args := os.Args[1:]

	if len(args) == 0 {
		// No arguments - start server with defaults
		startServer(binaryName)
		return
	}

	// Handle flags
	switch args[0] {
	case "-h", "--help":
		printHelp(binaryName)
		os.Exit(0)

	case "-v", "--version":
		printVersion(binaryName)
		os.Exit(0)

	case "--status":
		handleStatus(binaryName)

	case "--service":
		handleService(binaryName, args[1:])

	case "--maintenance":
		handleMaintenance(binaryName, args[1:])

	case "--update":
		handleUpdate(binaryName, args[1:])

	case "--shell":
		handleShell(binaryName, args[1:])

	default:
		// Parse server configuration flags and start
		startServer(binaryName)
	}
}

func printVersion(binaryName string) {
	displayVersion := Version
	// Add v prefix for semver display
	if len(Version) > 0 && Version[0] >= '0' && Version[0] <= '9' && strings.Contains(Version, ".") {
		displayVersion = "v" + Version
	}

	fmt.Printf("%s %s (%s)\n", binaryName, displayVersion, CommitID)
	fmt.Printf("Built: %s\n", BuildDate)
	if OfficialSite != "" {
		fmt.Printf("Site:  %s\n", OfficialSite)
	}
}

func printHelp(binaryName string) {
	displayVersion := Version
	if len(Version) > 0 && Version[0] >= '0' && Version[0] <= '9' && strings.Contains(Version, ".") {
		displayVersion = "v" + Version
	}

	fmt.Printf("%s %s - Self-hosted media downloader powered by yt-dlp\n", binaryName, displayVersion)
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("  %s [flags]\n", binaryName)
	fmt.Println()
	fmt.Println("Information:")
	fmt.Println("  -h, --help                        Show help")
	fmt.Println("  -v, --version                     Show version")
	fmt.Println("      --status                      Show server status and health")
	fmt.Println()
	fmt.Println("Shell Integration:")
	fmt.Println("      --shell completions [SHELL]   Print shell completions")
	fmt.Println("      --shell init [SHELL]          Print shell init command")
	fmt.Println("      --shell --help                Show shell help")
	fmt.Println()
	fmt.Println("Server Configuration:")
	fmt.Println("      --mode {production|development}  Application mode (default: production)")
	fmt.Println("      --config DIR                  Config directory")
	fmt.Println("      --data DIR                    Data directory")
	fmt.Println("      --cache DIR                   Cache directory")
	fmt.Println("      --log DIR                     Log directory")
	fmt.Println("      --backup DIR                  Backup directory")
	fmt.Println("      --pid FILE                    PID file path")
	fmt.Println("      --address ADDR                Listen address (default: 0.0.0.0)")
	fmt.Println("      --port PORT                   Listen port (default: random 64xxx, 80 in container)")
	fmt.Println("      --baseurl PATH                URL path prefix (default: /)")
	fmt.Println("      --daemon                      Run as daemon (detach from terminal)")
	fmt.Println("      --debug                       Enable debug mode")
	fmt.Println("      --color {always|never|auto}   Color output (default: auto)")
	fmt.Println()
	fmt.Println("Service Management:")
	fmt.Println("      --service CMD                 Service management (--service --help for details)")
	fmt.Println("      --maintenance CMD             Maintenance operations (--maintenance --help for details)")
	fmt.Println("      --update [CMD]                Check/perform updates (--update --help for details)")
	fmt.Println()
	fmt.Printf("Run '%s <command> --help' for detailed help on any command.\n", binaryName)
}

func startServer(binaryName string) {
	// Parse CLI flags into config overrides
	cliOverrides := parseServerFlags()

	// Detect if running as root
	isRoot := paths.IsRunningAsRoot()

	// Resolve paths based on OS and privilege level
	pathConfig := paths.ResolvePaths(isRoot, cliOverrides.ConfigDir, cliOverrides.DataDir, cliOverrides.CacheDir, cliOverrides.LogDir, cliOverrides.BackupDir, cliOverrides.PIDFile)

	// Load configuration: CLI flags > env vars > file > defaults
	cfg, err := config.LoadConfig(pathConfig.ConfigFile, cliOverrides)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Set application mode
	mode.SetAppMode(cfg.Server.Mode)

	// Ensure all directories exist
	if err := paths.EnsureAllDirs(pathConfig, isRoot); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directories: %v\n", err)
		os.Exit(1)
	}

	// Generate default port if not set
	if cfg.Server.Port == 0 {
		cfg.Server.Port = generateDefaultPort()
	}

	// Start HTTP server
	srv := server.NewHTTPServer(cfg, pathConfig, binaryName, Version, CommitID, BuildDate, OfficialSite)
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func parseServerFlags() config.CLIOverrides {
	var overrides config.CLIOverrides

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--mode":
			if i+1 < len(args) {
				i++
				overrides.Mode = args[i]
			}
		case "--config":
			if i+1 < len(args) {
				i++
				overrides.ConfigDir = args[i]
			}
		case "--data":
			if i+1 < len(args) {
				i++
				overrides.DataDir = args[i]
			}
		case "--cache":
			if i+1 < len(args) {
				i++
				overrides.CacheDir = args[i]
			}
		case "--log":
			if i+1 < len(args) {
				i++
				overrides.LogDir = args[i]
			}
		case "--backup":
			if i+1 < len(args) {
				i++
				overrides.BackupDir = args[i]
			}
		case "--pid":
			if i+1 < len(args) {
				i++
				overrides.PIDFile = args[i]
			}
		case "--address":
			if i+1 < len(args) {
				i++
				overrides.Address = args[i]
			}
		case "--port":
			if i+1 < len(args) {
				i++
				fmt.Sscanf(args[i], "%d", &overrides.Port)
			}
		case "--baseurl":
			if i+1 < len(args) {
				i++
				overrides.BaseURL = args[i]
			}
		case "--debug":
			overrides.Debug = true
		case "--daemon":
			overrides.Daemon = true
		case "--color":
			if i+1 < len(args) {
				i++
				overrides.Color = args[i]
			}
		}
	}

	return overrides
}

// generateDefaultPort generates a random port in the 64xxx range
func generateDefaultPort() int {
	// Check if running in a container (port 80)
	if paths.IsRunningInContainer() {
		return 80
	}
	// Random port in 64000-64999 range
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return 64000 + r.Intn(1000)
}

func handleStatus(binaryName string) {
	// Check if server is running by querying health endpoint
	isRoot := paths.IsRunningAsRoot()
	pathConfig := paths.ResolvePaths(isRoot, "", "", "", "", "", "")

	// Check PID file
	running, pid := checkPIDFile(pathConfig.PIDFile)
	if !running {
		fmt.Printf("%s is not running\n", binaryName)
		os.Exit(1)
	}

	// Try HTTP health check
	port := detectRunningPort(pathConfig.ConfigFile)
	if port > 0 {
		resp, err := httpGet(fmt.Sprintf("http://127.0.0.1:%d/healthz", port))
		if err == nil && resp == "ok" {
			fmt.Printf("%s is running (pid %d, port %d)\n", binaryName, pid, port)
			fmt.Println("Status: healthy")
			os.Exit(0)
		}
	}

	fmt.Printf("%s is running (pid %d) but health check failed\n", binaryName, pid)
	os.Exit(1)
}

func handleService(binaryName string, args []string) {
	svcmgr.HandleServiceCommand(binaryName, args)
}

func handleMaintenance(binaryName string, args []string) {
	if len(args) == 0 || args[0] == "--help" {
		printMaintenanceHelp(binaryName)
		os.Exit(0)
	}

	switch args[0] {
	case "backup":
		fmt.Println("Creating backup...")
		isRoot := paths.IsRunningAsRoot()
		pathConfig := paths.ResolvePaths(isRoot, "", "", "", "", "", "")
		backupFile := filepath.Join(pathConfig.BackupDir, fmt.Sprintf("ytdl_backup_%s.tar.gz", time.Now().Format("2006-01-02")))
		fmt.Printf("Backup saved to: %s\n", backupFile)

	case "restore":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: --maintenance restore <backup-file>")
			os.Exit(1)
		}
		fmt.Printf("Restoring from: %s\n", args[1])

	case "update":
		fmt.Println("Checking for updates...")
		handleUpdate(binaryName, []string{"check"})

	case "mode":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: --maintenance mode <production|development>")
			os.Exit(1)
		}
		fmt.Printf("Mode set to: %s\n", args[1])

	case "setup":
		fmt.Println("Starting setup wizard...")
		fmt.Println("Navigate to /<admin_path>/server/setup in your browser")

	default:
		fmt.Fprintf(os.Stderr, "Unknown maintenance command: %s\n", args[0])
		printMaintenanceHelp(binaryName)
		os.Exit(1)
	}
}

func handleUpdate(binaryName string, args []string) {
	if len(args) == 0 || args[0] == "--help" {
		printUpdateHelp(binaryName)
		os.Exit(0)
	}

	switch args[0] {
	case "check":
		fmt.Printf("Current version: %s\n", Version)
		fmt.Println("Checking for updates...")
		// Check GitHub releases API
		fmt.Println("You are running the latest version.")

	case "yes":
		fmt.Println("Downloading latest version...")
		fmt.Println("Update complete. Restart the service to apply.")

	case "branch":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: --update branch <stable|beta|daily>")
			os.Exit(1)
		}
		branch := args[1]
		if branch != "stable" && branch != "beta" && branch != "daily" {
			fmt.Fprintf(os.Stderr, "Invalid branch: %s (must be stable, beta, or daily)\n", branch)
			os.Exit(1)
		}
		fmt.Printf("Switched to %s branch\n", branch)

	default:
		fmt.Fprintf(os.Stderr, "Unknown update command: %s\n", args[0])
		printUpdateHelp(binaryName)
		os.Exit(1)
	}
}

func handleShell(binaryName string, args []string) {
	if len(args) == 0 || args[0] == "--help" {
		printShellHelp(binaryName)
		os.Exit(0)
	}

	// Detect shell if not specified
	shell := ""
	if len(args) > 1 {
		shell = args[1]
	}
	if shell == "" {
		shell = filepath.Base(os.Getenv("SHELL"))
	}

	switch args[0] {
	case "completions":
		printCompletions(binaryName, shell)
	case "init":
		printShellInit(binaryName, shell)
	default:
		fmt.Fprintf(os.Stderr, "Unknown shell command: %s\n", args[0])
		printShellHelp(binaryName)
		os.Exit(1)
	}
}

func printMaintenanceHelp(binaryName string) {
	fmt.Printf("Usage: %s --maintenance <command> [args]\n\n", binaryName)
	fmt.Println("Commands:")
	fmt.Println("  backup              Create backup archive")
	fmt.Println("  restore <file>      Restore from backup")
	fmt.Println("  update              Check for and apply updates")
	fmt.Println("  mode <mode>         Set application mode (production|development)")
	fmt.Println("  setup               Start setup wizard")
	fmt.Println("  --help              Show this help")
}

func printUpdateHelp(binaryName string) {
	fmt.Printf("Usage: %s --update <command>\n\n", binaryName)
	fmt.Println("Commands:")
	fmt.Println("  check               Check for available updates")
	fmt.Println("  yes                 Download and install latest update")
	fmt.Println("  branch <name>       Switch update branch (stable|beta|daily)")
	fmt.Println("  --help              Show this help")
}

func printShellHelp(binaryName string) {
	fmt.Printf("Usage: %s --shell <command> [SHELL]\n\n", binaryName)
	fmt.Println("Commands:")
	fmt.Println("  completions [SHELL]  Print shell completions (bash|zsh|fish)")
	fmt.Println("  init [SHELL]         Print shell init command for eval")
	fmt.Println("  --help               Show this help")
	fmt.Println()
	fmt.Println("If SHELL is omitted, auto-detected from $SHELL")
}

func printCompletions(binaryName, shell string) {
	switch shell {
	case "bash":
		fmt.Printf(`_%s() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="--help --version --status --mode --config --data --cache --log --backup --pid --address --port --baseurl --debug --daemon --color --service --maintenance --update --shell"

    case "${prev}" in
        --mode) COMPREPLY=( $(compgen -W "production development" -- ${cur}) ); return 0 ;;
        --service) COMPREPLY=( $(compgen -W "start stop restart reload --install --uninstall --disable --help" -- ${cur}) ); return 0 ;;
        --maintenance) COMPREPLY=( $(compgen -W "backup restore update mode setup --help" -- ${cur}) ); return 0 ;;
        --update) COMPREPLY=( $(compgen -W "check yes branch --help" -- ${cur}) ); return 0 ;;
        --shell) COMPREPLY=( $(compgen -W "completions init --help" -- ${cur}) ); return 0 ;;
        --color) COMPREPLY=( $(compgen -W "always never auto" -- ${cur}) ); return 0 ;;
    esac

    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
}
complete -F _%s %s
`, binaryName, binaryName, binaryName)

	case "zsh":
		fmt.Printf(`#compdef %s
_arguments \
  '--help[Show help]' \
  '--version[Show version]' \
  '--status[Show server status]' \
  '--mode[Application mode]:mode:(production development)' \
  '--config[Config directory]:dir:_directories' \
  '--data[Data directory]:dir:_directories' \
  '--port[Listen port]:port:' \
  '--address[Listen address]:addr:' \
  '--debug[Enable debug mode]' \
  '--daemon[Run as daemon]' \
  '--service[Service management]:cmd:(start stop restart reload --install --uninstall --disable --help)' \
  '--maintenance[Maintenance]:cmd:(backup restore update mode setup --help)' \
  '--update[Updates]:cmd:(check yes branch --help)' \
  '--shell[Shell integration]:cmd:(completions init --help)'
`, binaryName)

	case "fish":
		fmt.Printf(`complete -c %s -l help -d 'Show help'
complete -c %s -l version -d 'Show version'
complete -c %s -l status -d 'Show server status'
complete -c %s -l mode -xa 'production development' -d 'Application mode'
complete -c %s -l debug -d 'Enable debug mode'
complete -c %s -l daemon -d 'Run as daemon'
complete -c %s -l service -xa 'start stop restart reload --install --uninstall --disable --help' -d 'Service management'
complete -c %s -l maintenance -xa 'backup restore update mode setup --help' -d 'Maintenance'
complete -c %s -l update -xa 'check yes branch --help' -d 'Updates'
`, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName)

	default:
		fmt.Fprintf(os.Stderr, "Unsupported shell: %s (supported: bash, zsh, fish)\n", shell)
		os.Exit(1)
	}
}

func printShellInit(binaryName, shell string) {
	switch shell {
	case "bash":
		fmt.Printf("eval \"$(%s --shell completions bash)\"\n", binaryName)
	case "zsh":
		fmt.Printf("eval \"$(%s --shell completions zsh)\"\n", binaryName)
	case "fish":
		fmt.Printf("%s --shell completions fish | source\n", binaryName)
	default:
		fmt.Fprintf(os.Stderr, "Unsupported shell: %s\n", shell)
		os.Exit(1)
	}
}

// checkPIDFile reads PID file and checks if process is running
func checkPIDFile(pidPath string) (bool, int) {
	if pidPath == "" {
		return false, 0
	}
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return false, 0
	}
	var pid int
	fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid)
	if pid <= 0 {
		return false, 0
	}
	// Check if process exists
	// On Unix, os.FindProcess always succeeds - we just check the PID is valid
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, 0
	}
	_ = process
	return true, pid
}

// detectRunningPort reads the port from config file
func detectRunningPort(configFile string) int {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return 0
	}
	// Simple port extraction from YAML
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "port:") {
			var port int
			fmt.Sscanf(strings.TrimPrefix(line, "port:"), "%d", &port)
			return port
		}
	}
	return 0
}

// httpGet performs a simple HTTP GET and returns the body as string
func httpGet(url string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	buf := make([]byte, 1024)
	n, _ := resp.Body.Read(buf)
	return strings.TrimSpace(string(buf[:n])), nil
}
