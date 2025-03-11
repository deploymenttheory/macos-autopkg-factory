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

var recipeRegex = regexp.MustCompile(`(?i)^.*\.recipe(?:\.yaml|\.plist)?$`)

// ResolveRecipeDependencies resolves all repository dependencies for a recipe.
func ResolveRecipeDependencies(recipeName string, useToken bool, prefsPath string) ([]RecipeRepo, error) {
	logger.Logger(fmt.Sprintf("🔍 Resolving dependencies for: %s", recipeName), logger.LogDebug)

	repo, path, err := Search(recipeName, useToken, prefsPath)
	if err != nil {
		return nil, err
	}

	if !VerifyRepoExists(repo) {
		return nil, fmt.Errorf("repository %s does not exist", repo)
	}

	identifier, dependencies, err := ParseRecipeFile(repo, path)
	if err != nil {
		return nil, err
	}

	logger.Logger("✅ Dependencies resolved", logger.LogDebug)

	allDependencies := map[string]RecipeRepo{
		identifier: {RecipeIdentifier: identifier, RepoName: repo, RepoURL: fmt.Sprintf("https://github.com/autopkg/%s", repo), IsParent: false},
	}

	for _, dep := range dependencies {
		allDependencies[dep.RecipeIdentifier] = dep
	}

	return mapToSlice(allDependencies), nil
}

// Search searches for a recipe using autopkg search command.
func Search(recipeName string, useToken bool, prefsPath string) (string, string, error) {
	logger.Logger(fmt.Sprintf("🔍 Searching for recipe: %s", recipeName), logger.LogDebug)

	if !recipeRegex.MatchString(recipeName) {
		logger.Logger("❌ Invalid recipe name format", logger.LogError)
		return "", "", fmt.Errorf("invalid recipe name: %s", recipeName)
	}

	// Handle tilde expansion in prefsPath
	if strings.HasPrefix(prefsPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			prefsPath = filepath.Join(homeDir, prefsPath[2:])
		}
	}

	// Use the existing SearchRecipes function
	searchOptions := &SearchOptions{
		PrefsPath: prefsPath,
		UseToken:  useToken,
	}

	outputStr, err := SearchRecipes(recipeName, searchOptions)
	if err != nil {
		logger.Logger(fmt.Sprintf("❌ autopkg search command failed: %v", err), logger.LogError)
		logger.Logger(fmt.Sprintf("Output: %s", outputStr), logger.LogDebug)
		return "", "", fmt.Errorf("autopkg search failed: %w", err)
	}

	// Parse the output
	lines := strings.Split(outputStr, "\n")

	// Skip header lines and look for the first valid result
	foundHeader := false
	for _, line := range lines {
		// Check for header line
		if !foundHeader {
			if strings.Contains(line, "Name") && strings.Contains(line, "Repo") && strings.Contains(line, "Path") {
				foundHeader = true
				continue
			}
		}

		// Skip separator line or empty lines
		if strings.HasPrefix(line, "----") || strings.TrimSpace(line) == "" {
			continue
		}

		// Only process lines after the header
		if foundHeader {
			// Split the line into fields
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				// Fields should be Name, Repo, Path
				// But the name might contain spaces, so we need to be careful

				// Find the repo column - it's typically the second-to-last column
				repoIndex := len(fields) - 2
				pathIndex := len(fields) - 1

				if repoIndex >= 0 && pathIndex > repoIndex {
					repo := fields[repoIndex]
					path := fields[pathIndex]
					logger.Logger(fmt.Sprintf("✅ Recipe found: Repo=%s, Path=%s", repo, path), logger.LogDebug)
					return repo, path, nil
				}
			}
		}
	}

	logger.Logger("⚠️ No valid recipe found", logger.LogWarning)
	logger.Logger(fmt.Sprintf("Search output: %s", outputStr), logger.LogDebug)
	return "", "", fmt.Errorf("no valid recipe found for %s", recipeName)
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
func ParseRecipeFile(repo, path string) (string, []RecipeRepo, error) {
	repoURL := fmt.Sprintf("https://github.com/autopkg/%s/blob/master/%s", repo, path)
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
		parentRepo, _, err := Search(parent, true, "")
		if err == nil {
			deps = append(deps, RecipeRepo{
				RecipeIdentifier: parent,
				RepoName:         parentRepo,
				RepoURL:          fmt.Sprintf("https://github.com/autopkg/%s", parentRepo),
				IsParent:         true,
			})
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
