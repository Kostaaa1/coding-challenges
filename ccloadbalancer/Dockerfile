# Stage 1: Build the Go binary
FROM golang:1.24 AS builder
WORKDIR /app

# Copy go.mod and go.sum first (for caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary with static linking
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o lb ./cmd/lb

# Stage 2: Create a lightweight image
FROM alpine:latest
WORKDIR /app

# Install libc compatibility (required for Go binaries)
RUN apk add --no-cache libc6-compat

# Copy the compiled binary from the builder stage
COPY --from=builder /app/lb /app/lb

# Copy the config file into the container
COPY lb_config.json /app/lb_config.json

# Set file ownership to root
RUN chown root:root /app/lb_config.json

# Set permissions: read-only for everyone except root
RUN chmod 644 /app/lb_config.json

# Ensure the binary is executable
RUN chmod +x /app/lb

# Expose port 8080
EXPOSE 8080

# Run the binary
CMD ["/app/lb"]