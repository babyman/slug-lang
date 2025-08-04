# Stage 1: Builder
FROM golang:1.24 AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates zip && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Set environment variables for Go
ENV CGO_ENABLED=0
ENV GOOS=linux

# Copy the project sources into the build image
WORKDIR /app
COPY . .

# Fetch dependencies and build the project
RUN go mod tidy && make package

# Stage 2: Runtime
FROM debian:bullseye-slim

# Set up a working directory
WORKDIR /data

# Copy the compiled CLI binary from the builder stage
COPY --from=builder /app/dist/slug/bin/slug /usr/local/bin/slug

# Copy the required runtime libraries and docs
COPY --from=builder /app/dist/slug/lib /app/lib
COPY --from=builder /app/dist/slug/docs /app/docs

# Export SLUG_HOME to point to the lib/directory
ENV SLUG_HOME=/app

# Make the binary executable
RUN chmod +x /usr/local/bin/slug

# Set entrypoint to execute the CLI tool
ENTRYPOINT ["slug"]
