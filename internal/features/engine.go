package features

import (
	"context"
	"errors"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/aki-lua87/marumeshi_bot/internal/config"
	"github.com/aki-lua87/marumeshi_bot/internal/integrations"
)

var japan, _ = time.LoadLocation("Asia/Tokyo")

type Engine struct{ API *integrations.Client }

func (e Engine) Reply(ctx context.Context, guild config.Guild, text string, now time.Time) (string, error) {
	text = strings.TrimSpace(text)
	now = now.In(japan)
	if now.Month() == time.January && now.Day() <= 3 {
		switch {
		case strings.HasPrefix(text, "おみくじ"):
			return Omikuji(), nil
		case strings.HasPrefix(text, "あけおめ"), strings.HasPrefix(text, "あけましておめでとう"):
			return "あけおめし", nil
		}
	}
	switch {
	case text == "help" || text == "ヘルプ":
		return "呼びかけてくれたら反応するよ\nお昼 / 晩飯 / 酒 / の実況 / mtg / メカゴジラ / 晴れる屋 カード名 / オセロ\n天気 / 東京の天気 / 福岡の天気 / 大阪の天気 / 石川の天気\nセリフ / 今日の緊急 / 明日の緊急 / 覇者\nおみくじ・あけおめは1月1〜3日限定", nil
	case strings.HasPrefix(text, "晴れる屋"):
		return integrations.Hareruya(strings.TrimSpace(strings.TrimPrefix(text, "晴れる屋"))), nil
	case containsAny(text, "お昼", "昼飯", "晩飯", "ばんめし", "ひるめし", "おひる", "夕飯"):
		return GetHirumeshi(), nil
	case strings.Contains(text, "酒"):
		return GetSake(), nil
	case strings.Contains(text, "の実況"):
		return getTodayJikkyou(), nil
	case strings.Contains(strings.ToLower(text), "mtg"):
		return e.API.RandomCard(ctx)
	case strings.Contains(text, "メカゴジラ"):
		return getMekaGozzira(), nil
	case strings.Contains(text, "オセロ"):
		return e.API.Othello(ctx)
	case strings.HasPrefix(text, "セリフ"):
		if guild.SpreadsheetURL == "" {
			return "セリフのリンクはまだ設定されてないお", nil
		}
		return guild.SpreadsheetURL, nil
	case strings.HasPrefix(text, "天気"), strings.HasPrefix(text, "東京の天気"):
		return e.API.Weather(ctx, "130010")
	case strings.HasPrefix(text, "福岡の天気"):
		return e.API.Weather(ctx, "400040")
	case strings.HasPrefix(text, "大阪の天気"):
		return e.API.Weather(ctx, "270000")
	case strings.HasPrefix(text, "石川の天気"):
		return e.API.Weather(ctx, "170010")
	case strings.HasPrefix(text, "明日の緊急"):
		if guild.EmergencyURL == "" {
			return "緊急クエストのAPIはまだ設定されてないお", nil
		}
		return e.API.Emergency(ctx, guild.EmergencyURL, "明日", now.AddDate(0, 0, 1))
	case strings.HasPrefix(text, "今日の緊急"), strings.HasPrefix(text, "緊急"):
		if guild.EmergencyURL == "" {
			return "緊急クエストのAPIはまだ設定されてないお", nil
		}
		return e.API.Emergency(ctx, guild.EmergencyURL, "今日", now)
	case strings.HasPrefix(text, "覇者"):
		if guild.CoatOfArmsURL == "" {
			return "紋章キャンペーンのAPIはまだ設定されてないお", nil
		}
		return e.API.CoatOfArms(ctx, guild.CoatOfArmsURL)
	}
	fallback := guild.FallbackReply
	if fallback == "" {
		fallback = randMessege()
	}
	if guild.SpreadsheetAPI == "" || text == "" {
		return fallback, nil
	}
	reply, err := e.API.Conversation(ctx, guild.SpreadsheetAPI, text)
	if err != nil {
		return fallback, errors.New("会話APIを取得できないため代替の返答を使いました")
	}
	if strings.TrimSpace(reply) == "" {
		return fallback, nil
	}
	return reply, nil
}

func containsAny(s string, words ...string) bool {
	for _, word := range words {
		if strings.Contains(s, word) {
			return true
		}
	}
	return false
}
