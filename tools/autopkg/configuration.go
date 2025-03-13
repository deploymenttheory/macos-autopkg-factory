package autopkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
	"howett.net/plist"
)

// GetAutoPkgPreferences retrieves current plist values.
func GetAutoPkgPreferences(prefsPath string) (map[string]interface{}, error) {
	if prefsPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		prefsPath = filepath.Join(homeDir, "Library/Preferences/com.github.autopkg.plist")
	}

	// Check if plist exists
	if _, err := os.Stat(prefsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("preferences file does not exist: %s", prefsPath)
	}

	// Read the plist
	data, err := os.ReadFile(prefsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read preferences file: %w", err)
	}

	// Parse the plist
	var prefs map[string]interface{}
	if _, err := plist.Unmarshal(data, &prefs); err != nil {
		return nil, fmt.Errorf("failed to parse preferences: %w", err)
	}

	logger.Logger("📖 AutoPkg preferences retrieved successfully", logger.LogInfo)
	return prefs, nil
}

// UpdateAutoPkgPreferences updates the plist with provided key-value pairs.
// Environment variables take precedence over CLI flags.
func UpdateAutoPkgPreferences(prefsPath string, inputValues map[string]interface{}) error {
	if prefsPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		prefsPath = filepath.Join(homeDir, "Library/Preferences/com.github.autopkg.plist")
	}

	// Load existing plist
	var prefs map[string]interface{}
	if _, err := os.Stat(prefsPath); err == nil {
		data, err := os.ReadFile(prefsPath)
		if err != nil {
			return fmt.Errorf("failed to read preferences file: %w", err)
		}
		if _, err := plist.Unmarshal(data, &prefs); err != nil {
			return fmt.Errorf("failed to parse plist: %w", err)
		}
	} else {
		prefs = make(map[string]interface{})
	}

	// Merge input values, preferring environment variables
	for key, value := range inputValues {
		if envValue, found := os.LookupEnv(strings.ToUpper(strings.ReplaceAll(key, "-", "_"))); found {
			logger.Logger(fmt.Sprintf("🔄 Using environment variable for %s", key), logger.LogInfo)
			prefs[key] = envValue
		} else {
			prefs[key] = value
		}
	}

	// Save updated preferences
	data, err := plist.MarshalIndent(prefs, plist.XMLFormat, "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plist: %w", err)
	}
	if err := os.WriteFile(prefsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write preferences file: %w", err)
	}

	logger.Logger("✅ AutoPkg preferences updated successfully", logger.LogSuccess)
	return nil
}
