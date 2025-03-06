package virustotal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// Default configuration values
const (
	// DefaultAPIKey is a dedicated API key for this tool - please don't abuse
	DefaultAPIKey = "3858a94a911f47707717f6d090dbb8f86badb750b0f7bfe74a55c0c6143e3de6"

	// DefaultSleepSeconds is the default wait time between API requests
	DefaultSleepSeconds = 60

	// DefaultAlwaysReport determines whether to always request reports
	DefaultAlwaysReport = false

	// DefaultAutoSubmit determines whether to automatically submit unknown files
	DefaultAutoSubmit = false

	// DefaultAutoSubmitMaxSize is the maximum file size for auto submission (400MB)
	DefaultAutoSubmitMaxSize = 419430400

	// LastRunTimeEnvVar is the environment variable to track last API request time
	LastRunTimeEnvVar = "AUTOPKG_VIRUSTOTAL_LAST_RUN_TIME"
)

// Analyzer is the main struct for interacting with VirusTotal
type Analyzer struct {
	config *Config
	client *http.Client
}

// NewAnalyzer creates a new VirusTotal analyzer with the given configuration
func NewAnalyzer(config *Config) *Analyzer {
	if config == nil {
		config = DefaultConfig()
	}

	return &Analyzer{
		config: config,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// CalculateSHA256 computes the SHA256 hash of a file
func (a *Analyzer) CalculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate hash: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// GetReportForHash requests a VirusTotal report for a file hash
func (a *Analyzer) GetReportForHash(fileHash string) (*AnalysisResult, error) {
	apiURL := "https://www.virustotal.com/vtapi/v2/file/report"

	// Create form data
	data := url.Values{}
	data.Set("resource", fileHash)
	data.Set("apikey", a.config.APIKey)

	// Create request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	// Execute request
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result AnalysisResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// SubmitFile submits a file to VirusTotal for scanning
func (a *Analyzer) SubmitFile(filePath string) (*AnalysisResult, error) {
	// First, get the upload URL
	uploadURLResp, err := a.getUploadURL()
	if err != nil {
		return nil, fmt.Errorf("failed to get upload URL: %w", err)
	}

	uploadURL := uploadURLResp.UploadURL
	if uploadURL == "" {
		return nil, fmt.Errorf("received empty upload URL")
	}

	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := newMultipartWriter(body)

	// Add file part
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file data: %w", err)
	}

	// Add API key
	if err := writer.WriteField("apikey", a.config.APIKey); err != nil {
		return nil, fmt.Errorf("failed to add API key: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", uploadURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Execute request
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result AnalysisResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse upload response: %w", err)
	}

	return &result, nil
}

// uploadURLResponse represents the response from the upload URL request
type uploadURLResponse struct {
	ResponseCode int    `json:"response_code"`
	VerboseMsg   string `json:"verbose_msg"`
	UploadURL    string `json:"upload_url"`
}

// getUploadURL gets a file upload URL from VirusTotal
func (a *Analyzer) getUploadURL() (*uploadURLResponse, error) {
	apiURL := "https://www.virustotal.com/vtapi/v2/file/scan/upload_url"

	// Create URL with query parameters
	reqURL, err := url.Parse(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	query := reqURL.Query()
	query.Set("apikey", a.config.APIKey)
	reqURL.RawQuery = query.Encode()

	// Create and execute request
	resp, err := a.client.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result uploadURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// multipartWriter is a small wrapper for creating multipart form data
type multipartWriter struct {
	*multipart.Writer
}

// newMultipartWriter creates a new multipart writer
func newMultipartWriter(body io.Writer) *multipartWriter {
	return &multipartWriter{
		Writer: multipart.NewWriter(body),
	}
}

// WriteField adds a field to the multipart form
func (w *multipartWriter) WriteField(fieldname, value string) error {
	part, err := w.CreateFormField(fieldname)
	if err != nil {
		return err
	}
	_, err = part.Write([]byte(value))
	return err
}

// FormDataContentType returns the content type of the form
func (w *multipartWriter) FormDataContentType() string {
	return w.Writer.FormDataContentType()
}

// AnalyzeFile is the main function that analyzes a file with VirusTotal
func (a *Analyzer) AnalyzeFile(filePath string, downloadChanged bool) (*SummaryResult, error) {
	// Check if analysis is disabled
	if a.config.Disabled {
		logger.Logger("Skipped VirusTotal analysis...", logger.LogInfo)
		return &SummaryResult{
			FileName:  filepath.Base(filePath),
			Result:    "SKIPPED",
			Permalink: "None",
		}, nil
	}

	// Validate file path
	if filePath == "" {
		logger.Logger("Skipping VirusTotal analysis: no input path defined.", logger.LogInfo)
		return &SummaryResult{
			Result:    "SKIPPED",
			Permalink: "None",
		}, nil
	}

	// Check if API key is available
	if a.config.APIKey == "" {
		return nil, fmt.Errorf("no API key available")
	}

	// Skip analysis if file hasn't changed and AlwaysReport is false
	if !downloadChanged && !a.config.AlwaysReport {
		logger.Logger("Skipping VirusTotal analysis: no new download.", logger.LogInfo)
		return &SummaryResult{
			FileName:  filepath.Base(filePath),
			Result:    "SKIPPED",
			Permalink: "None",
		}, nil
	}

	// Calculate file hash
	logger.Logger(fmt.Sprintf("🔍 Calculating checksum for %s", filePath), logger.LogInfo)
	fileHash, err := a.CalculateSHA256(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate hash: %w", err)
	}

	// Check if we need to wait before making a request
	if err := a.checkAndSleep(); err != nil {
		return nil, err
	}

	// Request the report
	logger.Logger("🔍 Requesting VirusTotal report...", logger.LogInfo)
	result, err := a.GetReportForHash(fileHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get report: %w", err)
	}

	// Update last run time
	if err := a.updateLastRunTime(); err != nil {
		logger.Logger(fmt.Sprintf("⚠️ Warning: Failed to update last run time: %v", err), logger.LogWarning)
	}

	// Process the report
	summary := &SummaryResult{
		FileName:  filepath.Base(filePath),
		Permalink: result.Permalink,
	}

	logger.Logger(fmt.Sprintf("📊 VirusTotal response code: %d", result.ResponseCode), logger.LogInfo)

	switch result.ResponseCode {
	case 0:
		// File not in VirusTotal database
		logger.Logger(fmt.Sprintf("🔍 No information found in VirusTotal for %s", filePath), logger.LogInfo)
		summary.Result = "NOT_FOUND"

		// Attempt to submit the file if auto-submit is enabled
		if a.config.AutoSubmit {
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to get file info: %w", err)
			}

			if fileInfo.Size() < a.config.AutoSubmitMaxSize {
				logger.Logger("📤 Submitting the file to VirusTotal for analysis...", logger.LogInfo)
				submitResult, err := a.SubmitFile(filePath)
				if err != nil {
					return nil, fmt.Errorf("failed to submit file: %w", err)
				}

				logger.Logger(fmt.Sprintf("📊 VirusTotal submission response code: %d", submitResult.ResponseCode), logger.LogInfo)
				logger.Logger(fmt.Sprintf("📝 Message: %s", submitResult.VerboseMsg), logger.LogInfo)
				logger.Logger(fmt.Sprintf("🔑 Scan ID: %s", submitResult.ScanID), logger.LogInfo)
				logger.Logger(fmt.Sprintf("🔗 Permalink: %s", submitResult.Permalink), logger.LogInfo)

				summary.Result = "SUBMITTED"
				summary.Permalink = submitResult.Permalink
			} else {
				logger.Logger("⚠️ File is too large to submit to VirusTotal...", logger.LogWarning)
			}
		} else {
			logger.Logger("💡 Consider submitting the file for analysis at https://www.virustotal.com/", logger.LogInfo)
		}

	case 1:
		// VirusTotal has information about the file
		logger.Logger(fmt.Sprintf("📝 Message: %s", result.VerboseMsg), logger.LogInfo)
		logger.Logger(fmt.Sprintf("🔑 Scan ID: %s", result.ScanID), logger.LogInfo)
		logger.Logger(fmt.Sprintf("📊 Detection ratio: %d/%d", result.Positives, result.Total), logger.LogInfo)
		logger.Logger(fmt.Sprintf("📅 Scan date: %s", result.ScanDate), logger.LogInfo)
		logger.Logger(fmt.Sprintf("🔗 Permalink: %s", result.Permalink), logger.LogInfo)

		summary.Result = "ANALYZED"
		summary.Ratio = fmt.Sprintf("%d/%d", result.Positives, result.Total)

		// Log warning or error based on detection ratio
		if result.Positives > 0 {
			if float64(result.Positives)/float64(result.Total) > 0.1 {
				// More than 10% of scanners detected something
				logger.Logger(fmt.Sprintf("❌ WARNING: File detected as potentially malicious by %d/%d scanners!",
					result.Positives, result.Total), logger.LogError)
			} else {
				// Less than 10% - probably a false positive
				logger.Logger(fmt.Sprintf("⚠️ File flagged by %d/%d scanners - possible false positive",
					result.Positives, result.Total), logger.LogWarning)
			}
		} else {
			logger.Logger("✅ File is clean according to VirusTotal", logger.LogSuccess)
		}

	case -2:
		// File is queued for analysis
		logger.Logger(fmt.Sprintf("⏳ Message: %s", result.VerboseMsg), logger.LogInfo)
		logger.Logger(fmt.Sprintf("🔑 Scan ID: %s", result.ScanID), logger.LogInfo)
		logger.Logger(fmt.Sprintf("🔗 Permalink: %s", result.Permalink), logger.LogInfo)

		summary.Result = "QUEUED"

	default:
		// Unexpected response code
		logger.Logger(fmt.Sprintf("⚠️ Unexpected VirusTotal response code: %d", result.ResponseCode), logger.LogWarning)
		logger.Logger(fmt.Sprintf("📝 Message: %s", result.VerboseMsg), logger.LogInfo)

		summary.Result = "ERROR"
	}

	return summary, nil
}

// checkAndSleep checks if we need to wait before making a request
func (a *Analyzer) checkAndSleep() error {
	// Get the last run time from the environment variable
	lastRunTimeStr := os.Getenv(LastRunTimeEnvVar)
	if lastRunTimeStr == "" {
		return nil // No last run time, proceed
	}

	lastRunTime, err := strconv.ParseInt(lastRunTimeStr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse last run time: %w", err)
	}

	if lastRunTime > 0 && a.config.SleepSeconds > 0 {
		now := time.Now().Unix()
		nextTime := lastRunTime + int64(a.config.SleepSeconds)

		if now < nextTime {
			sleepTime := nextTime - now
			logger.Logger(fmt.Sprintf("⏱️ Sleeping %d seconds before requesting VirusTotal report...", sleepTime), logger.LogInfo)
			time.Sleep(time.Duration(sleepTime) * time.Second)
		}
	}

	return nil
}

// updateLastRunTime updates the environment variable with the current time
func (a *Analyzer) updateLastRunTime() error {
	// Set the environment variable with the current time
	return os.Setenv(LastRunTimeEnvVar, strconv.FormatInt(time.Now().Unix(), 10))
}
