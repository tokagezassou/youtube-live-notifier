# youtube-live-notifier

YouTubeの「もあちの配信」チャンネル(https://www.youtube.com/@moa-ch1)の配信枠を監視し、Discordの「おともあちハウス」サーバーの「配信告知」チャンネルに次の2種類を通知する bot。

- **枠立て通知**：新しい配信予定枠ができたとき
- **配信開始通知**：予定枠の配信が始まったとき

## 構成

```
Cloud Scheduler ──(認証付きで定期呼び出し)──▶ Cloud Run
                                                  │
                                                  ├─▶ YouTube RSS（新着動画の取得）
                                                  ├─▶ YouTube Data API（枠の状態確認・配信予定の検索）
                                                  ├─▶ Firestore（検出済みの枠と通知状態の管理）
                                                  └─▶ Discord Webhook（通知の送信）
```

| 項目 | 内容 |
|---|---|
| Google Cloud プロジェクト | `youtube-live-notifier-test`（ID: `youtube-live-notifier-test`） |
| GitHub | `tokagezassou/youtube-live-notifier`|
| 実行環境 | Cloud Run（サービス名: `youtube-live-notifier` / リージョン: `asia-northeast1`） |
| 呼び出し | Cloud Scheduler（2ジョブ。下記「実行スケジュール」参照）。Cloud Run は認証必須で、Scheduler のサービスアカウントのみ起動可能 |
| データベース | Firestore（`<データベース名>`） |
| 言語 | Go（バージョンは `go.mod` を参照） |
| コンテナ | `Dockerfile`（ビルド: golang-alpine / 実行: alpine、一般ユーザーで実行） |

命名にtestが入っているのは、配信者に権限を譲渡して配信者環境で動かしてもらう想定だったため。実際には保守運用の観点から権限譲渡を行わずに、私の環境で動かし続けている。

## 実行スケジュール

YouTube Data API の利用枠（1日 10,000 ユニット）に収めるため、取得方法ごとに間隔を分けている。

| ジョブ | 間隔 | 呼び出す処理 | 内容 |
|---|---|---|---|
| `youtube-live-notifier-trigger` | 1分ごと | `/check` → `CheckAndNotify` | RSS で新着を検出し、videos.list で枠の状態を確認して枠立て通知。あわせて監視中の枠の配信開始を確認して開始通知 |
| `youtube-live-notifier-manual` | 30分ごと | `/check_by_search` → `SearchAndNotify` | search.list で配信予定を検索し、RSS で拾えなかった枠を補う |

### 間隔を分けている理由
- **RSS** は API の利用枠を消費しないため、1分ごとに取得できる。
- **videos.list** は1回1ユニットと軽い。新着候補があるときと、配信予定時刻の前後（開始130分前〜250分後）の監視中だけ呼ばれる。
- **search.list** は1回100ユニットと重いため、30分ごと（1日48回 = 4,800ユニット）に抑えている。

間隔を変えるときは、合計が1日 10,000 ユニットを超えないか計算すること。

## 環境変数

値はSecret Managerを参照。

| 変数名 | 内容 |
|---|---|
| `YOUTUBE_CHANNEL_ID` | 監視する YouTube チャンネルの ID |
| `YOUTUBE_API_KEY` | YouTube Data API キー |
| `DISCORD_WEBHOOK_URL` | 通知先の Discord Webhook URL） |
| `DISCORD_ROLE_ID` | 通知でメンションする Discord ロールの ID |
| `GCP_PROJECT_ID` | Firestore に接続する Google Cloud プロジェクトの ID（`main.go` で読み込み） |
| `PORT` | 待ち受けポート（Dockerfile で 8080 を指定） |

ローカルでは `.env` に記述する。

## デプロイ

```
gcloud run deploy youtube-live-notifier \
  --source . \
  --region asia-northeast1 \
  --no-allow-unauthenticated \
  --set-secrets="GCP_PROJECT_ID=GCP_PROJECT_ID:latest,YOUTUBE_CHANNEL_ID=YOUTUBE_CHANNEL_ID:latest,DISCORD_ROLE_ID=DISCORD_ROLE_ID:latest,YOUTUBE_API_KEY=YOUTUBE_API_KEY:latest,DISCORD_WEBHOOK_URL=DISCORD_WEBHOOK_URL:latest"
```

環境変数に変更が無いときには`--set-secrets=`の行は省略してよい。

## 保守

### 自動チェック
- **govulncheck**（`.github/workflows/govulncheck.yml`）: 毎週月曜と main への push 時に実行。脆弱性があると失敗し、メールで通知される。
- **Dependabot**（`.github/dependabot.yml`）: 毎週月曜に Go の依存・Docker イメージ・Actions の更新 PR を作成する。

### 通知が来たら
1. PR の govulncheck が緑であることを確認してマージ
2. 再デプロイ（Go はバイナリに組み込まれるため、デプロイしないと反映されない）