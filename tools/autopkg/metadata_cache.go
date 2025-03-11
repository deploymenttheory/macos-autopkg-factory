// metadata_cache.go
package autopkg

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/deploymenttheory/macos-autopkg-factory/tools/logger"
)

// CacheRecipeMetadata is a post-processor that handles caching and restoring AutoPkg metadata
type CacheRecipeMetadata struct {
	CachePath string
}

// NewCacheRecipeMetadata creates a new metadata cache post-processor
func NewCacheRecipeMetadata(cachePath string) *CacheRecipeMetadata {
	if cachePath == "" {
		cachePath = "/tmp/autopkg_metadata.json"
	}
	return &CacheRecipeMetadata{
		CachePath: cachePath,
	}
}

// Process implements the post-processor interface for AutoPkg
// It extracts metadata from the report and updates the cache
func (c *CacheRecipeMetadata) Process(reportPath string) error {
	logger.Logger("📝 Processing recipe metadata cache", logger.LogInfo)

	// Extract metadata from the report
	metadata, err := c.extractMetadataFromReport(reportPath)
	if err != nil {
		return fmt.Errorf("failed to extract metadata: %w", err)
	}

	// Update the cache with the new metadata
	if err := c.updateCache(metadata); err != nil {
		return fmt.Errorf("failed to update cache: %w", err)
	}

	logger.Logger("✅ Recipe metadata cache updated successfully", logger.LogSuccess)
	return nil
}

// RestoreCache loads the metadata cache and creates cache files for AutoPkg
func (c *CacheRecipeMetadata) RestoreCache() error {
	logger.Logger("🔄 Restoring recipe metadata cache", logger.LogInfo)

	// Check if cache file exists
	if _, err := os.Stat(c.CachePath); os.IsNotExist(err) {
		logger.Logger("ℹ️ No metadata cache found, continuing without cache", logger.LogInfo)
		return nil
	}

	// Load the cache
	cache, err := c.loadCache()
	if err != nil {
		return fmt.Errorf("failed to load cache: %w", err)
	}

	// Create cache files for each recipe
	if err := c.createCacheFiles(cache); err != nil {
		return fmt.Errorf("failed to create cache files: %w", err)
	}

	logger.Logger("✅ Recipe metadata cache restored successfully", logger.LogSuccess)
	return nil
}

// extractMetadataFromReport extracts download metadata from an AutoPkg report plist
func (c *CacheRecipeMetadata) extractMetadataFromReport(reportPath string) (map[string]interface{}, error) {
	// Parse the report plist
	reportData, err := parseReport(reportPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse report: %w", err)
	}

	// Get recipe name from the report
	recipeName := c.extractRecipeNameFromReport(reportData)

	// Extract download metadata
	downloadMetadata := c.extractDownloadMetadata(reportData)

	if len(downloadMetadata) == 0 {
		return nil, fmt.Errorf("no download metadata found in report")
	}

	// Create metadata structure
	metadata := map[string]interface{}{
		recipeName: map[string]interface{}{
			"download_metadata": downloadMetadata,
		},
	}

	return metadata, nil
}

// extractRecipeNameFromReport tries to extract the recipe name from the report data
func (c *CacheRecipeMetadata) extractRecipeNameFromReport(reportData map[string]interface{}) string {
	// Try to extract from recipe_list if available
	if recipeList, ok := reportData["recipe_list"].([]interface{}); ok && len(recipeList) > 0 {
		if recipeName, ok := recipeList[0].(string); ok {
			// Extract just the base name without extension
			baseName := filepath.Base(recipeName)
			return strings.TrimSuffix(baseName, filepath.Ext(baseName))
		}
	}

	// Fallback to other potential fields
	if pathname, ok := reportData["pathname"].(string); ok {
		baseName := filepath.Base(pathname)
		return strings.TrimSuffix(baseName, filepath.Ext(baseName))
	}

	// Use default if nothing found
	return "unknown_recipe"
}

// extractDownloadMetadata extracts download metadata from the report data
func (c *CacheRecipeMetadata) extractDownloadMetadata(reportData map[string]interface{}) []map[string]string {
	var downloadMetadata []map[string]string

	// Check for summary_results which contains download information
	if summaryResults, ok := reportData["summary_results"].(map[string]interface{}); ok {
		// Process URL downloader results
		if urlResults, ok := summaryResults["url_downloader_summary_result"].(map[string]interface{}); ok {
			if dataRows, ok := urlResults["data_rows"].([]interface{}); ok {
				for _, row := range dataRows {
					if rowData, ok := row.(map[string]interface{}); ok {
						downloadPath, _ := rowData["download_path"].(string)
						if downloadPath == "" {
							continue
						}

						etag, _ := rowData["etag"].(string)
						lastModified, _ := rowData["last_modified"].(string)

						// Get file size
						var sizeInBytes string
						if fileInfo, err := os.Stat(downloadPath); err == nil {
							sizeInBytes = fmt.Sprintf("%d", fileInfo.Size())
						} else {
							sizeInBytes = "0"
						}

						// Create metadata entry
						metadata := map[string]string{
							"pathname":         downloadPath,
							"etag":             etag,
							"last_modified":    lastModified,
							"dl_size_in_bytes": sizeInBytes,
						}

						downloadMetadata = append(downloadMetadata, metadata)
					}
				}
			}
		}

		// Look for other downloaders
		for key, value := range summaryResults {
			if strings.Contains(strings.ToLower(key), "download") && key != "url_downloader_summary_result" {
				c.processOtherDownloaderResults(value, &downloadMetadata)
			}
		}
	}

	return downloadMetadata
}

// processOtherDownloaderResults processes results from non-standard downloaders
func (c *CacheRecipeMetadata) processOtherDownloaderResults(downloaderResults interface{}, downloadMetadata *[]map[string]string) {
	if results, ok := downloaderResults.(map[string]interface{}); ok {
		if dataRows, ok := results["data_rows"].([]interface{}); ok {
			for _, row := range dataRows {
				if rowData, ok := row.(map[string]interface{}); ok {
					// Try to find a field that contains the download path
					downloadPath := ""
					for rowKey, rowValue := range rowData {
						if strings.Contains(strings.ToLower(rowKey), "path") {
							if pathStr, ok := rowValue.(string); ok && strings.HasPrefix(pathStr, "/") {
								downloadPath = pathStr
								break
							}
						}
					}

					if downloadPath == "" {
						continue
					}

					// Get file size
					var sizeInBytes string
					if fileInfo, err := os.Stat(downloadPath); err == nil {
						sizeInBytes = fmt.Sprintf("%d", fileInfo.Size())
					} else {
						sizeInBytes = "0"
					}

					// Create metadata entry
					metadata := map[string]string{
						"pathname":         downloadPath,
						"etag":             "", // May not be available
						"last_modified":    "", // May not be available
						"dl_size_in_bytes": sizeInBytes,
					}

					*downloadMetadata = append(*downloadMetadata, metadata)
				}
			}
		}
	}
}

// loadCache reads the metadata cache from disk
func (c *CacheRecipeMetadata) loadCache() (map[string]interface{}, error) {
	// Read cache file
	data, err := os.ReadFile(c.CachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	// Parse cache JSON
	var cache map[string]interface{}
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse cache: %w", err)
	}

	return cache, nil
}

// updateCache updates the metadata cache with new metadata
func (c *CacheRecipeMetadata) updateCache(newMetadata map[string]interface{}) error {
	// Load existing cache
	existingCache := make(map[string]interface{})
	if data, err := os.ReadFile(c.CachePath); err == nil {
		if err := json.Unmarshal(data, &existingCache); err != nil {
			logger.Logger(fmt.Sprintf("⚠️ Failed to parse existing cache: %v", err), logger.LogWarning)
			// Continue with empty cache
		}
	}

	// Merge new metadata with existing
	for recipe, metadata := range newMetadata {
		existingCache[recipe] = metadata
	}

	// Write updated cache
	data, err := json.MarshalIndent(existingCache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(c.CachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// createCacheFiles creates cache files from metadata
func (c *CacheRecipeMetadata) createCacheFiles(cache map[string]interface{}) error {
	// Get console user
	consoleUser := c.getConsoleUser()

	// Process each recipe
	for recipeName, recipeData := range cache {
		recipeDataMap, ok := recipeData.(map[string]interface{})
		if !ok {
			continue
		}

		downloadMetadataList, ok := recipeDataMap["download_metadata"].([]interface{})
		if !ok {
			continue
		}

		logger.Logger(fmt.Sprintf("📦 Creating cache files for %s", recipeName), logger.LogInfo)

		for _, downloadMetadataItem := range downloadMetadataList {
			downloadMetadataMap, ok := downloadMetadataItem.(map[string]interface{})
			if !ok {
				continue
			}

			// Extract metadata values
			pathname, _ := downloadMetadataMap["pathname"].(string)
			etag, _ := downloadMetadataMap["etag"].(string)
			lastModified, _ := downloadMetadataMap["last_modified"].(string)
			dlSizeInBytes, _ := downloadMetadataMap["dl_size_in_bytes"].(string)

			if pathname == "" {
				continue
			}

			// Update pathname with current user if needed
			if consoleUser != "" {
				pathname = c.updatePathWithConsoleUser(pathname, consoleUser)
			}

			// Create directories
			dir := filepath.Dir(pathname)
			if err := os.MkdirAll(dir, 0755); err != nil {
				logger.Logger(fmt.Sprintf("⚠️ Failed to create directory %s: %v", dir, err), logger.LogWarning)
				continue
			}

			// Create file with specified size
			if err := c.createSizedFile(pathname, dlSizeInBytes); err != nil {
				logger.Logger(fmt.Sprintf("⚠️ Failed to create file %s: %v", pathname, err), logger.LogWarning)
				continue
			}

			// Set extended attributes
			if etag != "" {
				if err := c.setXattr(pathname, "com.github.autopkg.etag", etag); err != nil {
					logger.Logger(fmt.Sprintf("⚠️ Failed to set etag attribute: %v", err), logger.LogWarning)
				}
			}

			if lastModified != "" {
				if err := c.setXattr(pathname, "com.github.autopkg.last-modified", lastModified); err != nil {
					logger.Logger(fmt.Sprintf("⚠️ Failed to set last-modified attribute: %v", err), logger.LogWarning)
				}
			}

			logger.Logger(fmt.Sprintf("✅ Created cache file at %s with size %s", pathname, dlSizeInBytes), logger.LogSuccess)
		}
	}

	return nil
}

// getConsoleUser gets the current console user
func (c *CacheRecipeMetadata) getConsoleUser() string {
	cmd := exec.Command("/usr/bin/stat", "-f%Su", "/dev/console")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// updatePathWithConsoleUser updates paths with the current console user
func (c *CacheRecipeMetadata) updatePathWithConsoleUser(pathname, consoleUser string) string {
	// Check if path contains /Users/username pattern
	userDirPattern := "/Users/([^/]+)/"
	re := regexp.MustCompile(userDirPattern)
	matches := re.FindStringSubmatch(pathname)

	if len(matches) > 1 {
		targetUsername := matches[1]
		if targetUsername != consoleUser {
			logger.Logger(fmt.Sprintf("🔄 Updating path from user %s to %s", targetUsername, consoleUser), logger.LogDebug)
			return strings.Replace(pathname, "/Users/"+targetUsername+"/", "/Users/"+consoleUser+"/", 1)
		}
	}

	return pathname
}

// createSizedFile creates a file with the specified size
func (c *CacheRecipeMetadata) createSizedFile(pathname, size string) error {
	if size == "" {
		size = "0"
	}

	cmd := exec.Command("/usr/bin/mkfile", "-n", size, pathname)
	return cmd.Run()
}

// setXattr sets an extended attribute on a file
func (c *CacheRecipeMetadata) setXattr(pathname, attr, value string) error {
	cmd := exec.Command("/usr/bin/xattr", "-w", attr, value, pathname)
	return cmd.Run()
}

// AddCacheMetadataPostProcessor adds the CacheRecipeMetadata post-processor to RunOptions
func AddCacheMetadataPostProcessor(options *RunOptions, cachePath string) {
	// Initialize post-processors slice if needed
	if options.PostProcessors == nil {
		options.PostProcessors = []string{}
	}

	// Add CacheRecipeMetadata post-processor
	options.PostProcessors = append(options.PostProcessors, "io.kandji.cachedata/CacheRecipeMetadata")
}
