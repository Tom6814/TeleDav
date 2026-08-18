package store

type Directory struct {
	ID        int64  `json:"id"`
	ParentID  *int64 `json:"parent_id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type FileEntry struct {
	ID        int64  `json:"id"`
	ParentID  int64  `json:"parent_id"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	MIME      string `json:"mime"`
	SHA256    string `json:"sha256"`
	Status    string `json:"status"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type FileChunk struct {
	ID                int64  `json:"id"`
	FileID            int64  `json:"file_id"`
	ChunkIndex        int    `json:"chunk_index"`
	ChunkSize         int64  `json:"chunk_size"`
	ChunkSHA256       string `json:"chunk_sha256"`
	TelegramChatID    int64  `json:"telegram_chat_id"`
	TelegramMessageID int64  `json:"telegram_message_id"`
	CreatedAt         string `json:"created_at"`
}

type UploadJob struct {
	ID             int64  `json:"id"`
	FileID         int64  `json:"file_id"`
	Source         string `json:"source"`
	Stage          string `json:"stage"`
	RetryCount     int    `json:"retry_count"`
	LastError      string `json:"last_error"`
	LastChunkIndex int    `json:"last_chunk_index"`
	StagedPath     string `json:"staged_path"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type SystemConfig struct {
	TelegramSessionBlob  string `json:"telegram_session_blob"`
	TelegramTargetChatID int64  `json:"telegram_target_chat_id"`
	DefaultChunkSize     int64  `json:"default_chunk_size"`
	MaxStagingBytes      int64  `json:"max_staging_bytes"`
	DownloadCacheTTL     int64  `json:"download_cache_ttl_seconds"`
	AppPassword          string `json:"app_password"`
}

type CacheReservation struct {
	JobID         string `json:"job_id"`
	FileID        int64  `json:"file_id"`
	ReservedBytes int64  `json:"reserved_bytes"`
	ActualBytes   int64  `json:"actual_bytes"`
	State         string `json:"state"`
}
