// Package service - Download queue with background worker pool.
// Manages concurrent downloads, priority ordering, pause/resume/cancel.
package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/casapps/ytdl/src/server/store"
)

// DownloadQueue manages the download worker pool
type DownloadQueue struct {
	store          *store.Store
	ytdlp          *YTDLPService
	audioProcessor *AudioProcessor
	mediaProcessor *MediaProcessor
	lyricsClient   *LRCLIBClient
	workers        int
	downloadDir    string
	retentionHours int
	speedLimit     string

	// Channels
	progressCh  chan ProgressUpdate
	cancelFuncs map[int64]context.CancelFunc

	// Synchronization
	mu      sync.RWMutex
	running bool
	wg      sync.WaitGroup
	stopCh  chan struct{}

	// Callback for broadcasting progress to WebSocket clients
	OnProgress func(ProgressUpdate)
}

// NewDownloadQueue creates a new download queue
func NewDownloadQueue(st *store.Store, ytdlp *YTDLPService, ffmpegPath string, workers int, downloadDir string, retentionHours int) *DownloadQueue {
	return &DownloadQueue{
		store:          st,
		ytdlp:          ytdlp,
		audioProcessor: NewAudioProcessor(ffmpegPath),
		mediaProcessor: NewMediaProcessor(ffmpegPath),
		lyricsClient:   NewLRCLIBClient(),
		workers:        workers,
		downloadDir:    downloadDir,
		retentionHours: retentionHours,
		progressCh:     make(chan ProgressUpdate, 100),
		cancelFuncs:    make(map[int64]context.CancelFunc),
		stopCh:         make(chan struct{}),
	}
}

// Start begins processing the download queue
func (q *DownloadQueue) Start() {
	q.mu.Lock()
	if q.running {
		q.mu.Unlock()
		return
	}
	q.running = true
	q.mu.Unlock()

	// Recover interrupted downloads from crash
	recovered, err := q.store.RecoverInterruptedDownloads()
	if err != nil {
		log.Printf("Warning: failed to recover interrupted downloads: %v", err)
	} else if recovered > 0 {
		log.Printf("Recovered %d interrupted downloads", recovered)
	}

	// Start worker goroutines
	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}

	// Start progress forwarder
	q.wg.Add(1)
	go q.progressForwarder()

	log.Printf("Download queue started with %d workers", q.workers)
}

// Stop gracefully shuts down the queue
func (q *DownloadQueue) Stop() {
	q.mu.Lock()
	if !q.running {
		q.mu.Unlock()
		return
	}
	q.running = false
	q.mu.Unlock()

	close(q.stopCh)

	// Cancel all active downloads
	q.mu.RLock()
	for _, cancel := range q.cancelFuncs {
		cancel()
	}
	q.mu.RUnlock()

	q.wg.Wait()
	close(q.progressCh)
	log.Println("Download queue stopped")
}

// Submit adds a new download to the queue
func (q *DownloadQueue) Submit(url, format, quality, bitrate, priority string, presetID *int64) (int64, error) {
	d := &store.Download{
		URL:        url,
		Format:     format,
		Quality:    quality,
		Bitrate:    bitrate,
		Priority:   priority,
		PresetID:   presetID,
		MaxRetries: 3,
	}

	id, err := q.store.CreateDownload(d)
	if err != nil {
		return 0, fmt.Errorf("submitting download: %w", err)
	}

	return id, nil
}

// CancelDownload cancels an active or queued download
func (q *DownloadQueue) CancelDownload(id int64) error {
	q.mu.RLock()
	cancel, exists := q.cancelFuncs[id]
	q.mu.RUnlock()

	if exists {
		cancel()
	}

	return q.store.UpdateDownloadStatus(id, store.StatusCancelled)
}

// PauseDownload pauses a queued download
func (q *DownloadQueue) PauseDownload(id int64) error {
	return q.store.UpdateDownloadStatus(id, store.StatusPaused)
}

// ResumeDownload resumes a paused download
func (q *DownloadQueue) ResumeDownload(id int64) error {
	return q.store.UpdateDownloadStatus(id, store.StatusQueued)
}

// RetryDownload requeues a failed download
func (q *DownloadQueue) RetryDownload(id int64) error {
	return q.store.UpdateDownloadStatus(id, store.StatusQueued)
}

func (q *DownloadQueue) worker(workerID int) {
	defer q.wg.Done()

	for {
		select {
		case <-q.stopCh:
			return
		default:
		}

		// Get next download from queue
		download, err := q.store.GetNextQueuedDownload()
		if err != nil {
			log.Printf("Worker %d: error getting next download: %v", workerID, err)
			time.Sleep(5 * time.Second)
			continue
		}

		if download == nil {
			// No downloads in queue, wait
			time.Sleep(2 * time.Second)
			continue
		}

		q.processDownload(workerID, download)
	}
}

func (q *DownloadQueue) processDownload(workerID int, download *store.Download) {
	log.Printf("Worker %d: processing download %d: %s", workerID, download.ID, download.URL)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	q.mu.Lock()
	q.cancelFuncs[download.ID] = cancel
	q.mu.Unlock()

	defer func() {
		cancel()
		q.mu.Lock()
		delete(q.cancelFuncs, download.ID)
		q.mu.Unlock()
	}()

	// Mark as downloading
	if err := q.store.UpdateDownloadStatus(download.ID, store.StatusDownloading); err != nil {
		log.Printf("Worker %d: error updating status: %v", workerID, err)
		return
	}

	// Update started_at
	now := time.Now()
	q.store.DB().Exec(`UPDATE downloads SET started_at = ? WHERE id = ?`, now, download.ID)

	// Extract metadata first
	info, err := q.ytdlp.ExtractMediaInfo(ctx, download.URL)
	if err != nil {
		log.Printf("Worker %d: error extracting info for %d: %v", workerID, download.ID, err)
		q.store.FailDownload(download.ID, fmt.Sprintf("metadata extraction failed: %v", err))
		return
	}

	// Update download with metadata
	q.store.UpdateDownloadMetadata(
		download.ID,
		info.Title,
		info.Description,
		info.Extractor,
		info.Uploader,
		info.UploaderURL,
		info.ThumbnailURL,
		info.Duration,
	)

	// Build output path using template
	outputDir := filepath.Join(q.downloadDir, sanitizeFilename(info.Extractor), sanitizeFilename(info.Uploader))
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		q.store.FailDownload(download.ID, fmt.Sprintf("failed to create output directory: %v", err))
		return
	}

	outputPath := filepath.Join(outputDir, sanitizeFilename(info.Title)+".%(ext)s")

	// Build download options
	isAudioOnly := download.Format == "mp3" || download.Format == "m4a" || download.Format == "opus" || download.Format == "flac"
	opts := DownloadOptions{
		URL:               download.URL,
		Format:            download.Format,
		Quality:           download.Quality,
		Bitrate:           download.Bitrate,
		OutputPath:        outputPath,
		AudioOnly:         isAudioOnly,
		SubtitleLanguages: []string{"en", "es"},
		EmbedSubtitles:    true,
		EmbedLyrics:       true,
		EmbedThumbnail:    true,
		ProxyURL:          download.ProxyConfig,
		SpeedLimit:        q.speedLimit,
	}

	// Execute download
	if err := q.ytdlp.Download(ctx, opts, q.progressCh, download.ID); err != nil {
		if ctx.Err() == context.Canceled {
			log.Printf("Worker %d: download %d cancelled", workerID, download.ID)
			return
		}
		log.Printf("Worker %d: download %d failed: %v", workerID, download.ID, err)
		q.store.FailDownload(download.ID, err.Error())
		return
	}

	// Find the actual output file (yt-dlp replaces %(ext)s)
	actualPath := findDownloadedFile(outputDir, sanitizeFilename(info.Title))
	if actualPath == "" {
		q.store.FailDownload(download.ID, "downloaded file not found")
		return
	}

	// Post-download processing
	q.store.UpdateDownloadStatus(download.ID, store.StatusProcessing)
	log.Printf("Worker %d: post-processing download %d", workerID, download.ID)

	// Audio post-processing (MP3 files)
	if isAudioOnly {
		q.postProcessAudio(ctx, download, actualPath, info)
	} else {
		// Video post-processing: embed subtitles
		q.postProcessVideo(ctx, download, actualPath)
	}

	// Get file size after processing
	fileInfo, _ := os.Stat(actualPath)
	var fileSize int64
	if fileInfo != nil {
		fileSize = fileInfo.Size()
	}

	// Calculate expiration
	var expiresAt *time.Time
	if q.retentionHours > 0 {
		exp := time.Now().Add(time.Duration(q.retentionHours) * time.Hour)
		expiresAt = &exp
	}

	// Mark as completed
	if err := q.store.CompleteDownload(download.ID, actualPath, fileSize, expiresAt); err != nil {
		log.Printf("Worker %d: error completing download %d: %v", workerID, download.ID, err)
		return
	}

	// Record analytics
	q.store.CreateAnalyticsEvent("download_complete", &download.ID, info.Extractor, download.Format, fileSize, info.Duration)

	log.Printf("Worker %d: download %d completed: %s (%d bytes)", workerID, download.ID, actualPath, fileSize)
}

func (q *DownloadQueue) progressForwarder() {
	defer q.wg.Done()

	for {
		select {
		case <-q.stopCh:
			return
		case update, ok := <-q.progressCh:
			if !ok {
				return
			}
			// Update database
			q.store.UpdateDownloadProgress(update.DownloadID, update.Percent, update.FileSize)

			// Forward to WebSocket clients
			if q.OnProgress != nil {
				q.OnProgress(update)
			}
		}
	}
}

// sanitizeFilename removes or replaces characters unsafe for filenames
func sanitizeFilename(name string) string {
	if name == "" {
		return "unknown"
	}

	// Replace common problematic characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		"\n", " ",
		"\r", "",
	)

	result := replacer.Replace(name)

	// Trim spaces and dots from edges
	result = strings.TrimSpace(result)
	result = strings.Trim(result, ".")

	// Limit length
	if len(result) > 200 {
		result = result[:200]
	}

	if result == "" {
		return "unknown"
	}

	return result
}

// postProcessAudio handles audio-specific post-processing:
// ID3 tags, lyrics embedding (synced + unsynced, EN/ES), normalization, thumbnail
func (q *DownloadQueue) postProcessAudio(ctx context.Context, download *store.Download, filePath string, info *MediaInfo) {
	// Embed ID3 tags from source metadata
	tags := ID3Tags{
		Title:  info.Title,
		Artist: info.Uploader,
	}

	// Try to extract thumbnail for cover art
	thumbPath := filePath + ".thumb.jpg"
	if err := q.mediaProcessor.ExtractThumbnail(ctx, filePath, thumbPath); err == nil {
		tags.CoverArt = thumbPath
	}

	if err := q.audioProcessor.EmbedID3Tags(ctx, filePath, tags); err != nil {
		log.Printf("Warning: failed to embed ID3 tags for download %d: %v", download.ID, err)
	}

	// Fetch and embed lyrics (try LRCLIB API)
	lyrics := q.lyricsClient.FetchLyricsForLanguages(ctx, info.Title, info.Uploader, []string{"en", "es"})
	for lang, result := range lyrics {
		if result == nil {
			continue
		}
		// Prefer synced lyrics, fall back to unsynced
		lyricsText := result.SyncedLyrics
		if lyricsText == "" {
			lyricsText = result.UnsyncedLyrics
		}
		if lyricsText != "" {
			q.audioProcessor.EmbedLyrics(ctx, filePath, lyricsText, lang)
		}

		// Save to metadata table
		q.store.DB().Exec(
			`INSERT INTO media_metadata (download_id, title, artist, lyrics_synced, lyrics_unsynced, lyrics_language)
			 VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT(download_id) DO UPDATE SET
			 lyrics_synced=excluded.lyrics_synced, lyrics_unsynced=excluded.lyrics_unsynced, lyrics_language=excluded.lyrics_language`,
			download.ID, info.Title, info.Uploader, result.SyncedLyrics, result.UnsyncedLyrics, lang,
		)
	}

	// Embed cover art thumbnail
	if tags.CoverArt != "" {
		q.mediaProcessor.EmbedThumbnail(ctx, filePath, tags.CoverArt)
	}
}

// postProcessVideo handles video-specific post-processing:
// Embed EN/ES subtitle tracks into the video container
func (q *DownloadQueue) postProcessVideo(ctx context.Context, download *store.Download, filePath string) {
	// Find subtitle files downloaded by yt-dlp
	subtitleFiles := FindSubtitleFiles(filePath, []string{"en", "es"})

	if len(subtitleFiles) > 0 {
		if err := q.mediaProcessor.EmbedSubtitles(ctx, filePath, subtitleFiles); err != nil {
			log.Printf("Warning: failed to embed subtitles for download %d: %v", download.ID, err)
		} else {
			log.Printf("Embedded %d subtitle tracks for download %d", len(subtitleFiles), download.ID)
		}
	}
}

// findDownloadedFile searches for the most recently modified file matching the title prefix
func findDownloadedFile(dir, titlePrefix string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var bestPath string
	var bestTime time.Time

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), titlePrefix) {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(bestTime) {
				bestTime = info.ModTime()
				bestPath = filepath.Join(dir, entry.Name())
			}
		}
	}

	return bestPath
}
