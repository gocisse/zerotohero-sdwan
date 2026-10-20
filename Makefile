# Load .env (POSTGRES_* etc.) for targets that need it.
-include .env
export

.PHONY: help up down logs test fmt lint reset-db validate-content e2e e2e-clean observations-report course-map

COMPOSE := docker-compose
# An idle or maintenance sleep during a run makes Playwright's timers expire when the Mac wakes (seen as sporadic timeouts),
# so on macOS the suite runs under caffeinate.
KEEP_AWAKE := $(shell command -v caffeinate >/dev/null 2>&1 && echo caffeinate -i)

help: ## List the targets
	@grep -hE '^[a-z0-9-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-20s %s\n", $$1, $$2}'

up: ## Build and start all services (creates .env from .env.example if missing)
	@test -f .env || cp .env.example .env
	$(COMPOSE) up --build -d --renew-anon-volumes
	@echo "Frontend: http://localhost:$${FRONTEND_PORT:-5173}  Backend: http://localhost:$${BACKEND_PORT:-8090}/api/health"

down: ## Stop all services (data volume is kept)
	$(COMPOSE) down

logs: ## Tail logs from all services
	$(COMPOSE) logs -f

# Host-side dependencies for the checks. (docker-compose leaves an empty frontend/node_modules directory behind as a mount point, so the
# directory alone proves nothing; npm writes .package-lock.json when an install completed.)
frontend/node_modules/.package-lock.json: frontend/package-lock.json
	cd frontend && npm ci

test: frontend/node_modules/.package-lock.json ## Run backend tests (incl. Postgres integration tests; starts the db service), validate content, type-check
	$(COMPOSE) up -d --wait postgres
	cd backend && TEST_DATABASE_URL="postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable" go test -count=1 ./...
	$(MAKE) validate-content
	cd frontend && npm run typecheck && npm run test:unit

fmt: frontend/node_modules/.package-lock.json ## Format Go and TypeScript sources
	cd backend && gofmt -w .
	cd frontend && npm run fmt

lint: frontend/node_modules/.package-lock.json ## go vet, gofmt check, oxlint, prettier check
	cd backend && go vet ./... && test -z "$$(gofmt -l .)"
	cd frontend && npm run lint && npm run fmt:check

validate-content: ## Validate /content without starting the server (exits non-zero on errors)
	cd backend && go run ./cmd/contentcheck -dir ../content

# The e2e suite runs in its OWN compose project ("vcacademy-e2e"): its own containers, its own database volume and
# different host ports, so it can never read or write the learning database that `make up` uses.
E2E_PROJECT := vcacademy-e2e
E2E_ENV := POSTGRES_DB=academy_e2e POSTGRES_PORT=$${E2E_POSTGRES_PORT:-5433} BACKEND_PORT=$${E2E_BACKEND_PORT:-8091} FRONTEND_PORT=$${E2E_FRONTEND_PORT:-5174}
E2E_COMPOSE := $(E2E_ENV) $(COMPOSE) -p $(E2E_PROJECT)

e2e: ## Run the Playwright suite against an ISOLATED stack (own project, ports and database); never touches the learning DB
	@test -f .env || cp .env.example .env
	@if [ -z "$$E2E_KEEP_DB" ]; then echo "e2e: starting from a fresh e2e database (the learning database is not touched)"; $(E2E_COMPOSE) down -v; fi
	$(E2E_COMPOSE) up --build -d
	@echo "e2e: waiting for the isolated stack"; for i in $$(seq 1 90); do curl -sf http://localhost:$${E2E_FRONTEND_PORT:-5174}/api/health >/dev/null && break; sleep 1; done; curl -sf http://localhost:$${E2E_FRONTEND_PORT:-5174}/api/health >/dev/null || { echo "e2e stack did not become healthy"; exit 1; }
	cd e2e && ( test -d node_modules || npm install ) && npx playwright install chromium >/dev/null && E2E_BASE_URL=http://localhost:$${E2E_FRONTEND_PORT:-5174} $(KEEP_AWAKE) npx playwright test; rc=$$?; cd .. && $(E2E_COMPOSE) down; exit $$rc

e2e-clean: ## Delete the e2e database volume (the learning database is not touched)
	$(E2E_COMPOSE) down -v

observations-report: ## Write docs/OBSERVATION-PRIORITIES.md: every VERIFY row and assumption, what depends on it, ranked (your lab checklist)
	cd backend && go run ./cmd/obsreport -content ../content -sim internal/sim -model ../docs/SIMULATOR-MODEL.md -out ../docs/OBSERVATION-PRIORITIES.md

course-map: ## Write docs/COURSE-MAP.md from the content (sections, lessons, quiz topics, diagrams, labs, screens, goals)
	cd backend && go run ./cmd/obsreport -content ../content -course-map ../docs/COURSE-MAP.md

reset-db: ## Destroy the database volume (asks for confirmation)
	@printf "This deletes ALL progress data. Type 'yes' to continue: " && read ans && [ "$$ans" = "yes" ] || { echo "Aborted."; exit 1; }
	$(COMPOSE) down -v
	@echo "Database volume removed. Run 'make up' to recreate it."
