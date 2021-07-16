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

type UserState struct {
	Name      string
	CurrentVC string
	Jointime  time.Time
}

var (
	stopBot   = make(chan bool)
	vcsession *discordgo.VoiceConnection

	appConfig AppConfig

	discord *discordgo.Session
	usermap = map[string]*UserState{}
)

// AppConfig 設定秘匿情報
type AppConfig struct {
	DiscordToken   string `json:"DiscordToken"`   // DiscordBotトークン(要Bot Prefix)
	BotName        string `json:"BotName"`        // ボット名<@111111122222222333333>
	BotName2       string `json:"BotName2"`       // 信じられない仕様変更
	SpreadsheetURL string `json:"SpreadsheetURL"` // スプレッドシートURL
	SpreadsheetAPI string `json:"SpreadsheetAPI"` // 会話API
	CoatOfArmsURL  string `json:"CoatOfArmsURL"`  // 紋章キャンペーン取得API
	VcID           string `json:"VC_ID"`          // 参加ボイスチャンネルURL
	TextCannelID   string `json:"TC_ID"`          // 参加テキストチャンネルURL
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
	var err error
	discord, err = discordgo.New()
	discord.Token = appConfig.DiscordToken
	if err != nil {
		log.Println("Error logging in")
		log.Println(err)
	}

	discord.AddHandler(onMessageCreate)
	discord.AddHandler(onVoiceStateUpdate)
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

	nowTime := time.Now().UTC().Add(time.Hour * 9)
	log.Println(nowTime)
	fmt.Printf("onMessageCreate :%+v\n", m)
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

	// その他一問一答形式メッセージ
	if strings.HasPrefix(m.Content, fmt.Sprintf("%s", appConfig.BotName)) {
		sendMessage(s, c, messageCheck(m.Content))
		return
	}
	if strings.HasPrefix(m.Content, fmt.Sprintf("%s", appConfig.BotName2)) {
		sendMessage(s, c, messageCheck(m.Content))
		return
	}
}

func onVoiceReceived(vc *discordgo.VoiceConnection, vs *discordgo.VoiceSpeakingUpdate) {

}

func onVoiceStateUpdate(s *discordgo.Session, vs *discordgo.VoiceStateUpdate) {

	// defer func() {
	// 	s.ChannelMessageSend(appConfig.TextCannelID, "寝るわ！")
	// }()

	nowTime := time.Now().UTC().Add(time.Hour * 9)
	log.Println(nowTime)
	fmt.Printf("onVoiceStateUpdate :%+v\n", vs)

	_, ok := usermap[vs.UserID]
	if !ok {
		usermap[vs.UserID] = new(UserState)
		user, err := discord.User(vs.UserID)
		if err != nil {
			log.Println(err)
			return
		}
		usermap[vs.UserID].Name = user.Username
		log.Print("new user added : " + user.Username)
	}

	if vs.ChannelID == "" {
		usermap[vs.UserID].CurrentVC = ""
		time := nowTime.Sub(usermap[vs.UserID].Jointime)
		slice := strings.Split(time.String(), ".")
		timeString := slice[0]
		message := usermap[vs.UserID].Name + " が 通話からいなくなったお 滞在時間:[" + timeString + "s]"
		_, err := s.ChannelMessageSend(appConfig.TextCannelID, message)
		if err != nil {
			log.Println(err)
		}
		return
	}

	if usermap[vs.UserID].CurrentVC != vs.ChannelID {
		usermap[vs.UserID].CurrentVC = vs.ChannelID
		usermap[vs.UserID].Jointime = nowTime
		channel, _ := discord.Channel(vs.ChannelID)
		message := usermap[vs.UserID].Name + " が " + channel.Name + "にジョインしたお"
		log.Print(message)
		_, err := s.ChannelMessageSend(appConfig.TextCannelID, message)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

//メッセージを送信
func sendMessage(s *discordgo.Session, c *discordgo.Channel, msg string) {
	_, err := s.ChannelMessageSend(c.ID, msg)

	log.Println(">>> " + msg)
	if err != nil {
		log.Println("Error sending message: ", err)
	}
}
