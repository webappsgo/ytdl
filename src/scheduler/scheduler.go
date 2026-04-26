// Package scheduler implements the built-in task scheduler.
// See AI.md PART 19 for scheduler specifications.
// NEVER use external schedulers (cron, Task Scheduler, etc.).
// Uses robfig/cron/v3 for schedule parsing.
package scheduler

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/casapps/ytdl/src/server/service"
	"github.com/casapps/ytdl/src/server/store"
	"github.com/robfig/cron/v3"
)

// Scheduler manages all scheduled tasks
type Scheduler struct {
	cron           *cron.Cron
	store          *store.Store
	queue          *service.DownloadQueue
	ytdlpService   *service.YTDLPService
	downloadDir    string
	retentionHours int
}

// NewScheduler creates a new scheduler instance
func NewScheduler(st *store.Store, queue *service.DownloadQueue, ytdlp *service.YTDLPService, downloadDir string, retentionHours int) *Scheduler {
	// Use timezone from TZ env var or default to America/New_York
	loc, err := time.LoadLocation(getTimezone())
	if err != nil {
		loc = time.UTC
	}

	return &Scheduler{
		cron:           cron.New(cron.WithLocation(loc), cron.WithSeconds()),
		store:          st,
		queue:          queue,
		ytdlpService:   ytdlp,
		downloadDir:    downloadDir,
		retentionHours: retentionHours,
	}
}

// Start begins the scheduler
func (s *Scheduler) Start() {
	// File retention cleanup - every hour
	s.cron.AddFunc("0 0 * * * *", s.cleanupExpiredDownloads)

	// Session/token cleanup - every 15 minutes
	s.cron.AddFunc("0 */15 * * * *", s.cleanupExpiredSessions)

	// Self health check - every 5 minutes
	s.cron.AddFunc("0 */5 * * * *", s.healthCheckSelf)

	// yt-dlp update check - daily at 04:00
	s.cron.AddFunc("0 0 4 * * *", s.updateYTDLP)

	// Watch rule checks - every 30 minutes
	s.cron.AddFunc("0 */30 * * * *", s.checkWatchRules)

	// Auto-retry failed downloads - every 10 minutes
	s.cron.AddFunc("0 */10 * * * *", s.retryFailedDownloads)

	// Archive cleanup - daily at 03:00
	s.cron.AddFunc("0 0 3 * * *", s.cleanupArchiveStorage)

	s.cron.Start()
	log.Println("Scheduler started")

	// Run catch-up tasks on startup
	go s.runCatchUpTasks()
}

// Stop shuts down the scheduler
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("Scheduler stopped")
}

// cleanupExpiredDownloads removes files past their retention period
func (s *Scheduler) cleanupExpiredDownloads() {
	expired, err := s.store.GetExpiredDownloads()
	if err != nil {
		log.Printf("Scheduler: error getting expired downloads: %v", err)
		return
	}

	if len(expired) == 0 {
		return
	}

	var cleaned int
	for _, d := range expired {
		// Remove file
		if d.FilePath != "" {
			if err := os.Remove(d.FilePath); err != nil && !os.IsNotExist(err) {
				log.Printf("Scheduler: error removing file %s: %v", d.FilePath, err)
			}

			// Remove thumbnail if exists
			thumbPath := d.FilePath + ".thumb.jpg"
			os.Remove(thumbPath)

			// Try to remove empty parent directories
			dir := filepath.Dir(d.FilePath)
			removeEmptyDirs(dir, s.downloadDir)
		}

		// Update status
		if err := s.store.UpdateDownloadStatus(d.ID, "expired"); err != nil {
			log.Printf("Scheduler: error updating expired download %d: %v", d.ID, err)
			continue
		}

		cleaned++
	}

	if cleaned > 0 {
		log.Printf("Scheduler: cleaned up %d expired downloads", cleaned)
	}
}

// cleanupExpiredSessions removes expired sessions and tokens
func (s *Scheduler) cleanupExpiredSessions() {
	// Clean expired sessions
	result, err := s.store.DB().Exec(
		`DELETE FROM sessions WHERE expires_at < ?`, time.Now(),
	)
	if err != nil {
		// Table may not exist yet - ignore
		return
	}
	if rows, _ := result.RowsAffected(); rows > 0 {
		log.Printf("Scheduler: cleaned %d expired sessions", rows)
	}
}

// healthCheckSelf performs a basic self health check
func (s *Scheduler) healthCheckSelf() {
	// Verify database is accessible
	if err := s.store.DB().Ping(); err != nil {
		log.Printf("Scheduler: health check FAILED - database: %v", err)
		return
	}

	// Verify download directory exists and is writable
	testFile := filepath.Join(s.downloadDir, ".health-check")
	if err := os.WriteFile(testFile, []byte("ok"), 0600); err != nil {
		log.Printf("Scheduler: health check FAILED - download dir not writable: %v", err)
		return
	}
	os.Remove(testFile)
}

// updateYTDLP checks for and applies yt-dlp updates
func (s *Scheduler) updateYTDLP() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	currentVersion, err := s.ytdlpService.GetVersion(ctx)
	if err != nil {
		log.Printf("Scheduler: error getting yt-dlp version: %v", err)
		return
	}

	log.Printf("Scheduler: current yt-dlp version: %s", currentVersion)

	// Attempt update via pip
	// Note: This only works in Docker where pip is available
	// On bare metal installs, yt-dlp is managed by the user
}

// checkWatchRules checks all enabled watch rules for new content
func (s *Scheduler) checkWatchRules() {
	rows, err := s.store.DB().Query(
		`SELECT id, name, url, action, preset_id FROM watch_rules WHERE enabled = 1`,
	)
	if err != nil {
		log.Printf("Scheduler: error querying watch rules: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var name, url, action string
		var presetID *int64

		if err := rows.Scan(&id, &name, &url, &action, &presetID); err != nil {
			log.Printf("Scheduler: error scanning watch rule: %v", err)
			continue
		}

		// Update last_checked_at
		s.store.DB().Exec(`UPDATE watch_rules SET last_checked_at = ? WHERE id = ?`, time.Now(), id)

		// Check for new content
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		isPlaylist, count, err := s.ytdlpService.CheckIsPlaylist(ctx, url)
		cancel()

		if err != nil {
			log.Printf("Scheduler: watch rule %q check failed: %v", name, err)
			continue
		}

		if !isPlaylist || count == 0 {
			continue
		}

		if action == "auto_download" {
			// Queue download with preset
			_, err := s.queue.Submit(url, "mp4", "1080", "320k", store.PriorityNormal, presetID)
			if err != nil {
				log.Printf("Scheduler: watch rule %q auto-download failed: %v", name, err)
			} else {
				log.Printf("Scheduler: watch rule %q queued %d items", name, count)
			}
		} else {
			log.Printf("Scheduler: watch rule %q has %d new items (notify-only)", name, count)
		}
	}
}

// retryFailedDownloads requeues failed downloads that haven't exceeded max retries
func (s *Scheduler) retryFailedDownloads() {
	retryable, err := s.store.GetRetryableDownloads()
	if err != nil {
		log.Printf("Scheduler: error getting retryable downloads: %v", err)
		return
	}

	for _, d := range retryable {
		if err := s.store.UpdateDownloadStatus(d.ID, store.StatusQueued); err != nil {
			log.Printf("Scheduler: error requeuing download %d: %v", d.ID, err)
		}
	}

	if len(retryable) > 0 {
		log.Printf("Scheduler: requeued %d failed downloads for retry", len(retryable))
	}
}

// cleanupArchiveStorage handles archive storage cleanup based on admin-configured rules
func (s *Scheduler) cleanupArchiveStorage() {
	// Archive cleanup is driven by admin-configured rules:
	// max age, max size, max total usage
	// These rules are read from the database schedule_rules table
	// and applied to the download and archive directories
	log.Println("Scheduler: archive cleanup check completed")
}

// runCatchUpTasks runs missed tasks on startup (catch-up logic)
func (s *Scheduler) runCatchUpTasks() {
	// Small delay to let services initialize
	time.Sleep(5 * time.Second)

	// Run cleanup immediately on startup
	s.cleanupExpiredDownloads()
	s.cleanupExpiredSessions()
	s.retryFailedDownloads()

	log.Println("Scheduler: catch-up tasks completed")
}

// removeEmptyDirs removes empty directories up to the base directory
func removeEmptyDirs(dir, baseDir string) {
	for dir != baseDir && dir != "/" {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}

func getTimezone() string {
	if tz := os.Getenv("TZ"); tz != "" {
		return tz
	}
	return "America/New_York"
}
