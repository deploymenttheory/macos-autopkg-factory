// recipe_run.go
package autopkg

import (
	"fmt"
	"os"
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

	// Modified section for recipe list file handling
	if isRecipeListFile {
		logger.Logger(fmt.Sprintf("🚀 Running recipes from list file: %s", recipeInput), logger.LogInfo)

		// Read the list file to extract recipe names
		fileData, readErr := os.ReadFile(recipeInput)
		if readErr != nil {
			logger.Logger(fmt.Sprintf("❌ Failed to read recipe list file: %v", readErr), logger.LogError)
			return nil, readErr
		}

		// Parse individual recipe names from the file
		var recipeNames []string
		lines := strings.Split(string(fileData), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				// Store raw recipe name without .recipe suffix (parser will add it later)
				if strings.HasSuffix(line, ".recipe") {
					line = line[:len(line)-7]
				}
				recipeNames = append(recipeNames, line)
			}
		}

		logger.Logger(fmt.Sprintf("📋 Found %d recipes in list file", len(recipeNames)), logger.LogInfo)

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

		if len(recipeNames) > 0 {
			// Create individual result entries for each recipe in the list
			for _, recipeName := range recipeNames {
				// Try to determine status for this specific recipe
				recipeStatus := "unchanged" // Default

				if err != nil {
					recipeStatus = "failed"
				} else {
					// Look for recipe-specific output indicators
					recipeOutput := extractRecipeOutput(output, recipeName)
					if strings.Contains(recipeOutput, "new version") ||
						strings.Contains(recipeOutput, "Downloaded") ||
						strings.Contains(recipeOutput, "Installing") {
						recipeStatus = "updated"
					}
				}

				recipeResult := &RecipeBatchResult{
					Recipe:         recipeName,
					Output:         output, // Full output, could be filtered per recipe
					Executed:       true,
					ExecutionError: err, // Same error status for all recipes
					TrustVerified:  true,
					TrustUpdated:   options.UpdateTrustOnFailure,
					ExecutionTime:  executionTime, // Same execution time for all recipes
					Status:         recipeStatus,
				}

				results[recipeName] = recipeResult
				handleNotifications(recipeResult, options)
			}
		} else {
			// Fallback if we couldn't extract recipe names
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
		}

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

// extractRecipeOutput tries to extract output pertaining to a specific recipe
func extractRecipeOutput(fullOutput, recipeName string) string {
	lines := strings.Split(fullOutput, "\n")
	var recipeOutput []string
	inRecipeSection := false

	for _, line := range lines {
		if strings.Contains(line, recipeName) {
			inRecipeSection = true
			recipeOutput = append(recipeOutput, line)
		} else if inRecipeSection {
			// Continue collecting lines until we see another recipe header
			// or a clear section delimiter
			if strings.Contains(line, "Processing ") ||
				strings.Contains(line, "===") ||
				strings.Contains(line, "---") {
				inRecipeSection = false
			} else {
				recipeOutput = append(recipeOutput, line)
			}
		}
	}

	return strings.Join(recipeOutput, "\n")
}
