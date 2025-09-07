# GoReleaser Dockerfile - uses pre-built binary
FROM --platform=$TARGETPLATFORM alpine:latest

# Install git (required for gitagrip to function)
RUN apk add --no-cache git ca-certificates

# Create non-root user
RUN adduser -D -s /bin/sh gitagrip

# Copy the pre-built binary from GoReleaser
COPY gitagrip /usr/local/bin/gitagrip

# Set ownership and permissions
RUN chmod +x /usr/local/bin/gitagrip && \
    chown gitagrip:gitagrip /usr/local/bin/gitagrip

# Switch to non-root user
USER gitagrip

# Set working directory
WORKDIR /repos

# Entrypoint
ENTRYPOINT ["gitagrip"]