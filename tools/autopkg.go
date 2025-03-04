package autopkg

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Config holds all the configuration options for AutoPkg operations
type Config struct {
	MunkiRepoPath   string
	UsePrefsFile    bool
	PrefsFilePath   string
	PrefsFileContent string
	RecipeRepos     []string
	RecipeName      string
	UploadResults   bool
}

// SetupEnvironment prepares the environment for AutoPkg
func SetupEnvironment(config *Config) error {
	fmt.Println("Setting up environment for AutoPkg...")
	
	// Create Munki repo directory if specified
	if err := os.MkdirAll(config.MunkiRepoPath, 0755); err != nil {
		return fmt.Errorf("failed to create Munki repo directory: %w", err)
	}
	
	return nil
}

// InstallAutoPkg downloads and installs the latest AutoPkg release
func InstallAutoPkg() (string, error) {
	fmt.Println("Downloading latest AutoPkg release...")
	
	// Get latest release URL
	latestReleaseURL, err := getLatestAutoPkgReleaseURL()
	if err != nil {
		return "", fmt.Errorf("failed to get latest AutoPkg release URL: %w", err)
	}
	fmt.Printf("Latest AutoPkg release URL: %s\n", latestReleaseURL)
	
	// Download the package
	pkgPath := "/tmp/AutoPkg.pkg"
	if err := downloadFile(latestReleaseURL, pkgPath); err != nil {
		return "", fmt.Errorf("failed to download AutoPkg package: %w", err)
	}
	
	// Install the package
	cmd := exec.Command("sudo", "/usr/sbin/installer", "-pkg", pkgPath, "-target", "/")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to install AutoPkg package: %w", err)
	}
	
	// Verify installation and capture version
	versionCmd := exec.Command("autopkg", "version")
	versionOutput, err := versionCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get AutoPkg version: %w", err)
	}
	
	version := strings.TrimSpace(string(versionOutput))
	fmt.Printf("AutoPkg %s installed successfully\n", version)
	
	return version, nil
}

// ConfigurePrefsFile sets up the external preferences file for AutoPkg
func ConfigurePrefsFile(config *Config) (string, error) {
	if !config.UsePrefsFile {
		return "", nil
	}
	
	fmt.Println("Setting up external preferences file...")
	
	// If PrefsFileContent is provided, decode and write it to the file
	if config.PrefsFileContent != "" {
		fmt.Println("Creating preferences file from provided content")
		decoded, err := base64.StdEncoding.DecodeString(config.PrefsFileContent)
		if err != nil {
			return "", fmt.Errorf("failed to decode preferences file content: %w", err)
		}
		
		if err := os.WriteFile(config.PrefsFilePath, decoded, 0644); err != nil {
			return "", fmt.Errorf("failed to write preferences file: %w", err)
		}
	} else {
		// Check if the preferences file already exists
		if _, err := os.Stat(config.PrefsFilePath); os.IsNotExist(err) {
			fmt.Printf("Creating new preferences file at %s\n", config.PrefsFilePath)
			
			// Determine file format based on extension
			ext := filepath.Ext(config.PrefsFilePath)
			if ext == ".json" {
				// Create JSON preferences file
				prefs := map[string]interface{}{
					"MUNKI_REPO":          config.MunkiRepoPath,
					"CACHE_DIR":           "~/Library/AutoPkg/Cache",
					"RECIPE_SEARCH_DIRS":  []string{".", "~/Library/AutoPkg/Recipes", "/Library/AutoPkg/Recipes"},
					"RECIPE_OVERRIDE_DIRS": []string{"~/Library/AutoPkg/RecipeOverrides"},
					"RECIPE_REPO_DIR":     "~/Library/AutoPkg/RecipeRepos",
				}
				
				jsonData, err := json.MarshalIndent(prefs, "", "  ")
				if err != nil {
					return "", fmt.Errorf("failed to marshal JSON preferences: %w", err)
				}
				
				if err := os.WriteFile(config.PrefsFilePath, jsonData, 0644); err != nil {
					return "", fmt.Errorf("failed to write JSON preferences file: %w", err)
				}
			} else {
				// Create Plist preferences file using PlistBuddy
				if err := createPlistPrefs(config.PrefsFilePath, config.MunkiRepoPath); err != nil {
					return "", fmt.Errorf("failed to create plist preferences: %w", err)
				}
			}
		} else {
			fmt.Printf("Using existing preferences file at %s\n", config.PrefsFilePath)
			
			// Update MUNKI_REPO in the existing file
			ext := filepath.Ext(config.PrefsFilePath)
			if ext == ".json" {
				// Update JSON file (requires jq)
				if err := updateJSONPrefs(config.PrefsFilePath, config.MunkiRepoPath); err != nil {
					return "", fmt.Errorf("failed to update JSON preferences: %w", err)
				}
			} else {
				// Update plist file
				if err := updatePlistPrefs(config.PrefsFilePath, config.MunkiRepoPath); err != nil {
					return "", fmt.Errorf("failed to update plist preferences: %w", err)
				}
			}
		}
	}
	
	fmt.Printf("Preferences file ready at %s\n", config.PrefsFilePath)
	
	// Clear existing preferences to ensure we only use the file
	clearCmd := exec.Command("defaults", "delete", "com.github.autopkg")
	_ = clearCmd.Run() // Ignore errors if the domain doesn't exist
	
	return config.PrefsFilePath, nil
}

// ConfigureMunkiPreferences sets Munki preferences directly via defaults command
func ConfigureMunkiPreferences(config *Config) error {
	if config.UsePrefsFile {
		return nil
	}
	
	cmd := exec.Command("defaults", "write", "com.github.autopkg", "MUNKI_REPO", config.MunkiRepoPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set MUNKI_REPO preference: %w", err)
	}
	
	fmt.Printf("MUNKI_REPO set to %s\n", config.MunkiRepoPath)
	return nil
}

// AddRecipeRepositories adds AutoPkg recipe repositories
func AddRecipeRepositories(config *Config, prefsPath string) (int, error) {
	fmt.Println("Adding core AutoPkg recipe repository...")
	
	// Add the core recipes repository
	var args []string
	if prefsPath != "" {
		args = []string{"repo-add", "recipes", "--prefs", prefsPath}
		fmt.Printf("Using external preferences file: %s\n", prefsPath)
	} else {
		args = []string{"repo-add", "recipes"}
	}
	
	cmd := exec.Command("autopkg", args...)
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("failed to add core recipe repository: %w", err)
	}
	
	// Initialize repo counter
	repoCount := 1
	
	// Add additional repositories if specified
	for _, repo := range config.RecipeRepos {
		if repo == "" {
			continue
		}
		
		fmt.Printf("Adding recipe repository: %s\n", repo)
		
		args := []string{"repo-add", repo}
		if prefsPath != "" {
			args = append(args, "--prefs", prefsPath)
		}
		
		cmd := exec.Command("autopkg", args...)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: failed to add recipe repository %s: %v\n", repo, err)
			continue
		}
		
		repoCount++
	}
	
	return repoCount, nil
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
