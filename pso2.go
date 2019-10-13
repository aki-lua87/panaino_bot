package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// 紋章キャンペーン
func GetPSO2CoatOfArms() (string, error) {

	url := "https://xpow0wu0s5.execute-api.ap-northeast-1.amazonaws.com/v1"

	type respons struct {
		UpdateTime string   `json:"UpdateTime"`
		StringList []string `json:"StringList"`
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	var res respons
	if err := json.Unmarshal(byteArray, &res); err != nil {
		log.Println("GetList", err)
		return "不明なエラーです・・・", err
	}

	returnText := fmt.Sprintf("今週の覇者の紋章キャンペーンは以下のとおりです... \n \n")
	for _, v := range res.StringList {
		returnText = fmt.Sprintf("%s %s \n", returnText, v)
	}
	returnText = fmt.Sprintf("%s \n(データ更新 : %s)", returnText, res.UpdateTime)

	return returnText, nil
}

// 緊急クエスト
func GetPESO2EmergencyQuestString(date string, t time.Time) string {
	var postText string
	postText = fmt.Sprintln(date + "の緊急クエストは....")

	getKey := fmt.Sprintf("%d%02d%02d", t.Year(), int(t.Month()), t.Day())
	log.Println("Key => ", getKey)
	l := EmergencyListToString(GetPESO2EmergencyQuest(getKey))

	for _, v := range l {
		postText = postText + v + " \n"
	}

	return postText
}

func GetPESO2EmergencyQuest(getKey string) []EmaList {
	client := &http.Client{}

	apiurl := "https://pso2.akakitune87.net/api/emergency"

	jsonStr := `{"EventDate":"` + getKey + `"}`
	req, err := http.NewRequest(
		"POST",
		apiurl,
		bytes.NewBuffer([]byte(jsonStr)),
	)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Not Emag Get:", err)
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
	EventName string `json:"EventName"`
	EvantType string `json:"EventType"`
	Month     int    `json:"Month"`
	Date      int    `json:"Date"`
	Hour      int    `json:"Hour"`
	Minute    int    `json:"Minute"`
}

func EmergencyListToString(emaList []EmaList) []string {
	var emaListStr []string
	if len(emaList) == 0 {
		emaListStr = append(emaListStr, "予定なしです。")
		return emaListStr
	}
	for _, v := range emaList {
		ema := fmt.Sprintf("%02d:%02d %s", v.Hour, v.Minute, v.EventName)
		emaListStr = append(emaListStr, ema)
	}
	return emaListStr
}
