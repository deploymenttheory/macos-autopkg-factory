package autopkg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/helpers"
	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// InstallSuspiciousPackage checks for the presence of Suspicious Package,
// and if it's not installed (or forced update is requested), downloads and installs it.
func InstallSuspiciousPackage(config *Config) (string, error) {
	// Define the expected path of the Suspicious Package application
	appPath := "/Applications/Suspicious Package.app"

	// Only install if not present or forced update requested
	if _, err := os.Stat(appPath); err == nil && !config.ForceUpdate {
		// Optionally, you could try to extract version info from the app's Info.plist
		logger.Logger("Suspicious Package is already installed.", logger.LogInfo)
		return "installed", nil
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

	// Optionally, wait a short while for the system to register the new app
	time.Sleep(2 * time.Second)

	logger.Logger("✅ Suspicious Package installed", logger.LogSuccess)
	return "installed", nil
}
