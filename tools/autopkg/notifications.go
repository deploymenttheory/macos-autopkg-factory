package autopkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// NotifyMSTeams sends a notification to Microsoft Teams
func NotifyMSTeams(recipe *Recipe, webhookURL string) error {
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

// NotifySlack sends a notification to Slack using an incoming webhook,
// with a rich message card built in an "attachments" array.
func NotifySlack(recipe *Recipe, slackWebhookURL, slackUsername, slackChannel, slackIconEmoji string) error {
	if slackWebhookURL == "" {
		return fmt.Errorf("Slack webhook URL not provided")
	}

	name, err := recipe.Name()
	if err != nil {
		name = "Unknown Recipe"
	}

	var taskTitle, taskDescription, color string

	// Determine status based on recipe state.
	if recipe.Verified == nil || !*recipe.Verified {
		taskTitle = fmt.Sprintf("%s failed trust verification", name)

		if msg, ok := recipe.Results["message"].(string); ok {
			taskDescription = msg
		} else {
			taskDescription = "No additional message provided."
		}
		color = "warning"
	} else if recipe.Error {
		taskTitle = fmt.Sprintf("Failed to import %s", name)
		// Attempt to extract a failure message from recipe.Results["failed"]
		if failed, ok := recipe.Results["failed"].([]interface{}); ok && len(failed) > 0 {
			if first, ok := failed[0].(map[string]interface{}); ok {
				msg := first["message"]
				trace := first["traceback"]
				taskDescription = fmt.Sprintf("Error: %v\nTraceback: %v", msg, trace)
			}
		} else {
			taskDescription = "Unknown error"
		}
		// If the error message contains a particular phrase, skip notification.
		if strings.Contains(taskDescription, "No releases found for repo") {
			return nil
		}
		color = "danger"
	} else if recipe.Updated {
		taskTitle = fmt.Sprintf("Imported %s %s", name, recipe.UpdatedVersion())
		// Use details from recipe.Results["imported"] if available.
		if imported, ok := recipe.Results["imported"].([]interface{}); ok && len(imported) > 0 {
			if first, ok := imported[0].(map[string]interface{}); ok {
				catalogs, _ := first["catalogs"].(string)
				pkgRepoPath, _ := first["pkg_repo_path"].(string)
				pkginfoPath, _ := first["pkginfo_path"].(string)
				taskDescription = fmt.Sprintf("*Catalogs:* %s\n*Package Path:* `%s`\n*Pkginfo Path:* `%s`",
					catalogs, pkgRepoPath, pkginfoPath)
			}
		}
		color = "good"
	} else {
		// No updates or error – nothing to notify.
		return nil
	}

	// Build the Slack payload using an attachments array.
	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"username":  slackUsername,
				"as_user":   true,
				"title":     taskTitle,
				"color":     color,
				"text":      taskDescription,
				"mrkdwn_in": []string{"text"},
			},
		},
	}

	// Optionally override the channel.
	if slackChannel != "" {
		payload["channel"] = slackChannel
	}
	// Optionally set an icon emoji.
	if slackIconEmoji != "" {
		payload["icon_emoji"] = slackIconEmoji
	}

	// Marshal the payload into JSON.
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	// Send the HTTP POST request to the Slack webhook.
	resp, err := http.Post(slackWebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send Slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Slack notification failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
