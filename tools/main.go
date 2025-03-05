package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/deploymenttheory/macos-autopkg-factory/autopkg"
)

func main() {
	// Define command-line flags for different operation modes
	setupCmd := flag.NewFlagSet("setup", flag.ExitOnError)
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	verifyCmd := flag.NewFlagSet("verify", flag.ExitOnError)

	// Global flags for all commands
	var debugMode bool
	flag.BoolVar(&debugMode, "debug", false, "Enable debug mode with verbose output")
	flag.BoolVar(&debugMode, "d", false, "Enable debug mode with verbose output (shorthand)")

	// Print custom usage information if no arguments
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Parse the command
	switch os.Args[1] {
	case "setup":
		// Configuration for setup command (AutoPkg installation and configuration)
		setupConfig := autopkg.Config{}
		setupCmd.BoolVar(&setupConfig.ForceUpdate, "force", false, "Force the re-installation of the latest AutoPkg")
		setupCmd.BoolVar(&setupConfig.ForceUpdate, "f", false, "Force the re-installation of the latest AutoPkg (shorthand)")
		setupCmd.BoolVar(&setupConfig.UseBeta, "beta", false, "Force the installation of the pre-released version of AutoPkg")
		setupCmd.BoolVar(&setupConfig.UseBeta, "b", false, "Force the installation of the pre-released version of AutoPkg (shorthand)")
		setupCmd.BoolVar(&setupConfig.FailRecipes, "fail", true, "Fail runs if not verified")
		setupCmd.StringVar(&setupConfig.PrefsFilePath, "prefs", "", "Path to the preferences plist")
		setupCmd.BoolVar(&setupConfig.ReplacePrefs, "replace-prefs", false, "Delete the prefs file and rebuild from scratch")
		setupCmd.StringVar(&setupConfig.GitHubToken, "github-token", "", "A GitHub token - required to prevent hitting API limits")

		// Recipe and repo flags
		var recipeReposStr string
		setupCmd.StringVar(&recipeReposStr, "recipe-repos", "", "Additional recipe repositories to add (comma-separated)")
		setupCmd.StringVar(&setupConfig.RepoListPath, "repo-list", "", "Path to a repo-list file")

		var recipeListsStr string
		setupCmd.StringVar(&recipeListsStr, "recipe-list", "", "Path to recipe list file (comma-separated for multiple)")

		// Private repo flags
		setupCmd.StringVar(&setupConfig.PrivateRepoPath, "private-repo", "", "Path to a private repo")
		setupCmd.StringVar(&setupConfig.PrivateRepoURL, "private-repo-url", "", "The private repo url")

		// JamfUploader flags
		setupCmd.StringVar(&setupConfig.JSSUrl, "jss-url", "", "URL of the Jamf server")
		setupCmd.StringVar(&setupConfig.JSSUser, "jss-user", "", "API account username")
		setupCmd.StringVar(&setupConfig.JSSPass, "jss-pass", "", "API account password")
		setupCmd.StringVar(&setupConfig.SMBUrl, "smb-url", "", "URL of the FileShare Distribution Point")
		setupCmd.StringVar(&setupConfig.SMBUser, "smb-user", "", "Username of account that has access to the DP")
		setupCmd.StringVar(&setupConfig.SMBPass, "smb-pass", "", "Password of account that has access to the DP")
		setupCmd.BoolVar(&setupConfig.UseJamfUploader, "jamf-uploader-repo", false, "Use jamf-upload repo instead of grahampugh-recipes")
		setupCmd.BoolVar(&setupConfig.JCDS2Mode, "jcds2-mode", false, "Set to JCDS2 mode")
		setupCmd.BoolVar(&setupConfig.JCDS2Mode, "j", false, "Set to JCDS2 mode (shorthand)")

		// Slack flags
		setupCmd.StringVar(&setupConfig.SlackWebhook, "slack-webhook", "", "Slack webhook")
		setupCmd.StringVar(&setupConfig.SlackUsername, "slack-user", "", "A display name for the Slack notifications")

		if err := setupCmd.Parse(os.Args[2:]); err != nil {
			fmt.Printf("❌ Error parsing setup command flags: %v\n", err)
			os.Exit(1)
		}

		// Split comma-separated inputs into slices
		if recipeReposStr != "" {
			setupConfig.RecipeRepos = strings.Split(recipeReposStr, ",")
		}

		if recipeListsStr != "" {
			setupConfig.RecipeLists = strings.Split(recipeListsStr, ",")
		}

		// Read environment variables for any unset values
		readEnvVars(&setupConfig)

		// Execute setup
		if err := executeSetup(&setupConfig, debugMode); err != nil {
			autopkg.Logger(fmt.Sprintf("Setup failed: %v", err), autopkg.LogError)
			os.Exit(1)
		}

	case "run":
		// Setup for run command (Process recipes)
		var recipeListPath string
		var recipeNames string
		var overridesDir string
		var teamsWebhook string
		var disableVerification bool
		var cleanupListPath string
		var promoteListPath string

		runCmd.StringVar(&recipeListPath, "list", "", "Path to a plist or JSON list of recipe names")
		runCmd.StringVar(&recipeListPath, "l", "", "Path to a plist or JSON list of recipe names (shorthand)")
		runCmd.StringVar(&recipeNames, "recipes", "", "Comma-separated list of recipe names to run")
		runCmd.StringVar(&recipeNames, "r", "", "Comma-separated list of recipe names to run (shorthand)")
		runCmd.StringVar(&overridesDir, "overrides-dir", "", "Directory containing recipe overrides")
		runCmd.StringVar(&teamsWebhook, "teams-webhook", "", "Microsoft Teams webhook URL for notifications")
		runCmd.BoolVar(&disableVerification, "disable-verification", false, "Disable recipe verification")
		runCmd.BoolVar(&disableVerification, "v", false, "Disable recipe verification (shorthand)")
		runCmd.StringVar(&cleanupListPath, "cleanup-list", "", "Path to JSON file with apps to run cleanup for")
		runCmd.StringVar(&cleanupListPath, "c", "", "Path to JSON file with apps to run cleanup for (shorthand)")
		runCmd.StringVar(&promoteListPath, "promote-list", "", "Path to JSON file with apps to promote")

		if err := runCmd.Parse(os.Args[2:]); err != nil {
			fmt.Printf("❌ Error parsing run command flags: %v\n", err)
			os.Exit(1)
		}

		// Check required arguments
		recipes := os.Getenv("RECIPE_TO_RUN")
		if recipes == "" {
			recipes = recipeNames
		}

		if recipes == "" && recipeListPath == "" {
			autopkg.Logger("Recipe list or recipe names not provided. Use --list or --recipes flag.", autopkg.LogError)
			runCmd.PrintDefaults()
			os.Exit(1)
		}

		// Set default overrides directory if not specified
		if overridesDir == "" {
			overridesDir = os.Getenv("OVERRIDES_DIR")
			if overridesDir == "" {
				usr, err := os.UserHomeDir()
				if err == nil {
					overridesDir = usr + "/Library/AutoPkg/RecipeOverrides"
				}
			}
		}

		// Get Teams webhook from env var if not specified
		if teamsWebhook == "" {
			teamsWebhook = os.Getenv("TEAMS_WEBHOOK")
		}

		// Set up options
		opts := &autopkg.RecipeOptions{
			DisableVerification: disableVerification,
			CleanupList:         cleanupListPath,
			PromoteList:         promoteListPath,
			Debug:               debugMode,
		}

		// Execute run
		if recipeListPath != "" {
			if err := autopkg.ProcessRecipes(recipeListPath, overridesDir, opts, teamsWebhook); err != nil {
				autopkg.Logger(fmt.Sprintf("Failed to process recipes: %v", err), autopkg.LogError)
				os.Exit(1)
			}
		} else {
			if err := autopkg.ProcessRecipes(recipes, overridesDir, opts, teamsWebhook); err != nil {
				autopkg.Logger(fmt.Sprintf("Failed to process recipes: %v", err), autopkg.LogError)
				os.Exit(1)
			}
		}

	case "verify":
		// Setup for verify command (Verify or update trust info)
		var recipeListPath string
		var recipeNames string
		var overridesDir string
		var updateTrust bool

		verifyCmd.StringVar(&recipeListPath, "list", "", "Path to a plist or JSON list of recipe names")
		verifyCmd.StringVar(&recipeListPath, "l", "", "Path to a plist or JSON list of recipe names (shorthand)")
		verifyCmd.StringVar(&recipeNames, "recipes", "", "Comma-separated list of recipe names to verify")
		verifyCmd.StringVar(&recipeNames, "r", "", "Comma-separated list of recipe names to verify (shorthand)")
		verifyCmd.StringVar(&overridesDir, "overrides-dir", "", "Directory containing recipe overrides")
		verifyCmd.BoolVar(&updateTrust, "update", false, "Update trust info for recipes that fail verification")
		verifyCmd.BoolVar(&updateTrust, "u", false, "Update trust info for recipes that fail verification (shorthand)")

		if err := verifyCmd.Parse(os.Args[2:]); err != nil {
			fmt.Printf("❌ Error parsing verify command flags: %v\n", err)
			os.Exit(1)
		}

		// Check required arguments
		recipes := os.Getenv("RECIPE_TO_RUN")
		if recipes == "" {
			recipes = recipeNames
		}

		if recipes == "" && recipeListPath == "" {
			autopkg.Logger("Recipe list or recipe names not provided. Use --list or --recipes flag.", autopkg.LogError)
			verifyCmd.PrintDefaults()
			os.Exit(1)
		}

		// Set default overrides directory if not specified
		if overridesDir == "" {
			overridesDir = os.Getenv("OVERRIDES_DIR")
			if overridesDir == "" {
				usr, err := os.UserHomeDir()
				if err == nil {
					overridesDir = usr + "/Library/AutoPkg/RecipeOverrides"
				}
			}
		}

		// Execute verify
		err := executeVerify(recipes, recipeListPath, overridesDir, updateTrust, debugMode)
		if err != nil {
			autopkg.Logger(fmt.Sprintf("Verification failed: %v", err), autopkg.LogError)
			os.Exit(1)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

// printUsage prints usage information
func printUsage() {
	fmt.Println("🧩 AutoPkg Go - A comprehensive AutoPkg automation tool")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  autopkg-go [command] [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  setup    Install and configure AutoPkg")
	fmt.Println("  run      Run AutoPkg recipes")
	fmt.Println("  verify   Verify or update trust info for recipes")
	fmt.Println("")
	fmt.Println("Global Options:")
	fmt.Println("  -debug, -d    Enable debug mode with verbose output")
	fmt.Println("")
	fmt.Println("Run 'autopkg-go [command] --help' for more information on a command")
}

// readEnvVars reads environment variables for any unset config values
func readEnvVars(config *autopkg.Config) {
	if config.GitHubToken == "" {
		config.GitHubToken = os.Getenv("GITHUB_TOKEN")
	}

	if config.JSSUrl == "" {
		config.JSSUrl = os.Getenv("JSS_URL")
	}

	if config.JSSUser == "" {
		config.JSSUser = os.Getenv("JSS_API_USER")
	}

	if config.JSSPass == "" {
		config.JSSPass = os.Getenv("JSS_API_PW")
	}

	if config.SMBUrl == "" {
		config.SMBUrl = os.Getenv("SMB_URL")
	}

	if config.SMBUser == "" {
		config.SMBUser = os.Getenv("SMB_USERNAME")
	}

	if config.SMBPass == "" {
		config.SMBPass = os.Getenv("SMB_PASSWORD")
	}

	if config.SlackWebhook == "" {
		config.SlackWebhook = os.Getenv("SLACK_WEBHOOK")
	}

	if config.SlackUsername == "" {
		config.SlackUsername = os.Getenv("SLACK_USERNAME")
	}

	// Check for debugging
	if os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "1" || os.Getenv("DEBUG") == "yes" {
		autopkg.Logger("Debug mode enabled via environment variable", autopkg.LogDebug)
	}
}

// executeSetup handles the setup command execution
func executeSetup(config *autopkg.Config, debugMode bool) error {
	// Check not running as root
	if err := autopkg.RootCheck(); err != nil {
		return err
	}

	autopkg.Logger("Starting AutoPkg setup...", autopkg.LogInfo)

	// Check for command line tools
	if err := autopkg.CheckCommandLineTools(); err != nil {
		return err
	}

	// Install or check AutoPkg
	version, err := autopkg.InstallAutoPkg(config)
	if err != nil {
		return fmt.Errorf("error installing AutoPkg: %w", err)
	}
	autopkg.Logger(fmt.Sprintf("AutoPkg version: %s", version), autopkg.LogSuccess)

	// Set up preferences file
	prefsPath, err := autopkg.SetupPreferencesFile(config)
	if err != nil {
		return fmt.Errorf("error setting up preferences: %w", err)
	}
	autopkg.Logger(fmt.Sprintf("Preferences configured at: %s", prefsPath), autopkg.LogSuccess)

	// Configure Slack integration
	if config.SlackWebhook != "" || config.SlackUsername != "" {
		if err := autopkg.ConfigureSlack(config, prefsPath); err != nil {
			return fmt.Errorf("error configuring Slack: %w", err)
		}
		autopkg.Logger("Slack integration configured", autopkg.LogSuccess)
	}

	// Set up private repo if configured
	if config.PrivateRepoPath != "" && config.PrivateRepoURL != "" {
		if err := autopkg.SetupPrivateRepo(config, prefsPath); err != nil {
			return fmt.Errorf("error setting up private repo: %w", err)
		}
		autopkg.Logger(fmt.Sprintf("Private repo configured: %s", config.PrivateRepoPath), autopkg.LogSuccess)
	}

	// Configure JamfUploader if JSS URL is provided
	if config.JSSUrl != "" {
		if err := autopkg.ConfigureJamfUploader(config, prefsPath); err != nil {
			return fmt.Errorf("error configuring JamfUploader: %w", err)
		}
		autopkg.Logger("JamfUploader configured", autopkg.LogSuccess)
	}

	// Add AutoPkg repositories
	if err := autopkg.AddAutoPkgRepos(config, prefsPath); err != nil {
		return fmt.Errorf("error adding AutoPkg repos: %w", err)
	}
	autopkg.Logger("AutoPkg repositories configured", autopkg.LogSuccess)

	// Process recipe lists if provided
	if len(config.RecipeLists) > 0 {
		for _, listPath := range config.RecipeLists {
			autopkg.Logger(fmt.Sprintf("Processing recipe list: %s", listPath), autopkg.LogInfo)
		}

		if err := autopkg.ProcessRecipeLists(config, prefsPath); err != nil {
			return fmt.Errorf("error processing recipe lists: %w", err)
		}
		autopkg.Logger("Recipe lists processed", autopkg.LogSuccess)
	}

	autopkg.Logger("🎉 AutoPkg setup completed successfully!", autopkg.LogSuccess)
	return nil
}

// executeVerify handles the verify command execution
func executeVerify(recipes string, recipeListPath string, overridesDir string, updateTrust bool, debugMode bool) error {
	autopkg.Logger("Starting recipe verification...", autopkg.LogInfo)

	var recipesToVerify []*autopkg.Recipe
	var err error

	// Parse recipes from list or comma-separated string
	if recipeListPath != "" {
		recipesToVerify, err = autopkg.ParseRecipes(recipeListPath, overridesDir)
	} else {
		recipesToVerify, err = autopkg.ParseRecipes(recipes, overridesDir)
	}

	if err != nil {
		return fmt.Errorf("failed to parse recipes: %w", err)
	}

	failures := 0
	successes := 0

	// Verify each recipe
	for _, recipe := range recipesToVerify {
		name, _ := recipe.Name()
		autopkg.Logger(fmt.Sprintf("Verifying recipe: %s", name), autopkg.LogInfo)

		verified, err := recipe.VerifyTrustInfo(debugMode)
		if err != nil || !verified {
			failures++
			if updateTrust {
				autopkg.Logger(fmt.Sprintf("Updating trust info for recipe: %s", name), autopkg.LogInfo)
				if err := recipe.UpdateTrustInfo(debugMode); err != nil {
					autopkg.Logger(fmt.Sprintf("Failed to update trust info: %v", err), autopkg.LogError)
				} else {
					autopkg.Logger(fmt.Sprintf("Trust info updated for recipe: %s", name), autopkg.LogSuccess)
				}
			}
		} else {
			successes++
			autopkg.Logger(fmt.Sprintf("Recipe verified successfully: %s", name), autopkg.LogSuccess)
		}
	}

	// Report summary
	total := len(recipesToVerify)
	autopkg.Logger(fmt.Sprintf("Verification complete: %d total, %d successes, %d failures",
		total, successes, failures), autopkg.LogInfo)

	if failures > 0 {
		return fmt.Errorf("%d recipes failed verification", failures)
	}

	return nil
}
