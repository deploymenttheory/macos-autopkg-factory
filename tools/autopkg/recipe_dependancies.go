package autopkg

import (
	"fmt"
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

// SearchRecipe searches for a recipe using autopkg search command.
func SearchRecipe(recipeName string, useToken bool, prefsPath string) (string, string, error) {
	logger.Logger(fmt.Sprintf("🔍 Searching for recipe: %s", recipeName), logger.LogDebug)

	if !recipeRegex.MatchString(recipeName) {
		logger.Logger("❌ Invalid recipe name format", logger.LogError)
		return "", "", fmt.Errorf("invalid recipe name: %s", recipeName)
	}

	cmdArgs := []string{"search"}
	if prefsPath != "" {
		cmdArgs = append(cmdArgs, "--prefs", prefsPath)
	}
	if useToken {
		cmdArgs = append(cmdArgs, "--use-token")
	}
	cmdArgs = append(cmdArgs, recipeName)

	cmd := exec.Command("autopkg", cmdArgs...)
	logger.Logger(fmt.Sprintf("🖥️  Running command: %s", strings.Join(cmd.Args, " ")), logger.LogDebug)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Logger("❌ autopkg search command failed", logger.LogError)
		return "", "", fmt.Errorf("autopkg search failed: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 3 {
			logger.Logger(fmt.Sprintf("✅ Recipe found: Repo=%s, Path=%s", fields[1], fields[2]), logger.LogDebug)
			return fields[1], fields[2], nil
		}
	}

	logger.Logger("⚠️ No valid recipe found", logger.LogWarning)
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
		parentRepo, _, err := SearchRecipe(parent, true, "")
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

// ResolveRecipeDependencies resolves all repository dependencies for a recipe.
func ResolveRecipeDependencies(recipeName string, useToken bool, prefsPath string) ([]RecipeRepo, error) {
	logger.Logger(fmt.Sprintf("🔍 Resolving dependencies for: %s", recipeName), logger.LogDebug)

	repo, path, err := SearchRecipe(recipeName, useToken, prefsPath)
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

func mapToSlice(m map[string]RecipeRepo) []RecipeRepo {
	result := []RecipeRepo{}
	for _, v := range m {
		result = append(result, v)
	}
	return result
}

// import (
// 	"fmt"
// 	"os"
// 	"os/exec"
// 	"path/filepath"
// 	"regexp"
// 	"strings"

// 	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
// 	"gopkg.in/yaml.v2"
// 	"howett.net/plist"
// )

// // RecipeRepoAnalysisOptions contains options for analyzing recipe dependencies
// type RecipeRepoAnalysisOptions struct {
// 	RecipePaths       []string // Paths to recipe files to analyze
// 	RecipeIdentifiers []string // Recipe identifiers to analyze (alternative to paths)
// 	SearchDirs        []string // Directories to search for recipes by identifier
// 	IncludeParents    bool     // Whether to include parent recipes in analysis
// 	MaxDepth          int      // Maximum recursion depth for parent recipes (0 = unlimited)
// 	VerifyRepoExists  bool     // Check if repositories actually exist
// 	PrefsPath         string   // Path to AutoPkg preferences
// 	IncludeBase       bool     // Include base autopkg repo
// }

// // RecipeRepoDependency represents a repository dependency for a recipe
// type RecipeRepoDependency struct {
// 	RecipeIdentifier string // Recipe identifier
// 	RepoOwner        string // GitHub repository owner
// 	RepoName         string // Repository name
// 	RepoURL          string // Full repository URL
// 	IsParent         bool   // Whether this is from a parent recipe
// 	Source           string // Source recipe that referenced this dependency
// }

// // AnalyzeRecipeRepoDependencies analyzes recipes to determine required repositories
// // This function identifies the repositories needed by the specified recipes and their
// // parent recipes, helping to dynamically determine which repositories to add.
// func AnalyzeRecipeRepoDependencies(options *RecipeRepoAnalysisOptions) ([]RecipeRepoDependency, error) {
// 	if options == nil {
// 		options = &RecipeRepoAnalysisOptions{
// 			IncludeParents:   true,
// 			MaxDepth:         5,
// 			VerifyRepoExists: true,
// 			IncludeBase:      true,
// 		}
// 	}

// 	logger.Logger("🔍 Analyzing recipe repository dependencies", logger.LogInfo)

// 	// Result set
// 	var dependencies []RecipeRepoDependency

// 	// Process recipe paths first
// 	for _, recipePath := range options.RecipePaths {
// 		// Set of already processed identifiers to avoid duplicates and circular references
// 		processedIdentifiers := make(map[string]bool)

// 		deps, err := analyzeRecipeFile(recipePath, options, processedIdentifiers, 0)
// 		if err != nil {
// 			logger.Logger(fmt.Sprintf("⚠️ Error analyzing recipe file %s: %v", recipePath, err), logger.LogError)
// 			continue
// 		}
// 		dependencies = append(dependencies, deps...)
// 	}

// 	// Process recipe identifiers
// 	for _, recipeIdentifier := range options.RecipeIdentifiers {
// 		// A more robust approach using regex
// 		isFullIdentifier := regexp.MustCompile(`^com\.github\.[^.]+\.[^.]+\..+$`).MatchString(recipeIdentifier)
// 		hasRecipeExtension := regexp.MustCompile(`\.[a-zA-Z0-9_-]+\.recipe(?:\.yaml|\.plist)?$`).MatchString(recipeIdentifier)

// 		isSimpleName := !isFullIdentifier && (hasRecipeExtension || !strings.Contains(recipeIdentifier, "."))

// 		if isSimpleName {
// 			// Use the search method for simple names
// 			searchDeps, err := findRecipeByNameAndExtractRepo(recipeIdentifier, options.PrefsPath)
// 			if err != nil {
// 				logger.Logger(fmt.Sprintf("⚠️ Search failed for recipe %s: %v", recipeIdentifier, err), logger.LogError)
// 				continue
// 			}

// 			if len(searchDeps) > 0 {
// 				logger.Logger(fmt.Sprintf("✅ Found %d potential repositories for %s via search", len(searchDeps), recipeIdentifier), logger.LogSuccess)
// 				dependencies = append(dependencies, searchDeps...)
// 				continue
// 			}
// 		}

// 		// Try the traditional method with full identifiers
// 		// Set of already processed identifiers to avoid duplicates and circular references
// 		processedIdentifiers := make(map[string]bool)

// 		// Find the recipe file for this identifier
// 		recipePath, err := findRecipeByIdentifier(recipeIdentifier, options.SearchDirs, options.PrefsPath)
// 		if err != nil {
// 			logger.Logger(fmt.Sprintf("⚠️ Could not find recipe for identifier %s: %v", recipeIdentifier, err), logger.LogWarning)
// 			continue
// 		}

// 		deps, err := analyzeRecipeFile(recipePath, options, processedIdentifiers, 0)
// 		if err != nil {
// 			logger.Logger(fmt.Sprintf("⚠️ Error analyzing recipe for identifier %s: %v", recipeIdentifier, err), logger.LogError)
// 			continue
// 		}
// 		dependencies = append(dependencies, deps...)
// 	}

// 	// Add autopkg base repo if requested
// 	if options.IncludeBase {
// 		baseDep := RecipeRepoDependency{
// 			RecipeIdentifier: "com.github.autopkg.recipes",
// 			RepoOwner:        "autopkg",
// 			RepoName:         "recipes",
// 			RepoURL:          "https://github.com/autopkg/recipes",
// 			IsParent:         false,
// 			Source:           "base",
// 		}

// 		// Check if it's already in the dependencies
// 		found := false
// 		for _, dep := range dependencies {
// 			if dep.RepoURL == baseDep.RepoURL {
// 				found = true
// 				break
// 			}
// 		}

// 		if !found {
// 			dependencies = append(dependencies, baseDep)
// 		}
// 	}

// 	// Remove duplicates based on repository URL
// 	uniqueDeps := make(map[string]RecipeRepoDependency)
// 	for _, dep := range dependencies {
// 		uniqueDeps[dep.RepoURL] = dep
// 	}

// 	dependencies = []RecipeRepoDependency{}
// 	for _, dep := range uniqueDeps {
// 		dependencies = append(dependencies, dep)
// 	}

// 	// Verify repositories if requested
// 	if options.VerifyRepoExists {
// 		var verifiedDeps []RecipeRepoDependency
// 		for _, dep := range dependencies {
// 			exists, err := verifyRepoExists(dep.RepoURL)
// 			if err != nil {
// 				logger.Logger(fmt.Sprintf("⚠️ Error verifying repo %s: %v", dep.RepoURL, err), logger.LogError)
// 				continue
// 			}

// 			if exists {
// 				verifiedDeps = append(verifiedDeps, dep)
// 			} else {
// 				logger.Logger(fmt.Sprintf("⚠️ Repository does not exist: %s", dep.RepoURL), logger.LogWarning)
// 			}
// 		}
// 		dependencies = verifiedDeps
// 	}

// 	logger.Logger(fmt.Sprintf("✅ Found %d repository dependencies", len(dependencies)), logger.LogSuccess)
// 	return dependencies, nil
// }

// // findRecipeByNameAndExtractRepo searches for a recipe by name and extracts repository information
// // This function uses the autopkg search command to find repositories that contain recipes matching
// // the provided name, then extracts the repository information for each match.
// func findRecipeByNameAndExtractRepo(recipeName string, prefsPath string) ([]RecipeRepoDependency, error) {
// 	searchOptions := &SearchOptions{
// 		PrefsPath: prefsPath,
// 		UseToken:  true, // Use token to avoid GitHub API rate limiting
// 	}

// 	// Call the wrapped search function
// 	output, err := SearchRecipes(recipeName, searchOptions)
// 	if err != nil {
// 		return nil, fmt.Errorf("search recipes failed: %w", err)
// 	}

// 	// Parse the output
// 	lines := strings.Split(output, "\n")

// 	var dependencies []RecipeRepoDependency

// 	// Skip header lines
// 	foundHeader := false
// 	for _, line := range lines {
// 		if !foundHeader {
// 			if strings.Contains(line, "Name") && strings.Contains(line, "Repo") && strings.Contains(line, "Path") {
// 				foundHeader = true
// 				continue
// 			}
// 			continue
// 		}

// 		// Skip separator line with dashes
// 		if strings.HasPrefix(line, "----") {
// 			continue
// 		}

// 		// Skip empty lines
// 		if strings.TrimSpace(line) == "" {
// 			continue
// 		}

// 		// Parse the result line - columns are Name, Repo, Path
// 		// The spacing is inconsistent so we need to be careful
// 		fields := strings.Fields(line)
// 		if len(fields) < 3 {
// 			continue
// 		}

// 		// Since the fields might have spaces, we need to reconstruct
// 		var recipeName, repoName string

// 		// Find the repository name column - typically the second column but may vary
// 		repoStartIdx := -1
// 		for i := 1; i < len(fields); i++ {
// 			if strings.HasSuffix(fields[i], "-recipes") {
// 				repoStartIdx = i
// 				break
// 			}
// 		}

// 		if repoStartIdx == -1 {
// 			// Couldn't find the repo column
// 			continue
// 		}

// 		recipeName = strings.Join(fields[:repoStartIdx], " ")
// 		repoName = fields[repoStartIdx]

// 		// Extract owner from repo name
// 		var repoOwner string
// 		if strings.HasSuffix(repoName, "-recipes") {
// 			repoOwner = strings.TrimSuffix(repoName, "-recipes")
// 		} else {
// 			// Handle unusual repo naming
// 			repoOwner = repoName
// 		}

// 		// Build the repository URL
// 		repoURL := fmt.Sprintf("https://github.com/autopkg/%s", repoName)

// 		// Create dependency object
// 		dependency := RecipeRepoDependency{
// 			RecipeIdentifier: recipeName, // This is just the name, not the full identifier
// 			RepoOwner:        repoOwner,
// 			RepoName:         repoName,
// 			RepoURL:          repoURL,
// 			IsParent:         false,
// 			Source:           "search",
// 		}

// 		dependencies = append(dependencies, dependency)
// 	}

// 	return dependencies, nil
// }

// // analyzeRecipeFile parses a recipe file and extracts repository dependencies
// // This function reads a recipe file, extracts its repository information, and recursively
// // analyzes parent recipes to build a complete dependency tree.
// // analyzeRecipeFile analyzes a single recipe file for dependencies.
// func analyzeRecipeFile(recipePath string, options *RecipeRepoAnalysisOptions, processedIdentifiers map[string]bool, depth int) ([]RecipeRepoDependency, error) {
// 	if depth > options.MaxDepth && options.MaxDepth > 0 {
// 		return nil, nil // Stop recursion if depth limit is reached
// 	}

// 	fileExt := filepath.Ext(recipePath)
// 	isYAML := fileExt == ".yaml"

// 	logger.Logger(fmt.Sprintf("🔍 Analyzing recipe file: %s", recipePath), logger.LogInfo)

// 	// Read the recipe file
// 	data, err := os.ReadFile(recipePath)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read recipe file: %w", err)
// 	}

// 	var recipeData map[string]interface{}

// 	if isYAML {
// 		// Parse YAML recipe
// 		if err := yaml.Unmarshal(data, &recipeData); err != nil {
// 			return nil, fmt.Errorf("failed to parse YAML recipe: %w", err)
// 		}
// 	} else {
// 		// Parse Plist recipe
// 		if _, err := plist.Unmarshal(data, &recipeData); err != nil {
// 			return nil, fmt.Errorf("failed to parse Plist recipe: %w", err)
// 		}
// 	}

// 	// Extract Identifier
// 	identifier, _ := recipeData["Identifier"].(string)
// 	if identifier == "" {
// 		return nil, fmt.Errorf("recipe %s missing Identifier", recipePath)
// 	}

// 	if processedIdentifiers[identifier] {
// 		return nil, nil // Avoid duplicate processing
// 	}
// 	processedIdentifiers[identifier] = true

// 	// Extract Parent Recipe if exists
// 	parentIdentifier, _ := recipeData["ParentRecipe"].(string)

// 	var dependencies []RecipeRepoDependency
// 	if parentIdentifier != "" && options.IncludeParents {
// 		logger.Logger(fmt.Sprintf("🧩 Parent recipe found: %s", parentIdentifier), logger.LogInfo)

// 		parentPath, err := findRecipeByIdentifier(parentIdentifier, options.SearchDirs, options.PrefsPath)
// 		if err == nil {
// 			parentDeps, err := analyzeRecipeFile(parentPath, options, processedIdentifiers, depth+1)
// 			if err == nil {
// 				dependencies = append(dependencies, parentDeps...)
// 			}
// 		}
// 	}

// 	// Extract Repo Details
// 	repoOwner, _ := recipeData["RepoOwner"].(string)
// 	repoName, _ := recipeData["RepoName"].(string)
// 	repoURL, _ := recipeData["RepoURL"].(string)

// 	if repoOwner != "" && repoName != "" && repoURL != "" {
// 		dependencies = append(dependencies, RecipeRepoDependency{
// 			RecipeIdentifier: identifier,
// 			RepoOwner:        repoOwner,
// 			RepoName:         repoName,
// 			RepoURL:          repoURL,
// 			IsParent:         false,
// 			Source:           recipePath,
// 		})
// 	}

// 	return dependencies, nil
// }

// // extractRepoFromIdentifier parses a recipe identifier to extract repository information
// // Recipe identifiers typically follow patterns like com.github.username.type.RecipeName
// // or username-recipes.RecipeName, and this function extracts the repository information.
// func extractRepoFromIdentifier(identifier string, recipePath string, isParent bool) (*RecipeRepoDependency, error) {
// 	// Recipe identifier format is typically com.github.[owner].[recipes|other].[name]
// 	// or sometimes [owner]-recipes.[name]
// 	parts := strings.Split(identifier, ".")
// 	if len(parts) < 3 {
// 		return nil, fmt.Errorf("invalid recipe identifier format: %s", identifier)
// 	}

// 	// Check if it's a GitHub hosted recipe
// 	if parts[0] != "com" || parts[1] != "github" {
// 		// It might be using the [owner]-recipes format
// 		if strings.Contains(parts[0], "-recipes") {
// 			owner := strings.TrimSuffix(parts[0], "-recipes")
// 			repoName := fmt.Sprintf("%s-recipes", owner)
// 			repoURL := fmt.Sprintf("https://github.com/autopkg/%s", repoName)

// 			return &RecipeRepoDependency{
// 				RecipeIdentifier: identifier,
// 				RepoOwner:        owner,
// 				RepoName:         repoName,
// 				RepoURL:          repoURL,
// 				IsParent:         isParent,
// 				Source:           recipePath,
// 			}, nil
// 		}

// 		// Not a GitHub recipe - we'll skip it
// 		return nil, nil
// 	}

// 	// Extract the owner from the identifier
// 	owner := parts[2]

// 	// Determine the repo name - typically owner-recipes
// 	repoName := fmt.Sprintf("%s-recipes", owner)

// 	// Build the GitHub URL
// 	repoURL := fmt.Sprintf("https://github.com/autopkg/%s", repoName)

// 	return &RecipeRepoDependency{
// 		RecipeIdentifier: identifier,
// 		RepoOwner:        owner,
// 		RepoName:         repoName,
// 		RepoURL:          repoURL,
// 		IsParent:         isParent,
// 		Source:           recipePath,
// 	}, nil
// }

// // findRecipeByIdentifier searches for a recipe file by its identifier
// // This function looks for a recipe file in the specified search directories and,
// // if not found, uses the autopkg info command to locate it.
// func findRecipeByIdentifier(identifier string, searchDirs []string, prefsPath string) (string, error) {
// 	// If search dirs are provided, look there first
// 	if len(searchDirs) > 0 {
// 		for _, dir := range searchDirs {
// 			// Look for .recipe file
// 			matches, err := filepath.Glob(filepath.Join(dir, "*.recipe"))
// 			if err != nil {
// 				continue
// 			}

// 			for _, match := range matches {
// 				// Check if this file contains the identifier
// 				recipeData, err := os.ReadFile(match)
// 				if err != nil {
// 					continue
// 				}

// 				var recipeDict map[string]interface{}
// 				if _, err := plist.Unmarshal(recipeData, &recipeDict); err != nil {
// 					continue
// 				}

// 				if id, ok := recipeDict["Identifier"].(string); ok && id == identifier {
// 					return match, nil
// 				}
// 			}

// 			// Also look for .recipe.plist files
// 			matches, err = filepath.Glob(filepath.Join(dir, "*.recipe.plist"))
// 			if err != nil {
// 				continue
// 			}

// 			for _, match := range matches {
// 				// Check if this file contains the identifier
// 				recipeData, err := os.ReadFile(match)
// 				if err != nil {
// 					continue
// 				}

// 				var recipeDict map[string]interface{}
// 				if _, err := plist.Unmarshal(recipeData, &recipeDict); err != nil {
// 					continue
// 				}

// 				if id, ok := recipeDict["Identifier"].(string); ok && id == identifier {
// 					return match, nil
// 				}
// 			}
// 		}
// 	}

// 	// If not found in search dirs, use autopkg info to find it
// 	cmd := exec.Command("autopkg", "info", identifier)
// 	if prefsPath != "" {
// 		cmd.Args = append(cmd.Args, "--prefs", prefsPath)
// 	}

// 	output, err := cmd.Output()
// 	if err != nil {
// 		return "", fmt.Errorf("failed to find recipe with identifier %s: %w", identifier, err)
// 	}

// 	// Parse the output to find the recipe path
// 	lines := strings.Split(string(output), "\n")
// 	for _, line := range lines {
// 		if strings.Contains(line, "Recipe file:") {
// 			parts := strings.SplitN(line, ":", 2)
// 			if len(parts) == 2 {
// 				return strings.TrimSpace(parts[1]), nil
// 			}
// 		}
// 	}

// 	return "", fmt.Errorf("could not find recipe file for identifier %s", identifier)
// }

// // verifyRepoExists checks if a GitHub repository exists and is accessible
// // This function uses git ls-remote to verify that the repository URL is valid
// // and accessible.
// func verifyRepoExists(repoURL string) (bool, error) {
// 	// Add .git to the URL to use git ls-remote
// 	if !strings.HasSuffix(repoURL, ".git") {
// 		repoURL += ".git"
// 	}

// 	// Try to list the remote references
// 	cmd := exec.Command("git", "ls-remote", "--exit-code", repoURL, "HEAD")
// 	err := cmd.Run()

// 	if err != nil {
// 		// Exit code 2 from git ls-remote indicates the repository doesn't exist
// 		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 2 {
// 			return false, nil
// 		}
// 		return false, fmt.Errorf("error checking repository: %w", err)
// 	}

// 	return true, nil
// }
