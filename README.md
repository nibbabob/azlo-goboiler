<div align="center">

# 🚀 Azlo Go Template

### Production-Ready SaaS Starter Kit

*Built for speed, security, and scalability*

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-316192?style=for-the-badge&logo=postgresql)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-8-DC382D?style=for-the-badge&logo=redis)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)](LICENSE)

[Features](#-features) • [Quick Start](#-quick-start) • [Architecture](#-architecture) • [Documentation](#-documentation) • [Deploy](#-deployment)

---

### Free Resource by [Azlo.pro](https://www.azlo.pro/en/)

*I help businesses build custom automation workflows and high-performance backends. This template is provided free to the community to help you ship your MVP faster.*

</div>

---

## ✨ Features

### 🔐 **Security First**
- **JWT Authentication** - Secure, stateless auth with HTTP-only cookies
- **Redis-Backed Rate Limiting** - Protect against abuse with distributed rate limiting
- **Strict Security Headers** - CSP, HSTS, X-Frame-Options, and more
- **SSL/TLS Everywhere** - End-to-end encryption for all communications
- **Docker Secrets** - No plain-text credentials in containers

### 👁️ **Full Observability Stack**
Pre-configured **LGTM Stack** (Loki, Grafana, Tempo, Prometheus):
- 📊 **Metrics** - Real-time system and application metrics
- 📝 **Logs** - Centralized log aggregation and search
- 🔍 **Traces** - Distributed tracing across services
- 📈 **Dashboards** - Beautiful, pre-built Grafana dashboards

### 🐳 **Production Ready**
- **Multi-Stage Docker Builds** - Optimized images using Distroless
- **Non-Root Containers** - Security-hardened runtime environment
- **Health Checks** - Automated container health monitoring
- **Graceful Shutdown** - Clean service termination and resource cleanup

### ⚡ **High Performance**
- **Go 1.24+** - Latest Go runtime with improved performance
- **pgx Connection Pooling** - Optimized PostgreSQL connections
- **Redis Caching** - Fast, distributed caching layer
- **Efficient Resource Usage** - Minimal memory footprint

### 📱 **Modern UI**
- **Responsive Design** - Mobile-first, glassmorphism UI
- **Dark Mode** - Beautiful, easy-on-the-eyes interface
- **Vanilla JS** - No heavy frameworks, just clean JavaScript
- **RESTful Dashboard** - Complete user management interface

---

## 🛠️ Tech Stack

<table>
<tr>
<td align="center" width="25%">
<img src="https://go.dev/images/gophers/ladder.svg" width="60"><br>
<b>Backend</b><br>
Go (Golang)
</td>
<td align="center" width="25%">
<img src="https://www.postgresql.org/media/img/about/press/elephant.png" width="60"><br>
<b>Database</b><br>
PostgreSQL 18
</td>
<td align="center" width="25%">
<img src="https://logo.svgcdn.com/logos/redis.svg" width="60"><br>
<b>Cache</b><br>
Redis 8
</td>
<td align="center" width="25%">
<img src="https://logo.svgcdn.com/logos/nginx.svg" width="60"><br>
<b>Proxy</b><br>
Nginx
</td>
</tr>
<tr>
<td align="center" width="25%">
<img src="https://logo.svgcdn.com/logos/grafana.svg" width="60"><br>
<b>Visualization</b><br>
Grafana
</td>
<td align="center" width="25%">
<img src="https://logo.svgcdn.com/logos/prometheus.svg" width="60"><br>
<b>Metrics</b><br>
Prometheus
</td>
<td align="center" width="25%">
<img src="https://grafana.com/static/img/logos/logo-loki.svg" width="60"><br>
<b>Logs</b><br>
Loki
</td>
<td align="center" width="25%">
<img src="https://grafana.com/static/assets/img/logos/grafana-tempo.svg" width="60"><br>
<b>Tracing</b><br>
Tempo
</td>
</tr>
</table>

---

## 🚀 Quick Start

### Prerequisites

Ensure you have the following installed:

- **Docker** & **Docker Compose** (v2.0+)
- **Make** (optional, but recommended)
- **OpenSSL** (usually pre-installed on Linux/Mac/Git Bash)

### Installation

```bash
# 1. Clone the repository
git clone https://github.com/your-username/azlo-go-template.git
cd azlo-go-template

# 2. Configure environment
cp .env.example .env

# 3. Initialize Docker secrets
chmod +x migrate.sh
./migrate.sh

# 4. Generate SSL certificates
chmod +x scripts/generate-certs.sh
./scripts/generate-certs.sh

# 5. Start the stack
docker-compose up -d --build
```

### Access Your Application

| Service | URL | Credentials |
|---------|-----|-------------|
| 🌐 **Web App** | `https://localhost` | See below |
| 📊 **Grafana** | `http://localhost:3000` | admin / adminadmin |
| 📈 **Prometheus** | `http://localhost:9090` | - |
| 💚 **API Health** | `https://localhost/health` | - |

**Default User Credentials** (Development):
- Username: `admin`
- Password: `admin123!`

> ⚠️ **Security Note**: Change the `APP_SECRET`, `POSTGRES_PASSWORD`, and `REDIS_PASSWORD` in your `.env` file for production deployments.

---

## 🏗️ Architecture

### Project Structure

```
azlo-go-template/
├── api-service/              # Go API source code
│   ├── cmd/
│   │   ├── api/             # Main application entry point
│   │   └── healthcheck/     # Health check binary
│   ├── internal/
│   │   ├── config/          # Configuration management
│   │   ├── database/        # Database connection & migrations
│   │   ├── handlers/        # HTTP request handlers
│   │   ├── middleware/      # Auth, logging, rate limiting
│   │   ├── models/          # Data models
│   │   ├── router/          # Route definitions
│   │   ├── telemetry/       # OpenTelemetry setup
│   │   └── validation/      # Input validation
│   └── Dockerfile           # Multi-stage build definition
├── www/                      # Static frontend (HTML, JS, CSS)
├── nginx/                    # Reverse proxy configuration
├── grafana/                  # Dashboards and provisioning
├── prometheus/               # Metric scraping configuration
├── loki/                     # Log aggregation config
├── tempo/                    # Distributed tracing config
├── otel-collector/           # OpenTelemetry collector
├── certs/                    # SSL certificates (generated)
├── secrets/                  # Docker secrets (generated)
├── scripts/                  # Utility scripts
├── docker-compose.yml        # Development stack
├── docker-compose.prod.yml   # Production stack
└── migrate.sh                # Secret generation script
```

### System Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    External Traffic                      │
└────────────────────────┬────────────────────────────────┘
                         │
                    ┌────▼────┐
                    │  Nginx  │  ← SSL Termination
                    │ (Proxy) │  ← Rate Limiting
                    └────┬────┘  ← Security Headers
                         │
        ┌────────────────┼────────────────┐
        │                │                │
   ┌────▼────┐     ┌─────▼─────┐   ┌────▼────┐
   │   Go    │────▶│PostgreSQL │   │  Redis  │
   │   API   │     │    DB     │   │ Cache   │
   └────┬────┘     └───────────┘   └─────────┘
        │
        │ OpenTelemetry
        │
   ┌────▼──────────────────────────────────┐
   │         Observability Stack           │
   │  ┌──────────┐  ┌──────────────────┐  │
   │  │Prometheus│  │  Loki (Logs)     │  │
   │  │(Metrics) │  │  Tempo (Traces)  │  │
   │  └────┬─────┘  └────┬─────────────┘  │
   │       └─────────────┼─────────────┐  │
   │                ┌────▼────┐        │  │
   │                │ Grafana │◀───────┘  │
   │                └─────────┘           │
   └───────────────────────────────────────┘
```

---

## 🔧 Development Workflow

### Adding a New Route

1. **Create Handler** - Add a new function in `api-service/internal/handlers/`
2. **Register Route** - Add it to `api-service/internal/router/router.go`
3. **Test** - The API hot-reloads on restart

```bash
# Rebuild just the API service after code changes
docker-compose up -d --build api
```

### Database Migrations

The project uses `scripts/init-db-ssl.sh` for initial setup. For ongoing schema changes:

1. Add `.sql` files to the `docker-entrypoint-initdb.d` volume
2. Or integrate a migration tool like [golang-migrate](https://github.com/golang-migrate/migrate)

### Environment Variables

Key configuration options in `.env`:

```bash
# Application
APP_ENV=development           # or 'production'
APP_SECRET=your-secret-key   # Min 32 characters

# Database
POSTGRES_DB=apidb
POSTGRES_USER=apiuser
POSTGRES_PASSWORD=secure-password

# Redis
REDIS_PASSWORD=secure-password

# Monitoring
GRAFANA_PORT=3000
PROMETHEUS_PORT=9090
```

---

## 📦 Deployment

### Production Build

A production-ready `docker-compose.prod.yml` is included with:

- ✅ Stricter resource limits
- ✅ Removed exposed internal ports (Redis/Postgres)
- ✅ External secrets management for Swarm/Orchestrator
- ✅ Service replication and high availability

```bash
# Update image references in docker-compose.prod.yml
docker-compose -f docker-compose.prod.yml build

# Deploy to Docker Swarm
docker stack deploy -c docker-compose.prod.yml myapp
```

### CI/CD Pipeline

GitHub Actions workflow included (`.github/workflows/ci-cd.yml`):

- 🔄 **Automated Builds** - Triggers on push to `main` or `development`
- 📦 **Multi-Service Build** - API, Scraper, and Nginx images
- 🚀 **Container Registry** - Pushes to GitHub Container Registry
- 📝 **Automatic Updates** - Updates production config via Gist

### Security Checklist

Before going to production:

- [ ] Change all default passwords in `.env`
- [ ] Generate strong `APP_SECRET` (min 32 characters)
- [ ] Use proper SSL certificates (Let's Encrypt)
- [ ] Configure firewall rules
- [ ] Enable log aggregation
- [ ] Set up automated backups
- [ ] Configure monitoring alerts
- [ ] Review and update CORS origins
- [ ] Enable rate limiting appropriate for your traffic

---

## 📊 Monitoring & Observability

### Grafana Dashboards

Access pre-built dashboards at `http://localhost:3000`:

- **Main API Dashboard** - Request rates, error logs, traces
- **System Metrics** - CPU, memory, disk usage
- **Database Performance** - Connection pool stats, query performance
- **Redis Metrics** - Cache hit rates, memory usage

### Prometheus Metrics

Custom metrics exposed at `/metrics`:

- `http_request_duration_seconds` - Request latency histogram
- `http_requests_total` - Total HTTP requests by status code
- Database connection pool stats
- Redis operation metrics

### Distributed Tracing

OpenTelemetry traces are automatically collected:

- API request traces
- Database query spans
- Redis operation spans
- Cross-service correlation

---

## 🧪 Testing

Run the comprehensive test suite:

```bash
chmod +x test.sh
./test.sh
```

**Test Coverage:**
- ✅ Health endpoints
- ✅ Security headers
- ✅ User registration & authentication
- ✅ Protected endpoints
- ✅ Input validation
- ✅ Rate limiting
- ✅ Database connectivity
- ✅ Error handling

---

## 🤝 Need Custom Development?

This template is a great starting point, but every business has unique needs. If you need help scaling this, automating complex workflows, or building a custom MVP, I can help.

### Services at [Azlo.pro](https://www.azlo.pro/en/)

- 🤖 **Custom Automation** - Streamline operations and reduce manual data entry
- ⚡ **Backend Development** - High-performance APIs in Go & Rust
- 🚀 **Rapid MVPs** - Move from idea to product validation fast
- ☁️ **Cloud Architecture** - Scalable, resilient infrastructure design

<div align="center">

**[Contact Me](https://www.azlo.pro/en/contact)** • **[View Portfolio](https://www.azlo.pro/en/)**

</div>

---

## 📚 Additional Resources

- [Go Documentation](https://go.dev/doc/)
- [PostgreSQL Best Practices](https://www.postgresql.org/docs/)
- [Docker Security](https://docs.docker.com/engine/security/)
- [Prometheus Query Examples](https://prometheus.io/docs/prometheus/latest/querying/examples/)
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/instrumentation/go/)

---

## 📄 License

This project is open-source and available under the **MIT License**.

See the [LICENSE](LICENSE) file for more details.

---

<div align="center">

### ⭐ Star this repo if you find it helpful!

**Built with ❤️ by [Azlo.pro](https://www.azlo.pro/en/)**

*Helping businesses ship faster with production-ready code*

</div>