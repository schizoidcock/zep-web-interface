# Security Policy

## Overview
The Zep Web Interface takes security seriously. This document outlines our security practices, how to report vulnerabilities, and security considerations for deployment.

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.2.x   | :white_check_mark: |
| 0.1.x   | :x:                |

## Security Architecture

### Authentication & Authorization
- **API Authentication**: Uses Bearer token authentication for all Zep API calls
- **No Built-in User Auth**: The web interface itself has no user authentication system
- **Network Security**: Relies on network-level security and access controls
- **Recommendation**: Deploy behind a reverse proxy with authentication (nginx, Traefik, etc.)

### Data Security
- **No Data Storage**: The web interface does not store any session or user data
- **API Proxy Only**: Acts as a proxy between web browsers and the Zep API
- **Memory Only**: All data is processed in memory and not persisted
- **Secure Headers**: Implements security headers for CORS and proxy support

### Network Security
- **TLS Support**: Configurable HTTPS support via `TLS_ENABLED=true`
- **CORS Protection**: Configurable allowed origins via `CORS_ORIGINS`
- **Proxy Trust**: Configurable proxy header trust for secure deployments
- **IPv6 Ready**: Full IPv6 support for modern networking

## Security Best Practices

### Deployment Security

#### Production Configuration
```bash
# Use HTTPS in production
TLS_ENABLED=true

# Restrict CORS to your domains only
CORS_ORIGINS=https://your-domain.com,https://admin.your-domain.com

# Use strong API keys
ZEP_API_KEY=secure_production_key_with_sufficient_entropy

# Bind to specific interface if needed
HOST=::1  # Localhost only
# or
HOST=10.0.0.100  # Specific internal IP
```

#### Reverse Proxy Configuration

**Nginx Example:**
```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;
    
    # SSL configuration
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains";
    
    # Authentication (example with basic auth)
    auth_basic "Restricted Access";
    auth_basic_user_file /etc/nginx/.htpasswd;
    
    # Proxy to web interface
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**Traefik Example:**
```yaml
http:
  middlewares:
    auth:
      basicAuth:
        users:
          - "admin:$2y$10$..."
    
    security-headers:
      headers:
        customRequestHeaders:
          X-Forwarded-Proto: "https"
        customResponseHeaders:
          X-Frame-Options: "DENY"
          X-Content-Type-Options: "nosniff"
          X-XSS-Protection: "1; mode=block"
          Strict-Transport-Security: "max-age=31536000; includeSubDomains"

  routers:
    zep-web:
      rule: "Host(`your-domain.com`)"
      middlewares:
        - auth
        - security-headers
      service: zep-web-service
      tls:
        certResolver: letsencrypt

  services:
    zep-web-service:
      loadBalancer:
        servers:
          - url: "http://localhost:8080"
```

### API Key Management
- **Strong Keys**: Use cryptographically strong API keys with sufficient entropy
- **Key Rotation**: Regularly rotate API keys
- **Environment Variables**: Store keys in environment variables, never in code
- **Access Control**: Limit API key permissions to minimum required scope
- **Monitoring**: Monitor API key usage for unusual patterns

### Container Security

#### Docker Security
```dockerfile
# Use non-root user
FROM alpine:latest
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set secure permissions
COPY --chown=appuser:appgroup --from=builder /app/zep-web-interface .
COPY --chown=appuser:appgroup --from=builder /app/web ./web

# Drop to non-root user
USER appuser
```

#### Kubernetes Security
```yaml
apiVersion: v1
kind: Pod
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1001
    runAsGroup: 1001
    fsGroup: 1001
  containers:
  - name: zep-web-interface
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      capabilities:
        drop:
        - ALL
    volumeMounts:
    - name: tmp
      mountPath: /tmp
  volumes:
  - name: tmp
    emptyDir: {}
```

## Vulnerability Disclosure

### Reporting Security Issues
If you discover a security vulnerability, please report it responsibly:

1. **Do NOT create a public GitHub issue**
2. **Email**: Send details to the project maintainers
3. **Include**: 
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if known)

### Response Process
1. **Acknowledgment**: We'll acknowledge receipt within 48 hours
2. **Investigation**: We'll investigate and assess the severity
3. **Fix Development**: We'll develop and test a fix
4. **Disclosure**: We'll coordinate disclosure timing with you
5. **Release**: We'll release a security update
6. **Credit**: We'll credit you in the security advisory (if desired)

### Security Updates
- Security fixes are released as patch versions
- Critical security issues may result in emergency releases
- Security advisories are published on GitHub
- Users are notified via release notes and changelogs

## Known Security Considerations

### Current Limitations
1. **No User Authentication**: Web interface has no built-in user authentication
2. **Session Management**: No session management or user tracking
3. **Rate Limiting**: No built-in rate limiting (implement at proxy level)
4. **Input Validation**: Limited user input validation (minimal attack surface)

### Recommended Mitigations
1. **Deploy behind authenticated proxy** (nginx, Traefik, CloudFlare Access)
2. **Use network segmentation** to limit access
3. **Implement rate limiting** at proxy or firewall level
4. **Monitor access logs** for unusual patterns
5. **Regular security updates** for dependencies

### Dependencies Security
- Go standard library (maintained by Go team)
- Chi router (actively maintained, security-focused)
- Minimal external dependencies reduce attack surface
- Regular dependency updates via Dependabot (if configured)

## Security Headers

### Implemented Headers
```go
// CORS headers
"Access-Control-Allow-Origin": configurable
"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS"
"Access-Control-Allow-Headers": "Accept, Authorization, Content-Type, X-CSRF-Token, X-Forwarded-For, X-Real-IP"

// Security headers (add via reverse proxy)
"X-Frame-Options": "DENY"
"X-Content-Type-Options": "nosniff"
"X-XSS-Protection": "1; mode=block"
"Strict-Transport-Security": "max-age=31536000; includeSubDomains"
"Content-Security-Policy": "default-src 'self'"
```

## Compliance Considerations

### Data Privacy
- No personal data is stored by the web interface
- All data flows through to Zep API
- Logging should be configured to avoid sensitive data
- GDPR/CCPA compliance depends on Zep server configuration

### Access Logging
```go
// Recommended logging configuration
log.Printf("Request: %s %s from %s", method, path, clientIP)
// Avoid logging sensitive headers or parameters
```

### Audit Trail
- All API calls are logged by default
- Consider centralized logging for audit purposes
- Monitor failed authentication attempts
- Track configuration changes

## Security Testing

### Manual Security Testing
1. **Authentication Bypass**: Verify API key is required
2. **Authorization**: Test with invalid/expired keys
3. **Input Validation**: Test with malformed URLs/parameters
4. **CORS**: Verify origin restrictions work
5. **Headers**: Check security headers are present

### Automated Security Testing
Consider integrating:
- SAST (Static Application Security Testing)
- Dependency scanning (GitHub Dependabot)
- Container scanning
- Infrastructure scanning

### Penetration Testing
For production deployments:
- Regular penetration testing
- Focus on authentication bypass
- Test proxy configuration
- Verify network segmentation

## Incident Response

### Security Incident Process
1. **Detection**: Monitor for security events
2. **Assessment**: Evaluate severity and impact
3. **Containment**: Isolate affected systems
4. **Investigation**: Determine root cause
5. **Recovery**: Restore normal operations
6. **Lessons Learned**: Update security measures

### Emergency Contacts
- Maintain list of security contacts
- Include cloud provider support if applicable
- Document escalation procedures
- Test incident response regularly

---

For questions about security, please contact the project maintainers through appropriate channels. Thank you for helping keep the Zep Web Interface secure!