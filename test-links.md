# Link Testing Guide

## Test Scenarios

### 1. Default Configuration (no PROXY_PATH)
```bash
# Set environment variables
export ZEP_API_URL="http://localhost:8000"
export ZEP_API_KEY="your-api-key"
export HOST="localhost"
export PORT="8080"

# Start server
go run main.go

# Test URLs:
# - http://localhost:8080/admin (dashboard)
# - http://localhost:8080/admin/users (users list)
# - http://localhost:8080/admin/sessions (sessions list)
# - http://localhost:8080/admin/settings (settings)
```

### 2. Custom Proxy Path Configuration
```bash
# Set environment variables with proxy path
export ZEP_API_URL="http://localhost:8000"
export ZEP_API_KEY="your-api-key"
export HOST="localhost"
export PORT="8080"
export PROXY_PATH="/my-admin"

# Start server
go run main.go

# Test URLs:
# - http://localhost:8080/my-admin (dashboard)
# - http://localhost:8080/my-admin/users (users list)
# - http://localhost:8080/my-admin/sessions (sessions list)
# - http://localhost:8080/my-admin/settings (settings)
```

## Link Tests to Perform

### Navigation Links
- [x] Logo in sidebar links to dashboard
- [x] Dashboard menu item
- [x] Users menu item
- [x] Sessions menu item
- [x] Settings menu item
- [x] External documentation link

### Breadcrumb Links
- [x] Sessions list breadcrumb
- [x] Session details breadcrumbs
- [x] Users list breadcrumb
- [x] User details breadcrumbs
- [x] User sessions breadcrumbs
- [x] Create user breadcrumbs
- [x] Settings breadcrumb

### Table Action Links
- [x] User table "View" links
- [x] User table name links
- [x] Session table "View" links
- [x] Session table ID links
- [x] "Add user" button
- [x] Delete user button
- [x] Delete session button

### HTMX Navigation
- [x] HTMX redirects after delete operations
- [x] HTMX form submissions
- [x] HTMX content loading

## Fixed Issues

1. **Hardcoded /admin paths**: All paths now use dynamic `adminPath` template function
2. **Menu items**: Menu generation now uses `GetMenuItems(basePath)` function
3. **Breadcrumbs**: All breadcrumb paths use dynamic base path
4. **Redirects**: HTMX and regular redirects use handler's basePath
5. **Template links**: User and session tables use `adminPath` function
6. **Logo link**: Sidebar logo uses dynamic path

## Implementation Details

- Added `adminPath` template function that respects PROXY_PATH
- Updated Handlers struct to store basePath
- Modified all breadcrumb generation to use dynamic paths
- Updated all HTMX redirect headers to use basePath
- Fixed table template links to use adminPath function