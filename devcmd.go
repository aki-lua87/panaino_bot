package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func devCmd(cmd string) string {
	switch {
	case strings.HasPrefix(cmd, fmt.Sprintf("%s %s", appConfig.BotName, "-dev 覇者の紋章")):
		return devCoatOfArms()
	}

	return "コマンドが見つからないです・・・"
}

func devCoatOfArms() string {
	client := &http.Client{}

	req, err := http.NewRequest(
		"POST",
		appConfig.CoatOfArmsURL,
		nil,
	)
	if err != nil {
		return err.Error()
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Get Error:", err)
	}
	defer resp.Body.Close()

	byteArray, err := ioutil.ReadAll(resp.Body)

	var result bool
	if err := json.Unmarshal(byteArray, &result); err != nil {
		log.Println("Unmarshal Error", err)
	}
	if !result {
		return "覇者の紋章ゲットキャンペーン情報取得コマンドを送信しました。が失敗しました。"
	}

	return "覇者の紋章ゲットキャンペーン情報取得コマンドを送信しました。"
}
