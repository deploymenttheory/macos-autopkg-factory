package virustotal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// DetailedReport contains comprehensive information about a VirusTotal scan
type DetailedReport struct {
	// Basic file information
	FileName   string `json:"fileName"`
	FilePath   string `json:"filePath"`
	FileSize   int64  `json:"fileSize"`
	FileSHA256 string `json:"fileSHA256"`

	// Analysis results
	Status         string `json:"status"` // NOT_FOUND, SUBMITTED, QUEUED, ANALYZED, SKIPPED, ERROR
	DetectionRatio string `json:"detectionRatio,omitempty"`
	Permalink      string `json:"permalink,omitempty"`

	// VirusTotal specific data
	ResponseCode   int    `json:"responseCode,omitempty"`
	ScanID         string `json:"scanID,omitempty"`
	ScanDate       string `json:"scanDate,omitempty"`
	VerboseMessage string `json:"verboseMessage,omitempty"`
	Positives      int    `json:"positives,omitempty"`
	Total          int    `json:"total,omitempty"`

	// Scan metadata
	ScanTime   time.Time `json:"scanTime"`
	APIKeyUsed bool      `json:"apiKeyUsed"`
	AutoSubmit bool      `json:"autoSubmit"`
}

// ExportResultToJSON exports the scan results to a JSON file
func (a *Analyzer) ExportResultToJSON(result *SummaryResult, analysis *AnalysisResult, filePath string, jsonPath string) error {
	// Check if file path exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Calculate hash if we don't have it already
	fileSHA256 := ""
	if analysis != nil && analysis.SHA256 != "" {
		fileSHA256 = analysis.SHA256
	} else {
		hash, err := a.CalculateSHA256(filePath)
		if err != nil {
			logger.Logger(fmt.Sprintf("⚠️ Warning: Failed to calculate file hash: %v", err), logger.LogWarning)
		} else {
			fileSHA256 = hash
		}
	}

	// Create detailed report
	report := DetailedReport{
		FileName:       filepath.Base(filePath),
		FilePath:       filePath,
		FileSize:       fileInfo.Size(),
		FileSHA256:     fileSHA256,
		Status:         result.Result,
		DetectionRatio: result.Ratio,
		Permalink:      result.Permalink,
		ScanTime:       time.Now(),
		APIKeyUsed:     a.config.APIKey != DefaultAPIKey,
		AutoSubmit:     a.config.AutoSubmit,
	}

	// Add VirusTotal specific data if available
	if analysis != nil {
		report.ResponseCode = analysis.ResponseCode
		report.ScanID = analysis.ScanID
		report.ScanDate = analysis.ScanDate
		report.VerboseMessage = analysis.VerboseMsg
		report.Positives = analysis.Positives
		report.Total = analysis.Total
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(jsonPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Marshal the report to JSON
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON to file: %w", err)
	}

	logger.Logger(fmt.Sprintf("📄 Exported VirusTotal scan results to JSON: %s", jsonPath), logger.LogSuccess)
	return nil
}

// GenerateReportName creates a standardized filename for reports
func GenerateReportName(filePath string) string {
	baseName := filepath.Base(filePath)
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("virustotal-%s-%s.json", baseName, timestamp)
}
