// recipe_run.go
package autopkg

import (
	"fmt"
	"strings"
	"time"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// RecipeBatchRunOptions contains options for processing a batch of recipes through multiple steps
type RecipeBatchRunOptions struct {
	PrefsPath            string
	SearchDirs           []string
	OverrideDirs         []string
	VerifyTrust          bool
	UpdateTrustOnFailure bool
	IgnoreVerifyFailures bool
	ReportPlist          string
	VerboseLevel         int
	Variables            map[string]string
	PreProcessors        []string
	PostProcessors       []string
	StopOnFirstError     bool
	Notification         NotificationOptions
}

type NotificationOptions struct {
	EnableTeams   bool
	TeamsWebhook  string
	EnableSlack   bool
	SlackWebhook  string
	SlackUsername string
	SlackChannel  string
	SlackIcon     string
}

// RecipeBatchResult contains the results of a batch operation
type RecipeBatchResult struct {
	Recipe            string
	TrustVerified     bool
	TrustUpdated      bool
	Executed          bool
	Output            string
	VerificationError error
	ExecutionError    error
	ExecutionTime     time.Duration
	Status            string // "updated", "unchanged", "skipped", "failed"
}

// RecipeBatchSummary contains aggregated metrics from a batch run
type RecipeBatchSummary struct {
	TotalDuration    time.Duration
	TotalRecipes     int
	SuccessCount     int
	FailedCount      int
	SkippedCount     int
	UpdatedCount     int
	UnchangedCount   int
	UpdatedRecipes   []string
	UnchangedRecipes []string
	SkippedRecipes   []string
	FailedRecipes    []string
}

// RunRecipeBatch executes parsed recipes using appropriate flags and notifications.
func RunRecipeBatch(recipeInput string, options *RecipeBatchRunOptions) (map[string]*RecipeBatchResult, error) {
	batchStartTime := time.Now()

	if options == nil {
		options = &RecipeBatchRunOptions{}
	}

	results := make(map[string]*RecipeBatchResult)
	parser := NewParserFromInput(recipeInput)
	recipes, err := parser.Parse()
	if err != nil {
		logger.Logger(fmt.Sprintf("❌ Failed to parse recipes: %v", err), logger.LogError)
		return nil, err
	}

	isRecipeListFile := strings.HasSuffix(strings.ToLower(recipeInput), ".txt")

	if isRecipeListFile {
		logger.Logger(fmt.Sprintf("🚀 Running recipes from list file: %s", recipeInput), logger.LogInfo)
		startTime := time.Now()

		runOpts := &RunOptions{
			PrefsPath:      options.PrefsPath,
			PreProcessors:  options.PreProcessors,
			PostProcessors: options.PostProcessors,
			Variables:      options.Variables,
			ReportPlist:    options.ReportPlist,
			VerboseLevel:   options.VerboseLevel,
			SearchDirs:     options.SearchDirs,
			OverrideDirs:   options.OverrideDirs,
			RecipeList:     recipeInput,
			UpdateTrust:    options.UpdateTrustOnFailure,
		}

		output, err := RunRecipe("", runOpts)
		executionTime := time.Since(startTime)

		// Determine status based on output and error
		status := "failed"
		if err == nil {
			if strings.Contains(output, "No new updates available") || strings.Contains(output, "No changes") {
				status = "unchanged"
			} else if strings.Contains(output, "Downloaded") || strings.Contains(output, "Installing") || strings.Contains(output, "new version") {
				status = "updated"
			} else {
				status = "unchanged" // Default if we can't determine
			}
		}

		result := &RecipeBatchResult{
			Recipe:         recipeInput,
			Output:         output,
			Executed:       true,
			ExecutionError: err,
			TrustVerified:  true,
			TrustUpdated:   options.UpdateTrustOnFailure,
			ExecutionTime:  executionTime,
			Status:         status,
		}

		results[recipeInput] = result
		handleNotifications(result, options)

		if err != nil {
			logger.Logger(fmt.Sprintf("❌ Recipe list %s failed after %s: %v", recipeInput, executionTime, err), logger.LogError)

			// Generate and log summary even on error
			LogRecipeBatchSummary(results, batchStartTime)

			return results, err
		}

		logger.Logger(fmt.Sprintf("✅ Recipe list %s succeeded in %s", recipeInput, executionTime), logger.LogSuccess)

		// Generate and log summary
		LogRecipeBatchSummary(results, batchStartTime)

		return results, nil
	}

	for _, recipe := range recipes {
		logger.Logger(fmt.Sprintf("🚀 Running recipe: %s", recipe), logger.LogInfo)
		startTime := time.Now()

		if options.VerifyTrust {
			verifyOpts := &VerifyTrustInfoOptions{
				PrefsPath:    options.PrefsPath,
				SearchDirs:   options.SearchDirs,
				OverrideDirs: options.OverrideDirs,
			}

			success, _, _, verifyErr := VerifyTrustInfoForRecipes([]string{recipe}, verifyOpts)
			if verifyErr != nil || !success {
				logger.Logger(fmt.Sprintf("⚠️ Trust verification failed for recipe %s: %v", recipe, verifyErr), logger.LogWarning)
				if options.UpdateTrustOnFailure {
					_, updateErr := UpdateTrustInfoForRecipes([]string{recipe}, &UpdateTrustInfoOptions{
						PrefsPath:    options.PrefsPath,
						SearchDirs:   options.SearchDirs,
						OverrideDirs: options.OverrideDirs,
					})
					if updateErr == nil {
						logger.Logger(fmt.Sprintf("✅ Trust info updated for recipe %s", recipe), logger.LogSuccess)
					}
				}
				if !options.IgnoreVerifyFailures {
					// Add to results as "skipped"
					executionTime := time.Since(startTime)
					result := &RecipeBatchResult{
						Recipe:            recipe,
						Executed:          false,
						VerificationError: verifyErr,
						TrustVerified:     false,
						TrustUpdated:      options.UpdateTrustOnFailure,
						ExecutionTime:     executionTime,
						Status:            "skipped",
					}
					results[recipe] = result
					handleNotifications(result, options)

					if options.StopOnFirstError {
						break
					}
					continue
				}
			}
		}

		runOpts := &RunOptions{
			PrefsPath:      options.PrefsPath,
			PreProcessors:  options.PreProcessors,
			PostProcessors: options.PostProcessors,
			Variables:      options.Variables,
			ReportPlist:    options.ReportPlist,
			VerboseLevel:   options.VerboseLevel,
			SearchDirs:     options.SearchDirs,
			OverrideDirs:   options.OverrideDirs,
			UpdateTrust:    options.UpdateTrustOnFailure,
		}

		output, err := RunRecipe(recipe, runOpts)
		executionTime := time.Since(startTime)

		// Determine status based on output and error
		status := "failed"
		if err == nil {
			if strings.Contains(output, "No new updates available") || strings.Contains(output, "No changes") {
				status = "unchanged"
			} else if strings.Contains(output, "Downloaded") || strings.Contains(output, "Installing") || strings.Contains(output, "new version") {
				status = "updated"
			} else {
				status = "unchanged" // Default if we can't determine
			}
		}

		result := &RecipeBatchResult{
			Recipe:         recipe,
			Output:         output,
			Executed:       true,
			ExecutionError: err,
			ExecutionTime:  executionTime,
			Status:         status,
		}
		results[recipe] = result
		handleNotifications(result, options)

		if err != nil {
			logger.Logger(fmt.Sprintf("❌ Recipe %s failed after %s: %v", recipe, executionTime, err), logger.LogError)
			if options.StopOnFirstError {
				break
			}
		} else {
			logger.Logger(fmt.Sprintf("✅ Recipe %s succeeded in %s", recipe, executionTime), logger.LogSuccess)
		}
	}

	// Generate and log summary
	LogRecipeBatchSummary(results, batchStartTime)

	return results, nil
}

// LogRecipeBatchSummary logs a summary of the recipe batch execution
func LogRecipeBatchSummary(results map[string]*RecipeBatchResult, startTime time.Time) {
	// Calculate summary metrics
	summary := &RecipeBatchSummary{
		TotalDuration:    time.Since(startTime),
		TotalRecipes:     len(results),
		UpdatedRecipes:   make([]string, 0),
		UnchangedRecipes: make([]string, 0),
		SkippedRecipes:   make([]string, 0),
		FailedRecipes:    make([]string, 0),
	}

	// Categorize recipes by status
	for recipe, result := range results {
		switch result.Status {
		case "updated":
			summary.SuccessCount++
			summary.UpdatedCount++
			summary.UpdatedRecipes = append(summary.UpdatedRecipes, recipe)
		case "unchanged":
			summary.SuccessCount++
			summary.UnchangedCount++
			summary.UnchangedRecipes = append(summary.UnchangedRecipes, recipe)
		case "skipped":
			summary.SkippedCount++
			summary.SkippedRecipes = append(summary.SkippedRecipes, recipe)
		case "failed":
			summary.FailedCount++
			summary.FailedRecipes = append(summary.FailedRecipes, recipe)
		}
	}

	// Log the summary
	logger.Logger("\n🚀 Pipeline Execution Summary", logger.LogInfo)
	logger.Logger(fmt.Sprintf("Total execution time: %s", summary.TotalDuration), logger.LogInfo)
	logger.Logger(fmt.Sprintf("Total recipes processed: %d", summary.TotalRecipes), logger.LogInfo)
	logger.Logger(fmt.Sprintf("✅ Successful: %d", summary.SuccessCount), logger.LogSuccess)
	logger.Logger(fmt.Sprintf("  - Updated: %d", summary.UpdatedCount), logger.LogSuccess)
	logger.Logger(fmt.Sprintf("  - Unchanged: %d", summary.UnchangedCount), logger.LogInfo)
	logger.Logger(fmt.Sprintf("⏩ Skipped: %d", summary.SkippedCount), logger.LogInfo)
	logger.Logger(fmt.Sprintf("❌ Failed: %d", summary.FailedCount), logger.LogError)

	// Log detailed recipe lists by category
	if len(summary.UpdatedRecipes) > 0 {
		logger.Logger("\n📦 Updated Recipes:", logger.LogSuccess)
		for _, recipe := range summary.UpdatedRecipes {
			logger.Logger(fmt.Sprintf("  • %s", recipe), logger.LogSuccess)
		}
	}

	if len(summary.UnchangedRecipes) > 0 {
		logger.Logger("\n🔄 Unchanged Recipes:", logger.LogInfo)
		for _, recipe := range summary.UnchangedRecipes {
			logger.Logger(fmt.Sprintf("  • %s", recipe), logger.LogInfo)
		}
	}

	if len(summary.SkippedRecipes) > 0 {
		logger.Logger("\n⏩ Skipped Recipes:", logger.LogInfo)
		for _, recipe := range summary.SkippedRecipes {
			logger.Logger(fmt.Sprintf("  • %s", recipe), logger.LogInfo)
		}
	}

	if len(summary.FailedRecipes) > 0 {
		logger.Logger("\n❌ Failed Recipes:", logger.LogError)
		for _, recipe := range summary.FailedRecipes {
			logger.Logger(fmt.Sprintf("  • %s", recipe), logger.LogError)
		}
	}

	// Final summary
	if summary.FailedCount > 0 {
		logger.Logger("🚨 Pipeline status: FAILURE - Some recipes failed.", logger.LogError)
	} else {
		logger.Logger("🎉 Pipeline status: SUCCESS - All recipes succeeded.", logger.LogSuccess)
	}
}

// Helper function to handle notification
func handleNotifications(result *RecipeBatchResult, options *RecipeBatchRunOptions) {
	if options.VerboseLevel <= 1 {
		if options.Notification.EnableTeams {
			teamsNotifier := &MSTeamsNotifier{
				WebhookURL: options.Notification.TeamsWebhook,
			}

			recipeLifecycle := &RecipeLifecycle{
				Name:     result.Recipe,
				Error:    result.ExecutionError != nil,
				Updated:  result.TrustUpdated,
				Verified: &result.TrustVerified,
				Results:  map[string]interface{}{}, // Populate if necessary
			}

			teamsNotifier.NotifyTeams(recipeLifecycle, options)
		}

		if options.Notification.EnableSlack {
			slackNotifier := &SlackNotifier{
				WebhookURL: options.Notification.SlackWebhook,
				Username:   options.Notification.SlackUsername,
				Channel:    options.Notification.SlackChannel,
				IconEmoji:  options.Notification.SlackIcon,
			}

			recipeLifecycle := &RecipeLifecycle{
				Name:     result.Recipe,
				Error:    result.ExecutionError != nil,
				Updated:  result.TrustUpdated,
				Verified: &result.TrustVerified,
				Results:  map[string]interface{}{}, // Populate if necessary
			}

			slackNotifier.NotifySlack(recipeLifecycle)
		}
	}
}
