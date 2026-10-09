package features

import "math/rand/v2"

func randMessege() string {
	var messageList []string
	// 基本まるめし構文
	messageList = append(messageList, "まるい", "り", "それ", "そり", "まるめし", "まるくなりたい", "……ｫ'ﾝ", "んまっ！？", "んまー", "マ？", "はやめで", "マァ～")
	// スタンプ
	messageList = append(messageList, ":bread: ", ":moyai: ", ":cactus: ")
	// GOD
	messageList = append(messageList, "それになった", "すず", "ぱないの", "ﾎﾟｸｼﾎﾟｸｼ", "にょわ～")
	randNum := rand.IntN(len(messageList))
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
	randNum := rand.IntN(len(OhiruList))
	return OhiruList[randNum]
}

func Omikuji() string {
	var OmikujiList []string
	OmikujiList = append(OmikujiList, "大吉", "中吉", "吉", "小吉", "凶", "大凶", "まるめし吉", "はずれ")
	randNum := rand.IntN(len(OmikujiList))
	return OmikujiList[randNum]
}

func GetSake() string {
	var SakeList []string
	SakeList = append(SakeList, "日本酒", "ハイボール", "ほっぴー", "焼酎", "びーる", "白ワイン", // 種類
		"ほろよい", "カシオレ", "黒霧島", "綾鷹", "澪", "99.99", // by name
		getHakutsuru(), getHakutsuru(), getHakutsuru())
	randNum := rand.IntN(len(SakeList))
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
	randNum := rand.IntN(len(SakeList))
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
	randNum1 := rand.IntN(len(GameList))
	randNum2 := rand.IntN(len(HitoList))
	return HitoList[randNum2] + "の" + GameList[randNum1] + "実況"
}

func getMekaGozzira() string {
	var List []string
	List = append(List,
		"飛行", "先制攻撃", "接死", "呪禁", "絆魂", "威迫", "到達", "トランプル", "警戒", "＋１/＋１",
	)
	randNum := rand.IntN(len(List))
	return List[randNum]
}
