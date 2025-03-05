package main

import (
	"log"

	autopkg "github.com/deploymenttheory/macos-autopkg-factory/tools/suspicious_package"
)

func main() {
	if err := autopkg.SetupGitHubActionsRunner(); err != nil {
		log.Fatalf("Failed to set up AutoPkg: %v", err)
	}
}
