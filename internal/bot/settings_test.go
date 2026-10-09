package bot

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aki-lua87/marumeshi_bot/internal/storage"
	"github.com/bwmarrin/discordgo"
)

type settingsFixture struct {
	b                 *Bot
	s                 *discordgo.Session
	store             *storage.Store
	managerPermission int64
	botPermission     int64
	channelGuild      string
	channelType       discordgo.ChannelType
	requests          []string
	replies           []string
	callbacks         []discordgo.InteractionResponse
}

func newSettingsFixture(t *testing.T) *settingsFixture {
	t.Helper()
	b, s := testBot(t)
	store, err := storage.Open(filepath.Join(t.TempDir(), "bot.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	if err = store.Import(b.cfg); err != nil {
		t.Fatal(err)
	}
	b.AttachStore(store)
	f := &settingsFixture{b: b, s: s, store: store, managerPermission: discordgo.PermissionManageServer, botPermission: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages, channelGuild: "123", channelType: discordgo.ChannelTypeGuildText}
	s.Client.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		f.requests = append(f.requests, r.Method+" "+r.URL.Path)
		var data any = map[string]any{}
		switch {
		case strings.HasSuffix(r.URL.Path, "/callback"):
			var callback discordgo.InteractionResponse
			if err = json.NewDecoder(r.Body).Decode(&callback); err != nil {
				t.Fatal(err)
			}
			f.callbacks = append(f.callbacks, callback)
		case r.Method == http.MethodPatch:
			var edit discordgo.WebhookEdit
			if err = json.NewDecoder(r.Body).Decode(&edit); err != nil {
				t.Fatal(err)
			}
			if edit.Content != nil {
				f.replies = append(f.replies, *edit.Content)
			}
			if edit.AllowedMentions == nil || len(edit.AllowedMentions.Parse) != 0 {
				t.Fatal("settings replies could ping users")
			}
		case strings.HasSuffix(r.URL.Path, "/guilds/123"):
			data = &discordgo.Guild{ID: "123", OwnerID: "888", Roles: []*discordgo.Role{{ID: "123"}, {ID: "manager-role", Permissions: f.managerPermission}, {ID: "bot-role", Permissions: f.botPermission}}}
		case strings.HasSuffix(r.URL.Path, "/members/777"):
			data = &discordgo.Member{User: &discordgo.User{ID: "777"}, Roles: []string{"manager-role"}}
		case strings.HasSuffix(r.URL.Path, "/members/999"):
			data = &discordgo.Member{User: &discordgo.User{ID: "999"}, Roles: []string{"bot-role"}}
		case strings.Contains(r.URL.Path, "/channels/"):
			parts := strings.Split(r.URL.Path, "/")
			data = &discordgo.Channel{ID: parts[len(parts)-1], GuildID: f.channelGuild, Type: f.channelType}
		default:
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
		raw, _ := json.Marshal(data)
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw))), Request: r}, nil
	})
	return f
}

func managerInteraction(data discordgo.InteractionData) *discordgo.Interaction {
	return &discordgo.Interaction{ID: "456", AppID: "999", Token: "fake-interaction-token", GuildID: "123", Member: &discordgo.Member{User: &discordgo.User{ID: "777"}, Permissions: discordgo.PermissionManageServer}, Data: data}
}

func setupSelection(channel string) *discordgo.Interaction {
	return managerInteraction(discordgo.MessageComponentInteractionData{CustomID: "setup:777", ComponentType: discordgo.ChannelSelectMenuComponent, Values: []string{channel}})
}

func TestSetupSavesAndImmediatelyRoutesVoice(t *testing.T) {
	f := newSettingsFixture(t)
	f.b.configure(f.s, setupSelection("126"), "select:setup")
	cfg, err := f.store.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Guilds["123"].VoiceTextChannelID != "126" || cfg.Guilds["223"].VoiceTextChannelID != "224" || cfg.Guilds["123"].FallbackReply != "A" {
		t.Fatalf("wrong persisted settings: %+v", cfg)
	}
	f.b.onVoice(f.s, &discordgo.VoiceStateUpdate{VoiceState: &discordgo.VoiceState{GuildID: "123", ChannelID: "125", UserID: "778", Member: &discordgo.Member{User: &discordgo.User{ID: "778", Username: "member"}}}})
	if message := <-f.b.queue; message.channelID != "126" {
		t.Fatalf("configuration not applied: %+v", message)
	}
	f.b.configure(f.s, managerInteraction(discordgo.ApplicationCommandInteractionData{Name: "voice"}), "voice:disable")
	f.b.onVoice(f.s, &discordgo.VoiceStateUpdate{VoiceState: &discordgo.VoiceState{GuildID: "123", ChannelID: "125", UserID: "779"}})
	if len(f.b.queue) != 0 {
		t.Fatal("disabled notifications still generated")
	}
	cfg, _ = f.store.Snapshot()
	if cfg.Guilds["123"].VoiceTextChannelID != "" {
		t.Fatal("disabled state not persisted")
	}
}

func TestRejectsWrongGuildTypePermissionsAndRevokedManager(t *testing.T) {
	for _, name := range []string{"wrong guild", "wrong type", "cannot post", "revoked manager"} {
		t.Run(name, func(t *testing.T) {
			f := newSettingsFixture(t)
			switch name {
			case "wrong guild":
				f.channelGuild = "223"
			case "wrong type":
				f.channelType = discordgo.ChannelTypeGuildVoice
			case "cannot post":
				f.botPermission = discordgo.PermissionViewChannel
			case "revoked manager":
				f.managerPermission = 0
			}
			f.b.configure(f.s, setupSelection("126"), "select:setup")
			cfg, _ := f.store.Snapshot()
			if cfg.Guilds["123"].VoiceTextChannelID != "124" {
				t.Fatal("invalid selection persisted")
			}
			if len(f.replies) != 1 || strings.Contains(f.replies[0], "設定を保存しました") {
				t.Fatal("invalid selection accepted")
			}
		})
	}
}

func TestInteractionAcknowledgesPrivatelyAndRejectsNonManagers(t *testing.T) {
	f := newSettingsFixture(t)
	i := managerInteraction(discordgo.ApplicationCommandInteractionData{Name: "setup"})
	i.Type = discordgo.InteractionApplicationCommand
	f.b.onInteraction(f.s, &discordgo.InteractionCreate{Interaction: i})
	if len(f.callbacks) != 1 || f.callbacks[0].Type != discordgo.InteractionResponseDeferredChannelMessageWithSource || f.callbacks[0].Data.Flags != discordgo.MessageFlagsEphemeral || len(f.b.tasks) != 1 {
		t.Fatal("not acknowledged before network work")
	}
	i.Member.Permissions = 0
	f.b.onInteraction(f.s, &discordgo.InteractionCreate{Interaction: i})
	if len(f.b.tasks) != 1 || len(f.callbacks) != 2 || f.callbacks[1].Type != discordgo.InteractionResponseChannelMessageWithSource {
		t.Fatal("non-manager entered settings worker")
	}
	i.GuildID = ""
	i.Member.Permissions = discordgo.PermissionManageServer
	f.b.onInteraction(f.s, &discordgo.InteractionCreate{Interaction: i})
	if len(f.b.tasks) != 1 {
		t.Fatal("DM entered settings worker")
	}
	if _, ok := interactionAction(managerInteraction(discordgo.MessageComponentInteractionData{CustomID: "setup:another-user"})); ok {
		t.Fatal("another user's picker accepted")
	}
}

func TestPickerVoiceTargetsAndAll(t *testing.T) {
	f := newSettingsFixture(t)
	f.channelType = discordgo.ChannelTypeGuildVoice
	i := managerInteraction(discordgo.MessageComponentInteractionData{CustomID: "voice:777", ComponentType: discordgo.ChannelSelectMenuComponent, Values: []string{"125", "127"}})
	f.b.configure(f.s, i, "select:voice")
	cfg, _ := f.store.Snapshot()
	if len(cfg.Guilds["123"].VoiceChannelIDs) != 2 {
		t.Fatal("targets not stored")
	}
	f.b.configure(f.s, managerInteraction(discordgo.ApplicationCommandInteractionData{Name: "voice"}), "voice:all")
	cfg, _ = f.store.Snapshot()
	if len(cfg.Guilds["123"].VoiceChannelIDs) != 0 {
		t.Fatal("all VC not applied")
	}
	picker := channelPicker("777", false)[0].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
	if picker.MenuType != discordgo.ChannelSelectMenu || picker.MaxValues != 1 {
		t.Fatal("not a channel picker")
	}
}

func TestNewGuildGetsOneGuideAndSettingsSurviveTemporaryOutage(t *testing.T) {
	f := newSettingsFixture(t)
	event := &discordgo.GuildCreate{Guild: &discordgo.Guild{ID: "323", SystemChannelID: "324"}}
	f.b.onGuildCreate(f.s, event)
	f.b.onGuildCreate(f.s, event)
	if len(f.b.queue) != 1 {
		t.Fatal("missing or repeated onboarding")
	}
	if g, ok := f.b.guild("323"); !ok || g.VoiceTextChannelID != "" {
		t.Fatal("unconfigured guild cannot use bot")
	}
	f.b.onGuildDelete(f.s, &discordgo.GuildDelete{Guild: &discordgo.Guild{ID: "323", Unavailable: true}})
	if _, ok := f.b.guild("323"); !ok {
		t.Fatal("temporary outage removed guild")
	}
	f.b.onGuildDelete(f.s, &discordgo.GuildDelete{Guild: &discordgo.Guild{ID: "323"}})
	if _, ok := f.b.guild("323"); ok {
		t.Fatal("departed guild stayed active")
	}
	cfg, _ := f.store.Snapshot()
	if _, ok := cfg.Guilds["323"]; ok {
		t.Fatal("departed guild not archived")
	}
}

func TestQueuedVoiceDestinationIsInvalidatedAfterChange(t *testing.T) {
	f := newSettingsFixture(t)
	g, _ := f.b.guild("123")
	g.VoiceTextChannelID = "126"
	f.b.setGuild("123", g)
	f.b.process(f.s, outgoing{guildID: "123", channelID: "124", text: "old", prepared: true, voice: true})
	if len(f.requests) != 0 {
		t.Fatal("queued voice sent to old destination")
	}
}

func TestReadyArchivesGuildsRemovedWhileOffline(t *testing.T) {
	f := newSettingsFixture(t)
	f.b.onReady(f.s, &discordgo.Ready{User: &discordgo.User{ID: "999"}, Guilds: []*discordgo.Guild{{ID: "123", Unavailable: true}}})
	if _, ok := f.b.guild("123"); !ok {
		t.Fatal("unavailable guild removed")
	}
	if _, ok := f.b.guild("223"); ok {
		t.Fatal("offline departure remained active")
	}
	cfg, err := f.store.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Guilds["223"]; ok {
		t.Fatal("offline departure not persisted")
	}
}
