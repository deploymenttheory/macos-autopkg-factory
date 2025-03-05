package suspiciouspackage

import (
	"os"
)

// Global variables to store environment variables
var (
	DEBUG                  bool
	FORCE_UPDATE           bool
	SUSPICIOUS_PACKAGE_URL string
	SP_DOWNLOAD_PATH       string
)

// LoadEnvironmentVariables loads all environment variables needed for Suspicious Package setup
func LoadEnvironmentVariables() {
	// Load debug mode
	DEBUG = os.Getenv("DEBUG") == "true"

	// Load force update setting
	FORCE_UPDATE = os.Getenv("FORCE_UPDATE") == "true"

	// Load Suspicious Package URL with default fallback
	SUSPICIOUS_PACKAGE_URL = os.Getenv("SUSPICIOUS_PACKAGE_URL")
	if SUSPICIOUS_PACKAGE_URL == "" {
		SUSPICIOUS_PACKAGE_URL = "https://mothersruin.com/software/downloads/SuspiciousPackage.dmg"
	}

	// Load download path with default fallback
	SP_DOWNLOAD_PATH = os.Getenv("SP_DOWNLOAD_PATH")
	if SP_DOWNLOAD_PATH == "" {
		SP_DOWNLOAD_PATH = "/tmp/SuspiciousPackage.dmg"
	}
}
