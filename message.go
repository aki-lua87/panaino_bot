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

	switch {
	// プログラム的処理が必要なもの
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName, "mtg")):
		cardName := strings.TrimLeft(message, appConfig.BotName)
		cardName = strings.TrimLeft(cardName, " mtg ")
		return FetchHareruyaCards(cardName)
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
	}

	// クソクソクソクソクソクソクソクソクソクソクソクソ
	switch {
	// プログラム的処理が必要なもの
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName2, "mtg")):
		cardName := strings.TrimLeft(message, appConfig.BotName2)
		cardName = strings.TrimLeft(cardName, " mtg ")
		return FetchHareruyaCards(cardName)
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName2, "セリフ")):
		return appConfig.SpreadsheetURL
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName2, "おひる")):
		return GetHirumeshi()
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName2, "天気")):
		return GetWether("130010")
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName2, "東京の天気")):
		return GetWether("130010")
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName2, "福岡の天気")):
		return GetWether("400040")
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName2, "大阪の天気")):
		return GetWether("270000")
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName2, "明日の緊急")):
		return GetPESO2EmergencyQuestString("明日", time.Now().UTC().Add(time.Hour*9).Add(time.Hour*24))
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName2, "今日の緊急")):
		return GetPESO2EmergencyQuestString("今日", time.Now().UTC().Add(time.Hour*9))
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName2, "緊急")):
		return GetPESO2EmergencyQuestString("今日", time.Now().UTC().Add(time.Hour*9))
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", appConfig.BotName2, "覇者")):
		text, _ := GetPSO2CoatOfArms()
		return text
	}

	// リファクタリング中
	if strings.HasPrefix(message, appConfig.BotName) {
		switch {
		case strings.Contains(message, "お昼"), strings.Contains(message, "昼飯"), strings.Contains(message, "晩飯"), strings.Contains(message, "ばんめし"), strings.Contains(message, "ひるめし"), strings.Contains(message, "おひる"), strings.Contains(message, "夕飯"):
			return GetHirumeshi()
		case strings.Contains(message, "酒"):
			return GetSake()
		case strings.Contains(message, "の実況"):
			return getTodayJikkyou()
		}
		return getGSSMessage(strings.TrimLeft(message, appConfig.BotName+" "))
	}

	if strings.HasPrefix(message, appConfig.BotName2) {
		switch {
		case strings.Contains(message, "お昼"), strings.Contains(message, "昼飯"), strings.Contains(message, "晩飯"), strings.Contains(message, "ばんめし"), strings.Contains(message, "ひるめし"), strings.Contains(message, "おひる"), strings.Contains(message, "夕飯"):
			return GetHirumeshi()
		case strings.Contains(message, "酒"):
			return GetSake()
		case strings.Contains(message, "の実況"):
			return getTodayJikkyou()
		}
		return getGSSMessage(strings.TrimLeft(message, appConfig.BotName2+" "))
	}
	return getGSSMessage("")
	// マッチしない場合はGoogleスプレッドシート or 基本セットより取得(一問一答形式)

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

func randMessege() string {
	var messageList []string
	rand.Seed(time.Now().UnixNano())
	// 基本まるめし構文
	messageList = append(messageList, "まるい", "り", "それ", "そり", "まるめし", "まるくなりたい", "……ｫ'ﾝ", "んまっ！？", "んまー", "マ？", "はやめで", "マァ～")
	// スタンプ
	messageList = append(messageList, ":bread: ", ":moyai: ", ":cactus: ")
	// GOD
	messageList = append(messageList, "それになった", "すず", "ぱないの", "ﾎﾟｸｼﾎﾟｸｼ", "にょわ～")
	randNum := rand.Intn(len(messageList))
	return messageList[randNum]
}

func GetHirumeshi() string {
	var OhiruList []string
	OhiruList = append(
		OhiruList,
		"うどん",
		"蕎麦",
		"きつねうどん :fox: ",
		"天ぷら蕎麦",
		"マックのフライドポテト",
		"ラーメン",
		"スパゲッティ",
		"パスタ",
		"つけ麺",
		"油そば",
		"カツ丼",
		"天丼",
		"カレー",
		"ぎゅうどん！",
		"唐揚げ定食",
		"寿司",
		"野菜炒め",
		"クロワッサン :croissant: ",
		"麻婆豆腐",
		"焼きそば",
		"ぐらたん",
		"ピッツァ :pizza: ",
		"ハンバーグ",
		"オムライス",
		"ケバブ :taco: ",
		"白ごはんと漬物とみそ汁",
		"オム・ライス",
		"日替わり定食 ",
		"コンビニめし",
		"カツ丼食えよｫｫｫｫx！！！！",
		"お好み焼き")
	randNum := rand.Intn(len(OhiruList))
	return OhiruList[randNum]
}

func Omikuji() string {
	var OmikujiList []string
	rand.Seed(time.Now().UnixNano())
	OmikujiList = append(OmikujiList, "大吉", "中吉", "吉", "小吉", "凶", "大凶", "まるめし吉", "はずれ")
	randNum := rand.Intn(len(OmikujiList))
	return OmikujiList[randNum]
}

func GetSake() string {
	var SakeList []string
	SakeList = append(SakeList, "日本酒", "ハイボール", "ほっぴー", "焼酎", "びーる", "白ワイン", // 種類
		"ほろよい", "カシオレ", "黒霧島", "綾鷹", "澪", "99.99", // by name
		getHakutsuru(), getHakutsuru(), getHakutsuru())
	randNum := rand.Intn(len(SakeList))
	return SakeList[randNum]
}

func getHakutsuru() string {
	var SakeList []string
	SakeList = append(SakeList,
		"https://youtu.be/AsEHZ4PZ9tg",
		"https://youtu.be/ajtHHp0dtmg",
		"https://youtu.be/AxMdgHtb-pU",
		"https://youtu.be/otuzzpuwhso",
		"https://youtu.be/E9diOTSlSGk",
	)
	randNum := rand.Intn(len(SakeList))
	return SakeList[randNum]
}

func getTodayJikkyou() string {
	var GameList []string
	var HitoList []string
	GameList = append(GameList,
		"Five Nights at Freddy's",
		"FF14高難易度",
		"ソロアルチ",
		"お絵かき",
	)
	HitoList = append(HitoList,
		"ぽくしさん",
		"致したさん",
		"うらめしえんたん",
		"しろくろ",
	)
	randNum1 := rand.Intn(len(GameList))
	randNum2 := rand.Intn(len(HitoList))
	return HitoList[randNum2] + "の" + GameList[randNum1] + "実況"
}
