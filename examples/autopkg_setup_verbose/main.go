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
			ForceUpdate: false,
			UseBeta:     false,
		}, true).

		// Configure preferences
		AddSetPreferencesStep(prefsData, true).

		// Repository management steps
		AddRepoAddStep([]string{
			"https://github.com/autopkg/recipes",
			"https://github.com/autopkg/homebysix-recipes",
			"https://github.com/autopkg/ahousseini-recipes",
			"https://github.com/autopkg/andredb90-recipes",
			"https://github.com/autopkg/andrewvalentine-recipes",
			"https://github.com/autopkg/apizz-recipes",
			"https://github.com/autopkg/blackthroat-recipes",
			"https://github.com/autopkg/bochoven-recipes",
			"https://github.com/autopkg/cgerke-recipes",
			"https://github.com/autopkg/crystalllized-recipes",
			"https://github.com/autopkg/dataJAR-recipes",
			"https://github.com/autopkg/drewdiver-recipes",
			"https://github.com/eth-its/autopkg-mac-recipes-yaml",
			"https://github.com/autopkg/faumac-recipes",
			"https://github.com/autopkg/gerardkok-recipes",
			"https://github.com/autopkg/grahampugh-recipes",
			"https://github.com/autopkg/hansen-m-recipes",
			"https://github.com/autopkg/hjuutilainen-recipes",
			"https://github.com/autopkg/its-unibas-recipes",
			"https://github.com/autopkg/jbaker10-recipes",
			"https://github.com/autopkg/jlehikoinen-recipes",
			"https://github.com/autopkg/joshua-d-miller-recipes",
			"https://github.com/autopkg/justinrummel-recipes",
			"https://github.com/autopkg/keeleysam-recipes",
			"https://github.com/autopkg/killahquam-recipes",
			"https://github.com/autopkg/MLBZ521-recipes",
			"https://github.com/autopkg/moofit-recipes",
			"https://github.com/autopkg/n8felton-recipes",
			"https://github.com/autopkg/neilmartin83-recipes",
			"https://github.com/autopkg/nmcspadden-recipes",
			"https://github.com/autopkg/novaksam-recipes",
			"https://github.com/autopkg/nstrauss-recipes",
			"https://github.com/autopkg/nzmacgeek-recipes",
			"https://github.com/autopkg/paul-cossey-recipes",
			"https://github.com/autopkg/peshay-recipes",
			"https://github.com/autopkg/peterkelm-recipes",
			"https://github.com/autopkg/precursorca-recipes",
			"https://github.com/autopkg/rtrouton-recipes",
			"https://github.com/autopkg/scriptingosx-recipes",
			"https://github.com/autopkg/sebtomasi-recipes",
			"https://github.com/autopkg/smithjw-recipes",
			"https://github.com/autopkg/tbridge-recipes",
			"https://github.com/autopkg/wardsparadox-recipes",
			"https://github.com/autopkg/kevinmcox-recipes",
			"https://github.com/autopkg/swy-recipes",
		}, true).

		// Repository update steps (use the correct repo names after adding)
		AddRepoListStep(true). // List repos to confirm they were added
		AddRepoUpdateStep([]string{
			"recipes",
			"homebysix-recipes",
			"ahousseini-recipes",
			"andredb90-recipes",
			"andrewvalentine-recipes",
			"apizz-recipes",
			"blackthroat-recipes",
			"bochoven-recipes",
			"cgerke-recipes",
			"crystalllized-recipes",
			"dataJAR-recipes",
			"drewdiver-recipes",
			"autopkg-mac-recipes-yaml",
			"faumac-recipes",
			"gerardkok-recipes",
			"grahampugh-recipes",
			"hansen-m-recipes",
			"hjuutilainen-recipes",
			"its-unibas-recipes",
			"jbaker10-recipes",
			"jlehikoinen-recipes",
			"joshua-d-miller-recipes",
			"justinrummel-recipes",
			"keeleysam-recipes",
			"killahquam-recipes",
			"MLBZ521-recipes",
			"moofit-recipes",
			"n8felton-recipes",
			"neilmartin83-recipes",
			"nmcspadden-recipes",
			"novaksam-recipes",
			"nstrauss-recipes",
			"nzmacgeek-recipes",
			"paul-cossey-recipes",
			"peshay-recipes",
			"peterkelm-recipes",
			"precursorca-recipes",
			"rtrouton-recipes",
			"scriptingosx-recipes",
			"sebtomasi-recipes",
			"smithjw-recipes",
			"tbridge-recipes",
			"wardsparadox-recipes",
			"kevinmcox-recipes",
			"swy-recipes",
		}, true).

		// Recipe validation and execution
		AddVerifyTrustInfoStep([]string{
			"Firefox.install",
			"GoogleChrome.install",
			"MicrosoftTeams.install",
		}, nil, true).
		AddBatchProcessingStep([]string{
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
