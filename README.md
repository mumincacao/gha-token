# gha-token

GitHub App の installation access token を取得するシンプルな CLI です。

## Install

### Binary
GitHub Releases から最新のバイナリをダウンロードしてインストール
1. [Releases](https://github.com/mumincacao/gha-token/releases) から最新のバイナリをダウンロード
2. ダウンロードしたファイルを実行可能にして、PATH の通った場所に移動
```bash
tar -xzf gha-token-*.tar.gz
chmod +x gha-token
mv gha-token /usr/local/bin/
```

### Build from source
Go 1.18+ がインストールされている環境で、以下のコマンドを実行してビルド
```bash
git clone https://github.com/mumincacao/gha-token.git
cd gha-token
go build -o gha-token ./cmd/gha-token
mv gha-token /usr/local/bin/
```

## Inputs

- `--app-id`: GitHub App の App ID
- `--private-key-path`: GitHub App の private key (PEM) ファイルパス
- `GITHUB_APP_PRIVATE_KEY`: GitHub App の private key (PEM) 本文
- `--owner`: リポジトリ owner (organization/user)
- `--repository`: リポジトリ名
- `--timeout`: HTTP timeout 秒数 (既定値: 30)
- `--debug`: デバッグログを標準エラーに出力（機密情報は出力されません）

`--private-key-path` と `GITHUB_APP_PRIVATE_KEY` の両方が指定された場合は、`--private-key-path` を優先します。

## Usage

```bash
# Fetch token
gha-token \
  --app-id 12345 \
  --private-key-path /path/to/private-key.pem \
  --owner my-org \
  --repository my-repo

# Fetch token with env var
export GITHUB_APP_PRIVATE_KEY="$(cat /path/to/private-key.pem)"
gha-token \
  --app-id 12345 \
  --owner my-org \
  --repository my-repo

# Show version
gha-token --version

# Show help
gha-token --help
```

## Development

### Build

```bash
go build -o bin/gha-token ./cmd/gha-token
./bin/gha-token --help
```

### Test

```bash
go test -v ./...
go test -cover ./...
```

### Lint

```bash
go fmt ./...
go vet ./...
```

### Local CI with act

[act](https://github.com/nektos/act) を使ってローカルで GitHub Actions を実行:

```bash
act -l                           # 利用可能なジョブを表示
act push                         # test, lint を実行
act -j test                      # test のみ実行
act -P ubuntu-latest=ghcr.io/catthehacker/ubuntu:full-latest  # カスタムイメージで実行
```

## Release

バージョン付きタグを作成してプッシュするだけで、自動的に Build + Test + Lint + Release が実行されます：

```bash
# タグを作成（例: v1.0.0）
git tag -a v1.0.0 -m "Release version 1.0.0"

# リモートへプッシュ
git push origin v1.0.0
```

GitHub Actions が以下を自動実行：
1. テスト実行（Go 1.18, 1.21 両バージョン）
2. Lint 実行
3. ビルド実行（ldflags でバージョンを埋め込み）
4. ビルド成果物を Release 

ビルド結果は GitHub UI から Artifacts セクションでダウンロード可能です。

ローカルでもビルドできます。

```bash
# 開発ビルド（バージョン: dev）
go build -o bin/gha-token ./cmd/gha-token
./bin/gha-token --version

# リリースビルド（バージョンを指定）
go build \
  -ldflags "-X github.com/mumincacao/gha-token/internal/version.Version=v1.0.0" \
  -o bin/gha-token ./cmd/gha-token
./bin/gha-token --version
```

## Feature Notes

- API 失敗時の再試行
- アクセストークンの JSON 出力
- GHES 対応

## CI/CD

GitHub Actions で以下を自動実行:
- Go 1.18, 1.21 でのテスト実行 (coverage 70%+)
- linting (go fmt, go vet)
- ビルド + リリース (タグ push 時のみ)

**トリガー:**
- `main` / `develop` ブランチへの push: テスト + lint（ビルドなし）
- `v*` タグへの push: テスト + lint + ビルド + Release
