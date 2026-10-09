// Package config loads non-secret per-guild configuration. Tokens come from the environment.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
)

var snowflake = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)

type Config struct {
	Guilds map[string]Guild `json:"guilds"`
}

type Guild struct {
	VoiceTextChannelID string   `json:"voice_text_channel_id,omitempty"`
	VoiceChannelIDs    []string `json:"voice_channel_ids,omitempty"`
	SpreadsheetURL     string   `json:"spreadsheet_url,omitempty"`
	SpreadsheetAPI     string   `json:"spreadsheet_api,omitempty"`
	FallbackReply      string   `json:"fallback_reply,omitempty"`
	CoatOfArmsURL      string   `json:"coat_of_arms_url,omitempty"`
	EmergencyURL       string   `json:"emergency_url,omitempty"`
}

func Load(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, errors.New("設定ファイルを開けません")
	}
	defer f.Close()
	return Decode(io.LimitReader(f, 1<<20))
}

func Decode(r io.Reader) (Config, error) {
	var cfg Config
	d := json.NewDecoder(r)
	d.DisallowUnknownFields()
	if err := d.Decode(&cfg); err != nil {
		// Do not echo JSON errors: unknown keys may contain accidentally pasted secrets.
		return cfg, errors.New("設定JSONが不正です（旧setting.jsonは移行が必要です）")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return cfg, errors.New("設定JSONの末尾に余分なデータがあります")
	}
	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	if len(c.Guilds) == 0 {
		return errors.New("guildsにDiscordサーバごとの設定が必要です")
	}
	for id, g := range c.Guilds {
		if !snowflake.MatchString(id) {
			return errors.New("guildsのキーはDiscordサーバIDにしてください")
		}
		if g.VoiceTextChannelID != "" && !snowflake.MatchString(g.VoiceTextChannelID) {
			return fmt.Errorf("サーバ%s: voice_text_channel_idが不正です", id)
		}
		seen := make(map[string]bool)
		for _, vc := range g.VoiceChannelIDs {
			if !snowflake.MatchString(vc) || seen[vc] {
				return fmt.Errorf("サーバ%s: voice_channel_idsが不正または重複しています", id)
			}
			seen[vc] = true
		}
		for _, endpoint := range []string{g.SpreadsheetURL, g.SpreadsheetAPI, g.CoatOfArmsURL, g.EmergencyURL} {
			if endpoint == "" {
				continue
			}
			u, err := url.Parse(endpoint)
			if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
				return fmt.Errorf("サーバ%s: API/リンクは認証情報を含まないHTTPS URLにしてください", id)
			}
		}
	}
	return nil
}

func TokenFromEnv() (string, error) {
	token := strings.TrimSpace(os.Getenv("DISCORD_TOKEN"))
	token = strings.TrimSpace(strings.TrimPrefix(token, "Bot "))
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return "", errors.New("DISCORD_TOKENに有効なBotトークンを設定してください")
	}
	return "Bot " + token, nil
}

func (g Guild) WatchesVoice(channelID string) bool {
	if channelID == "" || g.VoiceTextChannelID == "" {
		return false
	}
	if len(g.VoiceChannelIDs) == 0 {
		return true
	}
	for _, id := range g.VoiceChannelIDs {
		if channelID == id {
			return true
		}
	}
	return false
}
