package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	stopBot   = make(chan bool)
	vcsession *discordgo.VoiceConnection

	appConfig AppConfig

	ChannelVoiceJoin  = "通話"
	ChannelVoiceLeave = "もういいよ"
)

// AppConfig 設定秘匿情報
type AppConfig struct {
	DiscordToken   string `json:"DiscordToken"`   // DiscordBotトークン(要Bot Prefix)
	BotName        string `json:"BotName"`        // ボット名<@111111122222222333333>
	SpreadsheetURL string `json:"SpreadsheetURL"` // スプレッドシートURL
	SpreadsheetAPI string `json:"SpreadsheetAPI"` // 会話API
	CoatOfArmsURL  string `json:"CoatOfArmsURL"`  // 紋章キャンペーン取得API
	VcID           string `json:"VC_ID"`          // 参加ボイスチャンネルURL
}

func init() {
	settingInit()
}

func settingInit() error {
	// AppConfig用Settingファイル読み込み
	appConfigJSON, err := ioutil.ReadFile("./setting.json")
	if err != nil {
		fmt.Println(err)
		return err
	}
	json.Unmarshal(appConfigJSON, &appConfig)
	fmt.Println(string(appConfigJSON))

	return nil
}

func main() {
	discord, err := discordgo.New()
	discord.Token = appConfig.DiscordToken
	if err != nil {
		log.Println("Error logging in")
		log.Println(err)
	}

	discord.AddHandler(onMessageCreate)
	// websocket
	err = discord.Open()
	if err != nil {
		fmt.Println(err)
	}

	log.Println("Listening...")
	<-stopBot
	return
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	c, err := s.State.Channel(m.ChannelID)
	if err != nil {
		log.Println("Error getting channel: ", err)
		return
	}
	log.Println("%20s %20s %20s > %s\n", m.ChannelID, time.Now().Format(time.Stamp), m.Author.Username, m.Content)

	nowTime := time.Now().UTC().Add(time.Hour * 9)
	log.Println("JST now Time > ", nowTime)
	// 正月限定おみくじタイム
	if nowTime.Month() == 1 {
		if nowTime.Day() <= 3 {
			switch {
			case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", appConfig.BotName, "おみくじ")):
				sendMessage(s, c, Omikuji())
				return
			case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", appConfig.BotName, "あけおめ")):
				sendMessage(s, c, "あけおめし")
				return
			case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", appConfig.BotName, "あけましておめでとう")):
				sendMessage(s, c, "あけおめし")
				return
			}
		}
	}

	// まるめしとおはなししたい
	switch {
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", appConfig.BotName, ChannelVoiceJoin)):
		vcsession, err = s.ChannelVoiceJoin(c.GuildID, appConfig.VcID, false, false)
		if err != nil {
			log.Println("Error Join voice channel: ", err)
			sendMessage(s, c, err.Error())
			return
		}
		// vcsession.AddHandler(onVoiceReceived) //音声受信時のイベントハンドラ
		sendMessage(s, c, "まるめし！")
		return
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", appConfig.BotName, ChannelVoiceLeave)):
		vcsession.Disconnect()
		sendMessage(s, c, "んな～")
		return
	}

	// その他一問一答形式メッセージ
	if strings.HasPrefix(m.Content, fmt.Sprintf("%s", appConfig.BotName)) {
		sendMessage(s, c, messageCheck(m.Content))
	}
}

func onVoiceReceived(vc *discordgo.VoiceConnection, vs *discordgo.VoiceSpeakingUpdate) {

}

//メッセージを送信
func sendMessage(s *discordgo.Session, c *discordgo.Channel, msg string) {
	_, err := s.ChannelMessageSend(c.ID, msg)

	log.Println(">>> " + msg)
	if err != nil {
		log.Println("Error sending message: ", err)
	}
}
