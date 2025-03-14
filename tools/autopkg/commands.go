// commands.go contains a set of wrapped autopkg command line operations with centralized logging
package autopkg

import (
	"bytes"
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
// does not support verbosity levels.
func AuditRecipe(recipes []string, options *AuditOptions) (string, error) {
	if options == nil {
		options = &AuditOptions{}
	}

	args := []string{"audit"}

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

	args = append(args, recipes...)

	logger.Logger("🔍 Auditing recipes...", logger.LogInfo)

	logger.Logger(fmt.Sprintf("🖥️  Running command: autopkg %s", strings.Join(args, " ")), logger.LogDebug)

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("recipe audit failed: %w", err)
	}

	logger.Logger("✅ Recipe audit completed", logger.LogSuccess)
	return outputBuffer.String(), nil
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
func GetRecipeInfo(recipe string, options *InfoOptions) (string, error) {
	if options == nil {
		options = &InfoOptions{}
	}

	args := []string{"info"}

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

	args = append(args, recipe)

	logger.Logger(fmt.Sprintf("ℹ️ Getting info for recipe: %s", recipe), logger.LogInfo)

	logger.Logger(fmt.Sprintf("🖥️  Running command: autopkg %s", strings.Join(args, " ")), logger.LogDebug)

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("get recipe info failed: %w", err)
	}

	return outputBuffer.String(), nil
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
func InstallRecipe(recipes []string, options *InstallOptions) (string, error) {
	if options == nil {
		options = &InstallOptions{}
	}

	args := []string{"install"}

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
		args = append(args, "-key", fmt.Sprintf("%s=%s", key, value))
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

	args = append(args, recipes...)

	logger.Logger(fmt.Sprintf("📦 Installing recipes: %s", strings.Join(recipes, ", ")), logger.LogInfo)

	logger.Logger(fmt.Sprintf("🖥️  Running command: autopkg %s", strings.Join(args, " ")), logger.LogDebug)

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("install recipe failed: %w", err)
	}

	logger.Logger("✅ Recipe installation completed", logger.LogSuccess)
	return outputBuffer.String(), nil
}

// ListProcessors lists available core Processors
func ListProcessors(prefsPath string) (string, error) {

	args := []string{"list-processors"}
	if prefsPath != "" {
		args = append(args, "--prefs", prefsPath)
	}

	logger.Logger("📋 Listing available processors...", logger.LogInfo)

	logger.Logger(fmt.Sprintf("🖥️  Running command: autopkg %s", strings.Join(args, " ")), logger.LogDebug)

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("list processors failed: %w", err)
	}

	return outputBuffer.String(), nil
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
func ListRecipes(options *ListRecipeOptions) (string, error) {
	if options == nil {
		options = &ListRecipeOptions{}
	}

	args := []string{"list-recipes"}

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

	logger.Logger("📋 Listing available recipes...", logger.LogInfo)

	logger.Logger(fmt.Sprintf("🖥️  Running command: autopkg %s", strings.Join(args, " ")), logger.LogDebug)

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("list recipes failed: %w", err)
	}

	return outputBuffer.String(), nil
}

// ListRepos lists installed recipe repositories
func ListRepos(prefsPath string) (string, error) {

	args := []string{"repo-list"}
	if prefsPath != "" {
		args = append(args, "--prefs", prefsPath)
	}

	logger.Logger("📋 Listing installed recipe repositories...", logger.LogInfo)

	logger.Logger(fmt.Sprintf("🖥️  Running command: autopkg %s", strings.Join(args, " ")), logger.LogDebug)

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("list repo's failed: %w", err)
	}

	return outputBuffer.String(), nil
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
// Options:
//
//	--prefs=FILE_PATH
//	--search-dir=DIRECTORY
//	                      Directory to search for recipes. Can be specified
//	                      multiple times.
//	--override-dir=DIRECTORY
//	                      Directory to search for recipe overrides. Can be
//	                      specified multiple times.
//	--name=FILENAME
//	                      Name for override file.
//	--force               Force overwrite an override file.
//	--pull                Pull the parent repos if they can't be found in the
//	                      search path. Implies agreement to search GitHub.
//	--ignore-deprecation  Make an override even if the specified recipe or one
//	                      of its parents is deprecated.
//	--format=FORMAT       The format of the recipe override to be created. Valid
//	                      options include: 'plist' (default) or 'yaml'
func MakeOverride(recipe string, options *MakeOverrideOptions) (string, error) {
	if options == nil {
		options = &MakeOverrideOptions{}
	}

	args := []string{"make-override"}

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

	args = append(args, recipe)

	logger.Logger(fmt.Sprintf("🔧 Creating override for recipe: %s", recipe), logger.LogInfo)

	logger.Logger(fmt.Sprintf("🖥️  Running command: autopkg %s", strings.Join(args, " ")), logger.LogDebug)

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("make recipe override failed: %w", err)
	}

	logger.Logger(fmt.Sprintf("✅ Created override for recipe: %s", recipe), logger.LogSuccess)
	return outputBuffer.String(), nil
}

// NewRecipeOptions contains options for NewRecipeFile
type NewRecipeOptions struct {
	PrefsPath        string
	Identifier       string
	ParentIdentifier string
	Format           string // "plist" or "yaml"
}

// NewRecipeFile creates a new template recipe
func NewRecipeFile(recipePath string, options *NewRecipeOptions) (string, error) {
	if options == nil {
		options = &NewRecipeOptions{}
	}

	args := []string{"new-recipe"}

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

	args = append(args, recipePath)

	logger.Logger(fmt.Sprintf("🔧 Creating new recipe template: %s", recipePath), logger.LogInfo)

	logger.Logger(fmt.Sprintf("🖥️  Running command: autopkg %s", strings.Join(args, " ")), logger.LogDebug)

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("new recipe failed: %w", err)
	}

	logger.Logger(fmt.Sprintf("✅ Created new recipe template: %s", recipePath), logger.LogSuccess)
	return outputBuffer.String(), nil
}

// ProcessorInfoOptions contains options for GetProcessorInfo
type ProcessorInfoOptions struct {
	PrefsPath    string
	Recipe       string
	SearchDirs   []string
	OverrideDirs []string
}

// GetProcessorInfo gets information about a specific processor
func GetProcessorInfo(processor string, options *ProcessorInfoOptions) (string, error) {
	if options == nil {
		options = &ProcessorInfoOptions{}
	}

	args := []string{"processor-info"}

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

	args = append(args, processor)

	logger.Logger(fmt.Sprintf("ℹ️ Getting info for processor: %s", processor), logger.LogInfo)

	logger.Logger(fmt.Sprintf("🖥️  Running command: autopkg %s", strings.Join(args, " ")), logger.LogDebug)

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("get processor info failed: %w", err)
	}

	return outputBuffer.String(), nil
}

// AddRepo adds one or more recipe repositories from URLs
func AddRepo(repoURLs []string, prefsPath string) (string, error) {
	logger.Logger(fmt.Sprintf("📦 Adding recipe repositories: %s", strings.Join(repoURLs, ", ")), logger.LogInfo)

	var fullOutput bytes.Buffer

	for _, repoURL := range repoURLs {
		args := []string{"repo-add", repoURL}
		if prefsPath != "" {
			args = append(args, "--prefs", prefsPath)
		}

		logger.Logger(fmt.Sprintf("🖥️  Running command: autopkg %s", strings.Join(args, " ")), logger.LogDebug)

		cmd := exec.Command("autopkg", args...)

		var outputBuffer bytes.Buffer
		cmd.Stdout = &outputBuffer
		cmd.Stderr = &outputBuffer

		if err := cmd.Run(); err != nil {
			msg := fmt.Sprintf("⚠️ Failed to add repo %s: %v", repoURL, err)
			logger.Logger(msg, logger.LogWarning)
			fullOutput.WriteString(msg + "\n" + outputBuffer.String() + "\n")
			continue
		}

		msg := fmt.Sprintf("✅ Added repository: %s", repoURL)
		logger.Logger(msg, logger.LogSuccess)
		fullOutput.WriteString(msg + "\n" + outputBuffer.String() + "\n")
	}

	return fullOutput.String(), nil
}

// DeleteRepo deletes a recipe repository
func DeleteRepo(repoName string, prefsPath string) (string, error) {
	if repoName == "" {
		return "", fmt.Errorf("repository name is required")
	}

	args := []string{"repo-delete", repoName}
	if prefsPath != "" {
		args = append(args, "--prefs", prefsPath)
	}

	logger.Logger(fmt.Sprintf("🗑️ Deleting recipe repository: %s", repoName), logger.LogInfo)

	logger.Logger(fmt.Sprintf("🖥️  Running command: autopkg %s", strings.Join(args, " ")), logger.LogDebug)

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("delete repo failed: %w", err)
	}

	logger.Logger(fmt.Sprintf("✅ Deleted repository: %s", repoName), logger.LogSuccess)
	return outputBuffer.String(), nil
}

// UpdateRepo updates one or more recipe repositories
func UpdateRepo(repos []string, prefsPath string) (string, error) {
	repoDesc := "all repositories"
	if len(repos) > 0 {
		repoDesc = strings.Join(repos, ", ")
	}

	args := []string{"repo-update"}
	args = append(args, repos...)
	if prefsPath != "" {
		args = append(args, "--prefs", prefsPath)
	}

	logger.Logger(fmt.Sprintf("🔄 Updating %s", repoDesc), logger.LogInfo)

	logger.Logger(fmt.Sprintf("🖥️  Running command: autopkg %s", strings.Join(args, " ")), logger.LogDebug)

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("update repo failed: %w", err)
	}

	logger.Logger(fmt.Sprintf("✅ Updated %s", repoDesc), logger.LogSuccess)
	return outputBuffer.String(), nil
}

// SearchOptions contains options for SearchRecipes
type SearchOptions struct {
	PrefsPath string
	User      string
	PathOnly  bool
	UseToken  bool
}

// SearchRecipes searches for recipes on GitHub
func SearchRecipes(term string, options *SearchOptions) (string, error) {
	if options == nil {
		options = &SearchOptions{}
	}

	if term == "" {
		return "", fmt.Errorf("search term is required")
	}

	args := []string{"search"}

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

	args = append(args, term)

	logger.Logger(fmt.Sprintf("🔍 Searching for recipes matching: %s", term), logger.LogInfo)

	logger.Logger(fmt.Sprintf("🖥️  Running command: autopkg %s", strings.Join(args, " ")), logger.LogDebug)

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("recipe search failed: %w", err)
	}

	return outputBuffer.String(), nil
}

// GetVersion prints the current version of autopkg
func GetVersion() (string, error) {
	logger.Logger("ℹ️ Getting AutoPkg version", logger.LogInfo)

	cmd := exec.Command("autopkg", "version")

	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("get autopkg failed: %w", err)
	}

	version := strings.TrimSpace(outputBuffer.String())
	logger.Logger(fmt.Sprintf("📦 AutoPkg version: %s", version), logger.LogInfo)
	return version, nil
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
	VerboseLevel             int
}

// RunRecipe runs a recipe and captures the output
//
//	--prefs=FILE_PATH
//	--pre=PREPROCESSOR, --preprocessor=PREPROCESSOR
//	                      Name of a processor to run before each recipe. Can be
//	                      repeated to run multiple preprocessors.
//	--post=POSTPROCESSOR, --postprocessor=POSTPROCESSOR
//	                      Name of a processor to run after each recipe. Can be
//	                      repeated to run multiple postprocessors.
//	--check               Only check for new/changed downloads.
//	--ignore-parent-trust-verification-errors
//	                      Run recipes even if they fail parent trust
//	                      verification.
//	--key=KEY=VALUE
//	                      Provide key/value pairs for recipe input. Caution:
//	                      values specified here will be applied to all recipes.
//	--recipe-list=TEXT_FILE
//	                      Path to a text file with a list of recipes to run.
//	--pkg=PKG_OR_DMG
//	                      Path to a pkg or dmg to provide to a recipe.
//	                      Downloading will be skipped.
//	--report-plist=OUTPUT_PATH
//	                      File path to save run report plist.
//	--verbose         Verbose output.
//	--quiet           Don't offer to search Github if a recipe can't be
//	                      found.
//	--search-dir=DIRECTORY
//	                      Directory to search for recipes. Can be specified
//	                      multiple times.
//	--override-dir=DIRECTORY
//	                      Directory to search for recipe overrides. Can be
//	                      specified multiple times.
func RunRecipe(recipe string, options *RunOptions) (string, error) {
	if options == nil {
		options = &RunOptions{}
	}

	args := []string{"run"}

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
		args = append(args, "-key", fmt.Sprintf("%s=%s", key, value))
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

	if options.VerboseLevel > 0 {
		args = append(args, fmt.Sprintf("-%s", strings.Repeat("v", options.VerboseLevel)))
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

	if options.RecipeList == "" && recipe != "" {
		args = append(args, recipe)
	}

	logger.Logger(fmt.Sprintf("🖥️ Running command: autopkg %s", strings.Join(args, " ")), logger.LogDebug)

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

	if err := cmd.Run(); err != nil {
		outputStr := outputBuffer.String()
		logger.Logger(fmt.Sprintf("❌ Command output: %s", outputStr), logger.LogError)
		return outputStr, fmt.Errorf("recipe run failed: %w", err)
	}

	return outputBuffer.String(), nil
}

// CreateLocalRepository creates a new local repository
func CreateLocalRepository(repoName, repoPath string) (string, error) {
	if repoName == "" || repoPath == "" {
		return "", fmt.Errorf("repository name and path are required")
	}

	var outputBuffer bytes.Buffer

	// Ensure directory exists
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return outputBuffer.String(), fmt.Errorf("failed to create repository directory: %w", err)
	}

	// Initialize git repository
	cmd := exec.Command("git", "-C", repoPath, "init")
	gitOutput, err := cmd.CombinedOutput()
	outputBuffer.Write(gitOutput)

	if err != nil {
		return outputBuffer.String(), fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Create standard directories
	for _, dir := range []string{"Recipes", "RecipeRepos"} {
		path := filepath.Join(repoPath, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return outputBuffer.String(), fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	readmePath := filepath.Join(repoPath, "README.md")
	readmeContent := fmt.Sprintf("# %s\n\nAutomatic package repository for MacOS software. Created with autopkg.\n", repoName)
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return outputBuffer.String(), fmt.Errorf("failed to create README.md: %w", err)
	}

	msg := fmt.Sprintf("✅ Created local repository %s at %s", repoName, repoPath)
	logger.Logger(msg, logger.LogSuccess)
	outputBuffer.WriteString(msg + "\n")

	return outputBuffer.String(), nil
}

// VerifyTrustInfoOptions contains options for verifying trust info
type VerifyTrustInfoOptions struct {
	PrefsPath    string
	RecipeList   string
	VerboseLevel int // 0 = normal, 1 = -v, 2 = -vv, 3 = -vvv
	SearchDirs   []string
	OverrideDirs []string
}

// UpdateTrustInfoOptions contains options for updating trust info
type UpdateTrustInfoOptions struct {
	PrefsPath    string
	SearchDirs   []string
	OverrideDirs []string
	VerboseLevel int // 0 = normal, 1 = -v, 2 = -vv, 3 = -vvv
}

// VerifyTrustInfoForRecipes verifies parent recipe trust info for one or more recipe overrides
func VerifyTrustInfoForRecipes(recipes []string, options *VerifyTrustInfoOptions) (bool, []string, string, error) {
	if options == nil {
		options = &VerifyTrustInfoOptions{}
	}

	if len(recipes) == 0 && options.RecipeList == "" {
		return false, nil, "", fmt.Errorf("at least one recipe name or a recipe list file is required")
	}

	args := []string{"verify-trust-info"}

	if options.PrefsPath != "" {
		args = append(args, "--prefs", options.PrefsPath)
	}
	if options.RecipeList != "" {
		args = append(args, "--recipe-list", options.RecipeList)
	}

	if options.VerboseLevel > 0 {
		args = append(args, fmt.Sprintf("-%s", strings.Repeat("v", options.VerboseLevel)))
	}

	for _, dir := range options.SearchDirs {
		args = append(args, "--search-dir", dir)
	}
	for _, dir := range options.OverrideDirs {
		args = append(args, "--override-dir", dir)
	}

	args = append(args, recipes...)

	logger.Logger("🔒 Verifying trust info for recipes", logger.LogInfo)

	logger.Logger(fmt.Sprintf("🖥️  Running command: autopkg %s", strings.Join(args, " ")), logger.LogDebug)

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

	execErr := cmd.Run()
	outputStr := outputBuffer.String()

	logger.Logger(fmt.Sprintf("DEBUG: verify-trust-info output:\n%s", outputStr), logger.LogDebug)

	var failedRecipes []string
	failureReasons := make(map[string][]string)
	var currentRecipe string

	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect trust failures
		if strings.HasSuffix(line, ": FAILED") {
			currentRecipe = strings.Split(line, ":")[0]
			failedRecipes = append(failedRecipes, currentRecipe)
			failureReasons[currentRecipe] = []string{"Unknown failure reason. try including -vvv in the options"}
		} else if strings.HasPrefix(line, "No trust information present.") && currentRecipe != "" {
			// Capture missing trust info
			failureReasons[currentRecipe] = []string{"No trust information present."}
		} else if strings.HasPrefix(line, "Audit the recipe") && currentRecipe != "" {
			// Capture suggested remediation
			failureReasons[currentRecipe] = append(failureReasons[currentRecipe], line)
		} else if strings.Contains(line, "contents differ from expected") && currentRecipe != "" {
			// Handle processor mismatches
			failureReasons[currentRecipe] = append(failureReasons[currentRecipe], line)
		} else if strings.Contains(line, "processor path not found") {
			// Handle missing processor warnings
			logger.Logger(fmt.Sprintf("⚠️  %s", line), logger.LogWarning)
		}
	}

	if execErr != nil || len(failedRecipes) > 0 {
		logger.Logger(fmt.Sprintf("❌ Trust verification failed for %d recipes", len(failedRecipes)), logger.LogError)
		for _, recipe := range failedRecipes {
			logger.Logger(fmt.Sprintf("  - %s:", recipe), logger.LogWarning)
			for _, reason := range failureReasons[recipe] {
				logger.Logger(fmt.Sprintf("    • %s", reason), logger.LogWarning)
			}
		}

		if options.VerboseLevel > 0 {
			logger.Logger(outputStr, logger.LogDebug)
		}
		return false, failedRecipes, outputStr, fmt.Errorf("verify trust info failed for %d recipes", len(failedRecipes))
	}

	logger.Logger("✅ Trust verification passed for all recipes", logger.LogSuccess)
	return true, nil, outputStr, nil
}

// UpdateTrustInfoForRecipes updates or adds parent recipe trust info for one or more recipe overrides
func UpdateTrustInfoForRecipes(recipes []string, options *UpdateTrustInfoOptions) (string, error) {
	if options == nil {
		options = &UpdateTrustInfoOptions{}
	}

	if len(recipes) == 0 {
		return "", fmt.Errorf("at least one recipe name is required")
	}

	args := []string{"update-trust-info"}

	if options.PrefsPath != "" {
		args = append(args, "--prefs", options.PrefsPath)
	}

	for _, dir := range options.SearchDirs {
		args = append(args, "--search-dir", dir)
	}

	for _, dir := range options.OverrideDirs {
		args = append(args, "--override-dir", dir)
	}

	args = append(args, recipes...)

	logger.Logger("🔒 Updating trust info for recipes", logger.LogInfo)

	logger.Logger(fmt.Sprintf("🖥️  Running command: autopkg %s", strings.Join(args, " ")), logger.LogDebug)

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("update trust info for recipes failed: %w", err)
	}

	logger.Logger("✅ Trust info updated for all recipes", logger.LogSuccess)
	return outputBuffer.String(), nil
}
