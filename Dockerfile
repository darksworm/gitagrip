# Build stage
FROM golang:1.21-alpine AS builder

# Install git for go modules
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o gitagrip .

# Final stage
FROM alpine:latest

# Install git (required for gitagrip to function)
RUN apk add --no-cache git

# Create non-root user
RUN adduser -D -s /bin/sh gitagrip

# Copy binary from builder
COPY --from=builder /app/gitagrip /usr/local/bin/gitagrip

# Set ownership
RUN chown gitagrip:gitagrip /usr/local/bin/gitagrip

# Switch to non-root user
USER gitagrip

# Set working directory
WORKDIR /repos

# Entrypoint
ENTRYPOINT ["gitagrip"]