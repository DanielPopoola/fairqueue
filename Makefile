.PHONY: run build test test-unit test-integration migrate migrate-down lint swagger docker-up docker-down

# ── Local development ─────────────────────────────────────────

run:
	go run ./cmd/api

build:
	go build -o bin/fairqueue ./cmd/api

# ── Tests ─────────────────────────────────────────────────────

# Unit tests only (domain layer — fast, no containers)
test-unit:
	go test ./internal/domain/... ./internal/auth/...

# Integration tests (spins up real Postgres + Redis via testcontainers)
test-integration:
	go test ./internal/store/... ./internal/service/... ./internal/worker/... -timeout 120s

# All tests
test:
	go test ./... -timeout 120s

# ── Database ──────────────────────────────────────────────────

# Run migrations against the local DB (requires .env to be loaded)
migrate:
	@echo "Running migrations..."
	@for f in migrations/*.sql; do \
		echo "Applying $$f"; \
		psql "$$DATABASE_URL" -f "$$f"; \
	done

# ── Docs ──────────────────────────────────────────────────────

swagger:
	swag init -g internal/api/server.go -o docs/
	@echo "Swagger UI available at http://localhost:8080/swagger/index.html"

# ── Lint ──────────────────────────────────────────────────────

lint:
	golangci-lint run ./...

# ── Docker ───────────────────────────────────────────────────

docker-up:
	docker compose up --build -d
	@echo "FairQueue running at http://localhost:8080"
	@echo "Swagger UI at http://localhost:8080/swagger/index.html"

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f app

# Start only infrastructure (Postgres + Redis) — run app locally
infra-up:
	docker compose up postgres redis -d

infra-down:
	docker compose down postgres redis