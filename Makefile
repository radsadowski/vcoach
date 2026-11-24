.PHONY: dev dev-api dev-web test test-go test-web build compose-up compose-down e2e fmt swagger

swagger: ## Regenerate the OpenAPI spec from handler annotations (needs: go install github.com/swaggo/swag/cmd/swag@latest).
	cd server && swag init -g cmd/server/main.go --parseInternal

dev: ## Hot loop: postgres via compose + Go API + Vite dev server.
	./scripts/dev.sh

dev-api: ## Run the Go API with hot reload (air; expects postgres on localhost:5432).
	cd server && air

dev-web: ## Run the Vite dev server only.
	npm run dev --workspace @vcoach/web

test: ## Everything: Go tests, web tests, web build.
	$(MAKE) test-go test-web

test-go: ## Go tests with race detector (needs Docker for testcontainers).
	cd server && go test -race ./...

test-web: ## Web + shared vitest suites and the production build.
	npm test --workspace @vcoach/shared
	npm test --workspace @vcoach/web
	npm run build --workspace @vcoach/web

build: ## Build all docker images.
	docker compose build

compose-up: ## Start the full stack in the background.
	docker compose up --build -d

compose-down: ## Stop the stack.
	docker compose down

e2e: ## Run the end-to-end smoke test against the running stack.
	bash scripts/e2e.sh

fmt: ## gofmt + goimports.
	cd server && go fmt ./...
