package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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

	if isSupportedURL(message.Text) {
		// Generate random hash for storage
		randomHash := generateRandomHash()

		sendMessage(bot, message.Chat.ID, "📥 Downloading video...")

		videoDir := filepath.Join(MediaDir, randomHash)
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
			sendMessage(bot, message.Chat.ID, "📹 Video is larger than 50MB, splitting into chunks...")

			chunks, err := splitVideo(videoFile, videoDir)
			if err != nil {
				log.Printf("Failed to split video: %v", err)
				sendMessage(bot, message.Chat.ID, "❌ Error splitting video")
				cleanup(videoFile, videoDir)
				return
			}

			sendMessage(bot, message.Chat.ID, fmt.Sprintf("📤 Sending %d video chunks...", len(chunks)))

			for i, chunk := range chunks {
				// Check chunk size before sending
				chunkInfo, err := os.Stat(chunk)
				if err != nil {
					log.Printf("Failed to get chunk %d info: %v", i+1, err)
					continue
				}

				if chunkInfo.Size() > 50*1024*1024 {
					log.Printf("Chunk %d is too large (%d bytes), skipping", i+1, chunkInfo.Size())
					sendMessage(bot, message.Chat.ID, fmt.Sprintf("⚠️ Chunk %d is too large, skipping", i+1))
					continue
				}

				chunkMsg := tgbotapi.NewVideo(message.Chat.ID, tgbotapi.FilePath(chunk))
				chunkMsg.Caption = fmt.Sprintf("Part %d/%d (%.1fMB)", i+1, len(chunks), float64(chunkInfo.Size())/(1024*1024))

				if _, err := bot.Send(chunkMsg); err != nil {
					log.Printf("Failed to send video chunk %d: %v", i+1, err)
					sendMessage(bot, message.Chat.ID, fmt.Sprintf("❌ Error sending chunk %d", i+1))
				}
			}

			// Clean up chunks
			for _, chunk := range chunks {
				os.Remove(chunk)
			}
		} else {
			sendMessage(bot, message.Chat.ID, "📤 Sending video...")

			videoMsg := tgbotapi.NewVideo(message.Chat.ID, tgbotapi.FilePath(videoFile))
			if _, err := bot.Send(videoMsg); err != nil {
				log.Printf("Failed to send video: %v", err)
				sendMessage(bot, message.Chat.ID, "❌ Error sending video")
			}
		}

		// Clean up
		cleanup(videoFile, videoDir)
	}
}

func isSupportedURL(text string) bool {
	// Check for common video platform URLs
	patterns := []string{
		// YouTube
		`youtube\.com/watch`,
		`youtu\.be/`,
		`youtube\.com/embed/`,
		`youtube\.com/v/`,
		`youtube\.com/shorts/`,
		// Instagram
		`instagram\.com/p/`,
		`instagram\.com/reel/`,
		`instagram\.com/tv/`,
		`instagram\.com/stories/`,
		// TikTok
		`tiktok\.com/`,
		`vm\.tiktok\.com/`,
		// Reddit
		`reddit\.com/r/.*/comments/`,
		`v\.redd\.it/`,
		// Twitter/X
		`twitter\.com/.*/status/`,
		`x\.com/.*/status/`,
		// Facebook
		`facebook\.com/.*/videos/`,
		`fb\.watch/`,
		// Twitch
		`twitch\.tv/`,
		`clips\.twitch\.tv/`,
		// Vimeo
		`vimeo\.com/`,
		// Dailymotion
		`dailymotion\.com/video/`,
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, text); matched {
			return true
		}
	}
	return false
}

func generateRandomHash() string {
	bytes := make([]byte, 16) // 16 bytes = 128 bits
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based hash if crypto/rand fails
		return fmt.Sprintf("%d", os.Getpid())
	}
	return hex.EncodeToString(bytes)
}

func downloadVideo(url, outputDir string) (string, error) {
	cmd := exec.Command("yt-dlp",
		"--format", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
		"--merge-output-format", "mp4",
		"--postprocessor-args", "ffmpeg:-c:v libx264 -profile:v baseline -level 3.0 -pix_fmt yuv420p -c:a aac",
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

func splitVideo(videoFile, outputDir string) ([]string, error) {
	// Get video duration first
	durationCmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
		videoFile)

	durationOutput, err := durationCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get video duration: %v", err)
	}

	duration, err := strconv.ParseFloat(strings.TrimSpace(string(durationOutput)), 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse duration: %v", err)
	}

	// Calculate chunk duration to stay under 40MB (more conservative buffer)
	fileInfo, _ := os.Stat(videoFile)
	fileSizeMB := float64(fileInfo.Size()) / (1024 * 1024)
	targetSizeMB := 40.0 // 40MB to leave more buffer for bitrate variations

	chunkDuration := (duration * targetSizeMB) / fileSizeMB

	// Ensure minimum chunk duration of 30 seconds to avoid too many small chunks
	if chunkDuration < 30 {
		chunkDuration = 30
	}

	numChunks := int(duration/chunkDuration) + 1

	var chunks []string

	for i := 0; i < numChunks; i++ {
		startTime := float64(i) * chunkDuration
		chunkFile := filepath.Join(outputDir, fmt.Sprintf("chunk_%d.mp4", i+1))

		cmd := exec.Command("ffmpeg",
			"-i", videoFile,
			"-ss", fmt.Sprintf("%.2f", startTime),
			"-t", fmt.Sprintf("%.2f", chunkDuration),
			"-c:v", "libx264",
			"-profile:v", "baseline",
			"-level", "3.0",
			"-pix_fmt", "yuv420p",
			"-c:a", "aac",
			"-movflags", "+faststart",
			"-avoid_negative_ts", "make_zero",
			chunkFile)

		if err := cmd.Run(); err != nil {
			// Clean up any created chunks on error
			for _, chunk := range chunks {
				os.Remove(chunk)
			}
			return nil, fmt.Errorf("failed to create chunk %d: %v", i+1, err)
		}

		// Check if chunk file was actually created and has content
		if info, err := os.Stat(chunkFile); err == nil && info.Size() > 0 {
			chunks = append(chunks, chunkFile)
		}
	}

	return chunks, nil
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
