package storage

import (
	"database/sql"
	"path/filepath"
	"sync"
	"testing"

	"github.com/aki-lua87/marumeshi_bot/internal/config"
)

func testStore(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bot.db")
	s, err := Open(path, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, path
}

func TestImportIsAtomicAndDoesNotOverwriteDiscordSettings(t *testing.T) {
	s, path := testStore(t)
	legacy := config.Config{Guilds: map[string]config.Guild{
		"123": {VoiceTextChannelID: "124", VoiceChannelIDs: []string{"125"}, SpreadsheetURL: "https://example.org/sheet", FallbackReply: "existing"},
		"223": {VoiceTextChannelID: "224"},
	}}
	if err := s.Import(legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetVoice("123", "126", nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Import(legacy); err != nil {
		t.Fatal(err)
	}
	s.Close()
	reopened, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	cfg, err := reopened.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Guilds["123"].VoiceTextChannelID != "126" || cfg.Guilds["123"].FallbackReply != "existing" || cfg.Guilds["123"].SpreadsheetURL != "https://example.org/sheet" {
		t.Fatalf("migration overwrote settings: %+v", cfg)
	}
	if cfg.Guilds["223"].VoiceTextChannelID != "224" {
		t.Fatal("guild isolation lost")
	}
	if imported, err := reopened.Imported(); err != nil || !imported {
		t.Fatal("migration marker lost")
	}
}

func TestGuildLifecycleAndDisableSurviveReopen(t *testing.T) {
	s, path := testStore(t)
	if g, created, err := s.EnsureGuild("123"); err != nil || !created || g.VoiceTextChannelID != "" {
		t.Fatalf("new guild: %v %v", created, err)
	}
	if _, created, err := s.EnsureGuild("123"); err != nil || created {
		t.Fatal("duplicate invitation")
	}
	if _, err := s.SetVoice("123", "124", []string{"125"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetVoice("123", "", []string{"125"}); err != nil {
		t.Fatal(err)
	}
	if err := s.LeaveGuild("123"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetVoice("123", "124", nil); err == nil {
		t.Fatal("departed guild accepted a stale picker")
	}
	cfg, err := s.Snapshot()
	if err != nil || len(cfg.Guilds) != 0 {
		t.Fatal("departed guild is active")
	}
	s.Close()
	s, err = Open(path, false)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	g, created, err := s.EnsureGuild("123")
	if err != nil || created || g.VoiceTextChannelID != "" || len(g.VoiceChannelIDs) != 1 {
		t.Fatalf("lost disabled settings: %+v", g)
	}
}

func TestBackupIsReadableAndSourceRemainsUnchanged(t *testing.T) {
	s, _ := testStore(t)
	s.EnsureGuild("123")
	if _, err := s.SetVoice("123", "124", nil); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "backup.db")
	if err := s.Backup(path); err != nil {
		t.Fatal(err)
	}
	if err := s.Backup(path); err == nil {
		t.Fatal("overwrote backup")
	}
	s.SetVoice("123", "126", nil)
	backup, err := Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	cfg, err := backup.Snapshot()
	if err != nil || cfg.Guilds["123"].VoiceTextChannelID != "124" {
		t.Fatal("backup not independent")
	}
	if _, err := backup.SetVoice("123", "127", nil); err == nil {
		t.Fatal("readonly check wrote DB")
	}
}

func TestRejectsFutureSchemaAndInvalidChanges(t *testing.T) {
	s, path := testStore(t)
	s.EnsureGuild("123")
	if _, err := s.SetVoice("123", "not-an-id", nil); err == nil {
		t.Fatal("invalid ID saved")
	}
	if _, err := s.SetVoice("123", "124", []string{"125", "125"}); err == nil {
		t.Fatal("duplicate targets saved")
	}
	s.Close()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("PRAGMA user_version=999"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if db, err := Open(path, false); err == nil {
		db.Close()
		t.Fatal("downgraded future schema")
	}
}

func TestConcurrentChangesAreSerialized(t *testing.T) {
	s, _ := testStore(t)
	s.EnsureGuild("123")
	s.EnsureGuild("223")
	var wg sync.WaitGroup
	for n := 0; n < 20; n++ {
		wg.Go(func() {
			if _, err := s.SetVoice("123", "124", nil); err != nil {
				t.Error(err)
			}
			if _, err := s.SetVoice("223", "224", nil); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	cfg, err := s.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Guilds["123"].VoiceTextChannelID != "124" || cfg.Guilds["223"].VoiceTextChannelID != "224" {
		t.Fatal("concurrent updates crossed guilds")
	}
}
