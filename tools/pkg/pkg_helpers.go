package pkg

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// GetPackageSigningCertificate retrieves the signing certificate information for a package
func GetPackageSigningCertificate(packagePath string) (*PackageSigningCertificate, error) {
	logger.Logger(fmt.Sprintf("🔍 Checking package signature for: %s", packagePath), logger.LogInfo)

	cmd := exec.Command("pkgutil", "--check-signature", packagePath)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		logger.Logger(fmt.Sprintf("❌ Failed to check package signature: %v", err), logger.LogError)
		return nil, fmt.Errorf("failed to check package signature: %w", err)
	}

	output := out.String()
	lines := strings.Split(output, "\n")

	certDetails := &PackageSigningCertificate{
		SignatureStatus:  "Unknown",
		Notarized:        false,
		CertificateInfo:  "Unknown",
		CertificateChain: []string{},
		ExpiryDates:      []string{},
	}

	// Regex patterns
	signaturePattern := regexp.MustCompile(`Status:\s*(.+)`)
	notarizedPattern := regexp.MustCompile(`Notarization:\s*(.+)`)
	developerIDPattern := regexp.MustCompile(`Developer ID Installer:\s*(.+) \(([\w\d]+)\)`)
	expiryPattern := regexp.MustCompile(`Expires:\s*(.+)`)
	authorityPattern := regexp.MustCompile(`^\s*\d+\.\s*(.+)`)

	var lastCertIndex int

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Extract Signature Status
		if signaturePattern.MatchString(line) {
			matches := signaturePattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				certDetails.SignatureStatus = matches[1]
				logger.Logger(fmt.Sprintf("🔐 Signature Status: %s", certDetails.SignatureStatus), logger.LogSuccess)
			}
		}

		// Extract Notarization Status
		if notarizedPattern.MatchString(line) {
			matches := notarizedPattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				certDetails.Notarized = strings.Contains(matches[1], "trusted by the Apple notary service")
				logger.Logger(fmt.Sprintf("🔐 Notarized: %t", certDetails.Notarized), logger.LogSuccess)
			}
		}

		// Extract Developer ID Info
		if developerIDPattern.MatchString(line) {
			matches := developerIDPattern.FindStringSubmatch(line)
			if len(matches) == 3 {
				certDetails.CertificateInfo = matches[1] // Organization Name
				logger.Logger(fmt.Sprintf("🔐 Signed by: %s", certDetails.CertificateInfo), logger.LogInfo)
			}
		}

		// Extract Full Certificate Chain
		if authorityPattern.MatchString(line) {
			matches := authorityPattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				certDetails.CertificateChain = append(certDetails.CertificateChain, matches[1])
				lastCertIndex = len(certDetails.CertificateChain) - 1
			}
		}

		// Extract Certificate Expiry Date and associate it with the last certificate
		if expiryPattern.MatchString(line) {
			matches := expiryPattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				expiry := matches[1]
				if lastCertIndex >= 0 && lastCertIndex < len(certDetails.CertificateChain) {
					certDetails.ExpiryDates = append(certDetails.ExpiryDates, expiry)
				}
			}
		}
	}

	// Log Certificate Chain with Expiry Dates using arrows
	if len(certDetails.CertificateChain) > 0 {
		logger.Logger("🔐 Certificate Chain:", logger.LogInfo)
		for i, cert := range certDetails.CertificateChain {
			expiry := "Unknown"
			if i < len(certDetails.ExpiryDates) {
				expiry = certDetails.ExpiryDates[i]
			}

			if i == 0 {
				// First certificate in the chain
				logger.Logger(fmt.Sprintf("  • %s (Expires: %s)", cert, expiry), logger.LogInfo)
			} else {
				// Show relationship using arrows
				logger.Logger(fmt.Sprintf("    → %s (Expires: %s)", cert, expiry), logger.LogInfo)
			}
		}
	} else {
		logger.Logger("⚠️ No certificate chain found - package may be unsigned", logger.LogWarning)
	}

	return certDetails, nil
}

// GetPackageSupportedMacOSArchitecture extracts the supported macOS architectures from a package
func GetPackageSupportedMacOSArchitecture(packagePath string) ([]string, error) {
	logger.Logger(fmt.Sprintf("🔍 Checking supported macOS architectures for: %s", packagePath), logger.LogInfo)

	// Create a unique temp directory for expansion
	tempDir, err := os.MkdirTemp("", "expanded_pkg_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after function exits

	// Expand the package
	cmd := exec.Command("pkgutil", "--expand", packagePath, tempDir)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to expand package: %w", err)
	}

	// Check if Distribution file exists (indicating a distribution package)
	distFile := filepath.Join(tempDir, "Distribution")
	if _, err := os.Stat(distFile); os.IsNotExist(err) {
		logger.Logger("⚠️ No Distribution file found – assuming component package", logger.LogWarning)
		return []string{"x86_64"}, nil // Default assumption
	}

	// Read the Distribution XML file
	data, err := os.ReadFile(distFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read Distribution file: %w", err)
	}

	// Extract architectures from the <options> tag
	archRegex := regexp.MustCompile(`hostArchitectures="([^"]+)"`)
	matches := archRegex.FindStringSubmatch(string(data))

	if len(matches) < 2 {
		logger.Logger("⚠️ No explicit hostArchitectures found, package may default to x86_64", logger.LogWarning)
		return []string{"x86_64"}, nil // Default fallback
	}

	// Parse architectures
	architectures := strings.Split(matches[1], ",")
	logger.Logger(fmt.Sprintf("✅ Package supports architectures: %s", strings.Join(architectures, ", ")), logger.LogSuccess)
	return architectures, nil
}
