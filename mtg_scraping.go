package main

import (
	"fmt"
	"net/http"
	"strings"
)

func randMTGCard() string {
	return "hogr"
}

func FetchHareruyaCards(cardName string) string {
	const hareruyaURLHeader = "https://www.hareruyamtg.com/ja/products/search?sort=price&order=ASC&category=&cardset=&colorsType=0&cardtypesType=0&format=&illustrator=&stock=1"
	targetName := "&product=" + cardName
	reqURL := hareruyaURLHeader + targetName
	// 検索URLを直打ち
	// resp, err := http.Get(hareruyaURLHeader)
	// if err != nil {
	// 	return "", err
	// }
	// defer resp.Body.Close()
	// HTML内から最上位のカードのリンク先URLを取得
	// 取得したURLを返却
	return reqURL
}

func RandMTGjp() string {
	var text string
	req1, err := http.NewRequest("GET", "http://gatherer.wizards.com/Pages/Card/Details.aspx?action=random", nil)
	if err != nil {
		fmt.Println(err)
		return "エラー"
	}
	req1.Header.Set("Accept-Language", "ja,en-US;q=0.7,en;q=0.3")
	cli := new(http.Client)
	resp, err := cli.Do(req1)
	if err == nil {
		defer resp.Body.Close()
	}
	urlLeft := "http://gatherer.wizards.com/Handlers/Image.ashx?multiverseid="
	SliceURL := strings.Split(resp.Request.URL.String(), "=")
	urlRight := "&type=card"

	text = urlLeft + SliceURL[1] + urlRight

	return text
}
