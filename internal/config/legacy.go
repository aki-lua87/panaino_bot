package config

import (
	"encoding/json"
	"errors"
	"io"
	"os"
)

// ImportLegacy reads settings without copying the old DiscordToken to the new config.
// The production conversation API was disabled, so enabling it is an explicit config edit.
func ImportLegacy(path string) (Guild, error) {
	f, err := os.Open(path)
	if err != nil {
		return Guild{}, errors.New("旧設定ファイルを開けません")
	}
	defer f.Close()
	var old struct {
		SpreadsheetURL string
		CoatOfArmsURL  string
		TextChannelID  string `json:"TC_ID"`
	}
	if err := json.NewDecoder(io.LimitReader(f, 1<<20)).Decode(&old); err != nil {
		return Guild{}, errors.New("旧設定JSONが不正です")
	}
	coat := old.CoatOfArmsURL
	if coat == "" {
		coat = "https://xpow0wu0s5.execute-api.ap-northeast-1.amazonaws.com/v1"
	}
	return Guild{
		VoiceTextChannelID: old.TextChannelID,
		SpreadsheetURL:     old.SpreadsheetURL,
		FallbackReply:      "まるめし",
		CoatOfArmsURL:      coat,
		EmergencyURL:       "https://pso2.akakitune87.net/api/emergency",
	}, nil
}
