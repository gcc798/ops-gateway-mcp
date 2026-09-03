.PHONY: dev dev-backend dev-web build-web build test lint fmt
dev: dev-backend
dev-backend:
	go run ./cmd/ai-ops-gateway serve
dev-web:
	cd web && pnpm dev
build-web:
	cd web && pnpm install && pnpm build
	mkdir -p cmd/ai-ops-gateway/web/dist && cp -R web/dist/. cmd/ai-ops-gateway/web/dist/
build: build-web
	go build -o ai-ops-gateway ./cmd/ai-ops-gateway
test:
	go test ./...
lint:
	go vet ./...
fmt:
	gofmt -w $$(rg --files -g '*.go')
