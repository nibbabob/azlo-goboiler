# --- Configuration ---
include .env
export

.PHONY: setup up down clean

setup:
	@echo "🔧 Provisioning unprivileged filesystem context..."
	@# 1. Proactively create the exact path Tempo expects
	@# This avoids the container having to run 'mkdir'
	mkdir -p ./tempo/data/traces
	@# 2. Assign ownership to the non-root runtime UID
	sudo chown -R 10001:10001 ./tempo/data
	@# 3. Secure permissions
	chmod -R 775 ./tempo/data
	@# 4. Generate SSL certificates
	chmod +x scripts/generate-certs.sh
	./scripts/generate-certs.sh
	@echo "✅ Setup complete."

up:
	@echo "🏗️ Launching stack with unprivileged volumes..."
	docker-compose up -d --build

down:
	@echo "🛑 Shutting down..."
	docker-compose down

clean:
	@echo "🧹 Wiping state and generated artifacts..."
	docker-compose down -v --remove-orphans
