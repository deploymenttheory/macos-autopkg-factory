// recipe_dependancies.go
package autopkg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
	"gopkg.in/yaml.v2"
	"howett.net/plist"
)

// RecipeRepo represents a repository dependency.
type RecipeRepo struct {
	RecipeIdentifier string
	RepoName         string
	RepoURL          string
	IsParent         bool
}

// RecipeMatch represents a single match found for a recipe.
type RecipeMatch struct {
	Repo string
	Path string
}

var recipeRegex = regexp.MustCompile(`(?i)^.*\.recipe(?:\.yaml|\.plist)?$`)

// ResolveRecipeDependencies resolves all repository dependencies for a recipe.
// Now handles multiple matches for a recipe.
// ResolveRecipeDependencies resolves all repository dependencies for a recipe and optionally adds them.
func ResolveRecipeDependencies(recipeName string, useToken bool, prefsPath string, dryRun bool) ([]RecipeRepo, error) {
	logger.Logger(fmt.Sprintf("🔍 Resolving dependencies for: %s", recipeName), logger.LogDebug)

	// Search for all matching recipes
	matches, err := Search(recipeName, useToken, prefsPath)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no matches found for recipe: %s", recipeName)
	}

	// Track all dependencies across all matches
	allDependencies := make(map[string]RecipeRepo)

	// Track repositories that need to be added
	reposToAdd := make(map[string]string) // map[repoName]repoUrl (as expected by AddRepo)

	// Process each match to find its dependencies
	for _, match := range matches {
		if !VerifyRepoExists(match.Repo) {
			logger.Logger(fmt.Sprintf("⚠️ Repository %s does not exist, skipping", match.Repo), logger.LogWarning)
			continue
		}

		// Add this repo to the list that needs to be added
		// Just add the repo name itself, not the full URL
		// The AddRepo function will format it correctly
		reposToAdd[match.Repo] = match.Repo

		// Parse the recipe file to find dependencies
		identifier, dependencies, err := ParseRecipeFile(match.Repo, match.Path, useToken, prefsPath)
		if err != nil {
			logger.Logger(fmt.Sprintf("⚠️ Error parsing recipe file for %s: %v, continuing with other matches", match.Path, err), logger.LogWarning)
			continue
		}

		// Add this recipe to dependencies
		allDependencies[identifier] = RecipeRepo{
			RecipeIdentifier: identifier,
			RepoName:         match.Repo,
			RepoURL:          fmt.Sprintf("https://github.com/autopkg/%s", match.Repo),
			IsParent:         false,
		}

		// Add all parent dependencies
		for _, dep := range dependencies {
			allDependencies[dep.RecipeIdentifier] = dep

			// Add parent repo to the list that needs to be added
			if dep.RepoName != "" && dep.RepoName != "unknown" {
				reposToAdd[dep.RepoName] = dep.RepoName
			}
		}
	}

	logger.Logger("✅ Dependencies resolved", logger.LogDebug)

	if len(allDependencies) == 0 {
		return nil, fmt.Errorf("no valid dependencies found for recipe: %s", recipeName)
	}

	// If not in dry run mode, add the repositories
	if !dryRun && len(reposToAdd) > 0 {
		var repoNames []string
		for _, repoName := range reposToAdd {
			repoNames = append(repoNames, repoName)
		}

		logger.Logger(fmt.Sprintf("📦 Adding %d repositories for recipe %s", len(repoNames), recipeName), logger.LogInfo)
		_, err := AddRepo(repoNames, prefsPath)
		if err != nil {
			logger.Logger(fmt.Sprintf("⚠️ Error adding repositories: %v", err), logger.LogWarning)
			// Continue anyway to return the dependencies
		}
	}

	return mapToSlice(allDependencies), nil
}

// Search searches for a recipe using autopkg search command.
// Now returns all matches instead of just the first one.
func Search(recipeName string, useToken bool, prefsPath string) ([]RecipeMatch, error) {
	logger.Logger(fmt.Sprintf("🔍 Searching for recipe: %s", recipeName), logger.LogDebug)

	isRecipeFile := recipeRegex.MatchString(recipeName)
	isRecipeIdentifier := strings.Contains(recipeName, ".")

	// If it's neither a recipe file nor an identifier, it's invalid
	if !isRecipeFile && !isRecipeIdentifier {
		logger.Logger("❌ Invalid recipe name format", logger.LogError)
		return nil, fmt.Errorf("invalid recipe name: %s", recipeName)
	}

	if strings.HasPrefix(prefsPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			prefsPath = filepath.Join(homeDir, prefsPath[2:])
		}
	}

	searchOptions := &SearchOptions{
		PrefsPath: prefsPath,
		UseToken:  useToken,
	}

	var searchTerm string
	if isRecipeIdentifier && !isRecipeFile {
		// Convert identifier to recipe name for search
		// Example: com.github.rtrouton.pkg.microsoftteams -> MicrosoftTeams.pkg.recipe
		parts := strings.Split(recipeName, ".")

		// Look for recipe type (pkg, download, install, etc.)
		var recipeType string
		var appName string

		// Find the app name part (usually the last part)
		if len(parts) > 0 {
			appName = parts[len(parts)-1]

			// Check for recipe type before the app name
			if len(parts) > 1 {
				possibleType := parts[len(parts)-2]
				// Common recipe types
				recipeTypes := []string{"pkg", "download", "install", "munki", "jamf", "intune"}

				for _, rt := range recipeTypes {
					if possibleType == rt {
						recipeType = rt
						break
					}
				}
			}
		}

		// Simple camel case conversion for app name
		titleCaseAppName := appName
		if len(appName) > 0 {
			titleCaseAppName = strings.ToUpper(appName[:1]) + appName[1:]
		}

		if recipeType != "" {
			searchTerm = titleCaseAppName + "." + recipeType + ".recipe"
		} else {
			searchTerm = titleCaseAppName + ".recipe"
		}

		logger.Logger(fmt.Sprintf("🔄 Converting identifier to recipe name: %s -> %s", recipeName, searchTerm), logger.LogDebug)
	} else {
		searchTerm = recipeName
	}

	outputStr, err := SearchRecipes(searchTerm, searchOptions)
	if err != nil {
		logger.Logger(fmt.Sprintf("❌ autopkg search command failed: %v", err), logger.LogError)
		logger.Logger(fmt.Sprintf("Output: %s", outputStr), logger.LogDebug)
		return nil, fmt.Errorf("autopkg search failed: %w", err)
	}

	matches, err := parseSearchOutput(outputStr)
	if err != nil {
		logger.Logger(fmt.Sprintf("❌ Failed to parse search output: %v", err), logger.LogError)
		return nil, err
	}

	if len(matches) == 0 {
		logger.Logger("⚠️ No valid recipes found", logger.LogWarning)
		logger.Logger(fmt.Sprintf("Search output: %s", outputStr), logger.LogDebug)
		return nil, fmt.Errorf("no valid recipes found for %s", recipeName)
	}

	for i, match := range matches {
		logger.Logger(fmt.Sprintf("✅ Recipe found (%d/%d): Repo=%s, Path=%s",
			i+1, len(matches), match.Repo, match.Path), logger.LogDebug)
	}

	return matches, nil
}

// parseSearchOutput parses the output of autopkg search command to extract all matches.
func parseSearchOutput(output string) ([]RecipeMatch, error) {
	lines := strings.Split(output, "\n")
	var matches []RecipeMatch

	headerIndex := -1
	for i, line := range lines {
		if strings.Contains(line, "Name") && strings.Contains(line, "Repo") && strings.Contains(line, "Path") {
			headerIndex = i
			break
		}
	}

	if headerIndex == -1 {
		return nil, fmt.Errorf("could not find header in search output")
	}

	sepIndex := -1
	for i := headerIndex + 1; i < len(lines); i++ {
		if strings.Contains(lines[i], "----") {
			sepIndex = i
			break
		}
	}

	if sepIndex == -1 {
		return nil, fmt.Errorf("could not find separator in search output")
	}

	// Analyze the header line to determine column positions
	headerLine := lines[headerIndex]

	// Find starting positions of each column
	namePos := strings.Index(headerLine, "Name")
	repoPos := strings.Index(headerLine, "Repo")
	pathPos := strings.Index(headerLine, "Path")

	if namePos == -1 || repoPos == -1 || pathPos == -1 {
		return nil, fmt.Errorf("invalid header format")
	}

	// Process each line after the separator
	for i := sepIndex + 1; i < len(lines); i++ {
		line := lines[i]

		// Skip empty lines or lines with "To add a new recipe repo"
		if strings.TrimSpace(line) == "" || strings.Contains(line, "To add a new recipe repo") {
			continue
		}

		if len(line) < pathPos {
			continue
		}

		// Extract repo and path based on column positions
		// We need to handle the case where the content of a column might be shorter than the column width
		var repo, path string

		// Extract repo
		repoStart := repoPos
		repoEnd := pathPos
		if repoStart < len(line) {
			repoStr := line[repoStart:min(repoEnd, len(line))]
			repo = strings.TrimSpace(repoStr)
		}

		// Extract path
		if pathPos < len(line) {
			path = strings.TrimSpace(line[pathPos:])
		}

		// Only add valid matches
		if repo != "" && path != "" {
			matches = append(matches, RecipeMatch{
				Repo: repo,
				Path: path,
			})
		}
	}

	return matches, nil
}

// helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// VerifyRepoExists checks if a repository exists on GitHub.
func VerifyRepoExists(repoName string) bool {
	repoURL := fmt.Sprintf("https://github.com/autopkg/%s", repoName)
	logger.Logger(fmt.Sprintf("🔍 Verifying repository: %s", repoURL), logger.LogDebug)

	cmd := exec.Command("git", "ls-remote", "--exit-code", repoURL+".git")
	if err := cmd.Run(); err != nil {
		logger.Logger(fmt.Sprintf("⚠️ Repository does not exist: %s", repoURL), logger.LogWarning)
		return false
	}
	logger.Logger(fmt.Sprintf("✅ Repository exists: %s", repoURL), logger.LogDebug)
	return true
}

// ParseRecipeFile parses a recipe file (YAML or plist) and extracts details.
func ParseRecipeFile(repo, path string, useToken bool, prefsPath string) (string, []RecipeRepo, error) {
	// Build the raw URL for GitHub content (not blob view)
	repoURL := fmt.Sprintf("https://raw.githubusercontent.com/autopkg/%s/master/%s", repo, path)
	logger.Logger(fmt.Sprintf("🔍 Fetching recipe file: %s", repoURL), logger.LogDebug)

	cmd := exec.Command("curl", "-sL", repoURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Logger("❌ Failed to fetch recipe file", logger.LogError)
		return "", nil, fmt.Errorf("failed to fetch recipe file: %w", err)
	}

	fileExt := filepath.Ext(path)
	var recipeData map[string]interface{}
	if fileExt == ".yaml" {
		logger.Logger("📄 Parsing YAML recipe file", logger.LogDebug)
		if err := yaml.Unmarshal(output, &recipeData); err != nil {
			logger.Logger("❌ YAML parsing failed", logger.LogError)
			return "", nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	} else {
		logger.Logger("📄 Parsing Plist recipe file", logger.LogDebug)
		var plistData interface{}
		if _, err := plist.Unmarshal(output, &plistData); err != nil {
			logger.Logger("❌ Plist parsing failed", logger.LogError)
			return "", nil, fmt.Errorf("failed to parse Plist: %w", err)
		}
		recipeData, _ = plistData.(map[string]interface{})
	}

	identifier, _ := recipeData["Identifier"].(string)
	parent, _ := recipeData["ParentRecipe"].(string)
	deps := []RecipeRepo{}
	if parent != "" {
		logger.Logger(fmt.Sprintf("🧩 Found parent recipe: %s", parent), logger.LogDebug)

		// Search for parent recipe(s)
		parentMatches, err := Search(parent, useToken, prefsPath)
		if err != nil || len(parentMatches) == 0 {
			logger.Logger(fmt.Sprintf("⚠️ Could not find parent recipe: %s, error: %v", parent, err), logger.LogWarning)
			// Add the parent as a dependency even if we can't resolve it further
			// This preserves the dependency information
			deps = append(deps, RecipeRepo{
				RecipeIdentifier: parent,
				RepoName:         "unknown",
				RepoURL:          "",
				IsParent:         true,
			})
		} else {
			// Add all found parent recipes
			for _, parentMatch := range parentMatches {
				if VerifyRepoExists(parentMatch.Repo) {
					deps = append(deps, RecipeRepo{
						RecipeIdentifier: parent,
						RepoName:         parentMatch.Repo,
						RepoURL:          fmt.Sprintf("https://github.com/autopkg/%s", parentMatch.Repo),
						IsParent:         true,
					})

					// Recursively resolve this parent's dependencies
					parentIdentifier, parentDeps, err := ParseRecipeFile(parentMatch.Repo, parentMatch.Path, useToken, prefsPath)
					if err == nil && parentIdentifier != "" {
						deps = append(deps, parentDeps...)
					}
				}
			}
		}
	}
	return identifier, deps, nil
}

func mapToSlice(m map[string]RecipeRepo) []RecipeRepo {
	result := []RecipeRepo{}
	for _, v := range m {
		result = append(result, v)
	}
	return result
}
