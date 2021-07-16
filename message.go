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
	if strings.HasPrefix(message, appConfig.BotName) {
		return createMessage(message, appConfig.BotName)
	}

	if strings.HasPrefix(message, appConfig.BotName2) {
		return createMessage(message, appConfig.BotName2)
	}
	return "んにゃぴ"
}

func createMessage(message string, botName string) string {
	switch {
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", botName, "晴れる屋")):
		cardName := strings.TrimLeft(message, botName)
		cardName = strings.TrimLeft(cardName, " 晴れる屋 ")
		return FetchHareruyaCards(cardName)
	case strings.Contains(message, "お昼"), strings.Contains(message, "昼飯"), strings.Contains(message, "晩飯"), strings.Contains(message, "ばんめし"), strings.Contains(message, "ひるめし"), strings.Contains(message, "おひる"), strings.Contains(message, "夕飯"):
		return GetHirumeshi()
	case strings.Contains(message, "酒"):
		return GetSake()
	case strings.Contains(message, "の実況"):
		return getTodayJikkyou()
	case strings.Contains(message, "mtg"):
		return RandMTGjp()
	case strings.Contains(message, "メカゴジラ"):
		return getMekaGozzira()
	case strings.Contains(message, "オセロ"):
		return getOthello()
		// ここから未リファクタリング
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", botName, "セリフ")):
		return appConfig.SpreadsheetURL
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", botName, "おひる")):
		return GetHirumeshi()
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", botName, "天気")):
		return GetWether("130010")
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", botName, "東京の天気")):
		return GetWether("130010")
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", botName, "福岡の天気")):
		return GetWether("400040")
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", botName, "大阪の天気")):
		return GetWether("270000")
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", botName, "石川の天気")):
		return GetWether("170010")
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", botName, "明日の緊急")):
		return GetPESO2EmergencyQuestString("明日", time.Now().UTC().Add(time.Hour*9).Add(time.Hour*24))
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", botName, "今日の緊急")):
		return GetPESO2EmergencyQuestString("今日", time.Now().UTC().Add(time.Hour*9))
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", botName, "緊急")):
		return GetPESO2EmergencyQuestString("今日", time.Now().UTC().Add(time.Hour*9))
	case strings.HasPrefix(message, fmt.Sprintf("%s %s", botName, "覇者")):
		text, _ := GetPSO2CoatOfArms()
		return text
	}
	return getGSSMessage(strings.TrimLeft(message, botName+" "))
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

func getMekaGozzira() string {
	var List []string
	List = append(List,
		"飛行", "先制攻撃", "接死", "呪禁", "絆魂", "威迫", "到達", "トランプル", "警戒", "＋１/＋１",
	)
	randNum := rand.Intn(len(List))
	return List[randNum]
}

func getOthello() string {
	form := url.Values{}
	form.Add("m", "open")
	body := strings.NewReader(form.Encode())
	req1, err := http.NewRequest("POST", "https://el-ement.com/blog/wp-content/uploads/moonrev/api.php", body)
	if err != nil {
		fmt.Println(err)
		return "エラー"
	}
	req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	cli := new(http.Client)
	resp, err := cli.Do(req1)
	if err == nil {
		defer resp.Body.Close()
	}
	b, err := ioutil.ReadAll(resp.Body)
	text := "https://el-ement.com/blog/wp-content/uploads/moonrev/#" + string(b)
	return text
}
