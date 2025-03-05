package main

import (
	"log"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/autopkg"
)

func main() {
	if err := autopkg.SetupGitHubActionsRunner(); err != nil {
		log.Fatalf("Failed to set up AutoPkg: %v", err)
	}
}
