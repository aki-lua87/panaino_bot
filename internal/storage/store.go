// Package storage persists guild settings in a local, CGO-free SQLite database.
package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aki-lua87/marumeshi_bot/internal/config"
	_ "modernc.org/sqlite"
)

const schemaVersion = 1

type Store struct{ db *sql.DB }

func Open(path string, readOnly bool) (*Store, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if !readOnly {
		// Create privately before SQLite opens the file, including on a normal shell umask.
		f, err := os.OpenFile(absolute, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			f.Close()
		} else if !os.IsExist(err) {
			return nil, err
		}
		if err = os.Chmod(absolute, 0600); err != nil {
			return nil, err
		}
	}
	uriPath := filepath.ToSlash(absolute)
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	u := url.URL{Scheme: "file", Path: uriPath}
	q := u.Query()
	if readOnly {
		q.Set("mode", "ro")
	} else {
		q.Set("mode", "rw")
	}
	q.Add("_pragma", "busy_timeout(5000)")
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err = s.initialize(readOnly); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) initialize(readOnly bool) error {
	var version int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version > schemaVersion {
		return errors.New("このDBは新しいBot版で作成されています。対応する版を使用してください")
	}
	if readOnly {
		if version != schemaVersion {
			return errors.New("DBの初期化が必要です")
		}
	} else if version == 0 {
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		for _, statement := range []string{
			`CREATE TABLE guilds (guild_id TEXT PRIMARY KEY, settings TEXT NOT NULL, active INTEGER NOT NULL DEFAULT 1)`,
			`CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT NOT NULL)`,
			`PRAGMA user_version=1`,
		} {
			if _, err = tx.Exec(statement); err != nil {
				return err
			}
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	if !readOnly {
		// DELETE journaling permits safe backups on Ubuntu's old Python SQLite module.
		if _, err := s.db.Exec("PRAGMA journal_mode=DELETE"); err != nil {
			return err
		}
		if _, err := s.db.Exec("PRAGMA synchronous=FULL"); err != nil {
			return err
		}
	}
	var result string
	if err := s.db.QueryRow("PRAGMA quick_check").Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return errors.New("DBの整合性検査に失敗しました")
	}
	_, err := s.Snapshot()
	return err
}

func (s *Store) Close() error { return s.db.Close() }

// Snapshot is independent of the database and excludes guilds the Bot has left.
func (s *Store) Snapshot() (config.Config, error) {
	cfg := config.Config{Guilds: map[string]config.Guild{}}
	rows, err := s.db.Query("SELECT guild_id,settings FROM guilds WHERE active=1")
	if err != nil {
		return cfg, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, raw string
		var g config.Guild
		if err = rows.Scan(&id, &raw); err != nil {
			return cfg, err
		}
		if err = json.Unmarshal([]byte(raw), &g); err != nil {
			return cfg, errors.New("DB内の設定を読み込めません")
		}
		if err = (config.Config{Guilds: map[string]config.Guild{id: g}}).Validate(); err != nil {
			return cfg, err
		}
		cfg.Guilds[id] = g
	}
	return cfg, rows.Err()
}

// Import runs exactly once; existing Discord-managed settings are never overwritten.
func (s *Store) Import(cfg config.Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var imported int
	if err = tx.QueryRow("SELECT COUNT(*) FROM metadata WHERE key='legacy_imported'").Scan(&imported); err != nil {
		return err
	}
	if imported != 0 {
		return nil
	}
	for id, g := range cfg.Guilds {
		raw, _ := json.Marshal(g)
		if _, err = tx.Exec("INSERT INTO guilds(guild_id,settings) VALUES(?,?) ON CONFLICT(guild_id) DO NOTHING", id, string(raw)); err != nil {
			return err
		}
	}
	if _, err = tx.Exec("INSERT INTO metadata(key,value) VALUES('legacy_imported','1')"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Imported() (bool, error) {
	var n int
	err := s.db.QueryRow("SELECT COUNT(*) FROM metadata WHERE key='legacy_imported'").Scan(&n)
	return n != 0, err
}

func (s *Store) EnsureGuild(id string) (config.Guild, bool, error) {
	g := config.Guild{FallbackReply: "まるめし"}
	if err := (config.Config{Guilds: map[string]config.Guild{id: g}}).Validate(); err != nil {
		return g, false, err
	}
	raw, _ := json.Marshal(g)
	result, err := s.db.Exec("INSERT INTO guilds(guild_id,settings) VALUES(?,?) ON CONFLICT(guild_id) DO NOTHING", id, string(raw))
	if err != nil {
		return g, false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return g, false, err
	}
	if _, err = s.db.Exec("UPDATE guilds SET active=1 WHERE guild_id=?", id); err != nil {
		return g, false, err
	}
	var text string
	if err = s.db.QueryRow("SELECT settings FROM guilds WHERE guild_id=?", id).Scan(&text); err != nil {
		return g, false, err
	}
	err = json.Unmarshal([]byte(text), &g)
	return g, n != 0, err
}

func (s *Store) SetVoice(id, channel string, channels []string) (config.Guild, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return config.Guild{}, err
	}
	defer tx.Rollback()
	var raw string
	var g config.Guild
	if err = tx.QueryRow("SELECT settings FROM guilds WHERE guild_id=? AND active=1", id).Scan(&raw); err != nil {
		return g, err
	}
	if err = json.Unmarshal([]byte(raw), &g); err != nil {
		return g, err
	}
	g.VoiceTextChannelID = channel
	g.VoiceChannelIDs = channels
	if err = (config.Config{Guilds: map[string]config.Guild{id: g}}).Validate(); err != nil {
		return g, err
	}
	data, _ := json.Marshal(g)
	if _, err = tx.Exec("UPDATE guilds SET settings=? WHERE guild_id=?", string(data), id); err != nil {
		return g, err
	}
	return g, tx.Commit()
}

func (s *Store) LeaveGuild(id string) error {
	_, err := s.db.Exec("UPDATE guilds SET active=0 WHERE guild_id=?", id)
	return err
}

// Backup creates a consistent snapshot even while the running Bot modifies settings.
func (s *Store) Backup(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if _, err = os.Stat(absolute); !os.IsNotExist(err) {
		return errors.New("バックアップ先には未使用のパスを指定してください")
	}
	f, err := os.OpenFile(absolute, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	// VACUUM INTO accepts an empty file, keeping it private throughout the backup.
	// It is executed by the bundled SQLite, not the OS sqlite3 version.
	if _, err = s.db.Exec("VACUUM INTO ?", absolute); err != nil {
		os.Remove(absolute)
		return fmt.Errorf("DBをバックアップできません: %w", err)
	}
	return os.Chmod(absolute, 0600)
}
