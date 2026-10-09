package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/aki-lua87/marumeshi_bot/internal/bot"
	"github.com/aki-lua87/marumeshi_bot/internal/config"
	"github.com/aki-lua87/marumeshi_bot/internal/features"
	"github.com/aki-lua87/marumeshi_bot/internal/integrations"
	"github.com/bwmarrin/discordgo"
)

var version = "dev"

type legacyFlags []string

func (l *legacyFlags) String() string         { return "" }
func (l *legacyFlags) Set(value string) error { *l = append(*l, value); return nil }

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if err := run(logger); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	path := flag.String("config", "config.json", "サーバ別設定のJSONファイル")
	check := flag.Bool("check", false, "設定だけを検証（Discord接続なし、トークン不要）")
	checkDiscord := flag.Bool("check-discord", false, "Botの参加先と通知チャンネルをREST APIで検証（投稿なし）")
	diagnose := flag.Bool("diagnose", false, "実行環境・DNS・TLSを確認（Discordログインなし）")
	showVersion := flag.Bool("version", false, "ビルド情報を表示")
	var legacy legacyFlags
	flag.Var(&legacy, "import-legacy", "旧設定を変換: DiscordサーバID=setting.json（複数指定可能、トークンは出力しない）")
	flag.Parse()
	if *showVersion {
		fmt.Printf("panaino-bot %s (%s %s/%s)\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		return nil
	}
	if len(legacy) > 0 {
		cfg := config.Config{Guilds: make(map[string]config.Guild)}
		for _, item := range legacy {
			id, source, ok := strings.Cut(item, "=")
			if !ok {
				return errors.New("import-legacyはDiscordサーバID=旧設定パスで指定してください")
			}
			if _, exists := cfg.Guilds[id]; exists {
				return errors.New("import-legacyのDiscordサーバIDが重複しています")
			}
			g, err := config.ImportLegacy(source)
			if err != nil {
				return err
			}
			cfg.Guilds[id] = g
		}
		if err := cfg.Validate(); err != nil {
			return err
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(cfg)
	}
	if *diagnose {
		return diagnoseRuntime(logger)
	}
	cfg, err := config.Load(*path)
	if err != nil {
		return err
	}
	if *check {
		logger.Info("設定の検証が完了しました", "guilds", len(cfg.Guilds))
		return nil
	}
	token, err := config.TokenFromEnv()
	if err != nil {
		return err
	}
	s, err := discordgo.New(token)
	if err != nil {
		return errors.New("Discordセッションを作成できません")
	}
	s.Client.Timeout = 15 * time.Second
	if *checkDiscord {
		return inspectDiscord(s, cfg, logger)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	b := bot.New(ctx, cfg, features.Engine{API: integrations.New()}, logger)
	b.Register(s)
	if err := s.Open(); err != nil {
		cancel()
		b.Wait()
		return errors.New("Discordに接続できません。トークン・Intents・通信環境を確認してください")
	}
	<-ctx.Done()
	logger.Info("Botを終了します")
	err = s.Close()
	b.Wait()
	if err != nil {
		return errors.New("Discord接続の終了処理に失敗しました")
	}
	return nil
}

func inspectDiscord(s *discordgo.Session, cfg config.Config, logger *slog.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	user, err := s.User("@me", discordgo.WithContext(ctx))
	if err != nil {
		return errors.New("Botの認証確認に失敗しました")
	}
	logger.Info("Botの認証を確認しました", "bot_id", user.ID, "name", user.Username)
	for id, g := range cfg.Guilds {
		guild, err := s.Guild(id, discordgo.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("Botがサーバ%sにアクセスできません。招待を確認してください", id)
		}
		logger.Info("対象サーバへの参加を確認しました", "guild_id", id, "name", guild.Name)
		member, err := s.GuildMember(id, user.ID, discordgo.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("サーバ%s: Bot自身のメンバー情報を取得できません", id)
		}
		member.GuildID = id
		if err := s.State.GuildAdd(guild); err != nil {
			return errors.New("サーバ状態を検証できません")
		}
		if err := s.State.MemberAdd(member); err != nil {
			return errors.New("Botの権限を検証できません")
		}
		for _, channelID := range append([]string{g.VoiceTextChannelID}, g.VoiceChannelIDs...) {
			if channelID == "" {
				continue
			}
			channel, err := s.Channel(channelID, discordgo.WithContext(ctx))
			if err != nil || channel.GuildID != id {
				return fmt.Errorf("サーバ%s: 通知先またはVCが対象サーバにありません", id)
			}
			if channelID == g.VoiceTextChannelID {
				if channel.Type != discordgo.ChannelTypeGuildText && channel.Type != discordgo.ChannelTypeGuildNews {
					return fmt.Errorf("サーバ%s: VC通知先にはテキストチャンネルを指定してください", id)
				}
			} else if channel.Type != discordgo.ChannelTypeGuildVoice && channel.Type != discordgo.ChannelTypeGuildStageVoice {
				return fmt.Errorf("サーバ%s: voice_channel_idsにはVCを指定してください", id)
			}
			if err := s.State.ChannelAdd(channel); err != nil {
				return errors.New("チャンネル状態を検証できません")
			}
			permissions, err := s.State.UserChannelPermissions(user.ID, channelID)
			required := int64(discordgo.PermissionViewChannel)
			if channelID == g.VoiceTextChannelID {
				required |= discordgo.PermissionSendMessages
			}
			if err != nil || permissions&required != required {
				return fmt.Errorf("サーバ%s: Botにチャンネル表示または通知投稿の権限がありません", id)
			}
		}
	}
	logger.Info("Discord設定の検証が完了しました（メッセージは投稿していません）")
	return nil
}

func diagnoseRuntime(logger *slog.Logger) error {
	logger.Info("実行環境", "version", version, "go", runtime.Version(), "os", runtime.GOOS, "arch", runtime.GOARCH)
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return errors.New("日本時間のタイムゾーンを読み込めません")
	}
	logger.Info("日本時間の処理を確認しました", "zone", time.Now().In(jst).Format("-07:00"))
	client := &http.Client{Timeout: 10 * time.Second}
	for _, endpoint := range []string{"https://discord.com/api/v10/gateway", "https://weather.tsukumijima.net/api/forecast?city=130010", "https://api.scryfall.com/cards/random?q=lang%3Aja"} {
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return errors.New("診断用のリクエストを作成できません")
		}
		req.Header.Set("User-Agent", "panaino-bot/2 (+https://github.com/aki-lua87/panaino_bot)")
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return errors.New("DNSまたはHTTPS通信の確認に失敗しました")
		}
		io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTPS通信の確認でHTTP %dが返りました", resp.StatusCode)
		}
		logger.Info("HTTPS通信を確認しました", "host", resp.Request.URL.Hostname(), "status", resp.StatusCode)
	}
	return nil
}
