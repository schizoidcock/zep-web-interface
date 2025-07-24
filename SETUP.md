# Zep Web Interface Setup Guide

## Prerequisites

Before running the web interface, you need:

### 1. PostgreSQL Database
Set up a PostgreSQL database for Zep:

```bash
# Using Docker (recommended for development)
docker run -d \
  --name zep-postgres \
  -e POSTGRES_USER=zep \
  -e POSTGRES_PASSWORD=zep \
  -e POSTGRES_DB=zep \
  -p 5432:5432 \
  postgres:15

# Or use an existing PostgreSQL instance
```

### 2. Zep v1.0.2 Server
Run the Zep v1.0.2 server connected to PostgreSQL:

```bash
# Using Docker (recommended)
docker run -d \
  --name zep-server \
  -p 8000:8000 \
  -e ZEP_STORE_TYPE=postgres \
  -e ZEP_STORE_POSTGRES_DSN="postgres://zep:zep@host.docker.internal:5432/zep?sslmode=disable" \
  -e ZEP_AUTH_SECRET="your-secret-key" \
  ghcr.io/getzep/zep:v1.0.2

# Or build from source following Zep documentation
```

### 3. Environment Variables
Set these required environment variables for the web interface:

```bash
# Required - Point to your Zep v1.0.2 server
export ZEP_API_URL="http://localhost:8000"

# Required - API key from your Zep server
export ZEP_API_KEY="your-api-key-here"

# Optional - Web interface configuration
export HOST="localhost"     # Default: :: (IPv6)
export PORT="8080"          # Default: 8080
export PROXY_PATH=""        # For reverse proxy deployments
export TRUST_PROXY="false"  # Default: true (for Railway/Heroku)
```

## Quick Start

### 1. Start Dependencies
```bash
# Start PostgreSQL
docker run -d --name zep-postgres \
  -e POSTGRES_USER=zep \
  -e POSTGRES_PASSWORD=zep \
  -e POSTGRES_DB=zep \
  -p 5432:5432 postgres:15

# Start Zep Server
docker run -d --name zep-server \
  -p 8000:8000 \
  -e ZEP_STORE_TYPE=postgres \
  -e ZEP_STORE_POSTGRES_DSN="postgres://zep:zep@host.docker.internal:5432/zep?sslmode=disable" \
  -e ZEP_AUTH_SECRET="your-secret-key" \
  ghcr.io/getzep/zep:v1.0.2
```

### 2. Configure Web Interface
```bash
export ZEP_API_URL="http://localhost:8000"
export ZEP_API_KEY="your-api-key"  # Get this from Zep server setup
```

### 3. Run Web Interface
```bash
# Test API connectivity first
go run test_api.go

# Start web interface
./zep-web-interface.exe

# Access at: http://localhost:8080/admin
```

## Production Deployment

### Railway Deployment

1. **Deploy Zep Server first**:
   ```yaml
   # railway.toml for Zep server
   [services.zep-server]
   image = "ghcr.io/getzep/zep:v1.0.2"
   
   [services.zep-server.env]
   ZEP_STORE_TYPE = "postgres"
   ZEP_STORE_POSTGRES_DSN = "${{Postgres.DATABASE_URL}}"
   ZEP_AUTH_SECRET = "${{secrets.ZEP_AUTH_SECRET}}"
   ```

2. **Deploy Web Interface**:
   ```yaml
   # railway.toml for web interface
   [services.zep-web]
   build.cmd = "go build -o zep-web-interface ./main.go"
   start.cmd = "./zep-web-interface"
   
   [services.zep-web.env]
   ZEP_API_URL = "${{services.zep-server.url}}"
   ZEP_API_KEY = "${{secrets.ZEP_API_KEY}}"
   HOST = "0.0.0.0"
   PORT = "${{PORT}}"
   TRUST_PROXY = "true"
   ```

### Docker Compose
```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_USER: zep
      POSTGRES_PASSWORD: zep
      POSTGRES_DB: zep
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  zep-server:
    image: ghcr.io/getzep/zep:v1.0.2
    depends_on:
      - postgres
    environment:
      ZEP_STORE_TYPE: postgres
      ZEP_STORE_POSTGRES_DSN: "postgres://zep:zep@postgres:5432/zep?sslmode=disable"
      ZEP_AUTH_SECRET: "your-secret-key"
    ports:
      - "8000:8000"

  zep-web:
    build: .
    depends_on:
      - zep-server
    environment:
      ZEP_API_URL: http://zep-server:8000
      ZEP_API_KEY: your-api-key
      HOST: 0.0.0.0
      PORT: 8080
    ports:
      - "8080:8080"

volumes:
  postgres_data:
```

## Troubleshooting

### Connection Issues
- **"Connection refused"**: Verify Zep server is running and accessible
- **"Database connection failed"**: Check PostgreSQL is running and accessible to Zep server
- **"Invalid API key"**: Verify API key matches what's configured in Zep server

### Template/Asset Issues  
- **"Template not found"**: Ensure all template files are present in `web/templates/`
- **"Assets not loading"**: Check static files are in `web/static/` and server has read access
- **"len error"**: Fixed with `safeLen` template function for empty data

### API Issues
- **"Unauthorized"**: Check API key is valid and has proper permissions
- **"Not found"**: Verify API endpoints match Zep v1.0.2 API specification

## API Key Setup

To get an API key for Zep v1.0.2:

1. **Check Zep server logs** for auto-generated key on first run
2. **Use Zep CLI** to create keys: `zep auth create-key`
3. **Environment variable**: Some deployments auto-create keys

## Data Flow

```
User Browser → Web Interface → Zep v1.0.2 API → PostgreSQL
     ↑              ↑                ↑               ↑
  HTML/CSS/JS   Go Templates    JSON REST API    Database
```

The web interface is stateless - all data comes from Zep server which manages the PostgreSQL database.