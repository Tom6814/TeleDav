package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"telegram-webdav/internal/api"
	"telegram-webdav/internal/config"
	"telegram-webdav/internal/jobs"
	"telegram-webdav/internal/store"
	"telegram-webdav/internal/telegram"
	"telegram-webdav/internal/vfs"
	appwebdav "telegram-webdav/internal/webdav"
)

func main() {
	env := map[string]string{
		"APP_LISTEN_ADDR":           os.Getenv("APP_LISTEN_ADDR"),
		"APP_DB_PATH":               os.Getenv("APP_DB_PATH"),
		"APP_STAGING_DIR":           os.Getenv("APP_STAGING_DIR"),
		"APP_WEB_DIR":               os.Getenv("APP_WEB_DIR"),
		"APP_PASSWORD":              os.Getenv("APP_PASSWORD"),
		"APP_SESSION_SECRET":        os.Getenv("APP_SESSION_SECRET"),
		"APP_DEFAULT_CHUNK_SIZE":    os.Getenv("APP_DEFAULT_CHUNK_SIZE"),
		"APP_MAX_STAGING_BYTES":     os.Getenv("APP_MAX_STAGING_BYTES"),
		"APP_TELEGRAM_CHAT_ID":      os.Getenv("APP_TELEGRAM_CHAT_ID"),
		"APP_TELEGRAM_API_ID":       os.Getenv("APP_TELEGRAM_API_ID"),
		"APP_TELEGRAM_API_HASH":     os.Getenv("APP_TELEGRAM_API_HASH"),
		"APP_TELEGRAM_SESSION_PATH": os.Getenv("APP_TELEGRAM_SESSION_PATH"),
	}
	cfg, err := config.Load(env)
	if err != nil {
		log.Fatal(err)
	}
	if cfg.AppPassword == "" {
		log.Fatal("APP_PASSWORD is required")
	}
	if cfg.SessionSecret == "" {
		log.Fatal("APP_SESSION_SECRET is required")
	}

	repo, err := store.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Close()
	if err := repo.Migrate(context.Background()); err != nil {
		log.Fatal(err)
	}
	if _, err := repo.EnsureRoot(context.Background()); err != nil {
		log.Fatal(err)
	}
	storedCfg, err := repo.GetSystemConfig(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	cfg = applyStoredConfig(cfg, env, storedCfg)
	if err := os.MkdirAll(cfg.StagingDir, 0o755); err != nil {
		log.Fatal(err)
	}
	telegramClient := telegram.NewGOTDClient(
		cfg.TelegramChatID,
		cfg.TelegramSessionPath,
		cfg.TelegramAPIID,
		cfg.TelegramAPIHash,
	)
	uploader := jobs.NewUploader(repo, telegramClient)
	downloader := jobs.NewDownloader(repo, telegramClient)
	fsService := vfs.New(repo)
	jobController := jobs.NewJobController(repo, uploader, cfg.DefaultChunkSize)
	if err := jobs.RunRecovery(context.Background(), jobs.NewRecoveryService(repo, uploader, cfg.DefaultChunkSize)); err != nil {
		log.Fatal(err)
	}
	handler := api.NewRouter(api.Dependencies{
		AppPassword:      cfg.AppPassword,
		SessionSecret:    cfg.SessionSecret,
		WebDir:           cfg.WebDir,
		StagingDir:       cfg.StagingDir,
		DefaultChunkSize: cfg.DefaultChunkSize,
		Quota:            jobs.NewQuota(cfg.MaxStagingBytes),
		ConfigStore:      repo,
		FS:               fsService,
		Jobs:             jobController,
		Retryer:          jobController,
		Uploader:         uploader,
		Downloader:       downloader,
		WebDAV: appwebdav.New(&appwebdav.Service{
			FS:               fsService,
			Uploader:         uploader,
			Downloader:       downloader,
			StagingDir:       cfg.StagingDir,
			DefaultChunkSize: cfg.DefaultChunkSize,
		}),
	})

	log.Printf("listening on %s", cfg.ListenAddr)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, handler))
}

func applyStoredConfig(cfg config.Config, env map[string]string, stored store.SystemConfig) config.Config {
	if env["APP_PASSWORD"] == "" && stored.AppPassword != "" {
		cfg.AppPassword = stored.AppPassword
	}
	if env["APP_DEFAULT_CHUNK_SIZE"] == "" && stored.DefaultChunkSize > 0 {
		cfg.DefaultChunkSize = stored.DefaultChunkSize
	}
	if env["APP_MAX_STAGING_BYTES"] == "" && stored.MaxStagingBytes > 0 {
		cfg.MaxStagingBytes = stored.MaxStagingBytes
	}
	if env["APP_TELEGRAM_CHAT_ID"] == "" && stored.TelegramTargetChatID != 0 {
		cfg.TelegramChatID = stored.TelegramTargetChatID
	}
	return cfg
}
