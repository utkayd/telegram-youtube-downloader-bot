# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Running the Bot
```bash
go run main.go
```

### Building
```bash
go build -o telegram-youtube-bot .
```

### Docker Commands
```bash
# Build Docker image
docker build -t telegram-youtube-bot .

# Run with environment variables
docker run -e TELEGRAM_BOT_TOKEN=your_token -e TELEGRAM_BOT_WHITELIST_USERS=user1,user2 telegram-youtube-bot

# Using Docker Compose
docker-compose up -d
```

### Dependencies
```bash
# Install Go dependencies
go mod download

# Install yt-dlp (required system dependency)
pip install yt-dlp
```

## Architecture

This is a single-file Go application that implements a Telegram bot for downloading videos from various platforms.

### Core Components

**Main Application Flow:**
- `main()`: Initializes bot, creates media directory, starts message handling loop
- `handleMessage()`: Processes incoming messages, validates users, orchestrates download/send flow
- Message processing is handled concurrently using goroutines

**Video Processing Pipeline:**
1. URL validation via `isSupportedURL()` - checks against regex patterns for supported platforms
2. Random hash generation for unique storage directories (`generateRandomHash()`)
3. Video download using `downloadVideo()` - executes yt-dlp with specific format parameters
4. File size checking (50MB Telegram limit)
5. Video splitting with `splitVideo()` if needed - uses ffmpeg for chunking
6. File transmission to Telegram
7. Cleanup with `cleanup()`

**Key Features:**
- **Multi-platform support**: YouTube, Instagram, TikTok, Reddit, Twitter/X, Facebook, Twitch, Vimeo, Dailymotion
- **User authorization**: Whitelist-based access control via `TELEGRAM_BOT_WHITELIST_USERS`
- **Large file handling**: Automatic video splitting into <40MB chunks using ffmpeg
- **Organized storage**: Videos stored in `./media/{random_hash}/` directories
- **Automatic cleanup**: Files and directories removed after successful transmission

### Dependencies

**Go Dependencies:**
- `github.com/go-telegram-bot-api/telegram-bot-api/v5` - Telegram Bot API wrapper

**System Dependencies:**
- `yt-dlp` - Video downloading (supports all major platforms)
- `ffmpeg` - Video processing and splitting
- `ffprobe` - Video metadata extraction

### Environment Configuration

Required:
- `TELEGRAM_BOT_TOKEN` - Telegram bot token from @BotFather

Optional:
- `TELEGRAM_BOT_WHITELIST_USERS` - Comma-separated usernames for access control (if empty, allows all users)

### File Structure

- `main.go` - Single-file application containing all functionality
- `media/` - Download directory (auto-created, contains subdirectories per download)
- `Dockerfile` - Multi-stage build with Go compilation and runtime dependencies
- `docker-compose.yml` - Contains actual bot token and configuration