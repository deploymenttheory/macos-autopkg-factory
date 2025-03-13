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
	logLevel     string
	prefsPath    string
	repoListPath string

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
	reportPath           string
	concurrency          int
	teamsWebhook         string
	stopOnFirstError     bool
	verboseLevel         int
	verifyTrust          bool
	updateTrustOnFailure bool
	ignoreVerifyFailures bool
	searchDirs           []string
	slackWebhookRun      string // Separate from the configure flag
	slackUsernameRun     string // Separate from the configure flag
	slackChannel         string
	slackIcon            string
	variables            map[string]string
	preprocessors        []string
	postprocessors       []string

	// Cleanup command flags
	removeDownloads   bool
	removeRecipeCache bool
	keepDays          int

	// Configure command flags
	gitHubToken      string
	jssURL           string
	apiUsername      string
	apiPassword      string
	smbURL           string
	smbUsername      string
	smbPassword      string
	clientID         string
	clientSecret     string
	tenantID         string
	teamsWebhookUrl  string
	slackUsername    string
	slackWebhook     string
	failWithoutTrust bool
	overrideDir      string
	cacheDir         string
	jcds2Mode        bool

	// Make-override command flags
	overrideSearchDirs   []string
	overrideDirs         []string
	overrideName         string
	overrideForce        bool
	overridePull         bool
	overrideIgnoreDeprec bool
	overrideFormat       string
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

	// Jamf Pro integration
	configureCmd.Flags().StringVar(&jssURL, "jss-url", "", "Jamf Pro server URL (e.g., https://jamf.example.com)")
	configureCmd.Flags().StringVar(&apiUsername, "api-username", "", "API username for Jamf Pro authentication")
	configureCmd.Flags().StringVar(&apiPassword, "api-password", "", "API password for Jamf Pro authentication")
	configureCmd.Flags().StringVar(&smbURL, "smb-url", "", "SMB share URL for package distribution (e.g., smb://server/share)")
	configureCmd.Flags().StringVar(&smbUsername, "smb-username", "", "Username for authenticating to the SMB share")
	configureCmd.Flags().StringVar(&smbPassword, "smb-password", "", "Password for authenticating to the SMB share")
	configureCmd.Flags().BoolVar(&jcds2Mode, "jcds2-mode", false, "Enable JCDS2 mode for Jamf Cloud Distribution Service v2")

	// Microsoft Intune/Graph API
	configureCmd.Flags().StringVar(&clientID, "client-id", "", "Client ID (Application ID) for Microsoft Graph API authentication or Client ID for Jamf Pro API")
	configureCmd.Flags().StringVar(&clientSecret, "client-secret", "", "Client Secret for Microsoft Graph API authentication or Client secret for Jamf Pro API")
	configureCmd.Flags().StringVar(&tenantID, "tenant-id", "", "Microsoft Entra Tenant ID for Graph API authentication")

	// Notification services
	configureCmd.Flags().StringVar(&teamsWebhookUrl, "teams-webhook", "", "Microsoft Teams webhook URL for notifications")
	configureCmd.Flags().StringVar(&slackUsername, "slack-username", "", "Username to show in Slack notifications")
	configureCmd.Flags().StringVar(&slackWebhook, "slack-webhook", "", "Slack webhook URL for notifications")

	// AutoPkg behavior settings
	configureCmd.Flags().BoolVar(&failWithoutTrust, "fail-without-trust", false, "Fail recipes without trust info for improved security")
	configureCmd.Flags().StringVar(&overrideDir, "override-dir", "", "Directory path for storing recipe overrides")
	configureCmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Custom directory for AutoPkg cache storage")
	configureCmd.Flags().StringVar(&gitHubToken, "github-token", "", "GitHub API token for accessing private repositories and higher rate limits")

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
	recipeDepsCmd.Flags().StringVar(&repoListPath, "repo-list-path", "", "Location to export added repo's to a text file for future autopkg runs")

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

	// Make-override command
	makeOverrideCmd := &cobra.Command{
		Use:   "make-override [recipe]",
		Short: "Create an AutoPkg recipe override",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			recipe := args[0]
			logger.Logger(fmt.Sprintf("🔧 Creating override for recipe: %s", recipe), logger.LogInfo)

			options := &autopkg.MakeOverrideOptions{
				PrefsPath:         prefsPath, // Uses existing global prefsPath
				SearchDirs:        overrideSearchDirs,
				OverrideDirs:      overrideDirs,
				Name:              overrideName,
				Force:             overrideForce,
				Pull:              overridePull,
				IgnoreDeprecation: overrideIgnoreDeprec,
				Format:            overrideFormat,
			}

			output, err := autopkg.MakeOverride(recipe, options)
			if err != nil {
				logger.Logger(fmt.Sprintf("❌ Failed to create override: %v", err), logger.LogError)
				fmt.Fprintln(os.Stderr, output)
				return err
			}

			fmt.Println(output)
			return nil
		},
	}

	makeOverrideCmd.Flags().StringSliceVar(&overrideSearchDirs, "search-dir", []string{}, "Directories to search for recipes (can be specified multiple times)")
	makeOverrideCmd.Flags().StringSliceVar(&overrideDirs, "override-dir", []string{}, "Directories to search for recipe overrides (can be specified multiple times)")
	makeOverrideCmd.Flags().StringVar(&overrideName, "name", "", "Name for the override file")
	makeOverrideCmd.Flags().BoolVar(&overrideForce, "force", false, "Force overwrite an existing override file")
	makeOverrideCmd.Flags().BoolVar(&overridePull, "pull", false, "Pull the parent repos if they are missing")
	makeOverrideCmd.Flags().BoolVar(&overrideIgnoreDeprec, "ignore-deprecation", false, "Ignore deprecation warnings and create the override")
	makeOverrideCmd.Flags().StringVar(&overrideFormat, "format", "plist", "Format of the override file (default: plist, options: plist, yaml)")

	// Run command
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run AutoPkg recipes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRecipes()
		},
	}

	// Basic run options
	runCmd.Flags().StringVar(&recipesStr, "recipes", "", "Comma-separated list of autopkg recipes to run")
	runCmd.Flags().StringVar(&reportPath, "report", "", "Path to save the report")
	runCmd.Flags().IntVar(&concurrency, "concurrency", 4, "Maximum concurrent recipes")
	runCmd.Flags().BoolVar(&stopOnFirstError, "stop-on-error", false, "Stop processing if any recipe fails")
	runCmd.Flags().IntVar(&verboseLevel, "verbose", 2, "autopkg run verbosity level (0-3)")

	// Trust verification options
	runCmd.Flags().BoolVar(&verifyTrust, "verify-trust", true, "Verify trust info before running recipes")
	runCmd.Flags().BoolVar(&updateTrustOnFailure, "update-trust", true, "Update trust info if verification fails")
	runCmd.Flags().BoolVar(&ignoreVerifyFailures, "ignore-verify-failures", false, "Run recipes even if trust verification fails")

	// Search and override directories
	runCmd.Flags().StringSliceVar(&searchDirs, "search-dir", []string{}, "Additional recipe search directories")
	runCmd.Flags().StringSliceVar(&overrideDirs, "override-dir", []string{}, "Additional recipe override directories")

	// Notification options - Teams
	runCmd.Flags().StringVar(&teamsWebhook, "notify-teams", "", "Microsoft Teams webhook for notifications")

	// Notification options - Slack
	runCmd.Flags().StringVar(&slackWebhook, "notify-slack", "", "Slack webhook for notifications")
	runCmd.Flags().StringVar(&slackUsername, "slack-username", "AutoPkg Bot", "Username to display in Slack notifications")
	runCmd.Flags().StringVar(&slackChannel, "slack-channel", "", "Slack channel for notifications")
	runCmd.Flags().StringVar(&slackIcon, "slack-icon", ":package:", "Emoji icon for Slack notifications")

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
	rootCmd.AddCommand(makeOverrideCmd)

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
			fmt.Printf("❌ Root account check failed: %v\n", err)
			return err
		}
		fmt.Println("✅ Root account check passed - not running as root")
	}

	if checkGit {
		if err := autopkg.CheckGit(); err != nil {
			fmt.Printf("❌ Git install check failed: %v\n", err)
			return err
		}
		fmt.Println("✅ Git install check passed")
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

	// Get existing preferences or initialize empty map
	_, err := autopkg.GetAutoPkgPreferences(expandedPrefsPath)
	if err != nil {
		logger.Logger("ℹ️ Creating new preferences file", logger.LogInfo)
	}

	// Prepare updates to preferences
	updates := make(map[string]interface{})

	// GitHub token
	if gitHubToken != "" {
		// Save token to the autopkg token file
		homeDir, err := os.UserHomeDir()
		if err == nil {
			tokenPath := filepath.Join(homeDir, ".autopkg_gh_token")
			if err := os.WriteFile(tokenPath, []byte(gitHubToken), 0600); err == nil {
				logger.Logger(fmt.Sprintf("✅ Wrote GitHub token to %s", tokenPath), logger.LogSuccess)
				updates["GITHUB_TOKEN_PATH"] = tokenPath
			} else {
				logger.Logger(fmt.Sprintf("❌ Failed to write GitHub token file: %v", err), logger.LogError)
			}
		}
	}

	// Jamf Pro integration
	if jssURL != "" {
		updates["JSS_URL"] = jssURL
	}
	if apiUsername != "" {
		updates["API_USERNAME"] = apiUsername
	}
	if apiPassword != "" {
		updates["API_PASSWORD"] = apiPassword
	}
	if smbURL != "" {
		updates["SMB_URL"] = smbURL
	}
	if smbUsername != "" {
		updates["SMB_USERNAME"] = smbUsername
	}
	if smbPassword != "" {
		updates["SMB_PASSWORD"] = smbPassword
	}
	if cmd.Flags().Changed("jcds2-mode") {
		updates["jcds2_mode"] = jcds2Mode
	}

	// Microsoft Intune/Graph API
	if clientID != "" {
		updates["CLIENT_ID"] = clientID
	}
	if clientSecret != "" {
		updates["CLIENT_SECRET"] = clientSecret
	}
	if tenantID != "" {
		updates["TENANT_ID"] = tenantID
	}

	// Notification services
	if teamsWebhookUrl != "" {
		updates["TEAMS_WEBHOOK"] = teamsWebhookUrl
	}
	if slackUsername != "" {
		updates["SLACK_USERNAME"] = slackUsername
	}
	if slackWebhook != "" {
		updates["SLACK_WEBHOOK"] = slackWebhook
	}

	// AutoPkg behavior settings
	if cmd.Flags().Changed("fail-without-trust") {
		updates["FAIL_RECIPES_WITHOUT_TRUST_INFO"] = failWithoutTrust
	}
	if overrideDir != "" {
		updates["RECIPE_OVERRIDE_DIRS"] = overrideDir
	}
	if cacheDir != "" {
		updates["CACHE_DIR"] = cacheDir
	}

	// Check environment variables if flags weren't provided
	// GitHub token from environment
	if gitHubToken == "" && os.Getenv("GITHUB_TOKEN") != "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			tokenPath := filepath.Join(homeDir, ".autopkg_gh_token")
			if err := os.WriteFile(tokenPath, []byte(os.Getenv("GITHUB_TOKEN")), 0600); err == nil {
				logger.Logger(fmt.Sprintf("✅ Wrote GitHub token from environment to %s", tokenPath), logger.LogSuccess)
				updates["GITHUB_TOKEN_PATH"] = tokenPath
			}
		}
	}

	// Jamf Pro environment variables
	if jssURL == "" && os.Getenv("JSS_URL") != "" {
		updates["JSS_URL"] = os.Getenv("JSS_URL")
	}
	if apiUsername == "" && os.Getenv("API_USERNAME") != "" {
		updates["API_USERNAME"] = os.Getenv("API_USERNAME")
	}
	if apiPassword == "" && os.Getenv("API_PASSWORD") != "" {
		updates["API_PASSWORD"] = os.Getenv("API_PASSWORD")
	}
	if smbURL == "" && os.Getenv("SMB_URL") != "" {
		updates["SMB_URL"] = os.Getenv("SMB_URL")
	}
	if smbUsername == "" && os.Getenv("SMB_USERNAME") != "" {
		updates["SMB_USERNAME"] = os.Getenv("SMB_USERNAME")
	}
	if smbPassword == "" && os.Getenv("SMB_PASSWORD") != "" {
		updates["SMB_PASSWORD"] = os.Getenv("SMB_PASSWORD")
	}

	// Microsoft Intune/Graph API environment variables
	if clientID == "" && os.Getenv("CLIENT_ID") != "" {
		updates["CLIENT_ID"] = os.Getenv("CLIENT_ID")
	}
	if clientSecret == "" && os.Getenv("CLIENT_SECRET") != "" {
		updates["CLIENT_SECRET"] = os.Getenv("CLIENT_SECRET")
	}
	if tenantID == "" && os.Getenv("TENANT_ID") != "" {
		updates["TENANT_ID"] = os.Getenv("TENANT_ID")
	}

	// Notification services environment variables
	if teamsWebhookUrl == "" && os.Getenv("TEAMS_WEBHOOK") != "" {
		updates["TEAMS_WEBHOOK"] = os.Getenv("TEAMS_WEBHOOK")
	}
	if slackUsername == "" && os.Getenv("SLACK_USERNAME") != "" {
		updates["SLACK_USERNAME"] = os.Getenv("SLACK_USERNAME")
	}
	if slackWebhook == "" && os.Getenv("SLACK_WEBHOOK") != "" {
		updates["SLACK_WEBHOOK"] = os.Getenv("SLACK_WEBHOOK")
	}

	// AutoPkg behavior environment variables
	if overrideDir == "" && os.Getenv("RECIPE_OVERRIDE_DIRS") != "" {
		updates["RECIPE_OVERRIDE_DIRS"] = os.Getenv("RECIPE_OVERRIDE_DIRS")
	}
	if cacheDir == "" && os.Getenv("CACHE_DIR") != "" {
		updates["CACHE_DIR"] = os.Getenv("CACHE_DIR")
	}

	// Check if we have any updates to make
	if len(updates) > 0 {
		// Update preferences
		if err := autopkg.UpdateAutoPkgPreferences(expandedPrefsPath, updates); err != nil {
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

		dependencies, err := autopkg.ResolveRecipeDependencies(recipe, useToken, prefsPath, dryRun, repoListPath)
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
		SearchDirs:           searchDirs,
		OverrideDirs:         overrideDirs,
		VerifyTrust:          verifyTrust,
		UpdateTrustOnFailure: updateTrustOnFailure,
		IgnoreVerifyFailures: ignoreVerifyFailures,
		ReportPlist:          reportPath,
		VerboseLevel:         verboseLevel,
		Variables:            variables,
		MaxConcurrentRecipes: concurrency,
		StopOnFirstError:     stopOnFirstError,
		Notification: autopkg.NotificationOptions{
			EnableTeams:   teamsWebhook != "",
			TeamsWebhook:  teamsWebhook,
			EnableSlack:   slackWebhook != "",
			SlackWebhook:  slackWebhook,
			SlackUsername: slackUsername,
			SlackChannel:  slackChannel,
			SlackIcon:     slackIcon,
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
