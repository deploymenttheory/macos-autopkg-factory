package autopkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// RecipeLifecycle represents an AutoPkg recipe and its state.
type RecipeLifecycle struct {
	Name           string                 // Name of the recipe
	Error          bool                   // Indicates if the recipe encountered an error
	Updated        bool                   // Indicates if the recipe was updated
	UpdatedVersion string                 // The new version if updated
	Removed        bool                   // Indicates if old versions were cleaned up
	Promoted       bool                   // Indicates if the recipe was promoted to production
	Verified       *bool                  // Indicates if the recipe passed verification
	Results        map[string]interface{} // Additional details about the recipe execution
}

// SlackNotifier is responsible for sending notifications to Slack.
type SlackNotifier struct {
	WebhookURL string
	Username   string
	Channel    string
	IconEmoji  string
}

// SlackMessage represents the Slack payload.
type SlackMessage struct {
	Username    string            `json:"username,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Attachments []SlackAttachment `json:"attachments"`
}

// SlackAttachment represents a Slack message attachment.
type SlackAttachment struct {
	Title    string   `json:"title"`
	Text     string   `json:"text"`
	Color    string   `json:"color"`
	Markdown []string `json:"mrkdwn_in"`
}

// Notify sends a notification to Slack.
func (s *SlackNotifier) Notify(title, message, color string) error {
	if s.WebhookURL == "" {
		return fmt.Errorf("slack webhook URL not provided")
	}

	payload := SlackMessage{
		Username:  s.Username,
		Channel:   s.Channel,
		IconEmoji: s.IconEmoji,
		Attachments: []SlackAttachment{
			{
				Title:    title,
				Text:     message,
				Color:    color,
				Markdown: []string{"text"},
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	resp, err := http.Post(s.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send Slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack notification failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// NotifySlack sends all relevant notifications for a recipe.
func (s *SlackNotifier) NotifySlack(recipe *RecipeLifecycle) {
	var title, message, color string

	if recipe.Verified != nil && !*recipe.Verified {
		title = fmt.Sprintf("❌ %s failed trust verification", recipe.Name)
		message = "Update trust verification manually."
		color = "warning"
	} else if recipe.Error {
		title = fmt.Sprintf("❌ %s failed", recipe.Name)
		message = "Unknown error"
		if failed, ok := recipe.Results["failed"].([]interface{}); ok && len(failed) > 0 {
			if first, ok := failed[0].(map[string]interface{}); ok {
				if msg, exists := first["message"].(string); exists {
					message = msg
				}
			}
		}
		if strings.Contains(message, "No releases found for repo") {
			return
		}
		color = "danger"
	} else if recipe.Updated {
		title = fmt.Sprintf("✅ Imported %s %s", recipe.Name, recipe.UpdatedVersion)
		message = fmt.Sprintf("**Name:** %s\n\n", recipe.Name)
		if imported, ok := recipe.Results["imported"].([]interface{}); ok && len(imported) > 0 {
			if first, ok := imported[0].(map[string]interface{}); ok {
				if appID, exists := first["intune_app_id"].(string); exists {
					message += fmt.Sprintf("**Intune App ID:** %s\n\n", appID)
				}
				if contentVersion, exists := first["content_version_id"].(string); exists {
					message += fmt.Sprintf("**Content Version ID:** %s\n\n", contentVersion)
				}
			}
		}
		color = "good"
	} else {
		return
	}

	s.Notify(title, message, color)
}
