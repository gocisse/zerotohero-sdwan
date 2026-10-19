# Load .env (POSTGRES_* etc.) for targets that need it.
-include .env
export

.PHONY: up down logs test fmt lint reset-db validate-content e2e

COMPOSE := docker-compose

up: ## Build and start all services (creates .env from .env.example if missing)
	@test -f .env || cp .env.example .env
	$(COMPOSE) up --build -d --renew-anon-volumes
	@echo "Frontend: http://localhost:$${FRONTEND_PORT:-5173}  Backend: http://localhost:$${BACKEND_PORT:-8090}/api/health"

down: ## Stop all services (data volume is kept)
	$(COMPOSE) down

logs: ## Tail logs from all services
	$(COMPOSE) logs -f

test: ## Run backend tests (incl. Postgres integration tests; starts the db service), validate content, type-check
	$(COMPOSE) up -d --wait postgres
	cd backend && TEST_DATABASE_URL="postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable" go test -count=1 ./...
	$(MAKE) validate-content
	cd frontend && npm run typecheck && npm run test:unit

fmt: ## Format Go and TypeScript sources
	cd backend && gofmt -w .
	cd frontend && npm run fmt

lint: ## go vet, gofmt check, oxlint, prettier check
	cd backend && go vet ./... && test -z "$$(gofmt -l .)"
	cd frontend && npm run lint && npm run fmt:check

validate-content: ## Validate /content without starting the server (exits non-zero on errors)
	cd backend && go run ./cmd/contentcheck -dir ../content

e2e: ## Run the Playwright regression suite on a FRESH database (deletes the DB volume unless E2E_KEEP_DB=1)
	@test -f .env || cp .env.example .env
	@if [ -z "$$E2E_KEEP_DB" ]; then echo "e2e: resetting the database volume for a fresh start"; $(COMPOSE) down -v; fi
	$(COMPOSE) up --build -d
	@echo "e2e: waiting for the stack"; for i in $$(seq 1 60); do curl -sf http://localhost:$${FRONTEND_PORT:-5173}/api/health >/dev/null && break; sleep 1; done; curl -sf http://localhost:$${FRONTEND_PORT:-5173}/api/health >/dev/null || { echo "stack did not become healthy"; exit 1; }
	cd e2e && ( test -d node_modules || npm install ) && npx playwright install chromium >/dev/null && E2E_BASE_URL=http://localhost:$${FRONTEND_PORT:-5173} npx playwright test

reset-db: ## Destroy the database volume (asks for confirmation)
	@printf "This deletes ALL progress data. Type 'yes' to continue: " && read ans && [ "$$ans" = "yes" ] || { echo "Aborted."; exit 1; }
	$(COMPOSE) down -v
	@echo "Database volume removed. Run 'make up' to recreate it."
