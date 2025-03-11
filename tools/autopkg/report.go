// report.go
package autopkg

import (
	"fmt"
	"os"

	"howett.net/plist"
)

func parseReport(reportPath string) (map[string]interface{}, error) {
	file, err := os.Open(reportPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open report file: %w", err)
	}
	defer file.Close()

	decoder := plist.NewDecoder(file)
	var reportData map[string]interface{}
	if err := decoder.Decode(&reportData); err != nil {
		return nil, fmt.Errorf("failed to decode report plist: %w", err)
	}

	parsedResults := map[string]interface{}{
		"imported": []interface{}{},
		"failed":   []interface{}{},
		"removed":  []interface{}{},
		"promoted": []interface{}{},
	}

	// Extract failures
	if failures, exists := reportData["failures"].([]interface{}); exists {
		parsedResults["failed"] = failures
	}

	// Extract summary results if present
	if summaryResults, exists := reportData["summary_results"].(map[string]interface{}); exists {
		if intuneResults, ok := summaryResults["intuneappuploader_summary_result"].(map[string]interface{}); ok {
			parsedResults["imported"] = intuneResults["data_rows"]
		}
		if removedResults, ok := summaryResults["intuneappcleaner_summary_result"].(map[string]interface{}); ok {
			parsedResults["removed"] = removedResults["data_rows"]
		}
		if promotedResults, ok := summaryResults["intuneapppromoter_summary_result"].(map[string]interface{}); ok {
			parsedResults["promoted"] = promotedResults["data_rows"]
		}
	}

	return parsedResults, nil
}
