package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const MediaDir = "./media"

func main() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable not set")
	}

	// Get whitelist users
	whitelistUsers := getWhitelistUsers()

	// Create media directory
	if err := os.MkdirAll(MediaDir, 0755); err != nil {
		log.Fatal("Failed to create media directory:", err)
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal("Failed to create bot:", err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		go handleMessage(bot, update.Message, whitelistUsers)
	}
}

func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, whitelistUsers []string) {
	if message.Text == "" {
		return
	}

	// Check if user is whitelisted
	if !isUserWhitelisted(message.From.UserName, whitelistUsers) {
		sendMessage(bot, message.Chat.ID, "❌ You are not authorized to operate this bot")
		return
	}

	if isYouTubeURL(message.Text) {
		videoID := extractVideoID(message.Text)
		if videoID == "" {
			sendMessage(bot, message.Chat.ID, "Could not extract video ID from URL")
			return
		}

		sendMessage(bot, message.Chat.ID, "📥 Downloading video...")

		videoDir := filepath.Join(MediaDir, videoID)
		if err := os.MkdirAll(videoDir, 0755); err != nil {
			log.Printf("Failed to create video directory: %v", err)
			sendMessage(bot, message.Chat.ID, "❌ Failed to create download directory")
			return
		}

		videoFile, err := downloadVideo(message.Text, videoDir)
		if err != nil {
			log.Printf("Failed to download video: %v", err)
			sendMessage(bot, message.Chat.ID, "❌ Failed to download video")
			return
		}

		if videoFile == "" {
			sendMessage(bot, message.Chat.ID, "❌ No video file found after download")
			return
		}

		// Check file size (Telegram has a 50MB limit for bots)
		fileInfo, err := os.Stat(videoFile)
		if err != nil {
			log.Printf("Failed to get file info: %v", err)
			sendMessage(bot, message.Chat.ID, "❌ Error checking video file")
			return
		}

		if fileInfo.Size() > 50*1024*1024 { // 50MB
			sendMessage(bot, message.Chat.ID, "❌ Video is too large (>50MB) to send via Telegram")
			cleanup(videoFile, videoDir)
			return
		}

		sendMessage(bot, message.Chat.ID, "📤 Sending video...")

		videoMsg := tgbotapi.NewVideo(message.Chat.ID, tgbotapi.FilePath(videoFile))
		if _, err := bot.Send(videoMsg); err != nil {
			log.Printf("Failed to send video: %v", err)
			sendMessage(bot, message.Chat.ID, "❌ Error sending video")
		}

		// Clean up
		cleanup(videoFile, videoDir)
	}
}

func isYouTubeURL(text string) bool {
	patterns := []string{
		`youtube\.com/watch\?v=`,
		`youtu\.be/`,
		`youtube\.com/embed/`,
		`youtube\.com/v/`,
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, text); matched {
			return true
		}
	}
	return false
}

func extractVideoID(url string) string {
	patterns := []string{
		`youtube\.com/watch\?v=([^&]+)`,
		`youtu\.be/([^?]+)`,
		`youtube\.com/embed/([^?]+)`,
		`youtube\.com/v/([^?]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(url)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

func downloadVideo(url, outputDir string) (string, error) {
	cmd := exec.Command("yt-dlp",
		"--format", "bestvideo+bestaudio",
		"--merge-output-format", "mp4",
		"--output", filepath.Join(outputDir, "%(title)s.%(ext)s"),
		url)

	// Log the exact command being executed
	log.Printf("Executing yt-dlp command: %s", strings.Join(cmd.Args, " "))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("yt-dlp failed: %v, output: %s", err, string(output))
	}

	// Find the downloaded file
	files, err := filepath.Glob(filepath.Join(outputDir, "*"))
	if err != nil {
		return "", fmt.Errorf("failed to find downloaded file: %v", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file, ".mp4") || strings.HasSuffix(file, ".mkv") || strings.HasSuffix(file, ".webm") {
			return file, nil
		}
	}

	return "", fmt.Errorf("no video file found")
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}

func getWhitelistUsers() []string {
	whitelistEnv := os.Getenv("TELEGRAM_BOT_WHITELIST_USERS")
	if whitelistEnv == "" {
		return []string{}
	}

	users := strings.Split(whitelistEnv, ",")
	for i, user := range users {
		users[i] = strings.TrimSpace(user)
	}
	return users
}

func isUserWhitelisted(username string, whitelistUsers []string) bool {
	if len(whitelistUsers) == 0 {
		return true // If no whitelist is set, allow all users
	}

	for _, whitelistedUser := range whitelistUsers {
		if whitelistedUser == username {
			return true
		}
	}
	return false
}

func cleanup(videoFile, videoDir string) {
	if err := os.Remove(videoFile); err != nil {
		log.Printf("Failed to remove video file: %v", err)
	}

	// Remove directory if empty
	if files, err := os.ReadDir(videoDir); err == nil && len(files) == 0 {
		if err := os.Remove(videoDir); err != nil {
			log.Printf("Failed to remove video directory: %v", err)
		}
	}
}

