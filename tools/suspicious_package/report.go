package suspiciouspackage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// ExportResultsToJSON exports the security scan results to a JSON file
func ExportResultsToJSON(result *SecurityScanResult,
	issues []PackageIssue,
	privilegedScripts []PrivilegedInstallerScript,
	launchdJobs []LaunchdJob,
	nonStdPerms []NonStandardPermission,
	components []ComponentInfo,
	sandboxedApps []SandboxedApp,
	osRequirements []OSRequirement,
	incompatibleItems []OSRequirement,
	supportedArchitectures []string,
	spVersion string,
	scanDuration time.Duration,
	outputPath string) error {

	enhancedResult := SecurityScanResult{
		PackagePath:             result.PackagePath,
		SignatureStatus:         result.SignatureStatus,
		Notarized:               result.Notarized,
		CertificateInfo:         result.CertificateInfo,
		CertificateIssuer:       result.CertificateIssuer,
		CertificateExpiry:       result.CertificateExpiry,
		CriticalIssues:          result.CriticalIssues,
		WarningIssues:           result.WarningIssues,
		PrivilegedScripts:       result.PrivilegedScripts,
		LaunchdJobs:             result.LaunchdJobs,
		NonStdPermissions:       result.NonStdPermissions,
		ComponentCount:          result.ComponentCount,
		SandboxedApps:           result.SandboxedApps,
		AppleSiliconSupport:     result.AppleSiliconSupport,
		Issues:                  issues,
		PrivilegedScriptDetails: privilegedScripts,
		LaunchdJobDetails:       launchdJobs,
		NonStandardPermDetails:  nonStdPerms,
		Components:              components,
		SandboxedAppDetails:     sandboxedApps,
		OSRequirements:          osRequirements,
		IncompatibleItems:       incompatibleItems,
		SupportedArchitectures:  supportedArchitectures,

		// Add scan metadata
		ScanDate:                 time.Now(),
		ScanDuration:             scanDuration.String(),
		SuspiciousPackageVersion: spVersion,
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Marshal the results to JSON
	jsonData, err := json.MarshalIndent(enhancedResult, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal scan results to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON to file: %w", err)
	}

	logger.Logger(fmt.Sprintf("📄 Exported detailed scan results to JSON: %s", outputPath), logger.LogSuccess)
	return nil
}
