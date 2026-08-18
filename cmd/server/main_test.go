package main

import (
	"testing"

	"telegram-webdav/internal/config"
	"telegram-webdav/internal/store"
)

func TestApplyStoredConfigUsesPersistedValuesWhenEnvUnset(t *testing.T) {
	cfg := config.Config{
		AppPassword:       "",
		DefaultChunkSize:  20 << 20,
		MaxStagingBytes:   1 << 30,
		TelegramChatID:    0,
		SessionSecret:     "session-secret",
		TelegramSessionPath: "data/telegram-spool",
	}
	env := map[string]string{
		"APP_PASSWORD":           "",
		"APP_DEFAULT_CHUNK_SIZE": "",
		"APP_MAX_STAGING_BYTES":  "",
		"APP_TELEGRAM_CHAT_ID":   "",
	}
	stored := store.SystemConfig{
		AppPassword:          "saved-password",
		DefaultChunkSize:     4096,
		MaxStagingBytes:      8192,
		TelegramTargetChatID: 777,
	}

	got := applyStoredConfig(cfg, env, stored)

	if got.AppPassword != "saved-password" {
		t.Fatalf("got.AppPassword = %q, want %q", got.AppPassword, "saved-password")
	}
	if got.DefaultChunkSize != 4096 {
		t.Fatalf("got.DefaultChunkSize = %d, want 4096", got.DefaultChunkSize)
	}
	if got.MaxStagingBytes != 8192 {
		t.Fatalf("got.MaxStagingBytes = %d, want 8192", got.MaxStagingBytes)
	}
	if got.TelegramChatID != 777 {
		t.Fatalf("got.TelegramChatID = %d, want 777", got.TelegramChatID)
	}
}

func TestApplyStoredConfigPreservesExplicitEnvOverrides(t *testing.T) {
	cfg := config.Config{
		AppPassword:       "env-password",
		DefaultChunkSize:  16384,
		MaxStagingBytes:   32768,
		TelegramChatID:    888,
		SessionSecret:     "session-secret",
		TelegramSessionPath: "data/telegram-spool",
	}
	env := map[string]string{
		"APP_PASSWORD":           "env-password",
		"APP_DEFAULT_CHUNK_SIZE": "16384",
		"APP_MAX_STAGING_BYTES":  "32768",
		"APP_TELEGRAM_CHAT_ID":   "888",
	}
	stored := store.SystemConfig{
		AppPassword:          "saved-password",
		DefaultChunkSize:     4096,
		MaxStagingBytes:      8192,
		TelegramTargetChatID: 777,
	}

	got := applyStoredConfig(cfg, env, stored)

	if got.AppPassword != "env-password" {
		t.Fatalf("got.AppPassword = %q, want %q", got.AppPassword, "env-password")
	}
	if got.DefaultChunkSize != 16384 {
		t.Fatalf("got.DefaultChunkSize = %d, want 16384", got.DefaultChunkSize)
	}
	if got.MaxStagingBytes != 32768 {
		t.Fatalf("got.MaxStagingBytes = %d, want 32768", got.MaxStagingBytes)
	}
	if got.TelegramChatID != 888 {
		t.Fatalf("got.TelegramChatID = %d, want 888", got.TelegramChatID)
	}
}
