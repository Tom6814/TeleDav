package main

import (
	"log"
	"net/http"
	"os"

	"telegram-webdav/internal/config"
)

func main() {
	env := map[string]string{
		"APP_LISTEN_ADDR":        os.Getenv("APP_LISTEN_ADDR"),
		"APP_DB_PATH":            os.Getenv("APP_DB_PATH"),
		"APP_STAGING_DIR":        os.Getenv("APP_STAGING_DIR"),
		"APP_WEB_DIR":            os.Getenv("APP_WEB_DIR"),
		"APP_PASSWORD":           os.Getenv("APP_PASSWORD"),
		"APP_SESSION_SECRET":     os.Getenv("APP_SESSION_SECRET"),
		"APP_DEFAULT_CHUNK_SIZE": os.Getenv("APP_DEFAULT_CHUNK_SIZE"),
		"APP_MAX_STAGING_BYTES":  os.Getenv("APP_MAX_STAGING_BYTES"),
	}
	cfg, err := config.Load(env)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("listening on %s", cfg.ListenAddr)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, http.NotFoundHandler()))
}
