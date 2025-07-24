# Changelog

All notable changes to the Zep Web Interface project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive project documentation (CLAUDE.md, ARCHITECTURE.md, DEPLOYMENT.md, API.md, CONTRIBUTING.md)

## [0.2.0] - 2024-01-XX

### Added
- Comprehensive environment variable configuration and validation
- IPv6 support with `::` as default host
- Proxy configuration support via `PROXY_URL` environment variable
- Configurable CORS origins via `CORS_ORIGINS` environment variable
- TLS configuration option via `TLS_ENABLED` environment variable
- Trust proxy headers configuration via `TRUST_PROXY` environment variable
- Environment variable validation at startup
- `.env.example` template file for easy configuration
- Dockerfile for containerized deployment
- Configuration validation for all environment variables
- URL validation for API endpoints and proxy URLs
- Port range validation (1-65535)
- CORS origin validation

### Changed
- **BREAKING**: Removed hardcoded default URLs - `ZEP_API_URL` and `ZEP_API_KEY` are now required
- **BREAKING**: Default host changed from `0.0.0.0` to `::` for IPv6 support
- Default port changed from 8082 to 8080 for consistency
- Enhanced error messages for configuration validation failures
- Improved CORS configuration with proxy header support
- API client now supports HTTP proxy configuration

### Security
- Removed all hardcoded credentials and URLs
- Added comprehensive input validation for all configuration
- Enhanced security headers for proxy deployments

## [0.1.0] - 2024-01-XX

### Added
- Initial implementation of Zep v1.0.2 web interface
- Complete Go web service with Chi router
- HTML templates adapted from Zep v0.27
- API client layer for Zep v1.0.2 HTTP endpoints
- Bearer token authentication for API access
- Session management interface
  - Session list view with pagination support
  - Session details view
  - Session search and filtering capabilities
- User management interface
  - User list view with pagination support
  - User details view
  - User sessions view
- Dashboard with quick links and overview
- Settings page for configuration
- HTMX integration for dynamic content loading
- Alpine.js for interactive components
- TailwindCSS styling with dark mode support
- Responsive design for desktop and mobile
- Static file serving for CSS, JavaScript, and images
- Health check endpoint (`/health`) for load balancer integration
- Graceful shutdown handling
- Template system with helper functions
  - `formatTime()` for timestamp formatting
  - `truncate()` for text truncation
- Error handling and logging
- CORS support for cross-origin requests
- Request timeout middleware
- Recovery middleware for panic handling

### Technical Implementation
- Go 1.21+ compatibility
- Chi router v5 for HTTP routing
- HTML template rendering with component system
- HTTP client with timeout configuration
- Environment-based configuration management
- Structured logging for debugging
- Template caching for performance

### Web Assets
- Compiled TailwindCSS stylesheets
- HTMX 1.9+ for dynamic content
- Alpine.js 3.13+ for reactivity
- Preline UI components
- Dark mode support with system preference detection
- Mobile-responsive navigation
- Accessible UI components

### API Integration
- Full Zep v1.0.2 API compatibility
- Session management endpoints
- User management endpoints
- Bearer token authentication
- Error handling for API failures
- JSON response parsing
- HTTP status code handling

### Deployment Support
- Railway deployment compatibility
- Heroku deployment compatibility
- Docker containerization
- Environment variable configuration
- Health check endpoint for load balancers
- Graceful shutdown for container orchestration

### Documentation
- Comprehensive README with setup instructions
- Configuration examples for different deployment scenarios
- API endpoint documentation
- Architecture overview

---

## Development Notes

### Version 0.1.0 - Initial Release
This version successfully adapts the Zep v0.27 web interface to work with Zep v1.0.2 servers. The main challenge was converting the internal database calls used in v0.27 to HTTP API calls for v1.0.2, while maintaining the same user experience and functionality.

Key architectural decisions:
- Standalone Go service rather than embedded web interface
- Template-first approach to maintain familiar UI
- API adapter pattern for clean separation of concerns
- Environment-based configuration for deployment flexibility

### Version 0.2.0 - Configuration Enhancement
This version focuses on production readiness and deployment flexibility. Major improvements include comprehensive environment variable validation, IPv6 support, and proxy configuration for modern deployment scenarios.

Breaking changes were necessary to remove hardcoded values and ensure secure deployments, but the migration path is straightforward with the provided `.env.example` template.

---

### Migration Guide

#### From 0.1.0 to 0.2.0

**Required Changes:**
1. Set required environment variables:
   ```bash
   ZEP_API_URL=http://your-zep-server:8000
   ZEP_API_KEY=your-api-key
   ```

2. Update host configuration for IPv6 (optional):
   ```bash
   HOST=::  # IPv6 (new default)
   # or
   HOST=0.0.0.0  # IPv4 (previous default)
   ```

3. Review port configuration:
   ```bash
   PORT=8080  # New default (was 8082)
   ```

**Optional Enhancements:**
```bash
# Proxy support
PROXY_URL=http://proxy-server:8080

# Custom CORS origins
CORS_ORIGINS=https://your-domain.com,https://admin.your-domain.com

# Production security
TRUST_PROXY=true
TLS_ENABLED=true
```

The service will validate all configuration at startup and provide clear error messages for any issues.