package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsChecksDoNotCreateOrMigrateDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "bot.db")
	jsonPath := filepath.Join(dir, "config.json")
	cfg, store, err := loadSettings(dbPath, jsonPath, true)
	if err != nil || store != nil || len(cfg.Guilds) != 0 {
		t.Fatalf("fresh check: %v", err)
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatal("check created a DB")
	}
	if err := os.WriteFile(jsonPath, []byte(`{"guilds":{"123":{"voice_text_channel_id":"124"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, store, err = loadSettings(dbPath, jsonPath, true)
	if err != nil || store != nil || cfg.Guilds["123"].VoiceTextChannelID != "124" {
		t.Fatalf("legacy preview: %v", err)
	}
	cfg, store, err = loadSettings(dbPath, jsonPath, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetVoice("123", "126", nil); err != nil {
		t.Fatal(err)
	}
	store.Close()
	// Stale or broken JSON no longer controls the already-migrated database.
	os.WriteFile(jsonPath, []byte("not json"), 0600)
	cfg, store, err = loadSettings(dbPath, jsonPath, true)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if cfg.Guilds["123"].VoiceTextChannelID != "126" {
		t.Fatal("stale JSON overwrote Discord settings")
	}
}
