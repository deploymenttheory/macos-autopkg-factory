package suspiciouspackage

import "time"

// PackageInfo represents basic information about a package
type PackageInfo struct {
	Name            string `json:"name"`
	SignatureStatus string `json:"signatureStatus"`
	Notarized       bool   `json:"notarized"`
}

// InstalledItem represents an item installed by a package
type InstalledItem struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Kind        string `json:"kind"`
	BundleID    string `json:"bundleID,omitempty"`
	Permissions string `json:"permissions,omitempty"`
}

// InstallerScript represents a script in the package
type InstallerScript struct {
	Name     string `json:"name"`
	RunsWhen string `json:"runsWhen"`
	Binary   bool   `json:"binary"`
	Text     string `json:"text,omitempty"`
}

// PackageIssue represents an issue found in a package
type PackageIssue struct {
	Details  string `json:"details"`
	Priority string `json:"priority"`
	Path     string `json:"path,omitempty"`
}

// ComponentWithEntitlement represents a component with a specific entitlement
type ComponentWithEntitlement struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

// ArchitectureSupport represents information about architecture support
type ArchitectureSupport struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Supports bool   `json:"supports"`
}

// LaunchdJob represents information about a launchd job configuration file
type LaunchdJob struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Owner       string `json:"owner"`
	OwnerID     int    `json:"ownerID"`
	Permissions string `json:"permissions"`
}

// PrivilegedInstallerScript represents an installer script running as root
type PrivilegedInstallerScript struct {
	Name       string `json:"name"`
	ShortName  string `json:"shortName"`
	When       string `json:"when"`
	Binary     bool   `json:"binary"`
	ScriptText string `json:"scriptText"`
	ScriptPath string `json:"scriptPath"`
}

// OSRequirement represents OS version requirements for an executable component
type OSRequirement struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Platform string `json:"platform"`
	Version  string `json:"version"`
	Major    int    `json:"major"`
}

// NonStandardPermission represents a file with unusual permissions
type NonStandardPermission struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Permissions string `json:"symPerm"`
	Owner       string `json:"owner"`
	Group       string `json:"group"`
}

// ComponentInfo represents information about a component package
type ComponentInfo struct {
	ID               string    `json:"id"`
	Version          string    `json:"version"`
	Installed        bool      `json:"installed"`
	InstalledVersion string    `json:"installedVersion,omitempty"`
	InstalledDate    time.Time `json:"installedDate,omitempty"`
	InstallingApp    string    `json:"installingApp,omitempty"`
}

// UTIItem represents an item that conforms to a specific UTI
type UTIItem struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	UTIName string `json:"utiName"`
	Kind    string `json:"kind"`
}

// SandboxedApp represents a sandboxed application with its network entitlements
type SandboxedApp struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	BundleID      string `json:"bundleID"`
	ClientNetwork bool   `json:"clientNetwork"`
	ServerNetwork bool   `json:"serverNetwork"`
}

// PackageSigningCertificate represents certificate info
type PackageSigningCertificate struct {
	SignatureStatus  string
	Notarized        bool
	CertificateInfo  string
	CertificateChain []string
	ExpiryDates      []string
}

// PackageArchitecture represents the supported architectures found in a pkg
type PackageArchitecture struct {
	HostArchitectures []string `xml:"options>hostArchitectures"`
}
