package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/aki-lua87/marumeshi_bot/internal/config"
	"github.com/bwmarrin/discordgo"
)

type fakeDiscord func(*http.Request) (*http.Response, error)

func (f fakeDiscord) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDiscordInspectionChecksPermissionsWithoutPosting(t *testing.T) {
	for _, tc := range []struct {
		name, permissions, channelGuild string
		valid                           bool
	}{
		{"valid", "3072", "123", true},
		{"cannot send", "1024", "123", false},
		{"wrong guild", "3072", "223", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := discordgo.New("Bot fake-token")
			if err != nil {
				t.Fatal(err)
			}
			s.Client.Transport = fakeDiscord(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodGet {
					t.Fatal("inspection must never post or mutate")
				}
				var data any
				switch {
				case strings.HasSuffix(r.URL.Path, "/users/@me"):
					data = map[string]any{"id": "999", "username": "まるめし"}
				case strings.HasSuffix(r.URL.Path, "/guilds/123"):
					data = map[string]any{"id": "123", "name": "server", "roles": []any{map[string]any{"id": "123", "permissions": tc.permissions}}}
				case strings.HasSuffix(r.URL.Path, "/guilds/123/members/999"):
					data = map[string]any{"user": map[string]any{"id": "999"}, "roles": []string{}}
				case strings.HasSuffix(r.URL.Path, "/channels/124"):
					data = map[string]any{"id": "124", "guild_id": tc.channelGuild, "type": 0}
				default:
					t.Fatalf("unexpected request: %s", r.URL.Path)
				}
				body, _ := json.Marshal(data)
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
			})
			err = inspectDiscord(s, config.Config{Guilds: map[string]config.Guild{"123": {VoiceTextChannelID: "124"}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
}
