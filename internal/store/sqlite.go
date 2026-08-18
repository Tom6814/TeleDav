package store

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

type Repository struct {
	db *sql.DB
}

func Open(dsn string) (*Repository, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	return &Repository{db: db}, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) Migrate(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, schemaSQL)
	return err
}

func (r *Repository) EnsureRoot(ctx context.Context) (Directory, error) {
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO directories (id, parent_id, name, path)
VALUES (1, NULL, '', '/')
ON CONFLICT(path) DO NOTHING`); err != nil {
		return Directory{}, err
	}
	row := r.db.QueryRowContext(ctx, `
SELECT id, parent_id, name, path, created_at, updated_at
FROM directories
WHERE path = '/'`)
	var d Directory
	err := row.Scan(&d.ID, &d.ParentID, &d.Name, &d.Path, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}

func (r *Repository) CreateDirectory(ctx context.Context, parentID int64, name, path string) (Directory, error) {
	res, err := r.db.ExecContext(ctx, `
INSERT INTO directories (parent_id, name, path)
VALUES (?, ?, ?)`, parentID, name, path)
	if err != nil {
		return Directory{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Directory{}, err
	}
	row := r.db.QueryRowContext(ctx, `
SELECT id, parent_id, name, path, created_at, updated_at
FROM directories WHERE id = ?`, id)
	var d Directory
	err = row.Scan(&d.ID, &d.ParentID, &d.Name, &d.Path, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}

func (r *Repository) GetDirectory(ctx context.Context, id int64) (Directory, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, parent_id, name, path, created_at, updated_at
FROM directories
WHERE id = ?`, id)
	var d Directory
	err := row.Scan(&d.ID, &d.ParentID, &d.Name, &d.Path, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}

func (r *Repository) ListDirectories(ctx context.Context, parentID int64) ([]Directory, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, parent_id, name, path, created_at, updated_at
FROM directories
WHERE parent_id = ?
ORDER BY name`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Directory
	for rows.Next() {
		var d Directory
		if err := rows.Scan(&d.ID, &d.ParentID, &d.Name, &d.Path, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) CreateFile(ctx context.Context, parentID int64, name string, size int64, status string) (FileEntry, error) {
	res, err := r.db.ExecContext(ctx, `
INSERT INTO file_entries (parent_id, name, size, status)
VALUES (?, ?, ?, ?)`, parentID, name, size, status)
	if err != nil {
		return FileEntry{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return FileEntry{}, err
	}
	return r.GetFile(ctx, id)
}

func (r *Repository) GetFile(ctx context.Context, fileID int64) (FileEntry, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, parent_id, name, size, mime, sha256, status, source, created_at, updated_at
FROM file_entries WHERE id = ?`, fileID)
	var f FileEntry
	err := row.Scan(&f.ID, &f.ParentID, &f.Name, &f.Size, &f.MIME, &f.SHA256, &f.Status, &f.Source, &f.CreatedAt, &f.UpdatedAt)
	return f, err
}

func (r *Repository) MarkFileReady(ctx context.Context, fileID int64) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE file_entries
SET status = 'ready', updated_at = CURRENT_TIMESTAMP
WHERE id = ?`, fileID)
	return err
}

func (r *Repository) DeleteFileLogical(ctx context.Context, fileID int64) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE file_entries
SET status = 'deleting', deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE id = ?`, fileID)
	return err
}

func (r *Repository) ListReadyFiles(ctx context.Context, parentID int64) ([]FileEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, parent_id, name, size, mime, sha256, status, source, created_at, updated_at
FROM file_entries
WHERE parent_id = ? AND status = 'ready'
ORDER BY name`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FileEntry
	for rows.Next() {
		var f FileEntry
		if err := rows.Scan(&f.ID, &f.ParentID, &f.Name, &f.Size, &f.MIME, &f.SHA256, &f.Status, &f.Source, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repository) AppendChunk(ctx context.Context, fileID int64, chunk FileChunk) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO file_chunks (file_id, chunk_index, chunk_size, chunk_sha256, telegram_chat_id, telegram_message_id)
VALUES (?, ?, ?, ?, ?, ?)`,
		fileID, chunk.ChunkIndex, chunk.ChunkSize, chunk.ChunkSHA256, chunk.TelegramChatID, chunk.TelegramMessageID)
	return err
}

func (r *Repository) ListChunks(ctx context.Context, fileID int64) ([]FileChunk, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, file_id, chunk_index, chunk_size, chunk_sha256, telegram_chat_id, telegram_message_id, created_at
FROM file_chunks
WHERE file_id = ?
ORDER BY chunk_index`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FileChunk
	for rows.Next() {
		var fc FileChunk
		if err := rows.Scan(&fc.ID, &fc.FileID, &fc.ChunkIndex, &fc.ChunkSize, &fc.ChunkSHA256, &fc.TelegramChatID, &fc.TelegramMessageID, &fc.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, fc)
	}
	return out, rows.Err()
}

func (r *Repository) ResetFileChunks(ctx context.Context, fileID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM file_chunks WHERE file_id = ?`, fileID)
	return err
}

func (r *Repository) CreateUploadJob(ctx context.Context, source, stage, stagedPath string, fileID int64) (UploadJob, error) {
	res, err := r.db.ExecContext(ctx, `
INSERT INTO upload_jobs (file_id, source, stage, staged_path)
VALUES (?, ?, ?, ?)`, fileID, source, stage, stagedPath)
	if err != nil {
		return UploadJob{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return UploadJob{}, err
	}
	return r.GetUploadJob(ctx, id)
}

func (r *Repository) GetUploadJob(ctx context.Context, jobID int64) (UploadJob, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, file_id, source, stage, retry_count, last_error, last_chunk_index, staged_path, created_at, updated_at
FROM upload_jobs WHERE id = ?`, jobID)
	var job UploadJob
	err := row.Scan(&job.ID, &job.FileID, &job.Source, &job.Stage, &job.RetryCount, &job.LastError, &job.LastChunkIndex, &job.StagedPath, &job.CreatedAt, &job.UpdatedAt)
	return job, err
}

func (r *Repository) UpdateUploadJob(ctx context.Context, jobID int64, stage string, lastChunkIndex int, lastError string) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE upload_jobs
SET stage = ?, last_chunk_index = ?, last_error = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?`, stage, lastChunkIndex, lastError, jobID)
	return err
}

func (r *Repository) ListPendingJobs(ctx context.Context) ([]UploadJob, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, file_id, source, stage, retry_count, last_error, last_chunk_index, staged_path, created_at, updated_at
FROM upload_jobs
WHERE stage IN ('staged', 'chunking', 'uploading', 'failed')
ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []UploadJob
	for rows.Next() {
		var job UploadJob
		if err := rows.Scan(&job.ID, &job.FileID, &job.Source, &job.Stage, &job.RetryCount, &job.LastError, &job.LastChunkIndex, &job.StagedPath, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertSystemConfig(ctx context.Context, cfg SystemConfig) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO system_config (id, telegram_session_blob, telegram_target_chat_id, default_chunk_size, max_staging_bytes, download_cache_ttl_seconds, app_password)
VALUES (1, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
telegram_session_blob = excluded.telegram_session_blob,
telegram_target_chat_id = excluded.telegram_target_chat_id,
default_chunk_size = excluded.default_chunk_size,
max_staging_bytes = excluded.max_staging_bytes,
download_cache_ttl_seconds = excluded.download_cache_ttl_seconds,
app_password = excluded.app_password,
updated_at = CURRENT_TIMESTAMP`,
		cfg.TelegramSessionBlob, cfg.TelegramTargetChatID, cfg.DefaultChunkSize, cfg.MaxStagingBytes, cfg.DownloadCacheTTL, cfg.AppPassword)
	return err
}

func (r *Repository) GetSystemConfig(ctx context.Context) (SystemConfig, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT telegram_session_blob, telegram_target_chat_id, default_chunk_size, max_staging_bytes, download_cache_ttl_seconds, app_password
FROM system_config WHERE id = 1`)
	var cfg SystemConfig
	err := row.Scan(&cfg.TelegramSessionBlob, &cfg.TelegramTargetChatID, &cfg.DefaultChunkSize, &cfg.MaxStagingBytes, &cfg.DownloadCacheTTL, &cfg.AppPassword)
	if errors.Is(err, sql.ErrNoRows) {
		return SystemConfig{}, nil
	}
	return cfg, err
}

func (r *Repository) ReserveCache(ctx context.Context, reservation CacheReservation) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO cache_ledger (job_id, file_id, reserved_bytes, actual_bytes, state)
VALUES (?, ?, ?, ?, ?)`,
		reservation.JobID, reservation.FileID, reservation.ReservedBytes, reservation.ActualBytes, reservation.State)
	return err
}

func (r *Repository) ReleaseCache(ctx context.Context, jobID string) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE cache_ledger
SET state = 'released', updated_at = CURRENT_TIMESTAMP
WHERE job_id = ?`, jobID)
	return err
}

func (r *Repository) StreamChunks(ctx context.Context, fileID int64) ([]FileChunk, error) {
	return r.ListChunks(ctx, fileID)
}

func (r *Repository) MustFile(ctx context.Context, fileID int64) (FileEntry, error) {
	file, err := r.GetFile(ctx, fileID)
	if err != nil {
		return FileEntry{}, fmt.Errorf("get file %d: %w", fileID, err)
	}
	return file, nil
}
