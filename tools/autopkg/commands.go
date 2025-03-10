// commands.go contains a set of wrapped autopkg command line operations with centralized logging
package autopkg

import (
	"bytes"
	"fmt"
	"io"
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

	var outputBuffer bytes.Buffer
	outWriter := io.MultiWriter(os.Stdout, &outputBuffer)
	errWriter := io.MultiWriter(os.Stderr, &outputBuffer)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

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

	var outputBuffer bytes.Buffer
	outWriter := io.MultiWriter(os.Stdout, &outputBuffer)
	errWriter := io.MultiWriter(os.Stderr, &outputBuffer)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

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

	// Add recipes
	args = append(args, recipes...)

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	outWriter := io.MultiWriter(os.Stdout, &outputBuffer)
	errWriter := io.MultiWriter(os.Stderr, &outputBuffer)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("recipe installation failed: %w", err)
	}

	logger.Logger("✅ Recipe installation completed", logger.LogSuccess)
	return outputBuffer.String(), nil
}

// ListProcessors lists available core Processors
func ListProcessors(prefsPath string) (string, error) {
	logger.Logger("📋 Listing available processors...", logger.LogInfo)

	args := []string{"list-processors"}
	if prefsPath != "" {
		args = append(args, "--prefs", prefsPath)
	}

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	outWriter := io.MultiWriter(os.Stdout, &outputBuffer)
	errWriter := io.MultiWriter(os.Stderr, &outputBuffer)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

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

	var outputBuffer bytes.Buffer
	outWriter := io.MultiWriter(os.Stdout, &outputBuffer)
	errWriter := io.MultiWriter(os.Stderr, &outputBuffer)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("list recipes failed: %w", err)
	}

	return outputBuffer.String(), nil
}

// ListRepos lists installed recipe repositories
func ListRepos(prefsPath string) (string, error) {
	logger.Logger("📋 Listing installed recipe repositories...", logger.LogInfo)

	args := []string{"repo-list"}
	if prefsPath != "" {
		args = append(args, "--prefs", prefsPath)
	}

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	outWriter := io.MultiWriter(os.Stdout, &outputBuffer)
	errWriter := io.MultiWriter(os.Stderr, &outputBuffer)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("list repos failed: %w", err)
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
func MakeOverride(recipe string, options *MakeOverrideOptions) (string, error) {
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

	var outputBuffer bytes.Buffer
	outWriter := io.MultiWriter(os.Stdout, &outputBuffer)
	errWriter := io.MultiWriter(os.Stderr, &outputBuffer)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("make override failed: %w", err)
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

	var outputBuffer bytes.Buffer
	outWriter := io.MultiWriter(os.Stdout, &outputBuffer)
	errWriter := io.MultiWriter(os.Stderr, &outputBuffer)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("new recipe creation failed: %w", err)
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

	var outputBuffer bytes.Buffer
	outWriter := io.MultiWriter(os.Stdout, &outputBuffer)
	errWriter := io.MultiWriter(os.Stderr, &outputBuffer)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

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

		cmd := exec.Command("autopkg", args...)

		var outputBuffer bytes.Buffer
		outWriter := io.MultiWriter(os.Stdout, &outputBuffer)
		errWriter := io.MultiWriter(os.Stderr, &outputBuffer)

		cmd.Stdout = outWriter
		cmd.Stderr = errWriter

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

	logger.Logger(fmt.Sprintf("🗑️ Deleting recipe repository: %s", repoName), logger.LogInfo)

	args := []string{"repo-delete", repoName}
	if prefsPath != "" {
		args = append(args, "--prefs", prefsPath)
	}

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	outWriter := io.MultiWriter(os.Stdout, &outputBuffer)
	errWriter := io.MultiWriter(os.Stderr, &outputBuffer)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

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

	logger.Logger(fmt.Sprintf("🔄 Updating %s", repoDesc), logger.LogInfo)

	args := []string{"repo-update"}
	args = append(args, repos...)
	if prefsPath != "" {
		args = append(args, "--prefs", prefsPath)
	}

	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	outWriter := io.MultiWriter(os.Stdout, &outputBuffer)
	errWriter := io.MultiWriter(os.Stderr, &outputBuffer)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

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

	var outputBuffer bytes.Buffer
	outWriter := io.MultiWriter(os.Stdout, &outputBuffer)
	errWriter := io.MultiWriter(os.Stderr, &outputBuffer)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("search recipes failed: %w", err)
	}

	return outputBuffer.String(), nil
}

// GetVersion prints the current version of autopkg
func GetVersion() (string, error) {
	logger.Logger("ℹ️ Getting AutoPkg version", logger.LogInfo)

	cmd := exec.Command("autopkg", "version")

	var outputBuffer bytes.Buffer
	outWriter := io.MultiWriter(os.Stdout, &outputBuffer)
	errWriter := io.MultiWriter(os.Stderr, &outputBuffer)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("get version failed: %w", err)
	}

	version := strings.TrimSpace(outputBuffer.String())
	logger.Logger(fmt.Sprintf("📦 AutoPkg version: %s", version), logger.LogInfo)
	return version, nil
}

// RunRecipes runs one or more recipes with extended options
func RunRecipes(recipes []string, options *RunOptions) (string, error) {
	if options == nil {
		options = &RunOptions{}
	}

	if len(recipes) == 0 {
		return "", fmt.Errorf("at least one recipe name is required")
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

	var outputBuffer bytes.Buffer
	outWriter := io.MultiWriter(os.Stdout, &outputBuffer)
	errWriter := io.MultiWriter(os.Stderr, &outputBuffer)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("recipe run failed: %w", err)
	}

	logger.Logger("✅ Recipe run completed", logger.LogSuccess)
	return outputBuffer.String(), nil
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

// BatchRunRecipes runs multiple recipes with the same options
func BatchRunRecipes(recipes []string, options *RunOptions) (string, error) {
	if len(recipes) == 0 {
		return "", fmt.Errorf("at least one recipe name is required")
	}

	logger.Logger(fmt.Sprintf("🚀 Batch running %d recipes", len(recipes)), logger.LogInfo)

	// Run the recipes with the provided options
	output, err := RunRecipes(recipes, options)
	if err != nil {
		return output, fmt.Errorf("batch run failed: %w", err)
	}

	logger.Logger("✅ Batch recipe run completed", logger.LogSuccess)
	return output, nil
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

	// Create README.md
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

// RunRecipeWithOutput runs a recipe and captures the output
func RunRecipeWithOutput(recipe string, options *RunOptions) (string, error) {
	if options == nil {
		options = &RunOptions{}
	}

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

	// Handle verbosity levels
	if options.VerboseLevel > 0 {
		for i := 0; i < options.VerboseLevel; i++ {
			args = append(args, "-v")
		}
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

	logger.Logger(fmt.Sprintf("🚀 Running recipe and capturing output: %s", recipe), logger.LogInfo)

	logger.Logger(fmt.Sprintf("[DEBUG] Running command: autopkg %s", strings.Join(args, " ")), logger.LogDebug)

	var outputBuffer bytes.Buffer

	cmd := exec.Command("autopkg", args...)

	outWriter := io.MultiWriter(os.Stdout, &outputBuffer)
	errWriter := io.MultiWriter(os.Stderr, &outputBuffer)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

	err := cmd.Run()
	if err != nil {
		return outputBuffer.String(), fmt.Errorf("run recipe failed: %w", err)
	}

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
}

// VerifyTrustInfoForRecipes verifies parent recipe trust info for one or more recipe overrides
func VerifyTrustInfoForRecipes(recipes []string, options *VerifyTrustInfoOptions) (bool, []string, string, error) {
	if options == nil {
		options = &VerifyTrustInfoOptions{}
	}

	if len(recipes) == 0 && options.RecipeList == "" {
		return false, nil, "", fmt.Errorf("at least one recipe name or a recipe list file is required")
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

	// Handle verbosity levels
	if options.VerboseLevel > 0 {
		for i := 0; i < options.VerboseLevel; i++ {
			args = append(args, "-v")
		}
	}

	for _, dir := range options.SearchDirs {
		args = append(args, "--search-dir", dir)
	}
	for _, dir := range options.OverrideDirs {
		args = append(args, "--override-dir", dir)
	}

	// Add recipes
	args = append(args, recipes...)

	// Run the AutoPkg command
	cmd := exec.Command("autopkg", args...)

	var outputBuffer bytes.Buffer
	outWriter := io.MultiWriter(os.Stdout, &outputBuffer)
	errWriter := io.MultiWriter(os.Stderr, &outputBuffer)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

	execErr := cmd.Run()
	outputStr := outputBuffer.String()

	// Debug log to check exact output
	logger.Logger(fmt.Sprintf("DEBUG: verify-trust-info output:\n%s", outputStr), logger.LogDebug)

	// Collect failed recipes and reasons
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

	// Handle error scenario
	if execErr != nil || len(failedRecipes) > 0 {
		logger.Logger(fmt.Sprintf("❌ Trust verification failed for %d recipes", len(failedRecipes)), logger.LogError)

		// Log detailed failure reasons
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

	var outputBuffer bytes.Buffer
	outWriter := io.MultiWriter(os.Stdout, &outputBuffer)
	errWriter := io.MultiWriter(os.Stderr, &outputBuffer)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

	if err := cmd.Run(); err != nil {
		return outputBuffer.String(), fmt.Errorf("update trust info failed: %w", err)
	}

	logger.Logger("✅ Trust info updated for all recipes", logger.LogSuccess)
	return outputBuffer.String(), nil
}
