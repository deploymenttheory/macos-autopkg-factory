// run.go contains various abstractions for common autopkg operations using commands.go
package autopkg

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
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
	MaxConcurrentRecipes int
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

func RecipeBatchProcessing(recipes []string, options *RecipeBatchRunOptions) (map[string]*RecipeBatchResult, error) {
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

// RecipeFilterCriteria defines the criteria for filtering recipes
type RecipeFilterCriteria struct {
	NamePattern       string    // Regex pattern for recipe names
	ExcludePattern    string    // Regex pattern for excluding recipes
	RecipeTypes       []string  // Specific recipe types (e.g., "download", "pkg", "install")
	ModifiedAfter     time.Time // Only include recipes modified after this time
	ModifiedBefore    time.Time // Only include recipes modified before this time
	TrustInfoRequired bool      // Only include recipes with trust info
	VerifiedTrustOnly bool      // Only include recipes that pass trust verification
	IncludeOverrides  bool      // Include recipe overrides
	IncludeDisabled   bool      // Include disabled recipes (with "disabled" in name)
	MaxRecipes        int       // Maximum number of recipes to return (0 = all)
}

// FilterRecipesResult contains information about filtered recipes
type FilterRecipesResult struct {
	MatchingRecipes []string              // List of recipes that match the filter criteria
	TrustStatus     map[string]bool       // Trust verification status for each recipe
	RecipeInfo      map[string]RecipeInfo // Additional information about each recipe
}

// RecipeInfo contains metadata about a recipe
type RecipeInfo struct {
	Path          string
	Identifier    string
	Type          string
	ParentRecipes []string
	IsOverride    bool
	IsDisabled    bool
	ModTime       time.Time
}

// FilterRecipes filters recipes based on various criteria
func FilterRecipes(options *RecipeFilterCriteria, prefsPath string) (*FilterRecipesResult, error) {
	if options == nil {
		options = &RecipeFilterCriteria{
			IncludeOverrides: true,
			IncludeDisabled:  false,
		}
	}

	logger.Logger("🔍 Filtering recipes based on criteria", logger.LogInfo)

	// We'll capture the output of the list-recipes command
	cmd := exec.Command("autopkg", "list-recipes", "--with-identifiers", "--with-paths")
	if prefsPath != "" {
		cmd.Args = append(cmd.Args, "--prefs", prefsPath)
	}
	if options.IncludeOverrides {
		cmd.Args = append(cmd.Args, "--show-all")
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list recipes: %w", err)
	}

	// Parse the output
	lines := strings.Split(string(output), "\n")
	result := &FilterRecipesResult{
		MatchingRecipes: []string{},
		TrustStatus:     make(map[string]bool),
		RecipeInfo:      make(map[string]RecipeInfo),
	}

	// Regular expressions for filtering
	var nameRegex, excludeRegex *regexp.Regexp
	if options.NamePattern != "" {
		nameRegex, err = regexp.Compile(options.NamePattern)
		if err != nil {
			return nil, fmt.Errorf("invalid name pattern: %w", err)
		}
	}
	if options.ExcludePattern != "" {
		excludeRegex, err = regexp.Compile(options.ExcludePattern)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude pattern: %w", err)
		}
	}

	// Process each line
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// The format is: name (identifier) - path
		parts := strings.SplitN(line, " (", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		remainingParts := strings.SplitN(parts[1], ") - ", 2)
		if len(remainingParts) != 2 {
			continue
		}

		identifier := strings.TrimSpace(remainingParts[0])
		path := strings.TrimSpace(remainingParts[1])

		// Apply name pattern filter
		if nameRegex != nil && !nameRegex.MatchString(name) {
			continue
		}

		// Apply exclude pattern filter
		if excludeRegex != nil && excludeRegex.MatchString(name) {
			continue
		}

		// Check if it's disabled (has "disabled" in the name)
		isDisabled := strings.Contains(strings.ToLower(name), "disabled")
		if isDisabled && !options.IncludeDisabled {
			continue
		}

		// Determine recipe type
		recipeType := ""
		if strings.HasSuffix(name, ".download") {
			recipeType = "download"
		} else if strings.HasSuffix(name, ".pkg") {
			recipeType = "pkg"
		} else if strings.HasSuffix(name, ".install") {
			recipeType = "install"
		} else if strings.HasSuffix(name, ".munki") {
			recipeType = "munki"
		} else if strings.HasSuffix(name, ".jamf") {
			recipeType = "jamf"
		} else if strings.HasSuffix(name, ".intune") {
			recipeType = "intune"
		}

		// Filter by recipe type if specified
		if len(options.RecipeTypes) > 0 {
			typeMatched := false
			for _, t := range options.RecipeTypes {
				if t == recipeType {
					typeMatched = true
					break
				}
			}
			if !typeMatched {
				continue
			}
		}

		// Check if it's an override
		isOverride := strings.Contains(path, "RecipeOverrides") || strings.Contains(identifier, ".override.")

		// Get file modification time
		fileInfo, err := os.Stat(path)
		if err != nil {
			logger.Logger(fmt.Sprintf("⚠️ Could not stat recipe file %s: %v", path, err), logger.LogWarning)
			continue
		}
		modTime := fileInfo.ModTime()

		// Filter by modification time
		if !options.ModifiedAfter.IsZero() && modTime.Before(options.ModifiedAfter) {
			continue
		}
		if !options.ModifiedBefore.IsZero() && modTime.After(options.ModifiedBefore) {
			continue
		}

		// Create recipe info
		recipeInfo := RecipeInfo{
			Path:       path,
			Identifier: identifier,
			Type:       recipeType,
			IsOverride: isOverride,
			IsDisabled: isDisabled,
			ModTime:    modTime,
		}

		// Add parent recipes info if it's an override
		if isOverride {
			// Run autopkg info to get parent recipes
			infoCmd := exec.Command("autopkg", "info", "-p", name)
			if prefsPath != "" {
				infoCmd.Args = append(infoCmd.Args, "--prefs", prefsPath)
			}
			infoOutput, err := infoCmd.Output()
			if err == nil {
				infoLines := strings.Split(string(infoOutput), "\n")
				for _, infoLine := range infoLines {
					if strings.Contains(infoLine, "Parent Recipe:") {
						parentParts := strings.SplitN(infoLine, ":", 2)
						if len(parentParts) == 2 {
							parentRecipe := strings.TrimSpace(parentParts[1])
							recipeInfo.ParentRecipes = append(recipeInfo.ParentRecipes, parentRecipe)
						}
					}
				}
			}
		}

		// If trust info verification is required, check it
		if options.TrustInfoRequired || options.VerifiedTrustOnly {
			if isOverride {
				// Just check a single recipe
				verifyOptions := &VerifyTrustInfoOptions{
					PrefsPath: prefsPath,
				}

				success, failedRecipes, verifyOutput, verifyErr := VerifyTrustInfoForRecipes([]string{name}, verifyOptions)

				// Consider the trust verified only if both the verification process succeeded and no recipes failed
				trustVerified := verifyErr == nil && success && len(failedRecipes) == 0

				// Log debug output for failed verifications
				if !trustVerified {
					if verifyErr != nil {
						logger.Logger(fmt.Sprintf("⚠️ Trust verification error for %s: %v", name, verifyErr), logger.LogWarning)
					}
					logger.Logger(fmt.Sprintf("🔍 Trust verification output for %s:\n%s", name, verifyOutput), logger.LogDebug)
				}

				result.TrustStatus[name] = trustVerified

				if options.VerifiedTrustOnly && !trustVerified {
					continue
				}
			} else if options.TrustInfoRequired {
				continue
			}
		}

		// Add the recipe to the result
		result.MatchingRecipes = append(result.MatchingRecipes, name)
		result.RecipeInfo[name] = recipeInfo

		// Limit the number of recipes if specified
		if options.MaxRecipes > 0 && len(result.MatchingRecipes) >= options.MaxRecipes {
			break
		}
	}

	logger.Logger(fmt.Sprintf("✅ Found %d matching recipes", len(result.MatchingRecipes)), logger.LogSuccess)
	return result, nil
}

// ImportRecipesFromRepoOptions contains options for importing recipes from a repo
type ImportRecipesFromRepoOptions struct {
	PrefsPath            string
	VerifyTrust          bool
	UpdateTrustOnFailure bool
	RequiredRecipes      []string
	RecipePattern        string
	IgnoreRecipePattern  string
}

// ImportRecipesFromRepo adds a repo and imports all its recipes in one operation
func ImportRecipesFromRepo(repoURL string, options *ImportRecipesFromRepoOptions) ([]string, error) {
	if options == nil {
		options = &ImportRecipesFromRepoOptions{
			VerifyTrust:          true,
			UpdateTrustOnFailure: true,
		}
	}

	logger.Logger(fmt.Sprintf("🔄 Importing recipes from repo: %s", repoURL), logger.LogInfo)

	// Add the repo using the AddRepo function
	repoOutput, err := AddRepo([]string{repoURL}, options.PrefsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to add recipe repo: %w", err)
	}
	logger.Logger(fmt.Sprintf("📦 Repo add output:\n%s", repoOutput), logger.LogDebug)

	// Parse the repo name from the URL
	repoName := repoURL
	if strings.Contains(repoURL, "/") {
		parts := strings.Split(repoURL, "/")
		repoName = parts[len(parts)-1]
	}
	if strings.HasSuffix(repoName, ".git") {
		repoName = repoName[:len(repoName)-4]
	}

	// Compile regex patterns if specified
	var recipeRegex, ignoreRegex *regexp.Regexp
	var regexErr error
	if options.RecipePattern != "" {
		recipeRegex, regexErr = regexp.Compile(options.RecipePattern)
		if regexErr != nil {
			return nil, fmt.Errorf("invalid recipe pattern: %w", regexErr)
		}
	}
	if options.IgnoreRecipePattern != "" {
		ignoreRegex, regexErr = regexp.Compile(options.IgnoreRecipePattern)
		if regexErr != nil {
			return nil, fmt.Errorf("invalid ignore recipe pattern: %w", regexErr)
		}
	}

	// Get list of all recipes
	listArgs := []string{"list-recipes", "--with-identifiers"}
	if options.PrefsPath != "" {
		listArgs = append(listArgs, "--prefs", options.PrefsPath)
	}
	listCmd := exec.Command("autopkg", listArgs...)
	listOutput, err := listCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list recipes: %w", err)
	}

	// Find recipes from the repo
	var repoRecipes []string
	lines := strings.Split(string(listOutput), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " (", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		identifier := strings.TrimSpace(parts[1])
		identifier = strings.TrimSuffix(identifier, ")")

		// Check if this recipe is from our repo
		if strings.Contains(identifier, repoName) {
			// Apply regex pattern filters
			if recipeRegex != nil && !recipeRegex.MatchString(name) {
				continue
			}
			if ignoreRegex != nil && ignoreRegex.MatchString(name) {
				continue
			}

			repoRecipes = append(repoRecipes, name)
		}
	}

	// Add required recipes if specified
	if len(options.RequiredRecipes) > 0 {
		for _, required := range options.RequiredRecipes {
			found := false
			for _, recipe := range repoRecipes {
				if recipe == required {
					found = true
					break
				}
			}
			if !found {
				repoRecipes = append(repoRecipes, required)
			}
		}
	}

	// Make overrides for each recipe
	var importedRecipes []string
	for _, recipe := range repoRecipes {
		// Make an override using MakeOverride function
		overrideOptions := &MakeOverrideOptions{
			PrefsPath: options.PrefsPath,
			Force:     true,
		}

		overrideOutput, err := MakeOverride(recipe, overrideOptions)
		if err != nil {
			logger.Logger(fmt.Sprintf("⚠️ Failed to make override for %s: %v\n%s", recipe, err, overrideOutput), logger.LogWarning)
			continue
		}

		logger.Logger(fmt.Sprintf("✅ Created override for recipe: %s", recipe), logger.LogSuccess)
		logger.Logger(fmt.Sprintf("🧾 Override output for %s:\n%s", recipe, overrideOutput), logger.LogDebug)

		// If verify trust is enabled, run verification
		if options.VerifyTrust {
			// Use VerifyTrustInfoForRecipes
			verifyOptions := &VerifyTrustInfoOptions{
				PrefsPath: options.PrefsPath,
			}

			success, _, verifyOutput, err := VerifyTrustInfoForRecipes([]string{recipe + ".override"}, verifyOptions)
			logger.Logger(fmt.Sprintf("🔍 Trust verification output for %s:\n%s", recipe, verifyOutput), logger.LogDebug)

			if !success || err != nil {
				if options.UpdateTrustOnFailure {
					// Try to update trust info using UpdateTrustInfoForRecipes
					updateOptions := &UpdateTrustInfoOptions{
						PrefsPath: options.PrefsPath,
					}

					updateOutput, updateErr := UpdateTrustInfoForRecipes([]string{recipe + ".override"}, updateOptions)
					logger.Logger(fmt.Sprintf("🔄 Trust update output for %s:\n%s", recipe, updateOutput), logger.LogDebug)

					if updateErr != nil {
						logger.Logger(fmt.Sprintf("⚠️ Failed to update trust info for %s: %v", recipe, updateErr), logger.LogWarning)
						continue
					}

					logger.Logger(fmt.Sprintf("✅ Trust info updated for recipe: %s", recipe), logger.LogSuccess)

					// Verify again after update
					success, _, verifyOutput, verifyErr := VerifyTrustInfoForRecipes([]string{recipe + ".override"}, verifyOptions)
					logger.Logger(fmt.Sprintf("🔍 Second trust verification for %s:\n%s", recipe, verifyOutput), logger.LogDebug)

					if !success || verifyErr != nil {
						logger.Logger(fmt.Sprintf("⚠️ Failed to verify trust info for %s even after update", recipe), logger.LogWarning)
						continue
					}
				} else {
					logger.Logger(fmt.Sprintf("⚠️ Trust verification failed for %s", recipe), logger.LogWarning)
					continue
				}
			}

			logger.Logger(fmt.Sprintf("✅ Trust verification passed for recipe: %s", recipe), logger.LogSuccess)
		}

		importedRecipes = append(importedRecipes, recipe+".override")
		logger.Logger(fmt.Sprintf("✅ Successfully imported recipe: %s", recipe), logger.LogSuccess)
	}

	logger.Logger(fmt.Sprintf("✅ Imported %d recipes from repo %s", len(importedRecipes), repoURL), logger.LogSuccess)
	return importedRecipes, nil
}

// ValidateRecipeListOptions contains options for validating a recipe list
type ValidateRecipeListOptions struct {
	PrefsPath            string
	SearchDirs           []string
	OverrideDirs         []string
	VerifyTrust          bool
	UpdateTrustOnFailure bool
	AllowNonExistent     bool
}

// ValidateRecipeListResult contains the result of a recipe list validation
type ValidateRecipeListResult struct {
	ValidRecipes       []string
	InvalidRecipes     []string
	TrustFailedRecipes []string
	MissingRecipes     []string
}

// ValidateRecipeList checks if all recipes in a list exist and are accessible
func ValidateRecipeList(recipes []string, options *ValidateRecipeListOptions) (*ValidateRecipeListResult, error) {
	if options == nil {
		options = &ValidateRecipeListOptions{
			VerifyTrust:          true,
			UpdateTrustOnFailure: true,
			AllowNonExistent:     false,
		}
	}

	logger.Logger(fmt.Sprintf("🔍 Validating %d recipes", len(recipes)), logger.LogInfo)

	result := &ValidateRecipeListResult{
		ValidRecipes:       []string{},
		InvalidRecipes:     []string{},
		TrustFailedRecipes: []string{},
		MissingRecipes:     []string{},
	}

	// Get list of all available recipes
	listCmd := exec.Command("autopkg", "list-recipes")
	if options.PrefsPath != "" {
		listCmd.Args = append(listCmd.Args, "--prefs", options.PrefsPath)
	}
	for _, dir := range options.SearchDirs {
		listCmd.Args = append(listCmd.Args, "--search-dir", dir)
	}
	for _, dir := range options.OverrideDirs {
		listCmd.Args = append(listCmd.Args, "--override-dir", dir)
	}

	listOutput, err := listCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list available recipes: %w", err)
	}

	// Parse the available recipes
	availableRecipes := map[string]bool{}
	lines := strings.Split(string(listOutput), "\n")
	for _, line := range lines {
		recipeName := strings.TrimSpace(line)
		if recipeName != "" {
			availableRecipes[recipeName] = true
		}
	}

	// Check each recipe in the list
	for _, recipe := range recipes {
		recipe = strings.TrimSpace(recipe)
		if recipe == "" {
			continue
		}

		// Check if the recipe exists
		if !availableRecipes[recipe] {
			result.MissingRecipes = append(result.MissingRecipes, recipe)
			if !options.AllowNonExistent {
				result.InvalidRecipes = append(result.InvalidRecipes, recipe)
			}
			continue
		}

		// If it's an override and trust verification is enabled, verify it
		if options.VerifyTrust && strings.HasSuffix(recipe, ".override") {
			verifyOptions := &VerifyTrustInfoOptions{
				PrefsPath:    options.PrefsPath,
				SearchDirs:   options.SearchDirs,
				OverrideDirs: options.OverrideDirs,
			}

			success, _, verifyOutput, err := VerifyTrustInfoForRecipes([]string{recipe}, verifyOptions)
			logger.Logger(fmt.Sprintf("🔍 Trust verification for %s:\n%s", recipe, verifyOutput), logger.LogDebug)

			if err != nil || !success {
				if options.UpdateTrustOnFailure {
					// Try to update trust info
					updateOptions := &UpdateTrustInfoOptions{
						PrefsPath:    options.PrefsPath,
						SearchDirs:   options.SearchDirs,
						OverrideDirs: options.OverrideDirs,
					}

					updateOutput, updateErr := UpdateTrustInfoForRecipes([]string{recipe}, updateOptions)
					logger.Logger(fmt.Sprintf("🔄 Trust update output for %s:\n%s", recipe, updateOutput), logger.LogDebug)

					if updateErr != nil {
						logger.Logger(fmt.Sprintf("⚠️ Failed to update trust info for %s: %v\n%s", recipe, updateErr, updateOutput), logger.LogWarning)
						result.TrustFailedRecipes = append(result.TrustFailedRecipes, recipe)
						result.InvalidRecipes = append(result.InvalidRecipes, recipe)
						continue
					}

					// Verify again after update
					success, _, secondVerifyOutput, verifyErr := VerifyTrustInfoForRecipes([]string{recipe}, verifyOptions)
					logger.Logger(fmt.Sprintf("🔍 Second trust verification for %s:\n%s", recipe, secondVerifyOutput), logger.LogDebug)

					if verifyErr != nil || !success {
						logger.Logger(fmt.Sprintf("⚠️ Failed to verify trust info for %s even after update:\n%s", recipe, secondVerifyOutput), logger.LogWarning)
						result.TrustFailedRecipes = append(result.TrustFailedRecipes, recipe)
						result.InvalidRecipes = append(result.InvalidRecipes, recipe)
						continue
					}
				} else {
					logger.Logger(fmt.Sprintf("⚠️ Trust verification failed for %s:\n%s", recipe, verifyOutput), logger.LogWarning)
					result.TrustFailedRecipes = append(result.TrustFailedRecipes, recipe)
					result.InvalidRecipes = append(result.InvalidRecipes, recipe)
					continue
				}
			}
		}

		// If we got here, the recipe is valid
		result.ValidRecipes = append(result.ValidRecipes, recipe)
	}

	logger.Logger(fmt.Sprintf("✅ Validation complete: %d valid, %d invalid, %d trust failed, %d missing",
		len(result.ValidRecipes), len(result.InvalidRecipes), len(result.TrustFailedRecipes), len(result.MissingRecipes)),
		logger.LogSuccess)

	return result, nil
}

// GenerateReportFromRun generates a structured report from recipe run results
func GenerateReportFromRun(results map[string]*RecipeBatchResult, format string) (string, error) {
	if format == "" {
		format = "text" // Default format
	}

	// Count statistics
	var total, success, failed, updated, clean int
	for _, result := range results {
		total++
		if result.ExecutionError == nil {
			success++
			if strings.Contains(result.Output, "Nothing new to download") {
				clean++
			} else {
				updated++
			}
		} else {
			failed++
		}
	}

	// Generate report based on format
	switch strings.ToLower(format) {
	case "text":
		var sb strings.Builder
		sb.WriteString("AutoPkg Run Report\n")
		sb.WriteString("=================\n\n")
		sb.WriteString(fmt.Sprintf("Total recipes: %d\n", total))
		sb.WriteString(fmt.Sprintf("Successful: %d\n", success))
		sb.WriteString(fmt.Sprintf("Failed: %d\n", failed))
		sb.WriteString(fmt.Sprintf("Updated: %d\n", updated))
		sb.WriteString(fmt.Sprintf("No changes: %d\n\n", clean))

		if failed > 0 {
			sb.WriteString("Failed Recipes:\n")
			for recipe, result := range results {
				if result.ExecutionError != nil {
					sb.WriteString(fmt.Sprintf("- %s: %v\n", recipe, result.ExecutionError))
				}
			}
			sb.WriteString("\n")
		}

		if updated > 0 {
			sb.WriteString("Updated Recipes:\n")
			for recipe, result := range results {
				if result.ExecutionError == nil && !strings.Contains(result.Output, "Nothing new to download") {
					sb.WriteString(fmt.Sprintf("- %s\n", recipe))
				}
			}
		}
		return sb.String(), nil

	case "json":
		type ReportEntry struct {
			Recipe  string `json:"recipe"`
			Success bool   `json:"success"`
			Error   string `json:"error,omitempty"`
			Output  string `json:"output,omitempty"`
			Updated bool   `json:"updated"`
		}

		reportData := struct {
			Total     int           `json:"total"`
			Success   int           `json:"success"`
			Failed    int           `json:"failed"`
			Updated   int           `json:"updated"`
			Clean     int           `json:"clean"`
			Timestamp string        `json:"timestamp"`
			Recipes   []ReportEntry `json:"recipes"`
		}{
			Total:     total,
			Success:   success,
			Failed:    failed,
			Updated:   updated,
			Clean:     clean,
			Timestamp: time.Now().Format(time.RFC3339),
			Recipes:   []ReportEntry{},
		}

		for recipe, result := range results {
			entry := ReportEntry{
				Recipe:  recipe,
				Success: result.ExecutionError == nil,
				Updated: result.ExecutionError == nil && !strings.Contains(result.Output, "Nothing new to download"),
			}
			if result.ExecutionError != nil {
				entry.Error = result.ExecutionError.Error()
			}
			entry.Output = result.Output
			reportData.Recipes = append(reportData.Recipes, entry)
		}

		jsonData, err := json.MarshalIndent(reportData, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON report: %w", err)
		}
		return string(jsonData), nil

	case "markdown":
		var sb strings.Builder
		sb.WriteString("# AutoPkg Run Report\n\n")
		sb.WriteString(fmt.Sprintf("- **Total recipes:** %d\n", total))
		sb.WriteString(fmt.Sprintf("- **Successful:** %d\n", success))
		sb.WriteString(fmt.Sprintf("- **Failed:** %d\n", failed))
		sb.WriteString(fmt.Sprintf("- **Updated:** %d\n", updated))
		sb.WriteString(fmt.Sprintf("- **No changes:** %d\n\n", clean))

		if failed > 0 {
			sb.WriteString("## Failed Recipes\n\n")
			for recipe, result := range results {
				if result.ExecutionError != nil {
					sb.WriteString(fmt.Sprintf("### %s\n\n", recipe))
					sb.WriteString(fmt.Sprintf("Error: %v\n\n", result.ExecutionError))
					if result.Output != "" {
						sb.WriteString("```\n")
						sb.WriteString(result.Output)
						sb.WriteString("\n```\n\n")
					}
				}
			}
		}

		if updated > 0 {
			sb.WriteString("## Updated Recipes\n\n")
			for recipe, result := range results {
				if result.ExecutionError == nil && !strings.Contains(result.Output, "Nothing new to download") {
					sb.WriteString(fmt.Sprintf("### %s\n\n", recipe))
					if result.Output != "" {
						sb.WriteString("```\n")
						sb.WriteString(result.Output)
						sb.WriteString("\n```\n\n")
					}
				}
			}
		}
		return sb.String(), nil

	default:
		return "", fmt.Errorf("unsupported report format: %s", format)
	}
}
