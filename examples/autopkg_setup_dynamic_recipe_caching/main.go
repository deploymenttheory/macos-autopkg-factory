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

	// Define the recipes we want to run
	targetRecipes := []string{
		//"MicrosoftTeams.install.recipe",
		//"MicrosoftTeams.intune.recipe",
		"MicrosoftTeams.jamf.recipe",
		//"MicrosoftTeams.ws1.recipe",
		//"MicrosoftTeamsForWorkOrSchool.pkg.recipe",
		//"MicrosoftTeams.munki.recipe",
		//"MicrosoftTeams2.jamf-upload.recipe.yaml",
	}

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
		// Only include base recipes as a starting point
		RECIPE_REPOS: map[string]interface{}{
			"~/Library/AutoPkg/RecipeRepos/recipes": map[string]string{
				"URL": "https://github.com/autopkg/recipes",
			},
		},
		// Set any additional preferences as needed
		GIT_PATH:                        "/usr/bin/git",
		FAIL_RECIPES_WITHOUT_TRUST_INFO: true,
	}

	// Set up the workflow steps
	orchestrator.
		// Initialization steps
		AddRootCheckStep(false).
		AddGitCheckStep(true).
		AddInstallAutoPkgStep(&autopkg.InstallConfig{
			ForceUpdate: false,
			UseBeta:     false,
		}, true).

		// Configure preferences
		AddSetPreferencesStep(prefsData, true).

		// Add the base autopkg repo to ensure we can run the analysis
		AddRepoAddStep([]string{
			"https://github.com/autopkg/recipes",
		}, true).

		// Analyze recipe dependencies and add required repos
		AddRecipeRepoAnalysisStep(
			targetRecipes,
			&autopkg.RecipeRepoAnalysisOptions{
				IncludeParents:   true,
				MaxDepth:         5,
				VerifyRepoExists: true,
				IncludeBase:      true,
			},
			true, // Add the repos
			true, // Continue on error
		).

		// List and update repositories
		AddRepoListStep(true).

		// Recipe validation and execution
		AddBatchProcessingStep(targetRecipes, &autopkg.RecipeBatchRunOptions{
			VerboseLevel: 5, // Keep verbosity high for debugging
		}, false).

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
