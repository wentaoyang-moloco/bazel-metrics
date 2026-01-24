# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./
RUN go mod download

# Copy source code
COPY . .

# Build the analyzer
RUN CGO_ENABLED=0 GOOS=linux go build -o /analyzer ./cmd/main.go

# Runtime stage
FROM google/cloud-sdk:alpine

# Install git for cloning repos
RUN apk add --no-cache git bash

# Copy the analyzer binary
COPY --from=builder /analyzer /usr/local/bin/analyzer

# Copy the run script
COPY run-job.sh /usr/local/bin/run-job.sh
RUN chmod +x /usr/local/bin/run-job.sh

# Set the entrypoint
ENTRYPOINT ["/usr/local/bin/run-job.sh"]
