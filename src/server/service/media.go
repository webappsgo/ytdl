// Package service - Media processing: subtitles, thumbnails, format conversion, metadata.
// Subtitles: auto-download EN/ES (SRT+VTT), embed into MKV/MP4 as selectable tracks.
// Thumbnails: extract and embed into audio/video files.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MediaProcessor handles post-download media processing
type MediaProcessor struct {
	ffmpegPath string
}

// NewMediaProcessor creates a new media processor
func NewMediaProcessor(ffmpegPath string) *MediaProcessor {
	return &MediaProcessor{ffmpegPath: ffmpegPath}
}

// EmbedSubtitles embeds subtitle files into a video container (MKV/MP4)
// as selectable tracks that can be enabled/disabled by the user
func (m *MediaProcessor) EmbedSubtitles(ctx context.Context, videoPath string, subtitleFiles []SubtitleFile) error {
	if len(subtitleFiles) == 0 {
		return nil
	}

	args := []string{"-i", videoPath, "-y"}

	// Add each subtitle file as input
	for _, sub := range subtitleFiles {
		if _, err := os.Stat(sub.Path); os.IsNotExist(err) {
			continue
		}
		args = append(args, "-i", sub.Path)
	}

	// Map video and audio from first input
	args = append(args, "-map", "0:v", "-map", "0:a")

	// Map each subtitle track
	validSubs := 0
	for i, sub := range subtitleFiles {
		if _, err := os.Stat(sub.Path); os.IsNotExist(err) {
			continue
		}
		args = append(args, "-map", fmt.Sprintf("%d:s", i+1))
		validSubs++

		// Set subtitle metadata
		streamIdx := validSubs - 1
		args = append(args, fmt.Sprintf("-metadata:s:s:%d", streamIdx), fmt.Sprintf("language=%s", sub.Language))
		args = append(args, fmt.Sprintf("-metadata:s:s:%d", streamIdx), fmt.Sprintf("title=%s", sub.Title))
	}

	if validSubs == 0 {
		return nil
	}

	// Copy codecs (no re-encoding)
	args = append(args, "-c:v", "copy", "-c:a", "copy")

	// Subtitle codec based on container
	ext := strings.ToLower(filepath.Ext(videoPath))
	switch ext {
	case ".mkv":
		args = append(args, "-c:s", "srt")
	case ".mp4":
		args = append(args, "-c:s", "mov_text")
	default:
		args = append(args, "-c:s", "srt")
	}

	outputPath := videoPath + ".subs" + ext
	args = append(args, outputPath)

	cmd := exec.CommandContext(ctx, m.ffmpegPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("embedding subtitles: %w (output: %s)", err, string(output))
	}

	return exec.CommandContext(ctx, "mv", outputPath, videoPath).Run()
}

// SubtitleFile represents a subtitle file to embed
type SubtitleFile struct {
	Path     string
	Language string
	Title    string
}

// FindSubtitleFiles searches for subtitle files matching the video file
func FindSubtitleFiles(videoPath string, languages []string) []SubtitleFile {
	dir := filepath.Dir(videoPath)
	baseName := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))

	var files []SubtitleFile

	for _, lang := range languages {
		langTitle := languageTitle(lang)

		// Check for SRT files
		srtPath := filepath.Join(dir, fmt.Sprintf("%s.%s.srt", baseName, lang))
		if _, err := os.Stat(srtPath); err == nil {
			files = append(files, SubtitleFile{
				Path:     srtPath,
				Language: lang,
				Title:    langTitle,
			})
		}

		// Check for VTT files (convert to SRT for embedding)
		vttPath := filepath.Join(dir, fmt.Sprintf("%s.%s.vtt", baseName, lang))
		if _, err := os.Stat(vttPath); err == nil {
			// If no SRT exists, use VTT
			if _, err := os.Stat(srtPath); os.IsNotExist(err) {
				files = append(files, SubtitleFile{
					Path:     vttPath,
					Language: lang,
					Title:    langTitle,
				})
			}
		}
	}

	return files
}

func languageTitle(langCode string) string {
	switch langCode {
	case "en":
		return "English"
	case "es":
		return "Spanish"
	case "fr":
		return "French"
	case "de":
		return "German"
	case "pt":
		return "Portuguese"
	case "ja":
		return "Japanese"
	case "ko":
		return "Korean"
	case "zh":
		return "Chinese"
	default:
		return strings.ToUpper(langCode)
	}
}

// ExtractThumbnail extracts thumbnail from a video file
func (m *MediaProcessor) ExtractThumbnail(ctx context.Context, videoPath, outputPath string) error {
	args := []string{
		"-i", videoPath,
		"-y",
		"-ss", "00:00:05",
		"-vframes", "1",
		"-vf", "scale=640:-1",
		"-q:v", "5",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, m.ffmpegPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("extracting thumbnail: %w (output: %s)", err, string(output))
	}

	return nil
}

// EmbedThumbnail embeds a thumbnail image into an audio file (cover art)
func (m *MediaProcessor) EmbedThumbnail(ctx context.Context, audioPath, thumbnailPath string) error {
	if _, err := os.Stat(thumbnailPath); os.IsNotExist(err) {
		return nil
	}

	ext := strings.ToLower(filepath.Ext(audioPath))
	outputPath := audioPath + ".thumb" + ext

	args := []string{
		"-i", audioPath,
		"-i", thumbnailPath,
		"-y",
		"-map", "0:a",
		"-map", "1:v",
		"-c:a", "copy",
		"-c:v", "mjpeg",
		"-disposition:v", "attached_pic",
		"-id3v2_version", "3",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, m.ffmpegPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("embedding thumbnail: %w (output: %s)", err, string(output))
	}

	return exec.CommandContext(ctx, "mv", outputPath, audioPath).Run()
}

// RemuxToMP4 remuxes a video file to MP4 container without re-encoding
func (m *MediaProcessor) RemuxToMP4(ctx context.Context, inputPath string) (string, error) {
	outputPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + ".mp4"
	if inputPath == outputPath {
		return inputPath, nil
	}

	args := []string{
		"-i", inputPath,
		"-y",
		"-c", "copy",
		"-movflags", "+faststart",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, m.ffmpegPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("remuxing to MP4: %w (output: %s)", err, string(output))
	}

	// Remove original
	os.Remove(inputPath)

	return outputPath, nil
}

// Transcode transcodes a video file with specified parameters
func (m *MediaProcessor) Transcode(ctx context.Context, inputPath, outputPath string, opts TranscodeOptions) error {
	args := []string{"-i", inputPath, "-y"}

	// Video codec
	if opts.VideoCodec != "" {
		args = append(args, "-c:v", opts.VideoCodec)
	} else {
		args = append(args, "-c:v", "copy")
	}

	// Audio codec
	if opts.AudioCodec != "" {
		args = append(args, "-c:a", opts.AudioCodec)
	} else {
		args = append(args, "-c:a", "copy")
	}

	// Video bitrate
	if opts.VideoBitrate != "" {
		args = append(args, "-b:v", opts.VideoBitrate)
	}

	// Audio bitrate
	if opts.AudioBitrate != "" {
		args = append(args, "-b:a", opts.AudioBitrate)
	}

	// Resolution
	if opts.Resolution != "" {
		args = append(args, "-vf", fmt.Sprintf("scale=-2:%s", opts.Resolution))
	}

	// Faststart for MP4
	if strings.HasSuffix(strings.ToLower(outputPath), ".mp4") {
		args = append(args, "-movflags", "+faststart")
	}

	args = append(args, outputPath)

	cmd := exec.CommandContext(ctx, m.ffmpegPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("transcoding: %w (output: %s)", err, string(output))
	}

	return nil
}

// TranscodeOptions configures transcoding parameters
type TranscodeOptions struct {
	VideoCodec   string
	AudioCodec   string
	VideoBitrate string
	AudioBitrate string
	Resolution   string
}

// GetMediaInfo returns media file information using ffprobe
func (m *MediaProcessor) GetMediaInfo(ctx context.Context, filePath string) (*MediaFileInfo, error) {
	ffprobePath := strings.TrimSuffix(m.ffmpegPath, "ffmpeg") + "ffprobe"

	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	}

	cmd := exec.CommandContext(ctx, ffprobePath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running ffprobe: %w", err)
	}

	var info MediaFileInfo
	if err := parseFFProbeOutput(output, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// MediaFileInfo holds information about a media file
type MediaFileInfo struct {
	Duration    float64 `json:"duration"`
	Size        int64   `json:"size"`
	Bitrate     int64   `json:"bitrate"`
	FormatName  string  `json:"format_name"`
	VideoCodec  string  `json:"video_codec"`
	AudioCodec  string  `json:"audio_codec"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	HasVideo    bool    `json:"has_video"`
	HasAudio    bool    `json:"has_audio"`
	HasSubtitle bool    `json:"has_subtitle"`
}

func parseFFProbeOutput(data []byte, info *MediaFileInfo) error {
	var result struct {
		Format struct {
			Duration string `json:"duration"`
			Size     string `json:"size"`
			BitRate  string `json:"bit_rate"`
			Name     string `json:"format_name"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("parsing ffprobe output: %w", err)
	}

	info.FormatName = result.Format.Name
	fmt.Sscanf(result.Format.Duration, "%f", &info.Duration)
	fmt.Sscanf(result.Format.Size, "%d", &info.Size)
	fmt.Sscanf(result.Format.BitRate, "%d", &info.Bitrate)

	for _, stream := range result.Streams {
		switch stream.CodecType {
		case "video":
			info.HasVideo = true
			info.VideoCodec = stream.CodecName
			info.Width = stream.Width
			info.Height = stream.Height
		case "audio":
			info.HasAudio = true
			info.AudioCodec = stream.CodecName
		case "subtitle":
			info.HasSubtitle = true
		}
	}

	return nil
}
