// Package main provides the entry point for the macOS package scanner
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	sp "github.com/deploymenttheory/macos-autopkg-factory/tools/suspicious_package"
)

func main() {
	// Parse command line arguments
	packagePath := flag.String("package", "", "Path to the package file to analyze")
	outputDir := flag.String("output", "", "Directory to export results (optional)")
	checkTerm := flag.String("check-scripts", "", "Term to search for in installer scripts (optional)")
	checkOSVersion := flag.String("check-os", "", "Check compatibility with this macOS version (e.g. '14.0') (optional)")
	jsonOutput := flag.String("json", "", "Path to export JSON results (optional)")
	flag.Parse()

	if *packagePath == "" {
		fmt.Println("Usage: package-scanner -package /path/to/package.pkg [-output /path/to/export] [-check-scripts term] [-check-os version]")
		os.Exit(1)
	}

	// Check if the package exists
	if _, err := os.Stat(*packagePath); os.IsNotExist(err) {
		fmt.Printf("Error: Package not found at %s\n", *packagePath)
		os.Exit(1)
	}

	// Set up options for scanner
	scanOptions := sp.ScanOptions{
		PackagePath:    *packagePath,
		OutputDir:      *outputDir,
		CheckTerm:      *checkTerm,
		CheckOSVersion: *checkOSVersion,
		JSONOutput:     *jsonOutput,
	}

	if err := sp.PackageSecurityScanner(scanOptions); err != nil {
		log.Fatalf("Failed to scan package: %v", err)
	}
}
