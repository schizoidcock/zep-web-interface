# Contributing Guide

## Overview
Thank you for your interest in contributing to the Zep Web Interface! This guide will help you understand the codebase, development workflow, and contribution process.

## Development Setup

### Prerequisites
- Go 1.21 or later
- Git
- Access to a Zep v1.0.2 server for testing
- Basic knowledge of Go, HTML templates, and HTMX

### Local Development Setup
```bash
# Clone the repository
git clone https://github.com/schizoidcock/zep-web-interface.git
cd zep-web-interface

# Install dependencies
go mod tidy

# Copy environment template
cp .env.example .env

# Configure environment variables
# Edit .env with your Zep server details
ZEP_API_URL=http://localhost:8000
ZEP_API_KEY=your-dev-api-key

# Run the development server
go run main.go
```

### Testing Your Setup
1. Start your Zep v1.0.2 server
2. Configure `.env` with correct API URL and key
3. Run the web interface: `go run main.go`
4. Visit `http://localhost:8080/admin`
5. Verify you can see sessions and users

## Project Structure

```
zep-web-interface/
├── main.go                 # Application entry point
├── internal/               # Private application code
│   ├── config/            # Configuration management
│   ├── server/            # HTTP server setup
│   ├── handlers/          # Route handlers
│   └── zepapi/            # Zep API client
├── web/                   # Web assets
│   ├── static/            # CSS, JS, images
│   └── templates/         # HTML templates
├── .env.example           # Environment template
├── Dockerfile             # Container configuration
└── docs/                  # Documentation
```

### Key Files to Understand
- `internal/server/server.go` - HTTP server and routing
- `internal/handlers/handlers.go` - Web request handlers
- `internal/zepapi/client.go` - Zep API communication
- `internal/config/config.go` - Configuration and validation
- `web/templates/` - HTML templates using Go template syntax

## Development Workflow

### Making Changes

1. **Create a Feature Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make Your Changes**
   - Follow Go conventions and best practices
   - Update tests if applicable
   - Update documentation if needed

3. **Test Your Changes**
   ```bash
   # Build and test
   go build -o zep-web-interface ./main.go
   ./zep-web-interface
   
   # Test with different configurations
   ZEP_API_URL=http://test-server:8000 ./zep-web-interface
   ```

4. **Commit Your Changes**
   ```bash
   git add .
   git commit -m "Add feature: description of your changes"
   ```

5. **Push and Create PR**
   ```bash
   git push origin feature/your-feature-name
   # Create pull request on GitHub
   ```

### Code Style Guidelines

#### Go Code Style
- Follow standard Go formatting (`go fmt`)
- Use meaningful variable and function names
- Add comments for exported functions
- Handle errors properly
- Use structured logging

Example:
```go
// GetSession retrieves a session by ID from the Zep API
func (c *Client) GetSession(sessionID string) (*Session, error) {
    if sessionID == "" {
        return nil, fmt.Errorf("session ID cannot be empty")
    }
    
    resp, err := c.get("/api/v1/sessions/" + sessionID)
    if err != nil {
        return nil, fmt.Errorf("failed to get session: %w", err)
    }
    
    var session Session
    if err := decodeResponse(resp, &session); err != nil {
        return nil, fmt.Errorf("failed to decode session: %w", err)
    }
    
    return &session, nil
}
```

#### HTML Template Style
- Use semantic HTML elements
- Follow existing template patterns
- Use consistent indentation (2 spaces)
- Include proper template comments

Example:
```html
{{ define "SessionDetails" }}
<div class="session-details">
    <!-- Session information -->
    <h2>Session: {{ .Session.SessionID }}</h2>
    
    <!-- Session metadata -->
    <div class="metadata">
        <span>Created: {{ formatTime .Session.CreatedAt }}</span>
        <span>Updated: {{ formatTime .Session.UpdatedAt }}</span>
    </div>
    
    <!-- Session content -->
    {{ if .Session.Summary }}
        <div class="summary">
            <h3>Summary</h3>
            <pre>{{ .Session.Summary }}</pre>
        </div>
    {{ end }}
</div>
{{ end }}
```

## Types of Contributions

### Bug Fixes
- Fix broken functionality
- Improve error handling
- Resolve security issues
- Performance improvements

### New Features
- Additional Zep API endpoints
- New UI components
- Enhanced navigation
- Search and filtering capabilities

### Documentation
- Improve README
- Add code comments
- Update API documentation
- Create tutorials

### Infrastructure
- Docker improvements
- CI/CD enhancements
- Deployment guides
- Testing infrastructure

## Common Development Tasks

### Adding a New Page
1. Create template in `web/templates/pages/`
2. Add route in `internal/server/server.go`
3. Create handler in `internal/handlers/handlers.go`
4. Add navigation link if needed

Example:
```go
// In internal/server/server.go
r.Get("/admin/analytics", h.Analytics)

// In internal/handlers/handlers.go
func (h *Handlers) Analytics(w http.ResponseWriter, r *http.Request) {
    data := map[string]interface{}{
        "Title": "Analytics",
        "Page":  "analytics",
    }
    
    if err := h.templates.ExecuteTemplate(w, "Layout", data); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}
```

### Adding New Zep API Integration
1. Add data model to `internal/zepapi/client.go`
2. Add API method to client
3. Create handler to use the API
4. Add template to display data

### Adding Environment Variables
1. Add field to `Config` struct in `internal/config/config.go`
2. Add to `Load()` function with default
3. Add validation in `validate()` method
4. Update `.env.example` and documentation

## Testing

### Manual Testing
- Test all pages load correctly
- Verify API integration works
- Test different screen sizes
- Test with different Zep server versions

### Automated Testing (Future)
Currently no automated tests exist. Consider adding:
- Unit tests for API client
- Handler tests with mock API
- Template rendering tests
- Integration tests

### Testing Checklist
- [ ] Application starts without errors
- [ ] Health check endpoint works
- [ ] All pages render correctly
- [ ] API calls succeed with valid data
- [ ] Error handling works properly
- [ ] Environment validation works
- [ ] Docker build succeeds

## Debugging

### Common Issues

#### Template Errors
```bash
# Check template syntax
go run main.go
# Look for template parsing errors in logs
```

#### API Connection Issues
```bash
# Test API connectivity
curl -H "Authorization: Bearer $ZEP_API_KEY" $ZEP_API_URL/api/v1/sessions

# Check logs for API errors
go run main.go
# Make requests and check server logs
```

#### Configuration Issues
```bash
# Validate environment variables
go run main.go
# Check for configuration validation errors
```

### Logging
Add debug logging to understand issues:
```go
import "log"

log.Printf("Making API request to: %s", url)
log.Printf("Response status: %d", resp.StatusCode)
```

## Pull Request Guidelines

### Before Submitting
- [ ] Code builds without errors
- [ ] All pages still work
- [ ] No hardcoded values added
- [ ] Documentation updated if needed
- [ ] Commit message is descriptive

### PR Description Template
```markdown
## Description
Brief description of changes made.

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Documentation update
- [ ] Refactoring

## Testing
- [ ] Tested locally
- [ ] Tested with production-like setup
- [ ] All existing functionality still works

## Screenshots (if applicable)
Include screenshots of UI changes.

## Additional Notes
Any additional context or considerations.
```

### Review Process
1. Automated checks (if available)
2. Code review for style and correctness
3. Testing by maintainers
4. Merge after approval

## Release Process

### Version Management
- Follow semantic versioning (MAJOR.MINOR.PATCH)
- Tag releases in Git
- Update CHANGELOG.md

### Release Checklist
- [ ] All tests pass
- [ ] Documentation updated
- [ ] Docker image builds
- [ ] Deployment tested
- [ ] CHANGELOG.md updated
- [ ] Git tag created

## Getting Help

### Resources
- [Go Documentation](https://golang.org/doc/)
- [Chi Router](https://github.com/go-chi/chi)
- [HTMX Documentation](https://htmx.org/docs/)
- [TailwindCSS](https://tailwindcss.com/docs)

### Communication
- Create GitHub issues for bugs or features
- Use discussions for questions
- Tag maintainers for urgent issues

### Code Review Feedback
- Be constructive and specific
- Suggest improvements with examples
- Ask questions to understand the approach
- Appreciate the contribution

Thank you for contributing to the Zep Web Interface!