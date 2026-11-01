#!/bin/sh
set -eu
cd "$(dirname "$0")"
(cd web && npm ci --no-audit --no-fund && npm run build)
rm -rf internal/frontend/dist
mkdir -p internal/frontend/dist bin
cp -R web/dist/. internal/frontend/dist/
CGO_ENABLED=0 go build -trimpath -o bin/skilltree ./cmd/server
CGO_ENABLED=0 go build -trimpath -o bin/skilltree-migrate ./cmd/migrate-postgres
CGO_ENABLED=0 go build -trimpath -o bin/skilltree-repair ./cmd/repair-migration
CGO_ENABLED=0 go build -trimpath -o bin/skilltree-backup ./cmd/backup
