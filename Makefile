BACKEND_DIR  := backend
FRONTEND_DIR := frontend
EMBED_DIR    := $(BACKEND_DIR)/internal/web/dist
BIN_DIR      := bin

.PHONY: build build-frontend build-backend embed-frontend \
	dev-backend dev-frontend test lint clean

## build: frontend をビルドして backend のバイナリに埋め込む
build: build-frontend embed-frontend build-backend

build-frontend:
	pnpm -C $(FRONTEND_DIR) install --frozen-lockfile
	pnpm -C $(FRONTEND_DIR) build

## embed-frontend: frontend/dist を backend の embed ディレクトリへコピーする
embed-frontend:
	find $(EMBED_DIR) -mindepth 1 ! -name .gitkeep -exec rm -rf {} +
	cp -R $(FRONTEND_DIR)/dist/. $(EMBED_DIR)/

build-backend:
	cd $(BACKEND_DIR) && go build -o ../$(BIN_DIR)/server ./cmd/server

## dev-backend: API サーバを起動する (:4000)
dev-backend:
	cd $(BACKEND_DIR) && go run ./cmd/server

## dev-frontend: Vite の開発サーバを起動する (/api は :4000 へ転送)
dev-frontend:
	pnpm -C $(FRONTEND_DIR) dev



## run: 本番と同じ 1 ポート構成で起動する
run: build
	./$(BIN_DIR)/server

test:
	cd $(BACKEND_DIR) && go test ./...

lint:
	cd $(BACKEND_DIR) && go vet ./...
	pnpm -C $(FRONTEND_DIR) lint

clean:
	rm -rf $(BIN_DIR) $(FRONTEND_DIR)/dist
	find $(EMBED_DIR) -mindepth 1 ! -name .gitkeep -exec rm -rf {} +
