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
	stopBot           = make(chan bool)
	vcsession         *discordgo.VoiceConnection
	HelloWorld        = "うらめしえんたんかわいい"
	ChannelVoiceJoin  = "!vcjoin"
	ChannelVoiceLeave = "!vcleave"
)

func main() {
	//Discordのセッションを作成
	discord, err := discordgo.New()
	discord.Token = Token
	if err != nil {
		fmt.Println("Error logging in")
		fmt.Println(err)
	}

	discord.AddHandler(onMessageCreate) //全てのWSAPIイベントが発生した時のイベントハンドラを追加
	// websocketを開いてlistening開始
	err = discord.Open()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Listening...")
	<-stopBot //プログラムが終了しないようロック
	return
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	c, err := s.State.Channel(m.ChannelID) //チャンネル取得
	if err != nil {
		log.Println("Error getting channel: ", err)
		return
	}
	fmt.Printf("%20s %20s %20s > %s\n", m.ChannelID, time.Now().Format(time.Stamp), m.Author.Username, m.Content)

	switch {
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, HelloWorld)): //Bot宛に!helloworld コマンドが実行された時
		sendMessage(s, c, "うらめしえんたんかわいい")
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "天気")): //Bot宛に!helloworld コマンドが実行された時
		sendMessage(s, c, getWether())
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "おやすみ")): //Bot宛に!helloworld コマンドが実行された時
		sendMessage(s, c, "おやす§")
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "レイスト")): //Bot宛に!helloworld コマンドが実行された時
		sendMessage(s, c, "レイストちゃんかわいい")
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s", BotName)): //Bot宛に!helloworld コマンドが実行された時
		sendMessage(s, c, "俺は神")
	}
}

//メッセージを受信した時の、声の初めと終わりにPrintされるようだ
func onVoiceReceived(vc *discordgo.VoiceConnection, vs *discordgo.VoiceSpeakingUpdate) {
	log.Print("しゃべったあああああ")
}

//メッセージを送信する関数
func sendMessage(s *discordgo.Session, c *discordgo.Channel, msg string) {
	_, err := s.ChannelMessageSend(c.ID, msg)

	log.Println(">>> " + msg)
	if err != nil {
		log.Println("Error sending message: ", err)
	}
}

func getWether() string {
	var text string
	url := "http://weather.livedoor.com/forecast/webservice/json/v1?city=130010"
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
