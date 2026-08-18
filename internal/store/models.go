package store

type Directory struct {
	ID        int64
	ParentID  *int64
	Name      string
	Path      string
	CreatedAt string
	UpdatedAt string
}

type FileEntry struct {
	ID        int64
	ParentID  int64
	Name      string
	Size      int64
	MIME      string
	SHA256    string
	Status    string
	Source    string
	CreatedAt string
	UpdatedAt string
}

type FileChunk struct {
	ID                int64
	FileID            int64
	ChunkIndex        int
	ChunkSize         int64
	ChunkSHA256       string
	TelegramChatID    int64
	TelegramMessageID int64
	CreatedAt         string
}

type UploadJob struct {
	ID             int64
	FileID         int64
	Source         string
	Stage          string
	RetryCount     int
	LastError      string
	LastChunkIndex int
	StagedPath     string
	CreatedAt      string
	UpdatedAt      string
}

type SystemConfig struct {
	TelegramSessionBlob  string
	TelegramTargetChatID int64
	DefaultChunkSize     int64
	MaxStagingBytes      int64
	DownloadCacheTTL     int64
	AppPassword          string
}

type CacheReservation struct {
	JobID         string
	FileID        int64
	ReservedBytes int64
	ActualBytes   int64
	State         string
}
