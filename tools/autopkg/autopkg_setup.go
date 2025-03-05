package autopkg

import (
	"fmt"
	"os/user"
	"path/filepath"
)

// SetupGitHubActionsRunner configures AutoPkg for use in GitHub Actions
func SetupGitHubActionsRunner() error {
	// Set up logging
	Logger("🚀 Setting up AutoPkg for GitHub Actions...", LogInfo)

	// Load all environment variables
	LoadEnvironmentVariables()

	// Force debug mode in Autopkg for testing
	if !DEBUG {
		DEBUG = true
		Logger("🔍 Debug mode enabled for Autopkg", LogInfo)
	}

	// Check if running as root (which we shouldn't be)
	if err := RootCheck(); err != nil {
		return fmt.Errorf("🚫 root check failed: %w", err)
	}

	// Ensure command line tools are installed
	if err := CheckCommandLineTools(); err != nil {
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
	// Configure contexual mdm uploader settings
	if USE_JAMF_UPLOADER {
		Logger("☁️ Configuring with JamfUploader integration", LogInfo)
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
		Logger("☁️ Configuring with IntuneUploader integration", LogInfo)
		config.INTUNE_TENANT_ID = INTUNE_TENANT_ID
		config.INTUNE_CLIENT_ID = INTUNE_CLIENT_ID
		config.INTUNE_CLIENT_SECRET = INTUNE_CLIENT_SECRET
	}

	// Install or update AutoPkg
	autopkgVersion, err := InstallAutoPkg(config)
	if err != nil {
		return fmt.Errorf("📦 failed to install AutoPkg: %w", err)
	}
	Logger(fmt.Sprintf("📦 AutoPkg %s ready", autopkgVersion), LogSuccess)

	// Set up preferences
	prefsPath, err := SetupPreferencesFile(config)
	if err != nil {
		return fmt.Errorf("⚙️ failed to set up preferences: %w", err)
	}

	// Configure specific uploaders based on what's enabled
	if USE_JAMF_UPLOADER {
		if err := ConfigureJamfUploader(config, prefsPath); err != nil {
			return fmt.Errorf("☁️ failed to configure JamfUploader: %w", err)
		}
		Logger("📱 JamfUploader configuration completed", LogSuccess)
	}

	if USE_INTUNE_UPLOADER {
		if err := ConfigureIntuneUploader(config, prefsPath); err != nil {
			return fmt.Errorf("📱 failed to configure IntuneUploader: %w", err)
		}
		Logger("📱 IntuneUploader configuration completed", LogSuccess)
	}

	// Setup Teams notifications if webhook is provided
	if TEAMS_WEBHOOK != "" {
		Logger("💬 Microsoft Teams notifications configured", LogSuccess)
	}

	// Add repos
	if err := AddAutoPkgRepos(config, prefsPath); err != nil {
		return fmt.Errorf("📚 failed to add repos: %w", err)
	}

	// Process any recipe lists if provided
	if len(RECIPE_LISTS) > 0 {
		config.RecipeLists = RECIPE_LISTS

		if err := ProcessRecipeLists(config, prefsPath); err != nil {
			return fmt.Errorf("📋 failed to process recipe lists: %w", err)
		}
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

	Logger("✅ AutoPkg setup for GitHub Actions completed successfully", LogSuccess)
	return nil
}
