🚀 Azlo Go Template

A production-ready SaaS Starter Kit built for speed, security, and scalability. This boilerplate provides a solid foundation for building high-performance backends with Go, complete with a full observability stack and enterprise-grade security practices.

Free Resource by Azlo.pro — I help businesses build custom automation workflows and high-performance backends. This template is provided free to the community to help you ship your MVP faster.

✨ Features

🔐 Security First: JWT authentication, Redis-backed rate limiting, strict CSP/Security headers, and SSL/TLS everywhere.

👁️ Full Observability: Pre-configured LGTM Stack (Loki, Grafana, Tempo, Prometheus) for logs, metrics, and traces.

🐳 Production Ready: Multi-stage Docker builds using Distroless images, running as non-root users.

⚡ High Performance: Go 1.24+ backend with pgx connection pooling and Redis caching.

🛡️ Secret Management: Docker Secrets integration for secure credential handling (no plain-text env vars in containers).

📱 Modern UI: Includes a responsive, vanilla JS/CSS frontend dashboard (Dark Mode) as a starting point.

🛠️ Tech Stack

Backend: Go (Golang)

Database: PostgreSQL 18 (Alpine)

Cache: Redis 8

Proxy: Nginx (Reverse Proxy & SSL Termination)

Monitoring: Prometheus, Grafana, Loki (Logs), Tempo (Tracing), OpenTelemetry

Infrastructure: Docker Compose

📋 Prerequisites

Docker & Docker Compose

Make (optional, but recommended) or a terminal (Bash/PowerShell)

OpenSSL (usually pre-installed on Linux/Mac/Git Bash)

🚀 Quick Start

1. Clone the Repository

git clone [https://github.com/your-username/azlo-go-template.git](https://github.com/your-username/azlo-go-template.git)
cd azlo-go-template


2. Configure Environment

Copy the example configuration. The defaults are set up for local development.

cp .env.example .env


Note: For production, you must change the APP_SECRET, POSTGRES_PASSWORD, and REDIS_PASSWORD in your .env file.

3. Initialize Secrets

This project uses Docker Secrets to securely manage credentials. Run the migration script to generate the secret files from your .env configuration:

Linux / macOS:

chmod +x migrate.sh
./migrate.sh


Windows (PowerShell):

./migrate.ps1


4. Generate SSL Certificates

Since certificates are not committed to the repository for security reasons, you must generate local development certificates before starting the stack.

chmod +x scripts/generate-certs.sh
./scripts/generate-certs.sh


5. Start the Stack

docker-compose up -d --build


6. Access the Application

Web App: https://localhost (Accept the self-signed certificate warning)

Grafana: http://localhost:3000 (User: admin, Pass: adminadmin)

Prometheus: http://localhost:9090

API Health: https://localhost/health

🏗️ Project Structure

├── api-service/        # Go API source code
│   ├── cmd/            # Entry points (api, healthcheck)
│   ├── internal/       # Private application code (handlers, models, middleware)
│   └── Dockerfile      # Multi-stage build definition
├── www/                # Static frontend (HTML, JS, CSS)
├── nginx/              # Nginx configuration
├── grafana/            # Dashboards and provisioning
├── prometheus/         # Metric scraping config
├── loki/               # Log aggregation config
├── tempo/              # Distributed tracing config
├── otel-collector/     # OpenTelemetry collector
├── certs/              # SSL certificates (Generated via script)
├── secrets/            # Generated secret files (do not commit)
└── scripts/            # Database init & cert generation scripts


🔧 Development Workflow

Adding a New Route

Handler: Create a new function in api-service/internal/handlers/.

Route: Register it in api-service/internal/router/router.go.

Test: The API hot-reloads on restart.

# Rebuild just the API service after code changes
docker-compose up -d --build api


Database Migrations

The project uses scripts/init-db-ssl.sh for initial setup. For ongoing schema changes, you can add .sql files to the docker-entrypoint-initdb.d volume or integrate a tool like golang-migrate.

📦 Deployment

A production-ready docker-compose.prod.yml is included. It differs from dev by:

Enforcing stricter resource limits.

Removing exposed internal ports (Redis/Postgres).

Assuming external secrets management in a Swarm/Orchestrator environment.

To build for production:

# Update the image reference in docker-compose.prod.yml first!
docker-compose -f docker-compose.prod.yml build


🤝 Need Custom Development?

This template is a great starting point, but every business has unique needs. If you need help scaling this, automating complex workflows, or building a custom MVP, I can help.

Services at Azlo.pro:

Custom Automation: Streamline operations and reduce manual data entry.

Backend Development: High-performance APIs in Go & Rust.

Rapid MVPs: Move from idea to product validation fast.

Contact Me: Azlo.pro

📄 License

This project is open-source and available under the MIT License.