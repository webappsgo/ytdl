// Package main provides the companion ytdl CLI client.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Build info - set via -ldflags at build time.
var (
	Version      = "dev"
	CommitID     = "unknown"
	BuildDate    = "unknown"
	OfficialSite = ""
)

type clientConfig struct {
	serverURL string
	token     string
	output    string
	timeout   time.Duration
}

type healthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version,omitempty"`
	GoVersion string `json:"go_version,omitempty"`
	Uptime    string `json:"uptime,omitempty"`
}

type versionResponse struct {
	Version   string `json:"version"`
	CommitID  string `json:"commit_id"`
	BuildDate string `json:"build_date"`
}

func main() {
	binaryName := filepath.Base(os.Args[0])

	cfg, command, err := parseArgs(binaryName, os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		printHelp(binaryName)
		os.Exit(1)
	}

	switch command {
	case "":
		printHelp(binaryName)
	case "health":
		if err := runHealth(binaryName, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "version":
		if err := runVersion(binaryName, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printHelp(binaryName)
		os.Exit(1)
	}
}

func parseArgs(binaryName string, args []string) (clientConfig, string, error) {
	var cfg clientConfig

	fs := flag.NewFlagSet(binaryName, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&cfg.serverURL, "server", "", "Server URL")
	fs.StringVar(&cfg.token, "token", "", "API token")
	fs.StringVar(&cfg.output, "output", "text", "Output format: text or json")
	fs.DurationVar(&cfg.timeout, "timeout", 10*time.Second, "HTTP timeout")

	showHelp := fs.Bool("help", false, "Show help")
	showVersion := fs.Bool("version", false, "Show version")

	if err := fs.Parse(args); err != nil {
		return cfg, "", err
	}

	if *showHelp {
		printHelp(binaryName)
		os.Exit(0)
	}

	if *showVersion {
		printClientVersion(binaryName)
		os.Exit(0)
	}

	if cfg.output != "text" && cfg.output != "json" {
		return cfg, "", fmt.Errorf("invalid --output value %q (must be text or json)", cfg.output)
	}

	command := ""
	if fs.NArg() > 0 {
		command = fs.Arg(0)
	}

	return cfg, command, nil
}

func printClientVersion(binaryName string) {
	fmt.Printf("%s %s (%s)\n", binaryName, displayVersion(Version), CommitID)
	fmt.Printf("Built: %s\n", BuildDate)
	if OfficialSite != "" {
		fmt.Printf("Site:  %s\n", OfficialSite)
	}
}

func printHelp(binaryName string) {
	fmt.Printf("%s %s - Companion CLI for the ytdl server\n", binaryName, displayVersion(Version))
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("  %s [flags] <command>\n", binaryName)
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  health      Check server health")
	fmt.Println("  version     Show remote server version")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --server URL        Server base URL (defaults to embedded official site when available)")
	fmt.Println("  --token TOKEN       API token for authenticated requests")
	fmt.Println("  --output FORMAT     Output format: text or json (default: text)")
	fmt.Println("  --timeout DURATION  HTTP timeout (default: 10s)")
	fmt.Println("  --help              Show help")
	fmt.Println("  --version           Show client version")
	fmt.Println()
	fmt.Printf("Examples:\n")
	fmt.Printf("  %s --server https://dl.csj.rocks health\n", binaryName)
	fmt.Printf("  %s --server https://dl.csj.rocks --output json version\n", binaryName)
}

func runHealth(binaryName string, cfg clientConfig) error {
	serverURL, err := resolveServerURL(cfg.serverURL)
	if err != nil {
		return err
	}

	endpoint := serverURL + "/api/v1/healthz?detail=true"
	body, err := doRequest(binaryName, endpoint, cfg)
	if err != nil {
		return err
	}

	if cfg.output == "json" {
		fmt.Println(string(body))
		return nil
	}

	var resp healthResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parsing health response: %w", err)
	}

	fmt.Printf("Status:  %s\n", defaultString(resp.Status, "unknown"))
	if resp.Version != "" {
		fmt.Printf("Version: %s\n", resp.Version)
	}
	if resp.Uptime != "" {
		fmt.Printf("Uptime:  %s\n", resp.Uptime)
	}
	if resp.GoVersion != "" {
		fmt.Printf("Go:      %s\n", resp.GoVersion)
	}

	return nil
}

func runVersion(binaryName string, cfg clientConfig) error {
	serverURL, err := resolveServerURL(cfg.serverURL)
	if err != nil {
		return err
	}

	endpoint := serverURL + "/api/v1/version"
	body, err := doRequest(binaryName, endpoint, cfg)
	if err != nil {
		return err
	}

	if cfg.output == "json" {
		fmt.Println(string(body))
		return nil
	}

	var resp versionResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parsing version response: %w", err)
	}

	fmt.Printf("Version: %s\n", defaultString(resp.Version, "unknown"))
	if resp.CommitID != "" {
		fmt.Printf("Commit:  %s\n", resp.CommitID)
	}
	if resp.BuildDate != "" {
		fmt.Printf("Built:   %s\n", resp.BuildDate)
	}

	return nil
}

func resolveServerURL(flagValue string) (string, error) {
	serverURL := strings.TrimSpace(flagValue)
	if serverURL == "" {
		serverURL = strings.TrimSpace(OfficialSite)
	}
	if serverURL == "" {
		return "", errors.New("missing server URL; use --server or build with an embedded official site")
	}
	return strings.TrimRight(serverURL, "/"), nil
}

func doRequest(binaryName, endpoint string, cfg clientConfig) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", binaryName, Version))
	if cfg.token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.token)
	}

	client := &http.Client{Timeout: cfg.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	body, err := ioReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return body, nil
}

func displayVersion(version string) string {
	if version != "" && version[0] >= '0' && version[0] <= '9' && strings.Contains(version, ".") {
		return "v" + version
	}
	return version
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func ioReadAll(body io.Reader) ([]byte, error) {
	return io.ReadAll(body)
}
