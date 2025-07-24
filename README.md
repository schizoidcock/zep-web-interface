# Zep Web Interface

A standalone web interface for Zep v1.0.2, adapted from the v0.27 built-in web interface.

## Overview

Zep v1.0.2 removed the built-in web interface that existed in v0.27, leaving only JSON API endpoints. This project restores the familiar web UI functionality by creating a separate service that uses the v0.27 templates but calls the v1.0.2 API endpoints.

## Features

- Dashboard with quick links and overview
- Session management and viewing
- User management and viewing  
- HTMX-powered dynamic content loading
- TailwindCSS styling with dark mode support
- Responsive design for desktop and mobile

## Configuration

Set these environment variables:

```bash
# Required
ZEP_API_URL=http://localhost:8000    # Your Zep v1.0.2 server URL
ZEP_API_KEY=your-api-key             # Your Zep API key

# Optional
HOST=localhost                       # Web interface host (default: localhost)
PORT=8080                           # Web interface port (default: 8080)
```

## Running

### Development
```bash
go run main.go
```

### Production
```bash
go build -o zep-web-interface
./zep-web-interface
```

The web interface will be available at http://localhost:8080/admin

## API Endpoints

The web interface provides these routes:

- `GET /admin/` - Dashboard
- `GET /admin/sessions` - Sessions list
- `GET /admin/sessions/{sessionId}` - Session details
- `GET /admin/users` - Users list  
- `GET /admin/users/{userId}` - User details
- `GET /admin/users/{userId}/sessions` - User sessions
- `GET /admin/settings` - Settings page

## Architecture

- **Go server** with Chi router
- **HTMX** for dynamic content loading
- **Alpine.js** for interactive components
- **TailwindCSS** for styling
- **Bearer token authentication** for Zep API access
- **Template-driven** architecture with component reuse

## Deployment

This service can be deployed anywhere that can run Go applications:

- Docker containers
- Railway, Heroku, or similar PaaS
- Traditional servers
- Kubernetes clusters

Just ensure the `ZEP_API_URL` points to your Zep v1.0.2 instance and provide a valid API key.