// recipe_run.go contains various abstractions for common autopkg operations using commands.go
package autopkg

import (
	"fmt"
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
	Error             error
	Output            string
	VerificationError error
	ExecutionError    error
}

func RunRecipeBatch(recipes []string, options *RecipeBatchRunOptions) (map[string]*RecipeBatchResult, error) {
	if options == nil {
		options = &RecipeBatchRunOptions{}
	}

	logger.Logger(fmt.Sprintf("🚀 Running batch of %d recipes sequentially", len(recipes)), logger.LogInfo)

	results := make(map[string]*RecipeBatchResult)
	var firstError error

	for _, recipe := range recipes {
		logger.Logger(fmt.Sprintf("🔄 Processing recipe: %s", recipe), logger.LogInfo)

		startTime := time.Now()
		result := &RecipeBatchResult{Recipe: recipe}

		// Run trust verification if enabled
		if options.VerifyTrust {
			verifyOptions := &VerifyTrustInfoOptions{
				PrefsPath:    options.PrefsPath,
				SearchDirs:   options.SearchDirs,
				OverrideDirs: options.OverrideDirs,
			}

			success, _, _, err := VerifyTrustInfoForRecipes([]string{recipe}, verifyOptions)
			if err != nil || !success {
				logger.Logger(fmt.Sprintf("⚠️ Trust verification failed for %s", recipe), logger.LogWarning)
				result.TrustVerified = false
				result.VerificationError = err
				if options.UpdateTrustOnFailure {
					updateOptions := &UpdateTrustInfoOptions{
						PrefsPath:    options.PrefsPath,
						SearchDirs:   options.SearchDirs,
						OverrideDirs: options.OverrideDirs,
					}

					_, err := UpdateTrustInfoForRecipes([]string{recipe}, updateOptions)
					if err != nil {
						logger.Logger(fmt.Sprintf("❌ Failed to update trust info for %s: %v", recipe, err), logger.LogError)
						continue
					}
					result.TrustUpdated = true
				}
			}
		}

		// Prepare RunOptions
		runOptions := &RunOptions{
			PrefsPath:      options.PrefsPath,
			PreProcessors:  options.PreProcessors,
			PostProcessors: options.PostProcessors,
			Variables:      options.Variables,
			ReportPlist:    options.ReportPlist,
			VerboseLevel:   options.VerboseLevel,
			SearchDirs:     options.SearchDirs,
			OverrideDirs:   options.OverrideDirs,
		}

		// Run the recipe
		output, err := RunRecipe(recipe, runOptions)
		result.Output = output
		result.Executed = true
		result.ExecutionError = err

		elapsedTime := time.Since(startTime)
		if err == nil {
			logger.Logger(fmt.Sprintf("✅ Recipe %s completed successfully in %s", recipe, elapsedTime), logger.LogSuccess)
		} else {
			logger.Logger(fmt.Sprintf("❌ Recipe %s failed after %s: %v", recipe, elapsedTime, err), logger.LogError)
			if firstError == nil {
				firstError = err
			}
			if options.StopOnFirstError {
				break
			}
		}

		results[recipe] = result

		// Send notifications if enabled and verbosity is at the lowest level (e.g., 0 or 1)
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

	logger.Logger("✅ Batch processing complete", logger.LogSuccess)
	return results, firstError
}
