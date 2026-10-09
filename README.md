# panaino_bot

「まるめし」。1つのBotトークン・1プロセスで複数のDiscordサーバに対応するGo製Botです。

## Discordでの設定

招待時に、投稿可能なシステムチャンネルで `/setup` を案内します。案内できない場合も管理者から `/setup` を実行できます。設定操作はサーバー管理権限・管理者権限を持つ人だけが使え、操作結果は本人にだけ表示されます。

| コマンド | 動作 |
| --- | --- |
| `/setup` | VC通知先のテキストチャンネルを選択・変更 |
| `/settings` | このサーバの通知設定を確認 |
| `/voice channels` | 通知対象のVCを選択（最大25件） |
| `/voice all` | 全VCを通知対象にする |
| `/voice disable` | VC通知を停止 |

設定はSQLiteの `bot.db` に保存され、再起動なしで反映します。未設定でもメンション機能は使え、VC通知は通知先を設定するまで停止しています。DMには応答しません。

```text
@まるめし お昼
@まるめし 晴れる屋 稲妻
@まるめし 天気
@まるめし help
```

食事・酒・実況などの抽選、正月のおみくじ・挨拶、天気、MTG、晴れる屋検索、オセロ、PSO2連携を引き継いでいます。PSO2・会話API・セリフのリンクなど、外部連携の個別設定は旧JSONから移行します。これらを変更するDiscordコマンドはまだありません。

## 起動とデータ

環境変数 `DISCORD_TOKEN` にBotトークンを設定して起動します。トークンはDBに保存しません。招待では `bot` と `applications.commands` を認可してください。メンション方式のため、Message Content Intentは要求しません。

```sh
./panaino-bot
./panaino-bot --db /path/to/data/bot.db
```

既定では作業ディレクトリの `bot.db` を使います。新規導入で `config.json` は不要です。既存JSONがあれば初回起動で一度だけ取り込み、既存DB設定を上書きしません。取り込み後のJSON編集は反映しません。

```sh
./panaino-bot --check
./panaino-bot --check-discord
./panaino-bot --diagnose
./panaino-bot --backup-db /path/to/backups/unused-name.db
./panaino-bot --migrate-config
./panaino-bot --import-legacy SERVER_ID=/path/to/setting.json > config.json
```

`--check` はトークン不要のDB・設定検査、`--check-discord` は認証と保存済み通知先の所属・権限検査です。両方ともDBの作成・移行やDiscordへの投稿を行いません。新規導入の空設定も検査できます。`--diagnose` はBotログインなしで実行環境とHTTPS通信を確認します。旧設定変換ではトークンを出力しません。

DBにはサーバ・チャンネルIDと設定を保存し、メッセージ履歴やVC滞在履歴は保存しません。滞在時間はメモリで計測し、開始時刻が不明な人には表示しません。個別設定、トークン、DB、バックアップはバージョン管理へ追加しません。

## 開発と配布

Go 1.26.9を使用します。SQLiteはCGO不要の実装で、サーバへのGo・SQLiteのインストールは不要です。

```sh
go test -race ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
python3 -m unittest discover -s tests -p 'test_*.py' -v
sh scripts/build.sh
```

Windowsでは `./scripts/build.ps1` で静的Linux amd64バイナリと `SHA256SUMS` を `dist/` に生成します。`master` へのPushで検査が成功するとGitHub Releasesへ公開します。

サーバでは `deploy/update.sh` で手動更新します。検証後にバイナリを切り替え、再起動失敗時は前の版へ戻します。DBは保持し、切替前にバックアップします。[配置と更新](docs/deployment.md)、[内部設計](docs/modernization-plan.md)を参照してください。
