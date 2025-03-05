package autopkg

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

// Config holds all the configuration options for AutoPkg setup operations
type Config struct {
	// Basic AutoPkg settings
	ForceUpdate         bool
	UseBeta             bool
	FailRecipes         bool
	DisableVerification bool
	PrefsFilePath       string
	ReplacePrefs        bool
	GitHubToken         string

	// Recipe and repo settings
	RecipeRepos         []string
	RecipeLists         []string
	AutopkgRepoListPath string

	// Private repo settings
	PrivateRepoPath string
	PrivateRepoURL  string

	// JamfUploader settings
	JAMFPRO_URL           string
	API_USERNAME          string
	API_PASSWORD          string
	JAMFPRO_CLIENT_ID     string
	JAMFPRO_CLIENT_SECRET string
	SMB_URL               string
	SMB_USERNAME          string
	SMB_PASSWORD          string
	UseJamfUploader       bool
	JCDS2Mode             bool

	// JamfUploader / IntuneUploader settings
	INTUNE_CLIENT_ID     string
	INTUNE_CLIENT_SECRET string
	INTUNE_TENANT_ID     string

	// Slack settings
	SlackWebhook  string
	SlackUsername string
}

// RootCheck ensures the script is not running as root
func RootCheck() error {
	if os.Geteuid() == 0 {
		return fmt.Errorf("this script is NOT MEANT to run as root; please run without sudo")
	}
	return nil
}

// CheckCommandLineTools verifies git is installed, and installs it if needed
func CheckCommandLineTools() error {
	// Check if git exists
	gitCmd := exec.Command("git", "--version")
	if err := gitCmd.Run(); err == nil {
		Logger("✅ Git is installed and functional", LogSuccess)
		return nil
	}

	// Git isn't working, so install it
	Logger("🔧 Git not found, installing...", LogInfo)
	return installGit()
}

// installGit installs git using the most direct method available
func installGit() error {
	// First check if Homebrew is available
	brewCmd := exec.Command("which", "brew")
	if err := brewCmd.Run(); err == nil {
		// Use Homebrew to install git
		Logger("🔄 Installing git via Homebrew...", LogInfo)
		brewInstall := exec.Command("brew", "install", "git")
		brewInstall.Stdout = os.Stdout
		brewInstall.Stderr = os.Stderr
		if err := brewInstall.Run(); err != nil {
			return fmt.Errorf("failed to install git via Homebrew: %w", err)
		}
	} else {
		// Fall back to Xcode Command Line Tools if Homebrew isn't available
		Logger("🔄 Installing git via Xcode Command Line Tools...", LogInfo)
		cmd := exec.Command("xcode-select", "--install")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install Xcode Command Line Tools: %w", err)
		}
	}

	// Verify installation worked
	gitCmd := exec.Command("git", "--version")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("git still not available after installation attempt: %w", err)
	}

	Logger("✅ Git successfully installed", LogSuccess)
	return nil
}

// InstallAutoPkg downloads and installs the latest AutoPkg release
func InstallAutoPkg(config *Config) (string, error) {
	autopkgPath := "/usr/local/bin/autopkg"

	// Only install if not present or forced update requested
	if _, err := os.Stat(autopkgPath); !os.IsNotExist(err) && !config.ForceUpdate {
		// Get current version
		versionCmd := exec.Command(autopkgPath, "version")
		versionOutput, err := versionCmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get AutoPkg version: %w", err)
		}
		version := strings.TrimSpace(string(versionOutput))
		return version, nil
	}

	Logger("⬇️ Downloading AutoPkg", LogInfo)

	var releaseURL string
	var err error

	// Get release URL based on config
	if config.UseBeta {
		releaseURL, err = getBetaAutoPkgReleaseURL()
		Logger("🧪 Installing latest Beta AutoPkg Release", LogInfo)
	} else {
		releaseURL, err = getLatestAutoPkgReleaseURL()
		Logger("🚀 Installing latest Stable AutoPkg Release", LogInfo)
	}

	if err != nil {
		return "", fmt.Errorf("failed to get AutoPkg release URL: %w", err)
	}
	fmt.Printf("AutoPkg release URL: %s\n", releaseURL)

	// Download the package
	pkgPath := "/tmp/autopkg-latest.pkg"
	if err := downloadFile(releaseURL, pkgPath); err != nil {
		return "", fmt.Errorf("failed to download AutoPkg package: %w", err)
	}

	// Install the package
	cmd := exec.Command("sudo", "installer", "-pkg", pkgPath, "-target", "/")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to install AutoPkg package: %w", err)
	}

	// Verify installation and capture version
	versionCmd := exec.Command(autopkgPath, "version")
	versionOutput, err := versionCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get AutoPkg version: %w", err)
	}

	version := strings.TrimSpace(string(versionOutput))
	Logger(fmt.Sprintf("✅ AutoPkg %s Installed", version), LogSuccess)

	return version, nil
}

// SetupPreferencesFile initializes or updates the AutoPkg preferences file
func SetupPreferencesFile(config *Config) (string, error) {
	// Determine preferences file location
	if config.PrefsFilePath == "" {
		currentUser, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("failed to get current user: %w", err)
		}
		config.PrefsFilePath = filepath.Join(currentUser.HomeDir, "Library/Preferences/com.github.autopkg.plist")
	}

	// Remove existing prefs if requested
	if config.ReplacePrefs {
		if err := os.Remove(config.PrefsFilePath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Warning: failed to remove existing preferences file: %v\n", err)
		}
	}

	// Set up Git path
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return "", fmt.Errorf("git executable not found: %w", err)
	}

	cmd := exec.Command("defaults", "write", config.PrefsFilePath, "GIT_PATH", gitPath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to set GIT_PATH: %w", err)
	}
	Logger(fmt.Sprintf("📝 Added GIT_PATH %s to %s", gitPath, config.PrefsFilePath), LogInfo)

	// Set up GitHub token
	if config.GitHubToken != "" {
		currentUser, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("failed to get current user: %w", err)
		}

		tokenPath := filepath.Join(currentUser.HomeDir, "Library/AutoPkg/gh_token")

		// Create directory if it doesn't exist
		tokenDir := filepath.Dir(tokenPath)
		if err := os.MkdirAll(tokenDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create token directory: %w", err)
		}

		// Write the token
		if err := os.WriteFile(tokenPath, []byte(config.GitHubToken), 0600); err != nil {
			return "", fmt.Errorf("failed to write GitHub token: %w", err)
		}
		Logger(fmt.Sprintf("📝 Added GITHUB_TOKEN to %s", tokenPath), LogInfo)

		// Set the token path in preferences
		cmd := exec.Command("defaults", "write", config.PrefsFilePath, "GITHUB_TOKEN_PATH", tokenPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to set GITHUB_TOKEN_PATH: %w", err)
		}
		Logger(fmt.Sprintf("📝 Added GITHUB_TOKEN_PATH to %s", config.PrefsFilePath), LogInfo)
	}

	// Configure FAIL_RECIPES_WITHOUT_TRUST_INFO
	failValue := "true"
	if !config.FailRecipes {
		failValue = "false"
	}

	cmd = exec.Command("defaults", "write", config.PrefsFilePath, "FAIL_RECIPES_WITHOUT_TRUST_INFO", "-bool", failValue)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to set FAIL_RECIPES_WITHOUT_TRUST_INFO: %w", err)
	}
	fmt.Printf("Wrote FAIL_RECIPES_WITHOUT_TRUST_INFO %s to %s\n", failValue, config.PrefsFilePath)

	// Configure JCDS2 mode
	if config.JCDS2Mode {
		cmd = exec.Command("defaults", "write", config.PrefsFilePath, "jcds2_mode", "-bool", "true")
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to set jcds2_mode: %w", err)
		}
		fmt.Printf("Wrote jcds2_mode true to %s\n", config.PrefsFilePath)
	} else {
		// Check if jcds2_mode exists and delete it if it does
		checkCmd := exec.Command("defaults", "read", config.PrefsFilePath, "jcds2_mode")
		if checkCmd.Run() == nil {
			cmd = exec.Command("defaults", "delete", config.PrefsFilePath, "jcds2_mode")
			_ = cmd.Run()
		}
	}

	// Ensure RECIPE_SEARCH_DIRS exists
	cmd = exec.Command("defaults", "read", config.PrefsFilePath, "RECIPE_SEARCH_DIRS")
	if cmd.Run() != nil {
		cmd = exec.Command("defaults", "write", config.PrefsFilePath, "RECIPE_SEARCH_DIRS", "-array")
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to create RECIPE_SEARCH_DIRS: %w", err)
		}
	}

	// Ensure RECIPE_REPOS exists
	cmd = exec.Command("defaults", "read", config.PrefsFilePath, "RECIPE_REPOS")
	if cmd.Run() != nil {
		cmd = exec.Command("defaults", "write", config.PrefsFilePath, "RECIPE_REPOS", "-dict")
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to create RECIPE_REPOS: %w", err)
		}
	}

	return config.PrefsFilePath, nil
}

// ConfigureSlack sets up Slack integration
func ConfigureSlack(config *Config, prefsPath string) error {
	if config.SlackUsername == "" && config.SlackWebhook == "" {
		return nil
	}

	if config.SlackUsername != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "SLACK_USERNAME", config.SlackUsername)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set SLACK_USERNAME: %w", err)
		}
		fmt.Printf("Wrote SLACK_USERNAME %s to %s\n", config.SlackUsername, prefsPath)
	}

	if config.SlackWebhook != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "SLACK_WEBHOOK", config.SlackWebhook)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set SLACK_WEBHOOK: %w", err)
		}
		fmt.Printf("Wrote SLACK_WEBHOOK %s to %s\n", config.SlackWebhook, prefsPath)
	}

	return nil
}

// SetupPrivateRepo adds a private AutoPkg repo
func SetupPrivateRepo(config *Config, prefsPath string) error {
	if config.PrivateRepoPath == "" || config.PrivateRepoURL == "" {
		return nil
	}

	// Clone the repo if it doesn't exist
	if _, err := os.Stat(config.PrivateRepoPath); os.IsNotExist(err) {
		cmd := exec.Command("git", "clone", config.PrivateRepoURL, config.PrivateRepoPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to clone private repo: %w", err)
		}
	}

	// Check if RECIPE_REPOS exists in prefs
	cmd := exec.Command("/usr/libexec/PlistBuddy", "-c", "Print :RECIPE_REPOS", prefsPath)
	if err := cmd.Run(); err != nil {
		// Need to create it
		cmd := exec.Command("/usr/libexec/PlistBuddy", "-c", "Add :RECIPE_REPOS dict", prefsPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create RECIPE_REPOS: %w", err)
		}
	}

	// Check if the private repo is already in RECIPE_REPOS
	cmd = exec.Command("/usr/libexec/PlistBuddy", "-c", fmt.Sprintf("Print :RECIPE_REPOS:%s", config.PrivateRepoPath), prefsPath)
	if err := cmd.Run(); err != nil {
		// Need to add it
		cmd := exec.Command("/usr/libexec/PlistBuddy", "-c", fmt.Sprintf("Add :RECIPE_REPOS:%s dict", config.PrivateRepoPath), prefsPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add private repo to RECIPE_REPOS: %w", err)
		}

		cmd = exec.Command("/usr/libexec/PlistBuddy", "-c", fmt.Sprintf("Add :RECIPE_REPOS:%s:URL string %s", config.PrivateRepoPath, config.PrivateRepoURL), prefsPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add private repo URL: %w", err)
		}
	}

	// Check if RECIPE_SEARCH_DIRS exists
	cmd = exec.Command("/usr/libexec/PlistBuddy", "-c", "Print :RECIPE_SEARCH_DIRS", prefsPath)
	if err := cmd.Run(); err != nil {
		// Need to create it
		cmd := exec.Command("/usr/libexec/PlistBuddy", "-c", "Add :RECIPE_SEARCH_DIRS array", prefsPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create RECIPE_SEARCH_DIRS: %w", err)
		}
	}

	// Get current RECIPE_SEARCH_DIRS to check if private repo is already there
	cmd = exec.Command("/usr/libexec/PlistBuddy", "-c", "Print :RECIPE_SEARCH_DIRS", prefsPath)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to read RECIPE_SEARCH_DIRS: %w", err)
	}

	// Check if private repo is already in RECIPE_SEARCH_DIRS
	if !strings.Contains(string(output), config.PrivateRepoPath) {
		cmd := exec.Command("/usr/libexec/PlistBuddy", "-c", fmt.Sprintf("Add :RECIPE_SEARCH_DIRS: string '%s'", config.PrivateRepoPath), prefsPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add private repo to RECIPE_SEARCH_DIRS: %w", err)
		}
	}

	Logger("✅ Private AutoPkg Repo Configured", LogSuccess)
	return nil
}

// AddAutoPkgRepos adds required and additional AutoPkg repositories
func AddAutoPkgRepos(config *Config, prefsPath string) error {
	// Start with the default repositories based on mdm uploader type
	var repos []string
	if USE_JAMF_UPLOADER {
		repos = []string{"recipes", "grahampugh/jamf-upload"}
	} else if USE_INTUNE_UPLOADER {
		repos = []string{"recipes", "almenscorner/autopkg-recipes"}
	} else {
		repos = []string{"recipes"}
	}

	// Load additional repos from repo list file if specified
	if config.AutopkgRepoListPath != "" && fileExists(config.AutopkgRepoListPath) {
		file, err := os.Open(config.AutopkgRepoListPath)
		if err != nil {
			return fmt.Errorf("failed to open repo list file: %w", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			repo := strings.TrimSpace(scanner.Text())
			if repo != "" {
				repos = append(repos, repo)
			}
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to read repo list file: %w", err)
		}
	}
	config.RecipeRepos = repos

	// Add all specified repositories using the autopkg command
	for _, repo := range repos {
		if repo == "" {
			continue
		}
		Logger(fmt.Sprintf("📦 Adding recipe repository: %s", repo), LogInfo)
		cmd := exec.Command("autopkg", "repo-add", repo, "--prefs", prefsPath)

		// Log error if repo-add fails, but continue with other repos
		if err := cmd.Run(); err != nil {
			fmt.Printf("ERROR: could not add %s to %s\n", repo, prefsPath)
		} else {
			Logger(fmt.Sprintf("✅ Added %s to %s", repo, prefsPath), LogInfo)

		}
	}

	Logger("✅ AutoPkg Repos Configured", LogSuccess)
	return nil
}

// ProcessRecipeLists processes any specified recipe lists to ensure parent recipe repos are added
func ProcessRecipeLists(config *Config, prefsPath string) error {
	if len(config.RecipeLists) == 0 {
		return nil
	}

	for _, listPath := range config.RecipeLists {
		if !fileExists(listPath) {
			fmt.Printf("Warning: Recipe list file %s does not exist\n", listPath)
			continue
		}

		file, err := os.Open(listPath)
		if err != nil {
			return fmt.Errorf("failed to open recipe list file: %w", err)
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			recipe := strings.TrimSpace(scanner.Text())
			if recipe == "" {
				continue
			}

			// Use autopkg info to get parent recipe info, which will add required repos
			cmd := exec.Command("autopkg", "info", "-p", recipe, "--prefs", prefsPath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			_ = cmd.Run() // We just run this to add parent repos, don't care about the output
		}

		file.Close()

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to read recipe list file: %w", err)
		}
	}

	Logger("✅ AutoPkg Repos for all parent recipes added", LogSuccess)
	return nil
}

// ListRecipes lists all available AutoPkg recipes
func ListRecipes(prefsPath string) error {
	Logger("📝 Available recipes:", LogInfo)

	args := []string{"list-recipes"}
	if prefsPath != "" {
		args = append(args, "--prefs", prefsPath)
	}

	cmd := exec.Command("autopkg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// RunRecipe runs a specified AutoPkg recipe
func RunRecipe(recipeName, prefsPath string) error {
	if recipeName == "" {
		return nil
	}

	fmt.Printf("Running recipe: %s\n", recipeName)

	args := []string{"run", recipeName}
	if prefsPath != "" {
		args = append(args, "--prefs", prefsPath)
	}

	cmd := exec.Command("autopkg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Helper functions

// getLatestAutoPkgReleaseURL retrieves the URL of the latest AutoPkg release
func getLatestAutoPkgReleaseURL() (string, error) {
	// Create a new request
	req, err := http.NewRequest("GET", "https://api.github.com/repos/autopkg/autopkg/releases/latest", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add GitHub token for authentication if available
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken != "" {
		req.Header.Set("Authorization", "token "+githubToken)
	}

	// Add headers to identify our client
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "AutoPkgGitHubActions/1.0")

	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	// Check for rate limiting or other errors
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Log raw response for debugging
	if DEBUG {
		body, _ := io.ReadAll(resp.Body)
		Logger(fmt.Sprintf("GitHub API response: %s", string(body)), LogDebug)

		// Create a new reader with the same data for subsequent decoding
		resp.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	// Parse the response
	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	// Find the .pkg asset
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, ".pkg") {
			Logger(fmt.Sprintf("🔍 Found release %s with package %s", release.TagName, asset.Name), LogInfo)
			return asset.BrowserDownloadURL, nil
		}
	}

	// If we get here, no package was found
	return "", fmt.Errorf("no pkg asset found in the latest release (tag: %s, assets count: %d)",
		release.TagName, len(release.Assets))
}

// getBetaAutoPkgReleaseURL retrieves the URL of the latest beta AutoPkg release
func getBetaAutoPkgReleaseURL() (string, error) {
	// Create a new request to get all releases including pre-releases
	req, err := http.NewRequest("GET", "https://api.github.com/repos/autopkg/autopkg/releases", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add GitHub token for authentication if available
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken != "" {
		req.Header.Set("Authorization", "token "+githubToken)
	}

	// Add headers to identify our client
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "AutoPkgGitHubActions/1.0")

	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	// Check for rate limiting or other errors
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var releases []struct {
		TagName    string `json:"tag_name"`
		Prerelease bool   `json:"prerelease"`
		Assets     []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	// Find the latest beta/prerelease
	for _, release := range releases {
		// Check if this is a prerelease
		if release.Prerelease {
			// Look for .pkg asset in this prerelease
			for _, asset := range release.Assets {
				if strings.HasSuffix(asset.Name, ".pkg") {
					Logger(fmt.Sprintf("🔍 Found beta release: %s", release.TagName), LogInfo)
					return asset.BrowserDownloadURL, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no beta release with pkg asset found")
}
