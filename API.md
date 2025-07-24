# API Documentation

## Overview
The Zep Web Interface provides both web pages and HTMX API endpoints for dynamic content loading. This document describes all available endpoints and their functionality.

## Web Interface Endpoints

### Dashboard
- **URL**: `/admin/`
- **Method**: `GET`
- **Description**: Main dashboard with overview cards and quick links
- **Template**: `dashboard.html`
- **Data**: Static content with links to Discord, documentation, and GitHub

### Sessions Management

#### Session List
- **URL**: `/admin/sessions`
- **Method**: `GET`
- **Description**: Displays paginated list of all sessions
- **Template**: `sessions.html`
- **API Call**: `GET /api/v1/sessions`
- **Data**: 
  ```json
  {
    "Sessions": [
      {
        "session_id": "string",
        "user_id": "string",
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z",
        "summary": {},
        "message_count": 0
      }
    ]
  }
  ```

#### Session Details
- **URL**: `/admin/sessions/{sessionId}`
- **Method**: `GET`
- **Description**: Detailed view of a specific session
- **Template**: `session_details.html`
- **API Call**: `GET /api/v1/sessions/{sessionId}`
- **Parameters**: 
  - `sessionId` (path): Session identifier

### User Management

#### User List
- **URL**: `/admin/users`
- **Method**: `GET`
- **Description**: Displays paginated list of all users
- **Template**: `users.html`
- **API Call**: `GET /api/v1/users`
- **Data**:
  ```json
  {
    "Users": [
      {
        "user_id": "string",
        "email": "string",
        "first_name": "string",
        "last_name": "string",
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
      }
    ]
  }
  ```

#### User Details
- **URL**: `/admin/users/{userId}`
- **Method**: `GET`
- **Description**: Detailed view of a specific user
- **Template**: `user_details.html`
- **API Call**: `GET /api/v1/users/{userId}`
- **Parameters**: 
  - `userId` (path): User identifier

#### User Sessions
- **URL**: `/admin/users/{userId}/sessions`
- **Method**: `GET`
- **Description**: Lists all sessions for a specific user
- **Template**: `user_sessions.html`
- **API Call**: `GET /api/v1/users/{userId}/sessions`
- **Parameters**: 
  - `userId` (path): User identifier

### Settings
- **URL**: `/admin/settings`
- **Method**: `GET`
- **Description**: Application settings and configuration
- **Template**: `settings.html`
- **Data**: Static configuration page

## HTMX API Endpoints

### Session List API
- **URL**: `/api/sessions`
- **Method**: `GET`
- **Description**: Returns session table HTML fragment for HTMX
- **Template**: `SessionTable`
- **Headers**: Expects `HX-Request` header
- **Response**: HTML table fragment

### User List API
- **URL**: `/api/users`
- **Method**: `GET`
- **Description**: Returns user table HTML fragment for HTMX
- **Template**: `UserTable`
- **Headers**: Expects `HX-Request` header
- **Response**: HTML table fragment

## System Endpoints

### Health Check
- **URL**: `/health`
- **Method**: `GET`
- **Description**: Service health check endpoint
- **Response**: 
  ```json
  {
    "status": "healthy",
    "service": "zep-web-interface"
  }
  ```

### Static Assets
- **URL**: `/static/*`
- **Method**: `GET`
- **Description**: Serves static files (CSS, JS, images)
- **Files**:
  - `/static/css/output.css` - Compiled TailwindCSS
  - `/static/js/htmx.min.js` - HTMX library
  - `/static/js/alpinejs-3.13.0.min.js` - Alpine.js library
  - `/static/js/dark-mode.js` - Dark mode toggle
  - `/static/favicon.png` - Site favicon

## Zep API Integration

The web interface communicates with Zep v1.0.2 API endpoints:

### Authentication
All API calls include Bearer token authentication:
```
Authorization: Bearer {ZEP_API_KEY}
```

### Endpoints Used

#### Sessions
- `GET /api/v1/sessions` - List all sessions
- `GET /api/v1/sessions/{sessionId}` - Get session details

#### Users
- `GET /api/v1/users` - List all users
- `GET /api/v1/users/{userId}` - Get user details
- `GET /api/v1/users/{userId}/sessions` - Get user sessions

### Data Models

#### Session Model
```go
type Session struct {
    SessionID   string                 `json:"session_id"`
    UserID      string                 `json:"user_id,omitempty"`
    CreatedAt   string                 `json:"created_at"`
    UpdatedAt   string                 `json:"updated_at"`
    Summary     map[string]interface{} `json:"summary,omitempty"`
    MessageCount int                   `json:"message_count,omitempty"`
}
```

#### User Model
```go
type User struct {
    UserID    string `json:"user_id"`
    Email     string `json:"email,omitempty"`
    FirstName string `json:"first_name,omitempty"`
    LastName  string `json:"last_name,omitempty"`
    CreatedAt string `json:"created_at"`
    UpdatedAt string `json:"updated_at"`
}
```

#### API Response Models
```go
type SessionsResponse struct {
    Sessions []Session `json:"sessions"`
    Total    int       `json:"total"`
}

type UsersResponse struct {
    Users []User `json:"users"`
    Total int    `json:"total"`
}
```

## Error Handling

### HTTP Status Codes
- `200` - Success
- `400` - Bad Request (invalid parameters)
- `401` - Unauthorized (invalid API key)
- `404` - Not Found (resource doesn't exist)
- `500` - Internal Server Error

### Error Response Format
```json
{
    "error": "error message",
    "status": 400
}
```

### Template Error Handling
When API calls fail, the web interface:
1. Logs the error
2. Returns HTTP 500 status
3. Displays generic error message to user

## Template Functions

### Available Functions
- `formatTime(time.Time) string` - Formats timestamp for display
- `truncate(string, int) string` - Truncates text to specified length

### Usage in Templates
```html
<!-- Format timestamp -->
<span>{{ formatTime .CreatedAt }}</span>

<!-- Truncate long text -->
<p>{{ truncate .Description 100 }}</p>
```

## HTMX Integration

### Dynamic Loading
Pages use HTMX for dynamic content updates:
```html
<!-- Auto-refresh session list every 30 seconds -->
<div hx-get="/api/sessions" 
     hx-trigger="every 30s"
     hx-target="#session-table">
</div>
```

### Form Interactions
```html
<!-- Submit form with HTMX -->
<form hx-post="/api/sessions/search"
      hx-target="#results"
      hx-indicator="#spinner">
</form>
```

## CORS Configuration

### Headers Supported
- `Accept`
- `Authorization`
- `Content-Type`
- `X-CSRF-Token`
- `X-Forwarded-For`
- `X-Real-IP`

### Methods Allowed
- `GET`
- `POST`
- `PUT`
- `DELETE`
- `OPTIONS`

## Rate Limiting

Currently no rate limiting is implemented. Consider adding:
- Request rate limits per IP
- API call limits to Zep server
- Connection pooling for database connections

## Caching Strategy

### Template Caching
- Templates loaded once at startup
- No runtime template compilation
- Efficient memory usage

### API Response Caching
- No caching currently implemented
- Consider Redis for session data
- Cache headers from Zep API respected

## Security Considerations

### Authentication
- No user authentication on web interface
- Relies on network security and API key protection
- Consider adding basic auth for production

### Input Validation
- URL parameters validated
- No user input forms currently
- XSS protection via template escaping

### API Security
- All calls use HTTPS in production
- Bearer token authentication
- Configurable CORS origins