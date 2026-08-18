package config

import (
	"fmt"
	"strconv"
)

type Config struct {
	ListenAddr          string
	DatabasePath        string
	StagingDir          string
	WebDir              string
	AppPassword         string
	SessionSecret       string
	DefaultChunkSize    int64
	MaxStagingBytes     int64
	TelegramChatID      int64
	TelegramAPIID       int
	TelegramAPIHash     string
	TelegramSessionPath string
}

func Load(env map[string]string) (Config, error) {
	cfg := Config{
		ListenAddr:          ":8080",
		DatabasePath:        "data/app.db",
		StagingDir:          "data/staging",
		WebDir:              "web/build/web",
		DefaultChunkSize:    20 << 20,
		MaxStagingBytes:     1 << 30,
		TelegramSessionPath: "data/telegram-spool",
	}
	if v := env["APP_LISTEN_ADDR"]; v != "" {
		cfg.ListenAddr = v
	}
	if v := env["APP_DB_PATH"]; v != "" {
		cfg.DatabasePath = v
	}
	if v := env["APP_STAGING_DIR"]; v != "" {
		cfg.StagingDir = v
	}
	if v := env["APP_WEB_DIR"]; v != "" {
		cfg.WebDir = v
	}
	if v := env["APP_PASSWORD"]; v != "" {
		cfg.AppPassword = v
	}
	if v := env["APP_SESSION_SECRET"]; v != "" {
		cfg.SessionSecret = v
	}
	if v := env["APP_DEFAULT_CHUNK_SIZE"]; v != "" {
		size, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("parse APP_DEFAULT_CHUNK_SIZE: %w", err)
		}
		cfg.DefaultChunkSize = size
	}
	if v := env["APP_MAX_STAGING_BYTES"]; v != "" {
		size, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("parse APP_MAX_STAGING_BYTES: %w", err)
		}
		cfg.MaxStagingBytes = size
	}
	if v := env["APP_TELEGRAM_CHAT_ID"]; v != "" {
		chatID, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("parse APP_TELEGRAM_CHAT_ID: %w", err)
		}
		cfg.TelegramChatID = chatID
	}
	if v := env["APP_TELEGRAM_API_ID"]; v != "" {
		apiID, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("parse APP_TELEGRAM_API_ID: %w", err)
		}
		cfg.TelegramAPIID = apiID
	}
	if v := env["APP_TELEGRAM_API_HASH"]; v != "" {
		cfg.TelegramAPIHash = v
	}
	if v := env["APP_TELEGRAM_SESSION_PATH"]; v != "" {
		cfg.TelegramSessionPath = v
	}
	return cfg, nil
}
