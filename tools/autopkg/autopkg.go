package autopkg

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

// Config holds all the configuration options for AutoPkg setup operations
type Config struct {
	// Basic AutoPkg settings
	ForceUpdate   bool
	UseBeta       bool
	FailRecipes   bool
	PrefsFilePath string
	ReplacePrefs  bool
	GitHubToken   string

	// Recipe and repo settings
	RecipeRepos  []string
	RecipeLists  []string
	RepoListPath string

	// Private repo settings
	PrivateRepoPath string
	PrivateRepoURL  string

	// JamfUploader settings
	JSS_URL      string
	API_USERNAME string
	API_PASSWORD string

	SMB_URL         string
	SMB_USERNAME    string
	SMB_PASSWORD    string
	UseJamfUploader bool
	JCDS2Mode       bool

	// JamfUploader / IntuneUploader settings
	CLIENT_ID     string
	CLIENT_SECRET string
	TENANT_ID     string

	// IntuneUploader settings

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
		Logger("Git is installed and functional")
		return nil
	}

	// Git isn't working, so install it
	Logger("Git not found, installing...")
	return installGit()
}

// installGit installs git using the most direct method available
func installGit() error {
	// First check if Homebrew is available
	brewCmd := exec.Command("which", "brew")
	if err := brewCmd.Run(); err == nil {
		// Use Homebrew to install git
		Logger("Installing git via Homebrew...")
		brewInstall := exec.Command("brew", "install", "git")
		brewInstall.Stdout = os.Stdout
		brewInstall.Stderr = os.Stderr
		if err := brewInstall.Run(); err != nil {
			return fmt.Errorf("failed to install git via Homebrew: %w", err)
		}
	} else {
		// Fall back to Xcode Command Line Tools if Homebrew isn't available
		Logger("Installing git via Xcode Command Line Tools...")
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

	Logger("Git successfully installed")
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

	fmt.Println("Downloading latest AutoPkg release...")

	var releaseURL string
	var err error

	// Get release URL based on config
	if config.UseBeta {
		releaseURL, err = getBetaAutoPkgReleaseURL()
	} else {
		releaseURL, err = getLatestAutoPkgReleaseURL()
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
	Logger(fmt.Sprintf("AutoPkg %s Installed", version))

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
	fmt.Printf("Wrote GIT_PATH %s to %s\n", gitPath, config.PrefsFilePath)

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
		fmt.Printf("Wrote GITHUB_TOKEN to %s\n", tokenPath)

		// Set the token path in preferences
		cmd := exec.Command("defaults", "write", config.PrefsFilePath, "GITHUB_TOKEN_PATH", tokenPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to set GITHUB_TOKEN_PATH: %w", err)
		}
		fmt.Printf("Wrote GITHUB_TOKEN_PATH to %s\n", config.PrefsFilePath)
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

	Logger("Private AutoPkg Repo Configured")
	return nil
}

// ConfigureJamfUploader sets up JamfUploader settings
func ConfigureJamfUploader(config *Config, prefsPath string) error {
	// Check for JSS_URL from config or environment
	jssURL := config.JSS_URL
	if jssURL == "" {
		jssURL = os.Getenv("JSS_URL")
	}

	// Only proceed if JSS_URL is set
	if jssURL == "" {
		return nil
	}

	// Set JSS_URL
	cmd := exec.Command("defaults", "write", prefsPath, "JSS_URL", jssURL)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set JSS_URL: %w", err)
	}

	// Set API_USERNAME if provided in config or environment
	apiUsername := config.API_USERNAME
	if apiUsername == "" {
		apiUsername = os.Getenv("API_USERNAME")
	}
	if apiUsername != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "API_USERNAME", apiUsername)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set API_USERNAME: %w", err)
		}
	}

	// Set API_PASSWORD if provided in config or environment
	apiPassword := config.API_PASSWORD
	if apiPassword == "" {
		apiPassword = os.Getenv("API_PASSWORD")
	}
	if apiPassword != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "API_PASSWORD", apiPassword)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set API_PASSWORD: %w", err)
		}
	}

	// Set CLIENT_ID if provided in config or environment
	clientID := config.CLIENT_ID
	if clientID == "" {
		clientID = os.Getenv("CLIENT_ID")
	}
	if clientID != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "CLIENT_ID", clientID)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set CLIENT_ID: %w", err)
		}
	}

	// Set CLIENT_SECRET if provided in config or environment
	clientSecret := config.CLIENT_SECRET
	if clientSecret == "" {
		clientSecret = os.Getenv("CLIENT_SECRET")
	}
	if clientSecret != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "CLIENT_SECRET", clientSecret)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set CLIENT_SECRET: %w", err)
		}
	}

	// Set SMB_URL if provided in config or environment
	smbURL := config.SMB_URL
	if smbURL == "" {
		smbURL = os.Getenv("SMB_URL")
	}
	if smbURL != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "SMB_URL", smbURL)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set SMB_URL: %w", err)
		}
	}

	// Set SMB_USERNAME if provided in config or environment
	smbUsername := config.SMB_USERNAME
	if smbUsername == "" {
		smbUsername = os.Getenv("SMB_USERNAME")
	}
	if smbUsername != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "SMB_USERNAME", smbUsername)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set SMB_USERNAME: %w", err)
		}
	}

	// Set SMB_PASSWORD if provided in config or environment
	smbPassword := config.SMB_PASSWORD
	if smbPassword == "" {
		smbPassword = os.Getenv("SMB_PASSWORD")
	}
	if smbPassword != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "SMB_PASSWORD", smbPassword)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set SMB_PASSWORD: %w", err)
		}
	}

	Logger("JamfUploader configured.")
	return nil
}

// ConfigureIntuneUploader configures the Microsoft Intune integration settings
func ConfigureIntuneUploader(config *Config, prefsPath string) error {
	// Set CLIENT_ID if provided in config or environment
	clientID := config.CLIENT_ID
	if clientID == "" {
		clientID = os.Getenv("CLIENT_ID")
	}
	if clientID != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "CLIENT_ID", clientID)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set CLIENT_ID: %w", err)
		}
		Logger(fmt.Sprintf("Set CLIENT_ID in %s", prefsPath))
	}

	// Set CLIENT_SECRET if provided in config or environment
	clientSecret := config.CLIENT_SECRET
	if clientSecret == "" {
		clientSecret = os.Getenv("CLIENT_SECRET")
	}
	if clientSecret != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "CLIENT_SECRET", clientSecret)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set CLIENT_SECRET: %w", err)
		}
		Logger(fmt.Sprintf("Set CLIENT_SECRET in %s", prefsPath))
	}

	// Set TENANT_ID if provided in config or environment
	tenantID := config.TENANT_ID
	if tenantID == "" {
		tenantID = os.Getenv("TENANT_ID")
	}
	if tenantID != "" {
		cmd := exec.Command("defaults", "write", prefsPath, "TENANT_ID", tenantID)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set TENANT_ID: %w", err)
		}
		Logger(fmt.Sprintf("Set TENANT_ID in %s", prefsPath))
	}

	// Check if we set at least some of the Intune configuration
	if clientID != "" || clientSecret != "" || tenantID != "" {
		Logger("Intune integration configured.")
	}

	return nil
}

// AddAutoPkgRepos adds required and additional AutoPkg repositories
func AddAutoPkgRepos(config *Config, prefsPath string) error {
	var repos []string

	// Determine which JamfUploader repo to use
	if config.UseJamfUploader {
		// Check if grahampugh-recipes exists and delete it
		repoOutput, err := exec.Command("autopkg", "list-repos", "--prefs", prefsPath).Output()
		if err == nil && strings.Contains(string(repoOutput), "grahampugh-recipes") {
			cmd := exec.Command("autopkg", "repo-delete", "grahampugh-recipes", "--prefs", prefsPath)
			_ = cmd.Run()
		}

		repos = append(repos, "grahampugh/jamf-upload")
	} else {
		// Check if grahampugh/jamf-upload exists and delete it
		repoOutput, err := exec.Command("autopkg", "list-repos", "--prefs", prefsPath).Output()
		if err == nil && strings.Contains(string(repoOutput), "grahampugh/jamf-upload") {
			cmd := exec.Command("autopkg", "repo-delete", "grahampugh/jamf-upload", "--prefs", prefsPath)
			_ = cmd.Run()
		}

		repos = append(repos, "grahampugh-recipes")
	}

	// Add repos from repo list file if specified
	if config.RepoListPath != "" && fileExists(config.RepoListPath) {
		file, err := os.Open(config.RepoListPath)
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

	// Add all specified repositories
	for _, repo := range repos {
		if repo == "" {
			continue
		}

		fmt.Printf("Adding recipe repository: %s\n", repo)
		cmd := exec.Command("autopkg", "repo-add", repo, "--prefs", prefsPath)

		// Just log an error but continue with other repos
		if err := cmd.Run(); err != nil {
			fmt.Printf("ERROR: could not add %s to %s\n", repo, prefsPath)
		} else {
			fmt.Printf("Added %s to %s\n", repo, prefsPath)
		}
	}

	Logger("AutoPkg Repos Configured")
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

	Logger("AutoPkg Repos for all parent recipes added")
	return nil
}

// Helper functions

// fileExists checks if a file exists
func fileExists(filepath string) bool {
	info, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// readJSONFile reads a JSON file into a map
func readJSONFile(filepath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// writeJSONFile writes a map to a JSON file
func writeJSONFile(filepath string, data map[string]interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, jsonData, 0644)
}

// updateJSONFile updates a specific key in a JSON file
func updateJSONFile(filepath string, key string, value interface{}) error {
	// Read the current JSON
	data, err := readJSONFile(filepath)
	if err != nil {
		// If the file doesn't exist or can't be parsed, create a new map
		if os.IsNotExist(err) || err.Error() == "unexpected end of JSON input" {
			data = make(map[string]interface{})
		} else {
			return err
		}
	}

	// Update the key
	data[key] = value

	// Write the updated JSON back to the file
	return writeJSONFile(filepath, data)
}

// getBetaAutoPkgReleaseURL retrieves the URL of the beta AutoPkg release
func getBetaAutoPkgReleaseURL() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/autopkg/autopkg/releases/tags/v3.0.0RC1")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var release struct {
		Assets []struct {
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.BrowserDownloadURL, ".pkg") {
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("no pkg asset found in the beta release")
}

// ListRecipes lists all available AutoPkg recipes
func ListRecipes(prefsPath string) error {
	fmt.Println("Available recipes:")

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
	resp, err := http.Get("https://api.github.com/repos/autopkg/autopkg/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var release struct {
		Assets []struct {
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.BrowserDownloadURL, ".pkg") {
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("no pkg asset found in the latest release")
}

// downloadFile downloads a file from the given URL to the specified path
func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// createPlistPrefs creates a new plist preferences file
func createPlistPrefs(filepath, munkiRepoPath string) error {
	commands := [][]string{
		{"Add", ":MUNKI_REPO", "string", munkiRepoPath},
		{"Add", ":CACHE_DIR", "string", "~/Library/AutoPkg/Cache"},
		{"Add", ":RECIPE_SEARCH_DIRS", "array"},
		{"Add", ":RECIPE_SEARCH_DIRS:0", "string", "."},
		{"Add", ":RECIPE_SEARCH_DIRS:1", "string", "~/Library/AutoPkg/Recipes"},
		{"Add", ":RECIPE_SEARCH_DIRS:2", "string", "/Library/AutoPkg/Recipes"},
		{"Add", ":RECIPE_OVERRIDE_DIRS", "array"},
		{"Add", ":RECIPE_OVERRIDE_DIRS:0", "string", "~/Library/AutoPkg/RecipeOverrides"},
		{"Add", ":RECIPE_REPO_DIR", "string", "~/Library/AutoPkg/RecipeRepos"},
	}

	for _, args := range commands {
		cmdArgs := append([]string{"-c"}, args...)
		cmdArgs = append(cmdArgs, filepath)

		cmd := exec.Command("/usr/libexec/PlistBuddy", cmdArgs...)
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

// updatePlistPrefs updates an existing plist preferences file
func updatePlistPrefs(filepath, munkiRepoPath string) error {
	// Check if MUNKI_REPO key exists
	checkCmd := exec.Command("/usr/libexec/PlistBuddy", "-c", "Print :MUNKI_REPO", filepath)
	err := checkCmd.Run()

	if err == nil {
		// Key exists, update it
		cmd := exec.Command("/usr/libexec/PlistBuddy", "-c", fmt.Sprintf("Set :MUNKI_REPO %s", munkiRepoPath), filepath)
		return cmd.Run()
	} else {
		// Key doesn't exist, add it
		cmd := exec.Command("/usr/libexec/PlistBuddy", "-c", fmt.Sprintf("Add :MUNKI_REPO string %s", munkiRepoPath), filepath)
		return cmd.Run()
	}
}

// updateJSONPrefs updates an existing JSON preferences file
func updateJSONPrefs(filepath, munkiRepoPath string) error {
	// Install jq if needed
	installCmd := exec.Command("brew", "install", "jq")
	_ = installCmd.Run()

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "autopkg-prefs-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	// Update JSON using jq
	cmd := exec.Command("jq", fmt.Sprintf(`.MUNKI_REPO = "%s"`, munkiRepoPath), filepath)
	cmd.Stdout = tmpFile
	if err := cmd.Run(); err != nil {
		return err
	}

	// Close the temp file before moving
	tmpFile.Close()

	// Move the temp file to the original
	return os.Rename(tmpFile.Name(), filepath)
}
