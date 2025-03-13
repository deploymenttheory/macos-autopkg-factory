// configuration.go contains functions for configuing autopkg settings
package autopkg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
	"howett.net/plist"
)

// PreferencesData represents the structure of AutoPkg preferences
type PreferencesData struct {
	RECIPE_REPOS                    map[string]interface{} `plist:"RECIPE_REPOS,omitempty" json:"recipe_repos,omitempty"`
	RECIPE_SEARCH_DIRS              []string               `plist:"RECIPE_SEARCH_DIRS,omitempty" json:"recipe_search_dirs,omitempty"`
	GITHUB_TOKEN_PATH               string                 `plist:"GITHUB_TOKEN_PATH,omitempty" json:"github_token_path,omitempty"`
	GIT_PATH                        string                 `plist:"GIT_PATH,omitempty" json:"git_path,omitempty"`
	JSS_URL                         string                 `plist:"JSS_URL,omitempty" json:"jss_url,omitempty"`
	API_USERNAME                    string                 `plist:"API_USERNAME,omitempty" json:"api_username,omitempty"`
	API_PASSWORD                    string                 `plist:"API_PASSWORD,omitempty" json:"api_password,omitempty"`
	CLIENT_ID                       string                 `plist:"CLIENT_ID,omitempty" json:"client_id,omitempty"`
	CLIENT_SECRET                   string                 `plist:"CLIENT_SECRET,omitempty" json:"client_secret,omitempty"`
	SMB_URL                         string                 `plist:"SMB_URL,omitempty" json:"smb_url,omitempty"`
	SMB_USERNAME                    string                 `plist:"SMB_USERNAME,omitempty" json:"smb_username,omitempty"`
	SMB_PASSWORD                    string                 `plist:"SMB_PASSWORD,omitempty" json:"smb_password,omitempty"`
	TENANT_ID                       string                 `plist:"TENANT_ID,omitempty" json:"tenant_id,omitempty"`
	SLACK_WEBHOOK                   string                 `plist:"SLACK_WEBHOOK,omitempty" json:"slack_webhook,omitempty"`
	SLACK_USERNAME                  string                 `plist:"SLACK_USERNAME,omitempty" json:"slack_username,omitempty"`
	JCDS2_MODE                      bool                   `plist:"jcds2_mode,omitempty" json:"jcds2_mode,omitempty"`
	FAIL_RECIPES_WITHOUT_TRUST_INFO bool                   `plist:"FAIL_RECIPES_WITHOUT_TRUST_INFO,omitempty" json:"fail_recipes_without_trust_info,omitempty"`
	AdditionalPreferences           map[string]interface{} `plist:"-" json:"additional_preferences,omitempty"`
}

// GetAutoPkgPreferences reads and parses the current autopkg preferences
func GetAutoPkgPreferences(prefsPath string) (*PreferencesData, error) {
	if prefsPath == "" {
		// Use default preferences path
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		prefsPath = filepath.Join(homeDir, "Library/Preferences/com.github.autopkg.plist")
	}

	// Check if the preferences file exists
	if _, err := os.Stat(prefsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("preferences file does not exist: %s", prefsPath)
	}

	// Read the preferences file
	data, err := os.ReadFile(prefsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read preferences file: %w", err)
	}

	// Parse the preferences
	var prefs PreferencesData
	if _, err := plist.Unmarshal(data, &prefs); err != nil {
		return nil, fmt.Errorf("failed to parse preferences: %w", err)
	}

	// Also run the defaults command to get all keys
	cmd := exec.Command("defaults", "read", prefsPath)
	output, err := cmd.Output()
	if err != nil {
		logger.Logger(fmt.Sprintf("⚠️ Could not read all preferences with defaults command: %v", err), logger.LogWarning)
	} else {
		// Parse the output to extract keys that weren't in our struct
		lines := strings.Split(string(output), "\n")
		prefs.AdditionalPreferences = make(map[string]interface{})

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "\"") || strings.HasPrefix(line, "{") || strings.HasPrefix(line, "(") {
				continue
			}

			parts := strings.SplitN(line, " = ", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				// Add keys that weren't in our struct
				if !isStandardPrefKey(key) {
					prefs.AdditionalPreferences[key] = value
				}
			}
		}
	}

	logger.Logger(fmt.Sprintf("📝 Read AutoPkg preferences from %s", prefsPath), logger.LogInfo)
	return &prefs, nil
}

// isStandardPrefKey checks if a key is one of the standard preferences we already parse
func isStandardPrefKey(key string) bool {
	standardKeys := []string{
		"RECIPE_REPOS", "RECIPE_SEARCH_DIRS", "GITHUB_TOKEN_PATH", "GIT_PATH",
		"JSS_URL", "API_USERNAME", "API_PASSWORD", "CLIENT_ID", "CLIENT_SECRET",
		"SMB_URL", "SMB_USERNAME", "SMB_PASSWORD", "TENANT_ID",
		"SLACK_WEBHOOK", "SLACK_USERNAME", "jcds2_mode", "FAIL_RECIPES_WITHOUT_TRUST_INFO",
	}

	for _, stdKey := range standardKeys {
		if key == stdKey {
			return true
		}
	}
	return false
}

// SetAutoPkgPreferences writes specific preferences to the autopkg preferences file
func SetAutoPkgPreferences(prefsPath string, prefs *PreferencesData) error {
	if prefsPath == "" {
		// Use default preferences path
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		prefsPath = filepath.Join(homeDir, "Library/Preferences/com.github.autopkg.plist")
	}

	// We'll use the defaults command to write preferences
	logger.Logger(fmt.Sprintf("📝 Writing AutoPkg preferences to %s", prefsPath), logger.LogInfo)

	// Write standard string preferences
	stringPrefs := map[string]string{
		"GIT_PATH":          prefs.GIT_PATH,
		"GITHUB_TOKEN_PATH": prefs.GITHUB_TOKEN_PATH,
		"JSS_URL":           prefs.JSS_URL,
		"API_USERNAME":      prefs.API_USERNAME,
		"API_PASSWORD":      prefs.API_PASSWORD,
		"CLIENT_ID":         prefs.CLIENT_ID,
		"CLIENT_SECRET":     prefs.CLIENT_SECRET,
		"SMB_URL":           prefs.SMB_URL,
		"SMB_USERNAME":      prefs.SMB_USERNAME,
		"SMB_PASSWORD":      prefs.SMB_PASSWORD,
		"TENANT_ID":         prefs.TENANT_ID,
		"SLACK_WEBHOOK":     prefs.SLACK_WEBHOOK,
		"SLACK_USERNAME":    prefs.SLACK_USERNAME,
	}

	for key, value := range stringPrefs {
		if value != "" {
			cmd := exec.Command("defaults", "write", prefsPath, key, value)
			if err := cmd.Run(); err != nil {
				logger.Logger(fmt.Sprintf("⚠️ Failed to write preference %s: %v", key, err), logger.LogWarning)
			}
		}
	}

	// Write boolean preferences
	boolPrefs := map[string]bool{
		"FAIL_RECIPES_WITHOUT_TRUST_INFO": prefs.FAIL_RECIPES_WITHOUT_TRUST_INFO,
		"jcds2_mode":                      prefs.JCDS2_MODE,
	}

	for key, value := range boolPrefs {
		boolValue := "false"
		if value {
			boolValue = "true"
		}
		cmd := exec.Command("defaults", "write", prefsPath, key, "-bool", boolValue)
		if err := cmd.Run(); err != nil {
			logger.Logger(fmt.Sprintf("⚠️ Failed to write preference %s: %v", key, err), logger.LogWarning)
		}
	}

	// Write additional preferences if present
	for key, value := range prefs.AdditionalPreferences {
		switch v := value.(type) {
		case string:
			cmd := exec.Command("defaults", "write", prefsPath, key, v)
			if err := cmd.Run(); err != nil {
				logger.Logger(fmt.Sprintf("⚠️ Failed to write additional preference %s: %v", key, err), logger.LogWarning)
			}
		case bool:
			boolValue := "false"
			if v {
				boolValue = "true"
			}
			cmd := exec.Command("defaults", "write", prefsPath, key, "-bool", boolValue)
			if err := cmd.Run(); err != nil {
				logger.Logger(fmt.Sprintf("⚠️ Failed to write additional preference %s: %v", key, err), logger.LogWarning)
			}
		case int:
			cmd := exec.Command("defaults", "write", prefsPath, key, "-int", fmt.Sprintf("%d", v))
			if err := cmd.Run(); err != nil {
				logger.Logger(fmt.Sprintf("⚠️ Failed to write additional preference %s: %v", key, err), logger.LogWarning)
			}
		case float64:
			cmd := exec.Command("defaults", "write", prefsPath, key, "-float", fmt.Sprintf("%f", v))
			if err := cmd.Run(); err != nil {
				logger.Logger(fmt.Sprintf("⚠️ Failed to write additional preference %s: %v", key, err), logger.LogWarning)
			}
		default:
			logger.Logger(fmt.Sprintf("⚠️ Skipping additional preference %s with unsupported type %T", key, value), logger.LogWarning)
		}
	}

	// Handle RECIPE_SEARCH_DIRS array
	if len(prefs.RECIPE_SEARCH_DIRS) > 0 {
		// First check if it exists
		checkCmd := exec.Command("defaults", "read", prefsPath, "RECIPE_SEARCH_DIRS")
		if err := checkCmd.Run(); err != nil {
			// Need to create it
			cmd := exec.Command("defaults", "write", prefsPath, "RECIPE_SEARCH_DIRS", "-array")
			if err := cmd.Run(); err != nil {
				logger.Logger("⚠️ Failed to create RECIPE_SEARCH_DIRS array", logger.LogWarning)
			}
		}

		// Clear existing array
		cmd := exec.Command("defaults", "delete", prefsPath, "RECIPE_SEARCH_DIRS")
		_ = cmd.Run() // Ignore errors if it doesn't exist

		// Create array
		cmd = exec.Command("defaults", "write", prefsPath, "RECIPE_SEARCH_DIRS", "-array")
		if err := cmd.Run(); err != nil {
			logger.Logger("⚠️ Failed to create RECIPE_SEARCH_DIRS array", logger.LogWarning)
		}

		// Add items
		for _, dir := range prefs.RECIPE_SEARCH_DIRS {
			cmd := exec.Command("defaults", "write", prefsPath, "RECIPE_SEARCH_DIRS", "-array-add", dir)
			if err := cmd.Run(); err != nil {
				logger.Logger(fmt.Sprintf("⚠️ Failed to add %s to RECIPE_SEARCH_DIRS: %v", dir, err), logger.LogWarning)
			}
		}
	}

	// RECIPE_REPOS is complex and requires PlistBuddy
	if prefs.RECIPE_REPOS != nil && len(prefs.RECIPE_REPOS) > 0 {
		// Check if RECIPE_REPOS exists
		checkCmd := exec.Command("/usr/libexec/PlistBuddy", "-c", "Print :RECIPE_REPOS", prefsPath)
		if err := checkCmd.Run(); err != nil {
			// Need to create it
			cmd := exec.Command("/usr/libexec/PlistBuddy", "-c", "Add :RECIPE_REPOS dict", prefsPath)
			if err := cmd.Run(); err != nil {
				logger.Logger("⚠️ Failed to create RECIPE_REPOS dictionary", logger.LogWarning)
			}
		}

		// Add or update each repo
		for repoPath, repoData := range prefs.RECIPE_REPOS {
			if repoDict, ok := repoData.(map[string]interface{}); ok {
				// Check if repo exists
				checkCmd := exec.Command("/usr/libexec/PlistBuddy", "-c", fmt.Sprintf("Print :RECIPE_REPOS:%s", repoPath), prefsPath)
				if err := checkCmd.Run(); err != nil {
					// Need to add it
					cmd := exec.Command("/usr/libexec/PlistBuddy", "-c", fmt.Sprintf("Add :RECIPE_REPOS:%s dict", repoPath), prefsPath)
					if err := cmd.Run(); err != nil {
						logger.Logger(fmt.Sprintf("⚠️ Failed to add repo %s: %v", repoPath, err), logger.LogWarning)
						continue
					}
				}

				// Add or update repo attributes
				for key, value := range repoDict {
					if strValue, ok := value.(string); ok {
						cmd := exec.Command("/usr/libexec/PlistBuddy", "-c",
							fmt.Sprintf("Set :RECIPE_REPOS:%s:%s %s", repoPath, key, strValue), prefsPath)
						if err := cmd.Run(); err != nil {
							logger.Logger(fmt.Sprintf("⚠️ Failed to set %s:%s: %v", repoPath, key, err), logger.LogWarning)
						}
					}
				}
			}
		}
	}

	cmd := exec.Command("defaults", "read", prefsPath)
	output, err := cmd.Output()
	if err != nil {
		logger.Logger(fmt.Sprintf("⚠️ Failed to read final preferences for debug output: %v", err), logger.LogWarning)
	} else {
		logger.Logger(fmt.Sprintf("🔍 Debug: AutoPkg preferences file at %s contains:\n%s", prefsPath, string(output)), logger.LogDebug)
	}

	logger.Logger("✅ AutoPkg preferences updated successfully", logger.LogSuccess)
	return nil
}
