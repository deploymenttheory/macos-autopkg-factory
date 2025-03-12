// cmd/autopkgctl/main.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/autopkg"
	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	logLevel  string
	prefsPath string

	// Setup command flags
	forceUpdate bool
	useBeta     bool
	checkGit    bool
	checkRoot   bool

	// Repo-add command flags
	reposStr string

	// Recipe-repo-deps command flags
	recipesStr   string
	useToken     bool
	skipExisting bool
	dryRun       bool

	// Verify-trust command flags
	updateTrust bool

	// Run command flags
	reportPath   string
	concurrency  int
	teamsWebhook string

	// Cleanup command flags
	removeDownloads   bool
	removeRecipeCache bool
	keepDays          int

	// Configure command flags
	gitHubToken      string
	jssURL           string
	apiUsername      string
	apiPassword      string
	clientID         string
	clientSecret     string
	tenantID         string
	teamsWebhookUrl  string
	failWithoutTrust bool
)

func main() {
	// Root command
	rootCmd := &cobra.Command{
		Use:   "autopkgctl",
		Short: "A CLI tool for managing AutoPkg",
		Long:  "autopkgctl is a command-line interface for managing AutoPkg operations in CI/CD environments",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Set log level
			level := getLogLevel(logLevel)
			logger.SetLogLevel(level)

			// Debug command arguments
			if level == logger.LogDebug {
				logger.Logger("Command-line arguments:", logger.LogDebug)
				for i, arg := range os.Args {
					logger.Logger(fmt.Sprintf("Arg[%d]: '%s'", i, arg), logger.LogDebug)
				}
			}
		},
	}

	// Add global flags
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "Set log level (DEBUG, INFO, WARNING, ERROR, SUCCESS)")
	rootCmd.PersistentFlags().StringVar(&prefsPath, "prefs", "", "Path to AutoPkg preferences file")

	// Setup command
	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Set up AutoPkg environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetup()
		},
	}

	setupCmd.Flags().BoolVar(&forceUpdate, "force-update", false, "Force update AutoPkg if already installed")
	setupCmd.Flags().BoolVar(&useBeta, "use-beta", false, "Use beta version of AutoPkg")
	setupCmd.Flags().BoolVar(&checkGit, "check-git", true, "Check if Git is installed")
	setupCmd.Flags().BoolVar(&checkRoot, "check-root", true, "Check if running as root")

	// Configure command
	configureCmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure AutoPkg settings",
		Long:  "Configure AutoPkg settings including GitHub token, Jamf credentials, and other preferences",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigure(cmd)
		},
	}

	// Configure command flags
	configureCmd.Flags().StringVar(&gitHubToken, "github-token", "", "GitHub token for API access")
	configureCmd.Flags().StringVar(&jssURL, "jss-url", "", "Jamf Pro server URL")
	configureCmd.Flags().StringVar(&apiUsername, "api-username", "", "API username for Jamf Pro")
	configureCmd.Flags().StringVar(&apiPassword, "api-password", "", "API password for Jamf Pro")
	configureCmd.Flags().StringVar(&clientID, "client-id", "", "Client ID for Jamf Pro API")
	configureCmd.Flags().StringVar(&clientSecret, "client-secret", "", "Client secret for Jamf Pro API")
	configureCmd.Flags().StringVar(&tenantID, "tenant-id", "", "Tenant ID for Microsoft services")
	configureCmd.Flags().StringVar(&teamsWebhookUrl, "teams-webhook", "", "Microsoft Teams webhook URL")
	configureCmd.Flags().BoolVar(&failWithoutTrust, "fail-without-trust", false, "Fail recipes without trust info")

	// Repo-add command
	repoAddCmd := &cobra.Command{
		Use:   "repo-add",
		Short: "Add AutoPkg repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRepoAdd()
		},
	}

	repoAddCmd.Flags().StringVar(&reposStr, "repos", "", "Comma-separated list of repositories to add")

	// Recipe-repo-deps command
	recipeDepsCmd := &cobra.Command{
		Use:   "recipe-repo-deps",
		Short: "Resolve recipe repository dependencies",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRecipeDeps()
		},
	}

	recipeDepsCmd.Flags().StringVar(&recipesStr, "recipes", "", "Comma-separated list of recipes to analyze")
	recipeDepsCmd.Flags().BoolVar(&useToken, "use-token", true, "Use GitHub token for authentication")
	recipeDepsCmd.Flags().BoolVar(&skipExisting, "skip-existing", true, "Skip repositories that are already added")
	recipeDepsCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Only show dependencies without adding them")

	// Verify-trust command
	verifyTrustCmd := &cobra.Command{
		Use:   "verify-trust",
		Short: "Verify trust info for recipes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVerifyTrust()
		},
	}

	verifyTrustCmd.Flags().BoolVar(&updateTrust, "update", true, "Update trust info if verification fails")
	verifyTrustCmd.Flags().StringVar(&recipesStr, "recipes", "", "Comma-separated list of recipes to verify")

	// Run command
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run AutoPkg recipes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRecipes()
		},
	}

	runCmd.Flags().StringVar(&recipesStr, "recipes", "", "Comma-separated list of recipes to run")
	runCmd.Flags().StringVar(&reportPath, "report", "", "Path to save the report")
	runCmd.Flags().IntVar(&concurrency, "concurrency", 4, "Maximum concurrent recipes")
	runCmd.Flags().StringVar(&teamsWebhook, "notify-teams", "", "Microsoft Teams webhook for notifications")

	// Cleanup command
	cleanupCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Clean AutoPkg cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCleanup()
		},
	}

	cleanupCmd.Flags().BoolVar(&removeDownloads, "remove-downloads", true, "Remove downloads cache")
	cleanupCmd.Flags().BoolVar(&removeRecipeCache, "remove-recipe-cache", true, "Remove recipe cache")
	cleanupCmd.Flags().IntVar(&keepDays, "keep-days", 0, "Keep files newer than this many days")

	// Add commands to root
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(configureCmd)
	rootCmd.AddCommand(repoAddCmd)
	rootCmd.AddCommand(recipeDepsCmd)
	rootCmd.AddCommand(verifyTrustCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(cleanupCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func runSetup() error {
	// Execute setup
	if checkRoot {
		if err := autopkg.RootCheck(); err != nil {
			fmt.Printf("❌ Root check failed: %v\n", err)
			return err
		}
		fmt.Println("✅ Root check passed - not running as root")
	}

	if checkGit {
		if err := autopkg.CheckGit(); err != nil {
			fmt.Printf("❌ Git check failed: %v\n", err)
			return err
		}
		fmt.Println("✅ Git check passed")
	}

	config := &autopkg.InstallConfig{
		ForceUpdate: forceUpdate,
		UseBeta:     useBeta,
	}

	version, err := autopkg.InstallAutoPkg(config)
	if err != nil {
		fmt.Printf("❌ AutoPkg installation failed: %v\n", err)
		return err
	}
	fmt.Printf("✅ AutoPkg %s installed successfully\n", version)

	return nil
}

func runConfigure(cmd *cobra.Command) error {
	logger.Logger("🔧 Configuring AutoPkg settings", logger.LogInfo)

	// Apply tilde expansion to prefsPath if needed
	expandedPrefsPath := prefsPath
	if strings.HasPrefix(prefsPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			expandedPrefsPath = filepath.Join(homeDir, prefsPath[2:])
		}
	}

	// Create preferences directory if it doesn't exist
	prefsDir := filepath.Dir(expandedPrefsPath)
	if err := os.MkdirAll(prefsDir, 0755); err != nil {
		logger.Logger(fmt.Sprintf("❌ Failed to create preferences directory: %v", err), logger.LogError)
		return err
	}

	// Try to read existing preferences
	var prefs *autopkg.PreferencesData
	existingPrefs, err := autopkg.GetAutoPkgPreferences(expandedPrefsPath)
	if err != nil {
		// If file doesn't exist, create a new preferences struct
		logger.Logger("ℹ️ Creating new preferences file", logger.LogInfo)
		prefs = &autopkg.PreferencesData{
			RECIPE_REPOS:       make(map[string]interface{}),
			RECIPE_SEARCH_DIRS: []string{},
		}
	} else {
		prefs = existingPrefs
	}

	// Update preferences with provided flags (if any)
	updated := false

	// GitHub token
	if gitHubToken != "" {
		// Save token to the autopkg token file as well
		homeDir, err := os.UserHomeDir()
		if err == nil {
			tokenPath := filepath.Join(homeDir, ".autopkg_gh_token")
			if err := os.WriteFile(tokenPath, []byte(gitHubToken), 0600); err == nil {
				logger.Logger(fmt.Sprintf("✅ Wrote GitHub token to %s", tokenPath), logger.LogSuccess)
				prefs.GITHUB_TOKEN_PATH = tokenPath
				updated = true
			} else {
				logger.Logger(fmt.Sprintf("❌ Failed to write GitHub token file: %v", err), logger.LogError)
			}
		}
	}

	// Jamf Pro settings
	if jssURL != "" {
		prefs.JSS_URL = jssURL
		updated = true
	}
	if apiUsername != "" {
		prefs.API_USERNAME = apiUsername
		updated = true
	}
	if apiPassword != "" {
		prefs.API_PASSWORD = apiPassword
		updated = true
	}
	if clientID != "" {
		prefs.CLIENT_ID = clientID
		updated = true
	}
	if clientSecret != "" {
		prefs.CLIENT_SECRET = clientSecret
		updated = true
	}
	if tenantID != "" {
		prefs.TENANT_ID = tenantID
		updated = true
	}

	// Microsoft Teams webhook
	if teamsWebhookUrl != "" {
		if prefs.AdditionalPreferences == nil {
			prefs.AdditionalPreferences = make(map[string]interface{})
		}
		prefs.AdditionalPreferences["TEAMS_WEBHOOK"] = teamsWebhookUrl
		updated = true
	}

	// Trust settings
	if cmd.Flags().Changed("fail-without-trust") {
		prefs.FAIL_RECIPES_WITHOUT_TRUST_INFO = failWithoutTrust
		updated = true
	}

	// Check environment variables if flags weren't provided
	if gitHubToken == "" && os.Getenv("GITHUB_TOKEN") != "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			tokenPath := filepath.Join(homeDir, ".autopkg_gh_token")
			if err := os.WriteFile(tokenPath, []byte(os.Getenv("GITHUB_TOKEN")), 0600); err == nil {
				logger.Logger(fmt.Sprintf("✅ Wrote GitHub token from environment to %s", tokenPath), logger.LogSuccess)
				prefs.GITHUB_TOKEN_PATH = tokenPath
				updated = true
			}
		}
	}

	if jssURL == "" && os.Getenv("JSS_URL") != "" {
		prefs.JSS_URL = os.Getenv("JSS_URL")
		updated = true
	}
	if apiUsername == "" && os.Getenv("API_USERNAME") != "" {
		prefs.API_USERNAME = os.Getenv("API_USERNAME")
		updated = true
	}
	if apiPassword == "" && os.Getenv("API_PASSWORD") != "" {
		prefs.API_PASSWORD = os.Getenv("API_PASSWORD")
		updated = true
	}
	if clientID == "" && os.Getenv("CLIENT_ID") != "" {
		prefs.CLIENT_ID = os.Getenv("CLIENT_ID")
		updated = true
	}
	if clientSecret == "" && os.Getenv("CLIENT_SECRET") != "" {
		prefs.CLIENT_SECRET = os.Getenv("CLIENT_SECRET")
		updated = true
	}
	if tenantID == "" && os.Getenv("TENANT_ID") != "" {
		prefs.TENANT_ID = os.Getenv("TENANT_ID")
		updated = true
	}
	if teamsWebhookUrl == "" && os.Getenv("TEAMS_WEBHOOK") != "" {
		if prefs.AdditionalPreferences == nil {
			prefs.AdditionalPreferences = make(map[string]interface{})
		}
		prefs.AdditionalPreferences["TEAMS_WEBHOOK"] = os.Getenv("TEAMS_WEBHOOK")
		updated = true
	}

	// If any settings were updated, write the preferences file
	if updated {
		if err := autopkg.SetAutoPkgPreferences(expandedPrefsPath, prefs); err != nil {
			logger.Logger(fmt.Sprintf("❌ Failed to write preferences: %v", err), logger.LogError)
			return err
		}
		logger.Logger("✅ AutoPkg preferences updated successfully", logger.LogSuccess)
	} else {
		logger.Logger("ℹ️ No changes to preferences", logger.LogInfo)
	}

	// Verify the configuration by running autopkg repo-list
	cmdExec := exec.Command("autopkg", "repo-list")
	if prefsPath != "" {
		cmdExec.Args = append(cmdExec.Args, "--prefs", expandedPrefsPath)
	}
	output, err := cmdExec.CombinedOutput()
	if err != nil {
		logger.Logger(fmt.Sprintf("⚠️ Failed to verify configuration: %v", err), logger.LogWarning)
		logger.Logger(fmt.Sprintf("Output: %s", string(output)), logger.LogDebug)
	} else {
		logger.Logger("✅ Configuration verified successfully", logger.LogSuccess)
		logger.Logger(fmt.Sprintf("Repository list:\n%s", string(output)), logger.LogInfo)
	}

	return nil
}

func runRepoAdd() error {
	// Parse repos
	var repos []string
	if reposStr != "" {
		for _, r := range strings.Split(reposStr, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				repos = append(repos, r)
			}
		}
	}

	if len(repos) == 0 {
		return fmt.Errorf("no repositories specified")
	}

	// Add repositories
	output, err := autopkg.AddRepo(repos, prefsPath)
	if err != nil {
		fmt.Printf("❌ Failed to add repositories: %v\n", err)
		fmt.Println(output)
		return err
	}
	fmt.Println("✅ Repositories added successfully")
	fmt.Println(output)

	return nil
}

func runRecipeDeps() error {
	logger.Logger(fmt.Sprintf("After parsing, recipes flag value: '%s'", recipesStr), logger.LogDebug)

	var recipes []string
	if recipesStr != "" {
		for _, r := range strings.Split(recipesStr, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				recipes = append(recipes, r)
			}
		}
	}

	logger.Logger(fmt.Sprintf("📋 Parsed Recipes: %v", recipes), logger.LogDebug)

	if len(recipes) == 0 {
		return fmt.Errorf("no recipes specified")
	}

	for _, recipe := range recipes {
		logger.Logger(fmt.Sprintf("🔄 Resolving dependencies for: %s", recipe), logger.LogInfo)

		dependencies, err := autopkg.ResolveRecipeDependencies(recipe, useToken, prefsPath, dryRun)
		if err != nil {
			logger.Logger(fmt.Sprintf("❌ Failed to resolve dependencies for %s: %v", recipe, err), logger.LogError)
			continue
		}

		logger.Logger(fmt.Sprintf("✅ Found %d dependencies for %s", len(dependencies), recipe), logger.LogSuccess)
		for _, dep := range dependencies {
			fmt.Printf("- %s: %s\n", dep.RecipeIdentifier, dep.RepoURL)
		}
	}

	return nil
}

func runVerifyTrust() error {
	var recipes []string
	if recipesStr != "" {
		for _, r := range strings.Split(recipesStr, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				recipes = append(recipes, r)
			}
		}
	}

	if len(recipes) == 0 {
		return fmt.Errorf("no recipes specified")
	}

	verifyOptions := &autopkg.VerifyTrustInfoOptions{
		PrefsPath:    prefsPath,
		VerboseLevel: 1,
	}

	success, failedRecipes, output, err := autopkg.VerifyTrustInfoForRecipes(recipes, verifyOptions)
	fmt.Println(output)

	if err != nil || !success {
		fmt.Printf("⚠️ Trust verification failed for %d recipes\n", len(failedRecipes))

		if updateTrust && len(failedRecipes) > 0 {
			fmt.Println("🔄 Attempting to update trust info...")

			updateOptions := &autopkg.UpdateTrustInfoOptions{
				PrefsPath: prefsPath,
			}

			updateOutput, updateErr := autopkg.UpdateTrustInfoForRecipes(failedRecipes, updateOptions)
			fmt.Println(updateOutput)

			if updateErr != nil {
				fmt.Printf("❌ Failed to update trust info: %v\n", updateErr)
				return updateErr
			}

			fmt.Println("✅ Trust info updated successfully")
		} else {
			fmt.Println("❌ Trust verification failed and update not requested")
			return fmt.Errorf("trust verification failed")
		}
	} else {
		fmt.Println("✅ Trust verification passed for all recipes")
	}

	return nil
}

func runRecipes() error {
	var recipes []string
	if recipesStr != "" {
		for _, r := range strings.Split(recipesStr, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				recipes = append(recipes, r)
			}
		}
	}

	if len(recipes) == 0 {
		return fmt.Errorf("no recipes specified")
	}

	options := &autopkg.RecipeBatchRunOptions{
		PrefsPath:            prefsPath,
		MaxConcurrentRecipes: concurrency,
		StopOnFirstError:     false,
		VerboseLevel:         2,
		ReportPlist:          reportPath,
		Notification: autopkg.NotificationOptions{
			EnableTeams:  teamsWebhook != "",
			TeamsWebhook: teamsWebhook,
		},
	}

	results, err := autopkg.RecipeBatchProcessing(recipes, options)

	successCount := 0
	failCount := 0

	fmt.Println("\nRecipe Execution Results:")
	fmt.Println("=========================")

	for recipe, result := range results {
		if result.ExecutionError == nil {
			successCount++
			fmt.Printf("✅ %s: Success\n", recipe)
		} else {
			failCount++
			fmt.Printf("❌ %s: %v\n", recipe, result.ExecutionError)
		}
	}

	fmt.Printf("\nSummary: %d successful, %d failed\n", successCount, failCount)

	if reportPath != "" {
		reportData, _ := json.MarshalIndent(results, "", "  ")
		if err := os.WriteFile(reportPath, reportData, 0644); err != nil {
			fmt.Printf("⚠️ Failed to write report: %v\n", err)
		} else {
			fmt.Printf("✅ Report written to %s\n", reportPath)
		}
	}

	if failCount > 0 || err != nil {
		return fmt.Errorf("recipe execution failed: %d recipes failed", failCount)
	}

	fmt.Println("✅ All recipes processed successfully")
	return nil
}

func runCleanup() error {
	options := &autopkg.CleanupOptions{
		PrefsPath:         prefsPath,
		RemoveDownloads:   removeDownloads,
		RemoveRecipeCache: removeRecipeCache,
		KeepDays:          keepDays,
	}

	if err := autopkg.CleanupCache(options); err != nil {
		fmt.Printf("⚠️ Cache cleanup failed: %v\n", err)
		return err
	}

	fmt.Println("✅ AutoPkg cache cleaned successfully")
	return nil
}

func getLogLevel(cliLogLevel string) int {
	// Use CLI flag if set, otherwise check the environment variable
	level := cliLogLevel
	if level == "" {
		level = os.Getenv("LOG_LEVEL")
	}

	switch strings.ToUpper(level) {
	case "DEBUG":
		return logger.LogDebug
	case "INFO":
		return logger.LogInfo
	case "WARNING":
		return logger.LogWarning
	case "ERROR":
		return logger.LogError
	case "SUCCESS":
		return logger.LogSuccess
	default:
		return logger.LogInfo
	}
}
