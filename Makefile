.PHONY: dev dev-api dev-web test build compose-up compose-down fmt

dev: ## Hot loop: postgres via compose + Go API + Vite dev server.
	./scripts/dev.sh

dev-api: ## Run the Go API locally (expects postgres on localhost:5432).
	cd server && go run ./cmd/server

dev-web: ## Run the Vite dev server only.
	npm run dev --workspace @vcoach/web

test: ## Go tests + web typecheck/build.
	cd server && go test -race ./...
	npm run build --workspace @vcoach/web

build: ## Build all docker images.
	docker compose build

compose-up: ## Start the full stack in the background.
	docker compose up --build -d

compose-down: ## Stop the stack.
	docker compose down

fmt: ## gofmt + goimports.
	cd server && go fmt ./...
