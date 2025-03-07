package main

import (
	"fmt"
	"os"
	"time"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/autopkg"
	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

func main() {
	// Initialize logger
	logger.SetLogLevel(logger.LogDebug)

	// Create a new orchestrator
	orchestrator := autopkg.NewAutoPkgOrchestrator()

	// Configure global options
	orchestrator.
		WithPrefsPath("~/Library/Preferences/com.github.autopkg.plist").
		WithConcurrency(4).
		WithTimeout(30 * time.Minute).
		WithStopOnFirstError(true).
		WithReportFile("autopkg_run_report.txt")

		// Create preferences configuration
	prefsData := &autopkg.PreferencesData{
		RECIPE_SEARCH_DIRS: []string{
			"~/Library/AutoPkg/RecipeRepos",
			"~/Library/AutoPkg/RecipeOverrides",
		},
		// Repository configuration
		RECIPE_REPOS: map[string]interface{}{
			"~/Library/AutoPkg/RecipeRepos/recipes": map[string]string{
				"URL": "https://github.com/autopkg/recipes",
			},
			"~/Library/AutoPkg/RecipeRepos/homebysix-recipes": map[string]string{
				"URL": "https://github.com/homebysix/recipes",
			},
		},
		// Set any additional preferences as needed
		GIT_PATH:                        "/usr/bin/git",
		FAIL_RECIPES_WITHOUT_TRUST_INFO: true,
		// Add Jamf/Intune/other credentials if needed
		// JSS_URL: "https://your.jamf.server",
		// API_USERNAME: "apiuser",
		// API_PASSWORD: "apipassword",
	}
	// Set up the workflow steps
	orchestrator.
		// Initialization steps
		AddRootCheckStep(false).
		AddGitCheckStep(true).
		AddInstallAutoPkgStep(&autopkg.InstallConfig{
			ForceUpdate: true,
			UseBeta:     false,
		}, true).

		// Configure preferences
		AddSetPreferencesStep(prefsData, true).

		// Repository management steps
		AddRepoAddStep([]string{
			"https://github.com/autopkg/recipes",
			"https://github.com/homebysix/recipes",
		}, true).

		// Repository update steps (use the correct repo names after adding)
		AddRepoListStep(true). // List repos to confirm they were added
		AddRepoUpdateStep([]string{
			"recipes",
			"homebysix-recipes",
		}, true).

		// Recipe validation and execution
		AddVerifyStep([]string{
			"Firefox.install",
			"GoogleChrome.install",
			"MicrosoftTeams.install",
		}, nil, true).
		AddParallelRunStep([]string{
			"Firefox.install",
			"GoogleChrome.install",
			"MicrosoftTeams.install",
		}, nil, false).

		// Cleanup
		AddCleanupStep(nil, true)

	// Execute the workflow
	result, err := orchestrator.Execute()
	if err != nil {
		logger.Logger(fmt.Sprintf("Workflow execution failed: %v", err), logger.LogError)
		os.Exit(1)
	}

	// Print workflow summary
	fmt.Printf("\nWorkflow Execution Summary:\n")
	fmt.Printf("- Duration: %s\n", result.ElapsedTime)
	fmt.Printf("- Success: %t\n", result.Success)
	fmt.Printf("- Completed Steps: %d\n", len(result.CompletedSteps))
	fmt.Printf("- Failed Steps: %d\n", len(result.FailedSteps))
	fmt.Printf("- Processed Recipes: %d\n", len(result.ProcessedRecipes))

	// If any steps failed, print the errors
	if len(result.FailedSteps) > 0 {
		fmt.Printf("\nFailed Steps:\n")
		for _, stepName := range result.FailedSteps {
			fmt.Printf("- %s: %v\n", stepName, result.Errors[stepName])
		}
	}
}
