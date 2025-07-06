# 🚀 Go API Boilerplate - Production Ready

A comprehensive, production-ready Go REST API boilerplate with security best practices, middleware, and modern tooling. Perfect for kickstarting your next API project.

## ✨ Features

### 🔒 Security First
- **HTTPS/TLS Support** - Production-ready TLS configuration
- **Security Headers** - OWASP recommended security headers
- **CORS Protection** - Configurable cross-origin resource sharing
- **Rate Limiting** - Per-IP rate limiting with configurable limits
- **JWT Authentication** - Token-based authentication middleware
- **Input Validation** - Request validation and sanitization

### 🛠️ Production Ready
- **Graceful Shutdown** - Proper server lifecycle management
- **Health Checks** - Kubernetes-ready health endpoints
- **Request Logging** - Structured HTTP request logging
- **Error Recovery** - Panic recovery middleware
- **Configuration Management** - Environment-based configuration
- **Docker Support** - Container-ready with multi-stage builds

### 🏗️ Developer Experience
- **Clean Architecture** - Well-organized, modular codebase
- **Middleware Chain** - Composable middleware system
- **JSON Responses** - Standardized API response format
- **Hot Reload** - Development with live reload (Air)
- **Testing Setup** - Unit and integration test examples

## 🚀 Quick Start

### Prerequisites
- Go 1.19 or later
- Optional: Docker for containerization

### Installation

1. **Clone or download this boilerplate:**
```bash
git clone <your-repo-url>
cd go-api-boilerplate
```

2. **Install dependencies:**
```bash
go mod init your-project-name
go mod tidy
```

3. **Set up environment variables:**
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. **Run the server:**
```bash
go run main.go
```

The API will be available at `http://localhost:8080`

## 📋 API Endpoints

### Public Endpoints
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check endpoint |
| POST | `/auth` | User authentication |

### Protected Endpoints (Require JWT)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/users` | Get all users |
| POST | `/api/v1/users` | Create a new user |
| GET | `/api/v1/protected` | Example protected endpoint |

## 🔧 Configuration

### Environment Variables

Create a `.env` file in the root directory:

```env
# Server Configuration
PORT=8080
ENVIRONMENT=development

# Database
DATABASE_URL=postgres://user:password@localhost/dbname?sslmode=disable

# Security
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
ENABLE_HTTPS=false
TLS_CERT_PATH=/path/to/cert.pem
TLS_KEY_PATH=/path/to/key.pem

# Rate Limiting
RATE_LIMIT=100
```

### HTTPS Configuration

For production HTTPS:

1. **Obtain SSL certificates** (Let's Encrypt, AWS Certificate Manager, etc.)
2. **Set environment variables:**
```env
ENABLE_HTTPS=true
TLS_CERT_PATH=/etc/ssl/certs/your-cert.pem
TLS_KEY_PATH=/etc/ssl/private/your-key.pem
```

3. **Update security headers** for your domain in the code

## 🐳 Docker Support

### Dockerfile
```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/.env .

EXPOSE 8080
CMD ["./main"]
```

### Docker Compose
```yaml
version: '3.8'

services:
  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - ENVIRONMENT=production
      - DATABASE_URL=postgres://user:password@db:5432/apidb
    depends_on:
      - db
    restart: unless-stopped

  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: apidb
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    restart: unless-stopped

volumes:
  postgres_data:
```

### Build and Run with Docker
```bash
# Build the image
docker build -t go-api-boilerplate .

# Run with docker-compose
docker-compose up -d

# Or run standalone
docker run -p 8080:8080 go-api-boilerplate
```

## 🧪 Testing

### Example Test Structure
```bash
mkdir -p tests/{unit,integration}
```

### Sample Unit Test
```go
// tests/unit/handlers_test.go
package tests

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestHealthHandler(t *testing.T) {
    req, err := http.NewRequest("GET", "/health", nil)
    if err != nil {
        t.Fatal(err)
    }

    rr := httptest.NewRecorder()
    handler := http.HandlerFunc(HealthHandler)
    handler.ServeHTTP(rr, req)

    if status := rr.Code; status != http.StatusOK {
        t.Errorf("handler returned wrong status code: got %v want %v",
            status, http.StatusOK)
    }
}
```

### Run Tests
```bash
go test ./...
go test -v ./tests/...
go test -cover ./...
```

## 🔄 Development Workflow

### Hot Reload with Air

1. **Install Air:**
```bash
go install github.com/cosmtrek/air@latest
```

2. **Create `.air.toml`:**
```toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ."
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  kill_delay = "0s"
  log = "build-errors.log"
  send_interrupt = false
  stop_on_root = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  time = false

[misc]
  clean_on_exit = false
```

3. **Run with hot reload:**
```bash
air
```

## 🚦 API Usage Examples

### Authentication
```bash
# Login
curl -X POST http://localhost:8080/auth \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "password"}'

# Response
{
  "success": true,
  "data": {
    "token": "mock-jwt-token"
  },
  "message": "Authentication successful"
}
```

### Using Protected Endpoints
```bash
# Get users (requires authentication)
curl -X GET http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer mock-jwt-token"

# Create user
curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer mock-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{"name": "John Doe", "email": "john@example.com"}'
```

### Health Check
```bash
curl http://localhost:8080/health

# Response
{
  "success": true,
  "data": {
    "status": "healthy",
    "timestamp": "2024-01-15T10:30:00Z",
    "version": "1.0.0",
    "uptime": "2h30m15s"
  },
  "message": "Service is healthy"
}
```

## 📚 Architecture Overview

```
├── main.go                 # Application entry point
├── config.go              # Configuration management
├── middleware/            # Custom middleware
├── handlers/              # HTTP handlers
├── models/               # Data models
├── services/             # Business logic
├── utils/                # Utility functions
├── tests/                # Test suites
├── docker/               # Docker configuration
├── scripts/              # Build and deployment scripts
└── docs/                 # API documentation
```

## 🔒 Security Checklist

- [ ] Change default JWT secret
- [ ] Enable HTTPS in production
- [ ] Configure proper CORS origins
- [ ] Set appropriate rate limits
- [ ] Implement proper input validation
- [ ] Add request size limits
- [ ] Configure security headers
- [ ] Set up monitoring and logging
- [ ] Implement proper error handling
- [ ] Use environment variables for secrets

## 🚀 Production Deployment

### Environment Setup
1. **Set production environment variables**
2. **Configure HTTPS certificates**
3. **Set up database connections**
4. **Configure reverse proxy (Nginx/Apache)**
5. **Set up monitoring (Prometheus/Grafana)**
6. **Configure log aggregation**

### Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: go-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: go-api
  template:
    metadata:
      labels:
        app: go-api
    spec:
      containers:
      - name: go-api
        image: your-registry/go-api:latest
        ports:
        - containerPort: 8080
        env:
        - name: PORT
          value: "8080"
        - name: ENVIRONMENT
          value: "production"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

## 📖 Additional Resources

- [Go Best Practices](https://golang.org/doc/effective_go.html)
- [REST API Design Guidelines](https://restfulapi.net/)
- [OWASP Security Guidelines](https://owasp.org/www-project-api-security/)
- [Twelve-Factor App](https://12factor.net/)

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 💡 Need More Features?

This boilerplate provides a solid foundation for most API projects. For enterprise features like:

- **Advanced Authentication (OAuth2, SAML)**
- **Database ORM Integration (GORM, SQLBoiler)**
- **Message Queue Integration (Redis, RabbitMQ)**
- **Microservices Architecture**
- **API Gateway Integration**
- **Advanced Monitoring & Observability**

Visit [YourWebsite.com](https://yourwebsite.com) for premium templates and consulting services.

---

**Built with ❤️ by [Your Company Name]** | [Website](https://yourwebsite.com) | [Documentation](https://docs.yourwebsite.com) | [Support](mailto:support@yourwebsite.com)