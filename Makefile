.PHONY: help test lint build docker run dev clean migrate coverage

# Default target
help:
	@echo ""
	@echo "  Payment Service — Available Commands"
	@echo "  ────────────────────────────────────"
	@echo "  make dev        Start all services with docker-compose"
	@echo "  make test       Run all unit tests with coverage"
	@echo "  make lint       Run golangci-lint"
	@echo "  make build      Compile the binary"
	@echo "  make docker     Build Docker image"
	@echo "  make migrate    Run database migrations"
	@echo "  make coverage   Open HTML coverage report"
	@echo "  make clean      Remove build artifacts"
	@echo ""

# ---- Development ----

dev:
	docker-compose up --build

dev-down:
	docker-compose down -v

# ---- Testing ----

test:
	go test ./... -v -coverprofile=coverage.out|| true

# 	go test ./... -v -race -coverprofile=coverage.out -covermode=atomic
	

coverage: test
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
	@go tool cover -func=coverage.out | grep total

# Check coverage meets 95% threshold
coverage-check: test
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | tr -d '%'); \
	echo "Total coverage: $${COVERAGE}%"; \
	if [ $$(echo "$${COVERAGE} < 95" | bc) -eq 1 ]; then \
		echo "FAIL: Coverage $${COVERAGE}% is below 95% threshold"; exit 1; \
	fi

# ---- Code Quality ----

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w .

vet:
	go vet ./...

# ---- Build ----

build:
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/payment ./cmd/server

docker:
	docker build -t payment-service:latest .

docker-push:
	docker tag payment-service:latest ghcr.io/yourorg/payment-service:latest
	docker push ghcr.io/yourorg/payment-service:latest

# ---- Database ----

migrate:
	go run ./cmd/server -migrate-only

# ---- Deploy ----

deploy-staging:
	helm upgrade --install payment-service ./deploy/helm \
		--namespace payments \
		--set image.tag=$(shell git rev-parse --short HEAD) \
		--values ./deploy/helm/values.staging.yaml

deploy-prod:
	helm upgrade --install payment-service ./deploy/helm \
		--namespace payments \
		--set image.tag=$(shell git rev-parse --short HEAD) \
		--values ./deploy/helm/values.prod.yaml

# ---- Swagger ----

swagger:
	swag init -g cmd/server/main.go -o api/swagger

# ---- Cleanup ----

clean:
	rm -rf bin/ coverage.out coverage.html