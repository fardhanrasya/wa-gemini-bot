.PHONY: help build start stop restart logs status update clean reset-session backup test

# Default target
help:
	@echo "WhatsApp Gemini Bot - Makefile Commands"
	@echo ""
	@echo "Docker Commands:"
	@echo "  make build              Build Docker image"
	@echo "  make start              Start bot"
	@echo "  make stop               Stop bot"
	@echo "  make restart            Restart bot"
	@echo "  make logs               Show logs (real-time)"
	@echo "  make status             Show container status"
	@echo "  make update             Update bot (git pull + rebuild + restart)"
	@echo "  make clean              Remove container and image"
	@echo "  make reset-session      Reset WhatsApp session"
	@echo ""
	@echo "Other Commands:"
	@echo "  make setup              Initial setup (copy .env.example)"
	@echo "  make backup             Backup database"
	@echo "  make test               Run Go tests"
	@echo ""
	@echo "Production:"
	@echo "  make prod-start         Start with production config"
	@echo "  make prod-stop          Stop production"
	@echo "  make prod-logs          Show production logs"

# Setup
setup:
	@echo "Setting up environment..."
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo " Created .env file"; \
		echo " Please edit .env and fill in required values:"; \
		echo "  - GEMINI_API_KEY"; \
		echo "  - ALLOWED_GROUP_JID"; \
	else \
		echo " .env already exists"; \
	fi
	@mkdir -p data
	@echo " Created data directory"

# Docker commands
build:
	@echo "Building Docker image..."
	docker-compose -f docker/docker-compose.yml build --no-cache

start: setup
	@echo "Starting bot..."
	docker-compose -f docker/docker-compose.yml up -d
	@echo " Bot started"
	@echo "ℹ View logs with: make logs"

stop:
	@echo "Stopping bot..."
	docker-compose -f docker/docker-compose.yml down
	@echo " Bot stopped"

restart:
	@echo "Restarting bot..."
	docker-compose -f docker/docker-compose.yml restart
	@echo " Bot restarted"

logs:
	docker-compose -f docker/docker-compose.yml logs -f

status:
	docker-compose -f docker/docker-compose.yml ps

update:
	@echo "Updating bot..."
	@echo "1. Pulling latest code..."
	git pull
	@echo "2. Stopping bot..."
	docker-compose -f docker/docker-compose.yml down
	@echo "3. Rebuilding image..."
	docker-compose -f docker/docker-compose.yml build --no-cache
	@echo "4. Starting bot..."
	docker-compose -f docker/docker-compose.yml up -d
	@echo " Update complete"

clean:
	@echo " This will remove container and image!"
	@read -p "Continue? (y/N) " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose -f docker/docker-compose.yml down --rmi all; \
		echo " Cleanup complete"; \
	else \
		echo " Cancelled"; \
	fi

reset-session:
	@echo " This will reset WhatsApp session!"
	@echo " You will need to scan QR code again."
	@read -p "Continue? (y/N) " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose -f docker/docker-compose.yml down; \
		rm -f data/wa-session.db*; \
		docker-compose -f docker/docker-compose.yml up -d; \
		echo " Session reset, scan QR code in logs"; \
		sleep 2; \
		docker-compose -f docker/docker-compose.yml logs -f; \
	else \
		echo " Cancelled"; \
	fi

# Production commands
prod-start: setup
	@echo "Starting bot (production mode)..."
	docker-compose -f docker/docker-compose.prod.yml up -d
	@echo " Bot started in production mode"

prod-stop:
	@echo "Stopping bot (production mode)..."
	docker-compose -f docker/docker-compose.prod.yml down
	@echo " Bot stopped"

prod-logs:
	docker-compose -f docker/docker-compose.prod.yml logs -f

prod-restart:
	@echo "Restarting bot (production mode)..."
	docker-compose -f docker/docker-compose.prod.yml restart
	@echo " Bot restarted"

# Backup
backup:
	@echo "Creating backup..."
	@mkdir -p backups
	tar -czf backups/backup-$$(date +%Y%m%d-%H%M%S).tar.gz data/
	@echo " Backup created in backups/"

# Test
test:
	@echo "Running tests..."
	go test ./...
