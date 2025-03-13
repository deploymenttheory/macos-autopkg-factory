package autopkg

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

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
