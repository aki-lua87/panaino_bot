package main


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