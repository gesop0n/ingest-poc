# ADR 0001: リポジトリ内で backend と frontend をトップレベルで分割する

## Status

Accepted

## Context

リポジトリのルートが Go module で、React の SPA は `web/` に置いていた。
SPA のビルド成果物を Go に embed するパッケージは `frontend/` という名前で、
Vite が `../frontend/dist` へ書き出していた。

この構成には次の問題があった。

- `frontend/` が Go のパッケージで、`web/` が React 本体という命名で混乱を招く
- `frontend/dist` は git 管理外なのに `//go:embed dist` が必須のため、
  clone 直後は `pnpm build` しないと `go build` や gopls が失敗する
- ルートが Go module なので、`go ./...` や gopls が `web/node_modules` まで走査する

## Decision

1 リポジトリのまま、トップレベルを `backend/` と `frontend/` に分ける。
構成は umputun/remark42 を参考にする。

- `backend/` に `go.mod` を置く (module `github.com/gesop0n/ingest-poc/backend`)
- `frontend/` は Vite 標準の `frontend/dist` に出力する
- `make build` と Dockerfile が `frontend/dist` を `backend/internal/web/dist` にコピーし、
  1 バイナリに embed する
- `backend/internal/web/dist` には `.gitkeep` だけを置き、`//go:embed all:dist` で
  frontend 未ビルドでもコンパイルできるようにする。未ビルド時、サーバは API のみを配信する

リポジトリ自体を分ける案は採用しない。
API と画面の変更を 1 コミットで運用できなくなり、
1 バイナリで配信する今の構成では手間が増えるだけのため。

## Consequences

### Pros

- backend は Node なしでビルド・起動でき、frontend は backend の内部パスを知らずに済む
- Go と Node のツールチェーンが互いのディレクトリを走査しない
- CI のパスフィルタや Docker のステージ単位でキャッシュを分けられる

### Cons

- 本番相当の動作確認には `make build` (frontend ビルド → コピー → go build) が必要
- Go の import パスに `/backend` が入る
