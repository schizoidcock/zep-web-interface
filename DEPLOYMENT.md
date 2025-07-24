# Deployment Guide

## Overview
This guide covers deploying the Zep Web Interface across different environments and platforms.

## Prerequisites
- Zep v1.0.2 server running and accessible
- Valid Zep API key with appropriate permissions
- Network connectivity between web interface and Zep server

## Environment Variables

### Required Configuration
```bash
ZEP_API_URL=http://your-zep-server:8000  # Your Zep v1.0.2 server URL
ZEP_API_KEY=your-api-key-here            # Your Zep API authentication key
```

### Optional Configuration
```bash
HOST=::                                  # Server host (default: :: for IPv6)
PORT=8080                               # Server port (default: 8080)
PROXY_URL=http://proxy:8080             # HTTP proxy URL (optional)
TRUST_PROXY=true                        # Trust proxy headers (default: true)
CORS_ORIGINS=*                          # Allowed origins (default: *)
TLS_ENABLED=false                       # Enable HTTPS (default: false)
```

## Local Development

### 1. Setup Environment
```bash
# Copy environment template
cp .env.example .env

# Edit configuration
nano .env
```

### 2. Run Development Server
```bash
# Install dependencies
go mod tidy

# Run server
go run main.go
```

### 3. Access Interface
Navigate to `http://localhost:8080/admin`

## Docker Deployment

### 1. Build Container
```bash
docker build -t zep-web-interface .
```

### 2. Run Container
```bash
docker run -d \
  --name zep-web-ui \
  -p 8080:8080 \
  -e ZEP_API_URL=http://zep-server:8000 \
  -e ZEP_API_KEY=your-api-key \
  zep-web-interface
```

### 3. Docker Compose Example
```yaml
version: '3.8'
services:
  zep-web-ui:
    build: .
    ports:
      - "8080:8080"
    environment:
      ZEP_API_URL: http://zep-server:8000
      ZEP_API_KEY: ${ZEP_API_KEY}
      TRUST_PROXY: "true"
    depends_on:
      - zep-server
    networks:
      - zep-network

  zep-server:
    image: ghcr.io/getzep/zep:latest
    ports:
      - "8000:8000"
    networks:
      - zep-network

networks:
  zep-network:
    driver: bridge
```

## Railway Deployment

### 1. Railway Configuration
Create `railway.json`:
```json
{
  "$schema": "https://railway.app/railway.schema.json",
  "build": {
    "builder": "DOCKERFILE"
  },
  "deploy": {
    "startCommand": "./zep-web-interface",
    "healthcheckPath": "/health",
    "healthcheckTimeout": 100
  }
}
```

### 2. Environment Variables
Set in Railway dashboard:
```bash
ZEP_API_URL=${{services.zep-server.url}}
ZEP_API_KEY=your-production-api-key
HOST=::
PORT=${{PORT}}
TRUST_PROXY=true
CORS_ORIGINS=https://your-domain.railway.app
```

### 3. Deploy
```bash
# Install Railway CLI
npm install -g @railway/cli

# Login and deploy
railway login
railway deploy
```

## Heroku Deployment

### 1. Create Heroku App
```bash
heroku create zep-web-interface
```

### 2. Set Environment Variables
```bash
heroku config:set ZEP_API_URL=https://your-zep-server.herokuapp.com
heroku config:set ZEP_API_KEY=your-api-key
heroku config:set TRUST_PROXY=true
```

### 3. Create Procfile
```
web: ./zep-web-interface
```

### 4. Deploy
```bash
git push heroku main
```

## Kubernetes Deployment

### 1. ConfigMap
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: zep-web-config
data:
  HOST: "::"
  PORT: "8080"
  TRUST_PROXY: "true"
  CORS_ORIGINS: "*"
```

### 2. Secret
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: zep-web-secrets
type: Opaque
stringData:
  ZEP_API_URL: "http://zep-server-service:8000"
  ZEP_API_KEY: "your-api-key"
```

### 3. Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: zep-web-interface
spec:
  replicas: 2
  selector:
    matchLabels:
      app: zep-web-interface
  template:
    metadata:
      labels:
        app: zep-web-interface
    spec:
      containers:
      - name: zep-web-interface
        image: zep-web-interface:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: zep-web-config
        - secretRef:
            name: zep-web-secrets
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
```

### 4. Service
```yaml
apiVersion: v1
kind: Service
metadata:
  name: zep-web-interface-service
spec:
  selector:
    app: zep-web-interface
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

## Production Considerations

### Security
```bash
# Use HTTPS in production
TLS_ENABLED=true

# Restrict CORS origins
CORS_ORIGINS=https://your-domain.com,https://admin.your-domain.com

# Use secure API keys
ZEP_API_KEY=prod_secure_key_here
```

### Performance
- Enable HTTP/2 with TLS
- Use CDN for static assets
- Configure load balancer health checks
- Set appropriate timeout values

### Monitoring
```bash
# Health check endpoint
curl http://your-domain.com/health

# Expected response
{"status":"healthy","service":"zep-web-interface"}
```

### Scaling
- Configure horizontal pod autoscaling in Kubernetes
- Use multiple Railway/Heroku dynos
- Load balance across multiple instances

## Troubleshooting

### Common Issues

#### 1. Cannot Connect to Zep Server
```bash
# Check API URL accessibility
curl $ZEP_API_URL/api/v1/health

# Verify network connectivity
ping zep-server-host
```

#### 2. Authentication Errors
```bash
# Test API key
curl -H "Authorization: Bearer $ZEP_API_KEY" $ZEP_API_URL/api/v1/sessions
```

#### 3. Template Loading Errors
```bash
# Check file permissions
ls -la web/templates/

# Verify template structure
find web/templates/ -name "*.html"
```

### Log Analysis
```bash
# View container logs
docker logs zep-web-ui

# Follow logs in real-time
docker logs -f zep-web-ui

# Railway logs
railway logs

# Heroku logs
heroku logs --tail
```

### Configuration Validation
The service validates all configuration at startup:
- Invalid URLs will cause startup failure
- Missing required variables will cause panic
- Port conflicts will prevent server start

### Network Debugging
```bash
# Test internal connectivity (Docker)
docker exec zep-web-ui ping zep-server

# Test from container
docker exec -it zep-web-ui wget -qO- $ZEP_API_URL/health
```