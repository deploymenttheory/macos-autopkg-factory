// commands.go contains a set of wrapped autopkg command line operations with centralized logging
package autopkg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// AuditOptions contains options for AuditRecipe
type AuditOptions struct {
	PrefsPath    string
	SearchDirs   []string
	OverrideDirs []string
	RecipeList   string
	PlistOutput  bool
}

// AuditRecipe audits one or more recipes for potential issues
func AuditRecipe(recipes []string, options *AuditOptions) error {
	if options == nil {
		options = &AuditOptions{}
	}

	logger.Logger("🔍 Auditing recipes...", logger.LogInfo)

	args := []string{"audit"}

	// Add options
	if options.PrefsPath != "" {
		args = append(args, "--prefs", options.PrefsPath)
	}

	for _, dir := range options.SearchDirs {
		args = append(args, "--search-dir", dir)
	}

	for _, dir := range options.OverrideDirs {
		args = append(args, "--override-dir", dir)
	}

	if options.RecipeList != "" {
		args = append(args, "--recipe-list", options.RecipeList)
	}

	if options.PlistOutput {
		args = append(args, "--plist")
	}

	// Add recipes
	args = append(args, recipes...)

	cmd := exec.Command("autopkg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("recipe audit failed: %w", err)
	}

	logger.Logger("✅ Recipe audit completed", logger.LogSuccess)
	return nil
}

// InfoOptions contains options for GetRecipeInfo
type InfoOptions struct {
	PrefsPath    string
	SearchDirs   []string
	OverrideDirs []string
	Quiet        bool
	Pull         bool
}

// GetRecipeInfo retrieves information about a recipe
func GetRecipeInfo(recipe string, options *InfoOptions) error {
	if options == nil {
		options = &InfoOptions{}
	}

	logger.Logger(fmt.Sprintf("ℹ️ Getting info for recipe: %s", recipe), logger.LogInfo)

	args := []string{"info"}

	// Add options
	if options.PrefsPath != "" {
		args = append(args, "--prefs", options.PrefsPath)
	}

	for _, dir := range options.SearchDirs {
		args = append(args, "--search-dir", dir)
	}

	for _, dir := range options.OverrideDirs {
		args = append(args, "--override-dir", dir)
	}

	if options.Quiet {
		args = append(args, "--quiet")
	}

	if options.Pull {
		args = append(args, "--pull")
	}

	// Add recipe
	args = append(args, recipe)

	cmd := exec.Command("autopkg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("get recipe info failed: %w", err)
	}

	return nil
}

// InstallOptions contains options for InstallRecipe
type InstallOptions struct {
	PrefsPath                string
	PreProcessors            []string
	PostProcessors           []string
	CheckOnly                bool
	IgnoreParentVerification bool
	Variables                map[string]string
	RecipeList               string
	PkgOrDmgPath             string
	ReportPlist              string
	Verbose                  bool
	Quiet                    bool
	SearchDirs               []string
	OverrideDirs             []string
}

// InstallRecipe runs one or more install recipes
func InstallRecipe(recipes []string, options *InstallOptions) error {
	if options == nil {
		options = &InstallOptions{}
	}

	logger.Logger(fmt.Sprintf("📦 Installing recipes: %s", strings.Join(recipes, ", ")), logger.LogInfo)

	args := []string{"install"}

	// Add options
	if options.PrefsPath != "" {
		args = append(args, "--prefs", options.PrefsPath)
	}

	for _, processor := range options.PreProcessors {
		args = append(args, "--pre", processor)
	}

	for _, processor := range options.PostProcessors {
		args = append(args, "--post", processor)
	}

	if options.CheckOnly {
		args = append(args, "--check")
	}

	if options.IgnoreParentVerification {
		args = append(args, "--ignore-parent-trust-verification-errors")
	}

	for key, value := range options.Variables {
		args = append(args, "-k", fmt.Sprintf("%s=%s", key, value))
	}

	if options.RecipeList != "" {
		args = append(args, "--recipe-list", options.RecipeList)
	}

	if options.PkgOrDmgPath != "" {
		args = append(args, "--pkg", options.PkgOrDmgPath)
	}

	if options.ReportPlist != "" {
		args = append(args, "--report-plist", options.ReportPlist)
	}

	if options.Verbose {
		args = append(args, "--verbose")
	}

	if options.Quiet {
		args = append(args, "--quiet")
	}

	for _, dir := range options.SearchDirs {
		args = append(args, "--search-dir", dir)
	}

	for _, dir := range options.OverrideDirs {
		args = append(args, "--override-dir", dir)
	}

	// Add recipes
	args = append(args, recipes...)

	cmd := exec.Command("autopkg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("recipe installation failed: %w", err)
	}

	logger.Logger("✅ Recipe installation completed", logger.LogSuccess)
	return nil
}

// ListProcessors lists available core Processors
func ListProcessors(prefsPath string) error {
	logger.Logger("📋 Listing available processors...", logger.LogInfo)

	args := []string{"list-processors"}
	if prefsPath != "" {
		args = append(args, "--prefs", prefsPath)
	}

	cmd := exec.Command("autopkg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("list processors failed: %w", err)
	}

	return nil
}

// ListRecipeOptions contains options for ListRecipes
type ListRecipeOptions struct {
	PrefsPath       string
	WithIdentifiers bool
	WithPaths       bool
	PlistOutput     bool
	ShowAll         bool
	SearchDirs      []string
	OverrideDirs    []string
}

// ListRecipes lists recipes available locally
func ListRecipes(options *ListRecipeOptions) error {
	if options == nil {
		options = &ListRecipeOptions{}
	}

	logger.Logger("📋 Listing available recipes...", logger.LogInfo)

	args := []string{"list-recipes"}

	// Add options
	if options.PrefsPath != "" {
		args = append(args, "--prefs", options.PrefsPath)
	}

	if options.WithIdentifiers {
		args = append(args, "--with-identifiers")
	}

	if options.WithPaths {
		args = append(args, "--with-paths")
	}

	if options.PlistOutput {
		args = append(args, "--plist")
	}

	if options.ShowAll {
		args = append(args, "--show-all")
	}

	for _, dir := range options.SearchDirs {
		args = append(args, "--search-dir", dir)
	}

	for _, dir := range options.OverrideDirs {
		args = append(args, "--override-dir", dir)
	}

	cmd := exec.Command("autopkg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("list recipes failed: %w", err)
	}

	return nil
}

// ListRepos lists installed recipe repositories
func ListRepos(prefsPath string) error {
	logger.Logger("📋 Listing installed recipe repositories...", logger.LogInfo)

	args := []string{"repo-list"}
	if prefsPath != "" {
		args = append(args, "--prefs", prefsPath)
	}

	cmd := exec.Command("autopkg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("list repos failed: %w", err)
	}

	return nil
}

// MakeOverrideOptions contains options for MakeOverride
type MakeOverrideOptions struct {
	PrefsPath         string
	SearchDirs        []string
	OverrideDirs      []string
	Name              string
	Force             bool
	Pull              bool
	IgnoreDeprecation bool
	Format            string // "plist" or "yaml"
}

// MakeOverride creates a recipe override
func MakeOverride(recipe string, options *MakeOverrideOptions) error {
	if options == nil {
		options = &MakeOverrideOptions{}
	}

	logger.Logger(fmt.Sprintf("🔧 Creating override for recipe: %s", recipe), logger.LogInfo)

	args := []string{"make-override"}

	// Add options
	if options.PrefsPath != "" {
		args = append(args, "--prefs", options.PrefsPath)
	}

	for _, dir := range options.SearchDirs {
		args = append(args, "--search-dir", dir)
	}

	for _, dir := range options.OverrideDirs {
		args = append(args, "--override-dir", dir)
	}

	if options.Name != "" {
		args = append(args, "--name", options.Name)
	}

	if options.Force {
		args = append(args, "--force")
	}

	if options.Pull {
		args = append(args, "--pull")
	}

	if options.IgnoreDeprecation {
		args = append(args, "--ignore-deprecation")
	}

	if options.Format != "" {
		args = append(args, "--format", options.Format)
	}

	// Add recipe
	args = append(args, recipe)

	cmd := exec.Command("autopkg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("make override failed: %w", err)
	}

	logger.Logger(fmt.Sprintf("✅ Created override for recipe: %s", recipe), logger.LogSuccess)
	return nil
}

// NewRecipeOptions contains options for NewRecipeFile
type NewRecipeOptions struct {
	PrefsPath        string
	Identifier       string
	ParentIdentifier string
	Format           string // "plist" or "yaml"
}

// NewRecipeFile creates a new template recipe
func NewRecipeFile(recipePath string, options *NewRecipeOptions) error {
	if options == nil {
		options = &NewRecipeOptions{}
	}

	logger.Logger(fmt.Sprintf("🔧 Creating new recipe template: %s", recipePath), logger.LogInfo)

	args := []string{"new-recipe"}

	// Add options
	if options.PrefsPath != "" {
		args = append(args, "--prefs", options.PrefsPath)
	}

	if options.Identifier != "" {
		args = append(args, "--identifier", options.Identifier)
	}

	if options.ParentIdentifier != "" {
		args = append(args, "--parent-identifier", options.ParentIdentifier)
	}

	if options.Format != "" {
		args = append(args, "--format", options.Format)
	}

	// Add recipe path
	args = append(args, recipePath)

	cmd := exec.Command("autopkg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("new recipe creation failed: %w", err)
	}

	logger.Logger(fmt.Sprintf("✅ Created new recipe template: %s", recipePath), logger.LogSuccess)
	return nil
}

// ProcessorInfoOptions contains options for GetProcessorInfo
type ProcessorInfoOptions struct {
	PrefsPath    string
	Recipe       string
	SearchDirs   []string
	OverrideDirs []string
}

// GetProcessorInfo gets information about a specific processor
func GetProcessorInfo(processor string, options *ProcessorInfoOptions) error {
	if options == nil {
		options = &ProcessorInfoOptions{}
	}

	logger.Logger(fmt.Sprintf("ℹ️ Getting info for processor: %s", processor), logger.LogInfo)

	args := []string{"processor-info"}

	// Add options
	if options.PrefsPath != "" {
		args = append(args, "--prefs", options.PrefsPath)
	}

	if options.Recipe != "" {
		args = append(args, "--recipe", options.Recipe)
	}

	for _, dir := range options.SearchDirs {
		args = append(args, "--search-dir", dir)
	}

	for _, dir := range options.OverrideDirs {
		args = append(args, "--override-dir", dir)
	}

	// Add processor name
	args = append(args, processor)

	cmd := exec.Command("autopkg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("get processor info failed: %w", err)
	}

	return nil
}

// AddRepo adds one or more recipe repositories from URLs
func AddRepo(repoURLs []string, prefsPath string) error {
	logger.Logger(fmt.Sprintf("📦 Adding recipe repositories: %s", strings.Join(repoURLs, ", ")), logger.LogInfo)

	for _, repoURL := range repoURLs {
		args := []string{"repo-add", repoURL}
		if prefsPath != "" {
			args = append(args, "--prefs", prefsPath)
		}

		cmd := exec.Command("autopkg", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			logger.Logger(fmt.Sprintf("⚠️ Failed to add repo %s: %v", repoURL, err), logger.LogWarning)
			continue
		}

		logger.Logger(fmt.Sprintf("✅ Added repository: %s", repoURL), logger.LogSuccess)
	}

	return nil
}

// DeleteRepo deletes a recipe repository
func DeleteRepo(repoName string, prefsPath string) error {
	if repoName == "" {
		return fmt.Errorf("repository name is required")
	}

	logger.Logger(fmt.Sprintf("🗑️ Deleting recipe repository: %s", repoName), logger.LogInfo)

	args := []string{"repo-delete", repoName}
	if prefsPath != "" {
		args = append(args, "--prefs", prefsPath)
	}

	cmd := exec.Command("autopkg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("delete repo failed: %w", err)
	}

	logger.Logger(fmt.Sprintf("✅ Deleted repository: %s", repoName), logger.LogSuccess)
	return nil
}

// UpdateRepo updates one or more recipe repositories
func UpdateRepo(repos []string, prefsPath string) error {
	repoDesc := "all repositories"
	if len(repos) > 0 {
		repoDesc = strings.Join(repos, ", ")
	}

	logger.Logger(fmt.Sprintf("🔄 Updating %s", repoDesc), logger.LogInfo)

	args := []string{"repo-update"}
	args = append(args, repos...)
	if prefsPath != "" {
		args = append(args, "--prefs", prefsPath)
	}

	cmd := exec.Command("autopkg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("update repo failed: %w", err)
	}

	logger.Logger(fmt.Sprintf("✅ Updated %s", repoDesc), logger.LogSuccess)
	return nil
}

// SearchOptions contains options for SearchRecipes
type SearchOptions struct {
	PrefsPath string
	User      string
	PathOnly  bool
	UseToken  bool
}

// SearchRecipes searches for recipes on GitHub
func SearchRecipes(term string, options *SearchOptions) error {
	if options == nil {
		options = &SearchOptions{}
	}

	if term == "" {
		return fmt.Errorf("search term is required")
	}

	logger.Logger(fmt.Sprintf("🔍 Searching for recipes matching: %s", term), logger.LogInfo)

	args := []string{"search"}

	// Add options
	if options.PrefsPath != "" {
		args = append(args, "--prefs", options.PrefsPath)
	}

	if options.User != "" {
		args = append(args, "--user", options.User)
	}

	if options.PathOnly {
		args = append(args, "--path-only")
	}

	if options.UseToken {
		args = append(args, "--use-token")
	}

	// Add search term
	args = append(args, term)

	cmd := exec.Command("autopkg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("search recipes failed: %w", err)
	}

	return nil
}

// GetVersion prints the current version of autopkg
func GetVersion() (string, error) {
	logger.Logger("ℹ️ Getting AutoPkg version", logger.LogInfo)

	cmd := exec.Command("autopkg", "version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("get version failed: %w", err)
	}

	version := strings.TrimSpace(string(output))
	logger.Logger(fmt.Sprintf("📦 AutoPkg version: %s", version), logger.LogInfo)
	return version, nil
}

// RunRecipes runs one or more recipes with extended options
func RunRecipes(recipes []string, options *RunOptions) error {
	if options == nil {
		options = &RunOptions{}
	}

	if len(recipes) == 0 {
		return fmt.Errorf("at least one recipe name is required")
	}

	logger.Logger(fmt.Sprintf("🚀 Running recipes: %s", strings.Join(recipes, ", ")), logger.LogInfo)

	// Build base arguments
	args := []string{"run"}

	// Add options
	if options.PrefsPath != "" {
		args = append(args, "--prefs", options.PrefsPath)
	}

	for _, processor := range options.PreProcessors {
		args = append(args, "--pre", processor)
	}

	for _, processor := range options.PostProcessors {
		args = append(args, "--post", processor)
	}

	if options.CheckOnly {
		args = append(args, "--check")
	}

	if options.IgnoreParentVerification {
		args = append(args, "--ignore-parent-trust-verification-errors")
	}

	for key, value := range options.Variables {
		args = append(args, "-k", fmt.Sprintf("%s=%s", key, value))
	}

	if options.RecipeList != "" {
		args = append(args, "--recipe-list", options.RecipeList)
	}

	if options.PkgOrDmgPath != "" {
		args = append(args, "--pkg", options.PkgOrDmgPath)
	}

	if options.ReportPlist != "" {
		args = append(args, "--report-plist", options.ReportPlist)
	}

	if options.Verbose {
		args = append(args, "--verbose")
	}

	if options.Quiet {
		args = append(args, "--quiet")
	}

	for _, dir := range options.SearchDirs {
		args = append(args, "--search-dir", dir)
	}

	for _, dir := range options.OverrideDirs {
		args = append(args, "--override-dir", dir)
	}

	if options.UpdateTrust {
		args = append(args, "--update-trust-info")
	}

	// Add recipes
	args = append(args, recipes...)

	// Execute the command
	cmd := exec.Command("autopkg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("recipe run failed: %w", err)
	}

	logger.Logger("✅ Recipe run completed", logger.LogSuccess)
	return nil
}

// RunOptions holds extended options for running a recipe
type RunOptions struct {
	PrefsPath                string
	PreProcessors            []string
	PostProcessors           []string
	CheckOnly                bool
	IgnoreParentVerification bool
	Variables                map[string]string
	RecipeList               string
	PkgOrDmgPath             string
	ReportPlist              string
	Verbose                  bool
	Quiet                    bool
	SearchDirs               []string
	OverrideDirs             []string
	UpdateTrust              bool
}

// BatchRunRecipes runs multiple recipes with the same options
func BatchRunRecipes(recipes []string, options *RunOptions) error {
	if len(recipes) == 0 {
		return fmt.Errorf("at least one recipe name is required")
	}

	logger.Logger(fmt.Sprintf("🚀 Batch running %d recipes", len(recipes)), logger.LogInfo)

	// Run the recipes with the provided options
	if err := RunRecipes(recipes, options); err != nil {
		return fmt.Errorf("batch run failed: %w", err)
	}

	logger.Logger("✅ Batch recipe run completed", logger.LogSuccess)
	return nil
}

// CreateLocalRepository creates a new local repository
func CreateLocalRepository(repoName, repoPath string) error {
	if repoName == "" || repoPath == "" {
		return fmt.Errorf("repository name and path are required")
	}

	// Ensure directory exists
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return fmt.Errorf("failed to create repository directory: %w", err)
	}

	// Initialize git repository
	cmd := exec.Command("git", "-C", repoPath, "init")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Create standard directories
	for _, dir := range []string{"Recipes", "RecipeRepos"} {
		path := filepath.Join(repoPath, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create README.md
	readmePath := filepath.Join(repoPath, "README.md")
	readmeContent := fmt.Sprintf("# %s\n\nAutomatic package repository for MacOS software. Created with autopkg.\n", repoName)
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}

	logger.Logger(fmt.Sprintf("✅ Created local repository %s at %s", repoName, repoPath), logger.LogSuccess)
	return nil
}

// RunRecipeWithOutput runs a recipe and captures the output
func RunRecipeWithOutput(recipe string, options *RunOptions) (string, error) {
	if options == nil {
		options = &RunOptions{}
	}

	logger.Logger(fmt.Sprintf("🚀 Running recipe and capturing output: %s", recipe), logger.LogInfo)

	// Build base arguments
	args := []string{"run"}

	// Add options
	if options.PrefsPath != "" {
		args = append(args, "--prefs", options.PrefsPath)
	}

	for _, processor := range options.PreProcessors {
		args = append(args, "--pre", processor)
	}

	for _, processor := range options.PostProcessors {
		args = append(args, "--post", processor)
	}

	if options.CheckOnly {
		args = append(args, "--check")
	}

	if options.IgnoreParentVerification {
		args = append(args, "--ignore-parent-trust-verification-errors")
	}

	for key, value := range options.Variables {
		args = append(args, "-k", fmt.Sprintf("%s=%s", key, value))
	}

	if options.RecipeList != "" {
		args = append(args, "--recipe-list", options.RecipeList)
	}

	if options.PkgOrDmgPath != "" {
		args = append(args, "--pkg", options.PkgOrDmgPath)
	}

	if options.ReportPlist != "" {
		args = append(args, "--report-plist", options.ReportPlist)
	}

	if options.Verbose {
		args = append(args, "--verbose")
	}

	if options.Quiet {
		args = append(args, "--quiet")
	}

	for _, dir := range options.SearchDirs {
		args = append(args, "--search-dir", dir)
	}

	for _, dir := range options.OverrideDirs {
		args = append(args, "--override-dir", dir)
	}

	if options.UpdateTrust {
		args = append(args, "--update-trust-info")
	}

	// Add recipe
	args = append(args, recipe)

	cmd := exec.Command("autopkg", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return string(output), fmt.Errorf("run recipe failed: %w", err)
	}

	return string(output), nil
}

// VerifyTrustInfoOptions contains options for verifying trust info
type VerifyTrustInfoOptions struct {
	PrefsPath    string
	RecipeList   string
	Verbose      int // 0 = normal, 1 = -v, 2 = -vv, 3 = -vvv
	SearchDirs   []string
	OverrideDirs []string
}

// UpdateTrustInfoOptions contains options for updating trust info
type UpdateTrustInfoOptions struct {
	PrefsPath    string
	SearchDirs   []string
	OverrideDirs []string
}

// VerifyTrustInfoForRecipes verifies parent recipe trust info for one or more recipe overrides
func VerifyTrustInfoForRecipes(recipes []string, options *VerifyTrustInfoOptions) (bool, []string, error) {
	if options == nil {
		options = &VerifyTrustInfoOptions{}
	}

	if len(recipes) == 0 && options.RecipeList == "" {
		return false, nil, fmt.Errorf("at least one recipe name or a recipe list file is required")
	}

	logger.Logger("🔒 Verifying trust info for recipes", logger.LogInfo)

	args := []string{"verify-trust-info"}

	// Add options
	if options.PrefsPath != "" {
		args = append(args, "--prefs", options.PrefsPath)
	}

	if options.RecipeList != "" {
		args = append(args, "--recipe-list", options.RecipeList)
	}

	// Add verbosity flags
	switch options.Verbose {
	case 1:
		args = append(args, "-v")
	case 2:
		args = append(args, "-v", "-v")
	case 3:
		args = append(args, "-v", "-v", "-v")
	}

	for _, dir := range options.SearchDirs {
		args = append(args, "--search-dir", dir)
	}

	for _, dir := range options.OverrideDirs {
		args = append(args, "--override-dir", dir)
	}

	// Add recipes
	args = append(args, recipes...)

	cmd := exec.Command("autopkg", args...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Collect failed and successful recipes
	var failedRecipes []string

	// Process output to find failed recipes
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "WARNING:") && strings.Contains(line, "trust verification failed") {
			parts := strings.Split(line, "for ")
			if len(parts) > 1 {
				recipePart := strings.TrimSpace(parts[1])
				recipePart = strings.TrimSuffix(recipePart, ":")
				failedRecipes = append(failedRecipes, recipePart)
			}
		}
	}

	if err != nil {
		logger.Logger("❌ Trust verification failed for one or more recipes", logger.LogError)
		if options.Verbose > 0 {
			logger.Logger(outputStr, logger.LogDebug)
		}
		return false, failedRecipes, fmt.Errorf("verify trust info failed: %w", err)
	}

	logger.Logger("✅ Trust verification passed for all recipes", logger.LogSuccess)
	return true, nil, nil
}

// UpdateTrustInfoForRecipes updates or adds parent recipe trust info for one or more recipe overrides
func UpdateTrustInfoForRecipes(recipes []string, options *UpdateTrustInfoOptions) error {
	if options == nil {
		options = &UpdateTrustInfoOptions{}
	}

	if len(recipes) == 0 {
		return fmt.Errorf("at least one recipe name is required")
	}

	logger.Logger("🔒 Updating trust info for recipes", logger.LogInfo)

	args := []string{"update-trust-info"}

	// Add options
	if options.PrefsPath != "" {
		args = append(args, "--prefs", options.PrefsPath)
	}

	for _, dir := range options.SearchDirs {
		args = append(args, "--search-dir", dir)
	}

	for _, dir := range options.OverrideDirs {
		args = append(args, "--override-dir", dir)
	}

	// Add recipes
	args = append(args, recipes...)

	cmd := exec.Command("autopkg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("update trust info failed: %w", err)
	}

	logger.Logger("✅ Trust info updated for all recipes", logger.LogSuccess)
	return nil
}
