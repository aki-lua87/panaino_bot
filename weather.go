package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

// 天気情報取得
func GetWether(id string) string {
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
