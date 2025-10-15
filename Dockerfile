# syntax=docker/dockerfile:1

# Build stage
FROM --platform=$BUILDPLATFORM golang:1.24.2-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
ARG TARGETPLATFORM
ARG BUILDPLATFORM
RUN GOOS=$(echo $TARGETPLATFORM | cut -d/ -f1) \
    GOARCH=$(echo $TARGETPLATFORM | cut -d/ -f2) \
    go build -o speedtest-go

# Final stage
FROM alpine:latest

# Install bash
RUN apk add --no-cache bash

# Create non-root user
RUN adduser -D -h /home/speedtest speedtest

WORKDIR /home/speedtest

# Copy the binary from builder
COPY --from=builder /app/speedtest-go /usr/local/bin/

# Switch to non-root user
USER speedtest

# Set default shell
SHELL ["/bin/bash", "-c"]

# Set the entrypoint to bash, we do this rather than using the speedtest command directly
# such that you can also use this container in an interactive way to run speedtests.
# see the README for more info and examples.
CMD ["/bin/bash"]