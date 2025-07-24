#!/bin/bash

echo "=== Zep Web Interface Complete Test ==="
echo ""

# Check if environment variables are set
if [ -z "$ZEP_API_URL" ] || [ -z "$ZEP_API_KEY" ]; then
    echo "❌ Missing required environment variables:"
    echo "   Please set ZEP_API_URL and ZEP_API_KEY"
    echo ""
    echo "Example:"
    echo "export ZEP_API_URL=\"http://localhost:8000\""
    echo "export ZEP_API_KEY=\"your-api-key\""
    exit 1
fi

echo "✅ Environment variables set:"
echo "   ZEP_API_URL: $ZEP_API_URL"
echo "   ZEP_API_KEY: ${ZEP_API_KEY:0:8}..."
echo ""

# Test 1: Build the application
echo "🔨 Building application..."
if go build -o zep-web-interface.exe ./main.go; then
    echo "✅ Build successful"
else
    echo "❌ Build failed"
    exit 1
fi
echo ""

# Test 2: API connectivity test
echo "🌐 Testing API connectivity..."
if go run test_api.go; then
    echo "✅ API connectivity test passed"
else
    echo "❌ API connectivity test failed"
    exit 1
fi
echo ""

# Test 3: Start server in background for quick test
echo "🚀 Starting web interface server (quick test)..."
export PORT=8081  # Use different port to avoid conflicts
export HOST=localhost

# Start server in background
./zep-web-interface.exe &
SERVER_PID=$!

# Wait a moment for server to start
sleep 3

# Test health endpoint
echo "🏥 Testing health endpoint..."
if curl -s http://localhost:8081/health | grep -q "healthy"; then
    echo "✅ Health endpoint working"
else
    echo "❌ Health endpoint failed"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

# Test admin redirect
echo "🔀 Testing admin redirect..."
if curl -s -I http://localhost:8081/ | grep -q "302"; then
    echo "✅ Admin redirect working"
else
    echo "❌ Admin redirect failed"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

# Test admin page loads
echo "📄 Testing admin page loads..."
if curl -s http://localhost:8081/admin | grep -q "<title>"; then
    echo "✅ Admin page loads successfully"
else
    echo "❌ Admin page failed to load"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

# Clean up
kill $SERVER_PID 2>/dev/null
sleep 1

echo ""
echo "🎉 All tests passed! Web interface is ready."
echo ""
echo "To run the web interface:"
echo "  ./zep-web-interface.exe"
echo ""
echo "Then access: http://localhost:8080/admin"