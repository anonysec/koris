package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// TelegramSender sends notifications to a configured Telegram chat.
// Set environment variables:
//   PANEL_TG_BOT_TOKEN  - Telegram bot token from @BotFather
//   PANEL_TG_CHAT_ID    - Telegram chat/group ID to send messages to
//   PANEL_TG_ENABLED    - "true" to enable (disabled by default)
type TelegramSender struct {
	BotToken string
	ChatID   string
	Enabled  bool
	client   *http.Client
}

// TelegramConfig holds Telegram notification configuration.
type TelegramConfig struct {
	BotToken string
	ChatID   string
	Enabled  bool
}

// LoadTelegramConfig reads Telegram configuration from the environment.
func LoadTelegramConfig() TelegramConfig {
	return TelegramConfig{
		BotToken: strings.TrimSpace(os.Getenv("PANEL_TG_BOT_TOKEN")),
		ChatID:   strings.TrimSpace(os.Getenv("PANEL_TG_CHAT_ID")),
		Enabled:  strings.ToLower(strings.TrimSpace(os.Getenv("PANEL_TG_ENABLED"))) == "true",
	}
}

// NewTelegramSender creates a TelegramSender from the provided config.
func NewTelegramSender(cfg TelegramConfig) *TelegramSender {
	return &TelegramSender{
		BotToken: cfg.BotToken,
		ChatID:   cfg.ChatID,
		Enabled:  cfg.Enabled,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// IsEnabled reports whether the Telegram sender is configured and enabled.
func (n *TelegramSender) IsEnabled() bool {
	return n.Enabled && n.BotToken != "" && n.ChatID != ""
}

// Send sends a message to the configured Telegram chat.
// It runs asynchronously and never blocks the caller.
func (n *TelegramSender) Send(message string) {
	if !n.IsEnabled() {
		return
	}
	go n.sendSync(message)
}

// SendEvent sends a formatted event notification.
func (n *TelegramSender) SendEvent(eventType, title, detail string) {
	icon := eventIcon(eventType)
	msg := fmt.Sprintf("%s *%s*\n%s", icon, escapeMarkdown(title), escapeMarkdown(detail))
	n.Send(msg)
}

// NotifyPayment sends a payment status notification.
func (n *TelegramSender) NotifyPayment(username string, amount float64, status string) {
	icon := "💳"
	if status == "approved" {
		icon = "✅"
	} else if status == "rejected" {
		icon = "❌"
	}
	msg := fmt.Sprintf("%s *Payment %s*\nUser: `%s`\nAmount: %.0f IRT", icon, status, escapeMarkdown(username), amount)
	n.Send(msg)
}

// NotifyCustomerCreated sends new customer notification.
func (n *TelegramSender) NotifyCustomerCreated(username, creator string) {
	msg := fmt.Sprintf("👤 *New Customer*\nUser: `%s`\nCreated by: %s", escapeMarkdown(username), escapeMarkdown(creator))
	n.Send(msg)
}

// NotifyExpiry sends a subscription expiry notification.
func (n *TelegramSender) NotifyExpiry(username string) {
	msg := fmt.Sprintf("⏰ *Subscription Expired*\nUser: `%s`\nStatus changed to expired.", escapeMarkdown(username))
	n.Send(msg)
}

// NotifyNodeOffline sends a node offline notification.
func (n *TelegramSender) NotifyNodeOffline(nodeName, nodeIP string) {
	msg := fmt.Sprintf("🔴 *Node Offline*\nNode: `%s`\nIP: %s\nLast seen more than 5 minutes ago.", escapeMarkdown(nodeName), escapeMarkdown(nodeIP))
	n.Send(msg)
}

// NotifyNodeOnline sends a node back online notification.
func (n *TelegramSender) NotifyNodeOnline(nodeName, nodeIP string) {
	msg := fmt.Sprintf("🟢 *Node Online*\nNode: `%s`\nIP: %s", escapeMarkdown(nodeName), escapeMarkdown(nodeIP))
	n.Send(msg)
}

func (n *TelegramSender) sendSync(message string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.BotToken)

	payload := map[string]any{
		"chat_id":    n.ChatID,
		"text":       message,
		"parse_mode": "Markdown",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[telegram] marshal error: %v", err)
		return
	}

	resp, err := n.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[telegram] send error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		log.Printf("[telegram] API returned %s for message: %.80s", resp.Status, message)
	}
}

func eventIcon(eventType string) string {
	switch eventType {
	case "payment":
		return "💰"
	case "customer":
		return "👤"
	case "plan":
		return "📋"
	case "session":
		return "🔌"
	case "node", "service":
		return "🖥️"
	case "reseller":
		return "🏪"
	case "account":
		return "🔑"
	default:
		return "ℹ️"
	}
}

func escapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"`", "\\`",
	)
	return replacer.Replace(s)
}
