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

	// Choose processing path based on input type
	if isRecipeListFile {
		err = processRecipeListFile(recipeInput, options, results, batchStartTime)
	} else {
		err = processIndividualRecipes(recipes, options, results, batchStartTime)
	}

	return results, err
}

// processRecipeListFile handles execution of recipes from a list file
func processRecipeListFile(recipeInput string, options *RecipeBatchRunOptions, results map[string]*RecipeBatchResult, batchStartTime time.Time) error {
	logger.Logger(fmt.Sprintf("🚀 Running recipes from list file: %s", recipeInput), logger.LogInfo)

	// Extract recipe names from file
	recipeNames, err := extractRecipeNamesFromFile(recipeInput)
	if err != nil {
		logger.Logger(fmt.Sprintf("❌ Failed to read recipe list file: %v", err), logger.LogError)
		return err
	}

	logger.Logger(fmt.Sprintf("📋 Found %d recipes in list file", len(recipeNames)), logger.LogInfo)

	// Verify trust for each recipe if enabled
	if options.VerifyTrust {
		// Create a map to track recipes that should be skipped
		skippedRecipes := make(map[string]bool)
		var verifyErr error

		for _, recipe := range recipeNames {
			startTime := time.Now()
			skip, err := verifyTrustForRecipe(recipe, options, results, startTime)
			if skip {
				skippedRecipes[recipe] = true
			}
			if err != nil && verifyErr == nil {
				verifyErr = err
			}
		}

		// If all recipes are skipped or we need to stop on first error and there was an error
		if len(skippedRecipes) == len(recipeNames) ||
			(verifyErr != nil && options.StopOnFirstError) {
			// Generate summary and exit
			LogRecipeBatchSummary(results, batchStartTime)
			return verifyErr
		}
	}

	// Run autopkg with recipe list (we run all recipes in the list, trust verification is handled by autopkg)
	startTime := time.Now()
	runOpts := createRunOptions(options, recipeInput, "")
	output, err := RunRecipe("", runOpts)
	executionTime := time.Since(startTime)

	// Create results for each recipe in the list
	populateResultsFromRecipeList(recipeNames, recipeInput, output, err, executionTime, options, results)

	// Log execution status
	if err != nil {
		logger.Logger(fmt.Sprintf("❌ Recipe list %s failed after %s: %v", recipeInput, executionTime, err), logger.LogError)
	} else {
		logger.Logger(fmt.Sprintf("✅ Recipe list %s succeeded in %s", recipeInput, executionTime), logger.LogSuccess)
	}

	// Generate summary
	LogRecipeBatchSummary(results, batchStartTime)

	return err
}

// processIndividualRecipes handles execution of individual recipes
func processIndividualRecipes(recipes []string, options *RecipeBatchRunOptions, results map[string]*RecipeBatchResult, batchStartTime time.Time) error {
	var firstError error

	for _, recipe := range recipes {
		logger.Logger(fmt.Sprintf("🚀 Running recipe: %s", recipe), logger.LogInfo)
		startTime := time.Now()

		// Perform trust verification if enabled
		if options.VerifyTrust {
			skipRecipe, err := verifyTrustForRecipe(recipe, options, results, startTime)
			if skipRecipe {
				if options.StopOnFirstError && err != nil && firstError == nil {
					firstError = err
					break
				}
				continue
			}
		}

		// Run the recipe
		runOpts := createRunOptions(options, "", recipe)
		output, err := RunRecipe(recipe, runOpts)
		executionTime := time.Since(startTime)

		// Create and store the result
		result := createRecipeResult(recipe, output, err, executionTime, true, false)
		results[recipe] = result
		handleNotifications(result, options)

		// Handle errors and logging
		if err != nil {
			logger.Logger(fmt.Sprintf("❌ Recipe %s failed after %s: %v", recipe, executionTime, err), logger.LogError)
			if firstError == nil {
				firstError = err
			}
			if options.StopOnFirstError {
				break
			}
		} else {
			logger.Logger(fmt.Sprintf("✅ Recipe %s succeeded in %s", recipe, executionTime), logger.LogSuccess)
		}
	}

	// Generate summary
	LogRecipeBatchSummary(results, batchStartTime)

	return firstError
}

// verifyTrustForRecipe performs trust verification for a single recipe
// Returns true if the recipe should be skipped, and any error that occurred
func verifyTrustForRecipe(recipe string, options *RecipeBatchRunOptions, results map[string]*RecipeBatchResult, startTime time.Time) (bool, error) {
	verifyOpts := &VerifyTrustInfoOptions{
		PrefsPath:    options.PrefsPath,
		SearchDirs:   options.SearchDirs,
		OverrideDirs: options.OverrideDirs,
	}

	success, _, _, verifyErr := VerifyTrustInfoForRecipes([]string{recipe}, verifyOpts)
	if verifyErr != nil || !success {
		logger.Logger(fmt.Sprintf("⚠️ Trust verification failed for recipe %s: %v", recipe, verifyErr), logger.LogWarning)

		trustUpdated := false
		if options.UpdateTrustOnFailure {
			_, updateErr := UpdateTrustInfoForRecipes([]string{recipe}, &UpdateTrustInfoOptions{
				PrefsPath:    options.PrefsPath,
				SearchDirs:   options.SearchDirs,
				OverrideDirs: options.OverrideDirs,
			})
			if updateErr == nil {
				logger.Logger(fmt.Sprintf("✅ Trust info updated for recipe %s", recipe), logger.LogSuccess)
				trustUpdated = true
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
				TrustUpdated:      trustUpdated,
				ExecutionTime:     executionTime,
				Status:            "skipped",
			}
			results[recipe] = result
			handleNotifications(result, options)
			return true, verifyErr
		}
	}

	return false, nil
}

// extractRecipeNamesFromFile reads a recipe list file and returns the recipe names
func extractRecipeNamesFromFile(filePath string) ([]string, error) {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var recipeNames []string
	lines := strings.Split(string(fileData), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			// Remove .recipe suffix if present
			if strings.HasSuffix(line, ".recipe") {
				line = line[:len(line)-7]
			}
			recipeNames = append(recipeNames, line)
		}
	}

	return recipeNames, nil
}

// createRunOptions creates RunOptions from RecipeBatchRunOptions
func createRunOptions(options *RecipeBatchRunOptions, recipeList string, recipe string) *RunOptions {
	return &RunOptions{
		PrefsPath:      options.PrefsPath,
		PreProcessors:  options.PreProcessors,
		PostProcessors: options.PostProcessors,
		Variables:      options.Variables,
		ReportPlist:    options.ReportPlist,
		VerboseLevel:   options.VerboseLevel,
		SearchDirs:     options.SearchDirs,
		OverrideDirs:   options.OverrideDirs,
		RecipeList:     recipeList,
		UpdateTrust:    options.UpdateTrustOnFailure,
	}
}

// populateResultsFromRecipeList creates results for each recipe in a list file
func populateResultsFromRecipeList(recipeNames []string, recipeInput string, output string, err error, executionTime time.Duration, options *RecipeBatchRunOptions, results map[string]*RecipeBatchResult) {
	if len(recipeNames) > 0 {
		// Create result for each recipe
		for _, recipeName := range recipeNames {
			status := determineRecipeStatus(output, recipeName, err)
			result := createRecipeResult(recipeName, output, err, executionTime, true, options.UpdateTrustOnFailure)
			result.Status = status

			results[recipeName] = result
			handleNotifications(result, options)
		}
	} else {
		// Fallback if no recipes were found in the file
		status := determineRecipeStatus(output, "", err)
		result := createRecipeResult(recipeInput, output, err, executionTime, true, options.UpdateTrustOnFailure)
		result.Status = status

		results[recipeInput] = result
		handleNotifications(result, options)
	}
}

// createRecipeResult creates a RecipeBatchResult with the given parameters
func createRecipeResult(recipe string, output string, err error, executionTime time.Duration, trustVerified bool, trustUpdated bool) *RecipeBatchResult {
	status := determineRecipeStatus(output, recipe, err)

	return &RecipeBatchResult{
		Recipe:         recipe,
		Output:         output,
		Executed:       true,
		ExecutionError: err,
		TrustVerified:  trustVerified,
		TrustUpdated:   trustUpdated,
		ExecutionTime:  executionTime,
		Status:         status,
	}
}

// determineRecipeStatus analyzes output to determine a recipe's status
func determineRecipeStatus(output string, recipeName string, err error) string {
	if err != nil {
		return "failed"
	}

	// Try to extract recipe-specific output
	recipeOutput := extractRecipeOutput(output, recipeName)

	if strings.Contains(recipeOutput, "new version") ||
		strings.Contains(recipeOutput, "Downloaded") ||
		strings.Contains(recipeOutput, "Installing") {
		return "updated"
	}

	return "unchanged" // Default
}

// extractRecipeOutput tries to extract output pertaining to a specific recipe
func extractRecipeOutput(fullOutput, recipeName string) string {
	if recipeName == "" {
		return fullOutput
	}

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

	if len(recipeOutput) == 0 {
		return fullOutput // Fallback to full output if nothing was found
	}

	return strings.Join(recipeOutput, "\n")
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
