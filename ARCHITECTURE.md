# Architecture Documentation

## System Overview

The Zep Web Interface is a standalone Go service that provides a web UI for Zep v1.0.2 servers. It acts as a bridge between the familiar v0.27 web interface and the v1.0.2 HTTP API.

```
┌─────────────────┐    HTTP/HTTPS    ┌──────────────────┐    HTTP API    ┌─────────────────┐
│   Web Browser   │ ────────────────► │  Zep Web UI      │ ──────────────► │   Zep v1.0.2    │
│                 │                   │  (This Service)  │                 │   Server        │
└─────────────────┘                   └──────────────────┘                 └─────────────────┘
```

## Component Architecture

### 1. HTTP Server Layer
**Location**: `internal/server/server.go`

- **Chi Router**: Fast HTTP router with middleware support
- **Middleware Stack**: 
  - Logger for request logging
  - Recoverer for panic recovery
  - Timeout for request timeouts
  - RealIP for proxy header handling
  - CORS for cross-origin requests
- **Static File Serving**: Serves CSS, JS, and images from `web/static/`
- **Route Handling**: Maps URLs to handler functions

### 2. Configuration Management
**Location**: `internal/config/config.go`

- **Environment Variables**: All configuration via env vars
- **Validation**: Comprehensive validation of all settings
- **IPv6 Support**: Modern networking with IPv6 defaults
- **Proxy Configuration**: Support for HTTP proxies
- **Security**: No hardcoded values, requires explicit configuration

### 3. API Client Layer
**Location**: `internal/zepapi/client.go`

- **HTTP Client**: Configured with timeouts and proxy support
- **Authentication**: Bearer token authentication
- **Data Models**: Structs matching Zep v1.0.2 API responses
- **Error Handling**: Proper HTTP error handling and propagation

### 4. Request Handlers
**Location**: `internal/handlers/handlers.go`

- **Page Handlers**: Render full HTML pages with templates
- **API Handlers**: Return partial HTML for HTMX requests
- **Data Fetching**: Retrieve data from Zep API
- **Template Rendering**: Use Go's html/template system

### 5. Template System
**Location**: `web/templates/`

- **Layouts**: Base page structure (`web/templates/layouts/`)
- **Pages**: Individual page content (`web/templates/pages/`)
- **Components**: Reusable UI components (`web/templates/components/`)
- **Template Functions**: Helper functions for formatting and display

## Data Flow

### 1. Web Request Flow
```
Browser Request → Chi Router → Handler → Zep API → Response → Template → HTML Response
```

### 2. API Request Flow (HTMX)
```
HTMX Request → Chi Router → API Handler → Zep API → Response → Template Fragment → HTML Fragment
```

### 3. Static Asset Flow
```
Browser Request → Chi Router → File Server → Static File → Direct Response
```

## Network Architecture

### Development Setup
```
Browser (localhost:8080) → Zep Web UI → Zep Server (localhost:8000)
```

### Production Setup (Railway/Heroku)
```
Internet → Load Balancer → Zep Web UI → Internal Network → Zep Server
                         (with proxy headers)
```

### Docker Setup
```
Host Network → Docker Container (Zep Web UI) → Container Network → Zep Server Container
```

## Security Architecture

### 1. Authentication
- **API Authentication**: Bearer token for all Zep API calls
- **No User Authentication**: Web interface relies on network security

### 2. Network Security
- **CORS Configuration**: Configurable allowed origins
- **Proxy Trust**: Configurable proxy header trust
- **TLS Support**: Optional HTTPS configuration

### 3. Input Validation
- **Environment Variables**: All config validated at startup
- **URL Validation**: API URLs and proxy URLs validated
- **Port Validation**: Network ports validated for valid ranges

## Template Architecture

### Template Hierarchy
```
Layout (base HTML structure)
├── Header (navigation, branding)
├── Sidebar (navigation menu)
├── Content (page-specific content)
│   ├── Breadcrumbs
│   ├── Page Titles
│   └── Page Components
└── Scripts (JavaScript includes)
```

### Component System
- **Layout Components**: Header, sidebar, navigation
- **Content Components**: Tables, forms, modals
- **Utility Components**: Breadcrumbs, titles, tooltips

## Technology Stack

### Backend
- **Go 1.21+**: Main programming language
- **Chi Router**: HTTP routing and middleware
- **html/template**: Template rendering
- **net/http**: HTTP client for API calls

### Frontend
- **HTMX**: Dynamic content loading
- **Alpine.js**: Reactive components
- **TailwindCSS**: Utility-first CSS framework
- **Preline**: UI component library

### Infrastructure
- **Docker**: Containerization
- **Railway/Heroku**: PaaS deployment
- **IPv6**: Modern networking support

## Performance Considerations

### Template Caching
- Templates loaded once at startup
- No runtime template parsing
- Efficient template execution

### HTTP Client
- Connection reuse for API calls
- Configurable timeouts
- Proxy support for network optimization

### Static Assets
- Direct file serving
- Browser caching headers
- Compressed assets (CSS minification)

## Monitoring and Observability

### Logging
- Request logging via Chi middleware
- Error logging for API failures
- Configuration validation logging

### Health Checks
- `/health` endpoint for load balancer checks
- JSON response with service status
- API connectivity validation (future enhancement)

## Deployment Patterns

### Standalone Deployment
- Single binary with embedded templates
- Environment variable configuration
- Direct network access to Zep server

### Container Deployment
- Docker container with Alpine Linux base
- Volume mounts for configuration
- Network bridge to Zep server container

### PaaS Deployment
- Railway/Heroku compatible
- Service discovery via environment variables
- Automatic scaling and load balancing