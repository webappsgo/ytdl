// Package service - Backup and restore operations.
// See AI.md PART 22 for backup specifications.
// Creates tar.gz archives of config, data, and database.
package service

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// BackupService handles backup and restore operations
type BackupService struct {
	configDir string
	dataDir   string
	dbDir     string
	backupDir string
}

// NewBackupService creates a new backup service
func NewBackupService(configDir, dataDir, dbDir, backupDir string) *BackupService {
	return &BackupService{
		configDir: configDir,
		dataDir:   dataDir,
		dbDir:     dbDir,
		backupDir: backupDir,
	}
}

// CreateBackup creates a full backup archive
// Format: ytdl_backup_YYYY-MM-DD.tar.gz
func (b *BackupService) CreateBackup() (string, error) {
	if err := os.MkdirAll(b.backupDir, 0700); err != nil {
		return "", fmt.Errorf("creating backup directory: %w", err)
	}

	filename := fmt.Sprintf("ytdl_backup_%s.tar.gz", time.Now().Format("2006-01-02"))
	archivePath := filepath.Join(b.backupDir, filename)

	file, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("creating backup file: %w", err)
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Backup config directory
	if err := addDirToTar(tw, b.configDir, "config"); err != nil {
		return "", fmt.Errorf("backing up config: %w", err)
	}

	// Backup database files
	if err := addDirToTar(tw, b.dbDir, "db"); err != nil {
		return "", fmt.Errorf("backing up database: %w", err)
	}

	return archivePath, nil
}

// RestoreBackup restores from a backup archive
func (b *BackupService) RestoreBackup(archivePath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening backup: %w", err)
	}
	defer file.Close()

	gr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("reading gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		// Determine destination based on prefix
		var destBase string
		name := header.Name
		if len(name) > 7 && name[:7] == "config/" {
			destBase = b.configDir
			name = name[7:]
		} else if len(name) > 3 && name[:3] == "db/" {
			destBase = b.dbDir
			name = name[3:]
		} else {
			continue
		}

		destPath := filepath.Join(destBase, name)

		// Security: prevent path traversal
		absPath, err := filepath.Abs(destPath)
		if err != nil {
			continue
		}
		absBase, _ := filepath.Abs(destBase)
		if len(absPath) < len(absBase) || absPath[:len(absBase)] != absBase {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(destPath, os.FileMode(header.Mode))
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(destPath), 0700)
			outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				continue
			}
			io.Copy(outFile, tr)
			outFile.Close()
		}
	}

	return nil
}

// ListBackups returns available backup files
func (b *BackupService) ListBackups() ([]BackupInfo, error) {
	entries, err := os.ReadDir(b.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, BackupInfo{
			Name:    entry.Name(),
			Size:    info.Size(),
			Created: info.ModTime(),
		})
	}

	return backups, nil
}

// BackupInfo holds metadata about a backup file
type BackupInfo struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	Created time.Time `json:"created"`
}

func addDirToTar(tw *tar.Writer, srcDir, prefix string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return nil
		}

		header.Name = filepath.Join(prefix, relPath)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		_, err = io.Copy(tw, file)
		return err
	})
}
