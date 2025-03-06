package suspiciouspackage

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/helpers"
	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// Config holds all the configuration options for AutoPkg setup operations
type Config struct {
	ForceUpdate bool
}

// InstallSuspiciousPackage checks for the presence of Suspicious Package,
// and if it's not installed (or forced update is requested), downloads and installs it.
func InstallSuspiciousPackage(config *Config) (string, error) {
	// Define the expected path of the Suspicious Package application
	appPath := "/Applications/Suspicious Package.app"

	// Only install if not present or forced update requested
	if _, err := os.Stat(appPath); err == nil && !config.ForceUpdate {
		// Try to get the version
		version := "installed"
		cmdVersion := exec.Command("defaults", "read", appPath+"/Contents/Info", "CFBundleShortVersionString")
		if versionBytes, err := cmdVersion.Output(); err == nil {
			version = strings.TrimSpace(string(versionBytes))
		}

		logger.Logger(fmt.Sprintf("Suspicious Package %s is already installed.", version), logger.LogInfo)
		return version, nil
	}

	logger.Logger("⬇️ Downloading Suspicious Package", logger.LogInfo)

	// Define the download URL and temporary DMG path
	dmgURL := "https://mothersruin.com/software/downloads/SuspiciousPackage.dmg"
	dmgPath := "/tmp/SuspiciousPackage.dmg"

	// Download the DMG file
	if err := helpers.DownloadFile(dmgURL, dmgPath); err != nil {
		return "", fmt.Errorf("failed to download Suspicious Package: %w", err)
	}

	// Mount the DMG using hdiutil.
	// We use a fixed mount point for simplicity.
	mountPoint := "/Volumes/SuspiciousPackage"
	cmdMount := exec.Command("hdiutil", "attach", dmgPath, "-mountpoint", mountPoint, "-nobrowse", "-quiet")
	if err := cmdMount.Run(); err != nil {
		return "", fmt.Errorf("failed to mount DMG: %w", err)
	}

	// The DMG is expected to contain "Suspicious Package.app".
	sourceAppPath := filepath.Join(mountPoint, "Suspicious Package.app")
	if _, err := os.Stat(sourceAppPath); os.IsNotExist(err) {
		// Unmount if the app is not found
		_ = exec.Command("hdiutil", "detach", mountPoint, "-quiet").Run()
		return "", fmt.Errorf("Suspicious Package.app not found in the mounted DMG")
	}

	// Copy the Suspicious Package app to /Applications
	cmdCopy := exec.Command("cp", "-R", sourceAppPath, "/Applications/")
	output, err := cmdCopy.CombinedOutput()
	if err != nil {
		// Unmount the DMG on error
		_ = exec.Command("hdiutil", "detach", mountPoint, "-quiet").Run()
		return "", fmt.Errorf("failed to copy Suspicious Package: %w; output: %s", err, string(output))
	}

	// Unmount the DMG
	if err := exec.Command("hdiutil", "detach", mountPoint, "-quiet").Run(); err != nil {
		return "", fmt.Errorf("failed to unmount DMG: %w", err)
	}

	// Get the installed version
	version := "installed"
	cmdVersion := exec.Command("defaults", "read", appPath+"/Contents/Info", "CFBundleShortVersionString")
	if versionBytes, err := cmdVersion.Output(); err == nil {
		version = strings.TrimSpace(string(versionBytes))
	}

	// Open and then close the application to ensure it's registered with the system
	cmdOpen := exec.Command("open", "-a", "Suspicious Package")
	if err := cmdOpen.Run(); err != nil {
		logger.Logger(fmt.Sprintf("Warning: Failed to open Suspicious Package: %v", err), logger.LogWarning)
	}

	// Wait a moment for the app to register
	time.Sleep(3 * time.Second)

	// Close the app
	cmdClose := exec.Command("osascript", "-e", `tell application "Suspicious Package" to quit`)
	if err := cmdClose.Run(); err != nil {
		logger.Logger(fmt.Sprintf("Warning: Failed to close Suspicious Package: %v", err), logger.LogWarning)
	}

	// Wait a moment for the app to fully close
	time.Sleep(1 * time.Second)

	logger.Logger(fmt.Sprintf("✅ Suspicious Package %s installed", version), logger.LogSuccess)
	return version, nil
}
