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
}

// RunRecipeBatch executes parsed recipes using appropriate flags and notifications.
func RunRecipeBatch(recipeInput string, options *RecipeBatchRunOptions) (map[string]*RecipeBatchResult, error) {
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

		result := &RecipeBatchResult{
			Recipe:         recipeInput,
			Output:         output,
			Executed:       true,
			ExecutionError: err,
			TrustVerified:  true,
			TrustUpdated:   options.UpdateTrustOnFailure,
		}

		results[recipeInput] = result
		handleNotifications(result, options)

		if err != nil {
			logger.Logger(fmt.Sprintf("❌ Recipe list %s failed after %s: %v", recipeInput, executionTime, err), logger.LogError)
			return results, err
		}

		logger.Logger(fmt.Sprintf("✅ Recipe list %s succeeded in %s", recipeInput, executionTime), logger.LogSuccess)
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

		result := &RecipeBatchResult{
			Recipe:         recipe,
			Output:         output,
			Executed:       true,
			ExecutionError: err,
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

	return results, nil
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
