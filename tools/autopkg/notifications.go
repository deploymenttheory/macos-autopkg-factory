package autopkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// NotifyTeams sends a notification to Microsoft Teams
func NotifyTeams(recipe *Recipe, webhookURL string) error {
	if webhookURL == "" {
		return fmt.Errorf("Teams webhook URL not provided")
	}

	name, _ := recipe.Name()

	var statusEmoji string
	var status string
	var themeColor string

	if recipe.Error {
		statusEmoji = "❌"
		status = "Failed"
		themeColor = "FF0000" // Red
	} else if recipe.Updated {
		statusEmoji = "🚀"
		status = "Updated to " + recipe.UpdatedVersion()
		themeColor = "00FF00" // Green
	} else if recipe.Removed {
		statusEmoji = "🧹"
		status = "Cleaned up old versions"
		themeColor = "00FFFF" // Cyan
	} else if recipe.Promoted {
		statusEmoji = "⭐"
		status = "Promoted to production"
		themeColor = "FFDD00" // Gold
	} else {
		statusEmoji = "ℹ️"
		status = "No changes"
		themeColor = "0078D7" // Blue
	}

	// Add emoji to status
	status = statusEmoji + " " + status

	// Create Teams message card
	message := map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "http://schema.org/extensions",
		"summary":    fmt.Sprintf("AutoPkg Recipe: %s", name),
		"themeColor": themeColor,
		"title":      fmt.Sprintf("🧩 AutoPkg Recipe: %s", name),
		"sections": []map[string]interface{}{
			{
				"facts": []map[string]string{
					{"name": "📦 Recipe", "value": name},
					{"name": "📊 Status", "value": status},
					{"name": "🕒 Timestamp", "value": time.Now().Format(time.RFC3339)},
				},
			},
		},
	}

	// Convert message to JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Teams message: %w", err)
	}

	// Send HTTP POST request to Teams webhook
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send Teams notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Teams notification failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
