# Telegram YouTube Downloader Bot

A Telegram bot service that downloads YouTube videos using yt-dlp and sends them back to users. The bot extracts video IDs from YouTube URLs, downloads videos in the best available quality, and delivers them through Telegram with automatic cleanup.

## Features

- **YouTube URL Detection**: Supports multiple YouTube URL formats (youtube.com/watch, youtu.be, embed, etc.)
- **Video Download**: Uses yt-dlp to download videos in best quality (mp4 preferred)
- **File Size Limit**: Automatically checks 50MB Telegram file size limit
- **User Authorization**: Whitelist-based access control
- **Automatic Cleanup**: Downloads are cleaned up after sending
- **Organized Storage**: Videos stored in `/media/{video_id}/` structure

## Setup

### Environment Variables

Create a `.env` file or set the following environment variables:

```bash
TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here
TELEGRAM_BOT_WHITELIST_USERS=username1,username2,username3
```

- `TELEGRAM_BOT_TOKEN`: Your Telegram bot token from [@BotFather](https://t.me/BotFather)
- `TELEGRAM_BOT_WHITELIST_USERS`: Comma-separated list of authorized usernames (optional - if not set, all users allowed)

### Local Development

1. Install dependencies:
```bash
go mod download
```

2. Install yt-dlp:
```bash
pip install yt-dlp
```

3. Run the bot:
```bash
go run main.go
```

### Docker Deployment

Build and run with Docker:

```bash
docker build -t telegram-youtube-bot .
docker run -e TELEGRAM_BOT_TOKEN=your_token -e TELEGRAM_BOT_WHITELIST_USERS=user1,user2 telegram-youtube-bot
```

## Usage

1. Start a conversation with your bot on Telegram
2. Send any YouTube URL (e.g., `https://youtube.com/watch?v=abc123`)
3. The bot will:
   - Extract the video ID (`abc123`)
   - Download the video using yt-dlp
   - Send the video file back to you
   - Clean up the downloaded file

## Technical Details

- **Language**: Go
- **Framework**: go-telegram-bot-api/v5
- **Download Tool**: yt-dlp
- **Storage**: Temporary files in `/media/{video_id}/`
- **File Formats**: MP4, MKV, WebM supported
- **Size Limit**: 50MB (Telegram bot API limit)

## Project Structure

```
├── main.go           # Main bot service code
├── go.mod           # Go module dependencies
├── Dockerfile       # Multi-stage Docker build
├── .env.example     # Environment variable template
├── README.md        # This file
└── LICENSE          # License file
```
