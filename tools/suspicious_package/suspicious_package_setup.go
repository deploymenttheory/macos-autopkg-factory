package suspiciouspackage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
	"github.com/deploymenttheory/macos-autopkg-factory/tools/pkg"
)

// SecurityScanResult represents the results of a package security scan
type SecurityScanResult struct {
	PackagePath         string
	SignatureStatus     string
	Notarized           bool
	CertificateInfo     string
	CertificateIssuer   string
	CertificateExpiry   string
	CriticalIssues      int
	WarningIssues       int
	PrivilegedScripts   int
	LaunchdJobs         int
	NonStdPermissions   int
	ComponentCount      int
	SandboxedApps       int
	AppleSiliconSupport int
}

// ScanOptions represents the options for package security scanning
type ScanOptions struct {
	PackagePath    string
	OutputDir      string
	CheckTerm      string
	CheckOSVersion string
}

// PackageSecurityScanner scans a macOS package for security issues
func PackageSecurityScanner(options ScanOptions) error {
	// Setup and install Suspicious Package if needed
	config := &Config{
		ForceUpdate: false, // Only install if not already present
	}

	version, err := InstallSuspiciousPackage(config)
	if err != nil {
		return fmt.Errorf("failed to set up Suspicious Package: %v", err)
	}

	logger.Logger(fmt.Sprintf("📦 Suspicious Package %s ready", version), logger.LogSuccess)

	// Begin scanning
	logger.Logger(fmt.Sprintf("🔍 Starting security scan of package: %s", options.PackagePath), logger.LogInfo)

	// Initialize scan results
	scanResult := SecurityScanResult{
		PackagePath:         options.PackagePath,
		SignatureStatus:     "Unknown",
		Notarized:           false,
		CertificateInfo:     "Unknown",
		CertificateIssuer:   "Unknown",
		CertificateExpiry:   "Unknown",
		CriticalIssues:      0,
		WarningIssues:       0,
		PrivilegedScripts:   0,
		LaunchdJobs:         0,
		NonStdPermissions:   0,
		ComponentCount:      0,
		SandboxedApps:       0,
		AppleSiliconSupport: 0,
	}

	// 1. Check package signature and notarization
	certInfo, err := pkg.GetPackageSigningCertificate(options.PackagePath)
	if err != nil {
		logger.Logger(fmt.Sprintf("❌ Failed to analyze package signature: %v", err), logger.LogError)
	} else {
		scanResult.SignatureStatus = certInfo.SignatureStatus
		scanResult.Notarized = certInfo.Notarized
		scanResult.CertificateInfo = certInfo.CertificateInfo

		// Use the first certificate in the chain as the issuer (if available)
		if len(certInfo.CertificateChain) > 0 {
			scanResult.CertificateIssuer = certInfo.CertificateChain[0]
		} else {
			scanResult.CertificateIssuer = "Unknown"
		}

		// Use the first expiry date if available
		if len(certInfo.ExpiryDates) > 0 {
			scanResult.CertificateExpiry = certInfo.ExpiryDates[0]
		} else {
			scanResult.CertificateExpiry = "Unknown"
		}
	}

	// 2. Check for critical issues
	issues, err := FindCriticalIssues(options.PackagePath)
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

	// 3. Find scripts run as root
	scripts, err := FindInstallerScriptsRunAsRoot(options.PackagePath)
	if err != nil {
		fmt.Println("Error:", err)
		logger.Logger(fmt.Sprintf("❌ CRITICAL: %d", scripts), logger.LogError)
	}

	// 4. Search for specific terms in installer scripts if requested
	if options.CheckTerm != "" {
		matchingScripts, err := SearchInstallerScripts(options.PackagePath, options.CheckTerm)
		if err != nil {
			fmt.Printf("Failed to search scripts for term '%s': %v\n", options.CheckTerm, err)
		} else if len(matchingScripts) > 0 {
			logger.Logger(fmt.Sprintf("🔎 Found %d scripts containing '%s':", len(matchingScripts), options.CheckTerm), logger.LogWarning)
			for _, script := range matchingScripts {
				// Use only fields that exist in the InstallerScript struct
				logger.Logger(fmt.Sprintf("  • %s (runs during %s)",
					script.Name, script.RunsWhen), logger.LogInfo)
			}
		}
	}

	// 5. Find launchd jobs
	launchdJobs, err := FindLaunchdJobs(options.PackagePath)
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
	nonStdPerms, err := FindNonStandardPermissions(options.PackagePath)
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
	components, err := GetComponentPackages(options.PackagePath)
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
	sandboxedApps, err := FindSandboxedApps(options.PackagePath)
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

	// 9. Check Apple Silicon support using package architecture metadata
	supportedArchitectures, err := pkg.GetPackageSupportedMacOSArchitecture(options.PackagePath)
	if err != nil {
		logger.Logger(fmt.Sprintf("❌ Failed to get package architectures: %v", err), logger.LogError)
	} else {
		logger.Logger(fmt.Sprintf("💻 Package supports: %s", strings.Join(supportedArchitectures, ", ")), logger.LogSuccess)

		// Determine if Apple Silicon (arm64) is supported
		supportsAppleSilicon := false
		for _, arch := range supportedArchitectures {
			if arch == "arm64" {
				supportsAppleSilicon = true
				break
			}
		}

		if supportsAppleSilicon {
			logger.Logger("✅ Package explicitly supports Apple Silicon (arm64)", logger.LogSuccess)
		} else {
			logger.Logger("⚠️ Package does not explicitly declare Apple Silicon support", logger.LogWarning)
		}
	}

	// 10. Check OS compatibility if requested
	if options.CheckOSVersion != "" {
		incompatible, err := CheckOSCompatibility(options.PackagePath, options.CheckOSVersion)
		if err != nil {
			fmt.Printf("Failed to check compatibility with macOS %s: %v\n", options.CheckOSVersion, err)
		} else if len(incompatible) > 0 {
			logger.Logger(fmt.Sprintf("⚠️ Found %d components incompatible with macOS %s:",
				len(incompatible), options.CheckOSVersion), logger.LogWarning)

			for _, comp := range incompatible {
				logger.Logger(fmt.Sprintf("  • %s requires macOS %s", comp.Name, comp.Version), logger.LogInfo)
			}
		} else {
			logger.Logger(fmt.Sprintf("✅ All components are compatible with macOS %s", options.CheckOSVersion), logger.LogSuccess)
		}
	}

	// 11. Get OS requirements for the package
	_, err = GetMacOSMinimumVersionRequirements(options.PackagePath)
	if err != nil {
		logger.Logger(fmt.Sprintf("❌ Failed to get OS requirements: %v", err), logger.LogError)
	}

	// 12. Export diffable manifest if requested
	if options.OutputDir != "" {
		// Make sure the output directory exists
		if _, err := os.Stat(options.OutputDir); os.IsNotExist(err) {
			if err := os.MkdirAll(options.OutputDir, 0755); err != nil {
				fmt.Printf("Failed to create output directory: %v\n", err)
			}
		}

		// Create a subdirectory with package name and timestamp
		pkgName := filepath.Base(options.PackagePath)
		pkgName = strings.TrimSuffix(pkgName, filepath.Ext(pkgName))
		timestamp := time.Now().Format("20060102-150405")
		manifestDir := filepath.Join(options.OutputDir, fmt.Sprintf("%s-%s", pkgName, timestamp))

		if err := ExportDiffableManifest(options.PackagePath, manifestDir); err != nil {
			fmt.Printf("Failed to export diffable manifest: %v\n", err)
		} else {
			logger.Logger(fmt.Sprintf("📄 Exported diffable manifest to: %s", manifestDir), logger.LogSuccess)
		}
	}

	// 13. Output summary
	logger.Logger("📊 Security Scan Summary:", logger.LogInfo)
	logger.Logger(fmt.Sprintf("  • Package: %s", filepath.Base(options.PackagePath)), logger.LogInfo)
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
	} else if !strings.Contains(scanResult.SignatureStatus, "signed by a developer certificate issued by Apple") {
		logger.Logger("⚠️ Package is not signed with an Apple-issued certificate", logger.LogWarning)
	} else {
		logger.Logger("✅ No critical security issues found in package", logger.LogSuccess)
	}

	return nil
}
