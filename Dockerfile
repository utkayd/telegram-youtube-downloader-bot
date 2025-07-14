FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies for yt-dlp
RUN apk add --no-cache \
    python3 \
    py3-pip \
    ffmpeg

# Install yt-dlp
RUN pip3 install --break-system-packages yt-dlp

# Copy go mod files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o telegram-youtube-bot .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    python3 \
    py3-pip \
    ffmpeg

# Install yt-dlp
RUN pip3 install --break-system-packages yt-dlp

# Create media directory
RUN mkdir -p /media

# Copy the binary from builder
COPY --from=builder /app/telegram-youtube-bot /usr/local/bin/

# Set working directory
WORKDIR /

# Expose port (not needed for Telegram bot but good practice)
EXPOSE 8080

# Run the bot
CMD ["telegram-youtube-bot"]
