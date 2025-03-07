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

	// Set up the workflow steps
	orchestrator.
		// Initialization steps
		AddRootCheckStep(false).
		AddGitCheckStep(true).
		AddInstallAutoPkgStep(&autopkg.InstallConfig{
			ForceUpdate: true,
			UseBeta:     false,
		}, true).

		// Repository management steps
		AddRepoUpdateStep([]string{
			"autopkg/recipes",
			"homebysix/recipes",
			// Add other repositories as needed
		}, true).
		AddRepoUpdateStep([]string{"recipes", "homebysix-recipes"}, true).

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
