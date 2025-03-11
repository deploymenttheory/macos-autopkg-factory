package autopkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// MSTeamsNotifier is responsible for sending notifications to Microsoft Teams.
type MSTeamsNotifier struct {
	WebhookURL string
	JSSURL     string // Jamf Pro URL from preferences
}

// TeamsMessage represents the Adaptive Card payload.
type TeamsMessage struct {
	Type        string            `json:"type"`
	Attachments []TeamsAttachment `json:"attachments"`
}

// TeamsAttachment represents the attachment structure.
type TeamsAttachment struct {
	ContentType string       `json:"contentType"`
	ContentURL  *string      `json:"contentUrl,omitempty"`
	Content     AdaptiveCard `json:"content"`
}

// AdaptiveCard represents the Microsoft Adaptive Card.
type AdaptiveCard struct {
	Schema  string            `json:"$schema"`
	Type    string            `json:"type"`
	Version string            `json:"version"`
	MSTeams map[string]string `json:"msteams"`
	Body    []interface{}     `json:"body"`
	Actions []TeamsAction     `json:"actions,omitempty"`
}

// TeamsAction represents an action button in Teams.
type TeamsAction struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// Notify sends a notification to Microsoft Teams.
func (n *MSTeamsNotifier) NotifyMSTeams(title, message string, error, imported bool, appID, jamfPkgID string) error {
	if n.WebhookURL == "" {
		return fmt.Errorf("Teams webhook URL not provided")
	}

	// Define notification style
	style := "good"
	if error {
		style = "attention"
	}

	// Construct the Adaptive Card message.
	data := TeamsMessage{
		Type: "message",
		Attachments: []TeamsAttachment{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				Content: AdaptiveCard{
					Schema:  "http://adaptivecards.io/schemas/adaptive-card.json",
					Type:    "AdaptiveCard",
					Version: "1.6",
					MSTeams: map[string]string{"width": "Full"},
					Body: []interface{}{
						map[string]interface{}{
							"type":  "Container",
							"style": style,
							"bleed": true,
							"size":  "stretch",
							"items": []map[string]interface{}{
								{"type": "TextBlock", "text": ""},
							},
						},
						map[string]interface{}{
							"type": "TextBlock",
							"text": "📦 AutoPkg",
							"wrap": true,
							"size": "large",
						},
						map[string]interface{}{
							"type": "ColumnSet",
							"columns": []map[string]interface{}{
								{
									"type": "Column",
									"items": []map[string]interface{}{
										{"type": "TextBlock", "text": title, "wrap": true, "isSubtle": true},
										{"type": "TextBlock", "text": message, "wrap": true},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Add an Intune action button if applicable
	if imported {
		action := TeamsAction{
			Type:  "Action.OpenUrl",
			Title: "View App in Intune",
			URL:   fmt.Sprintf("https://intune.microsoft.com/#view/Microsoft_Intune_Apps/SettingsMenu/~/0/appId/%s", appID),
		}
		data.Attachments[0].Content.Actions = append(data.Attachments[0].Content.Actions, action)
	}

	// Add a Jamf Pro action button if applicable
	if jamfPkgID != "" && n.JSSURL != "" {
		jamfURL := fmt.Sprintf("%s/view/settings/computer-management/packages/%s?tab=general", n.JSSURL, jamfPkgID)
		action := TeamsAction{
			Type:  "Action.OpenUrl",
			Title: "View Package in Jamf Pro",
			URL:   jamfURL,
		}
		data.Attachments[0].Content.Actions = append(data.Attachments[0].Content.Actions, action)
	}

	// Convert to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal Teams message: %w", err)
	}

	// Send HTTP POST request
	resp, err := http.Post(n.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send Teams notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Teams notification failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// NotifyTeams sends all relevant notifications for a recipe.
func (n *MSTeamsNotifier) NotifyTeams(recipe *RecipeLifecycle, opts *RecipeBatchRunOptions) {
	var jamfPkgID, appID string
	if pkgInfo, ok := recipe.Results["jamf"].(map[string]interface{}); ok {
		if pkgID, exists := pkgInfo["package_id"].(string); exists {
			jamfPkgID = pkgID
		}
	}

	if imported, ok := recipe.Results["imported"].([]interface{}); ok && len(imported) > 0 {
		if first, ok := imported[0].(map[string]interface{}); ok {
			appID, _ = first["intune_app_id"].(string)
		}
	}

	if recipe.Verified != nil && !*recipe.Verified {
		n.NotifyMSTeams(fmt.Sprintf("❌ %s failed trust verification", recipe.Name), "Update trust verification manually", true, false, "", jamfPkgID)
	} else if recipe.Error {
		message := "Unknown error"
		if failed, ok := recipe.Results["failed"].([]interface{}); ok && len(failed) > 0 {
			if first, ok := failed[0].(map[string]interface{}); ok {
				if msg, exists := first["message"].(string); exists {
					message = msg
				}
			}
		}
		n.NotifyMSTeams(fmt.Sprintf("❌ %s failed", recipe.Name), message, true, false, "", jamfPkgID)
	}

	if recipe.Updated {
		title := fmt.Sprintf("✅ Imported %s %s", recipe.Name, recipe.UpdatedVersion)
		message := fmt.Sprintf("**Name:** %s\r\n\r\n", recipe.Name)

		// Include Intune App ID if available
		if appID != "" {
			message += fmt.Sprintf("**Intune App ID:** %s\r\n\r\n", appID)
		}

		// Include Jamf Package ID if available
		if jamfPkgID != "" {
			message += fmt.Sprintf("**Jamf Package ID:** %s\r\n\r\n", jamfPkgID)
		}

		n.NotifyMSTeams(title, message, false, true, appID, jamfPkgID)
	}
}

// ErrorAlerts sends error-related packaging alerts to ms teams.
func (n *MSTeamsNotifier) ErrorAlerts(recipe *RecipeLifecycle) {
	if recipe.Verified != nil && !*recipe.Verified {
		title := fmt.Sprintf("❌ %s failed trust verification", recipe.Name)
		message := "Update trust verification manually"
		n.NotifyMSTeams(title, message, true, false, "", "")
	} else if recipe.Error {
		title := fmt.Sprintf("❌ %s failed", recipe.Name)
		message := "Unknown error"
		if failed, ok := recipe.Results["failed"].([]interface{}); ok && len(failed) > 0 {
			if first, ok := failed[0].(map[string]interface{}); ok {
				if msg, exists := first["message"].(string); exists {
					message = msg
				}
			}
		}
		n.NotifyMSTeams(title, message, true, false, "", "")
	}
}

// UpdatedAlerts sends mdm service package update-related alerts to ms teams.
func (n *MSTeamsNotifier) UpdatedAlerts(recipe *RecipeLifecycle) {
	if recipe.Updated {
		title := fmt.Sprintf("✅ Imported %s %s", recipe.Name, recipe.UpdatedVersion)
		var message, appID, jamfPkgID string

		// Extract Intune App ID and Content Version if available
		if imported, ok := recipe.Results["imported"].([]interface{}); ok && len(imported) > 0 {
			if first, ok := imported[0].(map[string]interface{}); ok {
				appID, _ = first["intune_app_id"].(string)
				contentVersion, _ := first["content_version_id"].(string)
				recipeName, _ := first["name"].(string)

				message = fmt.Sprintf(
					"**Name:** %s\r\n\r\n**Intune App ID:** %s\r\n\r\n**Content Version ID:** %s\r\n\r\n",
					recipeName, appID, contentVersion,
				)
			}
		}

		// Extract Jamf Package ID if available
		if pkgInfo, ok := recipe.Results["jamf"].(map[string]interface{}); ok {
			if pkgID, exists := pkgInfo["package_id"].(string); exists {
				jamfPkgID = pkgID
				message += fmt.Sprintf("**Jamf Package ID:** %s\r\n\r\n", jamfPkgID)
			}
		}

		n.NotifyMSTeams(title, message, false, true, appID, jamfPkgID)
	}
}

// RemovedAlerts sends mdm service cleanup-related alerts to ms teams.
func (n *MSTeamsNotifier) RemovedAlerts(recipe *RecipeLifecycle, opts *RecipeBatchRunOptions) {
	if opts != nil && opts.Notification.EnableTeams && recipe.Removed {
		title := fmt.Sprintf("🗑 Removed old versions of %s", recipe.Name)
		var message, jamfPkgID string

		// Extract removal details
		if removed, ok := recipe.Results["removed"].([]interface{}); ok && len(removed) > 0 {
			if first, ok := removed[0].(map[string]interface{}); ok {
				removedCount := first["removed count"]
				removedVersions := first["removed versions"]
				keepCount := first["keep count"]
				message = fmt.Sprintf(
					"**Remove Count:** %v\r\n\r\n**Removed Versions:** %v\r\n\r\n**Keep Count:** %v\r\n\r\n",
					removedCount, removedVersions, keepCount,
				)
			}
		}

		// Extract Jamf Package ID if available
		if pkgInfo, ok := recipe.Results["jamf"].(map[string]interface{}); ok {
			if pkgID, exists := pkgInfo["package_id"].(string); exists {
				jamfPkgID = pkgID
				message += fmt.Sprintf("**Jamf Package ID:** %s\r\n\r\n", jamfPkgID)
			}
		}

		n.NotifyMSTeams(title, message, false, false, "", jamfPkgID)
	}
}

// PromotedAlerts sends promotion-related alerts to ms teams.
func (n *MSTeamsNotifier) PromotedAlerts(recipe *RecipeLifecycle, opts *RecipeBatchRunOptions) {
	if opts != nil && opts.Notification.EnableTeams && recipe.Promoted {
		title := fmt.Sprintf("🚀 Promoted %s", recipe.Name)
		var message string
		if promoted, ok := recipe.Results["promoted"].([]interface{}); ok && len(promoted) > 0 {
			if first, ok := promoted[0].(map[string]interface{}); ok {
				promotions := first["promotions"]
				blacklistedVersions := first["blacklisted versions"]
				message = fmt.Sprintf(
					"**Promotions:** %v\r\n\r\n**Blacklisted Versions:** %v",
					promotions, blacklistedVersions,
				)
			}
		}
		n.NotifyMSTeams(title, message, false, false, "", "")
	}
}
