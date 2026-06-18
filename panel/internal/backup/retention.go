package backup

import (
	"context"
	"os"
	"path/filepath"
)

// ListBackups returns all backup records ordered by started_at descending.
func (s *Service) ListBackups(ctx context.Context) ([]BackupRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, filename, status, type, size_bytes, checksum, nodes_included, nodes_skipped, error_message, started_at, completed_at
		 FROM backups ORDER BY started_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []BackupRecord
	for rows.Next() {
		var r BackupRecord
		if err := rows.Scan(&r.ID, &r.Filename, &r.Status, &r.Type, &r.SizeBytes, &r.Checksum,
			&r.NodesIncluded, &r.NodesSkipped, &r.ErrorMessage, &r.StartedAt, &r.CompletedAt); err != nil {
			continue
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// DeleteBackup removes a single backup archive and its companion files from disk.
func (s *Service) DeleteBackup(ctx context.Context, backupID int64) error {
	var filename string
	err := s.db.QueryRowContext(ctx, `SELECT filename FROM backups WHERE id=?`, backupID).Scan(&filename)
	if err != nil {
		return err
	}

	archivePath := filepath.Join(s.cfg.StorageDir, filename)
	os.Remove(archivePath)
	os.Remove(archivePath + ".sha256")

	_, err = s.db.ExecContext(ctx, `DELETE FROM backups WHERE id=?`, backupID)
	return err
}

// VerifyIntegrity recomputes the checksum of a backup file and compares with stored value.
func (s *Service) VerifyIntegrity(ctx context.Context, backupID int64) (bool, error) {
	var filename string
	var storedChecksum string
	err := s.db.QueryRowContext(ctx, `SELECT filename, COALESCE(checksum,'') FROM backups WHERE id=?`, backupID).Scan(&filename, &storedChecksum)
	if err != nil {
		return false, err
	}
	if storedChecksum == "" {
		return false, nil
	}

	archivePath := filepath.Join(s.cfg.StorageDir, filename)
	return VerifyChecksum(archivePath, storedChecksum)
}
