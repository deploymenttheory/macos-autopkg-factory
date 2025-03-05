package suspiciouspackage

import (
	"os"
	"strconv"
)

// Global Environment variables for GitHub Actions integration.
var (
	DEBUG        bool
	FORCE_UPDATE bool
)

// LoadEnvironmentVariables loads all environment variables used by the package.
func LoadEnvironmentVariables() {
	// Check if DEBUG is enabled.
	debugEnv := os.Getenv("DEBUG")
	if debugEnv != "" {
		DEBUG, _ = strconv.ParseBool(debugEnv)
	}

	// Check if FORCE_UPDATE is enabled.
	forceUpdateEnv := os.Getenv("FORCE_UPDATE")
	if forceUpdateEnv != "" {
		FORCE_UPDATE, _ = strconv.ParseBool(forceUpdateEnv)
	}
}
