package suspiciouspackage

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
	"github.com/deploymenttheory/macos-autopkg-factory/tools/pkg"
)

// SecurityScanResult represents the results of a package security scan
type SecurityScanResult struct {
	// Basic package information
	PackagePath       string `json:"packagePath"`
	SignatureStatus   string `json:"signatureStatus"`
	Notarized         bool   `json:"notarized"`
	CertificateInfo   string `json:"certificateInfo"`
	CertificateIssuer string `json:"certificateIssuer"`
	CertificateExpiry string `json:"certificateExpiry"`

	// Summary counts
	CriticalIssues      int `json:"criticalIssuesCount"`
	WarningIssues       int `json:"warningIssuesCount"`
	PrivilegedScripts   int `json:"privilegedScriptsCount"`
	LaunchdJobs         int `json:"launchdJobsCount"`
	NonStdPermissions   int `json:"nonStandardPermissionsCount"`
	ComponentCount      int `json:"componentCount"`
	SandboxedApps       int `json:"sandboxedAppsCount"`
	AppleSiliconSupport int `json:"appleSiliconSupport"`

	// Detailed findings
	Issues                  []PackageIssue              `json:"issues,omitempty"`
	PrivilegedScriptDetails []PrivilegedInstallerScript `json:"privilegedScripts,omitempty"`
	LaunchdJobDetails       []LaunchdJob                `json:"launchdJobs,omitempty"`
	NonStandardPermDetails  []NonStandardPermission     `json:"nonStandardPermissions,omitempty"`
	Components              []ComponentInfo             `json:"components,omitempty"`
	SandboxedAppDetails     []SandboxedApp              `json:"sandboxedApps,omitempty"`
	OSRequirements          []OSRequirement             `json:"osRequirements,omitempty"`
	IncompatibleItems       []OSRequirement             `json:"incompatibleItems,omitempty"`
	SupportedArchitectures  []string                    `json:"supportedArchitectures,omitempty"`

	// Scan metadata
	ScanDate                 time.Time `json:"scanDate"`
	ScanDuration             string    `json:"scanDuration,omitempty"`
	SuspiciousPackageVersion string    `json:"suspiciousPackageVersion,omitempty"`
}

// ScanOptions represents the options for package security scanning
type ScanOptions struct {
	PackagePath    string
	OutputDir      string
	CheckTerm      string
	CheckOSVersion string
	JSONOutput     string
}

// PackageSecurityScanner scans a macOS package for security issues
func PackageSecurityScanner(options ScanOptions) error {
	// Record start time for duration calculation
	startTime := time.Now()

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

	// Variables to store detailed findings for JSON export
	var allIssues []PackageIssue
	var privilegedScripts []PrivilegedInstallerScript
	var launchdJobs []LaunchdJob
	var nonStdPerms []NonStandardPermission
	var components []ComponentInfo
	var sandboxedApps []SandboxedApp
	var osRequirements []OSRequirement
	var incompatibleItems []OSRequirement
	var supportedArchitectures []string

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
		// Store all issues for detailed export
		allIssues = issues

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
		logger.Logger(fmt.Sprintf("❌ Failed to find privileged scripts: %v", err), logger.LogError)
	} else {
		// Store for detailed export
		privilegedScripts = scripts
		scanResult.PrivilegedScripts = len(scripts)
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
	ljobs, err := FindLaunchdJobs(options.PackagePath)
	if err != nil {
		fmt.Printf("Failed to find launchd jobs: %v\n", err)
	} else {
		// Store for detailed export
		launchdJobs = ljobs
		scanResult.LaunchdJobs = len(ljobs)

		if len(ljobs) > 0 {
			logger.Logger(fmt.Sprintf("⚙️ Found %d launchd jobs:", len(ljobs)), logger.LogInfo)
			for _, job := range ljobs {
				logger.Logger(fmt.Sprintf("  • %s (owner: %s, permissions: %s)", job.Path, job.Owner, job.Permissions), logger.LogInfo)
			}
		}
	}

	// 6. Check for files with non-standard permissions
	perms, err := FindNonStandardPermissions(options.PackagePath)
	if err != nil {
		fmt.Printf("Failed to check for non-standard permissions: %v\n", err)
	} else {
		// Store for detailed export
		nonStdPerms = perms
		scanResult.NonStdPermissions = len(perms)

		if len(perms) > 0 {
			logger.Logger(fmt.Sprintf("🔓 Found %d files with non-standard permissions:", len(perms)), logger.LogWarning)

			// Only show the first 5 to avoid too much output
			maxShow := 5
			if len(perms) < maxShow {
				maxShow = len(perms)
			}

			for i := 0; i < maxShow; i++ {
				perm := perms[i]
				logger.Logger(fmt.Sprintf("  • %s (%s:%s, %s)", perm.Path, perm.Owner, perm.Group, perm.Permissions), logger.LogInfo)
			}

			if len(perms) > maxShow {
				logger.Logger(fmt.Sprintf("  • ... and %d more", len(perms)-maxShow), logger.LogInfo)
			}
		}
	}

	// 7. Check component packages
	comps, err := GetComponentPackages(options.PackagePath)
	if err != nil {
		fmt.Printf("Failed to check component packages: %v\n", err)
	} else {
		// Store for detailed export
		components = comps
		scanResult.ComponentCount = len(comps)

		installedCount := 0
		for _, comp := range comps {
			if comp.Installed {
				installedCount++
			}
		}

		if installedCount > 0 {
			logger.Logger(fmt.Sprintf("📦 Found %d component packages, %d already installed", len(comps), installedCount), logger.LogInfo)
			for _, comp := range comps {
				if comp.Installed {
					logger.Logger(fmt.Sprintf("  • %s: v%s installed on %s (current pkg: v%s)",
						comp.ID, comp.InstalledVersion, comp.InstalledDate.Format("2006-01-02"), comp.Version), logger.LogInfo)
				}
			}
		}
	}

	// 8. Find sandboxed applications
	sbApps, err := FindSandboxedApps(options.PackagePath)
	if err != nil {
		fmt.Printf("Failed to find sandboxed apps: %v\n", err)
	} else {
		// Store for detailed export
		sandboxedApps = sbApps
		scanResult.SandboxedApps = len(sbApps)

		if len(sbApps) > 0 {
			logger.Logger(fmt.Sprintf("🔒 Found %d sandboxed applications:", len(sbApps)), logger.LogInfo)
			for _, app := range sbApps {
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
	supArch, err := pkg.GetPackageSupportedMacOSArchitecture(options.PackagePath)
	if err != nil {
		logger.Logger(fmt.Sprintf("❌ Failed to get package architectures: %v", err), logger.LogError)
	} else {
		// Store for detailed export
		supportedArchitectures = supArch
		logger.Logger(fmt.Sprintf("💻 Package supports: %s", strings.Join(supArch, ", ")), logger.LogSuccess)

		// Determine if Apple Silicon (arm64) is supported
		supportsAppleSilicon := false
		for _, arch := range supArch {
			if arch == "arm64" {
				supportsAppleSilicon = true
				break
			}
		}

		if supportsAppleSilicon {
			logger.Logger("✅ Package explicitly supports Apple Silicon (arm64)", logger.LogSuccess)
			scanResult.AppleSiliconSupport = 1
		} else {
			logger.Logger("⚠️ Package does not explicitly declare Apple Silicon support", logger.LogWarning)
			scanResult.AppleSiliconSupport = 0
		}
	}

	// 10. Check OS compatibility if requested
	if options.CheckOSVersion != "" {
		incompatible, err := CheckOSCompatibility(options.PackagePath, options.CheckOSVersion)
		if err != nil {
			fmt.Printf("Failed to check compatibility with macOS %s: %v\n", options.CheckOSVersion, err)
		} else {
			// Store for detailed export
			incompatibleItems = incompatible

			if len(incompatible) > 0 {
				logger.Logger(fmt.Sprintf("⚠️ Found %d components incompatible with macOS %s:",
					len(incompatible), options.CheckOSVersion), logger.LogWarning)

				for _, comp := range incompatible {
					logger.Logger(fmt.Sprintf("  • %s requires macOS %s", comp.Name, comp.Version), logger.LogInfo)
				}
			} else {
				logger.Logger(fmt.Sprintf("✅ All components are compatible with macOS %s", options.CheckOSVersion), logger.LogSuccess)
			}
		}
	}

	// 11. Get OS requirements for the package
	requirements, err := GetMacOSMinimumVersionRequirements(options.PackagePath)
	if err != nil {
		logger.Logger(fmt.Sprintf("❌ Failed to get OS requirements: %v", err), logger.LogError)
	} else {
		// Store for detailed export
		osRequirements = requirements
	}

	// 13. Export results to JSON if requested
	if options.JSONOutput != "" {
		// Calculate scan duration
		scanDuration := time.Since(startTime)

		if err := ExportResultsToJSON(&scanResult,
			allIssues,
			privilegedScripts,
			launchdJobs,
			nonStdPerms,
			components,
			sandboxedApps,
			osRequirements,
			incompatibleItems,
			supportedArchitectures,
			version,
			scanDuration,
			options.JSONOutput); err != nil {
			logger.Logger(fmt.Sprintf("❌ Failed to export results to JSON: %v", err), logger.LogError)
		}
	}

	// 14. Output summary
	logger.Logger("📊 Security Scan Summary:", logger.LogInfo)
	logger.Logger(fmt.Sprintf("  • Package: %s", filepath.Base(options.PackagePath)), logger.LogInfo)
	logger.Logger(fmt.Sprintf("  • Signature: %s, Notarized: %t", scanResult.SignatureStatus, scanResult.Notarized), logger.LogInfo)
	logger.Logger(fmt.Sprintf("  • Issues: %d critical, %d warnings", scanResult.CriticalIssues, scanResult.WarningIssues), logger.LogInfo)
	logger.Logger(fmt.Sprintf("  • Privileged scripts: %d", scanResult.PrivilegedScripts), logger.LogInfo)
	logger.Logger(fmt.Sprintf("  • Launchd jobs: %d", scanResult.LaunchdJobs), logger.LogInfo)
	logger.Logger(fmt.Sprintf("  • Non-standard permissions: %d", scanResult.NonStdPermissions), logger.LogInfo)
	logger.Logger(fmt.Sprintf("  • Component packages: %d", scanResult.ComponentCount), logger.LogInfo)
	logger.Logger(fmt.Sprintf("  • Sandboxed applications: %d", scanResult.SandboxedApps), logger.LogInfo)
	//logger.Logger(fmt.Sprintf("  • Scan duration: %s", scanDuration), logger.LogInfo)

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
