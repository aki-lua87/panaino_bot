package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf16"

	"github.com/aki-lua87/marumeshi_bot/internal/config"
	"github.com/aki-lua87/marumeshi_bot/internal/features"
	"github.com/aki-lua87/marumeshi_bot/internal/storage"
	"github.com/aki-lua87/marumeshi_bot/internal/voice"
	"github.com/bwmarrin/discordgo"
)

type outgoing struct {
	guildID, channelID, text string
	prepared                 bool
	voice                    bool
}

type Bot struct {
	cfg        config.Config
	cfgMu      sync.RWMutex
	settingsMu sync.Mutex
	store      *storage.Store
	tasks      chan func(*discordgo.Session)
	engine     features.Engine
	voice      *voice.Tracker
	ctx        context.Context
	logger     *slog.Logger
	id         atomic.Value
	queue      chan outgoing
	workers    sync.WaitGroup
}

func New(ctx context.Context, cfg config.Config, engine features.Engine, logger *slog.Logger) *Bot {
	if cfg.Guilds == nil {
		cfg.Guilds = map[string]config.Guild{}
	}
	b := &Bot{cfg: cfg, engine: engine, voice: voice.New(), ctx: ctx, logger: logger, queue: make(chan outgoing, 128), tasks: make(chan func(*discordgo.Session), 32)}
	b.id.Store("")
	return b
}

func (b *Bot) AttachStore(store *storage.Store) { b.store = store }
func (b *Bot) guild(id string) (config.Guild, bool) {
	b.cfgMu.RLock()
	defer b.cfgMu.RUnlock()
	g, ok := b.cfg.Guilds[id]
	return g, ok
}
func (b *Bot) setGuild(id string, g config.Guild) {
	b.cfgMu.Lock()
	defer b.cfgMu.Unlock()
	b.cfg.Guilds[id] = g
}
func (b *Bot) task(work func(*discordgo.Session)) bool {
	if b.ctx.Err() != nil {
		return false
	}
	select {
	case b.tasks <- work:
		return true
	default:
		return false
	}
}

func (b *Bot) Register(s *discordgo.Session) {
	// Process Gateway callbacks in sequence so VC moves cannot overtake joins.
	// Network work goes through a bounded queue outside the callbacks.
	s.SyncEvents = true
	s.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates
	s.AddHandler(b.onReady)
	s.AddHandler(b.onGuildCreate)
	s.AddHandler(b.onGuildDelete)
	s.AddHandler(b.onMessage)
	s.AddHandler(b.onVoice)
	s.AddHandler(b.onInteraction)
	s.AddHandler(func(_ *discordgo.Session, _ *discordgo.Disconnect) {
		b.logger.Warn("Discordとの接続が切れました。再接続を待ちます")
	})
	b.workers.Add(1)
	go func() {
		defer b.workers.Done()
		for {
			select {
			case <-b.ctx.Done():
				return
			case message := <-b.queue:
				b.process(s, message)
			}
		}
	}()
	b.workers.Add(1)
	go func() {
		defer b.workers.Done()
		for {
			select {
			case <-b.ctx.Done():
				return
			case work := <-b.tasks:
				work(s)
			}
		}
	}()
}

func (b *Bot) Wait() { b.workers.Wait() }

func (b *Bot) onReady(_ *discordgo.Session, ready *discordgo.Ready) {
	if ready.User == nil {
		return
	}
	b.id.Store(ready.User.ID)
	// A non-resumable reconnect loses timing continuity. Do not retain stale durations.
	b.cfgMu.RLock()
	for guildID := range b.cfg.Guilds {
		b.voice.Seed(guildID, nil)
	}
	b.logger.Info("Discordに接続しました", "bot_id", ready.User.ID, "configured_guilds", len(b.cfg.Guilds))
	b.cfgMu.RUnlock()
	if b.store != nil {
		// READY lists all joined guilds, including temporarily unavailable ones.
		// Archive guilds removed while this process was offline.
		joined := map[string]bool{}
		for _, guild := range ready.Guilds {
			if guild != nil {
				joined[guild.ID] = true
			}
		}
		b.settingsMu.Lock()
		b.cfgMu.RLock()
		var missing []string
		for id := range b.cfg.Guilds {
			if !joined[id] {
				missing = append(missing, id)
			}
		}
		b.cfgMu.RUnlock()
		for _, id := range missing {
			if err := b.store.LeaveGuild(id); err != nil {
				b.logger.Error("退出済みサーバの設定を更新できません", "guild_id", id)
				continue
			}
			b.cfgMu.Lock()
			delete(b.cfg.Guilds, id)
			b.cfgMu.Unlock()
		}
		b.settingsMu.Unlock()
		b.task(func(s *discordgo.Session) { b.registerCommands(s, ready.User.ID) })
	}
}

func (b *Bot) onGuildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild == nil {
		return
	}
	g, ok := b.guild(event.ID)
	created := false
	if b.store != nil {
		b.settingsMu.Lock()
		var err error
		g, created, err = b.store.EnsureGuild(event.ID)
		if err != nil {
			b.settingsMu.Unlock()
			b.logger.Error("サーバ設定を保存できません", "guild_id", event.ID)
			return
		}
		b.setGuild(event.ID, g)
		b.settingsMu.Unlock()
		ok = true
	}
	if !ok {
		return
	}
	occupants := make(map[string]string)
	for _, state := range event.VoiceStates {
		if state != nil {
			occupants[state.UserID] = state.ChannelID
		}
	}
	b.voice.Seed(event.ID, occupants)
	if g.VoiceTextChannelID != "" {
		channel, err := s.State.Channel(g.VoiceTextChannelID)
		if err != nil || channel.GuildID != event.ID {
			b.logger.Error("VC通知先が対象サーバのチャンネルとして確認できません", "guild_id", event.ID)
		}
	}
	b.logger.Info("対象サーバの状態を取得しました", "guild_id", event.ID)
	if created && event.SystemChannelID != "" {
		b.enqueue(outgoing{guildID: event.ID, channelID: event.SystemChannelID, text: "招待ありがとう！サーバー管理権限を持つ人が /setup を実行して、VC通知先を選んでね。", prepared: true})
	}
}

func (b *Bot) onGuildDelete(_ *discordgo.Session, event *discordgo.GuildDelete) {
	if event.Guild != nil {
		b.voice.Seed(event.ID, nil)
		if b.store != nil && !event.Unavailable {
			b.settingsMu.Lock()
			defer b.settingsMu.Unlock()
			if err := b.store.LeaveGuild(event.ID); err != nil {
				b.logger.Error("サーバ退出を保存できません", "guild_id", event.ID)
				return
			}
			b.cfgMu.Lock()
			delete(b.cfg.Guilds, event.ID)
			b.cfgMu.Unlock()
		}
	}
}

func MentionText(content, botID string) (string, bool) {
	if botID == "" {
		return "", false
	}
	content = strings.TrimSpace(content)
	for _, mention := range []string{"<@" + botID + ">", "<@!" + botID + ">"} {
		if strings.HasPrefix(content, mention) {
			return strings.TrimSpace(strings.TrimPrefix(content, mention)), true
		}
	}
	return "", false
}

func (b *Bot) onMessage(_ *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Message == nil || m.Author == nil || m.Author.Bot || m.WebhookID != "" {
		return
	}
	if _, ok := b.guild(m.GuildID); !ok {
		return
	}
	text, mentioned := MentionText(m.Content, b.id.Load().(string))
	if !mentioned {
		return
	}
	b.enqueue(outgoing{guildID: m.GuildID, channelID: m.ChannelID, text: text})
}

func (b *Bot) enqueue(message outgoing) {
	if b.ctx.Err() != nil {
		return
	}
	select {
	case b.queue <- message:
	default:
		b.logger.Warn("処理待ちが上限に達したため通知を省略しました", "guild_id", message.guildID)
	}
}

func (b *Bot) process(s *discordgo.Session, message outgoing) {
	ctx, cancel := context.WithTimeout(b.ctx, 15*time.Second)
	defer cancel()
	g, ok := b.guild(message.guildID)
	if !ok {
		return
	}
	// Do not deliver queued notifications to an old destination after settings change.
	if message.prepared && message.voice && message.channelID != g.VoiceTextChannelID {
		return
	}
	text := message.text
	var err error
	if !message.prepared {
		text, err = b.engine.Reply(ctx, g, message.text, time.Now())
	}
	if err != nil {
		b.logger.Warn("外部情報を取得できませんでした", "guild_id", message.guildID)
		if text == "" {
			text = "今は情報を取得できないお。少し待ってから試してね"
		}
	}
	b.send(ctx, s, message.guildID, message.channelID, text)
}

func (b *Bot) send(ctx context.Context, s *discordgo.Session, guildID, channelID, text string) {
	if ctx.Err() != nil {
		return
	}
	// Fail closed if the configured notification channel belongs to another guild.
	channel, err := s.State.Channel(channelID)
	if err != nil || channel.GuildID != guildID {
		b.logger.Warn("送信先が対象サーバのチャンネルとして確認できません", "guild_id", guildID)
		return
	}
	if strings.TrimSpace(text) == "" {
		text = "まるめし"
	}
	_, err = s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:         LimitContent(text),
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}},
	}, discordgo.WithContext(ctx))
	if err != nil {
		b.logger.Warn("Discordへの送信に失敗しました", "guild_id", guildID)
	}
}

// LimitContent counts UTF-16 units as well as preserving valid UTF-8 for emoji-heavy responses.
func LimitContent(text string) string {
	text = strings.ToValidUTF8(text, "�")
	if len(utf16.Encode([]rune(text))) <= 2000 {
		return text
	}
	var b strings.Builder
	units := 0
	for _, r := range text {
		n := 1
		if r > 0xFFFF {
			n = 2
		}
		if units+n > 1999 {
			break
		}
		b.WriteRune(r)
		units += n
	}
	return b.String() + "…"
}

func (b *Bot) onVoice(s *discordgo.Session, event *discordgo.VoiceStateUpdate) {
	if event.VoiceState == nil {
		return
	}
	g, ok := b.guild(event.GuildID)
	if !ok || g.VoiceTextChannelID == "" || event.UserID == b.id.Load().(string) {
		return
	}
	member := event.Member
	if member == nil && event.BeforeUpdate != nil {
		member = event.BeforeUpdate.Member
	}
	if member == nil {
		member, _ = s.State.Member(event.GuildID, event.UserID)
	}
	if member != nil && member.User != nil && member.User.Bot {
		return
	}
	before := ""
	if event.BeforeUpdate != nil {
		before = event.BeforeUpdate.ChannelID
	}
	change, changed := b.voice.Update(voice.Key{GuildID: event.GuildID, UserID: event.UserID}, event.ChannelID, before, time.Now())
	if !changed {
		return
	}
	from, to := g.WatchesVoice(change.From), g.WatchesVoice(change.To)
	if !from && !to {
		return
	}
	name := "参加者"
	if member != nil && member.User != nil {
		name = member.User.Username
	}
	text := ""
	switch {
	case from && !to:
		if change.To == "" {
			text = name + " が 通話からいなくなったお"
		} else {
			text = name + " が 対象の通話から移動したお"
		}
	case to:
		channelName := "通話"
		if channel, err := s.State.Channel(change.To); err == nil && channel.GuildID == event.GuildID {
			channelName = channel.Name
		}
		if from {
			text = name + " が " + channelName + "に移動したお"
		} else {
			text = name + " が " + channelName + "にジョインしたお"
		}
	}
	if from && change.KnownDuration {
		text += fmt.Sprintf(" 滞在時間:[%s]", change.Duration.String())
	}
	// Voice notifications already contain the final reply; use a separate queue kind.
	b.enqueue(outgoing{guildID: event.GuildID, channelID: g.VoiceTextChannelID, text: text, prepared: true, voice: true})
}
