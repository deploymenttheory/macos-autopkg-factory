package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
	sp "github.com/deploymenttheory/macos-autopkg-factory/tools/suspicious_package"
)

// SecurityScanResult represents the results of a package security scan
type SecurityScanResult struct {
	PackagePath         string
	SignatureStatus     string
	Notarized           bool
	CriticalIssues      int
	WarningIssues       int
	PrivilegedScripts   int
	LaunchdJobs         int
	NonStdPermissions   int
	ComponentCount      int
	SandboxedApps       int
	AppleSiliconSupport int
}

func main() {
	// Parse command line arguments
	packagePath := flag.String("package", "", "Path to the package file to analyze")
	outputDir := flag.String("output", "", "Directory to export results (optional)")
	checkTerm := flag.String("check-scripts", "", "Term to search for in installer scripts (optional)")
	checkOSVersion := flag.String("check-os", "", "Check compatibility with this macOS version (e.g. '14.0') (optional)")
	flag.Parse()

	if *packagePath == "" {
		fmt.Println("Usage: package-scanner -package /path/to/package.pkg [-output /path/to/export] [-check-scripts term] [-check-os version]")
		os.Exit(1)
	}

	// Check if the package exists
	if _, err := os.Stat(*packagePath); os.IsNotExist(err) {
		fmt.Printf("Error: Package not found at %s\n", *packagePath)
		os.Exit(1)
	}

	// Setup and install Suspicious Package if needed
	config := &sp.Config{
		ForceUpdate: false, // Only install if not already present
	}

	version, err := sp.InstallSuspiciousPackage(config)
	if err != nil {
		fmt.Printf("Failed to set up Suspicious Package: %v\n", err)
		os.Exit(1)
	}

	logger.Logger(fmt.Sprintf("📦 Suspicious Package %s ready", version), logger.LogSuccess)

	// Begin scanning
	logger.Logger(fmt.Sprintf("🔍 Starting security scan of package: %s", *packagePath), logger.LogInfo)

	// Initialize scan results
	scanResult := SecurityScanResult{
		PackagePath: *packagePath,
	}

	// 1. Check package signature and notarization
	pkgInfo, err := sp.AnalyzePackage(*packagePath)
	if err != nil {
		fmt.Printf("Failed to analyze package signature: %v\n", err)
	} else {
		scanResult.SignatureStatus = pkgInfo.SignatureStatus
		scanResult.Notarized = pkgInfo.Notarized

		logger.Logger(fmt.Sprintf("🔐 Package signature status: %s, Notarized: %t",
			pkgInfo.SignatureStatus, pkgInfo.Notarized), logger.LogInfo)
	}

	// 2. Check for critical issues
	issues, err := sp.FindCriticalIssues(*packagePath)
	if err != nil {
		fmt.Printf("Failed to check for issues: %v\n", err)
	} else {
		for _, issue := range issues {
			if issue.Priority == "critical" {
				scanResult.CriticalIssues++
				logger.Logger(fmt.Sprintf("❌ CRITICAL: %s", issue.Details), logger.LogError)
			} else if issue.Priority == "warning" {
				scanResult.WarningIssues++
				logger.Logger(fmt.Sprintf("⚠️ WARNING: %s", issue.Details), logger.LogWarning)
			}
		}
	}

	// 3. Find privileged scripts
	scripts, err := sp.FindPrivilegedScripts(*packagePath)
	if err != nil {
		fmt.Printf("Failed to check for privileged scripts: %v\n", err)
	} else {
		scanResult.PrivilegedScripts = len(scripts)
		if len(scripts) > 0 {
			logger.Logger(fmt.Sprintf("🔐 Found %d scripts running as root:", len(scripts)), logger.LogWarning)
			for _, script := range scripts {
				logger.Logger(fmt.Sprintf("  • %s (%s)", script.ShortName, script.RunsWhen), logger.LogInfo)
			}
		}
	}

	// 4. Search for specific terms in installer scripts if requested
	if *checkTerm != "" {
		matchingScripts, err := sp.SearchInstallerScripts(*packagePath, *checkTerm)
		if err != nil {
			fmt.Printf("Failed to search scripts for term '%s': %v\n", *checkTerm, err)
		} else if len(matchingScripts) > 0 {
			logger.Logger(fmt.Sprintf("🔎 Found %d scripts containing '%s':", len(matchingScripts), *checkTerm), logger.LogWarning)
			for _, script := range matchingScripts {
				// Use only fields that exist in the InstallerScript struct
				logger.Logger(fmt.Sprintf("  • %s (runs during %s)",
					script.Name, script.RunsWhen), logger.LogInfo)
			}
		}
	}

	// 5. Find launchd jobs
	launchdJobs, err := sp.FindLaunchdJobs(*packagePath)
	if err != nil {
		fmt.Printf("Failed to find launchd jobs: %v\n", err)
	} else {
		scanResult.LaunchdJobs = len(launchdJobs)
		if len(launchdJobs) > 0 {
			logger.Logger(fmt.Sprintf("⚙️ Found %d launchd jobs:", len(launchdJobs)), logger.LogInfo)
			for _, job := range launchdJobs {
				logger.Logger(fmt.Sprintf("  • %s (owner: %s, permissions: %s)", job.Path, job.Owner, job.Permissions), logger.LogInfo)
			}
		}
	}

	// 6. Check for files with non-standard permissions
	nonStdPerms, err := sp.FindNonStandardPermissions(*packagePath)
	if err != nil {
		fmt.Printf("Failed to check for non-standard permissions: %v\n", err)
	} else {
		scanResult.NonStdPermissions = len(nonStdPerms)
		if len(nonStdPerms) > 0 {
			logger.Logger(fmt.Sprintf("🔓 Found %d files with non-standard permissions:", len(nonStdPerms)), logger.LogWarning)

			// Only show the first 5 to avoid too much output
			maxShow := 5
			if len(nonStdPerms) < maxShow {
				maxShow = len(nonStdPerms)
			}

			for i := 0; i < maxShow; i++ {
				perm := nonStdPerms[i]
				logger.Logger(fmt.Sprintf("  • %s (%s:%s, %s)", perm.Path, perm.Owner, perm.Group, perm.Permissions), logger.LogInfo)
			}

			if len(nonStdPerms) > maxShow {
				logger.Logger(fmt.Sprintf("  • ... and %d more", len(nonStdPerms)-maxShow), logger.LogInfo)
			}
		}
	}

	// 7. Check component packages
	components, err := sp.GetComponentPackages(*packagePath)
	if err != nil {
		fmt.Printf("Failed to check component packages: %v\n", err)
	} else {
		scanResult.ComponentCount = len(components)

		installedCount := 0
		for _, comp := range components {
			if comp.Installed {
				installedCount++
			}
		}

		if installedCount > 0 {
			logger.Logger(fmt.Sprintf("📦 Found %d component packages, %d already installed", len(components), installedCount), logger.LogInfo)
			for _, comp := range components {
				if comp.Installed {
					logger.Logger(fmt.Sprintf("  • %s: v%s installed on %s (current pkg: v%s)",
						comp.ID, comp.InstalledVersion, comp.InstalledDate.Format("2006-01-02"), comp.Version), logger.LogInfo)
				}
			}
		}
	}

	// 8. Find sandboxed applications
	sandboxedApps, err := sp.FindSandboxedApps(*packagePath)
	if err != nil {
		fmt.Printf("Failed to find sandboxed apps: %v\n", err)
	} else {
		scanResult.SandboxedApps = len(sandboxedApps)
		if len(sandboxedApps) > 0 {
			logger.Logger(fmt.Sprintf("🔒 Found %d sandboxed applications:", len(sandboxedApps)), logger.LogInfo)
			for _, app := range sandboxedApps {
				networkDesc := ""
				if app.ClientNetwork && app.ServerNetwork {
					networkDesc = "outgoing and incoming network access"
				} else if app.ClientNetwork {
					networkDesc = "outgoing network access"
				} else if app.ServerNetwork {
					networkDesc = "incoming network access"
				} else {
					networkDesc = "no network access"
				}

				logger.Logger(fmt.Sprintf("  • %s (%s, %s)", app.Name, app.BundleID, networkDesc), logger.LogInfo)
			}
		}
	}

	// 9. Check Apple Silicon support
	supportInfo, err := sp.CheckAppleSiliconSupport(*packagePath)
	if err != nil {
		fmt.Printf("Failed to check Apple Silicon support: %v\n", err)
	} else {
		supportedCount := 0
		unsupportedComponents := []string{}

		for _, comp := range supportInfo {
			if comp.Supports {
				supportedCount++
			} else {
				unsupportedComponents = append(unsupportedComponents, comp.Name)
			}
		}

		scanResult.AppleSiliconSupport = supportedCount

		if len(unsupportedComponents) > 0 {
			logger.Logger(fmt.Sprintf("⚠️ Found %d of %d executables that may not support Apple Silicon:",
				len(unsupportedComponents), len(supportInfo)), logger.LogWarning)

			// Only show the first 5 to avoid too much output
			maxShow := 5
			if len(unsupportedComponents) < maxShow {
				maxShow = len(unsupportedComponents)
			}

			for i := 0; i < maxShow; i++ {
				logger.Logger(fmt.Sprintf("  • %s", unsupportedComponents[i]), logger.LogInfo)
			}

			if len(unsupportedComponents) > maxShow {
				logger.Logger(fmt.Sprintf("  • ... and %d more", len(unsupportedComponents)-maxShow), logger.LogInfo)
			}
		} else if len(supportInfo) > 0 {
			logger.Logger(fmt.Sprintf("✅ All %d executables support Apple Silicon", len(supportInfo)), logger.LogSuccess)
		}
	}

	// 10. Check OS compatibility if requested
	if *checkOSVersion != "" {
		incompatible, err := sp.CheckOSCompatibility(*packagePath, *checkOSVersion)
		if err != nil {
			fmt.Printf("Failed to check compatibility with macOS %s: %v\n", *checkOSVersion, err)
		} else if len(incompatible) > 0 {
			logger.Logger(fmt.Sprintf("⚠️ Found %d components incompatible with macOS %s:",
				len(incompatible), *checkOSVersion), logger.LogWarning)

			for _, comp := range incompatible {
				logger.Logger(fmt.Sprintf("  • %s requires macOS %s", comp.Name, comp.Version), logger.LogInfo)
			}
		} else {
			logger.Logger(fmt.Sprintf("✅ All components are compatible with macOS %s", *checkOSVersion), logger.LogSuccess)
		}
	}

	// 11. Export diffable manifest if requested
	if *outputDir != "" {
		// Make sure the output directory exists
		if _, err := os.Stat(*outputDir); os.IsNotExist(err) {
			if err := os.MkdirAll(*outputDir, 0755); err != nil {
				fmt.Printf("Failed to create output directory: %v\n", err)
			}
		}

		// Create a subdirectory with package name and timestamp
		pkgName := filepath.Base(*packagePath)
		pkgName = strings.TrimSuffix(pkgName, filepath.Ext(pkgName))
		timestamp := time.Now().Format("20060102-150405")
		manifestDir := filepath.Join(*outputDir, fmt.Sprintf("%s-%s", pkgName, timestamp))

		if err := sp.ExportDiffableManifest(*packagePath, manifestDir); err != nil {
			fmt.Printf("Failed to export diffable manifest: %v\n", err)
		} else {
			logger.Logger(fmt.Sprintf("📄 Exported diffable manifest to: %s", manifestDir), logger.LogSuccess)
		}
	}

	// 12. Output summary
	logger.Logger("📊 Security Scan Summary:", logger.LogInfo)
	logger.Logger(fmt.Sprintf("  • Package: %s", filepath.Base(*packagePath)), logger.LogInfo)
	logger.Logger(fmt.Sprintf("  • Signature: %s, Notarized: %t", scanResult.SignatureStatus, scanResult.Notarized), logger.LogInfo)
	logger.Logger(fmt.Sprintf("  • Issues: %d critical, %d warnings", scanResult.CriticalIssues, scanResult.WarningIssues), logger.LogInfo)
	logger.Logger(fmt.Sprintf("  • Privileged scripts: %d", scanResult.PrivilegedScripts), logger.LogInfo)
	logger.Logger(fmt.Sprintf("  • Launchd jobs: %d", scanResult.LaunchdJobs), logger.LogInfo)
	logger.Logger(fmt.Sprintf("  • Non-standard permissions: %d", scanResult.NonStdPermissions), logger.LogInfo)
	logger.Logger(fmt.Sprintf("  • Component packages: %d", scanResult.ComponentCount), logger.LogInfo)
	logger.Logger(fmt.Sprintf("  • Sandboxed applications: %d", scanResult.SandboxedApps), logger.LogInfo)

	// Final results based on critical findings
	if scanResult.CriticalIssues > 0 {
		logger.Logger("❌ Package contains CRITICAL issues and should be carefully reviewed", logger.LogError)
	} else if scanResult.WarningIssues > 0 || scanResult.PrivilegedScripts > 0 || scanResult.NonStdPermissions > 5 {
		logger.Logger("⚠️ Package contains potential security concerns that should be reviewed", logger.LogWarning)
	} else if scanResult.SignatureStatus != "Apple Inc" && scanResult.SignatureStatus != "Developer ID" {
		logger.Logger("⚠️ Package is not signed with an Apple-issued certificate", logger.LogWarning)
	} else {
		logger.Logger("✅ No critical security issues found in package", logger.LogSuccess)
	}
}
