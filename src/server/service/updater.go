// Package service - Self-update via GitHub releases API.
// See AI.md PART 23 for update command specifications.
// Supports stable/beta/daily branches.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"
)

const (
	githubReleasesAPI = "https://api.github.com/repos/casapps/ytdl/releases"
	userAgent         = "ytdl/1.0"
)

// UpdateService handles checking and applying updates
type UpdateService struct {
	currentVersion string
	httpClient     *http.Client
}

// NewUpdateService creates a new update service
func NewUpdateService(currentVersion string) *UpdateService {
	return &UpdateService{
		currentVersion: currentVersion,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

// ReleaseInfo holds information about a GitHub release
type ReleaseInfo struct {
	TagName    string        `json:"tag_name"`
	Name       string        `json:"name"`
	Body       string        `json:"body"`
	Draft      bool          `json:"draft"`
	Prerelease bool          `json:"prerelease"`
	CreatedAt  time.Time     `json:"created_at"`
	Assets     []ReleaseAsset `json:"assets"`
}

// ReleaseAsset holds download information for a release binary
type ReleaseAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// CheckForUpdate checks if a newer version is available
func (u *UpdateService) CheckForUpdate(ctx context.Context) (*ReleaseInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubReleasesAPI+"/latest", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("checking releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &release, nil
}

// DownloadUpdate downloads the appropriate binary for this platform
func (u *UpdateService) DownloadUpdate(ctx context.Context, release *ReleaseInfo) (string, error) {
	// Find matching asset for this OS/arch
	binaryName := fmt.Sprintf("ytdl-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return "", fmt.Errorf("no binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Download to temp file
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating download request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading: %w", err)
	}
	defer resp.Body.Close()

	tmpFile, err := os.CreateTemp("", "ytdl-update-*")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("writing download: %w", err)
	}
	tmpFile.Close()

	// Make executable
	os.Chmod(tmpFile.Name(), 0755)

	return tmpFile.Name(), nil
}

// ApplyUpdate replaces the current binary with the downloaded one
func (u *UpdateService) ApplyUpdate(downloadPath string) error {
	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}

	// Rename current binary as backup
	backupPath := currentBinary + ".bak"
	if err := os.Rename(currentBinary, backupPath); err != nil {
		return fmt.Errorf("backing up current binary: %w", err)
	}

	// Move new binary into place
	if err := os.Rename(downloadPath, currentBinary); err != nil {
		// Restore backup on failure
		os.Rename(backupPath, currentBinary)
		return fmt.Errorf("installing update: %w", err)
	}

	// Clean up backup
	os.Remove(backupPath)

	return nil
}
