package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecode(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		valid       bool
	}{
		{"two guilds", `{"guilds":{"123":{"voice_text_channel_id":"456"},"789":{"fallback_reply":"まるめし"}}}`, true},
		{"empty", `{"guilds":{}}`, false},
		{"wrong id", `{"guilds":{"not-an-id":{}}}`, false},
		{"wrong channel", `{"guilds":{"123":{"voice_text_channel_id":"not-an-id"}}}`, false},
		{"duplicate VC", `{"guilds":{"123":{"voice_channel_ids":["456","456"]}}}`, false},
		{"unsecured API", `{"guilds":{"123":{"spreadsheet_api":"http://example.com"}}}`, false},
		{"URL credentials", `{"guilds":{"123":{"spreadsheet_api":"https://secret:password@example.com"}}}`, false},
		{"trailing JSON", `{"guilds":{"123":{}}} {}`, false},
		{"trailing invalid data", `{"guilds":{"123":{}}} secret`, false},
		{"old config", `{"DiscordToken":"secret-token"}`, false},
		{"unknown setting", `{"guilds":{"123":{"secret-token":"value"}}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decode(strings.NewReader(tc.input))
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
			if err != nil && (strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "password")) {
				t.Fatal("error leaks a secret")
			}
		})
	}
}

func TestLegacyDoesNotCopySecretsOrEnableDisabledAPI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "setting.json")
	if err := os.WriteFile(path, []byte(`{"DiscordToken":"old-secret", "SpreadsheetAPI":"https://example.com/disabled", "SpreadsheetURL":"https://example.com/sheet", "TC_ID":"456", "VC_ID":"999"}`), 0600); err != nil {
		t.Fatal(err)
	}
	g, err := ImportLegacy(path)
	if err != nil {
		t.Fatal(err)
	}
	if g.VoiceTextChannelID != "456" || g.SpreadsheetAPI != "" || g.FallbackReply != "まるめし" {
		t.Fatalf("wrong migration: %+v", g)
	}
	if len(g.VoiceChannelIDs) != 0 {
		t.Fatal("old unused VC_ID must not change monitoring scope")
	}
}

func TestTokenFromEnvironment(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", " Bot example-token ")
	token, err := TokenFromEnv()
	if err != nil || token != "Bot example-token" {
		t.Fatalf("unexpected token normalization: %v", err)
	}
	t.Setenv("DISCORD_TOKEN", "")
	if _, err := TokenFromEnv(); err == nil {
		t.Fatal("empty token was accepted")
	}
}
