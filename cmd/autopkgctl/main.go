// cmd/autopkgctl/main.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
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
	recipesStr string
	useToken   bool

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
	// Debug output
	logger.Logger(fmt.Sprintf("After parsing, recipes flag value: '%s'", recipesStr), logger.LogDebug)

	// Parse recipe list
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

	// Resolve dependencies
	for _, recipe := range recipes {
		logger.Logger(fmt.Sprintf("🔄 Resolving dependencies for: %s", recipe), logger.LogInfo)

		dependencies, err := autopkg.ResolveRecipeDependencies(recipe, useToken, prefsPath)
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
	// Parse recipes
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

	// Verify trust info
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
	// Parse recipes
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

	// Run recipes
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

	// Generate a summary report
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

	// Generate JSON report if requested
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
	// Clean cache
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
		level = os.Getenv("LOG_LEVEL") // Fallback to environment variable
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
		return logger.LogInfo // Default to INFO if invalid or unset
	}
}
