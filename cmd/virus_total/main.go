package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	virustotal "github.com/deploymenttheory/macos-autopkg-factory/tools/virus_total"
)

func main() {
	// Parse command line arguments
	filePath := flag.String("file", "", "Path to the file to analyze with VirusTotal")
	apiKey := flag.String("api-key", "", "Custom VirusTotal API key (optional)")
	autoSubmit := flag.Bool("auto-submit", false, "Automatically submit file to VirusTotal if not found (optional)")
	alwaysReport := flag.Bool("always-report", false, "Always request report even if file hasn't changed (optional)")
	jsonOutput := flag.String("json", "", "Path to export JSON results (optional)")
	disableAnalysis := flag.Bool("disable", false, "Disable VirusTotal analysis (optional)")
	flag.Parse()

	// Check required parameters
	if *filePath == "" {
		fmt.Println("Usage: vt-analyzer -file /path/to/file [-api-key key] [-auto-submit] [-always-report] [-json /path/to/results.json] [-disable]")
		os.Exit(1)
	}

	// Check if the file exists
	if _, err := os.Stat(*filePath); os.IsNotExist(err) {
		fmt.Printf("Error: File not found at %s\n", *filePath)
		os.Exit(1)
	}

	// Create configuration
	config := virustotal.DefaultConfig()

	// Apply custom options from command line
	if *apiKey != "" {
		config.APIKey = *apiKey
	}

	config.AutoSubmit = *autoSubmit
	config.AlwaysReport = *alwaysReport
	config.Disabled = *disableAnalysis

	// Create the analyzer
	analyzer := virustotal.NewAnalyzer(config)

	// Analyze the file
	result, err := analyzer.AnalyzeFile(*filePath, true)
	if err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	// Export results to JSON if requested
	if *jsonOutput != "" {
		if err := analyzer.ExportResultToJSON(result, nil, *filePath, *jsonOutput); err != nil {
			log.Fatalf("Failed to export results to JSON: %v", err)
		}
	}

	// Print summary to console
	fmt.Println("\n------- VirusTotal Analysis Summary -------")
	fmt.Printf("File: %s\n", result.FileName)
	fmt.Printf("Status: %s\n", result.Result)

	if result.Ratio != "" {
		fmt.Printf("Detection Ratio: %s\n", result.Ratio)
	}

	if result.Permalink != "None" {
		fmt.Printf("Permalink: %s\n", result.Permalink)
	}
}
