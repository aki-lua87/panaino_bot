package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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

	switch {
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "うらめしえんたんかわいい")):
		sendMessage(s, c, "うらめしえんたんかわいい")
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "天気")):
		sendMessage(s, c, getWether("130010"))
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "ちゃんみら天気")):
		sendMessage(s, c, getWether("410020"))
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
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s", BotName)):
		sendMessage(s, c, "俺は神")
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

// 天気情報取得
func getWether(id string) string {
	var text string
	url := "http://weather.livedoor.com/forecast/webservice/json/v1?city=" + id
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil {
		log.Println("Error1: ", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	var weatherData WeatherData
	err = json.Unmarshal(body, &weatherData)
	if err != nil {
		log.Println("Error2: ", err)
	}

	log.Println(weatherData.Description.Text)

	text = weatherData.Description.Text

	return text
}

type WeatherData struct {
	Location         Location
	Title            string
	Link             string
	PublicTime       string
	Description      Description
	Forecasts        []Forecasts
	PinpointLocation []PinpointLocation
	Copyright        Copyright
}

type Location struct {
	Area string
	Pref string
	City string
}

type Description struct {
	Text       string
	PublicTime string
}

type Forecasts struct {
	Date        string
	DateLabel   string
	Telop       string
	Image       Image
	Temperature Temperature
}

type Image struct {
	Title  string
	Link   string
	Url    string
	Width  int
	Height int
}

type Temperature struct {
	Celsius    MaxMin
	Fahrenheit MaxMin
}

type MaxMin struct {
	Max string
	Min string
}

type PinpointLocation struct {
	Name string
	Link string
}

type Copyright struct {
	Title    string
	Link     string
	Image    Image
	Provider string
}
