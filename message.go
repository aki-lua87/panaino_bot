package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// 一問一答形式メッセージ
func messageCheck(message string) string {

	// リファクタリング中
	if strings.HasPrefix(message, appConfig.BotName) {
		switch {
		case strings.Contains(message, "お昼"), strings.Contains(message, "昼飯"), strings.Contains(message, "晩飯"), strings.Contains(message, "ひるめし"):
			return GetHirumeshi()
		}
	}
	switch {
	// プログラム的処理が必要なもの
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName, "セリフ")):
		return appConfig.SpreadsheetURL
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName, "おひる")):
		return GetHirumeshi()
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName, "天気")):
		return GetWether("130010")
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName, "東京の天気")):
		return GetWether("130010")
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName, "福岡の天気")):
		return GetWether("400040")
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName, "大阪の天気")):
		return GetWether("270000")
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName, "明日の緊急")):
		return GetPESO2EmergencyQuestString("明日", time.Now().UTC().Add(time.Hour*9).Add(time.Hour*24))
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName, "今日の緊急")):
		return GetPESO2EmergencyQuestString("今日", time.Now().UTC().Add(time.Hour*9))
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName, "緊急")):
		return GetPESO2EmergencyQuestString("今日", time.Now().UTC().Add(time.Hour*9))
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName, "覇者")):
		text, _ := GetPSO2CoatOfArms()
		return text
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName, "help")):
		return getHelpMessage()
	}
	// マッチしない場合はGoogleスプレッドシート or 基本セットより取得(一問一答形式)
	return getGSSMessage(strings.TrimLeft(message, appConfig.BotName+" "))
}

func getGSSMessage(key string) string {
	log.Println("get api key: ", key)

	type respons struct {
		Result bool   `json:"response"`
		Text   string `json:"text"`
	}

	v := url.Values{}
	v.Set("key", key)
	apiurl := fmt.Sprintf("%s?%s", appConfig.SpreadsheetAPI, v.Encode())
	resp, err := http.Get(apiurl)
	if err != nil {
		return "error"
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	var res respons
	if err := json.Unmarshal(byteArray, &res); err != nil {
		return "不明なエラーです・・・"
	}

	if res.Result {
		return res.Text
	}
	return randMessege()
}

func getHelpMessage() string {
	return `以下のメンションを投げると反応してくれるよ

	天気 : 関東の天気情報を表示します。
	福岡の天気 : ちゃんみらの家付近の天気情報を表示します。
	大阪の天気 : 大阪付近の天気情報を表示します。
	お昼：おひるめしえん
	緊急 : 今日の緊急を表示します。
	覇者 : 今週の王者の紋章キャンペーン対象を表示します。
	明日の緊急 : 明日の緊急を表示します。
	`
}

func randMessege() string {
	var messageList []string
	rand.Seed(time.Now().UnixNano())
	// 基本まるめし構文
	messageList = append(messageList, "まるい", "り", "それ", "そり", "まるめし", "まるくなりたい", "……ｫ'ﾝ", "んまっ！？", "んまー", "マ？", "はやめで", "マァ～")
	// スタンプ
	messageList = append(messageList, ":bread: ", ":moyai: ", ":cactus: ")
	// GOD
	messageList = append(messageList, "俺は神 ", "いや完全にそれになった", "すず", "ぱないの")
	randNum := rand.Intn(len(messageList))
	return messageList[randNum]
}

func GetHirumeshi() string {
	var OhiruList []string
	OhiruList = append(OhiruList, "うどん", "蕎麦", "天ぷら蕎麦", "マックのフライドポテト", "ラーメン", "パスタ", // 麺類
		"カツ丼", "天丼", "カレー", "唐揚げ定食", "寿司", // 飯類
		"麻婆豆腐", "Spaghetti", "ぐらたん", // 中華とか
		"https://cookpad.com/recipe/4295725", "白ごはんと漬物とみそ汁", "砂に醤油かけて食ってろ", ":bread: ",  // 虚無1
		"コンビニめし", "魔剤", "日高屋", "カツ丼食えよｫｫｫｫx！！！！") // 虚無2
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
