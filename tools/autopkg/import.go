package autopkg

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

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
