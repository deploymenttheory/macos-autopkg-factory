package main

import (
	"log"

	sp "github.com/deploymenttheory/macos-autopkg-factory/tools/suspicious_package"
)

func main() {
	// Create a configuration object
	config := &sp.Config{
		ForceUpdate: true, // Or get this from environment variables
	}

	// Call SetupGitHubActionsRunner with the config and capture both return values
	version, err := sp.InstallSuspiciousPackage(config)
	if err != nil {
		log.Fatalf("Failed to set up Suspicious Package: %v", err)
	}

	log.Printf("Successfully installed Suspicious Package version: %s", version)
}
