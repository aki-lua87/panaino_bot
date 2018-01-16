package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	// Token   = "Bot 別ファイルに記述"
	// BotName = "<@別ファイルに記述>"
	stopBot   = make(chan bool)
	vcsession *discordgo.VoiceConnection
)

func main() {
	discord, err := discordgo.New()
	discord.Token = Token
	if err != nil {
		fmt.Println("Error logging in")
		fmt.Println(err)
	}

	discord.AddHandler(onMessageCreate)
	// websocket
	err = discord.Open()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Listening...")
	<-stopBot
	return
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	c, err := s.State.Channel(m.ChannelID)
	if err != nil {
		log.Println("Error getting channel: ", err)
		return
	}
	fmt.Printf("%20s %20s %20s > %s\n", m.ChannelID, time.Now().Format(time.Stamp), m.Author.Username, m.Content)

	nowTime := time.Now()
	if nowTime.Month() == 1 {
		if nowTime.Day() <= 3 {
			switch {
			case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "おみくじ")):
				sendMessage(s, c, Omikuji())
				return
			case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "あけおめ")):
				sendMessage(s, c, "あけおめし")
				return
			case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "あけましておめでとう")):
				sendMessage(s, c, "あけおめし")
				return
			}
		}
	}

	switch {
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "うらめしえんたんかわいい")):
		sendMessage(s, c, "うらめしえんたんかわいい")
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "お昼")):
		sendMessage(s, c, GetHirumeshi())
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "おひる")):
		sendMessage(s, c, GetHirumeshi())
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "天気")):
		sendMessage(s, c, GetWether("130010"))
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "東京の天気")):
		sendMessage(s, c, GetWether("130010"))
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "福岡の天気")):
		sendMessage(s, c, GetWether("400040")) // 410020
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "大阪の天気")):
		sendMessage(s, c, GetWether("270000"))
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "明日の緊急")):
		sendMessage(s, c, PSO2("明日", time.Now().Add(time.Hour*24)))
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "緊急")):
		sendMessage(s, c, PSO2("今日", time.Now()))
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "おやすみ")):
		sendMessage(s, c, "おやす§")
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "レイスト")):
		sendMessage(s, c, "レイストちゃんかわいい")
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "ちゃんみら")):
		sendMessage(s, c, "みらめしえんたんとうとい ")
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "致した")):
		sendMessage(s, c, "いためしえんたんしこいい")
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "さかい")):
		sendMessage(s, c, "はげめしえんたんさむいい ")
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "おこ？")):
		sendMessage(s, c, "おこめしえんたんこわいい ")
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "かわいいね")):
		sendMessage(s, c, "てれめしえんたんちょろいい ")
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "らんらんひとし")):
		sendMessage(s, c, "ふぁいあー！！")
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "help")):
		sendMessage(s, c, help())
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s", BotName)):
		sendMessage(s, c, randMessege())
	}
}

func onVoiceReceived(vc *discordgo.VoiceConnection, vs *discordgo.VoiceSpeakingUpdate) {

}

func help() string {
	return `以下のメンションを投げると反応してくれるよ

	天気 : 関東の天気情報を表示します。
	福岡の天気 : ちゃんみらの家付近の天気情報を表示します。
	大阪の天気 : 大阪付近の天気情報を表示します。
	お昼：おひるめしえん
	緊急 : 今日の緊急を表示します。
	明日の緊急 : 明日の緊急を表示します。
	`
}

func GetHirumeshi() string {
	var OhiruList []string
	OhiruList = append(OhiruList, "まるかめし", "カレー", "パスタ",
		"うどん", "松屋", "魔剤", "丸亀", "まるめし", "コンビニめし", "ぐらたん",
		"ジンギスカン", "ラーメン", "ラーメン", "ラーメン", "カツ丼食えよｫｫｫｫx！！！！")
	randNum := rand.Intn(len(OhiruList))
	return OhiruList[randNum]
}

func Omikuji() string {
	var OmikujiList []string
	rand.Seed(time.Now().UnixNano())
	// 基本まるめし構文
	OmikujiList = append(OmikujiList, "大吉", "中吉", "吉", "小吉", "凶", "大凶", "まるめし吉", "はずれ")
	randNum := rand.Intn(len(OmikujiList))
	return OmikujiList[randNum]
}

func randMessege() string {
	var messageList []string
	rand.Seed(time.Now().UnixNano())
	// 基本まるめし構文
	messageList = append(messageList, "まるい", "り", "それ", "そり", "まるめし", "まるくなりたい", "……。", "ぬくい", "んまー", "マ？", "はやめで", "マァ～")
	// スタンプ
	messageList = append(messageList, ":bread: ", ":moyai: ", ":cactus: ")
	// GOD
	messageList = append(messageList, "俺は神 ", "まるかめし ", "さむい ")
	randNum := rand.Intn(len(messageList))
	return messageList[randNum]
}

//メッセージを送信
func sendMessage(s *discordgo.Session, c *discordgo.Channel, msg string) {
	_, err := s.ChannelMessageSend(c.ID, msg)

	log.Println(">>> " + msg)
	if err != nil {
		log.Println("Error sending message: ", err)
	}
}

func PSO2(date string, t time.Time) string {
	var postText string
	postText = fmt.Sprintln(date + "の緊急クエストは....")

	getKey := fmt.Sprintf("%d%02d%02d", t.Year(), int(t.Month()), t.Day())
	log.Println("Key => ", getKey)
	l := CnvEmaList(GetEmaList(getKey))

	for _, v := range l {
		postText = postText + v + " \n"
	}

	return postText
}
func CnvEmaList(emaList []EmaList) []string {
	var emaListStr []string
	if len(emaList) == 0 {
		emaListStr = append(emaListStr, "予定なしです。")
		return emaListStr
	}
	for _, v := range emaList {
		ema := fmt.Sprintf("%02d:%02d %s", v.Hour, v.Minute, v.EventName)
		emaListStr = append(emaListStr, ema)
	}
	return emaListStr
}

func GetEmaList(getKey string) []EmaList {
	client := &http.Client{}

	apiurl := "https://akakitune87.net/api/v4/pso2emergency"

	jsonStr := `{"EventDate":"` + getKey + `"}`
	req, err := http.NewRequest(
		"POST",
		apiurl,
		bytes.NewBuffer([]byte(jsonStr)),
	)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Not Emag Get:", err)
	}
	defer resp.Body.Close()

	byteArray, err := ioutil.ReadAll(resp.Body)

	var emaList []EmaList
	if err := json.Unmarshal(byteArray, &emaList); err != nil {
		log.Println("Not Emag List", err)
	}

	return emaList
}

type EmaList struct {
	EventName string `json:"EventName"`
	EvantType string `json:"EventType"`
	Month     int    `json:"Month"`
	Date      int    `json:"Date"`
	Hour      int    `json:"Hour"`
	Minute    int    `json:"Minute"`
}
