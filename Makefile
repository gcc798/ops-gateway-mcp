.PHONY: dev dev-backend dev-web stop build-web build test lint fmt
dev: build-web
	go run ./cmd/ai-ops-gateway serve
dev-backend:
	go run ./cmd/ai-ops-gateway serve
stop:
	pkill -f '[a]i-ops-gateway serve' || true
dev-web:
	cd web && pnpm dev
build-web:
	cd web && pnpm install && pnpm build
	rm -rf cmd/ai-ops-gateway/web/dist
	mkdir -p cmd/ai-ops-gateway/web/dist && cp -R web/dist/. cmd/ai-ops-gateway/web/dist/
build: build-web
	go build -o ai-ops-gateway ./cmd/ai-ops-gateway
test:
	go test ./...
lint:
	go vet ./...
fmt:
	gofmt -w $$(rg --files -g '*.go')
