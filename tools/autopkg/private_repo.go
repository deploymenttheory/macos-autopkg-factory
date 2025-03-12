package autopkg

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// Config holds configuration details for setting up a private AutoPkg repo
type Config struct {
	PrivateRepoPath string // Local path to the private AutoPkg repository
	PrivateRepoURL  string // URL of the private AutoPkg repository
}

// SetupPrivateRepo adds a private AutoPkg repo
func SetupPrivateRepo(config *Config, prefsPath string) error {
	if config.PrivateRepoPath == "" || config.PrivateRepoURL == "" {
		return nil
	}

	// Clone the repo if it doesn't exist
	if _, err := os.Stat(config.PrivateRepoPath); os.IsNotExist(err) {
		cmd := exec.Command("git", "clone", config.PrivateRepoURL, config.PrivateRepoPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to clone private repo: %w", err)
		}
	}

	// Check if RECIPE_REPOS exists in prefs
	cmd := exec.Command("/usr/libexec/PlistBuddy", "-c", "Print :RECIPE_REPOS", prefsPath)
	if err := cmd.Run(); err != nil {
		// Need to create it
		cmd := exec.Command("/usr/libexec/PlistBuddy", "-c", "Add :RECIPE_REPOS dict", prefsPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create RECIPE_REPOS: %w", err)
		}
	}

	// Check if the private repo is already in RECIPE_REPOS
	cmd = exec.Command("/usr/libexec/PlistBuddy", "-c", fmt.Sprintf("Print :RECIPE_REPOS:%s", config.PrivateRepoPath), prefsPath)
	if err := cmd.Run(); err != nil {
		// Need to add it
		cmd := exec.Command("/usr/libexec/PlistBuddy", "-c", fmt.Sprintf("Add :RECIPE_REPOS:%s dict", config.PrivateRepoPath), prefsPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add private repo to RECIPE_REPOS: %w", err)
		}

		cmd = exec.Command("/usr/libexec/PlistBuddy", "-c", fmt.Sprintf("Add :RECIPE_REPOS:%s:URL string %s", config.PrivateRepoPath, config.PrivateRepoURL), prefsPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add private repo URL: %w", err)
		}
	}

	// Check if RECIPE_SEARCH_DIRS exists
	cmd = exec.Command("/usr/libexec/PlistBuddy", "-c", "Print :RECIPE_SEARCH_DIRS", prefsPath)
	if err := cmd.Run(); err != nil {
		// Need to create it
		cmd := exec.Command("/usr/libexec/PlistBuddy", "-c", "Add :RECIPE_SEARCH_DIRS array", prefsPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create RECIPE_SEARCH_DIRS: %w", err)
		}
	}

	// Get current RECIPE_SEARCH_DIRS to check if private repo is already there
	cmd = exec.Command("/usr/libexec/PlistBuddy", "-c", "Print :RECIPE_SEARCH_DIRS", prefsPath)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to read RECIPE_SEARCH_DIRS: %w", err)
	}

	// Check if private repo is already in RECIPE_SEARCH_DIRS
	if !strings.Contains(string(output), config.PrivateRepoPath) {
		cmd := exec.Command("/usr/libexec/PlistBuddy", "-c", fmt.Sprintf("Add :RECIPE_SEARCH_DIRS: string '%s'", config.PrivateRepoPath), prefsPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add private repo to RECIPE_SEARCH_DIRS: %w", err)
		}
	}

	logger.Logger("✅ Private AutoPkg Repo Configured", logger.LogSuccess)
	return nil
}
