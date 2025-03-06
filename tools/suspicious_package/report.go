package suspiciouspackage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// ExportResultsToJSON exports the security scan results to a JSON file
func ExportResultsToJSON(result *SecurityScanResult, outputPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Marshal the results to JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal scan results to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON to file: %w", err)
	}

	logger.Logger(fmt.Sprintf("📄 Exported scan results to JSON: %s", outputPath), logger.LogSuccess)
	return nil
}
