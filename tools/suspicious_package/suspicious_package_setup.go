package suspiciouspackage

import (
	"fmt"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// SetupGitHubActionsRunner configures Suspicious Package for use in GitHub Actions.
// It checks whether Suspicious Package is already installed, and if not (or if forced),
// downloads, installs, and logs the process.
func SetupGitHubActionsRunner() error {
	// Set up logging.
	logger.Logger("🚀 Setting up Suspicious Package for GitHub Actions...", logger.LogInfo)

	// Load all environment variables.
	LoadEnvironmentVariables()

	// Force debug mode for testing if not enabled.
	if !DEBUG {
		DEBUG = true
		logger.Logger("🔍 Debug mode enabled for Suspicious Package", logger.LogInfo)
	}

	// Use configuration (here we only need FORCE_UPDATE, but you could extend Config as needed)
	config := &Config{
		ForceUpdate: FORCE_UPDATE,
	}

	// Install Suspicious Package if it's not already installed.
	spVersion, err := InstallSuspiciousPackage(config)
	if err != nil {
		return fmt.Errorf("📦 failed to install Suspicious Package: %w", err)
	}
	logger.Logger(fmt.Sprintf("📦 Suspicious Package %s installed", spVersion), logger.LogSuccess)

	// Optionally, you could add additional configuration steps here.

	logger.Logger("✅ Suspicious Package setup for GitHub Actions completed successfully", logger.LogSuccess)
	return nil
}
