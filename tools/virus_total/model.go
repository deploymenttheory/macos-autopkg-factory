// Package virustotal provides functionality to analyze files using the VirusTotal API
package virustotal

// Config holds the configuration for the VirusTotal analyzer
type Config struct {
	// APIKey is the VirusTotal API key
	APIKey string

	// AlwaysReport forces report generation even if the file hasn't changed
	AlwaysReport bool

	// AutoSubmit enables automatic submission of unknown files
	AutoSubmit bool

	// AutoSubmitMaxSize is the maximum file size for auto submission
	AutoSubmitMaxSize int64

	// SleepSeconds is the wait time between API requests
	SleepSeconds int

	// Disabled allows disabling the analyzer
	Disabled bool
}

// AnalysisResult contains the results of a VirusTotal analysis
type AnalysisResult struct {
	ResponseCode int    `json:"response_code"`
	VerboseMsg   string `json:"verbose_msg"`
	Resource     string `json:"resource,omitempty"`
	ScanID       string `json:"scan_id,omitempty"`
	Permalink    string `json:"permalink,omitempty"`
	ScanDate     string `json:"scan_date,omitempty"`
	Positives    int    `json:"positives,omitempty"`
	Total        int    `json:"total,omitempty"`
	MD5          string `json:"md5,omitempty"`
	SHA1         string `json:"sha1,omitempty"`
	SHA256       string `json:"sha256,omitempty"`
}

// SummaryResult provides a summarized result of the analysis
type SummaryResult struct {
	FileName  string
	Ratio     string
	Permalink string
	Result    string // SKIPPED, SUBMITTED, QUEUED, ANALYZED
}

// DefaultConfig creates a new Config with default values
func DefaultConfig() *Config {
	return &Config{
		APIKey:            DefaultAPIKey,
		AlwaysReport:      DefaultAlwaysReport,
		AutoSubmit:        DefaultAutoSubmit,
		AutoSubmitMaxSize: DefaultAutoSubmitMaxSize,
		SleepSeconds:      DefaultSleepSeconds,
		Disabled:          false,
	}
}
