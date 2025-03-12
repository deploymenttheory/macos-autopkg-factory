// recipe_dependancies.go

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

// RecipeRepo represents a repository dependency.
type RecipeRepo struct {
	RecipeIdentifier string
	RepoName         string
	RepoURL          string
	IsParent         bool
}

// RecipeIndex represents the cached index of all recipes
type RecipeIndex struct {
	Identifiers map[string]RecipeIndexItem
	LastUpdated time.Time
}

// RecipeIndexItem represents a single recipe in the index
type RecipeIndexItem struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Repo        string   `json:"repo"`
	Path        string   `json:"path"`
	Parent      string   `json:"parent,omitempty"`
	Children    []string `json:"children,omitempty"`
	Shortname   string   `json:"shortname,omitempty"`
}

// Global cache of the recipe index
var recipeIndexCache *RecipeIndex

// Keep the existing regex pattern
var recipeRegex = regexp.MustCompile(`(?i)^.*\.recipe(?:\.yaml|\.plist)?$`)

// ResolveRecipeDependencies resolves all repository dependencies for a recipe using the index
func ResolveRecipeDependencies(recipeName string, useToken bool, prefsPath string, dryRun bool) ([]RecipeRepo, error) {
	logger.Logger(fmt.Sprintf("🔍 Resolving dependencies for: %s", recipeName), logger.LogDebug)

	// Check if recipeName is a valid recipe format
	isRecipeFile := recipeRegex.MatchString(recipeName)
	isRecipeIdentifier := strings.Contains(recipeName, ".")

	// If it's neither a recipe file nor an identifier, it's invalid
	if !isRecipeFile && !isRecipeIdentifier {
		logger.Logger("❌ Invalid recipe name format", logger.LogError)
		return nil, fmt.Errorf("invalid recipe name: %s", recipeName)
	}

	// Fetch the index
	index, err := FetchRecipeIndex(useToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recipe index: %w", err)
	}

	// Track all dependencies
	allDependencies := make(map[string]RecipeRepo)

	// Track repositories that need to be added
	reposToAdd := make(map[string]string)

	// Find the recipe in the index
	var matchedRecipes []string

	// Check if recipeName is already an identifier
	if _, exists := index.Identifiers[recipeName]; exists {
		matchedRecipes = []string{recipeName}
	} else {
		// Try to find by shortname, filename pattern, or other criteria
		for id, info := range index.Identifiers {
			// Match by shortname
			if info.Shortname == recipeName {
				matchedRecipes = append(matchedRecipes, id)
				continue
			}

			// Match by path/filename
			if strings.Contains(info.Path, recipeName) {
				matchedRecipes = append(matchedRecipes, id)
				continue
			}

			// Match by name
			if info.Name != "" && (info.Name == recipeName || strings.EqualFold(info.Name, recipeName)) {
				matchedRecipes = append(matchedRecipes, id)
				continue
			}
		}
	}

	// Process the matches
	if len(matchedRecipes) == 1 {
		logger.Logger(fmt.Sprintf("✅ Found single recipe match: %s", matchedRecipes[0]), logger.LogDebug)

		// Process the single recipe and its dependencies
		processRecipe(matchedRecipes[0], index, allDependencies, reposToAdd, useToken)
	} else if len(matchedRecipes) > 1 {
		logger.Logger(fmt.Sprintf("⚠️ Multiple matches found for recipe: %s (%d matches)", recipeName, len(matchedRecipes)), logger.LogWarning)

		// Log details about all matches
		for i, id := range matchedRecipes {
			info := index.Identifiers[id]
			logger.Logger(fmt.Sprintf("  Match %d: %s (from repo: %s)", i+1, id, info.Repo), logger.LogInfo)
		}

		// Process ALL matching recipes and their dependencies
		logger.Logger("📦 Adding repositories for all matching recipes and their parents", logger.LogInfo)

		for _, id := range matchedRecipes {
			logger.Logger(fmt.Sprintf("🔄 Processing dependencies for: %s", id), logger.LogDebug)

			// Process this recipe and add to dependencies
			processRecipe(id, index, allDependencies, reposToAdd, useToken)
		}
	} else {
		logger.Logger(fmt.Sprintf("❌ No matches found for recipe: %s", recipeName), logger.LogError)
		return nil, fmt.Errorf("no matches found for recipe: %s", recipeName)
	}

	if len(allDependencies) == 0 {
		logger.Logger(fmt.Sprintf("❌ No valid dependencies found for recipe: %s", recipeName), logger.LogError)
		return nil, fmt.Errorf("no valid dependencies found for recipe: %s", recipeName)
	}

	// If not in dry run mode, add the repositories
	if !dryRun && len(reposToAdd) > 0 {
		var repoNames []string
		for repoName := range reposToAdd {
			repoNames = append(repoNames, repoName)
		}

		logger.Logger(fmt.Sprintf("📦 Adding %d repositories for recipe %s", len(repoNames), recipeName), logger.LogInfo)

		// Use the existing AddRepo function
		_, err := AddRepo(repoNames, prefsPath)
		if err != nil {
			logger.Logger(fmt.Sprintf("⚠️ Error adding repositories: %v", err), logger.LogWarning)
			// Continue anyway to return the dependencies
		}
	}

	return mapToSlice(allDependencies), nil
}

// FetchRecipeIndex fetches and parses the AutoPkg index.json
func FetchRecipeIndex(useToken bool) (*RecipeIndex, error) {
	// Check if we have a recent cache
	if recipeIndexCache != nil && time.Since(recipeIndexCache.LastUpdated) < 24*time.Hour {
		return recipeIndexCache, nil
	}

	indexURL := "https://raw.githubusercontent.com/autopkg/index/refs/heads/main/index.json"

	logger.Logger("🔄 Fetching AutoPkg recipe index", logger.LogDebug)

	// Use token if provided
	var cmd *exec.Cmd
	if useToken {
		token := os.Getenv("GITHUB_TOKEN")
		if token != "" {
			cmd = exec.Command("curl", "-sL", "-H", fmt.Sprintf("Authorization: token %s", token), indexURL)
			logger.Logger("🔐 Using GitHub token for authentication", logger.LogDebug)
		} else {
			logger.Logger("⚠️ GitHub token requested but not found in environment", logger.LogWarning)
			cmd = exec.Command("curl", "-sL", indexURL)
		}
	} else {
		cmd = exec.Command("curl", "-sL", indexURL)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Logger("❌ Failed to fetch AutoPkg index", logger.LogError)
		return nil, fmt.Errorf("failed to fetch index: %w", err)
	}

	// Parse the JSON
	var index map[string]json.RawMessage
	if err := json.Unmarshal(output, &index); err != nil {
		logger.Logger("❌ Failed to parse AutoPkg index JSON", logger.LogError)
		return nil, fmt.Errorf("failed to parse index: %w", err)
	}

	// Extract the identifiers section
	identifiersRaw, exists := index["identifiers"]
	if !exists {
		logger.Logger("❌ Index JSON does not contain 'identifiers' section", logger.LogError)
		return nil, fmt.Errorf("invalid index format: missing identifiers section")
	}

	var identifiers map[string]RecipeIndexItem
	if err := json.Unmarshal(identifiersRaw, &identifiers); err != nil {
		logger.Logger("❌ Failed to parse identifiers section", logger.LogError)
		return nil, fmt.Errorf("failed to parse identifiers: %w", err)
	}

	// Update cache
	recipeIndexCache = &RecipeIndex{
		Identifiers: identifiers,
		LastUpdated: time.Now(),
	}

	logger.Logger(fmt.Sprintf("✅ Successfully loaded %d recipes from index", len(identifiers)), logger.LogDebug)

	return recipeIndexCache, nil
}

// processRecipe recursively processes a recipe and its dependencies
func processRecipe(identifier string, index *RecipeIndex, allDependencies map[string]RecipeRepo, reposToAdd map[string]string, useToken bool) {
	// Check if we already processed this recipe
	if _, exists := allDependencies[identifier]; exists {
		return
	}

	info, exists := index.Identifiers[identifier]
	if !exists {
		logger.Logger(fmt.Sprintf("⚠️ Recipe identifier not found in index: %s", identifier), logger.LogWarning)
		return
	}

	// Add this recipe's repository
	if info.Repo != "" {
		// Using existing VerifyRepoExists function to check if repo exists
		if VerifyRepoExists(info.Repo, useToken) {
			reposToAdd[info.Repo] = info.Repo

			// Add the recipe to dependencies
			allDependencies[identifier] = RecipeRepo{
				RecipeIdentifier: identifier,
				RepoName:         info.Repo,
				RepoURL:          fmt.Sprintf("https://github.com/%s", info.Repo),
				IsParent:         false,
			}

			// Process parent recipe if it exists
			if info.Parent != "" {
				logger.Logger(fmt.Sprintf("🧩 Found parent recipe: %s", info.Parent), logger.LogDebug)

				parentInfo, exists := index.Identifiers[info.Parent]
				if exists {
					// Process the parent recipe
					processRecipe(info.Parent, index, allDependencies, reposToAdd, useToken)

					if parentInfo.Repo != "" && VerifyRepoExists(parentInfo.Repo, useToken) {
						// Mark this parent in our dependencies
						allDependencies[info.Parent] = RecipeRepo{
							RecipeIdentifier: info.Parent,
							RepoName:         parentInfo.Repo,
							RepoURL:          fmt.Sprintf("https://github.com/%s", parentInfo.Repo),
							IsParent:         true,
						}
					}
				} else {
					logger.Logger(fmt.Sprintf("⚠️ Parent recipe not found in index: %s", info.Parent), logger.LogWarning)
					// Add the parent as unknown
					allDependencies[info.Parent] = RecipeRepo{
						RecipeIdentifier: info.Parent,
						RepoName:         "unknown",
						RepoURL:          "",
						IsParent:         true,
					}
				}
			}

			// Process children recipes if available
			if len(info.Children) > 0 {
				for _, childID := range info.Children {
					logger.Logger(fmt.Sprintf("🧩 Found child recipe: %s", childID), logger.LogDebug)
					processRecipe(childID, index, allDependencies, reposToAdd, useToken)
				}
			}
		} else {
			logger.Logger(fmt.Sprintf("⚠️ Repository %s does not exist, skipping", info.Repo), logger.LogWarning)
		}
	}
}

// Continue using the existing VerifyRepoExists function
// VerifyRepoExists checks if a repository exists on GitHub.
func VerifyRepoExists(repoName string, useToken bool) bool {
	repoURL := fmt.Sprintf("https://github.com/%s", repoName)
	logger.Logger(fmt.Sprintf("🔍 Verifying repository: %s", repoURL), logger.LogDebug)

	var cmd *exec.Cmd

	if useToken {
		// Try to use GITHUB_TOKEN for authentication if available
		token := os.Getenv("GITHUB_TOKEN")
		if token != "" {
			// For git operations with token, we need to format it as https://token@github.com/...
			authRepoURL := fmt.Sprintf("https://%s@github.com/%s", token, repoName)
			cmd = exec.Command("git", "ls-remote", "--exit-code", authRepoURL+".git")
			logger.Logger("🔐 Using GitHub token for authentication", logger.LogDebug)
		} else {
			logger.Logger("⚠️ GitHub token requested but not found in environment", logger.LogWarning)
			cmd = exec.Command("git", "ls-remote", "--exit-code", repoURL+".git")
		}
	} else {
		cmd = exec.Command("git", "ls-remote", "--exit-code", repoURL+".git")
	}

	if err := cmd.Run(); err != nil {
		logger.Logger(fmt.Sprintf("⚠️ Repository does not exist: %s", repoURL), logger.LogWarning)
		return false
	}
	logger.Logger(fmt.Sprintf("✅ Repository exists: %s", repoURL), logger.LogDebug)
	return true
}

func mapToSlice(m map[string]RecipeRepo) []RecipeRepo {
	result := []RecipeRepo{}
	for _, v := range m {
		result = append(result, v)
	}
	return result
}
