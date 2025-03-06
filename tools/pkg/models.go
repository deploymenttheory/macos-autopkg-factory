package pkg

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
