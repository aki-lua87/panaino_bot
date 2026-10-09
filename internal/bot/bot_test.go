package bot

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/aki-lua87/marumeshi_bot/internal/config"
	"github.com/aki-lua87/marumeshi_bot/internal/features"
	"github.com/aki-lua87/marumeshi_bot/internal/integrations"
	"github.com/bwmarrin/discordgo"
)

func testBot(t *testing.T) (*Bot, *discordgo.Session) {
	t.Helper()
	s, err := discordgo.New("Bot fake-token")
	if err != nil {
		t.Fatal(err)
	}
	for id, channels := range map[string][]string{"123": {"124", "125"}, "223": {"224", "225"}} {
		if err := s.State.GuildAdd(&discordgo.Guild{ID: id}); err != nil {
			t.Fatal(err)
		}
		for _, channelID := range channels {
			if err := s.State.ChannelAdd(&discordgo.Channel{ID: channelID, GuildID: id, Name: "VC"}); err != nil {
				t.Fatal(err)
			}
		}
	}
	b := New(context.Background(), config.Config{Guilds: map[string]config.Guild{
		"123": {VoiceTextChannelID: "124", FallbackReply: "A"},
		"223": {VoiceTextChannelID: "224", FallbackReply: "B"},
	}}, features.Engine{API: integrations.New()}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	b.id.Store("999")
	return b, s
}

func TestMentionParsing(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		valid       bool
	}{
		{"<@999> 晴れる屋 屋根", "晴れる屋 屋根", true},
		{"　<@!999>　ひるめし", "ひるめし", true},
		{"<@999>日本語", "日本語", true},
		{"<@999>", "", true},
		{"<@9999> hello", "", false},
		{"hello <@999>", "", false},
		{"<@other> hi", "", false},
	} {
		text, ok := MentionText(tc.input, "999")
		if text != tc.want || ok != tc.valid {
			t.Fatalf("input=%q got=%q/%v", tc.input, text, ok)
		}
	}
}

func TestMessageFiltersAndGuildRouting(t *testing.T) {
	b, s := testBot(t)
	for _, guildID := range []string{"123", "223"} {
		b.onMessage(s, &discordgo.MessageCreate{Message: &discordgo.Message{GuildID: guildID, ChannelID: guildID, Content: "<@999>こんにちは", Author: &discordgo.User{ID: "user"}}})
		message := <-b.queue
		if message.guildID != guildID || message.text != "こんにちは" {
			t.Fatalf("wrong routing: %+v", message)
		}
	}
	for _, m := range []*discordgo.Message{
		{GuildID: "123", Content: "<@999> hi", Author: &discordgo.User{Bot: true}},
		{GuildID: "123", Content: "<@999> hi", WebhookID: "hook", Author: &discordgo.User{}},
		{GuildID: "unknown", Content: "<@999> hi", Author: &discordgo.User{}},
		{GuildID: "", Content: "<@999> hi", Author: &discordgo.User{}},
		{GuildID: "123", Content: "ordinary chat", Author: &discordgo.User{}},
		{GuildID: "123", Content: "<@999> hi"},
	} {
		b.onMessage(s, &discordgo.MessageCreate{Message: m})
	}
	if len(b.queue) != 0 {
		t.Fatal("Bot, webhook, DM, or unconfigured guild was accepted")
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSendingCannotCrossGuildAndSuppressesMentions(t *testing.T) {
	b, s := testBot(t)
	var requests []string
	s.Client.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		requests = append(requests, r.URL.Path)
		var data struct {
			Content         string `json:"content"`
			AllowedMentions struct {
				Parse []string `json:"parse"`
			} `json:"allowed_mentions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			t.Fatal(err)
		}
		if len(data.AllowedMentions.Parse) != 0 {
			t.Fatal("external response may trigger mentions")
		}
		if data.Content == "" {
			t.Fatal("empty response")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"id":"message"}`)), Request: r}, nil
	})
	b.send(context.Background(), s, "123", "224", "@everyone")
	if len(requests) != 0 {
		t.Fatal("notification crossed guild boundary")
	}
	b.process(s, outgoing{guildID: "123", channelID: "124", text: "hello"})
	b.process(s, outgoing{guildID: "223", channelID: "224", text: "hello"})
	if len(requests) != 2 || !strings.Contains(requests[0], "/124/messages") || !strings.Contains(requests[1], "/224/messages") {
		t.Fatalf("wrong sends: %v", requests)
	}
}

func TestVoiceNotificationsAndMute(t *testing.T) {
	b, s := testBot(t)
	for _, pair := range [][2]string{{"123", "125"}, {"223", "225"}} {
		event := &discordgo.VoiceStateUpdate{VoiceState: &discordgo.VoiceState{GuildID: pair[0], UserID: "same-user", ChannelID: pair[1], Member: &discordgo.Member{User: &discordgo.User{Username: "user"}}}}
		b.onVoice(s, event)
		message := <-b.queue
		if message.channelID != b.cfg.Guilds[pair[0]].VoiceTextChannelID || !message.prepared {
			t.Fatalf("wrong VC route: %+v", message)
		}
		b.onVoice(s, event)
		if len(b.queue) != 0 {
			t.Fatal("mute generated duplicate join")
		}
		event.ChannelID = ""
		b.onVoice(s, event)
		message = <-b.queue
		if !strings.Contains(message.text, "滞在時間:[0s]") {
			t.Fatalf("invalid duration: %q", message.text)
		}
	}
}

func TestSeededVoiceOccupantHasUnknownDuration(t *testing.T) {
	b, s := testBot(t)
	b.onGuildCreate(s, &discordgo.GuildCreate{Guild: &discordgo.Guild{ID: "123", VoiceStates: []*discordgo.VoiceState{{UserID: "user", ChannelID: "125"}}}})
	b.onVoice(s, &discordgo.VoiceStateUpdate{VoiceState: &discordgo.VoiceState{GuildID: "123", UserID: "user"}})
	message := <-b.queue
	if strings.Contains(message.text, "滞在時間") {
		t.Fatal("startup occupant was assigned a fabricated duration")
	}
}

func TestContentLimitPreservesUnicode(t *testing.T) {
	for _, input := range []string{strings.Repeat("あ", 2100), strings.Repeat("🍞", 1500), "invalid\xffUTF8"} {
		out := LimitContent(input)
		if !utf8.ValidString(out) || len(utf16.Encode([]rune(out))) > 2000 {
			t.Fatalf("invalid Discord content: %q", out)
		}
	}
}
