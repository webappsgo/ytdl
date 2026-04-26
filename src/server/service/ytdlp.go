// Package service contains business logic.
// ytdlp.go wraps the yt-dlp binary for media extraction and downloading.
package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// YTDLPService wraps yt-dlp binary operations
type YTDLPService struct {
	// Path to yt-dlp binary
	binaryPath string
	// Path to ffmpeg binary
	ffmpegPath string
}

// NewYTDLPService creates a new yt-dlp service
func NewYTDLPService(binaryPath, ffmpegPath string) *YTDLPService {
	return &YTDLPService{
		binaryPath: binaryPath,
		ffmpegPath: ffmpegPath,
	}
}

// MediaInfo holds metadata extracted from a URL
type MediaInfo struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	Uploader     string `json:"uploader"`
	UploaderURL  string `json:"uploader_url"`
	ThumbnailURL string `json:"thumbnail"`
	Duration     int    `json:"duration"`
	WebpageURL   string `json:"webpage_url"`
	Extractor    string `json:"extractor"`
	IsPlaylist   bool   `json:"_type"`
	PlaylistCount int   `json:"playlist_count"`
}

// ProgressUpdate represents a download progress event
type ProgressUpdate struct {
	DownloadID int64
	Status     string
	Percent    float64
	Speed      string
	ETA        string
	FileSize   int64
}

// DownloadOptions configures a download
type DownloadOptions struct {
	URL               string
	Format            string
	Quality           string
	Bitrate           string
	OutputPath        string
	AudioOnly         bool
	SubtitleLanguages []string
	EmbedSubtitles    bool
	EmbedLyrics       bool
	EmbedThumbnail    bool
	NormalizeAudio    bool
	TrimSilence       bool
	ProxyURL          string
	CookieFile        string
	SpeedLimit        string
}

// progressRegex matches yt-dlp download progress lines
var progressRegex = regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%\s+of\s+~?\s*(\S+)\s+at\s+(\S+)\s+ETA\s+(\S+)`)

// ExtractMediaInfo gets metadata for a URL without downloading
func (y *YTDLPService) ExtractMediaInfo(ctx context.Context, url string) (*MediaInfo, error) {
	args := []string{
		"--dump-json",
		"--no-download",
		"--no-playlist",
		"--no-warnings",
		url,
	}

	cmd := exec.CommandContext(ctx, y.binaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("extracting info: %w", err)
	}

	var info MediaInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("parsing info: %w", err)
	}

	return &info, nil
}

// CheckIsPlaylist checks if a URL is a playlist
func (y *YTDLPService) CheckIsPlaylist(ctx context.Context, url string) (bool, int, error) {
	args := []string{
		"--flat-playlist",
		"--dump-json",
		"--no-download",
		"--no-warnings",
		url,
	}

	cmd := exec.CommandContext(ctx, y.binaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		// Not a playlist
		return false, 0, nil
	}

	// Count entries (one JSON per line)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) <= 1 {
		return false, 0, nil
	}

	return true, len(lines), nil
}

// Download executes a download with progress reporting
func (y *YTDLPService) Download(ctx context.Context, opts DownloadOptions, progressCh chan<- ProgressUpdate, downloadID int64) error {
	args := y.buildDownloadArgs(opts)

	cmd := exec.CommandContext(ctx, y.binaryPath, args...)

	// Capture stderr for progress
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}

	// Capture stdout for info
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting yt-dlp: %w", err)
	}

	// Parse progress from stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if update := parseProgressLine(line, downloadID); update != nil {
				select {
				case progressCh <- *update:
				default:
					// Don't block if channel is full
				}
			}
		}
	}()

	// Drain stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			// Consume stdout to prevent blocking
		}
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("yt-dlp exited with error: %w", err)
	}

	return nil
}

func (y *YTDLPService) buildDownloadArgs(opts DownloadOptions) []string {
	args := []string{
		"--no-warnings",
		"--newline",
		"--progress",
	}

	// FFmpeg location
	if y.ffmpegPath != "" {
		args = append(args, "--ffmpeg-location", y.ffmpegPath)
	}

	// Output path
	if opts.OutputPath != "" {
		args = append(args, "-o", opts.OutputPath)
	}

	// Format selection
	if opts.AudioOnly {
		args = append(args, "-x")
		switch opts.Format {
		case "mp3":
			args = append(args, "--audio-format", "mp3", "--audio-quality", "0")
			if opts.Bitrate != "" {
				args = append(args, "--postprocessor-args", fmt.Sprintf("ffmpeg:-b:a %s", opts.Bitrate))
			}
		case "m4a", "opus", "flac":
			args = append(args, "--audio-format", opts.Format)
		default:
			args = append(args, "--audio-format", "mp3", "--audio-quality", "0")
		}
	} else {
		// Video format
		switch opts.Quality {
		case "best":
			args = append(args, "-f", "bestvideo+bestaudio/best")
		case "2160", "4k":
			args = append(args, "-f", "bestvideo[height<=2160]+bestaudio/best[height<=2160]")
		case "1080":
			args = append(args, "-f", "bestvideo[height<=1080]+bestaudio/best[height<=1080]")
		case "720":
			args = append(args, "-f", "bestvideo[height<=720]+bestaudio/best[height<=720]")
		case "480":
			args = append(args, "-f", "bestvideo[height<=480]+bestaudio/best[height<=480]")
		default:
			args = append(args, "-f", "bestvideo[height<=1080]+bestaudio/best[height<=1080]")
		}

		// Container format
		if opts.Format == "mp4" {
			args = append(args, "--merge-output-format", "mp4")
		} else if opts.Format == "mkv" {
			args = append(args, "--merge-output-format", "mkv")
		}
	}

	// Subtitles
	if len(opts.SubtitleLanguages) > 0 {
		args = append(args, "--write-subs", "--write-auto-subs")
		args = append(args, "--sub-langs", strings.Join(opts.SubtitleLanguages, ","))
		args = append(args, "--sub-format", "srt/vtt/best")
		if opts.EmbedSubtitles {
			args = append(args, "--embed-subs")
		}
	}

	// Thumbnail
	if opts.EmbedThumbnail {
		args = append(args, "--embed-thumbnail", "--write-thumbnail")
	}

	// Metadata embedding
	args = append(args, "--embed-metadata")

	// Audio normalization via ffmpeg postprocessor
	if opts.NormalizeAudio {
		args = append(args, "--postprocessor-args", "ffmpeg:-af loudnorm=I=-16:TP=-1.5:LRA=11")
	}

	// Proxy
	if opts.ProxyURL != "" {
		args = append(args, "--proxy", opts.ProxyURL)
	}

	// Cookies
	if opts.CookieFile != "" {
		args = append(args, "--cookies", opts.CookieFile)
	}

	// Speed limit
	if opts.SpeedLimit != "" {
		args = append(args, "--limit-rate", opts.SpeedLimit)
	}

	// URL must be last
	args = append(args, opts.URL)

	return args
}

func parseProgressLine(line string, downloadID int64) *ProgressUpdate {
	matches := progressRegex.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	percent, _ := strconv.ParseFloat(matches[1], 64)

	return &ProgressUpdate{
		DownloadID: downloadID,
		Status:     "downloading",
		Percent:    percent,
		Speed:      matches[3],
		ETA:        matches[4],
	}
}

// GetVersion returns the yt-dlp version string
func (y *YTDLPService) GetVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, y.binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting yt-dlp version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// SearchSite searches a site using yt-dlp search extractors
func (y *YTDLPService) SearchSite(ctx context.Context, query string, site string, maxResults int) ([]MediaInfo, error) {
	searchQuery := fmt.Sprintf("%ssearch%d:%s", site, maxResults, query)
	if site == "" || site == "youtube" {
		searchQuery = fmt.Sprintf("ytsearch%d:%s", maxResults, query)
	}

	args := []string{
		"--dump-json",
		"--flat-playlist",
		"--no-download",
		"--no-warnings",
		searchQuery,
	}

	cmd := exec.CommandContext(ctx, y.binaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("searching: %w", err)
	}

	var results []MediaInfo
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		var info MediaInfo
		if err := json.Unmarshal([]byte(line), &info); err != nil {
			continue
		}
		results = append(results, info)
	}

	return results, nil
}
