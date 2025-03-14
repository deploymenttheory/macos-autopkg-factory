package autopkg

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

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
