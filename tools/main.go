package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	
	"github.com/yourusername/autopkg"
)

func main() {
	// Parse command line arguments
	var config autopkg.Config
	
	// Basic AutoPkg flags
	flag.BoolVar(&config.ForceUpdate, "force", false, "Force the re-installation of the latest AutoPkg")
	flag.BoolVar(&config.ForceUpdate, "f", false, "Force the re-installation of the latest AutoPkg (shorthand)")
	flag.BoolVar(&config.UseBeta, "beta", false, "Force the installation of the pre-released version of AutoPkg")
	flag.BoolVar(&config.UseBeta, "b", false, "Force the installation of the pre-released version of AutoPkg (shorthand)")
	flag.BoolVar(&config.FailRecipes, "fail", true, "Fail runs if not verified")
	flag.StringVar(&config.PrefsFilePath, "prefs", "", "Path to the preferences plist")
	flag.BoolVar(&config.ReplacePrefs, "replace-prefs", false, "Delete the prefs file and rebuild from scratch")
	flag.StringVar(&config.GitHubToken, "github-token", "", "A GitHub token - required to prevent hitting API limits")
	
	// Recipe and repo flags
	var recipeReposStr string
	flag.StringVar(&recipeReposStr, "recipe-repos", "", "Additional recipe repositories to add (comma-separated)")
	flag.StringVar(&config.RepoListPath, "repo-list", "", "Path to a repo-list file")
	
	var recipeListsStr string
	flag.StringVar(&recipeListsStr, "recipe-list", "", "Path to recipe list file (comma-separated for multiple)")
	
	// Private repo flags
	flag.StringVar(&config.PrivateRepoPath, "private-repo", "", "Path to a private repo")
	flag.StringVar(&config.PrivateRepoURL, "private-repo-url", "", "The private repo url")
	
	// JamfUploader flags
	flag.StringVar(&config.JSSUrl, "jss-url", "", "URL of the Jamf server")
	flag.StringVar(&config.JSSUser, "jss-user", "", "API account username")
	flag.StringVar(&config.JSSPass, "jss-pass", "", "API account password")
	flag.StringVar(&config.SMBUrl, "smb-url", "", "URL of the FileShare Distribution Point")
	flag.StringVar(&config.SMBUser, "smb-user", "", "Username of account that has access to the DP")
	flag.StringVar(&config.SMBPass, "smb-pass", "", "Password of account that has access to the DP")
	flag.BoolVar(&config.UseJamfUploader, "jamf-uploader-repo", false, "Use jamf-upload repo instead of grahampugh-recipes")
	flag.BoolVar(&config.JCDS2Mode, "jcds2-mode", false, "Set to JCDS2 mode")
	flag.BoolVar(&config.JCDS2Mode, "j", false, "Set to JCDS2 mode (shorthand)")
	
	// Slack flags
	flag.StringVar(&config.SlackWebhook, "slack-webhook", "", "Slack webhook")
	flag.StringVar(&config.SlackUsername, "slack-user", "", "A display name for the Slack notifications")
	
	flag.Parse()
	
	// Split comma-separated inputs into slices
	if recipeReposStr != "" {
		config.RecipeRepos = strings.Split(recipeReposStr, ",")
	}
	
	if recipeListsStr != "" {
		config.RecipeLists = strings.Split(recipeListsStr, ",")
	}
	
	// Check not running as root
	if err := autopkg.RootCheck(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(4)
	}
	
	// Check for command line tools
	if err := autopkg.CheckCommandLineTools(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	
	// Install or check AutoPkg
	version, err := autopkg.InstallAutoPkg(&config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing AutoPkg: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("AutoPkg version: %s\n", version)
	
	// Set up preferences file
	prefsPath, err := autopkg.SetupPreferencesFile(&config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up preferences: %v\n", err)
		os.Exit(1)
	}
	
	// Configure Slack integration
	if err := autopkg.ConfigureSlack(&config, prefsPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error configuring Slack: %v\n", err)
		os.Exit(1)
	}
	
	// Set up private repo if configured
	if config.PrivateRepoPath != "" && config.PrivateRepoURL != "" {
		if err := autopkg.SetupPrivateRepo(&config, prefsPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting up private repo: %v\n", err)
			os.Exit(1)
		}
	}
	
	// Configure JamfUploader if JSS URL is provided
	if config.JSSUrl != "" {
		if err := autopkg.ConfigureJamfUploader(&config, prefsPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error configuring JamfUploader: %v\n", err)
			os.Exit(1)
		}
	}
	
	// Add AutoPkg repositories
	if err := autopkg.AddAutoPkgRepos(&config, prefsPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding AutoPkg repos: %v\n", err)
		os.Exit(1)
	}
	
	// Process recipe lists if provided
	if len(config.RecipeLists) > 0 {
		if err := autopkg.ProcessRecipeLists(&config, prefsPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error processing recipe lists: %v\n", err)
			os.Exit(1)
		}
	}
	
	fmt.Println("AutoPkg setup completed successfully.")
}
