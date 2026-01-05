.PHONY: help build run test clean dev prod logs stop restart

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development commands
dev: ## Start development environment
	docker-compose up -d
	@echo "Development environment started!"
	@echo "Frontend: http://localhost:3000"
	@echo "Backend: http://localhost:8080"
	@echo "Kafka UI: http://localhost:8081"
	@echo "Analytics: http://localhost:9090/metrics"

dev-build: ## Build and start development environment
	docker-compose up -d --build

dev-logs: ## Show development logs
	docker-compose logs -f

dev-stop: ## Stop development environment
	docker-compose down

dev-clean: ## Stop and remove development containers, networks, and volumes
	docker-compose down -v --remove-orphans

# Production commands
prod: ## Start production environment
	docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
	@echo "Production environment started!"

prod-build: ## Build and start production environment
	docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build

prod-logs: ## Show production logs
	docker-compose -f docker-compose.yml -f docker-compose.prod.yml logs -f

prod-stop: ## Stop production environment
	docker-compose -f docker-compose.yml -f docker-compose.prod.yml down

prod-clean: ## Stop and remove production containers, networks, and volumes
	docker-compose -f docker-compose.yml -f docker-compose.prod.yml down -v --remove-orphans

# Build commands
build: ## Build Go backend locally
	go build -o bin/server cmd/server/main.go

build-consumer: ## Build analytics consumer locally
	go build -o bin/analytics-consumer cmd/analytics-consumer/main.go

build-docker: ## Build all Docker images
	docker-compose build

build-backend: ## Build backend Docker image
	docker build -f Dockerfile.backend -t connect-four-backend .

build-consumer-docker: ## Build consumer Docker image
	docker build -f Dockerfile.consumer -t connect-four-consumer .

build-frontend: ## Build frontend Docker image
	docker build -f web/Dockerfile -t connect-four-frontend ./web

# Local run commands
run: ## Run backend locally
	go run cmd/server/main.go

run-consumer: ## Run analytics consumer locally
	go run cmd/analytics-consumer/main.go

# Service-specific commands
backend: ## Start only backend services (postgres, kafka, backend)
	docker-compose up -d postgres kafka1 kafka2 kafka3 backend

frontend: ## Start only frontend
	docker-compose up -d frontend

consumer: ## Start only analytics consumer
	docker-compose up -d analytics-consumer

kafka: ## Start only Kafka cluster
	docker-compose up -d zookeeper kafka1 kafka2 kafka3

# Frontend commands
frontend-install: ## Install frontend dependencies
	cd web && npm install

frontend-build: ## Build frontend for production
	cd web && npm install && npm run build

frontend-dev: ## Run frontend in development mode
	cd web && npm start

frontend-test: ## Run frontend tests
	cd web && npm test

# Utility commands
logs: ## Show logs for all services
	docker-compose logs -f

logs-backend: ## Show backend logs
	docker-compose logs -f backend

logs-frontend: ## Show frontend logs
	docker-compose logs -f frontend

logs-consumer: ## Show consumer logs
	docker-compose logs -f analytics-consumer

logs-kafka: ## Show Kafka logs
	docker-compose logs -f kafka1 kafka2 kafka3

status: ## Show status of all services
	docker-compose ps

restart: ## Restart all services
	docker-compose restart

restart-backend: ## Restart backend service
	docker-compose restart backend

restart-frontend: ## Restart frontend service
	docker-compose restart frontend

restart-consumer: ## Restart consumer service
	docker-compose restart analytics-consumer

# Testing commands
test: ## Run Go tests locally
	go test ./...

test-coverage: ## Run tests with coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-integration: ## Run integration tests
	go test -tags=integration ./...

# Monitoring commands
metrics: ## Show analytics metrics
	curl http://localhost:9090/metrics

health: ## Check health of all services
	@echo "Backend health:"
	@curl -s http://localhost:8080/health || echo "Backend not responding"
	@echo "\nFrontend health:"
	@curl -s http://localhost:3000/health || echo "Frontend not responding"
	@echo "\nConsumer metrics:"
	@curl -s http://localhost:9090/metrics | head -5 || echo "Consumer not responding"

# Cleanup commands
clean: ## Remove build artifacts and containers
	rm -rf bin/
	rm -rf web/build/
	rm -f coverage.out coverage.html
	docker-compose down -v --remove-orphans

clean-all: ## Remove everything including images
	rm -rf bin/
	rm -rf web/build/
	rm -f coverage.out coverage.html
	docker-compose down -v --remove-orphans --rmi all
	docker system prune -af

# Go-specific commands
deps: ## Install/update Go dependencies
	go mod tidy
	go mod download

lint: ## Run Go linter
	golangci-lint run

fmt: ## Format Go code
	go fmt ./...

# Docker maintenance
docker-clean: ## Clean up Docker system
	docker system prune -f
	docker volume prune -f
	docker network prune -f

docker-stats: ## Show Docker container stats
	docker stats

# Environment setup
setup: ## Setup development environment
	cp .env.example .env
	@echo "Environment file created. Please update .env with your settings."

# Legacy commands (for backward compatibility)
docker-up: dev ## Start Docker services (alias for dev)
docker-down: dev-stop ## Stop Docker services (alias for dev-stop)

# Full setup commands
setup-dev: setup frontend-install dev ## Complete development setup
	@echo "Development environment is ready!"
	@echo "Visit http://localhost:3000 to play the game"

build-all: frontend-build build build-consumer ## Build all components
	@echo "All components built successfully!"

# Quick start
quick-start: setup-dev ## Quick start development environment
	@echo "Connect Four development environment is ready!"
	@echo "Visit http://localhost:3000 to play the game"