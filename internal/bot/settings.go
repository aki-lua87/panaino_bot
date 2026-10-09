package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func settingsCommands() []*discordgo.ApplicationCommand {
	permission := int64(discordgo.PermissionManageServer)
	contexts := []discordgo.InteractionContextType{discordgo.InteractionContextGuild}
	commands := []*discordgo.ApplicationCommand{
		{Name: "setup", Description: "VC通知先を選択・変更します"},
		{Name: "settings", Description: "このサーバのBot設定を確認します"},
		{Name: "voice", Description: "VC通知を管理します", Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "disable", Description: "VC通知を停止します"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "all", Description: "全VCを通知対象にします"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "channels", Description: "通知対象のVCを選択します"},
		}},
	}
	for _, command := range commands {
		command.DefaultMemberPermissions = &permission
		command.Contexts = &contexts
	}
	return commands
}

func (b *Bot) registerCommands(s *discordgo.Session, appID string) {
	ctx, cancel := context.WithTimeout(b.ctx, 30*time.Second)
	defer cancel()
	// Upsert our own commands without deleting other application commands.
	for _, command := range settingsCommands() {
		if _, err := s.ApplicationCommandCreate(appID, "", command, discordgo.WithContext(ctx)); err != nil {
			b.logger.Error("設定コマンドを登録できません。applications.commandsの認可を確認してください")
			return
		}
	}
	b.logger.Info("設定コマンドを登録しました")
}

func canManage(i *discordgo.Interaction) bool {
	return i != nil && i.GuildID != "" && i.Member != nil && i.Member.User != nil &&
		i.Member.Permissions&(discordgo.PermissionManageServer|discordgo.PermissionAdministrator) != 0
}

func interactionAction(i *discordgo.Interaction) (string, bool) {
	switch data := i.Data.(type) {
	case discordgo.ApplicationCommandInteractionData:
		switch data.Name {
		case "setup", "settings":
			return data.Name, true
		case "voice":
			if len(data.Options) == 1 && data.Options[0] != nil {
				switch data.Options[0].Name {
				case "disable", "all", "channels":
					return "voice:" + data.Options[0].Name, true
				}
			}
		}
	case discordgo.MessageComponentInteractionData:
		if i.Member == nil || i.Member.User == nil {
			return "", false
		}
		if data.CustomID == "setup:"+i.Member.User.ID {
			return "select:setup", true
		}
		if data.CustomID == "voice:"+i.Member.User.ID {
			return "select:voice", true
		}
	}
	return "", false
}

func (b *Bot) onInteraction(s *discordgo.Session, event *discordgo.InteractionCreate) {
	if event == nil || event.Interaction == nil {
		return
	}
	action, ok := interactionAction(event.Interaction)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(b.ctx, 2*time.Second)
	defer cancel()
	if !canManage(event.Interaction) {
		_ = s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "サーバー管理権限を持つ人が、サーバ内で実行してください。", Flags: discordgo.MessageFlagsEphemeral},
		}, discordgo.WithContext(ctx))
		return
	}
	if err := s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}, discordgo.WithContext(ctx)); err != nil {
		b.logger.Warn("設定操作を受け付けられませんでした")
		return
	}
	received := time.Now()
	if !b.task(func(s *discordgo.Session) {
		if time.Since(received) > 2*time.Minute {
			b.editSettings(s, event.Interaction, "受付から時間が経ったため、設定操作をやり直してください。", nil)
			return
		}
		b.configure(s, event.Interaction, action)
	}) {
		b.editSettings(s, event.Interaction, "処理待ちが多いため、少し待ってからやり直してください。", nil)
	}
}

func channelPicker(userID string, voice bool) []discordgo.MessageComponent {
	min := 1
	customID := "setup:" + userID
	placeholder := "VC通知先のテキストチャンネルを選んでください"
	types := []discordgo.ChannelType{discordgo.ChannelTypeGuildText, discordgo.ChannelTypeGuildNews}
	max := 1
	if voice {
		customID = "voice:" + userID
		placeholder = "通知対象のVCを選んでください（最大25件）"
		types = []discordgo.ChannelType{discordgo.ChannelTypeGuildVoice, discordgo.ChannelTypeGuildStageVoice}
		max = 25
	}
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.SelectMenu{MenuType: discordgo.ChannelSelectMenu, CustomID: customID, Placeholder: placeholder, MinValues: &min, MaxValues: max, ChannelTypes: types},
	}}}
}

func (b *Bot) editSettings(s *discordgo.Session, i *discordgo.Interaction, text string, components []discordgo.MessageComponent) {
	text = LimitContent(text)
	ctx, cancel := context.WithTimeout(b.ctx, 5*time.Second)
	defer cancel()
	if components == nil {
		components = []discordgo.MessageComponent{}
	}
	_, err := s.InteractionResponseEdit(i, &discordgo.WebhookEdit{Content: &text, Components: &components,
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}}, discordgo.WithContext(ctx))
	if err != nil {
		b.logger.Warn("設定操作の結果を表示できませんでした")
	}
}

func (b *Bot) configure(s *discordgo.Session, i *discordgo.Interaction, action string) {
	reply := func(text string) { b.editSettings(s, i, text, nil) }
	// Recheck authorization at execution time; no DM or cross-guild settings access.
	if !canManage(i) || b.store == nil {
		reply("この操作は利用できません。")
		return
	}
	g, ok := b.guild(i.GuildID)
	if !ok {
		reply("サーバの初期化が終わるまで待ってから、やり直してください。")
		return
	}
	if err := b.validateManager(s, i); err != nil {
		reply("現在のサーバー管理権限を確認できません。権限を確認してからやり直してください。")
		return
	}
	var err error
	switch action {
	case "setup":
		b.editSettings(s, i, "VC入退室の通知先を選んでください。保存後すぐに反映されます。", channelPicker(i.Member.User.ID, false))
		return
	case "settings":
		destination := "停止中・未設定"
		targets := "全VC"
		if g.VoiceTextChannelID != "" {
			destination = "<#" + g.VoiceTextChannelID + ">"
		}
		if len(g.VoiceChannelIDs) > 0 {
			targets = ""
			for _, id := range g.VoiceChannelIDs {
				targets += "<#" + id + "> "
			}
		}
		reply(fmt.Sprintf("VC通知先: %s\n通知対象: %s\n変更は /setup、対象VCの指定は /voice channels、停止は /voice disable を使ってね。", destination, targets))
		return
	case "voice:channels":
		if g.VoiceTextChannelID == "" {
			reply("先に /setup で通知先を設定してください。")
			return
		}
		b.editSettings(s, i, "通知するVCを選んでください。全VCへ戻す場合は /voice all を使ってね。", channelPicker(i.Member.User.ID, true))
		return
	case "voice:disable":
		g.VoiceTextChannelID = ""
	case "voice:all":
		if g.VoiceTextChannelID == "" {
			reply("先に /setup で通知先を設定してください。")
			return
		}
		g.VoiceChannelIDs = nil
	case "select:setup", "select:voice":
		data, ok := i.Data.(discordgo.MessageComponentInteractionData)
		if !ok || data.ComponentType != discordgo.ChannelSelectMenuComponent || len(data.Values) == 0 || len(data.Values) > 25 || action == "select:setup" && len(data.Values) != 1 {
			reply("チャンネルを選び直してください。")
			return
		}
		if action == "select:voice" && g.VoiceTextChannelID == "" {
			reply("先に /setup で通知先を設定してください。")
			return
		}
		if err = b.validateChannels(s, i.GuildID, data.Values, action == "select:voice"); err != nil {
			reply("対象サーバ内のチャンネルを選んで、Botの表示・投稿権限を確認してください。")
			return
		}
		if action == "select:setup" {
			g.VoiceTextChannelID = data.Values[0]
		} else {
			g.VoiceChannelIDs = data.Values
		}
	default:
		reply("操作をやり直してください。")
		return
	}
	// An old picker cannot write after the Bot has been removed from the guild.
	b.settingsMu.Lock()
	if _, ok := b.guild(i.GuildID); !ok {
		b.settingsMu.Unlock()
		reply("このサーバでは設定を変更できません。")
		return
	}
	g, err = b.store.SetVoice(i.GuildID, g.VoiceTextChannelID, g.VoiceChannelIDs)
	if err != nil {
		b.settingsMu.Unlock()
		reply("設定を保存できませんでした。変更は反映していません。")
		return
	}
	b.setGuild(i.GuildID, g)
	b.voice.Seed(i.GuildID, nil)
	b.settingsMu.Unlock()
	if g.VoiceTextChannelID == "" {
		reply("VC通知を停止しました。")
	} else {
		reply("設定を保存しました。VC通知先: <#" + g.VoiceTextChannelID + ">。再起動は不要です。")
	}
}

func (b *Bot) validateManager(s *discordgo.Session, i *discordgo.Interaction) error {
	ctx, cancel := context.WithTimeout(b.ctx, 10*time.Second)
	defer cancel()
	guild, err := s.Guild(i.GuildID, discordgo.WithContext(ctx))
	if err != nil {
		return err
	}
	member, err := s.GuildMember(i.GuildID, i.Member.User.ID, discordgo.WithContext(ctx))
	if err != nil {
		return err
	}
	if member.User == nil || member.User.ID != i.Member.User.ID {
		return fmt.Errorf("member mismatch")
	}
	if guild.OwnerID == member.User.ID {
		return nil
	}
	roles := map[string]bool{i.GuildID: true}
	for _, id := range member.Roles {
		roles[id] = true
	}
	var permissions int64
	for _, role := range guild.Roles {
		if role != nil && roles[role.ID] {
			permissions |= role.Permissions
		}
	}
	if permissions&(discordgo.PermissionManageServer|discordgo.PermissionAdministrator) == 0 {
		return fmt.Errorf("missing manage guild permission")
	}
	return nil
}

func (b *Bot) validateChannels(s *discordgo.Session, guildID string, ids []string, voice bool) error {
	ctx, cancel := context.WithTimeout(b.ctx, 15*time.Second)
	defer cancel()
	botID := b.id.Load().(string)
	guild, err := s.Guild(guildID, discordgo.WithContext(ctx))
	if err != nil {
		return err
	}
	member, err := s.GuildMember(guildID, botID, discordgo.WithContext(ctx))
	if err != nil {
		return err
	}
	member.GuildID = guildID
	if err = s.State.GuildAdd(guild); err != nil {
		return err
	}
	if err = s.State.MemberAdd(member); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] || strings.TrimSpace(id) == "" {
			return fmt.Errorf("invalid channel selection")
		}
		seen[id] = true
		ch, err := s.Channel(id, discordgo.WithContext(ctx))
		if err != nil {
			return err
		}
		if ch.GuildID != guildID {
			return fmt.Errorf("channel belongs to another guild")
		}
		valid := ch.Type == discordgo.ChannelTypeGuildText || ch.Type == discordgo.ChannelTypeGuildNews
		if voice {
			valid = ch.Type == discordgo.ChannelTypeGuildVoice || ch.Type == discordgo.ChannelTypeGuildStageVoice
		}
		if !valid {
			return fmt.Errorf("invalid channel type")
		}
		if err = s.State.ChannelAdd(ch); err != nil {
			return err
		}
		permissions, err := s.State.UserChannelPermissions(botID, id)
		required := int64(discordgo.PermissionViewChannel)
		if !voice {
			required |= discordgo.PermissionSendMessages
		}
		if err != nil || permissions&required != required {
			return fmt.Errorf("missing channel permissions")
		}
	}
	return nil
}
