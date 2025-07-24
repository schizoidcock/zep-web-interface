# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o zep-web-interface ./main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary and web assets
COPY --from=builder /app/zep-web-interface .
COPY --from=builder /app/web ./web

# Expose port
EXPOSE 8080

# Set default environment variables
ENV HOST=0.0.0.0
ENV PORT=8080
ENV TRUST_PROXY=true

# Run the application
CMD ["./zep-web-interface"]