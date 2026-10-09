# panaino_bot

まるめし。1つのBotトークン・1プロセスで複数のDiscordサーバに対応するGo製Botです。

## 使い方

Botをメンションして日本語で呼びかけます。

```text
@まるめし お昼
@まるめし 晴れる屋 稲妻
@まるめし 天気
@まるめし help
```

食事・酒・実況・メカゴジラの抽選、正月のおみくじ・挨拶、天気、MTG、晴れる屋検索、オセロ、PSO2連携を引き継いでいます。天気はlivedoor互換APIへ、ランダムMTGはScryfallの日本語カード取得へ交換しました。PSO2とオセロの現行サービスの稼働確認は別途必要です。

## 設定

`config.example.json` を `config.json` にコピーして、`guilds` のキーを実際のDiscordサーバIDに置き換えます。例のIDはダミーです。設定したサーバだけで応答し、DMには応答しません。

| 設定 | 意味 |
| --- | --- |
| `voice_text_channel_id` | VC入退室を通知する、そのサーバ内のテキストチャンネルID。空なら通知しない |
| `voice_channel_ids` | 監視するVCのID一覧。空配列ならサーバ内の全VC |
| `spreadsheet_url` | 「セリフ」に返すリンク |
| `spreadsheet_api` | 会話API。空なら外部APIを呼ばない |
| `fallback_reply` | 会話API未使用・取得失敗時の返答。`まるめし`で本番の固定返答を維持。空なら従来の候補から抽選 |
| `coat_of_arms_url` / `emergency_url` | PSO2 API。空なら未設定と案内 |

BotトークンはJSONに保存せず、環境変数 `DISCORD_TOKEN` に設定します。`Bot ` 接頭辞はあってもなくても構いません。トークン・設定本文・会話本文・API応答本文はログ出力しません。

旧設定の変換には次のコマンドを使えます。トークンは出力されません。旧コードで無効化されていた会話APIと、使用されていなかった `VC_ID` は自動で有効化しません。

```sh
./panaino-bot \
  --import-legacy SERVER_A_ID=/path/to/old-a/setting.json \
  --import-legacy SERVER_B_ID=/path/to/old-b/setting.json > config.json
```

## 開発とビルド

Go 1.26.8を使用します。サーバ側にGoをインストールする必要はありません。

```sh
go test -race ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
sh scripts/build.sh
```

Windowsでは `./scripts/build.ps1` でLinux用の静的バイナリを `dist/panaino-bot` に生成できます。Linux用バイナリをUbuntu 16.04上で起動・設定検証し、Discord・天気・ScryfallへのDNS/TLS通信を確認済みです。

```sh
./panaino-bot --version
./panaino-bot --config config.json --check
./panaino-bot --diagnose
./panaino-bot --config config.json --check-discord
```

`--check` はトークン不要の設定検証。`--diagnose` はBotログインなしでHTTPS接続を確認します。`--check-discord` は `DISCORD_TOKEN` を使い、Botの参加先、通知先の所属・種別・権限をREST APIで確認します。これらはDiscordへの投稿を行いません。

## 運用

`master`へのPushで検査が成功すると、GitHub Releasesに `build-<コミットSHA>` としてLinux用バイナリ `panaino-bot` と `SHA256SUMS` を公開します。トークンや設定は公開しません。サーバの更新は手動です。初回配置後は次のコマンドで最新版へ更新できます。

```sh
cd /home/akakitune87/panaino_bot/modern
bash update.sh
# 前の版へ戻す場合
bash update.sh --rollback
```

更新前にチェックサム・設定・Discordの参加先と権限を検証し、再起動後にプロセスが正常に維持されなければ旧バイナリへ戻します。初回配置は `bash update.sh --stage` を使います。詳細は下記の切替手順を参照してください。

[Ubuntu 16.04での切替手順](docs/deployment.md)と[実装・移行計画](docs/modernization-plan.md)を参照してください。サービス定義はsystemd 229で検証済みです。既存の `restart.sh` は旧版用として保持しており、新版の運用には使用しません。

初回は残すBotを両方のサーバに招待してください。メンションを受け取る方式なので、Message Content Intentを要求しません。通知先では「チャンネルを見る」「メッセージを送信」を許可し、カードなどのリンクプレビューを使う場合は「埋め込みリンク」も許可します。

VCの滞在時間は各チャンネルへの入室からの時間です。移動先では計測をリセットし、ミュート変更は通知しません。起動時にすでにVCにいた人や再接続後に開始時刻が不明な人には、滞在時間を表示しません。

処理待ちは128件までで順に処理します。超過分はログに記録して省略します。通知本文はDiscordの長さ制限に合わせて省略し、外部APIの返答や表示名に含まれるメンションは通知として発火させません。

GitHub Actionsにテスト・race検査・vet・govulncheck・静的Linuxビルド、DependabotにGoとActionsの週次更新を設定しています。2026-10-08時点のgovulncheckでは到達する脆弱性は0件。依存する `x/crypto` モジュール内に、BotがimportしていないOpenPGPの保守終了報告 `GO-2026-5932` が残ります。
