# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM node:24-alpine AS build-frontend

RUN npm install -g pnpm@12

WORKDIR /src/frontend

COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN --mount=type=cache,id=pnpm,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile

COPY frontend/ ./
RUN pnpm build


FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build-backend

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src/backend

COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY backend/ ./
# frontend の成果物を embed ディレクトリへ配置する
COPY --from=build-frontend /src/frontend/dist/ ./internal/web/dist/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server


FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build-backend /out/server /server

EXPOSE 4000

ENTRYPOINT ["/server"]
