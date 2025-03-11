// cmd/autopkgctl/main.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/autopkg"
	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

func main() {
	// Define log level flag (CLI override)
	logLevelFlag := flag.String("log-level", "", "Set log level (DEBUG, INFO, WARNING, ERROR, SUCCESS)")
	flag.Parse()
	logLevel := getLogLevel(*logLevelFlag)
	logger.SetLogLevel(logLevel)

	// Define root command
	setupCmd := flag.NewFlagSet("setup", flag.ExitOnError)
	repoAddCmd := flag.NewFlagSet("repo-add", flag.ExitOnError)
	analyzeDepsCmd := flag.NewFlagSet("recipe-repo-deps", flag.ExitOnError)
	verifyTrustCmd := flag.NewFlagSet("verify-trust", flag.ExitOnError)
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	cleanupCmd := flag.NewFlagSet("cleanup", flag.ExitOnError)

	// Set up shared flags
	prefsPath := ""

	// Parse subcommands
	if len(os.Args) < 2 {
		fmt.Println("Expected subcommand: setup, repo-add, analyze-deps, verify-trust, run, cleanup")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "setup":
		// Define setup flags
		prefsPathSetup := setupCmd.String("prefs", "", "Path to AutoPkg preferences file")
		forceUpdate := setupCmd.Bool("force-update", false, "Force update AutoPkg if already installed")
		useBeta := setupCmd.Bool("use-beta", false, "Use beta version of AutoPkg")
		checkGit := setupCmd.Bool("check-git", true, "Check if Git is installed")
		checkRoot := setupCmd.Bool("check-root", true, "Check if running as root")

		setupCmd.Parse(os.Args[2:])
		prefsPath = *prefsPathSetup

		// Execute setup
		if *checkRoot {
			if err := autopkg.RootCheck(); err != nil {
				fmt.Printf("❌ Root check failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✅ Root check passed - not running as root")
		}

		if *checkGit {
			if err := autopkg.CheckGit(); err != nil {
				fmt.Printf("❌ Git check failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✅ Git check passed")
		}

		config := &autopkg.InstallConfig{
			ForceUpdate: *forceUpdate,
			UseBeta:     *useBeta,
		}

		version, err := autopkg.InstallAutoPkg(config)
		if err != nil {
			fmt.Printf("❌ AutoPkg installation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ AutoPkg %s installed successfully\n", version)

	case "repo-add":
		// Define repo-add flags
		prefsPathRepo := repoAddCmd.String("prefs", "", "Path to AutoPkg preferences file")
		reposStr := repoAddCmd.String("repos", "", "Comma-separated list of repositories to add")

		repoAddCmd.Parse(os.Args[2:])
		prefsPath = *prefsPathRepo

		// Parse repos
		repos := strings.Split(*reposStr, ",")

		// Add repositories
		output, err := autopkg.AddRepo(repos, prefsPath)
		if err != nil {
			fmt.Printf("❌ Failed to add repositories: %v\n", err)
			fmt.Println(output)
			os.Exit(1)
		}
		fmt.Println("✅ Repositories added successfully")
		fmt.Println(output)

	case "recipe-repo-deps":
		prefsPathAnalyze := analyzeDepsCmd.String("prefs", "", "Path to AutoPkg preferences file")
		recipesStr := analyzeDepsCmd.String("recipes", "", "Comma-separated list of recipes to analyze")
		useToken := analyzeDepsCmd.Bool("use-token", true, "Use GitHub token for authentication")
		analyzeDepsCmd.Parse(os.Args[2:])
		prefsPath = *prefsPathAnalyze

		// Debug the raw input string
		logger.Logger(fmt.Sprintf("🔄 Raw recipes string: '%s'", *recipesStr), logger.LogDebug)

		var recipes []string
		if recipesStr != nil && *recipesStr != "" {
			// Trim the string and split more robustly
			trimmedStr := strings.TrimSpace(*recipesStr)
			rawRecipes := strings.Split(trimmedStr, ",")

			// Process each recipe name
			for _, r := range rawRecipes {
				r = strings.TrimSpace(r)
				if r != "" {
					recipes = append(recipes, r)
				}
			}
		}

		logger.Logger(fmt.Sprintf("📋 Parsed Recipes: %v", recipes), logger.LogDebug)

		// Resolve dependencies using the updated function
		for _, recipe := range recipes {
			logger.Logger(fmt.Sprintf("🔄 Resolving dependencies for: %s", recipe), logger.LogInfo)

			dependencies, err := autopkg.ResolveRecipeDependencies(recipe, *useToken, prefsPath)
			if err != nil {
				logger.Logger(fmt.Sprintf("❌ Failed to resolve dependencies for %s: %v", recipe, err), logger.LogError)
				continue
			}

			logger.Logger(fmt.Sprintf("✅ Found %d dependencies for %s", len(dependencies), recipe), logger.LogSuccess)
			for _, dep := range dependencies {
				fmt.Printf("- %s: %s\n", dep.RecipeIdentifier, dep.RepoURL)
			}
		}

	case "verify-trust":
		// Define verify-trust flags
		prefsPathVerify := verifyTrustCmd.String("prefs", "", "Path to AutoPkg preferences file")
		recipesStr := verifyTrustCmd.String("recipes", "", "Space-separated list of recipes to verify")
		updateTrust := verifyTrustCmd.Bool("update", true, "Update trust info if verification fails")

		verifyTrustCmd.Parse(os.Args[2:])
		prefsPath = *prefsPathVerify

		// Parse recipes
		recipes := strings.Fields(*recipesStr)

		// Verify trust info
		verifyOptions := &autopkg.VerifyTrustInfoOptions{
			PrefsPath:    prefsPath,
			VerboseLevel: 1,
		}

		success, failedRecipes, output, err := autopkg.VerifyTrustInfoForRecipes(recipes, verifyOptions)
		fmt.Println(output)

		if err != nil || !success {
			fmt.Printf("⚠️ Trust verification failed for %d recipes\n", len(failedRecipes))

			if *updateTrust && len(failedRecipes) > 0 {
				fmt.Println("🔄 Attempting to update trust info...")

				updateOptions := &autopkg.UpdateTrustInfoOptions{
					PrefsPath: prefsPath,
				}

				updateOutput, updateErr := autopkg.UpdateTrustInfoForRecipes(failedRecipes, updateOptions)
				fmt.Println(updateOutput)

				if updateErr != nil {
					fmt.Printf("❌ Failed to update trust info: %v\n", updateErr)
					os.Exit(1)
				}

				fmt.Println("✅ Trust info updated successfully")
			} else {
				fmt.Println("❌ Trust verification failed and update not requested")
				os.Exit(1)
			}
		} else {
			fmt.Println("✅ Trust verification passed for all recipes")
		}

	case "run":
		// Define run flags
		prefsPathRun := runCmd.String("prefs", "", "Path to AutoPkg preferences file")
		recipesStr := runCmd.String("recipes", "", "Space-separated list of recipes to run")
		reportPath := runCmd.String("report", "", "Path to save the report")
		concurrency := runCmd.Int("concurrency", 4, "Maximum concurrent recipes")
		teamsWebhook := runCmd.String("notify-teams", "", "Microsoft Teams webhook for notifications")

		runCmd.Parse(os.Args[2:])
		prefsPath = *prefsPathRun

		// Parse recipes
		recipes := strings.Fields(*recipesStr)

		// Run recipes
		options := &autopkg.RecipeBatchRunOptions{
			PrefsPath:            prefsPath,
			MaxConcurrentRecipes: *concurrency,
			StopOnFirstError:     false,
			VerboseLevel:         2,
			ReportPlist:          *reportPath,
			Notification: autopkg.NotificationOptions{
				EnableTeams:  *teamsWebhook != "",
				TeamsWebhook: *teamsWebhook,
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
		if *reportPath != "" {
			reportData, _ := json.MarshalIndent(results, "", "  ")
			err := os.WriteFile(*reportPath, reportData, 0644)
			if err != nil {
				fmt.Printf("⚠️ Failed to write report: %v\n", err)
			} else {
				fmt.Printf("✅ Report written to %s\n", *reportPath)
			}
		}

		if failCount > 0 || err != nil {
			os.Exit(1)
		}

		fmt.Println("✅ All recipes processed successfully")

	case "cleanup":
		// Define cleanup flags
		prefsPathCleanup := cleanupCmd.String("prefs", "", "Path to AutoPkg preferences file")
		removeDownloads := cleanupCmd.Bool("remove-downloads", true, "Remove downloads cache")
		removeRecipeCache := cleanupCmd.Bool("remove-recipe-cache", true, "Remove recipe cache")
		keepDays := cleanupCmd.Int("keep-days", 0, "Keep files newer than this many days")

		cleanupCmd.Parse(os.Args[2:])
		prefsPath = *prefsPathCleanup

		// Clean cache
		options := &autopkg.CleanupOptions{
			PrefsPath:         prefsPath,
			RemoveDownloads:   *removeDownloads,
			RemoveRecipeCache: *removeRecipeCache,
			KeepDays:          *keepDays,
		}

		err := autopkg.CleanupCache(options)
		if err != nil {
			fmt.Printf("⚠️ Cache cleanup failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ AutoPkg cache cleaned successfully")

	default:
		fmt.Printf("Unknown subcommand: %s\n", os.Args[1])
		os.Exit(1)
	}
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
