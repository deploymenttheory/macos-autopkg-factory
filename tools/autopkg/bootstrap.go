package autopkg

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/helpers"
	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// SetupGitHubActionsRunner configures AutoPkg for use in GitHub Actions
func SetupGitHubActionsRunner() error {
	// Set up logging
	logger.Logger("🚀 Setting up AutoPkg for GitHub Actions...", logger.LogInfo)

	// Load all environment variables
	LoadEnvironmentVariables()

	// Force debug mode in Autopkg for testing
	if !DEBUG {
		DEBUG = true
		logger.Logger("🔍 Debug mode enabled for Autopkg", logger.LogInfo)
	}

	// Check if running as root (which we shouldn't be)
	if err := RootCheck(); err != nil {
		return fmt.Errorf("🚫 root check failed: %w", err)
	}

	// Ensure git is installed
	if err := CheckGit(); err != nil {
		return fmt.Errorf("🛠️ command line tools check failed: %w", err)
	}

	// Set up common configuration using env vars
	config := &Config{
		ForceUpdate:         FORCE_UPDATE,
		FailRecipes:         FAIL_RECIPES,
		DisableVerification: DISABLE_VERIFICATION,
		UseBeta:             USE_BETA,
		AutopkgRepoListPath: AUTOPKG_REPO_LIST_PATH,
	}

	// Configure contextual MDM uploader settings
	if USE_JAMF_UPLOADER {
		logger.Logger("☁️ Configuring with JamfUploader integration", logger.LogInfo)
		config.UseJamfUploader = true
		config.JAMFPRO_URL = JAMFPRO_URL
		config.API_USERNAME = API_USERNAME
		config.API_PASSWORD = API_PASSWORD
		config.JAMFPRO_CLIENT_ID = JAMFPRO_CLIENT_ID
		config.JAMFPRO_CLIENT_SECRET = JAMFPRO_CLIENT_SECRET
		config.SMB_URL = SMB_URL
		config.SMB_USERNAME = SMB_USERNAME
		config.SMB_PASSWORD = SMB_PASSWORD
		config.JCDS2Mode = JCDS2_MODE
	}

	if USE_INTUNE_UPLOADER {
		logger.Logger("☁️ Configuring with IntuneUploader integration", logger.LogInfo)
		config.INTUNE_TENANT_ID = INTUNE_TENANT_ID
		config.INTUNE_CLIENT_ID = INTUNE_CLIENT_ID
		config.INTUNE_CLIENT_SECRET = INTUNE_CLIENT_SECRET
	}

	// Install or update AutoPkg
	autopkgVersion, err := InstallAutoPkg(config)
	if err != nil {
		return fmt.Errorf("📦 failed to install AutoPkg: %w", err)
	}
	logger.Logger(fmt.Sprintf("📦 AutoPkg %s ready", autopkgVersion), logger.LogSuccess)

	// Set up preferences
	prefsPath, err := SetupPreferencesFile(config)
	if err != nil {
		return fmt.Errorf("⚙️ failed to set up preferences: %w", err)
	}

	// Setup Teams notifications if webhook is provided
	if TEAMS_WEBHOOK != "" {
		logger.Logger("💬 Microsoft Teams notifications configured", logger.LogSuccess)
	}

	// Add default repos and repos from config
	var repos []string
	if USE_JAMF_UPLOADER {
		repos = []string{"recipes", "grahampugh/jamf-upload"}
	} else if USE_INTUNE_UPLOADER {
		repos = []string{"recipes", "almenscorner/autopkg-recipes"}
	} else {
		repos = []string{"recipes"}
	}

	// Load additional repos from repo list file if specified
	if config.AutopkgRepoListPath != "" && helpers.FileExists(config.AutopkgRepoListPath) {
		file, err := os.Open(config.AutopkgRepoListPath)
		if err == nil {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				repo := strings.TrimSpace(scanner.Text())
				if repo != "" {
					repos = append(repos, repo)
				}
			}
		} else {
			logger.Logger(fmt.Sprintf("⚠️ Warning: Could not read repo list file: %v", err), logger.LogWarning)
		}
	}

	// Add repositories using AddRepo function
	if err := AddRepo(repos, prefsPath); err != nil {
		return fmt.Errorf("📚 failed to add repos: %w", err)
	}

	// Process any recipe lists if provided
	if len(RECIPE_LISTS) > 0 {
		for _, listPath := range RECIPE_LISTS {
			if !helpers.FileExists(listPath) {
				logger.Logger(fmt.Sprintf("⚠️ Recipe list file %s does not exist", listPath), logger.LogWarning)
				continue
			}

			file, err := os.Open(listPath)
			if err != nil {
				return fmt.Errorf("📋 failed to open recipe list file: %w", err)
			}

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				recipe := strings.TrimSpace(scanner.Text())
				if recipe == "" {
					continue
				}

				// Get recipe info with pull option to ensure parent repos are added
				infoOptions := &InfoOptions{
					PrefsPath: prefsPath,
					Pull:      true,
				}

				if err := GetRecipeInfo(recipe, infoOptions); err != nil {
					logger.Logger(fmt.Sprintf("⚠️ Failed to process recipe %s: %v", recipe, err), logger.LogWarning)
				}
			}

			file.Close()
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("📋 failed to read recipe list file: %w", err)
			}
		}
		logger.Logger("✅ Recipe lists processed successfully", logger.LogSuccess)
	}

	// Set up private repo if provided
	if PRIVATE_REPO_URL != "" {
		config.PrivateRepoURL = PRIVATE_REPO_URL
		config.PrivateRepoPath = PRIVATE_REPO_PATH
		if config.PrivateRepoPath == "" {
			// Set default path if not provided
			usr, _ := user.Current()
			config.PrivateRepoPath = filepath.Join(usr.HomeDir, "Library/AutoPkg/RecipeRepos/private-repo")
		}

		if err := SetupPrivateRepo(config, prefsPath); err != nil {
			return fmt.Errorf("🔒 failed to set up private repo: %w", err)
		}
	}

	logger.Logger("✅ AutoPkg setup for GitHub Actions completed successfully", logger.LogSuccess)
	return nil
}
