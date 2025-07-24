# Claude Project Context

## Project Overview
This project adapts the Zep v0.27 web interface to work with Zep v1.0.2 by creating a standalone service that uses the original templates but calls the v1.0.2 HTTP API endpoints.

## Background
- **Zep v0.27** had a built-in web interface with HTML templates, CSS, and JavaScript
- **Zep v1.0.2** removed the web interface, leaving only JSON API endpoints  
- **This project** restores the familiar web UI by creating a separate Go service

## Architecture
- **Go service** with Chi router for HTTP handling
- **Template system** using Go's html/template with v0.27 templates
- **API client layer** that converts internal calls to HTTP API calls
- **Static file serving** for CSS, JavaScript, and images
- **Environment-based configuration** with comprehensive validation

## Key Components

### Server Layer (`internal/server/`)
- HTTP server setup with Chi router
- Middleware for logging, recovery, CORS, proxy support
- Route definitions matching v0.27 structure
- Template loading and rendering

### API Client (`internal/zepapi/`)
- HTTP client for Zep v1.0.2 API communication
- Bearer token authentication
- Proxy support for network configurations
- Data models matching v1.0.2 API responses

### Configuration (`internal/config/`)
- Environment variable-based configuration
- IPv6 support by default
- Comprehensive validation for all settings
- Required variables: ZEP_API_URL, ZEP_API_KEY

### Handlers (`internal/handlers/`)
- Route handlers for all web interface pages
- Dashboard, sessions, users, settings pages
- HTMX API endpoints for dynamic content

### Web Assets (`web/`)
- HTML templates from v0.27 (layouts, pages, components)
- CSS files (TailwindCSS compiled)
- JavaScript files (HTMX, Alpine.js, Preline components)
- Static assets (favicon, images)

## Environment Configuration
All configuration is done via environment variables with validation:

### Required
- `ZEP_API_URL` - Zep v1.0.2 server URL
- `ZEP_API_KEY` - API authentication key

### Optional
- `HOST` - Server host (default: `::` for IPv6)
- `PORT` - Server port (default: 8080)
- `PROXY_URL` - HTTP proxy for API requests
- `TRUST_PROXY` - Trust proxy headers (default: true)
- `CORS_ORIGINS` - Allowed origins (default: `*`)
- `TLS_ENABLED` - Enable HTTPS (default: false)

## Development Commands
```bash
# Install dependencies
go mod tidy

# Run in development
go run main.go

# Build binary
go build -o zep-web-interface ./main.go

# Build with Docker
docker build -t zep-web-interface .
```

## Deployment
- **Railway**: Use service variables and internal networking
- **Docker**: Use environment variables and container networking
- **Heroku**: Configure dyno with environment variables
- **Local**: Set environment variables and run binary

## Testing
- Set `ZEP_API_URL` to your Zep v1.0.2 instance
- Set `ZEP_API_KEY` to a valid API key
- Run the service and access http://localhost:8080/admin

## Security Considerations
- All API calls use Bearer token authentication
- CORS is configurable for production deployments
- No hardcoded credentials or URLs
- Proxy support for secure network configurations
- Input validation for all environment variables