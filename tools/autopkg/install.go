// setup.go provides autopkg setup related functions and wrappers
package autopkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/helpers"
	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// InstallConfig represents the configuration options for AutoPkg install operations
type InstallConfig struct {
	// Basic AutoPkg settings
	ForceUpdate bool
	UseBeta     bool
}

// RootCheck ensures the script is not running as root and logs the current user
func RootCheck() error {
	uid := os.Geteuid()

	// Get effective username (the account executor)
	currentUser, err := exec.Command("id", "-un").Output()
	effectiveUser := string(bytes.TrimSpace(currentUser))
	if err != nil {
		effectiveUser = fmt.Sprintf("unknown (error: %v)", err)
	}

	userGroups, err := exec.Command("id", "-Gn").Output()
	effectiveGroups := string(bytes.TrimSpace(userGroups))
	if err != nil {
		effectiveGroups = fmt.Sprintf("unknown (error: %v)", err)
	}

	hostname, _ := os.Hostname()

	logger.Logger(fmt.Sprintf("🔍 Debug: Execution Context:\n"+
		"• Effective User ID: %d\n"+
		"• Effective Username: %s\n"+
		"• User Groups: %s\n"+
		"• Hostname: %s\n"+
		"• Working Directory: %s",
		uid, effectiveUser, effectiveGroups, hostname, os.Getenv("PWD")), logger.LogDebug)

	if uid == 0 {
		return fmt.Errorf("this script is NOT MEANT to run as root; please run without sudo")
	}
	return nil
}

// CheckGit verifies git is installed, and installs it if needed
func CheckGit() error {
	gitCmd := exec.Command("git", "--version")
	if err := gitCmd.Run(); err == nil {
		logger.Logger("✅ Git is installed and functional", logger.LogSuccess)
		return nil
	}

	logger.Logger("🔧 Git not found, installing...", logger.LogInfo)
	return installGit()
}

// installGit installs git using the most direct method available
func installGit() error {
	brewCmd := exec.Command("which", "brew")
	if err := brewCmd.Run(); err == nil {
		// Use Homebrew to install git
		logger.Logger("🔄 Installing git via Homebrew...", logger.LogInfo)
		brewInstall := exec.Command("brew", "install", "git")
		brewInstall.Stdout = os.Stdout
		brewInstall.Stderr = os.Stderr
		if err := brewInstall.Run(); err != nil {
			return fmt.Errorf("failed to install git via Homebrew: %w", err)
		}
	} else {
		// Fall back to Xcode Command Line Tools if Homebrew isn't available
		logger.Logger("🔄 Installing git via Xcode Command Line Tools...", logger.LogInfo)
		cmd := exec.Command("xcode-select", "--install")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install Xcode Command Line Tools: %w", err)
		}
	}

	gitCmd := exec.Command("git", "--version")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("git still not available after installation attempt: %w", err)
	}

	logger.Logger("✅ Git successfully installed", logger.LogSuccess)
	return nil
}

// InstallAutoPkg ensures AutoPkg is installed and up to date.
// - If AutoPkg is already installed, it verifies the existing version and skips installation.
// - If 'ForceUpdate' is enabled, it will update AutoPkg instead of skipping.
// - If AutoPkg is not installed, it proceeds with installation.
func InstallAutoPkg(installConfig *InstallConfig) (string, error) {
	autopkgPath := "/Library/AutoPkg/autopkg"
	autopkgSymlinkPath := "/usr/local/bin/autopkg"

	autopkgExists := false
	actualPath := ""

	// Check if AutoPkg is installed via main path
	if _, err := os.Stat(autopkgPath); err == nil {
		autopkgExists = true
		actualPath = autopkgPath
	}

	// Check if AutoPkg is installed via symlink
	if _, err := os.Stat(autopkgSymlinkPath); err == nil {
		autopkgExists = true
		if actualPath == "" {
			actualPath = autopkgSymlinkPath
		}
	}

	// If AutoPkg exists and we're not forcing an update, just return the current version
	if autopkgExists && !installConfig.ForceUpdate {
		logger.Logger("✅ AutoPkg is already installed, checking version...", logger.LogInfo)

		versionCmd := exec.Command(actualPath, "version")
		versionOutput, err := versionCmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get AutoPkg version: %w", err)
		}

		version := strings.TrimSpace(string(versionOutput))
		logger.Logger(fmt.Sprintf("✅ AutoPkg %s is already installed. Skipping installation.", version), logger.LogSuccess)
		return version, nil
	}

	// If we're here, either AutoPkg is missing or a forced update is required
	if autopkgExists {
		logger.Logger("🔄 Force update enabled. Updating AutoPkg...", logger.LogInfo)
	} else {
		logger.Logger("⬇️ AutoPkg not found. Installing AutoPkg...", logger.LogInfo)
	}

	var releaseURL string
	var err error

	// Get the correct release URL (Beta or Stable)
	if installConfig.UseBeta {
		releaseURL, err = getBetaAutoPkgReleaseURL()
		logger.Logger("🧪 Fetching latest Beta AutoPkg Release...", logger.LogInfo)
	} else {
		releaseURL, err = getLatestAutoPkgReleaseURL()
		logger.Logger("🚀 Fetching latest Stable AutoPkg Release...", logger.LogInfo)
	}

	if err != nil {
		return "", fmt.Errorf("failed to retrieve AutoPkg release URL: %w", err)
	}

	logger.Logger(fmt.Sprintf("📥 AutoPkg release URL: %s", releaseURL), logger.LogInfo)

	// Proceed with downloading and installing AutoPkg
	pkgPath := "/tmp/autopkg-latest.pkg"
	if err := helpers.DownloadFile(releaseURL, pkgPath); err != nil {
		return "", fmt.Errorf("failed to download AutoPkg package: %w", err)
	}

	cmd := exec.Command("sudo", "installer", "-pkg", pkgPath, "-target", "/")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to install AutoPkg package: %w", err)
	}

	// Verify installation by checking the installed version
	versionCmd := exec.Command("/Library/AutoPkg/autopkg", "version")
	versionOutput, err := versionCmd.Output()
	if err != nil {
		// Fallback to checking the symlink if needed
		versionCmd = exec.Command(autopkgSymlinkPath, "version")
		versionOutput, err = versionCmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to retrieve AutoPkg version after installation: %w", err)
		}
	}

	version := strings.TrimSpace(string(versionOutput))
	logger.Logger(fmt.Sprintf("✅ AutoPkg %s successfully installed", version), logger.LogSuccess)

	return version, nil
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

	for _, release := range releases {
		if release.Prerelease {
			for _, asset := range release.Assets {
				if strings.HasSuffix(asset.Name, ".pkg") {
					logger.Logger(fmt.Sprintf("🔍 Found beta release: %s", release.TagName), logger.LogInfo)
					return asset.BrowserDownloadURL, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no beta release with pkg asset found")
}

// getLatestAutoPkgReleaseURL retrieves the URL of the latest AutoPkg release
func getLatestAutoPkgReleaseURL() (string, error) {

	req, err := http.NewRequest("GET", "https://api.github.com/repos/autopkg/autopkg/releases/latest", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add GitHub token for authentication if available
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken != "" {
		req.Header.Set("Authorization", "token "+githubToken)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "AutoPkgGitHubActions/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	if DEBUG {
		body, _ := io.ReadAll(resp.Body)
		logger.Logger(fmt.Sprintf("GitHub API response: %s", string(body)), logger.LogDebug)

		resp.Body = io.NopCloser(bytes.NewBuffer(body))
	}

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

	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, ".pkg") {
			logger.Logger(fmt.Sprintf("🔍 Found release %s with package %s", release.TagName, asset.Name), logger.LogInfo)
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("no pkg asset found in the latest release (tag: %s, assets count: %d)",
		release.TagName, len(release.Assets))
}
