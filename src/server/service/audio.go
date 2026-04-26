// Package service - Audio processing: ID3 tagging, lyrics embedding, normalization.
// MP3 default: CBR 320kbps with full ID3 tags.
// Lyrics: synced (SYLT) + unsynced (USLT), EN/ES required.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

// AudioProcessor handles post-download audio processing
type AudioProcessor struct {
	ffmpegPath string
}

// NewAudioProcessor creates a new audio processor
func NewAudioProcessor(ffmpegPath string) *AudioProcessor {
	return &AudioProcessor{ffmpegPath: ffmpegPath}
}

// ID3Tags holds metadata for audio file tagging
type ID3Tags struct {
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	Year        string `json:"year"`
	Genre       string `json:"genre"`
	TrackNumber string `json:"track_number"`
	CoverArt    string `json:"cover_art_path"`
}

// LyricsResult holds lyrics fetched from external sources
type LyricsResult struct {
	SyncedLyrics   string `json:"synced_lyrics"`
	UnsyncedLyrics string `json:"unsynced_lyrics"`
	Language       string `json:"language"`
	Source         string `json:"source"`
}

// EmbedID3Tags writes ID3 metadata to an MP3 file using ffmpeg
func (a *AudioProcessor) EmbedID3Tags(ctx context.Context, filePath string, tags ID3Tags) error {
	args := []string{
		"-i", filePath,
		"-y",
	}

	// Add cover art if available
	if tags.CoverArt != "" {
		args = append(args, "-i", tags.CoverArt)
		args = append(args,
			"-map", "0:a",
			"-map", "1:v",
			"-c:v", "mjpeg",
			"-disposition:v", "attached_pic",
		)
	} else {
		args = append(args, "-map", "0:a")
	}

	args = append(args,
		"-c:a", "copy",
		"-id3v2_version", "3",
	)

	// Add metadata tags
	if tags.Title != "" {
		args = append(args, "-metadata", fmt.Sprintf("title=%s", tags.Title))
	}
	if tags.Artist != "" {
		args = append(args, "-metadata", fmt.Sprintf("artist=%s", tags.Artist))
	}
	if tags.Album != "" {
		args = append(args, "-metadata", fmt.Sprintf("album=%s", tags.Album))
	}
	if tags.Year != "" {
		args = append(args, "-metadata", fmt.Sprintf("date=%s", tags.Year))
	}
	if tags.Genre != "" {
		args = append(args, "-metadata", fmt.Sprintf("genre=%s", tags.Genre))
	}
	if tags.TrackNumber != "" {
		args = append(args, "-metadata", fmt.Sprintf("track=%s", tags.TrackNumber))
	}

	// Output to temp file then replace
	outputPath := filePath + ".tagged.mp3"
	args = append(args, outputPath)

	cmd := exec.CommandContext(ctx, a.ffmpegPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("embedding ID3 tags: %w (output: %s)", err, string(output))
	}

	// Replace original with tagged version
	if err := exec.CommandContext(ctx, "mv", outputPath, filePath).Run(); err != nil {
		return fmt.Errorf("replacing file: %w", err)
	}

	return nil
}

// EmbedLyrics writes lyrics to an MP3 file using ffmpeg
// Embeds as USLT (unsynced lyrics) tag
func (a *AudioProcessor) EmbedLyrics(ctx context.Context, filePath string, lyrics string, language string) error {
	if lyrics == "" {
		return nil
	}

	args := []string{
		"-i", filePath,
		"-y",
		"-c", "copy",
		"-metadata", fmt.Sprintf("lyrics-XXX=%s", lyrics),
	}

	// Language tag
	if language != "" {
		args = append(args, "-metadata", fmt.Sprintf("language=%s", language))
	}

	outputPath := filePath + ".lyrics.mp3"
	args = append(args, outputPath)

	cmd := exec.CommandContext(ctx, a.ffmpegPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("embedding lyrics: %w (output: %s)", err, string(output))
	}

	if err := exec.CommandContext(ctx, "mv", outputPath, filePath).Run(); err != nil {
		return fmt.Errorf("replacing file: %w", err)
	}

	return nil
}

// NormalizeAudio applies loudness normalization using ffmpeg loudnorm filter
func (a *AudioProcessor) NormalizeAudio(ctx context.Context, filePath string) error {
	outputPath := filePath + ".normalized"

	args := []string{
		"-i", filePath,
		"-y",
		"-af", "loudnorm=I=-16:TP=-1.5:LRA=11",
		"-ar", "44100",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, a.ffmpegPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("normalizing audio: %w (output: %s)", err, string(output))
	}

	return exec.CommandContext(ctx, "mv", outputPath, filePath).Run()
}

// TrimSilence removes leading/trailing silence from audio
func (a *AudioProcessor) TrimSilence(ctx context.Context, filePath string) error {
	outputPath := filePath + ".trimmed"

	args := []string{
		"-i", filePath,
		"-y",
		"-af", "silenceremove=start_periods=1:start_duration=0.1:start_threshold=-50dB,areverse,silenceremove=start_periods=1:start_duration=0.1:start_threshold=-50dB,areverse",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, a.ffmpegPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("trimming silence: %w (output: %s)", err, string(output))
	}

	return exec.CommandContext(ctx, "mv", outputPath, filePath).Run()
}

// ConvertToMP3 converts any audio file to MP3 with specified bitrate
func (a *AudioProcessor) ConvertToMP3(ctx context.Context, inputPath, outputPath, bitrate string) error {
	if bitrate == "" {
		bitrate = "320k"
	}

	args := []string{
		"-i", inputPath,
		"-y",
		"-vn",
		"-acodec", "libmp3lame",
		"-b:a", bitrate,
		"-ar", "44100",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, a.ffmpegPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("converting to MP3: %w (output: %s)", err, string(output))
	}

	return nil
}

// LRCLIBClient fetches lyrics from the LRCLIB API
type LRCLIBClient struct {
	httpClient *http.Client
}

// NewLRCLIBClient creates a new LRCLIB lyrics client
func NewLRCLIBClient() *LRCLIBClient {
	return &LRCLIBClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// lrclibResponse is the JSON response from LRCLIB API
type lrclibResponse struct {
	ID             int    `json:"id"`
	TrackName      string `json:"trackName"`
	ArtistName     string `json:"artistName"`
	AlbumName      string `json:"albumName"`
	Duration       int    `json:"duration"`
	SyncedLyrics   string `json:"syncedLyrics"`
	PlainLyrics    string `json:"plainLyrics"`
}

// FetchLyrics searches LRCLIB for lyrics matching the given track
func (c *LRCLIBClient) FetchLyrics(ctx context.Context, title, artist string) (*LyricsResult, error) {
	if title == "" {
		return nil, fmt.Errorf("title is required for lyrics search")
	}

	// Build search URL
	params := url.Values{}
	params.Set("track_name", title)
	if artist != "" {
		params.Set("artist_name", artist)
	}

	searchURL := "https://lrclib.net/api/search?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "ytdl/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching lyrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LRCLIB returned status %d", resp.StatusCode)
	}

	var results []lrclibResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	// Use best match (first result)
	best := results[0]

	result := &LyricsResult{
		Source: "lrclib",
	}

	// Prefer synced lyrics
	if best.SyncedLyrics != "" {
		result.SyncedLyrics = best.SyncedLyrics
	}
	if best.PlainLyrics != "" {
		result.UnsyncedLyrics = best.PlainLyrics
	}

	return result, nil
}

// FetchLyricsForLanguages attempts to fetch lyrics in multiple languages
func (c *LRCLIBClient) FetchLyricsForLanguages(ctx context.Context, title, artist string, languages []string) map[string]*LyricsResult {
	results := make(map[string]*LyricsResult)

	// LRCLIB doesn't filter by language, so we get what's available
	lyrics, err := c.FetchLyrics(ctx, title, artist)
	if err != nil || lyrics == nil {
		return results
	}

	// Detect language (basic heuristic)
	lang := detectLyricsLanguage(lyrics.UnsyncedLyrics)
	lyrics.Language = lang
	results[lang] = lyrics

	return results
}

// detectLyricsLanguage performs basic language detection on lyrics text
func detectLyricsLanguage(text string) string {
	if text == "" {
		return "en"
	}

	// Simple heuristic: check for Spanish-specific characters and common words
	lowerText := strings.ToLower(text)
	spanishIndicators := []string{"ñ", "¿", "¡", " el ", " la ", " los ", " las ",
		" que ", " por ", " con ", " una ", " del ", " para "}

	spanishScore := 0
	for _, indicator := range spanishIndicators {
		if strings.Contains(lowerText, indicator) {
			spanishScore++
		}
	}

	if spanishScore >= 3 {
		return "es"
	}

	return "en"
}
