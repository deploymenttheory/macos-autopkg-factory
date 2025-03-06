package suspiciouspackage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// FindLaunchdJobs finds all launchd job configuration files in the package
func FindLaunchdJobs(packagePath string) ([]LaunchdJob, error) {
	logger.Logger(fmt.Sprintf("🔍 Finding launchd jobs in package: %s", packagePath), logger.LogInfo)

	output, err := runNodeScript("findLaunchdJobs", packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to find launchd jobs: %w", err)
	}

	var jobs []LaunchdJob
	if err := json.Unmarshal(output, &jobs); err != nil {
		return nil, fmt.Errorf("failed to parse launchd jobs: %w", err)
	}

	logger.Logger(fmt.Sprintf("⚙️ Found %d launchd jobs in package", len(jobs)), logger.LogSuccess)
	return jobs, nil
}

// FindInstallerScriptsRunAsRoot finds installer scripts with root privileges
func FindInstallerScriptsRunAsRoot(packagePath string) ([]PrivilegedInstallerScript, error) {
	logger.Logger(fmt.Sprintf("🔍 Finding privileged scripts in package: %s", packagePath), logger.LogInfo)

	output, err := runNodeScript("findInstallerScriptsRunAsRoot", packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to find privileged scripts: %w", err)
	}

	var scripts []PrivilegedInstallerScript
	if err := json.Unmarshal(output, &scripts); err != nil {
		return nil, fmt.Errorf("failed to parse privileged scripts: %w", err)
	}

	if len(scripts) == 0 {
		logger.Logger("✅ No privileged scripts found in package", logger.LogSuccess)
		return scripts, nil
	}

	logger.Logger(fmt.Sprintf("🔐 Found %d privileged scripts running as rootin package", len(scripts)), logger.LogSuccess)

	// Log script details at INFO level, skipping script content
	for _, script := range scripts {
		logger.Logger(fmt.Sprintf("📜 Script Name: %s (Short: %s)", script.Name, script.ShortName), logger.LogInfo)
		logger.Logger(fmt.Sprintf("   ⏳ Script runs: %s", script.When), logger.LogInfo)
	}

	// Remove the faulty warning log
	return scripts, nil
}

// FindCriticalIssues gets the most critical issues from a package
func FindCriticalIssues(packagePath string) ([]PackageIssue, error) {
	logger.Logger(fmt.Sprintf("🔍 Finding critical issues in package: %s", packagePath), logger.LogInfo)

	output, err := runNodeScript("findCriticalIssues", packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to find critical issues: %w", err)
	}

	var issues []PackageIssue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse critical issues: %w", err)
	}

	logger.Logger(fmt.Sprintf("⚠️ Found %d critical/warning issues in package", len(issues)), logger.LogSuccess)
	return issues, nil
}

// GetMacOSMinimumVersionRequirements gets the OS version requirements for executables in a package
func GetMacOSMinimumVersionRequirements(packagePath string) ([]OSRequirement, error) {
	logger.Logger(fmt.Sprintf("🔍 Getting OS requirements from package: %s", packagePath), logger.LogInfo)

	output, err := runNodeScript("getMacOSMinimumVersionRequirements", packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get OS requirements: %w", err)
	}

	var requirements []OSRequirement
	if err := json.Unmarshal(output, &requirements); err != nil {
		return nil, fmt.Errorf("failed to parse OS requirements: %w", err)
	}

	if len(requirements) == 0 {
		logger.Logger("✅ No specific macOS minimum version requirements found in package", logger.LogSuccess)
		return requirements, nil
	}

	logger.Logger(fmt.Sprintf("💻 Found macOS minimum version requirements for %d components in package", len(requirements)), logger.LogSuccess)

	// Log each requirement at INFO level
	for _, req := range requirements {
		logger.Logger(fmt.Sprintf("  • %s requires macOS %s", req.Name, req.Version), logger.LogInfo)
	}

	return requirements, nil
}

// CheckOSCompatibility checks if executables in a package support a specific macOS version
func CheckOSCompatibility(packagePath string, osVersion string) ([]OSRequirement, error) {
	logger.Logger(fmt.Sprintf("🔍 Checking compatibility with macOS %s in package: %s", osVersion, packagePath), logger.LogInfo)

	output, err := runNodeScript("checkOSCompatibility", packagePath, osVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to check OS compatibility: %w", err)
	}

	var incompatible []OSRequirement
	if err := json.Unmarshal(output, &incompatible); err != nil {
		return nil, fmt.Errorf("failed to parse OS compatibility results: %w", err)
	}

	if len(incompatible) == 0 {
		logger.Logger(fmt.Sprintf("✅ All components are compatible with macOS %s", osVersion), logger.LogSuccess)
	} else {
		logger.Logger(fmt.Sprintf("⚠️ Found %d components incompatible with macOS %s", len(incompatible), osVersion), logger.LogWarning)
	}

	return incompatible, nil
}

// FindNonStandardPermissions checks for files with non-standard permissions
func FindNonStandardPermissions(packagePath string) ([]NonStandardPermission, error) {
	logger.Logger(fmt.Sprintf("🔍 Finding non-standard permissions in package: %s", packagePath), logger.LogInfo)

	output, err := runNodeScript("findNonStandardPermissions", packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to find non-standard permissions: %w", err)
	}

	var permissions []NonStandardPermission
	if err := json.Unmarshal(output, &permissions); err != nil {
		return nil, fmt.Errorf("failed to parse non-standard permissions: %w", err)
	}

	logger.Logger(fmt.Sprintf("📝 Found %d items with non-standard permissions", len(permissions)), logger.LogSuccess)
	return permissions, nil
}

// SearchInstallerScripts searches for specific strings in installer scripts
func SearchInstallerScripts(packagePath string, searchTerm string) ([]InstallerScript, error) {
	logger.Logger(fmt.Sprintf("🔍 Searching for '%s' in installer scripts of package: %s", searchTerm, packagePath), logger.LogInfo)

	output, err := runNodeScript("searchInstallerScripts", packagePath, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("failed to search installer scripts: %w", err)
	}

	var scripts []InstallerScript
	if err := json.Unmarshal(output, &scripts); err != nil {
		return nil, fmt.Errorf("failed to parse script search results: %w", err)
	}

	logger.Logger(fmt.Sprintf("🔎 Found %d scripts containing '%s'", len(scripts), searchTerm), logger.LogSuccess)
	return scripts, nil
}

// GetComponentPackages gets information about component packages and their installation history
func GetComponentPackages(packagePath string) ([]ComponentInfo, error) {
	logger.Logger(fmt.Sprintf("🔍 Getting component packages from: %s", packagePath), logger.LogInfo)

	output, err := runNodeScript("getComponentPackages", packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get component packages: %w", err)
	}

	var components []ComponentInfo
	if err := json.Unmarshal(output, &components); err != nil {
		return nil, fmt.Errorf("failed to parse component packages: %w", err)
	}

	logger.Logger(fmt.Sprintf("📦 Found %d component packages", len(components)), logger.LogSuccess)
	return components, nil
}

// FindItemsByUTI finds items that match a specific UTI conformance pattern
func FindItemsByUTI(packagePath string, utiPattern string) ([]UTIItem, error) {
	logger.Logger(fmt.Sprintf("🔍 Finding items conforming to UTI '%s' in package: %s", utiPattern, packagePath), logger.LogInfo)

	output, err := runNodeScript("findItemsByUTI", packagePath, utiPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find items by UTI: %w", err)
	}

	var items []UTIItem
	if err := json.Unmarshal(output, &items); err != nil {
		return nil, fmt.Errorf("failed to parse UTI items: %w", err)
	}

	logger.Logger(fmt.Sprintf("🔖 Found %d items conforming to UTI '%s'", len(items), utiPattern), logger.LogSuccess)
	return items, nil
}

// FindSandboxedApps finds all sandboxed applications in a package
func FindSandboxedApps(packagePath string) ([]SandboxedApp, error) {
	logger.Logger(fmt.Sprintf("🔍 Finding sandboxed apps in package: %s", packagePath), logger.LogInfo)

	output, err := runNodeScript("findSandboxedApps", packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to find sandboxed apps: %w", err)
	}

	var apps []SandboxedApp
	if err := json.Unmarshal(output, &apps); err != nil {
		return nil, fmt.Errorf("failed to parse sandboxed apps: %w", err)
	}

	logger.Logger(fmt.Sprintf("🔒 Found %d sandboxed apps in package", len(apps)), logger.LogSuccess)
	return apps, nil
}

// runNodeScript executes the JavaScript helper function using Node
func runNodeScript(scriptName string, args ...string) ([]byte, error) {
	// Get the module's root directory
	_, currentFile, _, _ := runtime.Caller(0)
	moduleRoot := filepath.Dir(filepath.Dir(currentFile)) // Go up two directories from this file

	// Path to the JavaScript helper within the module structure
	scriptPath := filepath.Join(moduleRoot, "suspicious_package", "scripts", "suspiciousPackageHelpers.js")

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("JavaScript helper script not found at %s", scriptPath)
	}

	// Log the script path for debugging
	//fmt.Printf("Using script at: %s\n", scriptPath)

	// First check if Node.js is installed
	_, err := exec.LookPath("node")
	if err != nil {
		return nil, fmt.Errorf("Node.js not found on system: %w", err)
	}

	nodeScript := fmt.Sprintf(`
			try {
					const helpers = require('%s');
					if (typeof helpers.%s !== 'function') {
							console.error('Function "%s" not found in helpers module. Available functions: ' + Object.keys(helpers).join(', '));
							process.exit(1);
					}
					(async () => {
							try {
									console.error("Starting execution of %s");
									const result = await helpers.%s(%s);
									console.log(JSON.stringify(result || []));
							} catch (err) {
									console.error('Error executing helper function: ' + err.message);
									console.error(err.stack);
									process.exit(1);
							}
					})();
			} catch (err) {
					console.error('Error loading helpers module: ' + err.message);
					console.error(err.stack);
					process.exit(1);
			}
	`, scriptPath, scriptName, scriptName, scriptName, scriptName, formatArgs(args))

	// Create command with captured stderr
	nodeCmd := exec.Command("node", "-e", nodeScript)

	// Capture both stdout and stderr
	var stdout, stderr bytes.Buffer
	nodeCmd.Stdout = &stdout
	nodeCmd.Stderr = &stderr

	// Run the command
	err = nodeCmd.Run()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, stderr.String())
	}

	// If stdout is empty, return an empty array to avoid JSON parsing errors
	if stdout.Len() == 0 {
		return []byte("[]"), nil
	}

	return stdout.Bytes(), nil
}

// formatArgs formats arguments for JavaScript function call
func formatArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}

	formattedArgs := ""
	for i, arg := range args {
		if i > 0 {
			formattedArgs += ", "
		}
		formattedArgs += fmt.Sprintf("'%s'", arg)
	}

	return formattedArgs
}

// AnalyzePackage analyzes basic information about a package
func AnalyzePackage(packagePath string) (*PackageInfo, error) {
	logger.Logger(fmt.Sprintf("🔍 Analyzing package: %s", packagePath), logger.LogInfo)

	output, err := runNodeScript("checkPackageSignature", packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze package: %w", err)
	}

	var packageInfo PackageInfo
	if err := json.Unmarshal(output, &packageInfo); err != nil {
		return nil, fmt.Errorf("failed to parse package info: %w", err)
	}

	logger.Logger(fmt.Sprintf("📦 Package analysis complete: %s", packageInfo.Name), logger.LogSuccess)
	return &packageInfo, nil
}

// GetInstalledApps gets all applications installed by a package
func GetInstalledApps(packagePath string) ([]InstalledItem, error) {
	logger.Logger(fmt.Sprintf("🔍 Finding installed apps in package: %s", packagePath), logger.LogInfo)

	output, err := runNodeScript("getInstalledApps", packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get installed apps: %w", err)
	}

	var apps []InstalledItem
	if err := json.Unmarshal(output, &apps); err != nil {
		return nil, fmt.Errorf("failed to parse installed apps: %w", err)
	}

	logger.Logger(fmt.Sprintf("📱 Found %d apps in package", len(apps)), logger.LogSuccess)
	return apps, nil
}

// GetInstallerScripts gets all scripts in a package
func GetInstallerScripts(packagePath string) ([]InstallerScript, error) {
	logger.Logger(fmt.Sprintf("🔍 Finding installer scripts in package: %s", packagePath), logger.LogInfo)

	output, err := runNodeScript("getInstallerScripts", packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get installer scripts: %w", err)
	}

	var scripts []InstallerScript
	if err := json.Unmarshal(output, &scripts); err != nil {
		return nil, fmt.Errorf("failed to parse installer scripts: %w", err)
	}

	logger.Logger(fmt.Sprintf("📜 Found %d scripts in package", len(scripts)), logger.LogSuccess)
	return scripts, nil
}

// GetPackageIssues gets all issues in a package
func GetPackageIssues(packagePath string) ([]PackageIssue, error) {
	logger.Logger(fmt.Sprintf("🔍 Finding issues in package: %s", packagePath), logger.LogInfo)

	output, err := runNodeScript("findPackageIssues", packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get package issues: %w", err)
	}

	var issues []PackageIssue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse package issues: %w", err)
	}

	logger.Logger(fmt.Sprintf("⚠️ Found %d issues in package", len(issues)), logger.LogSuccess)
	return issues, nil
}

// FindComponentsWithEntitlement finds components with a specific entitlement
func FindComponentsWithEntitlement(packagePath string, entitlementKey string) ([]ComponentWithEntitlement, error) {
	logger.Logger(fmt.Sprintf("🔍 Finding components with entitlement '%s' in package: %s", entitlementKey, packagePath), logger.LogInfo)

	output, err := runNodeScript("findComponentsWithEntitlement", packagePath, entitlementKey)
	if err != nil {
		return nil, fmt.Errorf("failed to find components with entitlement: %w", err)
	}

	var components []ComponentWithEntitlement
	if err := json.Unmarshal(output, &components); err != nil {
		return nil, fmt.Errorf("failed to parse components with entitlement: %w", err)
	}

	logger.Logger(fmt.Sprintf("🔐 Found %d components with entitlement '%s'", len(components), entitlementKey), logger.LogSuccess)
	return components, nil
}

// ExportDiffableManifest exports a diffable manifest from a package
func ExportDiffableManifest(packagePath string, outputPath string) error {
	logger.Logger(fmt.Sprintf("📤 Exporting diffable manifest from package: %s", packagePath), logger.LogInfo)

	_, err := runNodeScript("exportDiffableManifest", packagePath, outputPath)
	if err != nil {
		return fmt.Errorf("failed to export diffable manifest: %w", err)
	}

	logger.Logger(fmt.Sprintf("✅ Exported diffable manifest to: %s", outputPath), logger.LogSuccess)
	return nil
}
