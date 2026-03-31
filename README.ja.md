# splunk-cli

Splunk REST API のパイプ対応 CLI クライアント。SPL 検索の実行、ジョブ管理、結果の取得をターミナルから直接行える。

[English README is here](README.md)

## 特徴

- **同期検索** — `run` でクエリを実行し、完了を待って結果を出力
- **非同期検索** — `start` → `status` → `results` で長時間ジョブを管理
- **パイプ対応** — JSON 出力を `jq`、`json-to-table` 等と組み合わせ可能
- **柔軟な認証** — トークン、ユーザー名/パスワード、環境変数、設定ファイル
- **App コンテキスト** — `--app` フラグで App 固有のルックアップやナレッジオブジェクトを使用
- **Ctrl+C 対応** — 実行中のジョブをキャンセルするかバックグラウンド継続するか選択可能

## インストール

[リリースページ](https://github.com/nlink-jp/splunk-cli/releases) からビルド済みバイナリをダウンロード。

またはソースからビルド:

```bash
git clone https://github.com/nlink-jp/splunk-cli.git
cd splunk-cli
make build
# バイナリ: dist/splunk-cli
```

## クイックスタート

```bash
# 認証情報の設定
export SPLUNK_HOST="https://your-splunk.example.com:8089"
export SPLUNK_TOKEN="your-token"

# 検索の実行
splunk-cli run --spl "index=_internal | head 10"

# jq にパイプ
splunk-cli run --spl "index=main | stats count by sourcetype" | jq .

# stdin から SPL を読み込み
cat query.spl | splunk-cli run -f -
```

## 設定

設定例ファイルをコピー:

```bash
mkdir -p ~/.config/splunk-cli
cp config.example.toml ~/.config/splunk-cli/config.toml
chmod 600 ~/.config/splunk-cli/config.toml
```

```toml
# ~/.config/splunk-cli/config.toml
[splunk]
host  = "https://your-splunk.example.com:8089"
token = "your-token"
# app = "search"
# insecure = false
# http_timeout = "30s"
# limit = 0
```

**優先順位（高い順）:** CLI フラグ → 環境変数 → 設定ファイル

### Windows: 設定ファイルのセキュリティ

Unix/macOS では、設定ファイルが他ユーザーから読み取り可能な場合に警告が表示されます（`chmod 600` を推奨）。Windows (NTFS) では、NTFS が Unix パーミッションビットをサポートしないため、このチェックは自動的にスキップされます。

**ただし、設定ファイルには認証情報が含まれる可能性があるため、保護は必要です。** Windows では NTFS ACL でアクセスを制限してください:

```powershell
# PowerShell: 設定ファイルを現在のユーザーのみに制限
$path = "$env:USERPROFILE\.config\splunk-cli\config.toml"
icacls $path /inheritance:r /grant:r "${env:USERNAME}:(R,W)"
```

または、設定ファイルに認証情報を保存せず、環境変数（`SPLUNK_TOKEN` 等）を使用する方法もあります。

| 環境変数 | 説明 |
|---|---|
| `SPLUNK_HOST` | Splunk サーバー URL（ポート含む） |
| `SPLUNK_TOKEN` | Bearer トークン（推奨） |
| `SPLUNK_USER` | ユーザー名（Basic 認証） |
| `SPLUNK_PASSWORD` | パスワード（Basic 認証） |
| `SPLUNK_APP` | 検索の App コンテキスト |

## 使い方

```
splunk-cli [command]

コマンド:
  run         SPL 検索を実行し結果を出力（同期）
  start       SPL 検索を非同期で開始し SID を出力
  status      検索ジョブのステータスを確認
  results     完了した検索ジョブの結果を取得

グローバルフラグ:
  -c, --config string           設定ファイルパス（デフォルト: ~/.config/splunk-cli/config.toml）
      --host string             Splunk サーバー URL（env: SPLUNK_HOST）
      --token string            Bearer トークン（env: SPLUNK_TOKEN）
      --user string             ユーザー名（env: SPLUNK_USER）
      --password string         パスワード（env: SPLUNK_PASSWORD）
      --app string              App コンテキスト（env: SPLUNK_APP）
      --owner string            ナレッジオブジェクトオーナー（デフォルト: nobody）
      --limit int               最大結果数（0 = 全件）
      --insecure                TLS 証明書検証をスキップ
      --http-timeout duration   リクエストごとの HTTP タイムアウト（例: 30s, 2m）
      --debug                   デバッグログを有効化
  -v, --version                 バージョン情報を表示
```

### `run` — 同期検索

```bash
# 時間範囲を指定して検索
splunk-cli run --spl "index=_internal" --earliest "-1h" --limit 10

# ファイルから SPL を読み込み
splunk-cli run -f query.spl

# stdin から SPL
echo 'index=main | stats count' | splunk-cli run -f -

# タイムアウト付き
splunk-cli run --spl "index=main | stats count by host" --timeout 5m
```

| フラグ | 説明 |
|---|---|
| `--spl <string>` | 実行する SPL クエリ |
| `-f, --file <path>` | ファイルから SPL を読み込み（`-` で stdin） |
| `--earliest <time>` | 開始時刻（例: `-1h`, `@d`, エポック） |
| `--latest <time>` | 終了時刻（例: `now`, `@d`, エポック） |
| `--timeout <duration>` | ジョブ全体のタイムアウト（例: `10m`, `1h`） |
| `--limit <int>` | 最大結果数（0 = 全件） |
| `--silent` | 進行状況メッセージを抑制 |

> **Ctrl+C**: `run` 実行中に Ctrl+C を押すと、ジョブのキャンセルまたはバックグラウンド継続を選択できます。

### `start` — 非同期検索

```bash
JOB_ID=$(splunk-cli start --spl "index=main | stats count by sourcetype")
echo "Started: $JOB_ID"
```

### `status` — ジョブステータス確認

```bash
splunk-cli status --sid "$JOB_ID"
```

### `results` — ジョブ結果取得

```bash
splunk-cli results --sid "$JOB_ID" --limit 50 --silent | jq .
```

## ビルド

```bash
make build            # 現在のプラットフォーム → dist/splunk-cli
make build-all        # 全プラットフォーム → dist/
make test             # ユニットテスト
make check            # vet → lint → test → build
make integration-test # 統合テスト（Podman + Splunk コンテナが必要）
make splunk-down      # Splunk テストコンテナを停止
make clean            # dist/ を削除
```

詳細は [BUILD.md](BUILD.md) を参照。

## ライセンス

MIT License — [LICENSE](LICENSE) を参照。
