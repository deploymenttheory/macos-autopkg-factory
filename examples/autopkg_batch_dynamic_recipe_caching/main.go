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
		"MicrosoftTeams.install.recipe",
		"MicrosoftTeams.intune.recipe",
		"MicrosoftTeams.jamf.recipe",
		"MicrosoftTeams.ws1.recipe",
		"MicrosoftTeamsForWorkOrSchool.pkg.recipe",
		"MicrosoftTeams.munki.recipe",
		"MicrosoftTeams2.jamf-upload.recipe.yaml",
	}

	// Configure global options
	orchestrator.
		WithPrefsPath("~/Library/Preferences/com.github.autopkg.plist").
		WithConcurrency(4).
		WithTimeout(30 * time.Minute).
		WithStopOnFirstError(true).
		WithReportFile("autopkg_run_report.txt")

	// Set up the workflow steps
	orchestrator.
		// Initialization steps
		AddRootCheckStep(false).
		AddGitCheckStep(true).
		AddInstallAutoPkgStep(&autopkg.InstallConfig{
			ForceUpdate: false,
			UseBeta:     false,
		}, true).

		// Add the base autopkg repo
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

		// Cleanup
		AddCleanupStep(nil, true)

	// **Instead of AddParallelRunStep, use RecipeBatchProcessing**
	batchOptions := &autopkg.RecipeBatchOptions{
		PrefsPath:            "~/Library/Preferences/com.github.autopkg.plist",
		MaxConcurrentRecipes: 4, // Max parallel executions
		StopOnFirstError:     true,
		Verbose:              true,
		SearchDirs: []string{
			"~/Library/AutoPkg/RecipeRepos",
			"~/Library/AutoPkg/RecipeOverrides",
		},
	}

	// Run the recipes
	results, err := autopkg.RecipeBatchProcessing(targetRecipes, batchOptions)
	if err != nil {
		logger.Logger(fmt.Sprintf("❌ Recipe batch execution failed: %v", err), logger.LogError)
		os.Exit(1)
	}

	// Print summary
	fmt.Printf("\nBatch Execution Summary:\n")
	successCount, failCount := 0, 0
	for _, result := range results {
		if result.ExecutionError == nil {
			successCount++
		} else {
			failCount++
		}
	}

	fmt.Printf("- ✅ Successful Recipes: %d\n", successCount)
	fmt.Printf("- ❌ Failed Recipes: %d\n", failCount)

	// Exit with failure if any recipe failed
	if failCount > 0 {
		os.Exit(1)
	}

	fmt.Println("✅ All recipes processed successfully!")
}
