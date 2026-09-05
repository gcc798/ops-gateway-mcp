.PHONY: dev dev-backend dev-web stop build-web build test lint fmt
dev: build-web
	go run ./cmd/ops-gateway-mcp serve
dev-backend:
	go run ./cmd/ops-gateway-mcp serve
stop:
	pkill -f '[o]ps-gateway-mcp serve' || true
dev-web:
	cd web && pnpm dev
build-web:
	cd web && pnpm install && pnpm build
	rm -rf cmd/ops-gateway-mcp/web/dist
	mkdir -p cmd/ops-gateway-mcp/web/dist && cp -R web/dist/. cmd/ops-gateway-mcp/web/dist/
build: build-web
	go build -o ops-gateway-mcp ./cmd/ops-gateway-mcp
test:
	go test ./...
lint:
	go vet ./...
fmt:
	gofmt -w $$(rg --files -g '*.go')
