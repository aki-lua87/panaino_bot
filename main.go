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

	switch {
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "うらめしえんたんかわいい")):
		sendMessage(s, c, "うらめしえんたんかわいい")
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "天気")):
		sendMessage(s, c, getWether("130010"))
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "福岡天気")):
		sendMessage(s, c, getWether("410020"))
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "geotest")):
		sendMessage(s, c, GeoTest())
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
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "明日の緊急")):
		sendMessage(s, c, PSO2(time.Now().Add(time.Hour*9)))
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "緊急")):
		sendMessage(s, c, PSO2(time.Now()))
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", BotName, "help")):
		sendMessage(s, c, help())
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s", BotName)):
		sendMessage(s, c, "まるい")
	}
}

func onVoiceReceived(vc *discordgo.VoiceConnection, vs *discordgo.VoiceSpeakingUpdate) {

}

func help() string {
	return `@うらめしえんたんかわいいBotで以下のメンションを投げると反応してくれるよ

	天気 : 関東の天気情報を表示します。
	福岡天気 : ちゃんみら付近の天気情報を表示します。
	緊急 : 今日の緊急を表示します。
	明日の緊急 : 明日の緊急を表示します。
	`
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

func GeoTest() string {
	rand.Seed(time.Now().UnixNano())
	lat := rand.Intn(15) + 130
	ulat := rand.Intn(99999)
	lon := rand.Intn(8) + 30
	ulon := rand.Intn(99999)
	url := "https://maps.googleapis.com/maps/api/geocode/json?"
	latlon := fmt.Sprintf("latlng=%d.%07d,%d.%07d&key=%s", lat, ulat, lon, ulon, GeoAPI)
	return url + latlon
}

func PSO2(t time.Time) string {
	var postText string
	postText = fmt.Sprintln("今日の緊急クエストは....")

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
	if emaList == nil {
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

	apiurl := "https://akakitune87.net/api/v2/pso2ema"

	req, err := http.NewRequest(
		"POST",
		apiurl,
		bytes.NewBuffer([]byte(`"`+getKey+`"`)),
	)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Not Emag Get", err)
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
	EventName string `json:"evant"`
	Month     int    `json:"month"`
	Date      int    `json:"date"`
	Hour      int    `json:"hour"`
	Minute    int    `json:"minute"`
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
